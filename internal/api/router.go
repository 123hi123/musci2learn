package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Router 路由器
type Router struct {
	engine         *gin.Engine
	fileService    FileServiceInterface
	lyricService   LyricServiceInterface
	processService ProcessServiceInterface
}

// FileServiceInterface 檔案服務介面
type FileServiceInterface interface {
	List() ([]interface{}, error)
	Get(id string) (interface{}, error)
	Upload(filename string, data []byte) (interface{}, error)
	Delete(id string) error
	UpdateSettings(id string, settings interface{}) error
	GetFilePath(id string) (string, error)
}

// LyricServiceInterface 歌詞服務介面
type LyricServiceInterface interface {
	GetLyrics(fileID string) (interface{}, error)
	DetectStartLine(fileID string) (int, error)
}

// ProcessServiceInterface 處理服務介面
type ProcessServiceInterface interface {
	StartProcess(fileID string, settings interface{}) error
	GetProgress(fileID string) (interface{}, error)
	GetSegments(fileID string) (interface{}, error)
	Export(fileID string) (string, error)
}

// NewRouter 建立新路由器
func NewRouter(fileService FileServiceInterface, lyricService LyricServiceInterface, processService ProcessServiceInterface) *Router {
	r := &Router{
		engine:         gin.Default(),
		fileService:    fileService,
		lyricService:   lyricService,
		processService: processService,
	}
	r.setupRoutes()
	return r
}

// setupRoutes 設定路由
func (r *Router) setupRoutes() {
	// 靜態檔案
	r.engine.Static("/static", "./web/static")
	r.engine.LoadHTMLGlob("web/templates/*")

	// 首頁
	r.engine.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// API 路由群組
	api := r.engine.Group("/api")
	{
		// 檔案管理
		files := api.Group("/files")
		{
			files.GET("", r.handleListFiles)
			files.POST("/upload", r.handleUploadFile)
			files.GET("/:id", r.handleGetFile)
			files.DELETE("/:id", r.handleDeleteFile)
			files.POST("/:id/settings", r.handleUpdateSettings)

			// 歌詞
			files.GET("/:id/lyrics", r.handleGetLyrics)
			files.POST("/:id/detect-start", r.handleDetectStart)

			// 處理
			files.POST("/:id/process", r.handleStartProcess)
			files.GET("/:id/status", r.handleGetProgress)
			files.GET("/:id/segments", r.handleGetSegments)

			// 音訊
			files.GET("/:id/audio", r.handleGetAudio)
			files.GET("/:id/segments/:segIdx/audio", r.handleGetSegmentAudio)
			files.GET("/:id/segments/:segIdx/tts", r.handleGetSegmentTTS)

			// 導出
			files.POST("/:id/export", r.handleExport)
			files.GET("/:id/export/download", r.handleDownloadExport)
		}
	}
}

// Run 啟動伺服器
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}

// Engine 獲取底層 Gin 引擎
func (r *Router) Engine() *gin.Engine {
	return r.engine
}
