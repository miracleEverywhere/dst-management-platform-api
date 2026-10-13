package logger

import (
	"fmt"
	"os"
	"runtime/debug"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// runtimePanicLogger 用于记录未被业务代码捕获的 panic 及其调用栈
	runtimePanicLogger *zap.Logger
	// restoreStdLog 恢复标准库 log 的原始输出
	restoreStdLog func()
)

// HandlePanic 捕获 main goroutine 中未被处理的 panic，把 panic 信息与调用栈写入 runtime 日志，
// 并在保持屏幕输出不变的前提下，以与 Go 运行时一致的退出码结束进程。
// 只应该在 main 入口处 defer 调用。
//
// 注意：recover 只能捕获当前 goroutine 的 panic；其他 goroutine 中未被处理的 panic
// 以及运行时的 fatal error，由 initRuntimeCapture 配置的崩溃输出兜底。
func HandlePanic() {
	r := recover()
	if r == nil {
		return
	}

	if runtimePanicLogger == nil {
		panic(r)
	}

	stack := debug.Stack()

	// 保持与 Go 运行时一致的屏幕输出，便于终端、docker logs 等位置查看
	_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n\n%s", r, stack)

	// 消息里保留换行，让 runtime 日志中的调用栈与 Go 运行时原始输出一样可读
	runtimePanicLogger.Error("panic: " + fmt.Sprintf("%v\n%s", r, stack))
	// zap 写入 os.File 默认没有缓冲，此处仍显式刷盘，保证异常退出前日志完整
	_ = runtimePanicLogger.Sync()

	// 与 Go 运行时未捕获 panic 的退出码保持一致
	os.Exit(2)
}

// Sync 刷盘 runtime 日志，在进程正常退出前调用
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// initRuntimeCapture 在 runtime 日志初始化完成后调用，
// 让未被业务代码捕获的 panic 与运行时致命错误也写入 runtime 日志。
func initRuntimeCapture() {
	runtimePanicLogger = Logger.Desugar()

	// 标准库 log（如 log.Fatal）默认写到 stderr，重定向到 zap，避免这些日志只出现在屏幕上
	if restoreStdLog == nil {
		if restore, err := zap.RedirectStdLogAt(runtimePanicLogger, zapcore.ErrorLevel); err == nil {
			restoreStdLog = restore
		}
	}

	// Go 运行时打印未捕获 panic 与 fatal error 时直接写文件描述符 2，
	// 绕过 os.Stderr 与所有 Go 日志设施，因此只能在这一层接管。
	// SetCrashOutput 会让运行时把崩溃输出额外写一份到 runtime 日志，屏幕输出保持不变；
	// 它由运行时自己完成写入，即使进程立刻退出也不会丢日志。
	if err := debug.SetCrashOutput(RuntimeWriter, debug.CrashOptions{}); err != nil {
		runtimePanicLogger.Warn("接管运行时崩溃输出失败，未捕获的 panic 可能不会写入 runtime 日志",
			zap.Error(err),
		)
	}
}
