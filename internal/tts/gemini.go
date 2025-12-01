package tts

import (
"bytes"
"context"
"encoding/base64"
"encoding/binary"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"os/exec"
"path/filepath"
)

type GeminiTTS struct {
apiKey  string
baseURL string
verbose bool
}

func NewGeminiTTS(apiKey string, verbose bool) (*GeminiTTS, error) {
if apiKey == "" {
return nil, fmt.Errorf("API key required")
}
return &GeminiTTS{apiKey: apiKey, baseURL: "https://generativelanguage.googleapis.com/v1beta", verbose: verbose}, nil
}

type ttsReq struct {
Contents []content  `json:"contents"`
GenCfg   genConfig  `json:"generationConfig"`
}
type content struct {
Parts []part `json:"parts"`
}
type part struct {
Text string `json:"text"`
}
type genConfig struct {
ResponseModalities []string   `json:"responseModalities"`
SpeechConfig       *speechCfg `json:"speechConfig,omitempty"`
}
type speechCfg struct {
VoiceConfig voiceCfg `json:"voiceConfig"`
}
type voiceCfg struct {
PrebuiltVoiceConfig prebuiltVoice `json:"prebuiltVoiceConfig"`
}
type prebuiltVoice struct {
VoiceName string `json:"voiceName"`
}
type ttsResp struct {
Candidates []cand  `json:"candidates"`
Error      *apiErr `json:"error,omitempty"`
}
type apiErr struct {
Code    int    `json:"code"`
Message string `json:"message"`
}
type cand struct {
Content contentResp `json:"content"`
}
type contentResp struct {
Parts []partResp `json:"parts"`
}
type partResp struct {
InlineData *inlineData `json:"inlineData,omitempty"`
}
type inlineData struct {
MimeType string `json:"mimeType"`
Data     string `json:"data"`
}

// GenerateSpeech generates speech and saves as MP3
func (g *GeminiTTS) GenerateSpeech(ctx context.Context, text string, outputPath string) error {
pcmData, err := g.generatePCM(ctx, text)
if err != nil {
return err
}

os.MkdirAll(filepath.Dir(outputPath), 0755)

// Save as WAV first
wavPath := outputPath + ".wav"
if err := writeWAV(wavPath, pcmData, 1, 24000, 16); err != nil {
return fmt.Errorf("write WAV: %w", err)
}
defer os.Remove(wavPath)

// Convert to MP3
cmd := exec.Command("ffmpeg", "-y", "-i", wavPath, "-acodec", "libmp3lame", "-ar", "44100", "-ac", "2", "-b:a", "192k", outputPath)
if err := cmd.Run(); err != nil {
return fmt.Errorf("ffmpeg convert: %w", err)
}

return nil
}

func (g *GeminiTTS) generatePCM(ctx context.Context, text string) ([]byte, error) {
url := fmt.Sprintf("%s/models/gemini-2.5-flash-preview-tts:generateContent?key=%s", g.baseURL, g.apiKey)
req := ttsReq{
Contents: []content{{Parts: []part{{Text: text}}}},
GenCfg: genConfig{
ResponseModalities: []string{"AUDIO"},
SpeechConfig:       &speechCfg{VoiceConfig: voiceCfg{PrebuiltVoiceConfig: prebuiltVoice{VoiceName: "Kore"}}},
},
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
var ttsR ttsResp
if err := json.Unmarshal(body, &ttsR); err != nil {
return nil, fmt.Errorf("parse error: %w", err)
}
if ttsR.Error != nil {
return nil, fmt.Errorf("API error [%d]: %s", ttsR.Error.Code, ttsR.Error.Message)
}
if len(ttsR.Candidates) > 0 && len(ttsR.Candidates[0].Content.Parts) > 0 {
for _, p := range ttsR.Candidates[0].Content.Parts {
if p.InlineData != nil && p.InlineData.Data != "" {
return base64.StdEncoding.DecodeString(p.InlineData.Data)
}
}
}
return nil, fmt.Errorf("no audio data in response")
}

func writeWAV(filename string, pcm []byte, channels, sampleRate, bitsPerSample int) error {
f, err := os.Create(filename)
if err != nil {
return err
}
defer f.Close()

byteRate := sampleRate * channels * bitsPerSample / 8
blockAlign := channels * bitsPerSample / 8
dataSize := len(pcm)

f.Write([]byte("RIFF"))
binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
f.Write([]byte("WAVE"))
f.Write([]byte("fmt "))
binary.Write(f, binary.LittleEndian, uint32(16))
binary.Write(f, binary.LittleEndian, uint16(1))
binary.Write(f, binary.LittleEndian, uint16(channels))
binary.Write(f, binary.LittleEndian, uint32(sampleRate))
binary.Write(f, binary.LittleEndian, uint32(byteRate))
binary.Write(f, binary.LittleEndian, uint16(blockAlign))
binary.Write(f, binary.LittleEndian, uint16(bitsPerSample))
f.Write([]byte("data"))
binary.Write(f, binary.LittleEndian, uint32(dataSize))
f.Write(pcm)

return nil
}
