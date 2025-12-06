package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 結構化日誌器
type Logger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	file        *os.File
	mu          sync.Mutex
}

var (
	instance *Logger
	once     sync.Once
)

// Init 初始化日誌系統
func Init(logDir string) error {
	var initErr error
	once.Do(func() {
		// 確保日誌目錄存在
		if err := os.MkdirAll(logDir, 0755); err != nil {
			initErr = err
			return
		}

		// 建立日誌檔案（按日期命名）
		logFileName := fmt.Sprintf("app_%s.log", time.Now().Format("2006-01-02"))
		logPath := filepath.Join(logDir, logFileName)

		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			initErr = err
			return
		}

		// 同時輸出到檔案和控制台
		multiWriter := io.MultiWriter(os.Stdout, file)

		instance = &Logger{
			infoLogger:  log.New(multiWriter, "INFO  ", log.Ldate|log.Ltime|log.Lshortfile),
			warnLogger:  log.New(multiWriter, "WARN  ", log.Ldate|log.Ltime|log.Lshortfile),
			errorLogger: log.New(multiWriter, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile),
			debugLogger: log.New(multiWriter, "DEBUG ", log.Ldate|log.Ltime|log.Lshortfile),
			file:        file,
		}
	})
	return initErr
}

// Close 關閉日誌檔案
func Close() {
	if instance != nil && instance.file != nil {
		instance.file.Close()
	}
}

// Info 記錄一般資訊
func Info(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		instance.infoLogger.Output(2, fmt.Sprintf(format, v...))
		instance.file.Sync() // 強制寫入磁碟
		instance.mu.Unlock()
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warn 記錄警告
func Warn(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		instance.warnLogger.Output(2, fmt.Sprintf(format, v...))
		instance.file.Sync() // 強制寫入磁碟
		instance.mu.Unlock()
	} else {
		log.Printf("[WARN] "+format, v...)
	}
}

// Error 記錄錯誤
func Error(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		instance.errorLogger.Output(2, fmt.Sprintf(format, v...))
		instance.file.Sync() // 強制寫入磁碟
		instance.mu.Unlock()
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Debug 記錄偵錯資訊
func Debug(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		instance.debugLogger.Output(2, fmt.Sprintf(format, v...))
		instance.file.Sync() // 強制寫入磁碟
		instance.mu.Unlock()
	} else {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// RequestLog 記錄 HTTP 請求
func RequestLog(method, path string, status int, latency time.Duration, clientIP string, errorMsg string) {
	// 根據狀態碼決定日誌等級
	var logFunc func(string, ...interface{})
	var icon string

	switch {
	case status >= 500:
		logFunc = Error
		icon = "❌"
	case status >= 400:
		logFunc = Warn
		icon = "⚠️"
	case status >= 300:
		logFunc = Info
		icon = "↪️"
	default:
		logFunc = Info
		icon = "✅"
	}

	msg := fmt.Sprintf("%s [%s] %s -> %d (%v) from %s", icon, method, path, status, latency, clientIP)
	if errorMsg != "" {
		msg += fmt.Sprintf(" | Error: %s", errorMsg)
	}
	logFunc(msg)
}

// APIError 記錄 API 錯誤詳情
func APIError(method, path string, status int, err error, details string) {
	Error("API Error: [%s] %s -> %d | %v | %s", method, path, status, err, details)
}
