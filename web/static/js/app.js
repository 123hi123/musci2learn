// ===== 應用程式狀態 =====
const state = {
    files: [],
    currentFile: null,
    lyrics: null,
    segments: null,
    playlist: [],
    playlistIndex: 0,
    isPlaying: false,
    isLooping: true,
    startLineIndex: 0,
    // 練習模式狀態
    practiceMode: false,
    practiceSettings: {
        ttsRepeat: 2,
        slowMode: false,
        showChinese: false
    },
    practicePlaylist: [],    // 練習模式播放清單
    practiceIndex: 0,        // 目前播放項目索引
    currentSegmentIndex: 0   // 目前段落索引
};

// ===== DOM 元素 =====
const elements = {
    fileList: document.getElementById('fileList'),
    fileInput: document.getElementById('fileInput'),
    uploadBtn: document.getElementById('uploadBtn'),
    emptyState: document.getElementById('emptyState'),
    detailSection: document.getElementById('detailSection'),
    fileName: document.getElementById('fileName'),
    fileStatus: document.getElementById('fileStatus'),
    fileDuration: document.getElementById('fileDuration'),
    fileLyricCount: document.getElementById('fileLyricCount'),
    languageSelect: document.getElementById('languageSelect'),
    showChinese: document.getElementById('showChinese'),
    autoDetectBtn: document.getElementById('autoDetectBtn'),
    lyricsContainer: document.getElementById('lyricsContainer'),
    processBtn: document.getElementById('processBtn'),
    progressSection: document.getElementById('progressSection'),
    progressMessage: document.getElementById('progressMessage'),
    progressPercent: document.getElementById('progressPercent'),
    progressFill: document.getElementById('progressFill'),
    audioPlayer: document.getElementById('audioPlayer'),
    ttsPlayer: document.getElementById('ttsPlayer'),
    // 模式按鈕
    practiceBtn: document.getElementById('practiceBtn'),
    playOriginalBtn: document.getElementById('playOriginalBtn'),
    backToEditBtn: document.getElementById('backToEditBtn'),
    // 練習模式
    editMode: document.getElementById('editMode'),
    practiceMode: document.getElementById('practiceMode'),
    practiceSettings: document.getElementById('practiceSettings'),
    practicePlayer: document.getElementById('practicePlayer'),
    startPracticeBtn: document.getElementById('startPracticeBtn'),
    slowModeGroup: document.getElementById('slowModeGroup'),
    // 練習播放器
    subtitleType: document.getElementById('subtitleType'),
    subtitleMain: document.getElementById('subtitleMain'),
    subtitleSecondary: document.getElementById('subtitleSecondary'),
    subtitleChinese: document.getElementById('subtitleChinese'),
    currentSegment: document.getElementById('currentSegment'),
    totalSegments: document.getElementById('totalSegments'),
    playbackType: document.getElementById('playbackType'),
    practicePrevBtn: document.getElementById('practicePrevBtn'),
    practicePlayBtn: document.getElementById('practicePlayBtn'),
    practiceNextBtn: document.getElementById('practiceNextBtn'),
    practiceLoop: document.getElementById('practiceLoop'),
    practiceShowChinese: document.getElementById('practiceShowChinese'),
    retranslateBtn: document.getElementById('retranslateBtn')
};

// ===== API 請求 =====
const api = {
    async getFiles() {
        const res = await fetch('/api/files');
        const data = await res.json();
        return data.files || [];
    },

    async uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);
        const res = await fetch('/api/files/upload', {
            method: 'POST',
            body: formData
        });
        return await res.json();
    },

    async getFile(id) {
        const res = await fetch(`/api/files/${id}`);
        return await res.json();
    },

    async deleteFile(id) {
        await fetch(`/api/files/${id}`, { method: 'DELETE' });
    },

    async updateSettings(id, settings) {
        await fetch(`/api/files/${id}/settings`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(settings)
        });
    },

    async getLyrics(id) {
        const res = await fetch(`/api/files/${id}/lyrics`);
        return await res.json();
    },

    async detectStart(id) {
        const res = await fetch(`/api/files/${id}/detect-start`, { method: 'POST' });
        return await res.json();
    },

    async startProcess(id) {
        await fetch(`/api/files/${id}/process`, { method: 'POST' });
    },

    async getProgress(id) {
        const res = await fetch(`/api/files/${id}/status`);
        return await res.json();
    },

    async getSegments(id) {
        const res = await fetch(`/api/files/${id}/segments`);
        return await res.json();
    },

    async exportFile(id) {
        await fetch(`/api/files/${id}/export`, { method: 'POST' });
    },

    async retranslateSegment(id, segmentIndex) {
        const res = await fetch(`/api/files/${id}/segments/${segmentIndex}/retranslate`, { 
            method: 'POST' 
        });
        return await res.json();
    }
};

// ===== 工具函數 =====
function formatTime(seconds) {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
}

function getStatusText(status) {
    const statusMap = {
        'uploaded': '已上傳',
        'parsed': '已解析',
        'processing': '處理中',
        'ready': '已就緒',
        'error': '錯誤'
    };
    return statusMap[status] || status;
}

// ===== 渲染函數 =====
function renderFileList() {
    if (state.files.length === 0) {
        elements.fileList.innerHTML = `
            <div class="empty-state">
                <p>尚無檔案</p>
                <p>點擊下方按鈕上傳</p>
            </div>
        `;
        return;
    }

    elements.fileList.innerHTML = state.files.map(file => `
        <div class="file-item ${state.currentFile?.id === file.id ? 'active' : ''}" 
             data-id="${file.id}">
            <span class="file-item-icon">🎵</span>
            <div class="file-item-info">
                <div class="file-item-name">${file.filename}</div>
                <div class="file-item-meta">${formatTime(file.duration)} • ${file.lyricCount || 0} 行</div>
            </div>
            <div class="file-item-status ${file.status}"></div>
        </div>
    `).join('');

    // 綁定點擊事件
    elements.fileList.querySelectorAll('.file-item').forEach(item => {
        item.addEventListener('click', () => selectFile(item.dataset.id));
    });
}

function renderLyrics() {
    if (!state.lyrics || !state.lyrics.lines) {
        elements.lyricsContainer.innerHTML = '<div class="lyrics-loading">無歌詞資料</div>';
        return;
    }

    let html = '';
    state.lyrics.lines.forEach((line, index) => {
        const isSkipped = index < state.startLineIndex;
        const isStartPoint = index === state.startLineIndex;
        
        if (isStartPoint && state.startLineIndex > 0) {
            html += '<div class="start-marker">────────── ▲ 起點線 ▲ ──────────</div>';
        }

        // 獲取翻譯文字
        const zhTranslation = line.translations?.zh || line.translations?.embedded || '';
        const enTranslation = line.translations?.en || '';
        
        html += `
            <div class="lyric-line ${isSkipped ? 'skipped' : ''} ${isStartPoint ? 'start-point' : ''} ${!line.isMeaningful ? 'non-meaningful' : ''}" 
                 data-index="${index}">
                <input type="radio" name="startLine" class="lyric-radio" 
                       ${isStartPoint ? 'checked' : ''}>
                <span class="lyric-timestamp">[${line.timestamp}]</span>
                <div class="lyric-content">
                    <div class="lyric-original">${line.original || '♪'}</div>
                    ${zhTranslation ? 
                        `<div class="lyric-translation lyric-zh">📝 ${zhTranslation}</div>` : ''}
                    ${enTranslation ? 
                        `<div class="lyric-translation lyric-en">🇬🇧 ${enTranslation}</div>` : ''}
                </div>
                ${isSkipped ? '<span class="lyric-badge">忽略</span>' : ''}
                ${!line.isMeaningful ? '<span class="lyric-badge badge-meta">元數據</span>' : ''}
            </div>
        `;
    });

    elements.lyricsContainer.innerHTML = html;

    // 綁定起點選擇事件
    elements.lyricsContainer.querySelectorAll('.lyric-line').forEach(line => {
        line.addEventListener('click', () => {
            const index = parseInt(line.dataset.index);
            setStartLine(index);
        });
    });
}

function renderCurrentLyric(lyricData) {
    if (!lyricData) {
        elements.currentLyric.innerHTML = `
            <div class="lyric-original">--</div>
            <div class="lyric-translation">--</div>
        `;
        return;
    }

    const showChinese = elements.showChinese.checked;
    const primaryLang = elements.languageSelect.value;
    
    let translation = '';
    if (primaryLang === 'en' && lyricData.translations?.en) {
        translation = lyricData.translations.en;
    } else if (lyricData.translations?.embedded) {
        translation = lyricData.translations.embedded;
    }

    let chineseHtml = '';
    if (showChinese && primaryLang !== 'zh' && lyricData.translations?.embedded) {
        chineseHtml = `<div class="lyric-chinese">${lyricData.translations.embedded}</div>`;
    }

    elements.currentLyric.innerHTML = `
        <div class="lyric-original">${lyricData.original || '♪'}</div>
        <div class="lyric-translation">${translation || '--'}</div>
        ${chineseHtml}
    `;
}

function updateProgress(progress) {
    elements.progressSection.style.display = 'block';
    elements.progressMessage.textContent = progress.message;
    elements.progressPercent.textContent = `${Math.round(progress.progress)}%`;
    elements.progressFill.style.width = `${progress.progress}%`;

    if (progress.status === 'done') {
        setTimeout(() => {
            elements.progressSection.style.display = 'none';
            // 啟用練習模式按鈕
            elements.practiceBtn.disabled = false;
            loadFile(state.currentFile.id);
        }, 1000);
    } else if (progress.status === 'error') {
        elements.progressMessage.textContent = `錯誤: ${progress.message}`;
        elements.progressFill.style.backgroundColor = 'var(--error-color)';
    }
}

// ===== 事件處理 =====
async function loadFiles() {
    state.files = await api.getFiles();
    renderFileList();
}

async function selectFile(id) {
    const file = state.files.find(f => f.id === id);
    if (!file) return;

    state.currentFile = file;
    renderFileList();

    // 顯示詳情區域
    elements.emptyState.style.display = 'none';
    elements.detailSection.style.display = 'block';

    // 更新檔案資訊
    elements.fileName.textContent = file.filename;
    elements.fileStatus.textContent = getStatusText(file.status);
    elements.fileStatus.className = `status-badge ${file.status}`;
    elements.fileDuration.textContent = `時長: ${formatTime(file.duration)}`;
    elements.fileLyricCount.textContent = `歌詞: ${file.lyricCount || 0} 行`;

    // 載入設定
    if (file.settings) {
        elements.languageSelect.value = file.settings.primaryLanguage || 'en';
        elements.repeatCount.value = file.settings.ttsRepeatCount || 2;
        elements.showChinese.checked = file.settings.showChineseTranslation !== false;
        state.startLineIndex = file.settings.startLineIndex || 0;
    }

    // 載入歌詞
    await loadLyrics(id);

    // 更新按鈕狀態
    elements.practiceBtn.disabled = file.status !== 'ready';
}

async function loadFile(id) {
    const file = await api.getFile(id);
    const index = state.files.findIndex(f => f.id === id);
    if (index !== -1) {
        state.files[index] = file;
    }
    if (state.currentFile?.id === id) {
        state.currentFile = file;
        elements.fileStatus.textContent = getStatusText(file.status);
        elements.fileStatus.className = `status-badge ${file.status}`;
        elements.practiceBtn.disabled = file.status !== 'ready';
    }
}

async function loadLyrics(id) {
    try {
        state.lyrics = await api.getLyrics(id);
        renderLyrics();
    } catch (e) {
        elements.lyricsContainer.innerHTML = '<div class="lyrics-loading">無法載入歌詞</div>';
    }
}

async function setStartLine(index) {
    state.startLineIndex = index;
    renderLyrics();
    
    if (state.currentFile) {
        await api.updateSettings(state.currentFile.id, {
            startLineIndex: index
        });
    }
}

async function handleUpload(file) {
    const result = await api.uploadFile(file);
    state.files.push(result);
    renderFileList();
    selectFile(result.id);
}

async function handleAutoDetect() {
    if (!state.currentFile) return;
    
    elements.autoDetectBtn.disabled = true;
    elements.autoDetectBtn.textContent = '判斷中...';
    
    try {
        const result = await api.detectStart(state.currentFile.id);
        setStartLine(result.startLineIndex);
    } finally {
        elements.autoDetectBtn.disabled = false;
        elements.autoDetectBtn.textContent = 'AI 自動判斷';
    }
}

async function handleProcess() {
    if (!state.currentFile) return;

    // 先儲存設定
    await api.updateSettings(state.currentFile.id, {
        primaryLanguage: elements.languageSelect.value,
        ttsRepeatCount: 2, // 預設
        showChineseTranslation: elements.showChinese.checked,
        startLineIndex: state.startLineIndex
    });

    // 開始處理
    await api.startProcess(state.currentFile.id);
    
    // 輪詢進度
    const pollProgress = async () => {
        try {
            const progress = await api.getProgress(state.currentFile.id);
            updateProgress(progress);
            
            if (progress.status !== 'done' && progress.status !== 'error') {
                setTimeout(pollProgress, 1000);
            }
        } catch (e) {
            console.error('Error polling progress:', e);
        }
    };
    
    pollProgress();
}

// ===== 練習模式 =====
async function enterPracticeMode() {
    if (!state.currentFile) return;
    
    // 載入段落資料
    try {
        state.segments = await api.getSegments(state.currentFile.id);
        console.log('Loaded segments:', state.segments);
    } catch (e) {
        console.error('Failed to load segments:', e);
        alert('無法載入段落資料');
        return;
    }
    
    // 切換到練習模式
    state.practiceMode = true;
    elements.editMode.style.display = 'none';
    elements.practiceMode.style.display = 'flex';
    elements.backToEditBtn.style.display = 'block';
    
    // 顯示設定面板
    elements.practiceSettings.style.display = 'block';
    elements.practicePlayer.style.display = 'none';
    
    // 更新段落總數
    if (state.segments?.segments) {
        elements.totalSegments.textContent = state.segments.segments.length;
    }
}

function exitPracticeMode() {
    state.practiceMode = false;
    stopPractice();
    
    elements.editMode.style.display = 'flex';
    elements.practiceMode.style.display = 'none';
    elements.backToEditBtn.style.display = 'none';
}

async function startPractice() {
    // 讀取設定
    const ttsRepeatRadio = document.querySelector('input[name="ttsRepeat"]:checked');
    const slowModeRadio = document.querySelector('input[name="slowMode"]:checked');
    
    state.practiceSettings.ttsRepeat = parseInt(ttsRepeatRadio?.value || 2);
    state.practiceSettings.slowMode = slowModeRadio?.value === 'slow';
    state.practiceSettings.showChinese = elements.practiceShowChinese?.checked || false;
    
    // 建立播放清單
    buildPracticePlaylist();
    
    // 切換到播放器
    elements.practiceSettings.style.display = 'none';
    elements.practicePlayer.style.display = 'flex';
    
    // 開始播放
    state.practiceIndex = 0;
    state.currentSegmentIndex = 0;
    updatePracticeDisplay();
    playCurrentPracticeItem();
}

function buildPracticePlaylist() {
    state.practicePlaylist = [];
    
    if (!state.segments?.segments) return;
    
    // 也需要歌詞資料來取得中文翻譯
    const lyricsLines = state.lyrics?.lines || [];
    
    state.segments.segments.forEach((segment, segmentIndex) => {
        // 從 segment 的 lineIndices 取得對應的歌詞行
        const segmentLyrics = (segment.lineIndices || []).map(idx => lyricsLines[idx]).filter(Boolean);
        
        // 取得中文翻譯 (從歌詞資料)
        const textZh = segmentLyrics.map(l => l.translations?.zh || l.translations?.embedded || '').filter(Boolean).join(' ');
        
        // 1. 原曲段落
        state.practicePlaylist.push({
            type: 'original',
            segmentIndex: segmentIndex,
            segment: segment,
            url: `/api/files/${state.currentFile.id}/segments/${segment.index}/audio`,
            label: '🎵 原曲',
            textJa: segment.originalText || '',  // 使用 segments.json 的 originalText
            textEn: segment.ttsText || '',       // 使用 segments.json 的 ttsText (英文翻譯)
            textZh: textZh
        });
        
        // 2. TTS 第一次 (原速)
        state.practicePlaylist.push({
            type: 'tts',
            segmentIndex: segmentIndex,
            segment: segment,
            url: `/api/files/${state.currentFile.id}/segments/${segment.index}/tts`,
            playbackRate: 1.0,
            label: '🗣️ TTS 英文',
            textJa: segment.originalText || '',
            textEn: segment.ttsText || '',
            textZh: textZh
        });
        
        // 3. TTS 第二次 (如果設定為 2 次)
        if (state.practiceSettings.ttsRepeat === 2) {
            state.practicePlaylist.push({
                type: 'tts-slow',
                segmentIndex: segmentIndex,
                segment: segment,
                url: `/api/files/${state.currentFile.id}/segments/${segment.index}/tts`,
                playbackRate: state.practiceSettings.slowMode ? 0.75 : 1.0,
                label: state.practiceSettings.slowMode ? '🗣️ TTS (0.75x)' : '🗣️ TTS (重複)',
                textJa: segment.originalText || '',
                textEn: segment.ttsText || '',
                textZh: textZh
            });
        }
    });
    
    console.log('Practice playlist built:', state.practicePlaylist.length, 'items');
}

function updatePracticeDisplay() {
    const item = state.practicePlaylist[state.practiceIndex];
    if (!item) return;
    
    state.currentSegmentIndex = item.segmentIndex;
    
    // 更新段落指示
    elements.currentSegment.textContent = item.segmentIndex + 1;
    
    // 更新播放類型標籤
    elements.subtitleType.textContent = item.label;
    elements.subtitleType.className = 'subtitle-type';
    if (item.type === 'original') {
        elements.subtitleType.classList.add('type-original');
    } else {
        elements.subtitleType.classList.add('type-tts');
    }
    
    // 更新字幕
    if (item.type === 'original') {
        // 播放原曲時顯示日文
        elements.subtitleMain.textContent = item.textJa || '--';
        elements.subtitleMain.className = 'subtitle-main lang-ja';
        elements.subtitleSecondary.textContent = '';
    } else {
        // 播放 TTS 時顯示英文
        elements.subtitleMain.textContent = item.textEn || '--';
        elements.subtitleMain.className = 'subtitle-main lang-en';
        elements.subtitleSecondary.textContent = '';
    }
    
    // 中文字幕
    if (state.practiceSettings.showChinese && item.textZh) {
        elements.subtitleChinese.textContent = item.textZh;
        elements.subtitleChinese.style.display = 'block';
    } else {
        elements.subtitleChinese.style.display = 'none';
    }
    
    // 播放類型
    elements.playbackType.textContent = item.label;
}

function playCurrentPracticeItem() {
    const item = state.practicePlaylist[state.practiceIndex];
    if (!item) return;
    
    updatePracticeDisplay();
    
    // 選擇播放器
    const player = item.type === 'original' ? elements.audioPlayer : elements.ttsPlayer;
    const otherPlayer = item.type === 'original' ? elements.ttsPlayer : elements.audioPlayer;
    
    // 停止另一個播放器
    otherPlayer.pause();
    
    // 設定來源並播放
    player.src = item.url;
    player.playbackRate = item.playbackRate || 1.0;
    player.play().catch(e => {
        console.error('Playback error:', e);
    });
    
    state.isPlaying = true;
    elements.practicePlayBtn.textContent = '⏸️';
}

function practiceNext() {
    state.practiceIndex++;
    
    if (state.practiceIndex >= state.practicePlaylist.length) {
        // 播放完畢
        if (elements.practiceLoop?.checked) {
            // 循環播放
            state.practiceIndex = 0;
            playCurrentPracticeItem();
        } else {
            // 停止
            state.practiceIndex = state.practicePlaylist.length - 1;
            state.isPlaying = false;
            elements.practicePlayBtn.textContent = '▶️';
        }
    } else {
        playCurrentPracticeItem();
    }
}

function practicePrev() {
    // 找到當前段落的起始位置
    const currentSegment = state.practicePlaylist[state.practiceIndex]?.segmentIndex || 0;
    
    // 往前找上一個段落
    let targetIndex = 0;
    for (let i = state.practiceIndex - 1; i >= 0; i--) {
        if (state.practicePlaylist[i].segmentIndex < currentSegment) {
            targetIndex = i;
            // 找到該段落的第一個項目
            while (targetIndex > 0 && state.practicePlaylist[targetIndex - 1].segmentIndex === state.practicePlaylist[targetIndex].segmentIndex) {
                targetIndex--;
            }
            break;
        }
    }
    
    state.practiceIndex = targetIndex;
    playCurrentPracticeItem();
}

function togglePracticePlay() {
    if (state.isPlaying) {
        elements.audioPlayer.pause();
        elements.ttsPlayer.pause();
        state.isPlaying = false;
        elements.practicePlayBtn.textContent = '▶️';
    } else {
        playCurrentPracticeItem();
    }
}

function stopPractice() {
    elements.audioPlayer.pause();
    elements.ttsPlayer.pause();
    state.isPlaying = false;
    state.practiceIndex = 0;
}

// 重新翻譯當前段落
async function handleRetranslate() {
    const item = state.practicePlaylist[state.practiceIndex];
    if (!item || !state.currentFile) return;

    const segmentIndex = item.segmentIndex;
    const btn = elements.retranslateBtn;
    
    // 禁用按鈕並顯示載入狀態
    btn.disabled = true;
    btn.classList.add('loading');
    btn.textContent = '⏳';
    
    try {
        const result = await api.retranslateSegment(state.currentFile.id, segmentIndex);
        
        if (result.translation) {
            // 更新播放列表中所有同一段落的項目
            state.practicePlaylist.forEach(playlistItem => {
                if (playlistItem.segmentIndex === segmentIndex) {
                    playlistItem.textEn = result.translation;
                }
            });
            
            // 更新 segments 資料
            if (state.segments && state.segments[segmentIndex]) {
                state.segments[segmentIndex].ttsText = result.translation;
            }
            
            // 更新當前顯示
            updatePracticeDisplay();
            
            // 顯示成功
            btn.textContent = '✅';
            setTimeout(() => {
                btn.textContent = '💡';
            }, 1500);
        } else if (result.error) {
            alert('重新翻譯失敗: ' + result.error);
            btn.textContent = '❌';
            setTimeout(() => {
                btn.textContent = '💡';
            }, 1500);
        }
    } catch (error) {
        console.error('Retranslate error:', error);
        alert('重新翻譯失敗');
        btn.textContent = '❌';
        setTimeout(() => {
            btn.textContent = '💡';
        }, 1500);
    } finally {
        btn.disabled = false;
        btn.classList.remove('loading');
    }
}

// 原曲播放器結束事件
elements.audioPlayer.addEventListener('ended', () => {
    if (state.practiceMode) {
        // 練習模式：自動播放下一項
        practiceNext();
    }
});

// TTS 播放器結束事件
elements.ttsPlayer.addEventListener('ended', () => {
    if (state.practiceMode) {
        // 練習模式：自動播放下一項
        practiceNext();
    }
});

// ===== 原始播放模式 =====
function playOriginal() {
    if (!state.currentFile) return;
    
    elements.audioPlayer.src = `/api/files/${state.currentFile.id}/audio`;
    elements.audioPlayer.play();
}

// ===== 事件綁定 =====
elements.uploadBtn.addEventListener('click', () => elements.fileInput.click());
elements.fileInput.addEventListener('change', (e) => {
    if (e.target.files[0]) {
        handleUpload(e.target.files[0]);
    }
});
elements.autoDetectBtn.addEventListener('click', handleAutoDetect);
elements.processBtn.addEventListener('click', handleProcess);

// 練習模式按鈕
elements.practiceBtn?.addEventListener('click', enterPracticeMode);
elements.playOriginalBtn?.addEventListener('click', playOriginal);
elements.backToEditBtn?.addEventListener('click', exitPracticeMode);
elements.startPracticeBtn?.addEventListener('click', startPractice);
elements.practicePlayBtn?.addEventListener('click', togglePracticePlay);
elements.practicePrevBtn?.addEventListener('click', practicePrev);
elements.practiceNextBtn?.addEventListener('click', practiceNext);
elements.retranslateBtn?.addEventListener('click', handleRetranslate);

// TTS 重複次數變更時，控制慢速選項顯示
document.querySelectorAll('input[name="ttsRepeat"]').forEach(radio => {
    radio.addEventListener('change', (e) => {
        const showSlowMode = e.target.value === '2';
        if (elements.slowModeGroup) {
            elements.slowModeGroup.style.display = showSlowMode ? 'block' : 'none';
        }
    });
});

// 中文字幕切換
elements.practiceShowChinese?.addEventListener('change', () => {
    state.practiceSettings.showChinese = elements.practiceShowChinese.checked;
    updatePracticeDisplay();
});

elements.languageSelect.addEventListener('change', () => {
    if (state.currentFile) {
        api.updateSettings(state.currentFile.id, {
            primaryLanguage: elements.languageSelect.value
        });
    }
});

elements.showChinese?.addEventListener('change', () => {
    if (state.currentFile) {
        api.updateSettings(state.currentFile.id, {
            showChineseTranslation: elements.showChinese.checked
        });
    }
});

// ===== 初始化 =====
document.addEventListener('DOMContentLoaded', () => {
    loadFiles();
});
