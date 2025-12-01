package services

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"multilang-learner/internal/models"
)

// LyricService 歌詞服務
type LyricService struct {
	dataDir     string
	fileService *FileService
}

// NewLyricService 建立歌詞服務
func NewLyricService(dataDir string, fileService *FileService) *LyricService {
	return &LyricService{
		dataDir:     dataDir,
		fileService: fileService,
	}
}

// GetLyrics 獲取歌詞
func (s *LyricService) GetLyrics(fileID string) (interface{}, error) {
	lyricsPath := filepath.Join(s.dataDir, fileID, "lyrics.json")
	data, err := os.ReadFile(lyricsPath)
	if err != nil {
		return nil, errors.New("歌詞不存在")
	}

	var lyrics models.LyricsData
	if err := json.Unmarshal(data, &lyrics); err != nil {
		return nil, err
	}

	return &lyrics, nil
}

// GetLyricsData 獲取歌詞資料（內部使用）
func (s *LyricService) GetLyricsData(fileID string) (*models.LyricsData, error) {
	result, err := s.GetLyrics(fileID)
	if err != nil {
		return nil, err
	}
	return result.(*models.LyricsData), nil
}

// SaveLyrics 儲存歌詞
func (s *LyricService) SaveLyrics(fileID string, lyrics *models.LyricsData) error {
	lyricsPath := filepath.Join(s.dataDir, fileID, "lyrics.json")
	data, err := json.MarshalIndent(lyrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(lyricsPath, data, 0644)
}

// DetectStartLine AI 判斷歌詞起點
func (s *LyricService) DetectStartLine(fileID string) (int, error) {
	lyrics, err := s.GetLyricsData(fileID)
	if err != nil {
		return 0, err
	}

	// 簡單的起點判斷邏輯
	// 跳過常見的元數據行（如歌名、作者等）
	for i, line := range lyrics.Lines {
		text := strings.ToLower(line.Original)

		// 跳過常見的元數據模式
		if strings.Contains(text, "作词") || strings.Contains(text, "作曲") ||
			strings.Contains(text, "lyrics") || strings.Contains(text, "composer") ||
			strings.Contains(text, "artist") || strings.Contains(text, "album") ||
			strings.Contains(text, "词：") || strings.Contains(text, "曲：") ||
			strings.Contains(text, "编曲") || strings.Contains(text, "混音") {
			continue
		}

		// 如果行太短（可能是標題或標記）
		if len(strings.TrimSpace(line.Original)) < 3 {
			continue
		}

		// 找到第一個看起來像歌詞的行
		return i, nil
	}

	return 0, nil
}

// UpdateTranslations 更新翻譯
func (s *LyricService) UpdateTranslations(fileID string, lang string, translations map[int]string) error {
	lyrics, err := s.GetLyricsData(fileID)
	if err != nil {
		return err
	}

	for idx, trans := range translations {
		if idx < len(lyrics.Lines) {
			switch lang {
			case "en":
				lyrics.Lines[idx].Translations.En = trans
			case "zh":
				lyrics.Lines[idx].Translations.Zh = trans
			}
		}
	}

	return s.SaveLyrics(fileID, lyrics)
}

// SetStartLine 設定起點行
func (s *LyricService) SetStartLine(fileID string, startLine int) error {
	lyrics, err := s.GetLyricsData(fileID)
	if err != nil {
		return err
	}

	lyrics.StartLineIndex = startLine

	// 更新每行的 IsSkipped 狀態
	for i := range lyrics.Lines {
		lyrics.Lines[i].IsSkipped = i < startLine
	}

	return s.SaveLyrics(fileID, lyrics)
}
