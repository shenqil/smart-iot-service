package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"rbac-admin/pkg/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// 全局日志实例
var Logger *slog.Logger

// logDir 日志目录
var logDir string

// dailyRotatingWriter 按天轮换的日志写入器
// 结合 lumberjack 实现按天切割和自动清理
type dailyRotatingWriter struct {
	serviceName string
	logDir      string
	currentDate string
	currentFile *lumberjack.Logger
	mu          sync.Mutex
}

var _ io.Writer = (*dailyRotatingWriter)(nil)

// newDailyRotatingWriter 创建按天轮换的日志写入器
func newDailyRotatingWriter(serviceName, logDir string) *dailyRotatingWriter {
	today := time.Now().Format("20060102")
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.%s.log", serviceName, today))

	lj := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    0,     // 不按大小切割，只按时间
		MaxAge:     7,     // 保留7天，自动清理旧文件
		MaxBackups: 0,     // 不限制备份数量，由 MaxAge 控制
		LocalTime:  true,  // 使用本地时间
		Compress:   false, // 不压缩（因为每天都是新文件）
	}

	return &dailyRotatingWriter{
		serviceName: serviceName,
		logDir:      logDir,
		currentDate: today,
		currentFile: lj,
	}
}

// Write 实现 io.Writer 接口，检查日期变化并切换文件
func (w *dailyRotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查日期是否变化
	today := time.Now().Format("20060102")
	if today != w.currentDate {
		// 日期变化，切换到新文件
		w.currentDate = today
		newLogPath := filepath.Join(w.logDir, fmt.Sprintf("%s.%s.log", w.serviceName, today))

		// 创建新的 lumberjack logger
		w.currentFile = &lumberjack.Logger{
			Filename:   newLogPath,
			MaxSize:    0,
			MaxAge:     7,
			MaxBackups: 0,
			LocalTime:  true,
			Compress:   false,
		}
	}

	// 写入日志
	return w.currentFile.Write(p)
}

// InitLogger 初始化全局日志器
func InitLogger() {
	// 使用 NewServiceLogger 创建全局 logger
	Logger = NewServiceLogger("app")
	slog.SetDefault(Logger)
}

// parseLogLevel 将配置级别字符串转换为 slog.Level
func parseLogLevel(level string) slog.Level {
	levelMap := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	if l, ok := levelMap[strings.ToLower(level)]; ok {
		return l
	}
	return slog.LevelInfo // 默认级别
}

// NewServiceLogger 为服务创建独立的 logger
// serviceName 是服务名称，例如 "app", "iot_service", "control_service", "active_report_service"
func NewServiceLogger(serviceName string) *slog.Logger {
	conf := config.Conf.Log
	level := parseLogLevel(conf.Level)

	// 获取日志目录（如果 logDir 未初始化，使用配置中的值）
	serviceLogDir := logDir
	if serviceLogDir == "" {
		serviceLogDir = conf.OutputFile
		logDir = serviceLogDir // 保存到全局变量
	}

	// 确保日志目录存在
	if strings.ToLower(conf.Output) == "file" {
		if err := os.MkdirAll(serviceLogDir, 0755); err != nil {
			panic(fmt.Sprintf("创建日志目录失败: %v", err))
		}
	}

	// 创建 handler
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	if strings.ToLower(conf.Output) == "file" {
		// 使用 lumberjack 实现按天切割和自动清理
		writer := newDailyRotatingWriter(serviceName, serviceLogDir)
		handler = slog.NewTextHandler(writer, opts)
	} else {
		// 控制台输出
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// 创建 logger
	return slog.New(handler)
}
