package logger

import (
    "dst-management-platform-api/utils"
    "fmt"
    "log/slog"
    "os"
    "strings"

    "github.com/gin-gonic/gin"
)

var Logger *slog.Logger

func InitLogger() {
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

    // 设置 runtime 日志
    customTimeFormat := "2006-01-02 15:04:05"
    replaceTime := func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.TimeKey {
            t := a.Value.Time()
            a.Value = slog.StringValue(t.Format(customTimeFormat))
        }
        return a
    }

    var (
        level     slog.Level
        addSource bool
    )
    switch strings.ToLower(utils.LogLevel) {
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

    Logger = slog.New(slog.NewJSONHandler(slogLogFile, &slog.HandlerOptions{
        AddSource:   addSource,
        Level:       level,
        ReplaceAttr: replaceTime,
    }))
}
