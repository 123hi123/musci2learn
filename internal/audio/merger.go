package audio

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"
)

type Merger struct {
verbose bool
}

func NewMerger(verbose bool) *Merger {
return &Merger{verbose: verbose}
}

func (m *Merger) MergeInterleaved(origFiles, ttsFiles []string, outputPath string, pattern string) error {
if len(origFiles) != len(ttsFiles) {
return fmt.Errorf("file count mismatch")
}
var files []string
for i := 0; i < len(origFiles); i++ {
orig := origFiles[i]
tts := ttsFiles[i]
switch pattern {
case "original-tts":
if orig != "" {
files = append(files, orig)
}
if tts != "" {
files = append(files, tts)
}
case "original-tts-original":
if orig != "" {
files = append(files, orig)
}
if tts != "" {
files = append(files, tts)
}
if orig != "" {
files = append(files, orig)
}
case "tts-original":
if tts != "" {
files = append(files, tts)
}
if orig != "" {
files = append(files, orig)
}
default:
return fmt.Errorf("unknown pattern: %s", pattern)
}
}
if len(files) == 0 {
return fmt.Errorf("no files")
}
return m.concat(files, outputPath)
}

func (m *Merger) concat(files []string, outputPath string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to concat")
	}
	
	// Filter out non-existent files
	var validFiles []string
	for _, f := range files {
		if f != "" {
			if _, err := os.Stat(f); err == nil {
				validFiles = append(validFiles, f)
			}
		}
	}
	
	if len(validFiles) == 0 {
		return fmt.Errorf("no valid files to concat")
	}
	
	concatPath := filepath.Join(filepath.Dir(outputPath), "concat_list.txt")
	defer os.Remove(concatPath)
	
	var sb strings.Builder
	for _, f := range validFiles {
		// Get absolute path and use forward slashes
		absPath, _ := filepath.Abs(f)
		safePath := strings.ReplaceAll(absPath, "\\", "/")
		sb.WriteString(fmt.Sprintf("file '%s'\n", safePath))
	}
	
	if err := os.WriteFile(concatPath, []byte(sb.String()), 0644); err != nil {
		return err
	}
	
	os.MkdirAll(filepath.Dir(outputPath), 0755)
	args := []string{"-f", "concat", "-safe", "0", "-i", concatPath, "-acodec", "libmp3lame", "-ar", "44100", "-ac", "2", "-b:a", "192k", "-y", outputPath}
	cmd := exec.Command("ffmpeg", args...)
	if m.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}
