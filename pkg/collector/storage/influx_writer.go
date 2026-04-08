package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"rbac-admin/pkg/logger"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// InfluxWriter 负责向 InfluxDB 写入数据点，并处理批处理和定时刷新逻辑。
type InfluxWriter struct {
	client        *influxdb3.Client
	ctx           context.Context
	cancel        context.CancelFunc
	dataPoints    []*influxdb3.Point
	dataPointsMu  sync.Mutex
	batchSize     int
	flushInterval time.Duration
	flushTicker   *time.Ticker
	flushStopChan chan struct{}
}

// NewInfluxWriter 创建一个新的 InfluxWriter 实例。
func NewInfluxWriter(client *influxdb3.Client) *InfluxWriter {
	ctx, cancel := context.WithCancel(context.Background())
	writer := &InfluxWriter{
		client:        client,
		ctx:           ctx,
		cancel:        cancel,
		dataPoints:    make([]*influxdb3.Point, 0),
		batchSize:     100,             // 默认批量插入100条
		flushInterval: 3 * time.Second, // 默认3秒插入一次
		flushStopChan: make(chan struct{}),
	}
	go writer.startFlushTicker()
	return writer
}

// AddDataPoint 添加数据点到缓存。
func (w *InfluxWriter) AddDataPoint(point *influxdb3.Point) {
	w.dataPointsMu.Lock()
	defer w.dataPointsMu.Unlock()

	w.dataPoints = append(w.dataPoints, point)

	// 如果达到批量大小，立即刷新
	if len(w.dataPoints) >= w.batchSize {
		go w.flushDataPoints()
	}
}

// StartFlushTicker 启动定时刷新器。
func (w *InfluxWriter) startFlushTicker() {
	w.flushTicker = time.NewTicker(w.flushInterval)
	defer w.flushTicker.Stop()

	for {
		select {
		case <-w.flushTicker.C:
			w.flushDataPoints()
		case <-w.flushStopChan:
			return
		}
	}
}

// flushDataPoints 刷新缓存中的数据点到 InfluxDB。
func (w *InfluxWriter) flushDataPoints() {
	w.dataPointsMu.Lock()
	if len(w.dataPoints) == 0 {
		w.dataPointsMu.Unlock()
		return
	}

	points := make([]*influxdb3.Point, len(w.dataPoints))
	copy(points, w.dataPoints)
	w.dataPoints = w.dataPoints[:0]
	w.dataPointsMu.Unlock()

	if len(points) > 0 {
		err := w.client.WritePoints(w.ctx, points)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("批量数据写入 InfluxDB 失败: %v", err))
			// 写入失败，将数据重新放回缓存
			w.dataPointsMu.Lock()
			w.dataPoints = append(points, w.dataPoints...)
			w.dataPointsMu.Unlock()
		} else {
			logger.Logger.Debug(fmt.Sprintf("成功批量插入 %d 条数据到 InfluxDB", len(points)))
		}
	}
}

// Close 关闭 InfluxWriter。
func (w *InfluxWriter) Close() error {
	if w.cancel != nil {
		w.cancel()
	}

	if w.flushTicker != nil {
		w.flushTicker.Stop()
	}
	close(w.flushStopChan)

	w.flushDataPoints()

	if w.client != nil {
		return w.client.Close()
	}

	logger.Logger.Info("InfluxWriter 已关闭")
	return nil
}
