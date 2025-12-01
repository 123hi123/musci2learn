# 多語言學習器 - Web 應用設計文檔

## 系統概述

一個 Web 介面的語言學習應用，用戶上傳音樂檔案後，系統解析歌詞、翻譯、生成 TTS，最終提供互動式學習播放器。

---

## 功能需求

### 1. 檔案管理
- 左側檔案列表顯示所有上傳的音樂
- 點選檔案後，右側顯示詳細資訊和操作選項
- 支援刪除檔案

### 2. 上傳與解析
- 上傳音樂檔案 (FLAC, MP3 等)
- 自動解析內嵌歌詞 (LRC 格式)
- 解析後在前端顯示歌詞列表

### 3. 歌詞起點選擇
- **手動選擇**: 用戶點選某行歌詞，設定為「正式開始」位置
- **AI 自動判斷**: 系統自動識別歌詞正式開始位置
- 起點之前的歌詞會被忽略（如歌曲資訊、作詞作曲等）

### 4. 語言設定
- **主要語言**: 用戶希望學習的語言（用於 TTS）
- 支援語言:
  - 英文 (English)
  - 中文 (Chinese)
- 未來可擴展更多語言

### 5. 字幕/翻譯管理
- **原文字幕**: 從音樂檔案解析出的原始歌詞
- **檔案內附翻譯**: 如果音樂檔案內有翻譯（如中文翻譯），保留使用
- **AI 生成翻譯**: 自動翻譯成主要語言
- **中文輔助翻譯**: 可選開啟，顯示中文翻譯幫助理解

### 6. 播放器功能
- **播放順序**: 原曲段落 → 主要語言 TTS (可重複 N 次) → 下一段
- **重複次數設定**: 用戶可設定 TTS 播放次數 (1, 2, 3...)
- **循環播放**: 整首播放完後自動從頭開始
- **字幕同步顯示**:
  - 播放原曲時: 顯示原文歌詞
  - 播放 TTS 時: 顯示主要語言翻譯
- **中文翻譯開關**: 可選顯示中文翻譯

### 7. 導出功能
- 導出合併後的音檔 (MP3)
- 不嵌入字幕
- 結構: 原曲 + TTS (按設定次數) + 原曲 + TTS...

---

## 介面設計

```
┌─────────────────────────────────────────────────────────────────────────┐
│  🎵 多語言學習器                                          [設定] [說明] │
├─────────────┬───────────────────────────────────────────────────────────┤
│             │  ┌─────────────────────────────────────────────────────┐  │
│  檔案列表    │  │  檔案名稱: Умри если меня не любишь.flac           │  │
│  ─────────  │  │  時長: 2:30  |  歌詞行數: 50                        │  │
│             │  ├─────────────────────────────────────────────────────┤  │
│  📁 歌曲1   │  │  設定                                               │  │
│  📁 歌曲2   │  │  ┌────────────────┐  ┌────────────────┐            │  │
│  📁 歌曲3   │  │  │ 主要語言: 英文 ▼│  │ TTS重複: 2次 ▼ │            │  │
│             │  │  └────────────────┘  └────────────────┘            │  │
│             │  │  ☑ 顯示中文翻譯                                     │  │
│             │  ├─────────────────────────────────────────────────────┤  │
│             │  │  歌詞 (點選設定起點)                    [AI自動判斷] │  │
│             │  │  ─────────────────────────────────────────────────  │  │
│             │  │  ⚪ [00:00.00] 歌曲標題                (忽略)       │  │
│             │  │  ⚪ [00:00.90] Lyrics by: XXX          (忽略)       │  │
│  ─────────  │  │  ────────── ▲ 起點線 ▲ ──────────                  │  │
│             │  │  🔘 [00:01.79] Шаг за 20 руки мокрые   ← 正在播放   │  │
│  [+ 上傳]   │  │     → Step for 20, hands wet                        │  │
│             │  │     → 20步的距离 潮湿的双手                          │  │
│             │  │  ⚪ [00:04.42] Мне не хватит ни сил...               │  │
│             │  │     → I won't have enough strength...               │  │
│             │  │     → 我甚至无法抓住你的一缕卷发                      │  │
│             │  ├─────────────────────────────────────────────────────┤  │
│             │  │            advancement                               │  │
│             │  │  ◀◀  │  ▶ 播放  │  ▶▶  │  🔁 循環  │  ────●────  │  │
│             │  │                                                     │  │
│             │  │  [處理音檔]  [導出 MP3]                              │  │
│             │  └─────────────────────────────────────────────────────┘  │
└─────────────┴───────────────────────────────────────────────────────────┘
```

---

## 資料結構

### 音樂檔案 (MusicFile)
```json
{
  "id": "uuid",
  "filename": "歌曲名.flac",
  "filepath": "/uploads/xxx.flac",
  "duration": 150.5,
  "uploadedAt": "2025-12-01T00:00:00Z",
  "status": "parsed|processing|ready",
  "settings": {
    "primaryLanguage": "en",
    "ttsRepeatCount": 2,
    "startLineIndex": 5,
    "showChineseTranslation": true
  }
}
```

### 歌詞行 (LyricLine)
```json
{
  "index": 0,
  "timestamp": "00:01.79",
  "startTime": 1.79,
  "endTime": 4.42,
  "original": "Шаг за 20 руки мокрые",
  "translations": {
    "embedded": "20步的距离 潮湿的双手",
    "en": "Step for 20, hands wet",
    "zh": "20步的距离 潮湿的双手"
  },
  "isMeaningful": true
}
```

### 段落 (Segment)
```json
{
  "index": 1,
  "startTime": 1.79,
  "endTime": 13.80,
  "duration": 12.01,
  "lines": [0, 1, 2, 3],
  "originalText": "合併的原文...",
  "ttsText": "合併的翻譯...",
  "isMeaningful": true,
  "audioPath": "/segments/segment_001.mp3",
  "ttsPath": "/tts/tts_001.mp3"
}
```

---

## API 設計

### 檔案管理

| Method | Endpoint | 說明 |
|--------|----------|------|
| GET | /api/files | 獲取所有檔案列表 |
| POST | /api/files/upload | 上傳音樂檔案 |
| GET | /api/files/:id | 獲取檔案詳情 |
| DELETE | /api/files/:id | 刪除檔案 |

### 歌詞與處理

| Method | Endpoint | 說明 |
|--------|----------|------|
| GET | /api/files/:id/lyrics | 獲取解析的歌詞 |
| POST | /api/files/:id/settings | 更新檔案設定 |
| POST | /api/files/:id/process | 開始處理（翻譯、切割、TTS） |
| GET | /api/files/:id/status | 獲取處理進度 |

### 播放與導出

| Method | Endpoint | 說明 |
|--------|----------|------|
| GET | /api/files/:id/segments | 獲取段落列表 |
| GET | /api/files/:id/segments/:idx/audio | 獲取段落音訊 |
| GET | /api/files/:id/segments/:idx/tts | 獲取段落 TTS |
| POST | /api/files/:id/export | 導出合併音檔 |
| GET | /api/files/:id/export/download | 下載導出的音檔 |

### AI 功能

| Method | Endpoint | 說明 |
|--------|----------|------|
| POST | /api/files/:id/detect-start | AI 自動判斷歌詞起點 |
| POST | /api/files/:id/translate | 翻譯歌詞 |

---

## 目錄結構

```
musci2learn/
├── cmd/
│   └── server/
│       └── main.go           # Web 服務入口
├── internal/
│   ├── api/
│   │   ├── router.go         # 路由設定
│   │   ├── handlers.go       # API 處理器
│   │   └── middleware.go     # 中間件
│   ├── models/
│   │   ├── file.go           # 檔案模型
│   │   ├── lyric.go          # 歌詞模型
│   │   └── segment.go        # 段落模型
│   ├── services/
│   │   ├── file_service.go   # 檔案服務
│   │   ├── lyric_service.go  # 歌詞服務
│   │   ├── process_service.go # 處理服務
│   │   └── export_service.go # 導出服務
│   ├── lrc/                  # LRC 解析 (已有)
│   ├── translator/           # 翻譯模組 (已有)
│   ├── tts/                  # TTS 模組 (已有)
│   ├── audio/                # 音訊處理 (已有)
│   ├── segment/              # 段落合併 (已有)
│   ├── analyzer/             # 意義分析 (已有)
│   └── langdetect/           # 語言檢測 (已有)
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css
│   │   └── js/
│   │       └── app.js
│   └── templates/
│       └── index.html
├── uploads/                  # 上傳的檔案
├── data/                     # 處理後的資料
│   └── {file_id}/
│       ├── original.flac
│       ├── lyrics.json
│       ├── segments/
│       └── tts/
└── docs/
```

---

## 技術選型

| 項目 | 技術 |
|------|------|
| 後端框架 | Go + Gin |
| 前端 | 原生 HTML + CSS + JavaScript |
| 音訊播放 | HTML5 Audio API |
| 資料存儲 | 檔案系統 (JSON) |
| 音訊處理 | FFmpeg |
| AI | Gemini API |

---

## 開發順序

1. **Phase 1: 基礎架構**
   - Web 服務器啟動
   - 靜態檔案服務
   - 基本 API 路由

2. **Phase 2: 檔案上傳與解析**
   - 檔案上傳 API
   - LRC 解析
   - 歌詞列表顯示

3. **Phase 3: 設定與處理**
   - 起點選擇功能
   - 語言設定
   - 翻譯與 TTS 生成

4. **Phase 4: 播放器**
   - 音訊播放控制
   - 字幕同步
   - 循環播放

5. **Phase 5: 導出功能**
   - 音檔合併
   - 下載功能
