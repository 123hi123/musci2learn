package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"multilang-learner/internal/langdetect"
)

type GeminiTranslator struct {
	apiKey   string
	baseURL  string
	verbose  bool
	detector *langdetect.Detector
}

func NewGeminiTranslator(apiKey string, verbose bool) (*GeminiTranslator, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required")
	}
	return &GeminiTranslator{
		apiKey:   apiKey,
		baseURL:  "https://generativelanguage.googleapis.com/v1beta",
		verbose:  verbose,
		detector: langdetect.NewDetector(),
	}, nil
}

// IsTargetLanguage checks if text is in the target language
func (g *GeminiTranslator) IsTargetLanguage(text string, targetLang string) bool {
	return g.detector.IsTargetLanguage(text, targetLang)
}

type transReq struct {
	Contents []content `json:"contents"`
	GenCfg   genConfig `json:"generationConfig,omitempty"`
}
type content struct {
	Parts []part `json:"parts"`
}
type part struct {
	Text string `json:"text"`
}
type genConfig struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxOutputTokens,omitempty"`
}
type transResp struct {
	Candidates []cand  `json:"candidates"`
	Error      *apiErr `json:"error,omitempty"`
}
type apiErr struct {
	Message string `json:"message"`
}
type cand struct {
	Content contentResp `json:"content"`
}
type contentResp struct {
	Parts []partResp `json:"parts"`
}
type partResp struct {
	Text string `json:"text"`
}

func (g *GeminiTranslator) TranslateLyric(ctx context.Context, text string, targetLang string) (string, error) {
	// 構建更精確的提示詞，確保翻譯結果只包含目標語言
	var prompt string
	switch targetLang {
	case "en", "English":
		prompt = fmt.Sprintf(`You are a professional translator. Translate the following lyrics to English.

Rules:
1. Output ONLY the English translation, nothing else
2. Do NOT include any Chinese, Japanese, Korean or other non-English characters
3. Preserve the original meaning as much as possible
4. Keep it natural and fluent in English

Original text:
%s

English translation:`, text)
	case "zh", "Chinese":
		prompt = fmt.Sprintf(`You are a professional translator. Translate the following lyrics to Chinese (Traditional).

Rules:
1. Output ONLY the Chinese translation, nothing else
2. Do NOT include any English, Japanese, Korean or other non-Chinese characters
3. Preserve the original meaning as much as possible
4. Use Traditional Chinese (繁體中文)

Original text:
%s

Chinese translation:`, text)
	default:
		prompt = fmt.Sprintf("Translate the following text to %s. Output ONLY the translation, nothing else:\n%s", targetLang, text)
	}

	url := fmt.Sprintf("%s/models/gemini-2.0-flash:generateContent?key=%s", g.baseURL, g.apiKey)
	req := transReq{
		Contents: []content{{Parts: []part{{Text: prompt}}}},
		GenCfg:   genConfig{Temperature: 0.3, MaxTokens: 150},
	}

	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var transR transResp
	if err := json.Unmarshal(body, &transR); err != nil {
		return "", err
	}

	if transR.Error != nil {
		return "", fmt.Errorf("API error: %s", transR.Error.Message)
	}

	if len(transR.Candidates) > 0 && len(transR.Candidates[0].Content.Parts) > 0 {
		translation := strings.TrimSpace(transR.Candidates[0].Content.Parts[0].Text)

		// Validate that translation is in target language
		if !g.detector.IsTargetLanguage(translation, targetLang) {
			if g.verbose {
				detectedLang, conf, _ := g.detector.Detect(translation)
				fmt.Printf("Translation validation failed: expected %s, got %s (%.2f)\n", targetLang, detectedLang, conf)
			}
			return "", fmt.Errorf("translation not in target language")
		}

		return translation, nil
	}
	return "", fmt.Errorf("no translation")
}

// RetranslateLyric 重新翻譯單句歌詞（用於使用者手動觸發的重新翻譯）
// 使用更嚴格的提示詞確保翻譯品質
func (g *GeminiTranslator) RetranslateLyric(ctx context.Context, originalText string, chineseText string, targetLang string) (string, error) {
	var prompt string

	// 提供原文和中文翻譯作為參考，要求重新翻譯成英文
	if targetLang == "en" || targetLang == "English" {
		prompt = fmt.Sprintf(`You are a professional translator specializing in song lyrics.

Task: Translate the following lyrics to natural, fluent English.

Original lyrics (may be Japanese, Korean, Russian, or other languages):
%s

Chinese translation for reference:
%s

Rules:
1. Output ONLY the English translation
2. The output must be 100%% in English - NO Chinese, Japanese, Korean, Russian or any other non-English characters allowed
3. Preserve the poetic meaning and emotional tone
4. Make it sound natural in English
5. If the Chinese reference helps understand the meaning, use it as context

English translation:`, originalText, chineseText)
	} else {
		prompt = fmt.Sprintf(`Translate to %s. Original: %s. Reference: %s. Output ONLY the translation.`, targetLang, originalText, chineseText)
	}

	url := fmt.Sprintf("%s/models/gemini-2.0-flash:generateContent?key=%s", g.baseURL, g.apiKey)
	req := transReq{
		Contents: []content{{Parts: []part{{Text: prompt}}}},
		GenCfg:   genConfig{Temperature: 0.2, MaxTokens: 200}, // 降低溫度以獲得更穩定的翻譯
	}

	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var transR transResp
	if err := json.Unmarshal(body, &transR); err != nil {
		return "", err
	}

	if transR.Error != nil {
		return "", fmt.Errorf("API error: %s", transR.Error.Message)
	}

	if len(transR.Candidates) > 0 && len(transR.Candidates[0].Content.Parts) > 0 {
		translation := strings.TrimSpace(transR.Candidates[0].Content.Parts[0].Text)

		// 驗證翻譯結果
		if targetLang == "en" || targetLang == "English" {
			if !g.detector.IsTargetLanguage(translation, "en") {
				// 如果驗證失敗，嘗試再次請求
				return "", fmt.Errorf("translation contains non-English characters, please try again")
			}
		}

		return translation, nil
	}
	return "", fmt.Errorf("no translation")
}

// TranslateToEnglish 將任意語言的句子一比一翻譯成英文
// 這是用戶手動輸入原句後的簡單翻譯，只輸出英文，不附加任何說明
func (g *GeminiTranslator) TranslateToEnglish(ctx context.Context, userInput string) (string, error) {
	prompt := fmt.Sprintf(`Translate the following sentence to English. 
Output ONLY the English translation, nothing else. No explanations, no notes, no original text.

Input: %s

English translation:`, userInput)

	url := fmt.Sprintf("%s/models/gemini-2.0-flash:generateContent?key=%s", g.baseURL, g.apiKey)
	req := transReq{
		Contents: []content{{Parts: []part{{Text: prompt}}}},
		GenCfg:   genConfig{Temperature: 0.2, MaxTokens: 200},
	}

	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var transR transResp
	if err := json.Unmarshal(body, &transR); err != nil {
		return "", err
	}

	if transR.Error != nil {
		return "", fmt.Errorf("API error: %s", transR.Error.Message)
	}

	if len(transR.Candidates) > 0 && len(transR.Candidates[0].Content.Parts) > 0 {
		translation := strings.TrimSpace(transR.Candidates[0].Content.Parts[0].Text)

		// 移除可能的引號
		translation = strings.Trim(translation, `"'`)

		return translation, nil
	}
	return "", fmt.Errorf("no translation")
}

func (g *GeminiTranslator) TranslateBatch(ctx context.Context, texts []string, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}
	var sb strings.Builder
	for i, t := range texts {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
	}
	prompt := fmt.Sprintf("Translate each line to %s. Output in same numbered format:\n%s", targetLang, sb.String())
	url := fmt.Sprintf("%s/models/gemini-2.0-flash:generateContent?key=%s", g.baseURL, g.apiKey)
	req := transReq{Contents: []content{{Parts: []part{{Text: prompt}}}}, GenCfg: genConfig{Temperature: 0.3, MaxTokens: len(texts) * 100}}
	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var transR transResp
	if err := json.Unmarshal(body, &transR); err != nil {
		return nil, err
	}
	if transR.Error != nil {
		return nil, fmt.Errorf("API error: %s", transR.Error.Message)
	}
	if len(transR.Candidates) > 0 && len(transR.Candidates[0].Content.Parts) > 0 {
		return parseNumbered(transR.Candidates[0].Content.Parts[0].Text, len(texts))
	}
	return nil, fmt.Errorf("no translation")
}

func parseNumbered(response string, count int) ([]string, error) {
	results := make([]string, count)
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var num int
		if _, err := fmt.Sscanf(line, "%d.", &num); err == nil {
			idx := strings.Index(line, ".")
			if idx != -1 && idx < len(line)-1 && num > 0 && num <= count {
				results[num-1] = strings.TrimSpace(line[idx+1:])
			}
		}
	}
	for i, r := range results {
		if r == "" {
			return nil, fmt.Errorf("missing %d", i+1)
		}
	}
	return results, nil
}
