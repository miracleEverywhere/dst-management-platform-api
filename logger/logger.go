package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

type CustomLogger struct {
	*slog.Logger
}

var Logger *CustomLogger

func InitLogger(logLevel string) {
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic("无法创建日志目录: " + err.Error())
		}
	}

	// 创建 runtime 日志文件
	slogLogPath := fmt.Sprintf("%s/runtime.log", logDir)
	slogLogFile, err := os.OpenFile(slogLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("无法创建 runtime 日志文件: " + err.Error())
	}

	// 创建 access 日志文件
	accessLogPath := fmt.Sprintf("%s/access.log", logDir)
	accessLogFile, err := os.OpenFile(accessLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("无法创建 access 日志文件: " + err.Error())
	}

	// 设置 access 的日志输出
	gin.DefaultWriter = accessLogFile      // 普通日志
	gin.DefaultErrorWriter = accessLogFile // 错误日志

	var (
		level     slog.Level
		addSource bool
	)
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
		addSource = true
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	slogLogger := slog.New(NewSimpleHandler(slogLogFile, &slog.HandlerOptions{
		AddSource: addSource,
		Level:     level,
	}))

	Logger = &CustomLogger{Logger: slogLogger}
}

// DebugF 格式化 Debug 日志
func (l *CustomLogger) DebugF(format string, args ...any) {
	if l.Enabled(context.Background(), slog.LevelDebug) {
		l.Debug(fmt.Sprintf(format, args...))
	}
}

// InfoF 格式化 Info 日志
func (l *CustomLogger) InfoF(format string, args ...any) {
	if l.Enabled(context.Background(), slog.LevelInfo) {
		l.Info(fmt.Sprintf(format, args...))
	}
}

// WarnF 格式化 Warn 日志
func (l *CustomLogger) WarnF(format string, args ...any) {
	if l.Enabled(context.Background(), slog.LevelWarn) {
		l.Warn(fmt.Sprintf(format, args...))
	}
}

// ErrorF 格式化 Error 日志
func (l *CustomLogger) ErrorF(format string, args ...any) {
	if l.Enabled(context.Background(), slog.LevelError) {
		l.Error(fmt.Sprintf(format, args...))
	}
}

type simpleHandler struct {
	opts *slog.HandlerOptions
	w    io.Writer
}

func (h *simpleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *simpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 简化实现，实际使用时可能需要更复杂的处理
	return &simpleHandler{
		opts: h.opts,
		w:    h.w,
	}
}

func (h *simpleHandler) WithGroup(name string) slog.Handler {
	// 简化实现，实际使用时可能需要更复杂的处理
	return &simpleHandler{
		opts: h.opts,
		w:    h.w,
	}
}

func (h *simpleHandler) Handle(ctx context.Context, r slog.Record) error {
	timeStr := r.Time.Format("2006-01-02 15:04:05")
	level := r.Level.String()
	msg := r.Message

	// 构建基础日志行
	var logLine string
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		// 只显示文件名和行号，不显示完整路径
		//file := filepath.Base(f.File)
		file := f.File
		logLine = fmt.Sprintf("%s [%s] %s:%d %s", timeStr, level, file, f.Line, msg)
	} else {
		logLine = fmt.Sprintf("%s [%s] %s", timeStr, level, msg)
	}

	// 添加附加属性
	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "" && attr.Value.String() != "" && attr.Key != slog.SourceKey {
			logLine += fmt.Sprintf(" %s=%v", attr.Key, attr.Value)
		}
		return true
	})

	logLine += "\n"
	_, err := h.w.Write([]byte(logLine))
	return err
}

func NewSimpleHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &simpleHandler{
		opts: opts,
		w:    w,
	}
}
