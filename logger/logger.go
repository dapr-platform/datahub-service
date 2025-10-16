package logger

import (
	"log/slog"
	"os"
)

// InitLogger 初始化全局日志记录器
// 创建 JSON 格式的日志处理器,输出到 stdout
func InitLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
