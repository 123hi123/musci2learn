"""
多語言學習器 - 主程式
使用 Gemini API 生成 TTS 並與原始音訊交替播放
"""

import os
import re
import sys
import argparse
from pathlib import Path
from dataclasses import dataclass
from typing import List, Optional
import subprocess
import tempfile
import shutil

# Google Generative AI
import google.generativeai as genai
from google.generativeai import types


@dataclass
class LyricLine:
    """歌詞行"""
    start_time: float  # 開始時間（秒）
    end_time: float    # 結束時間（秒）
    text: str          # 歌詞文字


def parse_lrc(content: str) -> List[LyricLine]:
    """解析 LRC 格式字幕"""
    lyrics = []
    # 匹配 [mm:ss.xx] 或 [mm:ss:xx] 格式
    pattern = r'\[(\d{1,2}):(\d{2})[\.:](\d{2,3})\](.*)'
    
    for line in content.split('\n'):
        line = line.strip()
        if not line:
            continue
        
        match = re.match(pattern, line)
        if match:
            minutes = int(match.group(1))
            seconds = int(match.group(2))
            ms_str = match.group(3)
            
            # 處理毫秒
            ms = int(ms_str)
            if len(ms_str) == 2:
                ms *= 10
            
            start_time = minutes * 60 + seconds + ms / 1000
            text = match.group(4).strip()
            
            if text:
                lyrics.append(LyricLine(
                    start_time=start_time,
                    end_time=0,  # 稍後計算
                    text=text
                ))
    
    # 按時間排序
    lyrics.sort(key=lambda x: x.start_time)
    
    # 計算結束時間
    for i in range(len(lyrics) - 1):
        lyrics[i].end_time = lyrics[i + 1].start_time
    
    # 最後一行結束時間 = 開始時間 + 5 秒
    if lyrics:
        lyrics[-1].end_time = lyrics[-1].start_time + 5
    
    return lyrics


def parse_lrc_file(file_path: str) -> List[LyricLine]:
    """從檔案解析 LRC"""
    with open(file_path, 'r', encoding='utf-8') as f:
        return parse_lrc(f.read())


class AudioProcessor:
    """音訊處理器"""
    
    def __init__(self, ffmpeg_path: str = "ffmpeg"):
        self.ffmpeg_path = ffmpeg_path
        self._check_ffmpeg()
    
    def _check_ffmpeg(self):
        """檢查 ffmpeg 是否可用"""
        try:
            subprocess.run([self.ffmpeg_path, "-version"], 
                         capture_output=True, check=True)
        except (subprocess.CalledProcessError, FileNotFoundError):
            raise RuntimeError("FFmpeg 未安裝或不在 PATH 中")
    
    def cut_segment(self, input_path: str, start: float, end: float, 
                    output_path: str) -> None:
        """切割音訊片段"""
        duration = end - start
        
        cmd = [
            self.ffmpeg_path, "-y",
            "-i", input_path,
            "-ss", f"{start:.3f}",
            "-t", f"{duration:.3f}",
            "-acodec", "libmp3lame",
            "-ar", "44100",
            "-ac", "2",
            "-b:a", "192k",
            output_path
        ]
        
        subprocess.run(cmd, capture_output=True, check=True)
    
    def create_silence(self, output_path: str, duration_sec: float) -> None:
        """建立靜音檔案"""
        cmd = [
            self.ffmpeg_path, "-y",
            "-f", "lavfi",
            "-i", f"anullsrc=r=44100:cl=stereo:d={duration_sec}",
            "-acodec", "libmp3lame",
            "-ar", "44100",
            "-ac", "2",
            "-b:a", "192k",
            output_path
        ]
        
        subprocess.run(cmd, capture_output=True, check=True)
    
    def concat_files(self, file_list: List[str], output_path: str) -> None:
        """合併多個音訊檔案"""
        # 建立臨時檔案列表
        with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', 
                                         delete=False, encoding='utf-8') as f:
            for file_path in file_list:
                # 使用絕對路徑
                abs_path = os.path.abspath(file_path)
                f.write(f"file '{abs_path}'\n")
            list_path = f.name
        
        try:
            cmd = [
                self.ffmpeg_path, "-y",
                "-f", "concat",
                "-safe", "0",
                "-i", list_path,
                "-acodec", "libmp3lame",
                "-ar", "44100",
                "-ac", "2",
                "-b:a", "192k",
                output_path
            ]
            
            subprocess.run(cmd, capture_output=True, check=True)
        finally:
            os.unlink(list_path)
    
    def convert_to_mp3(self, input_path: str, output_path: str) -> None:
        """轉換為 MP3"""
        cmd = [
            self.ffmpeg_path, "-y",
            "-i", input_path,
            "-acodec", "libmp3lame",
            "-ar", "44100",
            "-ac", "2",
            "-b:a", "192k",
            output_path
        ]
        
        subprocess.run(cmd, capture_output=True, check=True)


class GeminiTTS:
    """使用 Gemini API 生成語音"""
    
    LANGUAGE_MAP = {
        "ru-RU": "Russian",
        "en-US": "English",
        "zh-TW": "Traditional Chinese",
        "zh-CN": "Simplified Chinese",
        "ja-JP": "Japanese",
        "ko-KR": "Korean",
        "es-ES": "Spanish",
        "fr-FR": "French",
        "de-DE": "German",
    }
    
    def __init__(self, api_key: str):
        genai.configure(api_key=api_key)
        self.model = genai.GenerativeModel('gemini-2.0-flash-exp')
    
    def generate_speech(self, text: str, lang: str = "en-US") -> Optional[bytes]:
        """生成語音"""
        lang_name = self.LANGUAGE_MAP.get(lang, lang)
        
        prompt = f"""Please generate natural speech audio for the following text in {lang_name}.
Read it clearly at a moderate pace suitable for language learning.

Text: {text}"""
        
        try:
            response = self.model.generate_content(
                prompt,
                generation_config=types.GenerationConfig(
                    response_mime_type="audio/mp3"
                )
            )
            
            # 嘗試獲取音訊數據
            for part in response.candidates[0].content.parts:
                if hasattr(part, 'inline_data') and part.inline_data:
                    return part.inline_data.data
            
            return None
            
        except Exception as e:
            print(f"TTS 生成失敗: {e}")
            return None


def process_learning_audio(
    audio_path: str,
    lrc_path: str,
    output_path: str,
    api_key: str,
    language: str = "ru-RU",
    repeat_count: int = 1,
    max_segments: int = 0
) -> None:
    """處理並生成學習音訊"""
    
    # 1. 解析 LRC
    print("📝 解析 LRC 字幕...")
    lyrics = parse_lrc_file(lrc_path)
    print(f"   找到 {len(lyrics)} 行歌詞")
    
    if max_segments > 0:
        lyrics = lyrics[:max_segments]
        print(f"   限制處理前 {max_segments} 個片段")
    
    # 2. 建立臨時目錄
    temp_dir = Path("output/temp")
    segment_dir = temp_dir / "segments"
    tts_dir = temp_dir / "tts"
    merged_dir = temp_dir / "merged"
    
    for d in [segment_dir, tts_dir, merged_dir]:
        d.mkdir(parents=True, exist_ok=True)
    
    # 3. 初始化處理器
    processor = AudioProcessor()
    tts = GeminiTTS(api_key)
    
    # 4. 轉換輸入音訊為 MP3（如果需要）
    input_mp3 = str(temp_dir / "input.mp3")
    if not audio_path.lower().endswith('.mp3'):
        print("🔄 轉換音訊格式...")
        processor.convert_to_mp3(audio_path, input_mp3)
    else:
        input_mp3 = audio_path
    
    # 5. 處理每個片段
    merged_paths = []
    
    for i, lyric in enumerate(lyrics):
        print(f"\n🎵 處理片段 {i+1}/{len(lyrics)}: {lyric.text[:30]}...")
        
        # 切割原始音訊
        segment_path = str(segment_dir / f"segment_{i:03d}.mp3")
        processor.cut_segment(input_mp3, lyric.start_time, lyric.end_time, segment_path)
        print(f"   ✂️ 已切割: {lyric.start_time:.2f}s - {lyric.end_time:.2f}s")
        
        # 生成 TTS
        tts_path = str(tts_dir / f"tts_{i:03d}.mp3")
        audio_data = tts.generate_speech(lyric.text, language)
        
        if audio_data:
            with open(tts_path, 'wb') as f:
                f.write(audio_data)
            print(f"   🤖 TTS 生成成功")
        else:
            # 使用靜音代替
            processor.create_silence(tts_path, 1.0)
            print(f"   ⚠️ TTS 失敗，使用靜音")
        
        # 合併：原始 + TTS + 原始
        merged_path = str(merged_dir / f"merged_{i:03d}.mp3")
        files_to_concat = []
        
        for _ in range(repeat_count):
            files_to_concat.append(segment_path)
        files_to_concat.append(tts_path)
        for _ in range(repeat_count):
            files_to_concat.append(segment_path)
        
        processor.concat_files(files_to_concat, merged_path)
        merged_paths.append(merged_path)
        print(f"   🔗 已合併")
    
    # 6. 合併所有片段
    print(f"\n📦 生成最終學習音訊...")
    output_dir = Path(output_path).parent
    output_dir.mkdir(parents=True, exist_ok=True)
    
    processor.concat_files(merged_paths, output_path)
    
    print(f"\n✅ 完成！輸出檔案: {output_path}")


def main():
    parser = argparse.ArgumentParser(description='多語言學習器')
    parser.add_argument('-audio', required=True, help='輸入音訊檔案路徑')
    parser.add_argument('-lrc', required=True, help='LRC 字幕檔案路徑')
    parser.add_argument('-output', default='output/learning_audio.mp3', 
                       help='輸出檔案路徑')
    parser.add_argument('-lang', default='ru-RU', help='語言代碼')
    parser.add_argument('-repeat', type=int, default=1, help='原始音訊重複次數')
    parser.add_argument('-max', type=int, default=0, help='最大處理片段數')
    
    args = parser.parse_args()
    
    # 從環境變數或 .env 檔案取得 API Key
    api_key = os.environ.get('GEMINI_API_KEY')
    
    if not api_key:
        # 嘗試從 .env 檔案讀取
        env_path = Path('.env')
        if env_path.exists():
            with open(env_path, 'r') as f:
                for line in f:
                    if line.startswith('GEMINI_API_KEY='):
                        api_key = line.split('=', 1)[1].strip().strip('"\'')
                        break
    
    if not api_key:
        print("❌ 請設定 GEMINI_API_KEY 環境變數")
        sys.exit(1)
    
    process_learning_audio(
        audio_path=args.audio,
        lrc_path=args.lrc,
        output_path=args.output,
        api_key=api_key,
        language=args.lang,
        repeat_count=args.repeat,
        max_segments=args.max
    )


if __name__ == '__main__':
    main()
