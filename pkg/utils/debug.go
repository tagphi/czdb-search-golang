package utils

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	// DebugEnabled 控制调试信息是否输出，默认为false
	DebugEnabled bool = false
	
	// debugLock 保证并发安全
	debugLock sync.Mutex
	
	// DebugOutput 调试输出的目标，默认为标准输出
	DebugOutput io.Writer = os.Stdout
)

// SetDebugEnabled 设置是否启用调试输出
func SetDebugEnabled(enabled bool) {
	debugLock.Lock()
	defer debugLock.Unlock()
	DebugEnabled = enabled
}

// SetDebugOutput 设置调试输出的目标
func SetDebugOutput(output io.Writer) {
	debugLock.Lock()
	defer debugLock.Unlock()
	DebugOutput = output
}

// Debug 输出调试信息，仅当DebugEnabled为true时输出
func Debug(format string, args ...interface{}) {
	if DebugEnabled {
		debugLock.Lock()
		defer debugLock.Unlock()
		fmt.Fprintf(DebugOutput, format, args...)
	}
}

// Debugln 输出调试信息并换行，仅当DebugEnabled为true时输出
func Debugln(args ...interface{}) {
	if DebugEnabled {
		debugLock.Lock()
		defer debugLock.Unlock()
		fmt.Fprintln(DebugOutput, args...)
	}
}

// DebugfWithPrefix 输出带前缀的调试信息，仅当DebugEnabled为true时输出
func DebugfWithPrefix(prefix, format string, args ...interface{}) {
	if DebugEnabled {
		debugLock.Lock()
		defer debugLock.Unlock()
		fmt.Fprintf(DebugOutput, prefix+format, args...)
	}
}

// Warning 输出警告信息，无论DebugEnabled是什么值都会输出
func Warning(format string, args ...interface{}) {
	debugLock.Lock()
	defer debugLock.Unlock()
	fmt.Fprintf(DebugOutput, "Warning: "+format, args...)
}