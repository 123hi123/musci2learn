package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// LRCLine represents a single LRC line
type LRCLine struct {
	Timestamp string
	Text      string
	IsRussian bool
}

// SegmentInfo holds segment timing and content
type SegmentInfo struct {
	Index       int
	Duration    float64
	LineCount   int
	Original    string
	Translation string
	HasTTS      bool
}

func main() {
	processDir := flag.String("dir", "", "Process directory containing LRC and output.mp3")
	filterMeta := flag.Bool("filter-meta", true, "Filter metadata lines (title, lyrics by, composed by)")
	flag.Parse()

	if *processDir == "" {
		log.Fatal("Please specify -dir <process_directory>")
	}

	// Find files
	enhancedLRC := filepath.Join(*processDir, "02_enhanced.lrc")
	segmentsTxt := filepath.Join(*processDir, "03_segments.txt")
	outputMP3 := filepath.Join(*processDir, "output.mp3")
	splitDir := filepath.Join(*processDir, "split")
	ttsDir := filepath.Join(*processDir, "tts")

	if _, err := os.Stat(enhancedLRC); os.IsNotExist(err) {
		log.Fatalf("Enhanced LRC not found: %s", enhancedLRC)
	}
	if _, err := os.Stat(outputMP3); os.IsNotExist(err) {
		log.Fatalf("Output MP3 not found: %s", outputMP3)
	}

	// Parse enhanced LRC
	lines, err := parseLRC(enhancedLRC)
	if err != nil {
		log.Fatalf("Failed to parse LRC: %v", err)
	}
	fmt.Printf("Parsed %d lines from enhanced LRC\n", len(lines))

	// Parse segments
	segments, err := parseSegments(segmentsTxt)
	if err != nil {
		log.Printf("Warning: Could not parse segments: %v", err)
	}
	fmt.Printf("Parsed %d segments\n", len(segments))

	// Get audio file durations
	splitDurations := getFileDurations(splitDir, len(segments))
	ttsDurations := getFileDurations(ttsDir, len(segments))

	// Build the output LRC with correct timeline
	outputLRC := buildOutputLRC(lines, segments, splitDurations, ttsDurations, *filterMeta)

	// Save the new LRC
	newLRCPath := filepath.Join(*processDir, "04_output.lrc")
	if err := os.WriteFile(newLRCPath, []byte(outputLRC), 0644); err != nil {
		log.Fatalf("Failed to write output LRC: %v", err)
	}
	fmt.Printf("Generated output LRC: %s\n", newLRCPath)

	// Embed LRC into MP3
	outputWithLRC := filepath.Join(*processDir, "output_with_lrc.mp3")
	if err := embedLRCToMP3(outputMP3, outputLRC, outputWithLRC); err != nil {
		log.Fatalf("Failed to embed LRC: %v", err)
	}

	fmt.Printf("\nDone! Output with embedded LRC: %s\n", outputWithLRC)
}

func parseLRC(path string) ([]LRCLine, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []LRCLine
	scanner := bufio.NewScanner(file)
	timestampRe := regexp.MustCompile(`^\[(\d{2}:\d{2}\.\d{2})\](.*)$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := timestampRe.FindStringSubmatch(line)
		if matches != nil {
			timestamp := matches[1]
			text := strings.TrimSpace(matches[2])
			if text == "" || text == "//" {
				continue
			}
			isRussian := containsCyrillic(text)
			lines = append(lines, LRCLine{
				Timestamp: timestamp,
				Text:      text,
				IsRussian: isRussian,
			})
		}
	}
	return lines, scanner.Err()
}

func containsCyrillic(s string) bool {
	for _, r := range s {
		if r >= 0x0400 && r <= 0x04FF {
			return true
		}
	}
	return false
}

func parseSegments(path string) ([]SegmentInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var segments []SegmentInfo
	scanner := bufio.NewScanner(file)
	// Format: "Segment 1 [LYRICS]: 7.07s, 5 lines - Text..."
	// Or: "Segment 16 [LYRICS [NO-TTS]]: 10.80s, 3 lines - Text..."
	segRe := regexp.MustCompile(`^Segment (\d+) \[([^\]]+(?:\s*\[[^\]]+\])?)\]: ([\d.]+)s, (\d+) lines - (.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := segRe.FindStringSubmatch(line)
		if matches != nil {
			var idx int
			var dur float64
			var lineCount int
			fmt.Sscanf(matches[1], "%d", &idx)
			fmt.Sscanf(matches[3], "%f", &dur)
			fmt.Sscanf(matches[4], "%d", &lineCount)

			hasTTS := !strings.Contains(matches[2], "NO-TTS")

			segments = append(segments, SegmentInfo{
				Index:    idx,
				Duration: dur,
				LineCount: lineCount,
				Original: matches[5],
				HasTTS:   hasTTS,
			})
		}
	}
	return segments, scanner.Err()
}

func getFileDurations(dir string, count int) []float64 {
	durations := make([]float64, count)
	for i := 0; i < count; i++ {
		path := filepath.Join(dir, fmt.Sprintf("%03d_segment.mp3", i+1))
		if i < count {
			// For TTS, check tts file
			if strings.Contains(dir, "tts") {
				path = filepath.Join(dir, fmt.Sprintf("%03d_tts.mp3", i+1))
			}
		}
		durations[i] = getAudioDuration(path)
	}
	return durations
}

func getAudioDuration(path string) float64 {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0
	}
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	var dur float64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &dur)
	return dur
}

func isMetadataLine(text string) bool {
	lower := strings.ToLower(text)
	patterns := []string{"lyrics by", "composed by", "作詞", "作曲", "編曲", "produced by", "written by"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	// Title line check
	if strings.Contains(text, " - ") {
		// Check if it looks like "Song - Artist" format
		parts := strings.Split(text, " - ")
		if len(parts) == 2 {
			// If both parts are relatively short, it's likely a title
			if len(parts[0]) < 50 && len(parts[1]) < 30 {
				// But make sure it's not actual lyrics with " - " in them
				if !containsCyrillic(text) || strings.HasPrefix(lower, "умри") || strings.HasPrefix(lower, "die") {
					return true
				}
			}
		}
	}
	return false
}

func buildOutputLRC(lines []LRCLine, segments []SegmentInfo, splitDurations, ttsDurations []float64, filterMeta bool) string {
	var sb strings.Builder
	sb.WriteString("[ti:Learning Track]\n")
	sb.WriteString("[ar:Multi-Language Learner]\n")
	sb.WriteString("[by:multilang-learner]\n\n")

	// Group LRC lines by timestamp to pair original/translation
	type LyricPair struct {
		Timestamp string
		Original  string
		Translation string
	}
	
	var pairs []LyricPair
	timeMap := make(map[string]*LyricPair)
	
	for _, line := range lines {
		if filterMeta && isMetadataLine(line.Text) {
			continue
		}
		
		pair, exists := timeMap[line.Timestamp]
		if !exists {
			pair = &LyricPair{Timestamp: line.Timestamp}
			timeMap[line.Timestamp] = pair
			pairs = append(pairs, *pair)
		}
		
		if line.IsRussian {
			timeMap[line.Timestamp].Original = line.Text
		} else {
			timeMap[line.Timestamp].Translation = line.Text
		}
	}
	
	// Update pairs from map
	for i := range pairs {
		if p, ok := timeMap[pairs[i].Timestamp]; ok {
			pairs[i] = *p
		}
	}

	// Now calculate timeline based on segment durations
	// output.mp3 structure: [split1][tts1][split2][tts2]...
	currentTime := 0.0
	segIdx := 0
	pairIdx := 0
	
	for segIdx < len(segments) && pairIdx < len(pairs) {
		splitDur := splitDurations[segIdx]
		ttsDur := ttsDurations[segIdx]
		
		// Find all pairs that belong to this segment (by counting lines)
		linesInSeg := segments[segIdx].LineCount
		
		// Write original lyrics at start of split segment
		ts := formatLRCTime(currentTime)
		var origTexts []string
		var transTexts []string
		
		for j := 0; j < linesInSeg && pairIdx+j < len(pairs); j++ {
			p := pairs[pairIdx+j]
			if p.Original != "" {
				origTexts = append(origTexts, p.Original)
			}
			if p.Translation != "" {
				transTexts = append(transTexts, p.Translation)
			}
		}
		
		// Write combined original text
		if len(origTexts) > 0 {
			sb.WriteString(fmt.Sprintf("[%s]%s\n", ts, strings.Join(origTexts, " ")))
		}
		
		currentTime += splitDur
		
		// Write translation during TTS segment
		if ttsDur > 0 && len(transTexts) > 0 {
			ttsTs := formatLRCTime(currentTime)
			sb.WriteString(fmt.Sprintf("[%s]%s\n", ttsTs, strings.Join(transTexts, " ")))
			currentTime += ttsDur
		} else if ttsDur == 0 {
			// No TTS, just show translation after original
			if len(transTexts) > 0 {
				sb.WriteString(fmt.Sprintf("[%s]%s\n", ts, strings.Join(transTexts, " ")))
			}
		}
		
		pairIdx += linesInSeg / 2 // Each "line" in segment is orig+trans pair
		if linesInSeg / 2 == 0 {
			pairIdx += linesInSeg
		}
		segIdx++
	}

	return sb.String()
}

func formatLRCTime(seconds float64) string {
	mins := int(seconds) / 60
	secs := seconds - float64(mins*60)
	return fmt.Sprintf("%02d:%05.2f", mins, secs)
}

func embedLRCToMP3(mp3Path, lrcContent, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y",
		"-i", mp3Path,
		"-metadata", fmt.Sprintf("lyrics=%s", lrcContent),
		"-codec", "copy",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, string(output))
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return err
	}
	fmt.Printf("Output file size: %.2f MB\n", float64(info.Size())/1024/1024)
	return nil
}
