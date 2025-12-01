package models

// Translations 翻譯內容
type Translations struct {
	Embedded string `json:"embedded,omitempty"` // 檔案內嵌翻譯
	En       string `json:"en,omitempty"`       // 英文翻譯
	Zh       string `json:"zh,omitempty"`       // 中文翻譯
}

// LyricLine 歌詞行
type LyricLine struct {
	Index        int          `json:"index"`
	Timestamp    string       `json:"timestamp"`    // "00:01.79"
	StartTime    float64      `json:"startTime"`    // 秒
	EndTime      float64      `json:"endTime"`      // 秒
	Original     string       `json:"original"`     // 原文歌詞
	Translations Translations `json:"translations"` // 翻譯
	IsMeaningful bool         `json:"isMeaningful"` // 是否有意義（非空白、非標記）
	IsSkipped    bool         `json:"isSkipped"`    // 是否被跳過（在起點之前）
}

// LyricsData 歌詞資料
type LyricsData struct {
	FileID         string      `json:"fileId"`
	Lines          []LyricLine `json:"lines"`
	DetectedLang   string      `json:"detectedLang"`   // 檢測到的原文語言
	HasEmbedded    bool        `json:"hasEmbedded"`    // 是否有內嵌翻譯
	StartLineIndex int         `json:"startLineIndex"` // 起點行索引
}

// GetActiveLyrics 獲取有效歌詞（起點之後且有意義的）
func (ld *LyricsData) GetActiveLyrics() []LyricLine {
	var active []LyricLine
	for _, line := range ld.Lines {
		if line.Index >= ld.StartLineIndex && line.IsMeaningful {
			active = append(active, line)
		}
	}
	return active
}

// GetDisplayText 獲取顯示文字
func (l *LyricLine) GetDisplayText(lang string, showChinese bool) DisplayText {
	dt := DisplayText{
		Original: l.Original,
	}
	
	// 主要語言翻譯
	switch lang {
	case "en":
		if l.Translations.En != "" {
			dt.Primary = l.Translations.En
		}
	case "zh":
		if l.Translations.Zh != "" {
			dt.Primary = l.Translations.Zh
		} else if l.Translations.Embedded != "" {
			dt.Primary = l.Translations.Embedded
		}
	}
	
	// 中文輔助翻譯
	if showChinese && lang != "zh" {
		if l.Translations.Zh != "" {
			dt.Chinese = l.Translations.Zh
		} else if l.Translations.Embedded != "" {
			dt.Chinese = l.Translations.Embedded
		}
	}
	
	return dt
}

// DisplayText 顯示文字
type DisplayText struct {
	Original string `json:"original"` // 原文
	Primary  string `json:"primary"`  // 主要語言翻譯
	Chinese  string `json:"chinese"`  // 中文翻譯（可選）
}
