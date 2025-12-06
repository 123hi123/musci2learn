package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"multilang-learner/internal/services"

	"github.com/gin-gonic/gin"
)

// ===== 檔案管理 Handlers =====

func createListFilesHandler(fs *services.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		files, err := fs.List()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"files": files})
	}
}

func createUploadHandler(fs *services.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無法讀取上傳檔案"})
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法讀取檔案內容"})
			return
		}

		result, err := fs.Upload(header.Filename, data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func createGetFileHandler(fs *services.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		file, err := fs.Get(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "檔案不存在"})
			return
		}
		c.JSON(http.StatusOK, file)
	}
}

func createDeleteFileHandler(fs *services.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := fs.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "刪除成功"})
	}
}

func createUpdateSettingsHandler(fs *services.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var settings map[string]interface{}
		if err := c.ShouldBindJSON(&settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無效的設定資料"})
			return
		}
		if err := fs.UpdateSettings(id, settings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "設定已更新"})
	}
}

// ===== 歌詞 Handlers =====

func createGetLyricsHandler(ls *services.LyricService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		lyrics, err := ls.GetLyrics(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "無法取得歌詞"})
			return
		}
		c.JSON(http.StatusOK, lyrics)
	}
}

func createDetectStartHandler(ls *services.LyricService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		startLine, err := ls.DetectStartLine(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"startLineIndex": startLine})
	}
}

// ===== 處理 Handlers =====

func createStartProcessHandler(ps *services.ProcessService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var settings map[string]interface{}
		c.ShouldBindJSON(&settings)

		if err := ps.StartProcess(id, settings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "處理已開始"})
	}
}

func createGetProgressHandler(ps *services.ProcessService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		progress, err := ps.GetProgress(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "無法取得進度"})
			return
		}
		c.JSON(http.StatusOK, progress)
	}
}

func createGetSegmentsHandler(ps *services.ProcessService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		segments, err := ps.GetSegments(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "無法取得段落"})
			return
		}
		c.JSON(http.StatusOK, segments)
	}
}

// ===== 音訊 Handlers =====

func createGetAudioHandler(fs *services.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		filePath, err := fs.GetFilePath(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "檔案不存在"})
			return
		}
		c.File(filePath)
	}
}

func createGetSegmentAudioHandler(dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		segIdx, _ := strconv.Atoi(c.Param("segIdx"))

		// 使用三位數格式，例如 segment_000.mp3
		segmentPath := filepath.Join(dataDir, id, "segments", fmt.Sprintf("segment_%03d.mp3", segIdx))
		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "段落音訊不存在: " + segmentPath})
			return
		}
		c.File(segmentPath)
	}
}

func createGetSegmentTTSHandler(dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		segIdx, _ := strconv.Atoi(c.Param("segIdx"))

		// 使用三位數格式，例如 tts_000.mp3
		ttsPath := filepath.Join(dataDir, id, "tts", fmt.Sprintf("tts_%03d.mp3", segIdx))
		if _, err := os.Stat(ttsPath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "TTS 音訊不存在: " + ttsPath})
			return
		}
		c.File(ttsPath)
	}
}

// ===== 重新翻譯 Handler =====

func createRetranslateHandler(ps *services.ProcessService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		segIdx, err := strconv.Atoi(c.Param("segIdx"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無效的段落索引"})
			return
		}

		// 解析請求 body 取得用戶輸入
		var req struct {
			UserInput string `json:"userInput"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// 如果沒有 body 或解析失敗，使用空字串（向後相容）
			req.UserInput = ""
		}

		// 執行重新翻譯（傳入用戶輸入的原句）
		newTranslation, err := ps.RetranslateSegmentWithInput(id, segIdx, req.UserInput)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"translation":  newTranslation,
			"segmentIndex": segIdx,
		})
	}
}

// ===== 導出 Handlers =====

func createExportHandler(ps *services.ProcessService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		exportPath, err := ps.Export(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"path": exportPath})
	}
}

func createDownloadExportHandler(dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		exportPath := filepath.Join(dataDir, id, "export.mp3")
		if _, err := os.Stat(exportPath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "導出檔案不存在"})
			return
		}
		c.FileAttachment(exportPath, "export.mp3")
	}
}
