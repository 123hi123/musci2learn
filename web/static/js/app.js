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
        showChinese: false,
        shuffleMode: 'off'  // 'off' | 'playlist' | 'super'
    },
    practicePlaylist: [],    // 練習模式播放清單
    practiceIndex: 0,        // 目前播放項目索引
    currentSegmentIndex: 0,  // 目前段落索引
    _practiceReturnFileId: null, // 進入練習時的首頁選歌（離開練習要還原）
    // 歌單隨機模式的歌曲佇列
    shuffleQueue: [],        // 打亂後的歌曲 ID 列表
    shuffleQueueIndex: 0     // 目前在佇列中的位置
};

// ===== 預設設定（儲存在 localStorage）=====
const defaultSettings = {
    loop: true,
    shuffleMode: 'playlist',  // 'off' | 'playlist' | 'super'
    ttsRepeat: 1,
    ttsVolumeMultiplier: 10,  // 10 = 1.0x, 40 = 4.0x (實際倍數 = 值/10)
    showChinese: false
};

// 從 localStorage 載入設定
function loadSettings() {
    const saved = localStorage.getItem('practiceSettings');
    if (saved) {
        try {
            const parsed = JSON.parse(saved);
            Object.assign(defaultSettings, parsed);
        } catch (e) {
            console.error('Failed to load settings:', e);
        }
    }
    return defaultSettings;
}

// 儲存設定到 localStorage
function saveSettings() {
    localStorage.setItem('practiceSettings', JSON.stringify(defaultSettings));
}

// 格式化音量倍數顯示
function formatVolumeMultiplier(value) {
    return (value / 10).toFixed(1) + 'x';
}

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
    practiceShuffleMode: document.getElementById('practiceShuffleMode'),
    retranslateBtn: document.getElementById('retranslateBtn'),
    // 隨機模式資訊
    shuffleSongInfo: document.getElementById('shuffleSongInfo'),
    shuffleSongName: document.getElementById('shuffleSongName'),
    // 音量控制
    volumeControl: document.getElementById('volumeControl'),
    ttsVolume: document.getElementById('ttsVolume'),
    volumeValue: document.getElementById('volumeValue'),
    // 設定 Modal
    settingsBtn: document.getElementById('settingsBtn'),
    settingsModal: document.getElementById('settingsModal'),
    closeSettingsBtn: document.getElementById('closeSettingsBtn'),
    saveSettingsBtn: document.getElementById('saveSettingsBtn'),
    defaultLoop: document.getElementById('defaultLoop'),
    defaultShuffleMode: document.getElementById('defaultShuffleMode'),
    defaultTtsRepeat: document.getElementById('defaultTtsRepeat'),
    defaultVolume: document.getElementById('defaultVolume'),
    defaultVolumeValue: document.getElementById('defaultVolumeValue'),
    defaultShowChinese: document.getElementById('defaultShowChinese')
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

    async retranslateSegment(id, segmentIndex, userInput) {
        const res = await fetch(`/api/files/${id}/segments/${segmentIndex}/retranslate`, { 
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ userInput: userInput })
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
            <button class="btn-file-menu" data-id="${file.id}" title="更多選項">⋮</button>
        </div>
    `).join('');

    // 綁定點擊事件
    elements.fileList.querySelectorAll('.file-item').forEach(item => {
        item.addEventListener('click', (e) => {
            // 如果點擊的是選單按鈕，不要選擇檔案
            if (e.target.classList.contains('btn-file-menu')) return;
            selectFile(item.dataset.id);
        });
    });

    // 綁定選單按鈕事件
    elements.fileList.querySelectorAll('.btn-file-menu').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            showFileMenu(e, btn.dataset.id);
        });
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

    // 檢查是否已經處理過，如果是則顯示確認對話框
    if (state.currentFile.status === 'ready') {
        const confirmed = confirm(
            '⚠️ 此檔案已經處理過了！\n\n' +
            '重新處理將會：\n' +
            '• 覆蓋現有的翻譯內容\n' +
            '• 重新生成所有 TTS 語音\n' +
            '• 消耗 API 額度\n\n' +
            '確定要重新處理嗎？'
        );
        if (!confirmed) {
            return;
        }
    }

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
// 注意：如果在 click handler 中先 await（例如等 API 回來），再呼叫 audio.play()，
// 在不少瀏覽器會被視為「非使用者手勢」而被 autoplay policy 擋下，
// 就會出現「進入練習後不會馬上播放，要按下一句才開始」的狀況。
//
// 這裡做一個 best-effort 的「播放解鎖」：在第一次 await 之前，對目前已有 src 的播放器
// 做一次靜音 play -> pause，讓後續的播放更不容易被阻擋。
function unlockMediaPlayback() {
    const players = [elements.audioPlayer, elements.ttsPlayer].filter(Boolean);

    for (const player of players) {
        try {
            if (!player.src) continue;

            const prevMuted = player.muted;
            const prevVolume = player.volume;

            player.muted = true;
            player.volume = 0;

            const pr = player.play();
            if (pr && typeof pr.then === 'function') {
                pr.then(() => {
                    player.pause();
                }).catch(() => {
                    // ignore: best-effort unlock
                }).finally(() => {
                    player.muted = prevMuted;
                    player.volume = prevVolume;
                });
            } else {
                player.pause();
                player.muted = prevMuted;
                player.volume = prevVolume;
            }
        } catch (e) {
            // ignore
        }
    }
}

function setPracticePausedState() {
    // 停止任何正在播放的音訊（避免「回到主頁」後殘留上一首的聲音/狀態）
    elements.audioPlayer?.pause();
    elements.ttsPlayer?.pause();

    state.isPlaying = false;
    if (elements.practicePlayBtn) elements.practicePlayBtn.textContent = '\u25b6\ufe0f';
}

function getReadyFileIds() {
    return (state.files || []).filter(f => f.status === 'ready').map(f => f.id);
}

function buildPracticeQueue(seedFileId) {
    const readyIds = getReadyFileIds();
    if (readyIds.length === 0) {
        state.shuffleQueue = [];
        state.shuffleQueueIndex = 0;
        return false;
    }

    // 規格：進入練習模式的「第一首」必須是使用者點進來的那首（只要它是 ready）
    const seedOk = seedFileId && readyIds.includes(seedFileId);
    const rest = seedOk ? readyIds.filter(id => id !== seedFileId) : readyIds.slice();
    shuffleArray(rest);

    state.shuffleQueue = seedOk ? [seedFileId, ...rest] : rest;
    state.shuffleQueueIndex = 0;
    return true;
}

function reshufflePracticeQueueAvoidRepeat(currentFileId) {
    const readyIds = getReadyFileIds();
    if (readyIds.length === 0) {
        state.shuffleQueue = [];
        state.shuffleQueueIndex = 0;
        return false;
    }

    // 只有一首歌時沒辦法避免連播
    if (readyIds.length === 1) {
        state.shuffleQueue = readyIds.slice();
        state.shuffleQueueIndex = 0;
        return true;
    }

    const rest = readyIds.filter(id => id !== currentFileId);
    shuffleArray(rest);
    // 先放一首「不是目前這首」的，後面再把剩下的（含目前這首）洗牌接上
    const first = rest[0];
    const remaining = readyIds.filter(id => id !== first);
    shuffleArray(remaining);

    state.shuffleQueue = [first, ...remaining];
    state.shuffleQueueIndex = 0;
    return true;
}

function updatePracticeQueueSongInfo() {
    if (!elements.shuffleSongInfo || !elements.shuffleSongName) return;

    if (!state.practiceMode || !state.currentFile) {
        elements.shuffleSongInfo.style.display = 'none';
        return;
    }

    if (state.shuffleQueue && state.shuffleQueue.length > 0) {
        elements.shuffleSongInfo.style.display = 'flex';
        elements.shuffleSongName.textContent = `${state.currentFile.filename} (${state.shuffleQueueIndex + 1}/${state.shuffleQueue.length})`;
    } else {
        elements.shuffleSongInfo.style.display = 'none';
    }
}

async function loadPracticeSong(fileId, autoplay) {
    const file = state.files.find(f => f.id === fileId);
    if (!file) {
        console.error('Practice song not found:', fileId);
        return false;
    }

    // 載入段落和歌詞（切歌時必須載入該歌的資料）
    const [segmentsData, lyricsData] = await Promise.all([
        api.getSegments(fileId),
        api.getLyrics(fileId)
    ]);

    if (!segmentsData?.segments || segmentsData.segments.length === 0) {
        console.error('No segments for practice song:', file.filename);
        return false;
    }

    // 練習模式用的「目前歌曲」：直接切換 currentFile（離開練習時會還原）
    state.currentFile = file;
    state.segments = segmentsData;
    state.lyrics = lyricsData;

    // 段落永遠回到第一段（規格）
    buildPracticePlaylist();
    state.practiceIndex = 0;
    state.currentSegmentIndex = 0;

    // 更新顯示
    elements.currentSegment.textContent = 1;
    elements.totalSegments.textContent = segmentsData.segments.length;
    updatePracticeDisplay();
    updatePracticeQueueSongInfo();

    setPracticePausedState();
    if (autoplay) {
        playCurrentPracticeItem();
    }

    return true;
}

async function playNextPracticeQueueSong(autoplay) {
    if (!state.shuffleQueue || state.shuffleQueue.length === 0) {
        console.warn('Practice queue empty');
        return;
    }

    const currentId = state.currentFile?.id || null;
    let nextIndex = state.shuffleQueueIndex + 1;

    if (nextIndex >= state.shuffleQueue.length) {
        // 播完隊列：循環就重新洗牌，但避免下一首跟目前這首一樣
        if (elements.practiceLoop?.checked) {
            reshufflePracticeQueueAvoidRepeat(currentId);
            nextIndex = 0;
        } else {
            // 不循環就停在最後一首
            state.isPlaying = false;
            elements.practicePlayBtn.textContent = '\u25b6\ufe0f';
            return;
        }
    }

    state.shuffleQueueIndex = nextIndex;
    const nextId = state.shuffleQueue[state.shuffleQueueIndex];
    await loadPracticeSong(nextId, autoplay);
}

async function enterPracticeMode() {
    if (!state.currentFile) return;

    // 重要：必須在第一次 await 之前執行，才算「使用者手勢」延伸
    unlockMediaPlayback();

    // 進入練習模式時，建立新的隊列（規格：每次進入都是全新 session）
    const seedFileId = state.currentFile.id;
    state._practiceReturnFileId = seedFileId;
    
    // 載入段落資料
    try {
        const [segmentsData, lyricsData] = await Promise.all([
            api.getSegments(seedFileId),
            api.getLyrics(seedFileId)
        ]);
        state.segments = segmentsData;
        state.lyrics = lyricsData;
        console.log('Loaded segments:', state.segments);
    } catch (e) {
        console.error('Failed to load segments:', e);
        alert('無法載入段落資料');
        return;
    }
    
    // 確保不是隨機模式
    state.shuffleMode = false;
    
    // 切換到練習模式
    state.practiceMode = true;
    elements.editMode.style.display = 'none';
    elements.practiceMode.style.display = 'flex';
    elements.backToEditBtn.style.display = 'block';
    
    // 隱藏設定面板，直接顯示播放器
    elements.practiceSettings.style.display = 'none';
    elements.practicePlayer.style.display = 'flex';
    
    // 更新段落總數
    if (state.segments?.segments) {
        elements.totalSegments.textContent = state.segments.segments.length;
    }
    
    // 使用預設設定初始化（不自動播放：按播放鍵才開始）
    loadSettings();
    state.practiceSettings.ttsRepeat = defaultSettings.ttsRepeat;
    state.practiceSettings.slowMode = false;
    state.practiceSettings.showChinese = defaultSettings.showChinese;
    state.practiceSettings.shuffleMode = defaultSettings.shuffleMode;
    
    // 套用預設設定到 UI
    if (elements.practiceLoop) elements.practiceLoop.checked = defaultSettings.loop;
    if (elements.practiceShuffleMode) elements.practiceShuffleMode.value = defaultSettings.shuffleMode;
    if (elements.practiceShowChinese) elements.practiceShowChinese.checked = defaultSettings.showChinese;
    // TTS 音量倍數（slider 存的是 10=1.0x, 40=4.0x）
    if (elements.ttsVolume) elements.ttsVolume.value = defaultSettings.ttsVolumeMultiplier;
    if (elements.volumeValue) elements.volumeValue.textContent = formatVolumeMultiplier(defaultSettings.ttsVolumeMultiplier);
    if (elements.ttsPlayer) elements.ttsPlayer.volume = Math.min(defaultSettings.ttsVolumeMultiplier / 10, 1.0);
    
    // 隱藏隨機模式歌曲資訊（稍後會根據模式顯示）
    if (elements.shuffleSongInfo) {
        elements.shuffleSongInfo.style.display = 'none';
    }
    
    // 根據模式決定播放方式（本專案的「歌單」概念主要在 playlist 模式）
    const shuffleMode = state.practiceSettings.shuffleMode;
    
    if (shuffleMode === 'playlist') {
        // 規格：以「使用者點進來的歌」作為隊列第一首；段落從第一段開始；不自動播放
        buildPracticeQueue(seedFileId);
        updatePracticeQueueSongInfo();

        buildPracticePlaylist();
        state.practiceIndex = 0;
        state.currentSegmentIndex = 0;

        // 更新段落計數 & 顯示
        elements.currentSegment.textContent = 1;
        updatePracticeDisplay();
    } else if (shuffleMode === 'super') {
        // 超級隨機：保留原概念，但進入時仍先顯示目前這首（段落第一段）
        buildPracticePlaylist();
        state.practiceIndex = 0;
        state.currentSegmentIndex = 0;
        elements.currentSegment.textContent = 1;
        updatePracticeDisplay();
    } else {
        // 一般練習模式：只播放當前歌曲（不自動播放）
        buildPracticePlaylist();
        state.practiceIndex = 0;
        state.currentSegmentIndex = 0;
        elements.currentSegment.textContent = 1;
        updatePracticeDisplay();
        if (elements.shuffleSongInfo) elements.shuffleSongInfo.style.display = 'none';
    }

    setPracticePausedState();
}

async function exitPracticeMode() {
    state.practiceMode = false;
    state.shuffleMode = false; // 重置隨機模式（舊狀態）
    state.shuffleQueue = [];
    state.shuffleQueueIndex = 0;

    const returnId = state._practiceReturnFileId;
    state._practiceReturnFileId = null;

    stopPractice();

    elements.editMode.style.display = 'flex';
    elements.practiceMode.style.display = 'none';
    elements.backToEditBtn.style.display = 'none';

    // 隱藏隨機模式歌曲資訊
    if (elements.shuffleSongInfo) {
        elements.shuffleSongInfo.style.display = 'none';
    }

    // 練習模式會切歌並更新 currentFile；離開後把首頁選歌還原回「進入練習時那首」
    if (returnId) {
        try {
            await selectFile(returnId);
        } catch (e) {
            console.error('Failed to restore selected file after practice:', e);
        }
    }
}

async function startPractice() {
    // 讀取設定
    const ttsRepeatRadio = document.querySelector('input[name="ttsRepeat"]:checked');
    const slowModeRadio = document.querySelector('input[name="slowMode"]:checked');
    
    state.practiceSettings.ttsRepeat = parseInt(ttsRepeatRadio?.value || 2);
    state.practiceSettings.slowMode = slowModeRadio?.value === 'slow';
    state.practiceSettings.showChinese = elements.practiceShowChinese?.checked || false;
    state.practiceSettings.shuffleMode = elements.practiceShuffleMode?.value || 'off';
    
    // 切換到播放器
    elements.practiceSettings.style.display = 'none';
    elements.practicePlayer.style.display = 'flex';
    
    // 根據隨機模式決定播放方式
    const shuffleMode = state.practiceSettings.shuffleMode;
    
    if (shuffleMode === 'playlist') {
        // 歌單隊列：第一首是目前選到的歌；不自動播放（按播放鍵才開始）
        const seedFileId = state.currentFile?.id || null;
        if (!seedFileId) return;

        if (!buildPracticeQueue(seedFileId)) {
            alert('沒有已處理完成的音檔！');
            return;
        }

        updatePracticeQueueSongInfo();
        buildPracticePlaylist();
        state.practiceIndex = 0;
        state.currentSegmentIndex = 0;
        elements.currentSegment.textContent = 1;
        updatePracticeDisplay();
        setPracticePausedState();
    } else if (shuffleMode === 'super') {
        // 超級隨機模式：隨機跳到一個段落
        await shuffleToRandomSegment();
    } else if (state.shuffleMode) {
        // 舊的隨機模式（保留向後相容）
        await startShufflePlayback();
    } else {
        // 一般練習模式
        buildPracticePlaylist();
        state.practiceIndex = 0;
        state.currentSegmentIndex = 0;
        updatePracticeDisplay();
        setPracticePausedState();
    }
}

function buildPracticePlaylist(singleSegmentIndex = null) {
    state.practicePlaylist = [];
    
    if (!state.segments?.segments) return;
    
    // 也需要歌詞資料來取得中文翻譯
    const lyricsLines = state.lyrics?.lines || [];
    
    // 決定要處理哪些段落
    const segmentsToProcess = singleSegmentIndex !== null 
        ? [{ segment: state.segments.segments[singleSegmentIndex], index: singleSegmentIndex }]
        : state.segments.segments.map((seg, idx) => ({ segment: seg, index: idx }));
    
    segmentsToProcess.forEach(({ segment, index: segmentIndex }) => {
        if (!segment) return;
        
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
    
    console.log('Practice playlist built:', state.practicePlaylist.length, 'items', 
        singleSegmentIndex !== null ? `(single segment ${singleSegmentIndex})` : '(all segments)');

    // 如果是為單一段落建立播放清單，將歌詞起始行設為該段落的第一個 lineIndex，
    // 以便 renderLyrics() 能正確顯示哪一句是起點（解決隨機模式下段落/歌詞不同步問題）。
    if (singleSegmentIndex !== null) {
        const seg = state.segments.segments[singleSegmentIndex];
        const firstLine = seg?.lineIndices?.[0] ?? 0;
        state.startLineIndex = firstLine;
        // 立即更新歌詞顯示
        try { renderLyrics(); } catch (e) { /* ignore if render not available yet */ }
    }
}

function updatePracticeDisplay() {
    const item = state.practicePlaylist[state.practiceIndex];
    if (!item) return;
    
    state.currentSegmentIndex = item.segmentIndex;
    
    // 若該播放項目包含 segment 物件，將歌詞起始行設為該段落的第一個 lineIndex，確保歌詞顯示與目前段落同步
    if (item.segment && item.segment.lineIndices && item.segment.lineIndices.length > 0) {
        const firstLine = item.segment.lineIndices[0];
        if (state.startLineIndex !== firstLine) {
            state.startLineIndex = firstLine;
            try { renderLyrics(); } catch (e) { /* ignore */ }
        }
    }

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

    // 先把狀態視為「未播放」，等真的播放成功再切換成播放中
    state.isPlaying = false;
    elements.practicePlayBtn.textContent = '\u25b6\ufe0f';

    const playPromise = player.play();
    if (playPromise && typeof playPromise.then === 'function') {
        playPromise.then(() => {
            state.isPlaying = true;
            elements.practicePlayBtn.textContent = '\u23f8\ufe0f';
        }).catch(e => {
            // 常見：NotAllowedError（autoplay policy）
            console.error('Playback error:', e);
            state.isPlaying = false;
            elements.practicePlayBtn.textContent = '\u25b6\ufe0f';
        });
    } else {
        // 舊瀏覽器：假設會播放
        state.isPlaying = true;
        elements.practicePlayBtn.textContent = '\u23f8\ufe0f';
    }
}

function practiceNext() {
    state.practiceIndex++;
    
    if (state.practiceIndex >= state.practicePlaylist.length) {
        // 當前歌曲/段落播放完畢
        const shuffleMode = elements.practiceShuffleMode?.value || 'off';
        
        if (shuffleMode === 'super') {
            // 超級隨機模式：跳到隨機一首歌的隨機段落
            shuffleToRandomSegment();
        } else if (shuffleMode === 'playlist') {
            // 歌單隊列：播完整首歌後跳到下一首（延續自動播放）
            playNextPracticeQueueSong(true);
        } else if (elements.practiceLoop?.checked) {
            // 循環播放當前歌曲
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
    state.practicePlaylist = [];
    state.practiceIndex = 0;
    state.currentSegmentIndex = 0;
}

// 重新翻譯當前段落（用戶輸入原句）
async function handleRetranslate() {
    const item = state.practicePlaylist[state.practiceIndex];
    if (!item || !state.currentFile) return;

    const segmentIndex = item.segmentIndex;
    const btn = elements.retranslateBtn;
    
    // 取得當前顯示的原文作為預設值
    const currentJaText = item.textJa || '';
    
    // 彈出輸入框讓用戶輸入原句
    const userInput = prompt(
        '請輸入這句話的正確原文（任何語言皆可）：\n\n系統會將其翻譯成英文並重新生成語音。',
        currentJaText
    );
    
    // 如果用戶取消或輸入空白則不處理
    if (!userInput || userInput.trim() === '') {
        return;
    }
    
    // 禁用按鈕並顯示載入狀態
    btn.disabled = true;
    btn.classList.add('loading');
    btn.textContent = '⏳';
    
    try {
        const result = await api.retranslateSegment(state.currentFile.id, segmentIndex, userInput.trim());
        
        if (result.translation) {
            // 更新播放列表中所有同一段落的項目
            state.practicePlaylist.forEach(playlistItem => {
                if (playlistItem.segmentIndex === segmentIndex) {
                    playlistItem.textEn = result.translation;
                }
            });
            
            // 更新 segments 資料
            if (state.segments?.segments && state.segments.segments[segmentIndex]) {
                state.segments.segments[segmentIndex].ttsText = result.translation;
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
// ===== 隨機練習模式 (跨歌曲) =====

// Fisher-Yates 洗牌演算法
function shuffleArray(array) {
    for (let i = array.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [array[i], array[j]] = [array[j], array[i]];
    }
    return array;
}

// ===== 隨機模式：歌單隨機 =====
// 播完整首歌後跳到下一首隨機歌曲

// 初始化歌單隨機佇列
function initShuffleQueue(startFileId = null) {
    const readyFiles = state.files.filter(f => f.status === 'ready');
    if (readyFiles.length === 0) return false;

    const allIds = readyFiles.map(f => f.id);

    // 需求：從任何歌曲進入練習模式時，歌單必須以「目前選到的那首」作為起點。
    // 其他歌曲再隨機排列，避免一進練習就跳回上一首或跳到別首造成混亂。
    if (startFileId && allIds.includes(startFileId)) {
        const rest = allIds.filter(id => id !== startFileId);
        shuffleArray(rest);
        state.shuffleQueue = [startFileId, ...rest];
    } else {
        state.shuffleQueue = allIds;
        shuffleArray(state.shuffleQueue);
    }

    state.shuffleQueueIndex = 0;

    console.log('Shuffle queue initialized:', state.shuffleQueue.length, 'songs', startFileId ? `(start: ${startFileId})` : '');
    return true;
}

// 播放歌單中的下一首歌
async function playNextShuffledSong() {
    // 舊函數：保留相容性，改用新的「隊列」邏輯
    await playNextPracticeQueueSong(true);
}

// 載入並播放指定歌曲（從頭開始播放所有段落）
async function loadAndPlaySong(fileId) {
    const file = state.files.find(f => f.id === fileId);
    if (!file) {
        console.error('File not found:', fileId);
        playNextShuffledSong(); // 跳過這首
        return;
    }
    
    console.log('Loading song:', file.filename);
    
    try {
        // 載入段落和歌詞
        const segmentsData = await api.getSegments(fileId);
        const lyricsData = await api.getLyrics(fileId);
        
        if (!segmentsData.segments || segmentsData.segments.length === 0) {
            console.error('No segments in file:', file.filename);
            playNextShuffledSong();
            return;
        }
        
        // 更新狀態
        state.currentFile = file;
        state.segments = segmentsData;
        state.lyrics = lyricsData;
        
        // 更新歌曲資訊顯示
        if (elements.shuffleSongInfo) {
            elements.shuffleSongInfo.style.display = 'flex';
        }
        if (elements.shuffleSongName) {
            elements.shuffleSongName.textContent = `${file.filename} (${state.shuffleQueueIndex + 1}/${state.shuffleQueue.length})`;
        }
        
        // 建立整首歌的播放列表（所有段落）
        buildPracticePlaylist(); // 不傳參數 = 所有段落
        state.practiceIndex = 0;
        
        // 更新段落計數
        elements.currentSegment.textContent = 1;
        elements.totalSegments.textContent = segmentsData.segments.length;
        
        console.log(`Playing all ${segmentsData.segments.length} segments from ${file.filename}`);
        
        playCurrentPracticeItem();
        
    } catch (error) {
        console.error('Error loading song:', error);
        playNextShuffledSong();
    }
}

// ===== 隨機模式：超級隨機 =====
// 每個段落都隨機跳到任意歌曲的任意段落

async function shuffleToRandomSegment() {
    console.log('shuffleToRandomSegment called');
    
    // 取得所有已處理完成的檔案
    const readyFiles = state.files.filter(f => f.status === 'ready');
    console.log('Ready files:', readyFiles.length);
    
    if (readyFiles.length === 0) {
        alert('沒有已處理完成的音檔！');
        if (elements.practiceShuffleMode) elements.practiceShuffleMode.value = 'off';
        return;
    }
    
    // 隨機選擇一首歌（盡量不重複當前的）
    let candidates = readyFiles.filter(f => f.id !== state.currentFile?.id);
    if (candidates.length === 0) {
        candidates = readyFiles;
    }
    
    const randomFile = candidates[Math.floor(Math.random() * candidates.length)];
    console.log('Random file selected:', randomFile.filename);
    
    try {
        // 載入該歌曲的段落和歌詞
        const segmentsData = await api.getSegments(randomFile.id);
        const lyricsData = await api.getLyrics(randomFile.id);
        
        if (!segmentsData.segments || segmentsData.segments.length === 0) {
            console.error('No segments in file:', randomFile.filename);
            shuffleToRandomSegment();
            return;
        }
        
        // 更新狀態
        state.currentFile = randomFile;
        state.segments = segmentsData;
        state.lyrics = lyricsData;
        
        // 選擇隨機段落
        const randomSegmentIndex = Math.floor(Math.random() * segmentsData.segments.length);
        state.currentSegmentIndex = randomSegmentIndex;
        
        // 更新歌曲資訊顯示
        if (elements.shuffleSongInfo) {
            elements.shuffleSongInfo.style.display = 'flex';
        }
        if (elements.shuffleSongName) {
            elements.shuffleSongName.textContent = randomFile.filename;
        }
        
        // 建立播放列表（只建立這一個段落）
        buildPracticePlaylist(randomSegmentIndex);
        state.practiceIndex = 0;
        
        // 更新段落計數
        elements.currentSegment.textContent = randomSegmentIndex + 1;
        elements.totalSegments.textContent = segmentsData.segments.length;
        
        console.log(`Playing segment ${randomSegmentIndex + 1}/${segmentsData.segments.length} from ${randomFile.filename}`);
        
        playCurrentPracticeItem();
        
    } catch (error) {
        console.error('Error loading random segment:', error);
        alert('載入隨機段落失敗！');
    }
}

// ===== 檔案選單功能 =====

// 顯示檔案選單
function showFileMenu(event, fileId) {
    // 移除現有選單
    const existingMenu = document.querySelector('.file-context-menu');
    if (existingMenu) existingMenu.remove();
    
    const file = state.files.find(f => f.id === fileId);
    if (!file) return;
    
    // 建立選單
    const menu = document.createElement('div');
    menu.className = 'file-context-menu';
    menu.innerHTML = `
        <button class="menu-item menu-item-danger" data-action="delete">
            <span>🗑️</span> 刪除檔案
        </button>
    `;
    
    // 定位選單
    const rect = event.target.getBoundingClientRect();
    menu.style.position = 'fixed';
    menu.style.top = `${rect.bottom + 5}px`;
    menu.style.left = `${rect.left - 100}px`;
    menu.style.zIndex = '1000';
    
    document.body.appendChild(menu);
    
    // 綁定選單事件
    menu.querySelector('[data-action="delete"]').addEventListener('click', () => {
        menu.remove();
        deleteFile(fileId);
    });
    
    // 點擊其他地方關閉選單
    setTimeout(() => {
        document.addEventListener('click', function closeMenu(e) {
            if (!menu.contains(e.target)) {
                menu.remove();
                document.removeEventListener('click', closeMenu);
            }
        });
    }, 10);
}

// 刪除檔案
async function deleteFile(fileId) {
    const file = state.files.find(f => f.id === fileId);
    if (!file) return;
    
    const confirmed = confirm(`確定要刪除「${file.filename}」嗎？\n此操作無法復原。`);
    if (!confirmed) return;
    
    try {
        await api.deleteFile(fileId);
        
        // 如果刪除的是當前選中的檔案，清除選擇
        if (state.currentFile?.id === fileId) {
            state.currentFile = null;
            state.segments = [];
            elements.mainContent.style.display = 'none';
        }
        
        // 重新載入檔案列表
        await loadFiles();
        
        console.log(`File ${fileId} deleted successfully`);
    } catch (error) {
        console.error('Delete file error:', error);
        alert('刪除檔案失敗：' + error.message);
    }
}

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

// 隨機練習模式按鈕
elements.shufflePracticeBtn?.addEventListener('click', startShufflePractice);

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

// ===== 設定 Modal 事件 =====

// 開啟設定 Modal
elements.settingsBtn?.addEventListener('click', () => {
    console.log('Settings button clicked');
    // 載入目前設定到 Modal
    loadSettings();
    if (elements.defaultLoop) elements.defaultLoop.checked = defaultSettings.loop;
    if (elements.defaultShuffleMode) elements.defaultShuffleMode.value = defaultSettings.shuffleMode;
    if (elements.defaultTtsRepeat) elements.defaultTtsRepeat.value = defaultSettings.ttsRepeat;
    if (elements.defaultVolume) elements.defaultVolume.value = defaultSettings.ttsVolumeMultiplier;
    if (elements.defaultVolumeValue) elements.defaultVolumeValue.textContent = formatVolumeMultiplier(defaultSettings.ttsVolumeMultiplier);
    if (elements.defaultShowChinese) elements.defaultShowChinese.checked = defaultSettings.showChinese;
    
    if (elements.settingsModal) {
        elements.settingsModal.style.display = 'flex';
        console.log('Settings modal displayed');
    } else {
        console.error('settingsModal element not found!');
    }
});

// 關閉設定 Modal
elements.closeSettingsBtn?.addEventListener('click', () => {
    elements.settingsModal.style.display = 'none';
});

// 點擊外部關閉 Modal
elements.settingsModal?.addEventListener('click', (e) => {
    if (e.target === elements.settingsModal) {
        elements.settingsModal.style.display = 'none';
    }
});

// 設定面板音量滑桿即時更新
elements.defaultVolume?.addEventListener('input', (e) => {
    if (elements.defaultVolumeValue) {
        elements.defaultVolumeValue.textContent = formatVolumeMultiplier(parseInt(e.target.value));
    }
});

// 儲存設定
elements.saveSettingsBtn?.addEventListener('click', () => {
    defaultSettings.loop = elements.defaultLoop?.checked ?? true;
    defaultSettings.shuffleMode = elements.defaultShuffleMode?.value ?? 'playlist';
    defaultSettings.ttsRepeat = parseInt(elements.defaultTtsRepeat?.value ?? 1);
    defaultSettings.ttsVolumeMultiplier = parseInt(elements.defaultVolume?.value ?? 10);
    defaultSettings.showChinese = elements.defaultShowChinese?.checked ?? false;
    
    saveSettings();
    elements.settingsModal.style.display = 'none';
    
    // 更新練習模式的音量滑桿
    if (elements.ttsVolume) {
        elements.ttsVolume.value = defaultSettings.ttsVolumeMultiplier;
    }
    if (elements.volumeValue) {
        elements.volumeValue.textContent = formatVolumeMultiplier(defaultSettings.ttsVolumeMultiplier);
    }
    
    console.log('Settings saved:', defaultSettings);
});

// 練習模式音量控制（即時調整）
elements.ttsVolume?.addEventListener('input', (e) => {
    const multiplier = parseInt(e.target.value);
    if (elements.volumeValue) {
        elements.volumeValue.textContent = formatVolumeMultiplier(multiplier);
    }
    // 套用到 TTS 播放器（使用 Web Audio API 會更好，但這裡用 volume 屬性模擬）
    if (elements.ttsPlayer) {
        // volume 屬性最大只能是 1.0，所以倍數 > 1 需要其他方式
        // 這裡先設定為 1.0，實際音量放大會在播放時處理
        elements.ttsPlayer.volume = Math.min(multiplier / 10, 1.0);
    }
    // 同時更新預設設定
    defaultSettings.ttsVolumeMultiplier = multiplier;
    saveSettings();
});

// ===== 初始化 =====
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded, initializing...');
    // 載入設定
    loadSettings();
    
    // 初始化音量顯示
    if (elements.ttsVolume) {
        elements.ttsVolume.value = defaultSettings.ttsVolumeMultiplier;
    }
    if (elements.volumeValue) {
        elements.volumeValue.textContent = formatVolumeMultiplier(defaultSettings.ttsVolumeMultiplier);
    }
    
    loadFiles();
    console.log('Initialization complete');
});
