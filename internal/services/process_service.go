package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"multilang-learner/internal/audio"
	"multilang-learner/internal/models"
	"multilang-learner/internal/translator"
	"multilang-learner/internal/tts"
)

// ProcessService 處理服務
type ProcessService struct {
	dataDir      string
	fileService  *FileService
	lyricService *LyricService
	progress     map[string]*models.ProcessProgress
	mu           sync.RWMutex
	apiKey       string
}

// NewProcessService 建立處理服務
func NewProcessService(dataDir string, fileService *FileService, lyricService *LyricService) *ProcessService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	return &ProcessService{
		dataDir:      dataDir,
		fileService:  fileService,
		lyricService: lyricService,
		progress:     make(map[string]*models.ProcessProgress),
		apiKey:       apiKey,
	}
}

// StartProcess 開始處理
func (s *ProcessService) StartProcess(fileID string, settings interface{}) error {
	file, err := s.fileService.GetFile(fileID)
	if err != nil {
		return err
	}

	// 初始化進度
	s.mu.Lock()
	s.progress[fileID] = &models.ProcessProgress{
		FileID:      fileID,
		Status:      "starting",
		Progress:    0,
		Message:     "準備中...",
		TotalSteps:  4,
		CurrentStep: 0,
	}
	s.mu.Unlock()

	// 異步處理
	go s.process(file)

	return nil
}

// process 處理流程
func (s *ProcessService) process(file *models.MusicFile) {
	fileID := file.ID

	// 更新狀態
	s.fileService.UpdateStatus(fileID, models.StatusProcessing)

	// 同步 startLineIndex 到 lyrics.json（確保 isSkipped 也被更新）
	if err := s.lyricService.SetStartLine(fileID, file.Settings.StartLineIndex); err != nil {
		s.setError(fileID, "同步起始行失敗: "+err.Error())
		return
	}

	// Step 1: 翻譯
	s.updateProgress(fileID, "translating", 1, 25, "翻譯歌詞中...")
	if err := s.translateLyrics(fileID, file.Settings.PrimaryLanguage); err != nil {
		s.setError(fileID, "翻譯失敗: "+err.Error())
		return
	}

	// Step 2: 分割段落
	s.updateProgress(fileID, "segmenting", 2, 50, "分割音訊段落...")
	if err := s.createSegments(fileID, file); err != nil {
		s.setError(fileID, "分割失敗: "+err.Error())
		return
	}

	// Step 3: 生成 TTS
	s.updateProgress(fileID, "generating_tts", 3, 75, "生成 TTS 語音...")
	if err := s.generateTTS(fileID, file.Settings.PrimaryLanguage); err != nil {
		s.setError(fileID, "TTS 生成失敗: "+err.Error())
		return
	}

	// Step 4: 完成
	s.updateProgress(fileID, "done", 4, 100, "處理完成！")
	s.fileService.UpdateStatus(fileID, models.StatusReady)
}

// translateLyrics 翻譯歌詞
func (s *ProcessService) translateLyrics(fileID, targetLang string) error {
	lyrics, err := s.lyricService.GetLyricsData(fileID)
	if err != nil {
		return err
	}

	// 如果有 API key，使用真正的翻譯
	var trans *translator.GeminiTranslator
	if s.apiKey != "" {
		trans, err = translator.NewGeminiTranslator(s.apiKey, false)
		if err != nil {
			return fmt.Errorf("建立翻譯器失敗: %w", err)
		}
	}

	ctx := context.Background()
	totalLines := 0
	for _, line := range lyrics.Lines {
		if line.IsMeaningful && !line.IsSkipped {
			totalLines++
		}
	}

	processedLines := 0
	for i := range lyrics.Lines {
		if !lyrics.Lines[i].IsMeaningful || lyrics.Lines[i].IsSkipped {
			continue
		}

		processedLines++
		progress := 25.0 + (float64(processedLines)/float64(totalLines))*25.0
		s.updateProgress(fileID, "translating", 1, progress,
			fmt.Sprintf("翻譯中... (%d/%d)", processedLines, totalLines))

		// 如果目標語言是英文
		if targetLang == "en" && lyrics.Lines[i].Translations.En == "" {
			// 優先使用內嵌的中文翻譯來翻譯成英文
			sourceText := lyrics.Lines[i].Original
			if lyrics.Lines[i].Translations.Embedded != "" {
				sourceText = lyrics.Lines[i].Translations.Embedded
			}

			if trans != nil {
				translated, err := trans.TranslateLyric(ctx, sourceText, "English")
				if err == nil {
					lyrics.Lines[i].Translations.En = translated
				} else {
					// API 失敗時使用原文
					lyrics.Lines[i].Translations.En = sourceText
				}
				// 稍微延遲避免 API 限流
				time.Sleep(100 * time.Millisecond)
			} else {
				// 沒有 API key，使用中文翻譯或原文
				if lyrics.Lines[i].Translations.Embedded != "" {
					lyrics.Lines[i].Translations.En = lyrics.Lines[i].Translations.Embedded
				} else {
					lyrics.Lines[i].Translations.En = sourceText
				}
			}
		}

		// 如果目標語言是中文
		if targetLang == "zh" {
			// 優先使用內嵌翻譯
			if lyrics.Lines[i].Translations.Zh == "" && lyrics.Lines[i].Translations.Embedded != "" {
				lyrics.Lines[i].Translations.Zh = lyrics.Lines[i].Translations.Embedded
			}
		}
	}

	return s.lyricService.SaveLyrics(fileID, lyrics)
}

// createSegments 建立段落
func (s *ProcessService) createSegments(fileID string, file *models.MusicFile) error {
	lyrics, err := s.lyricService.GetLyricsData(fileID)
	if err != nil {
		return err
	}

	var segments []models.Segment
	segmentDir := filepath.Join(s.dataDir, fileID, "segments")
	os.MkdirAll(segmentDir, 0755)

	// 獲取有效歌詞
	activeLyrics := lyrics.GetActiveLyrics()
	if len(activeLyrics) == 0 {
		return errors.New("沒有有效的歌詞")
	}

	// 合併段落（最小 5 秒）
	const minDuration = 5.0
	var currentSegment *models.Segment
	segIndex := 0

	for _, line := range activeLyrics {
		if currentSegment == nil {
			currentSegment = &models.Segment{
				Index:        segIndex,
				StartTime:    line.StartTime,
				EndTime:      line.EndTime,
				LineIndices:  []int{line.Index},
				IsMeaningful: true,
			}
		} else {
			// 合併到當前段落
			currentSegment.EndTime = line.EndTime
			currentSegment.LineIndices = append(currentSegment.LineIndices, line.Index)
		}

		// 檢查是否達到最小時長
		currentSegment.Duration = currentSegment.EndTime - currentSegment.StartTime
		if currentSegment.Duration >= minDuration {
			// 生成段落文字
			s.generateSegmentText(currentSegment, lyrics.Lines, file.Settings.PrimaryLanguage)

			// 切割音訊
			audioPath := filepath.Join(segmentDir, fmt.Sprintf("segment_%03d.mp3", segIndex))
			if err := s.cutAudio(file.Filepath, audioPath, currentSegment.StartTime, currentSegment.EndTime); err != nil {
				return err
			}
			currentSegment.AudioPath = audioPath

			segments = append(segments, *currentSegment)
			currentSegment = nil
			segIndex++
		}
	}

	// 處理最後一個段落
	if currentSegment != nil {
		s.generateSegmentText(currentSegment, lyrics.Lines, file.Settings.PrimaryLanguage)
		audioPath := filepath.Join(segmentDir, fmt.Sprintf("segment_%03d.mp3", segIndex))
		if err := s.cutAudio(file.Filepath, audioPath, currentSegment.StartTime, currentSegment.EndTime); err != nil {
			return err
		}
		currentSegment.AudioPath = audioPath
		segments = append(segments, *currentSegment)
	}

	// 儲存段落資料
	segmentsData := &models.SegmentsData{
		FileID:   fileID,
		Segments: segments,
		Language: file.Settings.PrimaryLanguage,
	}
	return s.saveSegments(fileID, segmentsData)
}

// generateSegmentText 生成段落文字
func (s *ProcessService) generateSegmentText(seg *models.Segment, lines []models.LyricLine, lang string) {
	var originals, translations []string
	for _, idx := range seg.LineIndices {
		if idx < len(lines) {
			originals = append(originals, lines[idx].Original)
			switch lang {
			case "en":
				if lines[idx].Translations.En != "" {
					translations = append(translations, lines[idx].Translations.En)
				}
			case "zh":
				if lines[idx].Translations.Zh != "" {
					translations = append(translations, lines[idx].Translations.Zh)
				} else if lines[idx].Translations.Embedded != "" {
					translations = append(translations, lines[idx].Translations.Embedded)
				}
			}
		}
	}
	seg.OriginalText = strings.Join(originals, "\n")
	seg.TTSText = strings.Join(translations, " ")
}

// cutAudio 切割音訊
func (s *ProcessService) cutAudio(inputPath, outputPath string, start, end float64) error {
	duration := end - start
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputPath,
		"-ss", strconv.FormatFloat(start, 'f', 3, 64),
		"-t", strconv.FormatFloat(duration, 'f', 3, 64),
		"-acodec", "libmp3lame",
		"-b:a", "192k",
		outputPath,
	)
	return cmd.Run()
}

// generateTTS 生成 TTS
func (s *ProcessService) generateTTS(fileID, lang string) error {
	segments, err := s.GetSegmentsData(fileID)
	if err != nil {
		return err
	}

	ttsDir := filepath.Join(s.dataDir, fileID, "tts")
	os.MkdirAll(ttsDir, 0755)

	// 建立 TTS 生成器
	var ttsGen *tts.GeminiTTS
	if s.apiKey != "" {
		ttsGen, err = tts.NewGeminiTTS(s.apiKey, false)
		if err != nil {
			return fmt.Errorf("建立 TTS 失敗: %w", err)
		}
	}

	// 建立音訊處理器（用於音量匹配）
	audioProcessor := audio.NewProcessor(false)

	ctx := context.Background()
	totalSegments := 0
	for _, seg := range segments.Segments {
		if seg.TTSText != "" {
			totalSegments++
		}
	}

	processedSegments := 0
	for i, seg := range segments.Segments {
		if seg.TTSText == "" {
			continue
		}

		processedSegments++
		progress := 75.0 + (float64(processedSegments)/float64(totalSegments))*20.0
		s.updateProgress(fileID, "generating_tts", 3, progress,
			fmt.Sprintf("生成 TTS... (%d/%d)", processedSegments, totalSegments))

		ttsPath := filepath.Join(ttsDir, fmt.Sprintf("tts_%03d.mp3", i))
		ttsTempPath := filepath.Join(ttsDir, fmt.Sprintf("tts_%03d_temp.mp3", i))
		segments.Segments[i].TTSPath = ttsPath

		if ttsGen != nil {
			// 使用真正的 TTS API 生成到暫存檔案
			err := ttsGen.GenerateSpeech(ctx, seg.TTSText, ttsTempPath)
			if err != nil {
				// TTS 失敗時，嘗試生成靜音檔案作為佔位
				s.generateSilence(ttsPath, 2.0)
			} else {
				// TTS 成功，進行音量匹配
				segmentAudioPath := seg.AudioPath
				if segmentAudioPath != "" {
					// 音量匹配：讓 TTS 音量與原曲段落一致
					err = audioProcessor.MatchVolume(segmentAudioPath, ttsTempPath, ttsPath)
					if err != nil {
						// 音量匹配失敗，直接使用原始 TTS
						os.Rename(ttsTempPath, ttsPath)
					} else {
						// 刪除暫存檔案
						os.Remove(ttsTempPath)
					}
				} else {
					// 沒有段落音訊，直接使用原始 TTS
					os.Rename(ttsTempPath, ttsPath)
				}
			}
			// 延遲避免 API 限流
			time.Sleep(500 * time.Millisecond)
		} else {
			// 沒有 API key，生成靜音檔案
			s.generateSilence(ttsPath, 2.0)
		}
	}

	return s.saveSegments(fileID, segments)
}

// generateSilence 生成靜音音訊
func (s *ProcessService) generateSilence(outputPath string, duration float64) error {
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", duration),
		"-acodec", "libmp3lame",
		"-b:a", "128k",
		outputPath,
	)
	return cmd.Run()
}

// GetProgress 獲取進度
func (s *ProcessService) GetProgress(fileID string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if progress, ok := s.progress[fileID]; ok {
		return progress, nil
	}
	return nil, errors.New("無處理進度")
}

// GetSegments 獲取段落
func (s *ProcessService) GetSegments(fileID string) (interface{}, error) {
	return s.GetSegmentsData(fileID)
}

// GetSegmentsData 獲取段落資料（內部使用）
func (s *ProcessService) GetSegmentsData(fileID string) (*models.SegmentsData, error) {
	segmentsPath := filepath.Join(s.dataDir, fileID, "segments.json")
	data, err := os.ReadFile(segmentsPath)
	if err != nil {
		return nil, errors.New("段落資料不存在")
	}

	var segments models.SegmentsData
	if err := json.Unmarshal(data, &segments); err != nil {
		return nil, err
	}

	return &segments, nil
}

// saveSegments 儲存段落
func (s *ProcessService) saveSegments(fileID string, segments *models.SegmentsData) error {
	segmentsPath := filepath.Join(s.dataDir, fileID, "segments.json")
	data, err := json.MarshalIndent(segments, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(segmentsPath, data, 0644)
}

// Export 導出合併音檔
func (s *ProcessService) Export(fileID string) (string, error) {
	file, err := s.fileService.GetFile(fileID)
	if err != nil {
		return "", err
	}

	segments, err := s.GetSegmentsData(fileID)
	if err != nil {
		return "", err
	}

	// 建立合併列表
	exportDir := filepath.Join(s.dataDir, fileID)
	listPath := filepath.Join(exportDir, "concat_list.txt")
	var listContent strings.Builder

	for _, seg := range segments.Segments {
		// 原曲段落
		if seg.AudioPath != "" {
			listContent.WriteString(fmt.Sprintf("file '%s'\n", seg.AudioPath))
		}
		// TTS（根據重複次數）
		if seg.TTSPath != "" {
			for i := 0; i < file.Settings.TTSRepeatCount; i++ {
				listContent.WriteString(fmt.Sprintf("file '%s'\n", seg.TTSPath))
			}
		}
	}

	if err := os.WriteFile(listPath, []byte(listContent.String()), 0644); err != nil {
		return "", err
	}

	// 合併音檔
	exportPath := filepath.Join(exportDir, "export.mp3")
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c", "copy",
		exportPath,
	)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return exportPath, nil
}

// updateProgress 更新進度
func (s *ProcessService) updateProgress(fileID, status string, step int, progress float64, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p, ok := s.progress[fileID]; ok {
		p.Status = status
		p.CurrentStep = step
		p.Progress = progress
		p.Message = message
	}
}

// setError 設定錯誤
func (s *ProcessService) setError(fileID, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p, ok := s.progress[fileID]; ok {
		p.Status = "error"
		p.Message = message
	}
	s.fileService.UpdateStatus(fileID, models.StatusError)
}

// RetranslateSegment 重新翻譯指定段落
// 使用更強的提示詞重新翻譯，並更新 segments.json 和重新生成 TTS
func (s *ProcessService) RetranslateSegment(fileID string, segmentIndex int) (string, error) {
	// 檢查 API key
	if s.apiKey == "" {
		return "", errors.New("GEMINI_API_KEY 未設定")
	}

	// 取得段落資料
	segments, err := s.GetSegmentsData(fileID)
	if err != nil {
		return "", fmt.Errorf("無法取得段落資料: %w", err)
	}

	// 檢查段落索引
	if segmentIndex < 0 || segmentIndex >= len(segments.Segments) {
		return "", fmt.Errorf("無效的段落索引: %d", segmentIndex)
	}

	seg := &segments.Segments[segmentIndex]

	// 取得歌詞資料以獲取中文翻譯
	lyricsData, err := s.lyricService.GetLyrics(fileID)
	if err != nil {
		return "", fmt.Errorf("無法取得歌詞資料: %w", err)
	}

	// 轉換為 LyricsData 型別
	lyrics, ok := lyricsData.(*models.LyricsData)
	if !ok {
		return "", fmt.Errorf("歌詞資料格式錯誤")
	}

	// 收集該段落的中文翻譯
	var chineseTexts []string
	for _, lineIdx := range seg.LineIndices {
		if lineIdx < len(lyrics.Lines) {
			line := lyrics.Lines[lineIdx]
			if line.Translations.Zh != "" {
				chineseTexts = append(chineseTexts, line.Translations.Zh)
			} else if line.Translations.Embedded != "" {
				chineseTexts = append(chineseTexts, line.Translations.Embedded)
			}
		}
	}
	chineseText := strings.Join(chineseTexts, " ")

	// 建立翻譯器
	trans, err := translator.NewGeminiTranslator(s.apiKey, true)
	if err != nil {
		return "", fmt.Errorf("建立翻譯器失敗: %w", err)
	}

	// 執行重新翻譯
	ctx := context.Background()
	newTranslation, err := trans.RetranslateLyric(ctx, seg.OriginalText, chineseText, "en")
	if err != nil {
		return "", fmt.Errorf("重新翻譯失敗: %w", err)
	}

	// 更新段落的 TTS 文字
	seg.TTSText = newTranslation

	// 儲存更新後的段落資料
	if err := s.saveSegments(fileID, segments); err != nil {
		return "", fmt.Errorf("儲存段落失敗: %w", err)
	}

	// 重新生成該段落的 TTS
	ttsDir := filepath.Join(s.dataDir, fileID, "tts")
	ttsPath := filepath.Join(ttsDir, fmt.Sprintf("tts_%03d.mp3", segmentIndex))
	ttsTempPath := filepath.Join(ttsDir, fmt.Sprintf("tts_%03d_temp.mp3", segmentIndex))

	ttsGen, err := tts.NewGeminiTTS(s.apiKey, false)
	if err != nil {
		return newTranslation, nil // 翻譯成功但 TTS 失敗，仍然回傳翻譯
	}

	err = ttsGen.GenerateSpeech(ctx, newTranslation, ttsTempPath)
	if err != nil {
		return newTranslation, nil // 翻譯成功但 TTS 失敗
	}

	// 音量匹配
	audioProcessor := audio.NewProcessor(false)
	if seg.AudioPath != "" {
		err = audioProcessor.MatchVolume(seg.AudioPath, ttsTempPath, ttsPath)
		if err != nil {
			os.Rename(ttsTempPath, ttsPath)
		} else {
			os.Remove(ttsTempPath)
		}
	} else {
		os.Rename(ttsTempPath, ttsPath)
	}

	return newTranslation, nil
}

// RetranslateSegmentWithInput 根據用戶輸入的原句重新翻譯並生成 TTS
// userInput: 用戶輸入的原句（任何語言），會被翻譯成英文
func (s *ProcessService) RetranslateSegmentWithInput(fileID string, segmentIndex int, userInput string) (string, error) {
	// 如果沒有用戶輸入，回退到原本的邏輯
	if userInput == "" {
		return s.RetranslateSegment(fileID, segmentIndex)
	}

	// 檢查 API key
	if s.apiKey == "" {
		return "", errors.New("GEMINI_API_KEY 未設定")
	}

	// 取得段落資料
	segments, err := s.GetSegmentsData(fileID)
	if err != nil {
		return "", fmt.Errorf("無法取得段落資料: %w", err)
	}

	// 檢查段落索引
	if segmentIndex < 0 || segmentIndex >= len(segments.Segments) {
		return "", fmt.Errorf("無效的段落索引: %d", segmentIndex)
	}

	seg := &segments.Segments[segmentIndex]

	// 建立翻譯器
	trans, err := translator.NewGeminiTranslator(s.apiKey, true)
	if err != nil {
		return "", fmt.Errorf("建立翻譯器失敗: %w", err)
	}

	// 將用戶輸入翻譯成英文
	ctx := context.Background()
	englishTranslation, err := trans.TranslateToEnglish(ctx, userInput)
	if err != nil {
		return "", fmt.Errorf("翻譯失敗: %w", err)
	}

	// 更新段落的 TTS 文字
	seg.TTSText = englishTranslation

	// 儲存更新後的段落資料
	if err := s.saveSegments(fileID, segments); err != nil {
		return "", fmt.Errorf("儲存段落失敗: %w", err)
	}

	// 重新生成該段落的 TTS
	ttsDir := filepath.Join(s.dataDir, fileID, "tts")
	ttsPath := filepath.Join(ttsDir, fmt.Sprintf("tts_%03d.mp3", segmentIndex))
	ttsTempPath := filepath.Join(ttsDir, fmt.Sprintf("tts_%03d_temp.mp3", segmentIndex))

	ttsGen, err := tts.NewGeminiTTS(s.apiKey, false)
	if err != nil {
		return englishTranslation, nil // 翻譯成功但 TTS 建立失敗
	}

	err = ttsGen.GenerateSpeech(ctx, englishTranslation, ttsTempPath)
	if err != nil {
		return englishTranslation, nil // 翻譯成功但 TTS 生成失敗
	}

	// 音量匹配
	audioProcessor := audio.NewProcessor(false)
	if seg.AudioPath != "" {
		err = audioProcessor.MatchVolume(seg.AudioPath, ttsTempPath, ttsPath)
		if err != nil {
			os.Rename(ttsTempPath, ttsPath)
		} else {
			os.Remove(ttsTempPath)
		}
	} else {
		os.Rename(ttsTempPath, ttsPath)
	}

	return englishTranslation, nil
}
