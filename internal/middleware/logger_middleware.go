package middleware

import (
	"context"
	"log/slog"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 确保RequestID存在
		requestID := c.GetHeader(constant.RequestIDKey)
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set(constant.RequestIDKey, requestID)
		}

		// 日志添加字段
		contextx.SetLoggerToRequestContext(c, slog.String("requestID", requestID))

		// 处理请求
		c.Next()

		// 收集日志数据
		latencyMs := time.Since(start).Milliseconds()
		statusCode := c.Writer.Status()

		// 获取 logger
		log := contextx.GetLoggerFromRequestContext(c)

		// 构建日志消息
		msg := "[GIN]"
		if c.Request.URL.RawQuery != "" {
			msg = c.Request.URL.Path + "?" + c.Request.URL.RawQuery
		} else {
			msg = c.Request.URL.Path
		}

		// 构建日志属性
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", statusCode),
			slog.Int64("latency_ms", latencyMs),
			slog.String("client_ip", c.ClientIP()),
		}

		// 如果有错误，额外追加错误信息
		if len(c.Errors) > 0 {
			attrs = append(attrs,
				slog.String("user_agent", c.Request.UserAgent()),
				slog.Any("errors", c.Errors.Errors()),
			)
			log.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
		} else {
			level := slog.LevelInfo
			if latencyMs > 5000 {
				level = slog.LevelWarn
			}
			log.LogAttrs(context.Background(), level, msg, attrs...)
		}
	}
}
