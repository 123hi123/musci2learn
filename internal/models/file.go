package models

import (
	"time"
)

// FileStatus 檔案狀態
type FileStatus string

const (
	StatusUploaded   FileStatus = "uploaded"   // 已上傳
	StatusParsed     FileStatus = "parsed"     // 已解析歌詞
	StatusProcessing FileStatus = "processing" // 處理中
	StatusReady      FileStatus = "ready"      // 處理完成
	StatusError      FileStatus = "error"      // 錯誤
)

// FileSettings 檔案設定
type FileSettings struct {
	PrimaryLanguage         string `json:"primaryLanguage"`         // 主要語言 (en, zh)
	TTSRepeatCount          int    `json:"ttsRepeatCount"`          // TTS 重複次數
	StartLineIndex          int    `json:"startLineIndex"`          // 歌詞起點行索引
	ShowChineseTranslation  bool   `json:"showChineseTranslation"`  // 顯示中文翻譯
}

// MusicFile 音樂檔案
type MusicFile struct {
	ID          string       `json:"id"`
	Filename    string       `json:"filename"`
	Filepath    string       `json:"filepath"`
	Duration    float64      `json:"duration"`    // 秒
	UploadedAt  time.Time    `json:"uploadedAt"`
	Status      FileStatus   `json:"status"`
	Settings    FileSettings `json:"settings"`
	LyricCount  int          `json:"lyricCount"`  // 歌詞行數
	ErrorMsg    string       `json:"errorMsg,omitempty"`
	ProcessedAt *time.Time   `json:"processedAt,omitempty"`
}

// DefaultSettings 預設設定
func DefaultSettings() FileSettings {
	return FileSettings{
		PrimaryLanguage:        "en",
		TTSRepeatCount:         2,
		StartLineIndex:         0,
		ShowChineseTranslation: true,
	}
}

// FileListItem 檔案列表項目（簡化版）
type FileListItem struct {
	ID         string     `json:"id"`
	Filename   string     `json:"filename"`
	Duration   float64    `json:"duration"`
	Status     FileStatus `json:"status"`
	LyricCount int        `json:"lyricCount"`
	UploadedAt time.Time  `json:"uploadedAt"`
}

// ToListItem 轉換為列表項目
func (f *MusicFile) ToListItem() FileListItem {
	return FileListItem{
		ID:         f.ID,
		Filename:   f.Filename,
		Duration:   f.Duration,
		Status:     f.Status,
		LyricCount: f.LyricCount,
		UploadedAt: f.UploadedAt,
	}
}
