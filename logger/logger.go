package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger
var AccessWriter *os.File
var AccessFormatter = func(param gin.LogFormatterParams) string {
	return fmt.Sprintf(
		"[DMP] %s | %3d | %13v | %15s | %-7s %s\n",
		param.TimeStamp.Format("2006-01-02 15:04:05"),
		param.StatusCode,
		param.Latency,
		param.ClientIP,
		param.Method,
		param.Path,
	)
}

func InitLogger(level string) {
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.MkdirAll(logDir, os.ModePerm)
	}

	runtimeWriter, err := os.OpenFile(
		filepath.Join(logDir, "runtime.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		panic("无法创建 runtime 日志文件: " + err.Error())
	}

	AccessWriter, err = os.OpenFile(
		filepath.Join(logDir, "access.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		panic("无法创建 access 日志文件: " + err.Error())
	}

	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// ✅ 仅在 debug 显示 caller
	var encodeCaller zapcore.CallerEncoder
	if zapLevel == zapcore.DebugLevel {
		encodeCaller = zapcore.ShortCallerEncoder
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:    "time",
		LevelKey:   "level",
		CallerKey:  "caller",
		MessageKey: "msg",

		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},

		EncodeLevel: func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("[%s]", strings.ToUpper(l.String())))
		},

		EncodeCaller: encodeCaller,
	}

	runtimeCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(runtimeWriter),
		zapLevel,
	)

	Logger = zap.New(
		runtimeCore,
		zap.AddCaller(),
	).Sugar()

	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
}
