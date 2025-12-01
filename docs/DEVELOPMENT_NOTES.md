# 多語言學習器 - 開發紀錄與補充說明

## 概述

本文檔記錄開發過程中的技術發現、問題解決方案和額外實作的功能。

---

## 開發時間線

| 日期 | 主要工作 |
|------|----------|
| 2025/12/01 | 完整系統開發、測試、問題修復 |

---

## 主要開發成果

### 1. 核心功能模組

| 模組 | 檔案位置 | 功能說明 |
|------|----------|----------|
| 主程式 | `cmd/main.go` | 7 步驟流程控制 |
| LRC 解析 | `internal/lrc/parser.go` | 解析 LRC 時間戳和歌詞 |
| 翻譯模組 | `internal/translator/gemini.go` | Gemini API 翻譯 |
| 段落合併 | `internal/segment/merger.go` | 5秒規則合併 |
| 意義分析 | `internal/analyzer/analyzer.go` | 判斷歌詞是否有意義 |
| 語言檢測 | `internal/langdetect/detector.go` | 檢測翻譯語言正確性 |
| TTS 生成 | `internal/tts/gemini.go` | Gemini TTS 語音合成 |
| 音訊處理 | `internal/audio/splitter.go` | FFmpeg 音訊切割合併 |
| LRC 嵌入 | `cmd/embed_lrc/main.go` | MP3 歌詞嵌入工具 |

### 2. 獨立工具

| 工具 | 位置 | 用途 |
|------|------|------|
| embed_lyrics.py | 根目錄 | Python 歌詞嵌入腳本 |
| embed_lrc | `cmd/embed_lrc/` | Go 歌詞嵌入工具 |

---

## 技術發現與解決方案

### 問題 1: Gemini TTS 返回 PCM 格式

**發現**: Gemini 2.5 Flash Preview TTS API 返回的是原始 PCM 數據，不是 MP3。

**解決方案**:
```
PCM (24kHz, 16-bit, mono) → WAV → MP3
```

**程式碼位置**: `internal/tts/gemini.go`

### 問題 2: 翻譯語言錯誤

**發現**: 要求翻譯成繁體中文，但 Gemini 返回簡體中文。

**解決方案**: 
- 新增語言檢測模組 (`internal/langdetect/`)
- 使用 `whatlanggo` + Unicode 漢字範圍檢測
- 改為翻譯成英文（TTS 效果更好）

### 問題 3: 段落時長太短

**發現**: 原始歌詞每行只有 2-3 秒，學習效果不佳。

**解決方案**:
- 實作 5 秒最小段落合併邏輯
- 相鄰歌詞合併直到達到 5 秒

**程式碼位置**: `internal/segment/merger.go`

### 問題 4: 無意義歌詞的 TTS

**發現**: 純感嘆詞 (如 "啊啊啊", "喔喔喔") 翻譯和 TTS 無意義。

**解決方案**:
- 新增意義分析模組
- 使用 Gemini 判斷歌詞是否有實際意義
- 無意義段落跳過 TTS 生成

**程式碼位置**: `internal/analyzer/analyzer.go`

### 問題 5: LRC 嵌入 MP3 失敗

**發現**: 嵌入的歌詞無法在播放器顯示。

**排查過程**:
1. FFmpeg `-metadata lyrics=` → 失敗 (格式錯誤，每行加 `\`)
2. Python mutagen USLT → 失敗 (播放器不支援)
3. Python mutagen TXXX:USLT → 失敗 (缺少必要標籤)
4. 添加 TIT2, TPE1, TALB → 部分成功
5. 發現歌詞行太長 (160字元) → 失敗
6. 格式化歌詞 (≤45字元/行) → **成功**

**最終解決方案**:
```python
from mutagen.mp3 import MP3
from mutagen.id3 import TXXX, TIT2, TPE1, TALB, Encoding

audio = MP3('output.mp3')
audio.tags.add(TIT2(encoding=Encoding.UTF8, text=['標題']))
audio.tags.add(TPE1(encoding=Encoding.UTF8, text=['歌手']))
audio.tags.add(TALB(encoding=Encoding.UTF8, text=['專輯']))
audio.tags.add(TXXX(encoding=Encoding.UTF8, desc='USLT', text=lrc_content))
audio.save()
```

**關鍵發現**:
- 必須包含 TIT2, TPE1, TALB 標籤
- 歌詞每行不能超過 ~45 字元
- 使用 TXXX:USLT 格式 (desc='USLT')

---

## LRC 格式研究結果

### 標準 LRC 格式

```
[ti:歌曲標題]      # 可選
[ar:歌手]         # 可選
[al:專輯]         # 可選
[by:作者]         # 可選
[offset:0]        # 時間偏移 (毫秒)
[mm:ss.xx] 歌詞內容
```

### 播放器相容性要求

| 項目 | 建議值 | 說明 |
|------|--------|------|
| 每行長度 | ≤ 45 字元 | 超過可能無法顯示 |
| 時間戳格式 | [mm:ss.xx] | 分:秒.百分秒 |
| 時間戳後空格 | 必須有 | `[00:00.00] 歌詞` |
| 標頭 offset | 建議有 | `[offset:0]` |

### MP3 歌詞標籤格式

| 標籤類型 | 說明 | 相容性 |
|----------|------|--------|
| USLT | 標準非同步歌詞 | 部分播放器 |
| SYLT | 標準同步歌詞 (二進制) | 少數播放器 |
| TXXX:USLT | 自訂文字標籤 | **推薦** |
| TXXX:LYRICS | 自訂文字標籤 | 部分播放器 |

---

## 測試檔案說明

開發過程中產生的測試檔案：

| 檔案 | 用途 | 結果 |
|------|------|------|
| test_convert.mp3 | 原始 FLAC 轉換 | ✅ 成功 |
| test_direct_convert.mp3 | 重新從 FLAC 轉換 | ✅ 成功 |
| test_remerged_full.mp3 | 拆開重新合併 | ✅ 成功 |
| test_output_with_original_lrc.mp3 | output + 原始歌詞 | ✅ 成功 |
| test_remerged_python.mp3 | 少標籤版本 | ❌ 失敗 |
| test_with_short_lrc.mp3 | 截短歌詞 | ❌ 失敗 |
| test_simple_format.mp3 | 簡單格式測試 | ✅ 成功 |

---

## 依賴套件

### Go 模組

```go
require (
    github.com/abadojack/whatlanggo v1.0.1  // 語言檢測
)
```

### Python 套件

```
mutagen  // MP3 ID3 標籤處理
```

### 系統工具

```
FFmpeg 8.0.1  // 音訊處理
```

---

## API 使用說明

### Gemini 2.0 Flash (翻譯/分析)

```
模型: gemini-2.0-flash
用途: 翻譯歌詞、分析段落意義
```

### Gemini 2.5 Flash Preview TTS

```
模型: gemini-2.5-flash-preview-tts
輸出格式: PCM (audio/L16;codec=pcm;rate=24000)
轉換流程: PCM → WAV → MP3
```

---

## 未來改進建議

1. **支援更多輸入格式**: MP3, OGG, WAV 等
2. **多語言 TTS**: 支援中文、日文等 TTS
3. **自動歌詞獲取**: 整合線上歌詞 API
4. **GUI 介面**: 開發圖形化操作介面
5. **批次處理**: 支援多檔案批次轉換
6. **學習模式選擇**: 
   - 原文 → 翻譯
   - 翻譯 → 原文
   - 僅原文
   - 僅翻譯

---

## 參考資料

- [LRC (file format) - Wikipedia](https://en.wikipedia.org/wiki/LRC_(file_format))
- [ID3v2 標準](https://id3.org/id3v2.4.0-frames)
- [Mutagen 文檔](https://mutagen.readthedocs.io/)
- [FFmpeg 文檔](https://ffmpeg.org/documentation.html)
- [Gemini API 文檔](https://ai.google.dev/docs)

---

## 檔案清理建議

開發完成後可刪除的測試檔案：

```
test_*.mp3
test_*.lrc
test_*.txt
test_metadata.txt
```

保留的重要檔案：

```
docs/PROCESSING_FLOW.md      # 處理流程文檔
docs/DEVELOPMENT_NOTES.md    # 本文檔
docs/LRC_EMBEDDING_TEST_RESULTS.md  # 測試結果
```
