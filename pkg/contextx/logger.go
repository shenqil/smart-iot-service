package contextx

import (
	"context"
	"log/slog"
	"rbac-admin/pkg/logger"

	"github.com/gin-gonic/gin"
)

// loggerKey 是 context 中存储 Logger 的键
type loggerKeyType struct{}

var loggerKey = loggerKeyType{}

// SetLoggerToContext 将 Logger 存入 context.Context
// args 参数用于添加结构化日志字段，格式为 key1, value1, key2, value2...
func SetLoggerToContext(ctx context.Context, args ...interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	log := GetLoggerFromContext(ctx)

	// 只有当有附加字段时才创建新的 Logger 实例
	if len(args) > 0 {
		log = log.With(args...)
	}

	return context.WithValue(ctx, loggerKey, log)
}

// GetLoggerFromContext 从 context.Context 获取 Logger
// 如果不存在则返回全局 Logger
func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return logger.Logger
	}

	val := ctx.Value(loggerKey)
	if val == nil {
		return logger.Logger
	}

	log, ok := val.(*slog.Logger)
	if !ok {
		return logger.Logger
	}
	return log
}

// SetLoggerToRequestContext 将 Logger 存入 gin.Context 的 Request.Context
// args 参数用于添加结构化日志字段
func SetLoggerToRequestContext(c *gin.Context, args ...interface{}) {
	if c == nil || c.Request == nil {
		return
	}

	newCtx := SetLoggerToContext(c.Request.Context(), args...)
	c.Request = c.Request.WithContext(newCtx)
}

// GetLoggerFromRequestContext 从 gin.Context 的 Request.Context 获取 Logger
func GetLoggerFromRequestContext(c *gin.Context) *slog.Logger {
	if c == nil || c.Request == nil {
		return logger.Logger
	}
	return GetLoggerFromContext(c.Request.Context())
}
