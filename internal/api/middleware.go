package api

import (
	"strings"
	"time"

	"multilang-learner/internal/logger"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware 日誌中間件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		method := c.Request.Method

		// 完整路徑（含查詢參數）
		fullPath := path
		if rawQuery != "" {
			fullPath = path + "?" + rawQuery
		}

		// 處理請求
		c.Next()

		// 記錄日誌
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()

		// 收集錯誤訊息
		var errorMsg string
		if len(c.Errors) > 0 {
			var errMsgs []string
			for _, e := range c.Errors {
				errMsgs = append(errMsgs, e.Error())
			}
			errorMsg = strings.Join(errMsgs, "; ")
		}

		// 使用結構化日誌
		logger.RequestLog(method, fullPath, status, latency, clientIP, errorMsg)
	}
}

// CORSMiddleware CORS 中間件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// ErrorHandlerMiddleware 錯誤處理中間件
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 檢查是否有錯誤
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				logger.Error("Request error: %v", e.Err)
			}
		}
	}
}
