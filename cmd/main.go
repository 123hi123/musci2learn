package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"multilang-learner/internal/analyzer"
	"multilang-learner/internal/audio"
	"multilang-learner/internal/segment"
	"multilang-learner/internal/subtitle"
	"multilang-learner/internal/translator"
	"multilang-learner/internal/tts"
)

func main() {
	audioPath := flag.String("audio", "", "Audio file path")
	lrcPath := flag.String("lrc", "", "LRC file path (optional)")
	targetLang := flag.String("lang", "English", "Target language")
	pattern := flag.String("pattern", "original-tts", "Merge pattern")
	minDuration := flag.Float64("min-duration", 5.0, "Minimum segment duration")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	if *audioPath == "" {
		fmt.Println("Usage: go run cmd/main.go -audio <file> [-lang <lang>]")
		os.Exit(1)
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY not set")
	}

	if _, err := os.Stat(*audioPath); os.IsNotExist(err) {
		log.Fatalf("Audio file not found: %s", *audioPath)
	}

	ctx := context.Background()
	processDir, err := createProcessDir(*audioPath)
	if err != nil {
		log.Fatalf("Failed to create process dir: %v", err)
	}
	fmt.Printf("Process folder: %s\n", processDir)

	fmt.Println("\n=== Step 1: Parsing lyrics ===")
	var lrcContent string
	if *lrcPath != "" {
		data, _ := os.ReadFile(*lrcPath)
		lrcContent = string(data)
	} else {
		lrcContent, err = extractLRC(*audioPath)
		if err != nil {
			log.Fatalf("Failed to extract LRC: %v", err)
		}
	}
	origLrcPath := filepath.Join(processDir, "01_original.lrc")
	os.WriteFile(origLrcPath, []byte(lrcContent), 0644)
	fmt.Printf("Original LRC saved: %s\n", origLrcPath)

	parser := subtitle.NewParser()
	lyrics, err := parser.Parse(lrcContent)
	if err != nil {
		log.Fatalf("Failed to parse LRC: %v", err)
	}
	fmt.Printf("Parsed %d lines\n", len(lyrics.Lines))

	fmt.Printf("\n=== Step 2: Translating to %s ===\n", *targetLang)
	trans, _ := translator.NewGeminiTranslator(apiKey, *verbose)
	translatedCount := 0
	for i, line := range lyrics.Lines {
		if strings.TrimSpace(line.Text) == "" {
			continue
		}
		if line.Translation != "" && trans.IsTargetLanguage(line.Translation, *targetLang) {
			continue
		}
		t, err := trans.TranslateLyric(ctx, line.Text, *targetLang)
		if err != nil {
			continue
		}
		lyrics.Lines[i].Translation = t
		translatedCount++
		fmt.Printf("Translated %d: %s -> %s\n", translatedCount, truncate(line.Text, 20), truncate(t, 20))
		time.Sleep(300 * time.Millisecond)
	}
	enhancedPath := filepath.Join(processDir, "02_enhanced.lrc")
	os.WriteFile(enhancedPath, []byte(parser.GenerateEnhancedLRC(lyrics)), 0644)
	fmt.Printf("Enhanced LRC saved: %s\n", enhancedPath)

	fmt.Printf("\n=== Step 3: Merging segments (min %.1fs) ===\n", *minDuration)
	segMerger := segment.NewMerger(time.Duration(*minDuration*float64(time.Second)), *verbose)
	segments := segMerger.MergeByDuration(lyrics.Lines)
	fmt.Printf("Created %d merged segments\n", len(segments))

	fmt.Println("\n=== Step 4: Analyzing segment meaning ===")
	lyricAnalyzer, _ := analyzer.NewAnalyzer(apiKey, *verbose)
	var segmentsForAnalysis []struct {
		Index int
		Text  string
	}
	for _, seg := range segments {
		segmentsForAnalysis = append(segmentsForAnalysis, struct {
			Index int
			Text  string
		}{seg.Index, seg.OriginalText})
	}
	meaningResult, err := lyricAnalyzer.AnalyzeSegmentMeaning(ctx, segmentsForAnalysis)
	if err != nil {
		fmt.Printf("Warning: Meaning analysis failed: %v\n", err)
		for i := range segments {
			segments[i].IsMeaningful = true
		}
	} else {
		segMerger.MarkMeaningfulness(segments, meaningResult.MeaningfulIndices)
		fmt.Printf("Found %d meaningful, %d sound-only segments\n", len(meaningResult.MeaningfulIndices), len(meaningResult.UnmeaningfulIndices))
	}

	for _, seg := range segments {
		meaningStr := ""
		if !seg.IsMeaningful {
			meaningStr = " [NO-TTS]"
		}
		fmt.Printf("  Seg %d%s: %.2fs - %s\n", seg.Index+1, meaningStr, seg.Duration.Seconds(), truncate(seg.OriginalText, 30))
	}
	segInfoPath := filepath.Join(processDir, "03_segments.txt")
	os.WriteFile(segInfoPath, []byte(segMerger.GetSegmentInfo(segments)), 0644)

	fmt.Println("\n=== Step 5: Splitting audio ===")
	splitDir := filepath.Join(processDir, "split")
	os.MkdirAll(splitDir, 0755)
	proc := audio.NewProcessor(*verbose)
	var splitFiles []string
	for _, seg := range segments {
		outPath := filepath.Join(splitDir, fmt.Sprintf("%03d_segment.mp3", seg.Index+1))
		if err := proc.CutSegment(*audioPath, seg.StartTime, seg.EndTime, outPath); err != nil {
			splitFiles = append(splitFiles, "")
			continue
		}
		splitFiles = append(splitFiles, outPath)
		fmt.Printf("Split segment %d: %.2fs - %.2fs\n", seg.Index+1, seg.StartTime.Seconds(), seg.EndTime.Seconds())
	}

	fmt.Println("\n=== Step 6: Generating TTS ===")
	ttsDir := filepath.Join(processDir, "tts")
	os.MkdirAll(ttsDir, 0755)
	ttsClient, _ := tts.NewGeminiTTS(apiKey, *verbose)
	var ttsFiles []string
	for _, seg := range segments {
		if !seg.IsMeaningful {
			ttsFiles = append(ttsFiles, "")
			fmt.Printf("TTS %d: [Sound-only - skipped]\n", seg.Index+1)
			continue
		}
		text := strings.TrimSpace(seg.TTSText)
		if text == "" {
			ttsFiles = append(ttsFiles, "")
			continue
		}
		ttsPath := filepath.Join(ttsDir, fmt.Sprintf("%03d_tts.mp3", seg.Index+1))
		fmt.Printf("TTS %d: %s\n", seg.Index+1, truncate(text, 40))
		if err := ttsClient.GenerateSpeech(ctx, text, ttsPath); err != nil {
			fmt.Printf("TTS failed %d: %v\n", seg.Index+1, err)
			ttsFiles = append(ttsFiles, "")
			continue
		}
		ttsFiles = append(ttsFiles, ttsPath)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\n=== Step 7: Merging audio (pattern: %s) ===\n", *pattern)
	outputPath := filepath.Join(processDir, "output.mp3")
	merger := audio.NewMerger(*verbose)
	if err := merger.MergeInterleaved(splitFiles, ttsFiles, outputPath, *pattern); err != nil {
		log.Fatalf("Failed to merge: %v", err)
	}
	info, _ := os.Stat(outputPath)
	fmt.Printf("Output saved: %s (%.2f MB)\n", outputPath, float64(info.Size())/(1024*1024))

	fmt.Println("\n=== Step 8: Generating output LRC ===")
	outputLRC := generateOutputLRC(segments, splitFiles, ttsFiles, *pattern)
	outputLRCPath := filepath.Join(processDir, "04_output.lrc")
	os.WriteFile(outputLRCPath, []byte(outputLRC), 0644)
	fmt.Printf("Output LRC saved: %s\n", outputLRCPath)

	// Embed LRC into output MP3
	outputWithLRCPath := filepath.Join(processDir, "output_with_lrc.mp3")
	if err := embedLRCToMP3(outputPath, outputLRC, outputWithLRCPath); err != nil {
		fmt.Printf("Warning: Failed to embed LRC: %v\n", err)
	} else {
		fmt.Printf("Output with LRC: %s\n", outputWithLRCPath)
	}

	fmt.Println("\n=== Done! ===")
}

func createProcessDir(audioPath string) (string, error) {
	base := filepath.Base(audioPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	re := regexp.MustCompile("[<>:\"/\\\\|?*]")
	safe := re.ReplaceAllString(name, "_")
	ts := time.Now().Format("20060102_150405")
	dir := filepath.Join("process", fmt.Sprintf("%s_%s", ts, safe))
	return dir, os.MkdirAll(dir, 0755)
}

func extractLRC(audioPath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format_tags=LYRICS", "-of", "default=noprint_wrappers=1:nokey=1", audioPath)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lrc := strings.TrimSpace(string(out))
	if lrc == "" {
		return "", fmt.Errorf("no LYRICS metadata")
	}
	return lrc, nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// getAudioDuration returns the duration of an audio file in seconds
func getAudioDuration(path string) float64 {
	if path == "" {
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

// formatLRCTime formats seconds to LRC timestamp [mm:ss.cc]
func formatLRCTime(seconds float64) string {
	mins := int(seconds) / 60
	secs := seconds - float64(mins*60)
	return fmt.Sprintf("%02d:%05.2f", mins, secs)
}

// isMetadataLine checks if a line is metadata (title, lyrics by, etc.)
func isMetadataLine(text string) bool {
	lower := strings.ToLower(text)
	patterns := []string{"lyrics by", "composed by", "作詞", "作曲", "編曲", "produced by", "written by"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	// Title line: "Song - Artist" format
	if strings.Contains(text, " - ") && !strings.Contains(lower, "если") {
		return true
	}
	return false
}

// generateOutputLRC creates an LRC file that matches the output.mp3 timeline
func generateOutputLRC(segments []segment.Segment, splitFiles, ttsFiles []string, pattern string) string {
	var sb strings.Builder
	sb.WriteString("[ti:Learning Track]\n")
	sb.WriteString("[ar:Multi-Language Learner]\n")
	sb.WriteString("[by:multilang-learner]\n\n")

	currentTime := 0.0

	for i, seg := range segments {
		splitDur := getAudioDuration(splitFiles[i])
		ttsDur := getAudioDuration(ttsFiles[i])

		// Skip metadata lines
		if isMetadataLine(seg.OriginalText) {
			currentTime += splitDur
			if ttsDur > 0 {
				currentTime += ttsDur
			}
			continue
		}

		if pattern == "original-tts" {
			// Original segment plays first
			ts := formatLRCTime(currentTime)
			// Write original text
			sb.WriteString(fmt.Sprintf("[%s]%s\n", ts, seg.OriginalText))
			currentTime += splitDur

			// TTS plays (if exists)
			if ttsDur > 0 {
				ttsTs := formatLRCTime(currentTime)
				// Write translation during TTS
				sb.WriteString(fmt.Sprintf("[%s]%s\n", ttsTs, seg.TTSText))
				currentTime += ttsDur
			}
		} else {
			// tts-original: TTS first, then original
			if ttsDur > 0 {
				ts := formatLRCTime(currentTime)
				sb.WriteString(fmt.Sprintf("[%s]%s\n", ts, seg.TTSText))
				currentTime += ttsDur
			}
			origTs := formatLRCTime(currentTime)
			sb.WriteString(fmt.Sprintf("[%s]%s\n", origTs, seg.OriginalText))
			currentTime += splitDur
		}
	}

	return sb.String()
}

// embedLRCToMP3 embeds LRC content into MP3 file
func embedLRCToMP3(inputMP3, lrcContent, outputPath string) error {
	// Create temp LRC file
	tempLRC := outputPath + ".temp.lrc"
	if err := os.WriteFile(tempLRC, []byte(lrcContent), 0644); err != nil {
		return err
	}
	defer os.Remove(tempLRC)

	// Use ffmpeg to embed lyrics as USLT tag
	cmd := exec.Command("ffmpeg", "-y",
		"-i", inputMP3,
		"-metadata", fmt.Sprintf("lyrics=%s", lrcContent),
		"-codec", "copy",
		outputPath,
	)
	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}
	return nil
}
