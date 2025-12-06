# 多語言學習器 (Multi-Language Learner)

一個使用 Go 和 Gemini API 開發的語言學習工具，透過將原始音訊與 AI 複誦音訊交替播放的方式，幫助使用者進行語言學習。

## 功能特色

- 🎵 **音訊導入**：支援多種音訊格式（需 ffmpeg）
- 📝 **LRC 字幕解析**：自動解析 LRC 格式時間軸字幕
- ✂️ **智能切割**：根據字幕時間戳自動切割音訊片段
- 🤖 **AI 複誦**：使用 Gemini API 生成學習用複誦音訊
- 🔄 **交替播放**：原始音訊 → AI 複誦 → 原始音訊
- 🌐 **Web 介面**：現代化的網頁操作介面

## 前置需求

1. **Go 1.21+** - [下載 Go](https://golang.org/dl/)
2. **FFmpeg** - 用於音訊處理
   - Windows: `winget install ffmpeg` 或從 [官網下載](https://ffmpeg.org/download.html)
   - macOS: `brew install ffmpeg`
   - Linux: `sudo apt install ffmpeg`
3. **Gemini API Key** - 從 [Google AI Studio](https://aistudio.google.com/app/apikey) 免費取得

## 快速開始

### 1. 克隆專案
```bash
git clone https://github.com/your-username/multilang-learner.git
cd multilang-learner
```

### 2. 安裝依賴
```bash
go mod tidy
```

### 3. 設定 API Key
```bash
# 複製範例設定檔
cp .env.example .env

# 編輯 .env 檔案，填入你的 Gemini API Key
# GEMINI_API_KEY=your_actual_api_key_here
```

### 4. 啟動伺服器
```bash
go run cmd/server/main.go
```

### 5. 開啟瀏覽器
訪問 http://localhost:8080

## 使用方式

1. **上傳音檔**：點擊上傳按鈕，選擇帶有 LRC 歌詞的音訊檔案（.flac, .mp3 等）
2. **調整設定**：選擇起始行、目標語言
3. **處理音檔**：點擊「開始處理」，系統會自動翻譯並生成 TTS
4. **練習模式**：處理完成後進入練習模式，開始學習！

## 專案結構

```
multilang-learner/
├── cmd/
│   └── server/
│       ├── main.go          # Web 伺服器入口
│       └── handlers.go      # API 處理器
├── internal/
│   ├── audio/               # 音訊處理
│   ├── models/              # 資料模型
│   ├── services/            # 業務邏輯
│   ├── translator/          # 翻譯服務
│   └── tts/                 # TTS 服務
├── web/
│   ├── templates/           # HTML 模板
│   └── static/              # CSS, JS
├── data/                    # 使用者資料（不會上傳）
├── .env.example             # 環境變數範例
└── .gitignore
```

## 環境變數

| 變數 | 說明 | 必填 |
|------|------|------|
| `GEMINI_API_KEY` | Gemini API 金鑰 | ✅ |
| `PORT` | 伺服器埠號 | ❌ (預設 8080) |

## 注意事項

- 📁 `data/` 資料夾包含使用者上傳的音檔和處理結果，不會被 Git 追蹤
- 🔑 請勿將 `.env` 檔案上傳到公開儲存庫
- 🎵 請確保你有權使用上傳的音樂檔案

## 授權

MIT License
