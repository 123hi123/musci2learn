package main

import (
	"log"
	"os"

	"multilang-learner/internal/api"
	"multilang-learner/internal/logger"
	"multilang-learner/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化日誌系統
	logDir := "./logs"
	if err := logger.Init(logDir); err != nil {
		log.Printf("Warning: Failed to init logger: %v, using default logger", err)
	}
	defer logger.Close()

	// 設定目錄
	dataDir := "./data"
	uploadDir := "./uploads"

	// 確保目錄存在
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll("./web/static/css", 0755)
	os.MkdirAll("./web/static/js", 0755)
	os.MkdirAll("./web/templates", 0755)

	// 建立服務
	fileService := services.NewFileService(dataDir, uploadDir)
	lyricService := services.NewLyricService(dataDir, fileService)
	processService := services.NewProcessService(dataDir, fileService, lyricService)

	// 建立路由
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	// 中間件
	engine.Use(api.CORSMiddleware())
	engine.Use(api.LoggerMiddleware())
	engine.Use(api.ErrorHandlerMiddleware())

	// 靜態檔案
	engine.Static("/static", "./web/static")
	engine.LoadHTMLGlob("web/templates/*")

	// 首頁
	engine.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// API 路由
	apiGroup := engine.Group("/api")
	{
		// 檔案管理
		files := apiGroup.Group("/files")
		{
			files.GET("", createListFilesHandler(fileService))
			files.POST("/upload", createUploadHandler(fileService))
			files.GET("/:id", createGetFileHandler(fileService))
			files.DELETE("/:id", createDeleteFileHandler(fileService))
			files.POST("/:id/settings", createUpdateSettingsHandler(fileService))

			// 歌詞
			files.GET("/:id/lyrics", createGetLyricsHandler(lyricService))
			files.POST("/:id/detect-start", createDetectStartHandler(lyricService))

			// 處理
			files.POST("/:id/process", createStartProcessHandler(processService))
			files.GET("/:id/status", createGetProgressHandler(processService))
			files.GET("/:id/segments", createGetSegmentsHandler(processService))

			// 音訊
			files.GET("/:id/audio", createGetAudioHandler(fileService))
			files.GET("/:id/segments/:segIdx/audio", createGetSegmentAudioHandler(dataDir))
			files.GET("/:id/segments/:segIdx/tts", createGetSegmentTTSHandler(dataDir))

			// 重新翻譯
			files.POST("/:id/segments/:segIdx/retranslate", createRetranslateHandler(processService))

			// 導出
			files.POST("/:id/export", createExportHandler(processService))
			files.GET("/:id/export/download", createDownloadExportHandler(dataDir))
		}
	}

	// 啟動伺服器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🎵 多語言學習器啟動於 http://localhost:%s", port)
	if err := engine.Run(":" + port); err != nil {
		log.Fatal("伺服器啟動失敗:", err)
	}
}

// 以下是 Handler 建立函數...
