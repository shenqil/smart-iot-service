// weather_service.go
package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"rbac-admin/internal/model/dto"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
)

type WeatherService struct {
	apiKey     string
	cache      map[string]weatherCacheItem
	cacheMutex sync.RWMutex
	client     *http.Client
}

type weatherCacheItem struct {
	data      *dto.TencentRealtime
	timestamp time.Time
}

func NewWeatherService() *WeatherService {
	// 创建自定义Transport跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	ws := &WeatherService{
		apiKey: config.Conf.App.WeatherAPIKey,
		cache:  make(map[string]weatherCacheItem),
		client: client,
	}

	// Start cache cleanup goroutine
	go ws.startCacheCleanup()

	return ws
}

func (ws *WeatherService) startCacheCleanup() {
	for {
		now := time.Now()
		// Calculate duration until next midnight
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		durationUntilMidnight := nextMidnight.Sub(now)

		time.Sleep(durationUntilMidnight)

		ws.cacheMutex.Lock()
		ws.cache = make(map[string]weatherCacheItem) // Clear cache
		ws.cacheMutex.Unlock()
	}
}

func (ws *WeatherService) GetWeather(ctx context.Context, lat, lon float64) (*dto.TencentRealtime, error) {

	logger := contextx.GetLoggerFromContext(ctx)

	cacheKey := fmt.Sprintf("%.4f,%.4f", lat, lon)

	// Check cache first
	ws.cacheMutex.RLock()
	cached, exists := ws.cache[cacheKey]
	ws.cacheMutex.RUnlock()

	if exists && time.Since(cached.timestamp) < 24*time.Hour {
		return cached.data, nil
	}

	// 请求URL
	url := fmt.Sprintf("https://apis.map.qq.com/ws/weather/v1?key=%s&location=%s", ws.apiKey, cacheKey)

	resp, err := ws.client.Get(url)
	if err != nil {
		logger.Error(fmt.Sprintf("天气API请求失败: %s, %v", url, err))
		return nil, errx.New(constant.InternalServerError, "天气API请求失败", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("天气API返回错误状态码: %s, %d", url, resp.StatusCode))
		return nil, errx.New(constant.InternalServerError, "天气API返回错误状态码", nil)
	}

	// 解析JSON响应
	var tencentResp dto.TencentWeatherResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("读取天气API响应失败: %s, %v", url, err))
		return nil, errx.New(constant.InternalServerError, "读取天气API响应失败", err)
	}

	if err := json.Unmarshal(body, &tencentResp); err != nil {
		logger.Error(fmt.Sprintf("解析天气API响应JSON失败: %s, %v", url, err))
		return nil, errx.New(constant.InternalServerError, "解析天气API响应失败", err)
	}

	// 检查API返回状态
	if tencentResp.Status != 0 {
		logger.Error(fmt.Sprintf("天气API返回错误: %s, %s", url, tencentResp.Message))
		return nil, errx.New(constant.InternalServerError, "天气API返回错误", nil)
	}

	// 只缓存第一个元素，如果没有则不缓存并返回nil
	realtimeData := tencentResp.Result.Realtime
	if len(realtimeData) == 0 {
		return nil, nil
	}

	firstRealtime := &realtimeData[0]

	// Cache the first result only
	ws.cacheMutex.Lock()
	ws.cache[cacheKey] = weatherCacheItem{
		data:      firstRealtime,
		timestamp: time.Now(),
	}
	ws.cacheMutex.Unlock()

	return firstRealtime, nil
}
