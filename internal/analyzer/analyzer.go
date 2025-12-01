package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"multilang-learner/internal/subtitle"
)

// Analyzer uses Gemini to analyze lyrics content
type Analyzer struct {
	apiKey  string
	baseURL string
	verbose bool
}

// AnalysisResult contains the analysis of lyrics
type AnalysisResult struct {
	MusicStartIndex int      // Index where actual music/lyrics start (0-based)
	ValidIndices    []int    // Indices of lines that are actual lyrics
	NonLyricIndices []int    // Indices of metadata/non-lyric lines
}

// NewAnalyzer creates a new lyrics analyzer
func NewAnalyzer(apiKey string, verbose bool) (*Analyzer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required")
	}
	return &Analyzer{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		verbose: verbose,
	}, nil
}

type analyzeReq struct {
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
type analyzeResp struct {
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

// AnalyzeLyrics analyzes which lines are actual lyrics vs metadata
func (a *Analyzer) AnalyzeLyrics(ctx context.Context, lines []subtitle.Line) (*AnalysisResult, error) {
	if len(lines) == 0 {
		return &AnalysisResult{}, nil
	}

	// Build the prompt with numbered lines
	var sb strings.Builder
	sb.WriteString("Analyze these song lyrics. For each line, determine if it's:\n")
	sb.WriteString("- METADATA: Title, artist name, composer credits, album info, etc.\n")
	sb.WriteString("- LYRICS: Actual song lyrics/vocals\n\n")
	sb.WriteString("Lines:\n")
	
	for i, line := range lines {
		text := strings.TrimSpace(line.Text)
		if text == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i, formatDuration(line.StartTime), text))
	}
	
	sb.WriteString("\nRespond with JSON only, no other text:\n")
	sb.WriteString(`{"music_start_index": <first LYRICS line index>, "metadata_indices": [<indices of METADATA lines>]}`)

	url := fmt.Sprintf("%s/models/gemini-2.0-flash:generateContent?key=%s", a.baseURL, a.apiKey)
	req := analyzeReq{
		Contents: []content{{Parts: []part{{Text: sb.String()}}}},
		GenCfg:   genConfig{Temperature: 0.1, MaxTokens: 500},
	}
	
	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var analyzeR analyzeResp
	if err := json.Unmarshal(body, &analyzeR); err != nil {
		return nil, err
	}
	
	if analyzeR.Error != nil {
		return nil, fmt.Errorf("API error: %s", analyzeR.Error.Message)
	}
	
	if len(analyzeR.Candidates) == 0 || len(analyzeR.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from API")
	}
	
	// Parse the JSON response
	responseText := analyzeR.Candidates[0].Content.Parts[0].Text
	responseText = extractJSON(responseText)
	
	var result struct {
		MusicStartIndex int   `json:"music_start_index"`
		MetadataIndices []int `json:"metadata_indices"`
	}
	
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		if a.verbose {
			fmt.Printf("Failed to parse analysis response: %s\n", responseText)
		}
		// Fallback: assume first line with actual content is music start
		return a.fallbackAnalysis(lines), nil
	}
	
	// Build the analysis result
	analysis := &AnalysisResult{
		MusicStartIndex: result.MusicStartIndex,
		NonLyricIndices: result.MetadataIndices,
	}
	
	// Build valid indices (all indices not in metadata)
	metadataSet := make(map[int]bool)
	for _, idx := range result.MetadataIndices {
		metadataSet[idx] = true
	}
	
	for i := range lines {
		if !metadataSet[i] {
			analysis.ValidIndices = append(analysis.ValidIndices, i)
		}
	}
	
	return analysis, nil
}

// fallbackAnalysis provides a simple fallback when API fails
func (a *Analyzer) fallbackAnalysis(lines []subtitle.Line) *AnalysisResult {
	result := &AnalysisResult{}
	
	metadataKeywords := []string{
		"lyrics by", "composed by", "作詞", "作曲", "编曲", "編曲",
		"produced by", "written by", "music by", "arrangement",
	}
	
	foundMusicStart := false
	for i, line := range lines {
		text := strings.ToLower(line.Text)
		isMetadata := false
		
		for _, kw := range metadataKeywords {
			if strings.Contains(text, kw) {
				isMetadata = true
				break
			}
		}
		
		if isMetadata {
			result.NonLyricIndices = append(result.NonLyricIndices, i)
		} else {
			result.ValidIndices = append(result.ValidIndices, i)
			if !foundMusicStart && strings.TrimSpace(line.Text) != "" {
				result.MusicStartIndex = i
				foundMusicStart = true
			}
		}
	}
	
	return result
}

func extractJSON(s string) string {
	// Find the first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func formatDuration(d time.Duration) string {
	min := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}

// SegmentMeaningResult contains analysis of segment meaningfulness
type SegmentMeaningResult struct {
	MeaningfulIndices   []int // Indices of segments with meaningful content
	UnmeaningfulIndices []int // Indices of segments with only interjections/sounds
}

// AnalyzeSegmentMeaning analyzes which segments have meaningful content vs just sounds/interjections
func (a *Analyzer) AnalyzeSegmentMeaning(ctx context.Context, segments []struct {
	Index int
	Text  string
}) (*SegmentMeaningResult, error) {
	if len(segments) == 0 {
		return &SegmentMeaningResult{}, nil
	}

	// Build the prompt
	var sb strings.Builder
	sb.WriteString("Analyze these song lyric segments. For each segment, determine if it has:\n")
	sb.WriteString("- MEANINGFUL: Contains actual words with meaning (sentences, phrases)\n")
	sb.WriteString("- SOUND_ONLY: Contains only interjections, vocalizations, or sounds like 'oh', 'ah', 'la la', 'yeah', 'woah', 'mmm', etc.\n\n")
	sb.WriteString("Segments:\n")

	for _, seg := range segments {
		sb.WriteString(fmt.Sprintf("%d. %s\n", seg.Index, seg.Text))
	}

	sb.WriteString("\nRespond with JSON only, no other text:\n")
	sb.WriteString(`{"meaningful_indices": [<indices of MEANINGFUL segments>], "sound_only_indices": [<indices of SOUND_ONLY segments>]}`)

	url := fmt.Sprintf("%s/models/gemini-2.0-flash:generateContent?key=%s", a.baseURL, a.apiKey)
	req := analyzeReq{
		Contents: []content{{Parts: []part{{Text: sb.String()}}}},
		GenCfg:   genConfig{Temperature: 0.1, MaxTokens: 500},
	}

	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var analyzeR analyzeResp
	if err := json.Unmarshal(body, &analyzeR); err != nil {
		return nil, err
	}

	if analyzeR.Error != nil {
		return nil, fmt.Errorf("API error: %s", analyzeR.Error.Message)
	}

	if len(analyzeR.Candidates) == 0 || len(analyzeR.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	// Parse the JSON response
	responseText := analyzeR.Candidates[0].Content.Parts[0].Text
	responseText = extractJSON(responseText)

	var result struct {
		MeaningfulIndices []int `json:"meaningful_indices"`
		SoundOnlyIndices  []int `json:"sound_only_indices"`
	}

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		if a.verbose {
			fmt.Printf("Failed to parse meaning analysis response: %s\n", responseText)
		}
		// Fallback: assume all segments are meaningful
		return a.fallbackMeaningAnalysis(segments), nil
	}

	return &SegmentMeaningResult{
		MeaningfulIndices:   result.MeaningfulIndices,
		UnmeaningfulIndices: result.SoundOnlyIndices,
	}, nil
}

// fallbackMeaningAnalysis provides a simple fallback using pattern matching
func (a *Analyzer) fallbackMeaningAnalysis(segments []struct {
	Index int
	Text  string
}) *SegmentMeaningResult {
	result := &SegmentMeaningResult{}

	// Common interjection patterns
	interjectionPatterns := []string{
		"oh", "ah", "eh", "uh", "mm", "hmm", "la", "na", "da",
		"yeah", "yeh", "ya", "woah", "whoa", "wow",
		"ooh", "aah", "eeh", "uuh",
		"哦", "啊", "嗯", "呀", "喔", "噢", "唔",
	}

	for _, seg := range segments {
		text := strings.ToLower(seg.Text)
		// Remove common punctuation and spaces
		text = strings.ReplaceAll(text, " ", "")
		text = strings.ReplaceAll(text, "-", "")

		isSoundOnly := true
		// Check if text contains any substantial words
		for _, word := range strings.Fields(seg.Text) {
			word = strings.ToLower(strings.Trim(word, ".,!?-~"))
			if len(word) < 2 {
				continue
			}
			isInterjection := false
			for _, pattern := range interjectionPatterns {
				if strings.Contains(word, pattern) && len(word) <= len(pattern)+2 {
					isInterjection = true
					break
				}
			}
			if !isInterjection && len(word) > 3 {
				isSoundOnly = false
				break
			}
		}

		if isSoundOnly {
			result.UnmeaningfulIndices = append(result.UnmeaningfulIndices, seg.Index)
		} else {
			result.MeaningfulIndices = append(result.MeaningfulIndices, seg.Index)
		}
	}

	return result
}
