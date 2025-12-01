package audio

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"regexp"
"strings"
"time"
"multilang-learner/internal/subtitle"
)

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
