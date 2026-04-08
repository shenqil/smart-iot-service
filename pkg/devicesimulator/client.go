package devicesimulator

import (
	"context"
	"log/slog"
	"math/rand"
	"net"
	"sync"
	"time"
)

type IoTDeviceClient struct {
	deviceID   uint32
	serverAddr string
	logger     *slog.Logger

	conn   net.Conn
	mu     sync.Mutex // 保护继电器状态
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup // 等待所有goroutine完成

	relayState [32]byte
	interval   time.Duration // 上报间隔
}

func NewIoTDeviceClient(deviceID uint32, addr string, logger *slog.Logger) *IoTDeviceClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &IoTDeviceClient{
		deviceID:   deviceID,
		serverAddr: addr,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		interval:   0,
	}
}

// Start 启动客户端（在goroutine中运行，非阻塞）
func (c *IoTDeviceClient) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				err := c.run()
				if err != nil {
					c.logger.Error("connection closed", "error", err)
				}
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(1 * time.Minute):
				}
			}
		}
	}()
}

func (c *IoTDeviceClient) run() error {
	conn, err := net.Dial("tcp", c.serverAddr)
	if err != nil {
		c.logger.Error("connect failed", "error", err)
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	c.logger.Info("connected")

	defer conn.Close()

	buf := make([]byte, 0)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.reportLoop()
	}()

	recv := make([]byte, 1024)
	for {
		select {
		case <-c.ctx.Done():
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, err := conn.Read(recv)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return err
			}

			c.logger.Debug("recv raw", "hex", fmtHex(recv[:n]))
			buf = append(buf, recv[:n]...)

			for {
				pkt, remain := FindPacket(buf)
				if pkt == nil {
					break
				}
				buf = remain
				c.handlePacket(pkt)
			}
		}
	}
}

func (c *IoTDeviceClient) handlePacket(pkt []byte) {
	p, err := ParsePacket(pkt)
	if err != nil {
		c.logger.Error("parse failed", "error", err)
		return
	}

	switch p.CMD {

	case 0x01: // 读取设备ID
		c.logger.Info("CMD 01 read id")
		c.replyDeviceID()

	case 0x03: // 设置上报间隔
		c.logger.Info("CMD 03 set interval")
		if len(p.Data) == 2 {
			min := int(p.Data[0])<<8 | int(p.Data[1])
			// 限制最少为 1 分钟
			if min < 1 {
				min = 1
			}
			c.interval = time.Duration(min) * time.Minute
			c.replyInterval(min)
		}

	case 0x02: // 控制继电器
		c.logger.Info("CMD 02 relay control")
		if len(p.Data) == 2 {
			idx := int(p.Data[0])
			val := p.Data[1]

			if idx >= 0 && idx < 32 {
				c.mu.Lock()
				c.relayState[idx] = val
				c.mu.Unlock()
			}

			c.replyRelay(idx, val)
		}
	}
}

// 回复设备 ID
func (c *IoTDeviceClient) replyDeviceID() {
	data := []byte{
		byte(c.deviceID >> 24),
		byte(c.deviceID >> 16),
		byte(c.deviceID >> 8),
		byte(c.deviceID),
	}
	c.send(0x01, data)
}

// 回复设置间隔
func (c *IoTDeviceClient) replyInterval(min int) {
	data := []byte{byte(min >> 8), byte(min)}
	c.send(0x03, data)
}

// 回复继电器状态
func (c *IoTDeviceClient) replyRelay(idx int, val byte) {
	data := []byte{byte(idx), val}
	c.send(0x02, data)
}

// 上报数据
func (c *IoTDeviceClient) reportLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if c.interval <= 0 {
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(time.Second):
				}
				continue
			}

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.interval):
				c.sendReport()
			}
		}
	}
}

func (c *IoTDeviceClient) sendReport() {
	data := make([]byte, 64)

	// 通道随机数据(16*2 字节)
	for i := 0; i < 16; i++ {
		v := uint16(rand.Intn(65535))
		data[i*2] = byte(v >> 8)
		data[i*2+1] = byte(v)
	}

	// 继电器状态(32字节)
	c.mu.Lock()
	copy(data[32:], c.relayState[:])
	c.mu.Unlock()

	c.send(0x04, data)
}

func (c *IoTDeviceClient) send(cmd byte, data []byte) {
	p := BuildPacket(cmd, data)
	c.logger.Debug("send raw", "hex", fmtHex(p))
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		conn.Write(p)
	}
}

// Stop 停止客户端并等待所有goroutine完成
func (c *IoTDeviceClient) Stop() {
	c.cancel()
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
	// 等待所有goroutine完成，设置超时避免无限等待
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// 所有goroutine已完成
	case <-time.After(5 * time.Second):
		// 超时，强制返回
		c.logger.Warn("等待goroutine完成超时")
	}
}
