package audio

import (
	"bufio"
	"fmt"
	"multilang-learner/internal/subtitle"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// AudioStats 音訊統計資訊
type AudioStats struct {
	PeakDB     float64 // 峰值音量 (dB)
	MeanDB     float64 // 平均音量 (dB)
	MaxVolume  float64 // 最大音量 (線性)
}

type Processor struct {
	verbose bool
}

func NewProcessor(verbose bool) *Processor {
	return &Processor{verbose: verbose}
}

func (p *Processor) SplitByLyrics(inputPath string, lines []subtitle.Line, outputDir string) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}
	var results []string
	for i, line := range lines {
		safeName := sanitize(line.Text)
		if len(safeName) > 30 {
			safeName = safeName[:30]
		}
		outPath := filepath.Join(outputDir, fmt.Sprintf("%03d_%s.mp3", i+1, safeName))
		if err := p.cut(inputPath, line.StartTime, line.EndTime, outPath); err != nil {
			return nil, fmt.Errorf("cut %d failed: %w", i+1, err)
		}
		results = append(results, outPath)
	}
	return results, nil
}

func (p *Processor) cut(inputPath string, start, end time.Duration, outputPath string) error {
	startSec := start.Seconds()
	duration := end.Seconds() - startSec
	if duration <= 0 {
		duration = 0.1
	}
	args := []string{"-y", "-i", inputPath, "-ss", fmt.Sprintf("%.3f", startSec), "-t", fmt.Sprintf("%.3f", duration), "-acodec", "libmp3lame", "-ar", "44100", "-ac", "2", "-b:a", "192k", outputPath}
	cmd := exec.Command("ffmpeg", args...)
	if p.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// CutSegment cuts a segment from the audio file
func (p *Processor) CutSegment(inputPath string, start, end time.Duration, outputPath string) error {
	return p.cut(inputPath, start, end, outputPath)
}

func sanitize(s string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	s = re.ReplaceAllString(s, "_")
	s = strings.TrimSpace(s)
	if s == "" {
		s = "segment"
	}
	return s
}

// AnalyzeVolume 分析音檔的音量統計
// 使用 ffmpeg 的 volumedetect filter 來取得峰值和平均音量
func (p *Processor) AnalyzeVolume(inputPath string) (*AudioStats, error) {
	// 使用 ffmpeg volumedetect filter
	args := []string{
		"-i", inputPath,
		"-af", "volumedetect",
		"-f", "null",
		"-",
	}

	cmd := exec.Command("ffmpeg", args...)
	// volumedetect 輸出到 stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// ffmpeg 可能回傳非零但輸出有效，檢查輸出
		if len(output) == 0 {
			return nil, fmt.Errorf("volumedetect failed: %w", err)
		}
	}

	stats := &AudioStats{}
	outputStr := string(output)

	// 解析輸出
	// [Parsed_volumedetect_0 @ xxx] max_volume: -5.2 dB
	// [Parsed_volumedetect_0 @ xxx] mean_volume: -18.3 dB
	scanner := bufio.NewScanner(strings.NewReader(outputStr))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "max_volume:") {
			// 提取 max_volume 值
			parts := strings.Split(line, "max_volume:")
			if len(parts) >= 2 {
				valStr := strings.TrimSpace(strings.Replace(parts[1], "dB", "", 1))
				if val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64); err == nil {
					stats.PeakDB = val
				}
			}
		} else if strings.Contains(line, "mean_volume:") {
			// 提取 mean_volume 值
			parts := strings.Split(line, "mean_volume:")
			if len(parts) >= 2 {
				valStr := strings.TrimSpace(strings.Replace(parts[1], "dB", "", 1))
				if val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64); err == nil {
					stats.MeanDB = val
				}
			}
		}
	}

	return stats, nil
}

// AdjustVolume 調整音檔音量
// adjustment 是 dB 值，正數增加音量，負數減少音量
func (p *Processor) AdjustVolume(inputPath string, outputPath string, adjustmentDB float64) error {
	// 使用 volume filter 調整音量
	volumeFilter := fmt.Sprintf("volume=%.2fdB", adjustmentDB)

	args := []string{
		"-y",
		"-i", inputPath,
		"-af", volumeFilter,
		"-acodec", "libmp3lame",
		"-ar", "44100",
		"-ac", "2",
		"-b:a", "192k",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	if p.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// NormalizeToTarget 將音檔音量正規化到目標峰值
// targetPeakDB 是目標峰值 dB 值
func (p *Processor) NormalizeToTarget(inputPath string, outputPath string, targetPeakDB float64) error {
	// 先分析原始音量
	stats, err := p.AnalyzeVolume(inputPath)
	if err != nil {
		return fmt.Errorf("analyze volume failed: %w", err)
	}

	// 計算需要調整的 dB 值
	adjustmentDB := targetPeakDB - stats.PeakDB

	// 如果調整量太小（小於 0.5 dB），直接複製檔案
	if adjustmentDB > -0.5 && adjustmentDB < 0.5 {
		// 直接複製
		input, err := os.ReadFile(inputPath)
		if err != nil {
			return err
		}
		return os.WriteFile(outputPath, input, 0644)
	}

	// 調整音量
	return p.AdjustVolume(inputPath, outputPath, adjustmentDB)
}

// MatchVolume 將 TTS 音檔的音量調整為與原曲段落一致
// 回傳調整後的檔案路徑
func (p *Processor) MatchVolume(segmentPath string, ttsPath string, outputPath string) error {
	// 分析原曲段落音量
	segmentStats, err := p.AnalyzeVolume(segmentPath)
	if err != nil {
		return fmt.Errorf("analyze segment volume failed: %w", err)
	}

	// 分析 TTS 音量
	ttsStats, err := p.AnalyzeVolume(ttsPath)
	if err != nil {
		return fmt.Errorf("analyze TTS volume failed: %w", err)
	}

	// 計算調整量：讓 TTS 的峰值等於原曲的峰值
	adjustmentDB := segmentStats.PeakDB - ttsStats.PeakDB

	if p.verbose {
		fmt.Printf("Segment peak: %.2f dB, TTS peak: %.2f dB, Adjustment: %.2f dB\n",
			segmentStats.PeakDB, ttsStats.PeakDB, adjustmentDB)
	}

	// 如果調整量太小，直接複製
	if adjustmentDB > -0.5 && adjustmentDB < 0.5 {
		input, err := os.ReadFile(ttsPath)
		if err != nil {
			return err
		}
		return os.WriteFile(outputPath, input, 0644)
	}

	// 調整 TTS 音量
	return p.AdjustVolume(ttsPath, outputPath, adjustmentDB)
}
