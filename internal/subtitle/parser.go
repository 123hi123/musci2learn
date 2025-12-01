package subtitle

import (
"fmt"
"os"
"regexp"
"sort"
"strconv"
"strings"
"time"
)

type Line struct {
StartTime   time.Duration
EndTime     time.Duration
Text        string
Translation string
}

type Lyrics struct {
Title  string
Artist string
Album  string
Lines  []Line
}

type Parser struct{}

func NewParser() *Parser {
return &Parser{}
}

func (p *Parser) Parse(content string) (*Lyrics, error) {
result := &Lyrics{}
lines := strings.Split(content, "\n")

metaRe := regexp.MustCompile(`^\[(\w+):(.+)\]$`)
timeRe := regexp.MustCompile(`^\[(\d{1,2}):(\d{2})(?:[\.:](\d{2,3}))?\](.*)$`)

type rawLine struct {
time  time.Duration
texts []string
}
var rawLines []rawLine
timeMap := make(map[time.Duration]int)

for _, line := range lines {
line = strings.TrimSpace(line)
if line == "" {
continue
}

if metaMatch := metaRe.FindStringSubmatch(line); metaMatch != nil {
tag := strings.ToLower(metaMatch[1])
value := strings.TrimSpace(metaMatch[2])
switch tag {
case "ti", "title":
result.Title = value
case "ar", "artist":
result.Artist = value
case "al", "album":
result.Album = value
}
continue
}

if timeMatch := timeRe.FindStringSubmatch(line); timeMatch != nil {
min, _ := strconv.Atoi(timeMatch[1])
sec, _ := strconv.Atoi(timeMatch[2])
ms := 0
if timeMatch[3] != "" {
ms, _ = strconv.Atoi(timeMatch[3])
if len(timeMatch[3]) == 2 {
ms *= 10
}
}
t := time.Duration(min)*time.Minute + time.Duration(sec)*time.Second + time.Duration(ms)*time.Millisecond
text := strings.TrimSpace(timeMatch[4])
if text == "" || text == "//" {
continue
}
if idx, exists := timeMap[t]; exists {
rawLines[idx].texts = append(rawLines[idx].texts, text)
} else {
timeMap[t] = len(rawLines)
rawLines = append(rawLines, rawLine{time: t, texts: []string{text}})
}
}
}

sort.Slice(rawLines, func(i, j int) bool { return rawLines[i].time < rawLines[j].time })

for i, raw := range rawLines {
l := Line{StartTime: raw.time}
if len(raw.texts) > 0 {
l.Text = raw.texts[0]
}
if len(raw.texts) > 1 {
l.Translation = raw.texts[1]
}
if i < len(rawLines)-1 {
l.EndTime = rawLines[i+1].time
} else {
l.EndTime = raw.time + 5*time.Second
}
result.Lines = append(result.Lines, l)
}
return result, nil
}

func (p *Parser) GenerateEnhancedLRC(lyrics *Lyrics) string {
var sb strings.Builder
if lyrics.Title != "" {
sb.WriteString(fmt.Sprintf("[ti:%s]\n", lyrics.Title))
}
if lyrics.Artist != "" {
sb.WriteString(fmt.Sprintf("[ar:%s]\n", lyrics.Artist))
}
sb.WriteString("\n")
for _, line := range lyrics.Lines {
ts := formatTime(line.StartTime)
sb.WriteString(fmt.Sprintf("[%s]%s\n", ts, line.Text))
if line.Translation != "" {
sb.WriteString(fmt.Sprintf("[%s]%s\n", ts, line.Translation))
}
}
return sb.String()
}

func (p *Parser) ParseFile(path string) (*Lyrics, error) {
data, err := os.ReadFile(path)
if err != nil {
return nil, err
}
return p.Parse(string(data))
}

func formatTime(d time.Duration) string {
min := int(d.Minutes())
sec := int(d.Seconds()) % 60
ms := int(d.Milliseconds()) % 1000 / 10
return fmt.Sprintf("%02d:%02d.%02d", min, sec, ms)
}
