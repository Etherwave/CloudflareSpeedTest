package utils

import (
	"io"
	"log"
	"os"
)

var logger *log.Logger

// InitLog 初始化日志，如果 logFile 为空则输出到屏幕
func InitLog(logFile string) error {
	var writer io.Writer

	if logFile == "" {
		// 输出到屏幕（标准输出）
		writer = os.Stdout
	} else {
		// 打开日志文件（追加模式）
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		writer = file
	}

	// 创建日志器：带时间戳和文件名行号
	logger = log.New(writer, "", log.LstdFlags|log.Lshortfile)
	return nil
}

// 快捷日志函数
func LogInfo(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[INFO] "+format, v...)
	}
}

func LogError(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[ERROR] "+format, v...)
	}
}

func LogDebug(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[DEBUG] "+format, v...)
	}
}

func LogFatal(format string, v ...interface{}) {
	if logger != nil {
		logger.Fatalf("[FATAL] "+format, v...)
	}
}
