package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ===== 檔案管理 =====

// handleListFiles 獲取檔案列表
func (r *Router) handleListFiles(c *gin.Context) {
	files, err := r.fileService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": files})
}

// handleUploadFile 上傳檔案
func (r *Router) handleUploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法讀取上傳檔案"})
		return
	}
	defer file.Close()

	// 讀取檔案內容
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法讀取檔案內容"})
		return
	}

	// 儲存檔案
	result, err := r.fileService.Upload(header.Filename, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// handleGetFile 獲取檔案詳情
func (r *Router) handleGetFile(c *gin.Context) {
	id := c.Param("id")
	file, err := r.fileService.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "檔案不存在"})
		return
	}
	c.JSON(http.StatusOK, file)
}

// handleDeleteFile 刪除檔案
func (r *Router) handleDeleteFile(c *gin.Context) {
	id := c.Param("id")
	if err := r.fileService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "刪除成功"})
}

// handleUpdateSettings 更新檔案設定
func (r *Router) handleUpdateSettings(c *gin.Context) {
	id := c.Param("id")
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的設定資料"})
		return
	}
	if err := r.fileService.UpdateSettings(id, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "設定已更新"})
}

// ===== 歌詞 =====

// handleGetLyrics 獲取歌詞
func (r *Router) handleGetLyrics(c *gin.Context) {
	id := c.Param("id")
	lyrics, err := r.lyricService.GetLyrics(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "無法取得歌詞"})
		return
	}
	c.JSON(http.StatusOK, lyrics)
}

// handleDetectStart AI 判斷歌詞起點
func (r *Router) handleDetectStart(c *gin.Context) {
	id := c.Param("id")
	startLine, err := r.lyricService.DetectStartLine(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"startLineIndex": startLine})
}

// ===== 處理 =====

// handleStartProcess 開始處理
func (r *Router) handleStartProcess(c *gin.Context) {
	id := c.Param("id")
	var settings map[string]interface{}
	c.ShouldBindJSON(&settings) // 可選設定

	if err := r.processService.StartProcess(id, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "處理已開始"})
}

// handleGetProgress 獲取處理進度
func (r *Router) handleGetProgress(c *gin.Context) {
	id := c.Param("id")
	progress, err := r.processService.GetProgress(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "無法取得進度"})
		return
	}
	c.JSON(http.StatusOK, progress)
}

// handleGetSegments 獲取段落
func (r *Router) handleGetSegments(c *gin.Context) {
	id := c.Param("id")
	segments, err := r.processService.GetSegments(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "無法取得段落"})
		return
	}
	c.JSON(http.StatusOK, segments)
}

// ===== 音訊 =====

// handleGetAudio 獲取原始音訊
func (r *Router) handleGetAudio(c *gin.Context) {
	id := c.Param("id")
	filePath, err := r.fileService.GetFilePath(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "檔案不存在"})
		return
	}
	c.File(filePath)
}

// handleGetSegmentAudio 獲取段落音訊
func (r *Router) handleGetSegmentAudio(c *gin.Context) {
	id := c.Param("id")
	segIdx, _ := strconv.Atoi(c.Param("segIdx"))

	// 組合路徑
	segmentPath := filepath.Join("data", id, "segments", "segment_"+strconv.Itoa(segIdx)+".mp3")
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "段落音訊不存在"})
		return
	}
	c.File(segmentPath)
}

// handleGetSegmentTTS 獲取段落 TTS
func (r *Router) handleGetSegmentTTS(c *gin.Context) {
	id := c.Param("id")
	segIdx, _ := strconv.Atoi(c.Param("segIdx"))

	// 組合路徑
	ttsPath := filepath.Join("data", id, "tts", "tts_"+strconv.Itoa(segIdx)+".mp3")
	if _, err := os.Stat(ttsPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "TTS 音訊不存在"})
		return
	}
	c.File(ttsPath)
}

// ===== 導出 =====

// handleExport 開始導出
func (r *Router) handleExport(c *gin.Context) {
	id := c.Param("id")
	exportPath, err := r.processService.Export(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": exportPath})
}

// handleDownloadExport 下載導出檔案
func (r *Router) handleDownloadExport(c *gin.Context) {
	id := c.Param("id")
	exportPath := filepath.Join("data", id, "export.mp3")
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "導出檔案不存在"})
		return
	}
	c.FileAttachment(exportPath, "export.mp3")
}
