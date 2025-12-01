#!/usr/bin/env python3
"""
Embed LRC lyrics into MP3 file using multiple ID3v2 tags for maximum compatibility.
Supports:
- USLT: Standard unsynchronized lyrics (iTunes, WMP, Poweramp, etc.)
- TXXX:LYRICS: Custom tag used by Foobar2000 Lyrics Show Panel, MusicBee, etc.
"""

import sys
import os
from mutagen.mp3 import MP3
from mutagen.id3 import ID3, USLT, TXXX, ID3NoHeaderError

def embed_lyrics(mp3_path: str, lrc_path: str, output_path: str = None):
    """
    Embed LRC lyrics into MP3 file using multiple tag formats for compatibility.
    
    Args:
        mp3_path: Path to the MP3 file
        lrc_path: Path to the LRC file
        output_path: Optional output path (default: overwrite input)
    """
    # Read LRC content
    with open(lrc_path, 'r', encoding='utf-8') as f:
        lrc_content = f.read()
    
    # If output path specified, copy file first
    if output_path and output_path != mp3_path:
        import shutil
        shutil.copy2(mp3_path, output_path)
        mp3_path = output_path
    
    # Open MP3 file
    try:
        audio = MP3(mp3_path, ID3=ID3)
    except ID3NoHeaderError:
        # No ID3 tag, create one
        audio = MP3(mp3_path)
        audio.add_tags()
    
    # Remove existing lyrics tags
    audio.tags.delall('USLT')
    # Remove existing TXXX:LYRICS if any
    keys_to_remove = [k for k in audio.tags.keys() if k.startswith('TXXX:') and 'LYRICS' in k.upper()]
    for k in keys_to_remove:
        del audio.tags[k]
    
    # Add USLT tag (standard unsynchronized lyrics)
    # encoding=3 means UTF-8
    # lang='XXX' is for undefined/multiple languages
    audio.tags.add(USLT(
        encoding=3,  # UTF-8
        lang='XXX',  # Undefined language
        desc='',     # Description
        text=lrc_content
    ))
    print("✅ Added USLT tag (for iTunes, WMP, Poweramp, etc.)")
    
    # Add TXXX:LYRICS tag (for Foobar2000, MusicBee, etc.)
    audio.tags.add(TXXX(
        encoding=3,  # UTF-8
        desc='LYRICS',
        text=lrc_content
    ))
    print("✅ Added TXXX:LYRICS tag (for Foobar2000, MusicBee, etc.)")
    
    # Save
    audio.save()
    print(f"\n✅ Successfully embedded lyrics into: {mp3_path}")
    print(f"   LRC size: {len(lrc_content)} characters")

def read_lyrics(mp3_path: str):
    """Read and display embedded lyrics from MP3 file."""
    try:
        audio = MP3(mp3_path, ID3=ID3)
    except ID3NoHeaderError:
        print("No ID3 tag found")
        return None
    
    # Find USLT tags
    for key in audio.tags.keys():
        if key.startswith('USLT'):
            uslt = audio.tags[key]
            print(f"Found USLT tag:")
            print(f"  Language: {uslt.lang}")
            print(f"  Description: {uslt.desc}")
            print(f"  Text length: {len(uslt.text)} characters")
            print(f"\nFirst 500 characters:")
            print(uslt.text[:500])
            return uslt.text
    
    print("No USLT lyrics found")
    return None

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage:")
        print("  Embed:  python embed_lyrics.py <mp3_file> <lrc_file> [output_file]")
        print("  Read:   python embed_lyrics.py --read <mp3_file>")
        sys.exit(1)
    
    if sys.argv[1] == '--read':
        if len(sys.argv) < 3:
            print("Please specify MP3 file")
            sys.exit(1)
        read_lyrics(sys.argv[2])
    else:
        mp3_path = sys.argv[1]
        lrc_path = sys.argv[2] if len(sys.argv) > 2 else None
        output_path = sys.argv[3] if len(sys.argv) > 3 else None
        
        if not lrc_path:
            # Try to find LRC file with same name
            lrc_path = os.path.splitext(mp3_path)[0] + '.lrc'
        
        if not os.path.exists(lrc_path):
            print(f"LRC file not found: {lrc_path}")
            sys.exit(1)
        
        embed_lyrics(mp3_path, lrc_path, output_path)
