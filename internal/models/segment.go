package models

// Segment 音訊段落
type Segment struct {
	Index        int     `json:"index"`
	StartTime    float64 `json:"startTime"`    // 開始時間（秒）
	EndTime      float64 `json:"endTime"`      // 結束時間（秒）
	Duration     float64 `json:"duration"`     // 時長（秒）
	LineIndices  []int   `json:"lineIndices"`  // 包含的歌詞行索引
	OriginalText string  `json:"originalText"` // 合併的原文
	TTSText      string  `json:"ttsText"`      // TTS 用的翻譯文字
	IsMeaningful bool    `json:"isMeaningful"` // 是否有意義
	AudioPath    string  `json:"audioPath"`    // 段落音訊路徑
	TTSPath      string  `json:"ttsPath"`      // TTS 音訊路徑
}

// SegmentsData 段落資料
type SegmentsData struct {
	FileID   string    `json:"fileId"`
	Segments []Segment `json:"segments"`
	Language string    `json:"language"` // TTS 語言
}

// PlaybackItem 播放項目
type PlaybackItem struct {
	Type      string      `json:"type"`      // "original" 或 "tts"
	Index     int         `json:"index"`     // 段落索引
	AudioURL  string      `json:"audioUrl"`  // 音訊 URL
	StartTime float64     `json:"startTime"` // 開始時間
	Duration  float64     `json:"duration"`  // 時長
	Display   DisplayText `json:"display"`   // 顯示文字
}

// GeneratePlaylist 生成播放列表
func GeneratePlaylist(segments []Segment, lyrics []LyricLine, settings FileSettings) []PlaybackItem {
	var playlist []PlaybackItem

	for _, seg := range segments {
		if !seg.IsMeaningful {
			continue
		}

		// 獲取段落對應的顯示文字
		var displayOriginal, displayPrimary, displayChinese string
		for _, idx := range seg.LineIndices {
			if idx < len(lyrics) {
				dt := lyrics[idx].GetDisplayText(settings.PrimaryLanguage, settings.ShowChineseTranslation)
				if displayOriginal != "" {
					displayOriginal += "\n"
					displayPrimary += "\n"
					if displayChinese != "" {
						displayChinese += "\n"
					}
				}
				displayOriginal += dt.Original
				displayPrimary += dt.Primary
				if dt.Chinese != "" {
					displayChinese += dt.Chinese
				}
			}
		}

		display := DisplayText{
			Original: displayOriginal,
			Primary:  displayPrimary,
			Chinese:  displayChinese,
		}

		// 添加原曲段落
		playlist = append(playlist, PlaybackItem{
			Type:      "original",
			Index:     seg.Index,
			AudioURL:  seg.AudioPath,
			StartTime: seg.StartTime,
			Duration:  seg.Duration,
			Display:   display,
		})

		// 添加 TTS（根據重複次數）
		for i := 0; i < settings.TTSRepeatCount; i++ {
			playlist = append(playlist, PlaybackItem{
				Type:     "tts",
				Index:    seg.Index,
				AudioURL: seg.TTSPath,
				Duration: 0, // 會在前端播放時計算
				Display:  display,
			})
		}
	}

	return playlist
}

// ProcessProgress 處理進度
type ProcessProgress struct {
	FileID      string  `json:"fileId"`
	Status      string  `json:"status"`      // "translating", "segmenting", "generating_tts", "done", "error"
	Progress    float64 `json:"progress"`    // 0-100
	Message     string  `json:"message"`     // 目前步驟說明
	TotalSteps  int     `json:"totalSteps"`  // 總步驟數
	CurrentStep int     `json:"currentStep"` // 目前步驟
}
