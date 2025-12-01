package main

import (
"bytes"
"encoding/base64"
"encoding/binary"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
)

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

func main() {
apiKey := os.Getenv("GOOGLE_API_KEY")
if apiKey == "" {
fmt.Println("GOOGLE_API_KEY not set")
return
}

text := "Hello! This is a test of the Gemini text to speech API."
fmt.Printf("Generating TTS for: %s\n", text)

url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-preview-tts:generateContent?key=%s", apiKey)

req := ttsReq{
Contents: []content{{Parts: []part{{Text: text}}}},
GenCfg: genConfig{
ResponseModalities: []string{"AUDIO"},
SpeechConfig: &speechCfg{
VoiceConfig: voiceCfg{
PrebuiltVoiceConfig: prebuiltVoice{VoiceName: "Kore"},
},
},
},
}

jsonData, _ := json.Marshal(req)
fmt.Printf("Request: %s\n\n", string(jsonData))

httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
httpReq.Header.Set("Content-Type", "application/json")

resp, err := http.DefaultClient.Do(httpReq)
if err != nil {
fmt.Printf("HTTP error: %v\n", err)
return
}
defer resp.Body.Close()

body, _ := io.ReadAll(resp.Body)
fmt.Printf("Response status: %d\n", resp.StatusCode)

var ttsR ttsResp
if err := json.Unmarshal(body, &ttsR); err != nil {
fmt.Printf("JSON parse error: %v\n", err)
fmt.Printf("Body: %s\n", string(body))
return
}

if ttsR.Error != nil {
fmt.Printf("API error: code=%d, message=%s\n", ttsR.Error.Code, ttsR.Error.Message)
return
}

if len(ttsR.Candidates) > 0 && len(ttsR.Candidates[0].Content.Parts) > 0 {
for _, p := range ttsR.Candidates[0].Content.Parts {
if p.InlineData != nil {
fmt.Printf("MimeType: %s\n", p.InlineData.MimeType)
fmt.Printf("Data length: %d\n", len(p.InlineData.Data))

pcmData, err := base64.StdEncoding.DecodeString(p.InlineData.Data)
if err != nil {
fmt.Printf("Base64 decode error: %v\n", err)
return
}
fmt.Printf("PCM data size: %d bytes\n", len(pcmData))

// Save as WAV file (16-bit, 24kHz, mono)
wavPath := "test/test_output.wav"
if err := writeWAV(wavPath, pcmData, 1, 24000, 16); err != nil {
fmt.Printf("WAV write error: %v\n", err)
return
}
fmt.Printf("Saved WAV to: %s\n", wavPath)
return
}
}
}
fmt.Println("No audio data in response")
fmt.Printf("Full response: %s\n", string(body))
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

// RIFF header
f.Write([]byte("RIFF"))
binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
f.Write([]byte("WAVE"))

// fmt chunk
f.Write([]byte("fmt "))
binary.Write(f, binary.LittleEndian, uint32(16))
binary.Write(f, binary.LittleEndian, uint16(1)) // PCM
binary.Write(f, binary.LittleEndian, uint16(channels))
binary.Write(f, binary.LittleEndian, uint32(sampleRate))
binary.Write(f, binary.LittleEndian, uint32(byteRate))
binary.Write(f, binary.LittleEndian, uint16(blockAlign))
binary.Write(f, binary.LittleEndian, uint16(bitsPerSample))

// data chunk
f.Write([]byte("data"))
binary.Write(f, binary.LittleEndian, uint32(dataSize))
f.Write(pcm)

return nil
}
