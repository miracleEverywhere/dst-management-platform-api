package logger

import (
    "dst-management-platform-api/utils"
    "fmt"
    "log/slog"
    "os"
    "strings"
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
    logPath := fmt.Sprintf("%s/runtime.log", logDir)
    logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        panic("无法创建日志文件: " + err.Error())
    }

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

    Logger = slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
        AddSource:   addSource, // 记录错误位置，仅debug开启
        Level:       level,
        ReplaceAttr: replaceTime,
    }))
}
