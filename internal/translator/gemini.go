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
	// Build a more specific prompt for the target language
	prompt := fmt.Sprintf("Translate the following text to %s. Output ONLY the translation, nothing else:\n%s", targetLang, text)
	
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
