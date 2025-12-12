package services

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"multilang-learner/internal/models"
)

// generateID 生成隨機 ID
func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// FileService 檔案服務
type FileService struct {
	dataDir   string
	uploadDir string
	files     map[string]*models.MusicFile
	mu        sync.RWMutex
}

// NewFileService 建立檔案服務
func NewFileService(dataDir, uploadDir string) *FileService {
	// 確保目錄存在
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(uploadDir, 0755)

	fs := &FileService{
		dataDir:   dataDir,
		uploadDir: uploadDir,
		files:     make(map[string]*models.MusicFile),
	}

	// 載入已存在的檔案
	fs.loadExistingFiles()

	return fs
}

// loadExistingFiles 載入已存在的檔案
func (s *FileService) loadExistingFiles() {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			metaPath := filepath.Join(s.dataDir, entry.Name(), "meta.json")
			if data, err := os.ReadFile(metaPath); err == nil {
				var file models.MusicFile
				if json.Unmarshal(data, &file) == nil {
					s.files[file.ID] = &file
				}
			}
		}
	}
}

// List 列出所有檔案
func (s *FileService) List() ([]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []interface{}
	for _, f := range s.files {
		list = append(list, f.ToListItem())
	}
	return list, nil
}

// Get 獲取檔案
func (s *FileService) Get(id string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if file, ok := s.files[id]; ok {
		return file, nil
	}
	return nil, errors.New("檔案不存在")
}

// Upload 上傳檔案
func (s *FileService) Upload(filename string, data []byte) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 生成 ID
	id := generateID()

	// 建立檔案目錄
	fileDir := filepath.Join(s.dataDir, id)
	os.MkdirAll(fileDir, 0755)
	os.MkdirAll(filepath.Join(fileDir, "segments"), 0755)
	os.MkdirAll(filepath.Join(fileDir, "tts"), 0755)

	// 儲存原始檔案
	ext := filepath.Ext(filename)
	originalPath := filepath.Join(fileDir, "original"+ext)
	if err := os.WriteFile(originalPath, data, 0644); err != nil {
		return nil, err
	}

	// 獲取音訊時長
	duration := s.getAudioDuration(originalPath)

	// 建立檔案記錄
	file := &models.MusicFile{
		ID:         id,
		Filename:   filename,
		Filepath:   originalPath,
		Duration:   duration,
		UploadedAt: time.Now(),
		Status:     models.StatusUploaded,
		Settings:   models.DefaultSettings(),
	}

	// 嘗試解析歌詞
	lyrics, err := s.parseLyrics(originalPath)
	if err == nil && len(lyrics) > 0 {
		file.Status = models.StatusParsed
		file.LyricCount = len(lyrics)

		// 儲存歌詞
		lyricsData := &models.LyricsData{
			FileID: id,
			Lines:  lyrics,
		}
		s.saveLyrics(id, lyricsData)
	}

	// 儲存元數據
	s.saveFileMeta(file)
	s.files[id] = file

	return file, nil
}

// Delete 刪除檔案
func (s *FileService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.files[id]; !ok {
		return errors.New("檔案不存在")
	}

	// 刪除檔案目錄
	fileDir := filepath.Join(s.dataDir, id)
	os.RemoveAll(fileDir)

	delete(s.files, id)
	return nil
}

// UpdateSettings 更新設定
func (s *FileService) UpdateSettings(id string, settings interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, ok := s.files[id]
	if !ok {
		return errors.New("檔案不存在")
	}

	// 解析設定
	settingsMap, ok := settings.(map[string]interface{})
	if !ok {
		return errors.New("無效的設定格式")
	}

	if lang, ok := settingsMap["primaryLanguage"].(string); ok {
		file.Settings.PrimaryLanguage = lang
	}
	if count, ok := settingsMap["ttsRepeatCount"].(float64); ok {
		file.Settings.TTSRepeatCount = int(count)
	}
	if startLine, ok := settingsMap["startLineIndex"].(float64); ok {
		file.Settings.StartLineIndex = int(startLine)
	}
	if showChinese, ok := settingsMap["showChineseTranslation"].(bool); ok {
		file.Settings.ShowChineseTranslation = showChinese
	}

	s.saveFileMeta(file)
	return nil
}

// GetFilePath 獲取檔案路徑
func (s *FileService) GetFilePath(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if file, ok := s.files[id]; ok {
		return file.Filepath, nil
	}
	return "", errors.New("檔案不存在")
}

// GetFile 獲取檔案（內部使用）
func (s *FileService) GetFile(id string) (*models.MusicFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if file, ok := s.files[id]; ok {
		return file, nil
	}
	return nil, errors.New("檔案不存在")
}

// UpdateStatus 更新狀態
func (s *FileService) UpdateStatus(id string, status models.FileStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if file, ok := s.files[id]; ok {
		file.Status = status
		if status == models.StatusReady {
			now := time.Now()
			file.ProcessedAt = &now
		}
		s.saveFileMeta(file)
		return nil
	}
	return errors.New("檔案不存在")
}

// saveFileMeta 儲存檔案元數據
func (s *FileService) saveFileMeta(file *models.MusicFile) {
	metaPath := filepath.Join(s.dataDir, file.ID, "meta.json")
	data, _ := json.MarshalIndent(file, "", "  ")
	os.WriteFile(metaPath, data, 0644)
}

// saveLyrics 儲存歌詞
func (s *FileService) saveLyrics(id string, lyrics *models.LyricsData) {
	lyricsPath := filepath.Join(s.dataDir, id, "lyrics.json")
	data, _ := json.MarshalIndent(lyrics, "", "  ")
	os.WriteFile(lyricsPath, data, 0644)
}

// getAudioDuration 獲取音訊時長
func (s *FileService) getAudioDuration(filePath string) float64 {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	duration, _ := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	return duration
}

// parseLyrics 解析歌詞
func (s *FileService) parseLyrics(filePath string) ([]models.LyricLine, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 根據檔案類型選擇不同的提取方式
	var output []byte
	var err error

	switch ext {
	case ".flac":
		// FLAC 檔案：使用 format_tags=lyrics
		cmd := exec.Command("ffprobe",
			"-v", "error",
			"-show_entries", "format_tags=lyrics",
			"-of", "default=noprint_wrappers=1:nokey=1",
			filePath,
		)
		output, err = cmd.Output()

	case ".mp3":
		// MP3 檔案：嘗試多種方式提取歌詞
		// 方式 1: 使用 format_tags=lyrics
		cmd := exec.Command("ffprobe",
			"-v", "error",
			"-show_entries", "format_tags=lyrics",
			"-of", "default=noprint_wrappers=1:nokey=1",
			filePath,
		)
		output, err = cmd.Output()

		// 如果 format_tags=lyrics 失敗或為空，嘗試 USLT (Unsynchronized Lyrics)
		if err != nil || len(bytes.TrimSpace(output)) == 0 {
			cmd = exec.Command("ffprobe",
				"-v", "error",
				"-show_entries", "format_tags=lyrics-xxx", // ID3v2 USLT tag
				"-of", "default=noprint_wrappers=1:nokey=1",
				filePath,
			)
			output, err = cmd.Output()
		}

		// 如果還是失敗，嘗試讀取所有 format_tags
		if err != nil || len(bytes.TrimSpace(output)) == 0 {
			cmd = exec.Command("ffprobe",
				"-v", "error",
				"-show_entries", "format_tags",
				"-of", "json",
				filePath,
			)
			jsonOutput, jsonErr := cmd.Output()
			if jsonErr == nil {
				output = s.extractLyricsFromJSON(jsonOutput)
			}
		}

	default:
		// 其他格式：使用通用方式
		cmd := exec.Command("ffprobe",
			"-v", "error",
			"-show_entries", "format_tags=lyrics",
			"-of", "default=noprint_wrappers=1:nokey=1",
			filePath,
		)
		output, err = cmd.Output()
	}

	if err != nil || len(bytes.TrimSpace(output)) == 0 {
		return nil, errors.New("無法提取歌詞，請確認音檔包含內嵌 LRC 歌詞")
	}

	// 解析 LRC 格式
	return s.parseLRC(string(output))
}

// extractLyricsFromJSON 從 ffprobe JSON 輸出中提取歌詞
func (s *FileService) extractLyricsFromJSON(jsonData []byte) []byte {
	// 嘗試解析 JSON 並找到歌詞相關的標籤
	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil
	}

	// 檢查各種可能的歌詞標籤名稱
	lyricsKeys := []string{
		"lyrics", "LYRICS",
		"lyrics-XXX", "LYRICS-XXX",
		"USLT", "uslt",
		"SYLT", "sylt",
		"unsyncedlyrics", "UNSYNCEDLYRICS",
		"syncedlyrics", "SYNCEDLYRICS",
	}

	for _, key := range lyricsKeys {
		if lyrics, ok := result.Format.Tags[key]; ok && len(lyrics) > 0 {
			return []byte(lyrics)
		}
	}

	// 遍歷所有 tags 尋找包含 "[" 開頭的 LRC 內容
	for _, value := range result.Format.Tags {
		if strings.Contains(value, "[") && strings.Contains(value, "]") {
			// 看起來像 LRC 格式
			return []byte(value)
		}
	}

	return nil
}

// parseLRC 解析 LRC 格式歌詞
// 支援格式：日文行 + 中文翻譯行（相同時間戳）
func (s *FileService) parseLRC(content string) ([]models.LyricLine, error) {
	rawLines := strings.Split(content, "\n")

	// 第一遍：解析所有行
	type parsedLine struct {
		timestamp string
		startTime float64
		text      string
	}
	var parsed []parsedLine

	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "[") {
			continue
		}

		// 跳過元數據標籤
		if strings.HasPrefix(line, "[ti:") || strings.HasPrefix(line, "[ar:") ||
			strings.HasPrefix(line, "[al:") || strings.HasPrefix(line, "[by:") ||
			strings.HasPrefix(line, "[offset:") || strings.HasPrefix(line, "[kana:") {
			continue
		}

		// 解析時間戳 [mm:ss.xx]
		closeBracket := strings.Index(line, "]")
		if closeBracket == -1 {
			continue
		}

		timestamp := line[1:closeBracket]
		text := strings.TrimSpace(line[closeBracket+1:])

		// 跳過空行和純符號行
		if text == "" || text == "//" {
			continue
		}

		startTime := s.parseTimestamp(timestamp)
		parsed = append(parsed, parsedLine{
			timestamp: timestamp,
			startTime: startTime,
			text:      text,
		})
	}

	// 第二遍：合併相同時間戳的行（日文 + 中文翻譯）
	var lines []models.LyricLine
	index := 0

	for i := 0; i < len(parsed); i++ {
		p := parsed[i]

		var original, embedded string
		original = p.text

		// 檢查下一行是否是相同時間戳（翻譯行）
		if i+1 < len(parsed) && parsed[i+1].startTime == p.startTime {
			// 判斷哪個是原文，哪個是翻譯
			// 通常中文翻譯包含漢字
			if s.isChinese(parsed[i+1].text) && !s.isChinese(p.text) {
				embedded = parsed[i+1].text
			} else if s.isChinese(p.text) && !s.isChinese(parsed[i+1].text) {
				original = parsed[i+1].text
				embedded = p.text
			} else {
				// 兩者都是同語言，保持原順序
				embedded = parsed[i+1].text
			}
			i++ // 跳過下一行（已處理）
		}

		lyricLine := models.LyricLine{
			Index:     index,
			Timestamp: p.timestamp,
			StartTime: p.startTime,
			Original:  original,
			Translations: models.Translations{
				Embedded: embedded,
				Zh:       embedded, // 假設內嵌翻譯是中文
			},
			IsMeaningful: len(strings.TrimSpace(original)) > 0 && original != "//" && !s.isMetadataLine(original),
		}

		lines = append(lines, lyricLine)
		index++
	}

	// 計算結束時間
	for i := 0; i < len(lines)-1; i++ {
		lines[i].EndTime = lines[i+1].StartTime
	}
	if len(lines) > 0 {
		lines[len(lines)-1].EndTime = lines[len(lines)-1].StartTime + 5
	}

	return lines, nil
}

// isChinese 判斷是否包含中文字符
func (s *FileService) isChinese(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// isMetadataLine 判斷是否是元數據行
func (s *FileService) isMetadataLine(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "作词") || strings.Contains(text, "作曲") ||
		strings.Contains(text, "词：") || strings.Contains(text, "曲：") ||
		strings.Contains(text, "编曲") || strings.Contains(text, "lyrics") ||
		strings.HasPrefix(text, "lemon -") || strings.Contains(text, " - ")
}

// parseTimestamp 解析時間戳
func (s *FileService) parseTimestamp(ts string) float64 {
	// 格式: mm:ss.xx 或 mm:ss:xx
	ts = strings.Replace(ts, ":", ".", 1) // 只替換第一個 : 為 .
	parts := strings.Split(ts, ".")
	if len(parts) < 2 {
		return 0
	}

	minutes, _ := strconv.ParseFloat(parts[0], 64)
	seconds, _ := strconv.ParseFloat(parts[1], 64)

	var ms float64
	if len(parts) >= 3 {
		ms, _ = strconv.ParseFloat(parts[2], 64)
		if len(parts[2]) == 2 {
			ms = ms / 100
		} else if len(parts[2]) == 3 {
			ms = ms / 1000
		}
	}

	return minutes*60 + seconds + ms
}
