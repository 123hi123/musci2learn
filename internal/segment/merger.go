package segment

import (
	"fmt"
	"time"

	"multilang-learner/internal/subtitle"
)

// Segment represents a merged segment with multiple lines
type Segment struct {
	Index        int             // Segment index (0-based)
	Lines        []subtitle.Line // Original lines in this segment
	StartTime    time.Duration   // Start time of the segment
	EndTime      time.Duration   // End time of the segment
	Duration     time.Duration   // Total duration
	OriginalText string          // Combined original text
	TTSText      string          // Combined translation text for TTS
	IsMetadata   bool            // Whether this segment is metadata (not lyrics)
	IsMeaningful bool            // Whether this segment has meaningful content (not just interjections)
}

// Merger handles the 5-second minimum segment merging logic
type Merger struct {
	minDuration time.Duration
	verbose     bool
}

// NewMerger creates a new segment merger
func NewMerger(minDuration time.Duration, verbose bool) *Merger {
	if minDuration <= 0 {
		minDuration = 5 * time.Second
	}
	return &Merger{
		minDuration: minDuration,
		verbose:     verbose,
	}
}

// MergeByDuration merges ALL lines into segments of at least minDuration
// This is the main entry point - merges everything first, then we can analyze meaning later
func (m *Merger) MergeByDuration(lines []subtitle.Line) []Segment {
	if len(lines) == 0 {
		return nil
	}

	var segments []Segment
	currentSegment := Segment{
		Index:        0,
		IsMeaningful: true, // Assume meaningful by default
	}

	for i, line := range lines {
		// If this is the first line in current segment
		if len(currentSegment.Lines) == 0 {
			currentSegment.Lines = []subtitle.Line{line}
			currentSegment.StartTime = line.StartTime
			currentSegment.EndTime = line.EndTime
			currentSegment.OriginalText = line.Text
			currentSegment.TTSText = line.Translation
			if currentSegment.TTSText == "" {
				currentSegment.TTSText = line.Text
			}
			currentSegment.Duration = currentSegment.EndTime - currentSegment.StartTime
			continue
		}

		// Check if current segment is long enough
		if currentSegment.Duration >= m.minDuration {
			// Save current segment and start new one
			segments = append(segments, currentSegment)

			currentSegment = Segment{
				Index:        len(segments),
				Lines:        []subtitle.Line{line},
				StartTime:    line.StartTime,
				EndTime:      line.EndTime,
				OriginalText: line.Text,
				TTSText:      line.Translation,
				IsMeaningful: true,
			}
			if currentSegment.TTSText == "" {
				currentSegment.TTSText = line.Text
			}
			currentSegment.Duration = currentSegment.EndTime - currentSegment.StartTime
		} else {
			// Merge this line into current segment
			currentSegment.Lines = append(currentSegment.Lines, line)
			currentSegment.EndTime = line.EndTime
			currentSegment.OriginalText += " " + line.Text

			// Append translation for TTS
			trans := line.Translation
			if trans == "" {
				trans = line.Text
			}
			currentSegment.TTSText += " " + trans
			currentSegment.Duration = currentSegment.EndTime - currentSegment.StartTime
		}

		// Handle last line
		if i == len(lines)-1 && len(currentSegment.Lines) > 0 {
			segments = append(segments, currentSegment)
		}
	}

	// If we still have an unfinished segment
	if len(segments) == 0 || segments[len(segments)-1].Index != currentSegment.Index {
		if len(currentSegment.Lines) > 0 {
			segments = append(segments, currentSegment)
		}
	}

	// Re-index segments
	for i := range segments {
		segments[i].Index = i
	}

	return segments
}

// MarkMetadata marks segments as metadata based on analysis result
func (m *Merger) MarkMetadata(segments []Segment, metadataIndices []int) {
	// Build a set of segment indices that are metadata
	metadataSet := make(map[int]bool)
	for _, idx := range metadataIndices {
		metadataSet[idx] = true
	}

	// Mark segments based on their index
	for i := range segments {
		segments[i].IsMetadata = metadataSet[i]
	}
}

// MarkMeaningfulness marks segments as meaningful or not
func (m *Merger) MarkMeaningfulness(segments []Segment, meaningfulIndices []int) {
	// Build set of meaningful segment indices
	meaningfulSet := make(map[int]bool)
	for _, idx := range meaningfulIndices {
		meaningfulSet[idx] = true
	}

	for i := range segments {
		segments[i].IsMeaningful = meaningfulSet[i]
	}
}

// GetSegmentInfo returns a summary of merged segments
func (m *Merger) GetSegmentInfo(segments []Segment) string {
	var result string
	for _, seg := range segments {
		typeStr := "LYRICS"
		if seg.IsMetadata {
			typeStr = "META"
		}
		meaningStr := ""
		if !seg.IsMeaningful {
			meaningStr = " [NO-TTS]"
		}
		result += fmt.Sprintf("Segment %d [%s%s]: %.2fs, %d lines - %s\n",
			seg.Index+1, typeStr, meaningStr, seg.Duration.Seconds(), len(seg.Lines), truncateText(seg.OriginalText, 40))
	}
	return result
}

// GetTTSSegments returns only segments that should have TTS generated
func (m *Merger) GetTTSSegments(segments []Segment) []Segment {
	var result []Segment
	for _, seg := range segments {
		if !seg.IsMetadata && seg.IsMeaningful {
			result = append(result, seg)
		}
	}
	return result
}

func truncateText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
