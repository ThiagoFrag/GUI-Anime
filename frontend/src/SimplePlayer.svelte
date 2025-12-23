<script>
    import { onMount, onDestroy } from 'svelte';
    import { SetFullscreen } from '../wailsjs/go/main/App.js';
    
    // Props
    export let src = "";
    export let title = "";
    export let episodeTitle = "";
    /** @type {string|null} Capa do anime para exibir */
    export let animeCover = null;
    export let onClose = () => {};
    export let onNext = () => {};
    export let onPrevious = () => {};
    /**
     * skipTimes: informações de abertura/ending para pular
     * {
     *   hasOpening: boolean,
     *   openingStart: number,
     *   openingEnd: number,
     *   hasEnding: boolean,
     *   endingStart: number,
     *   endingEnd: number
     * }
     */
    export let skipTimes = /** @type {{hasOpening?:boolean,openingStart?:number,openingEnd?:number,hasEnding?:boolean,endingStart?:number,endingEnd?:number}|null} */ (null);
    
    // Elements
    let videoEl;
    let containerEl;
    let progressBarEl;
    
    // Video state
    let isPlaying = false;
    let currentTime = 0;
    let duration = 0;
    let volume = 1;
    let isMuted = false;
    let isFullscreen = false;
    let showControls = true;
    let controlsTimeout;
    let isLoading = true;
    let error = null;
    let buffered = 0;
    let isSeeking = false;
    let hoverTime = 0;
    let hoverX = 0;
    let showHoverTime = false;
    
    // Skip state
    let showSkipButton = false;
    let currentSkipType = null;
    
    // Animation states
    let showPlayPauseAnim = false;
    let playPauseAnimIcon = '▶';
    let animTimeout;
    
    // HLS
    let hls = null;
    
    // Audio/Subtitle tracks
    /** @type {{id: number, name: string, lang: string}[]} */
    let audioTracks = [];
    let selectedAudioTrack = -1;
    /** @type {{id: number, name: string, lang: string}[]} */
    let subtitleTracks = [];
    let selectedSubtitleTrack = -1;
    let showAudioMenu = false;
    let showSubtitleMenu = false;
    
    // Reactive skip times debug
    $: if (skipTimes) {
        console.log('[Player] Skip times received:', skipTimes);
    }
    
    function checkSkipSection() {
        if (!skipTimes || !duration || duration === 0) return;
        
        const time = currentTime;
        
        // Check opening (with tolerance)
        if (skipTimes.hasOpening && 
            time >= skipTimes.openingStart - 0.5 && 
            time < skipTimes.openingEnd - 2) {
            if (currentSkipType !== 'opening') {
                console.log('[Skip] Showing opening skip button');
                currentSkipType = 'opening';
                showSkipButton = true;
            }
            return;
        }
        
        // Check ending
        if (skipTimes.hasEnding && 
            time >= skipTimes.endingStart - 0.5 && 
            time < skipTimes.endingEnd - 2) {
            if (currentSkipType !== 'ending') {
                console.log('[Skip] Showing ending skip button');
                currentSkipType = 'ending';
                showSkipButton = true;
            }
            return;
        }
        
        // Not in skip section
        if (showSkipButton) {
            showSkipButton = false;
            currentSkipType = null;
        }
    }
    
    function doSkip() {
        if (!videoEl || !skipTimes) return;
        
        if (currentSkipType === 'opening' && skipTimes.hasOpening) {
            console.log('[Player] Skipping opening to:', skipTimes.openingEnd);
            videoEl.currentTime = skipTimes.openingEnd;
        } else if (currentSkipType === 'ending' && skipTimes.hasEnding) {
            console.log('[Player] Skipping ending to:', skipTimes.endingEnd);
            videoEl.currentTime = skipTimes.endingEnd;
        }
        
        showSkipButton = false;
        currentSkipType = null;
    }
    
    // ===== FUNÇÕES DE ÁUDIO E LEGENDA =====
    function selectAudioTrack(trackId) {
        if (!hls) return;
        console.log('[Player] Selecionando áudio:', trackId);
        hls.audioTrack = trackId;
        selectedAudioTrack = trackId;
        showAudioMenu = false;
    }
    
    function selectSubtitleTrack(trackId) {
        if (!hls) return;
        console.log('[Player] Selecionando legenda:', trackId);
        hls.subtitleTrack = trackId;
        selectedSubtitleTrack = trackId;
        showSubtitleMenu = false;
    }
    
    function toggleAudioMenu() {
        showAudioMenu = !showAudioMenu;
        showSubtitleMenu = false;
    }
    
    function toggleSubtitleMenu() {
        showSubtitleMenu = !showSubtitleMenu;
        showAudioMenu = false;
    }
    
    function getLanguageName(lang) {
        const names = {
            'por': 'Português',
            'pt': 'Português',
            'pt-BR': 'Português (BR)',
            'pt-PT': 'Português (PT)',
            'eng': 'English',
            'en': 'English',
            'jpn': 'Japonês',
            'ja': 'Japonês',
            'spa': 'Espanhol',
            'es': 'Espanhol',
            'und': 'Desconhecido'
        };
        return names[lang] || lang;
    }
    
    function initPlayer() {
        if (!src || !videoEl) return;
        
        console.log('[Player] Initializing with src:', src);
        
        // Cleanup previous HLS
        if (hls) {
            hls.destroy();
            hls = null;
        }
        
        const isHLS = src.includes('.m3u8');
        
        if (isHLS && window['Hls'] && window['Hls'].isSupported()) {
            console.log('[Player] Using HLS.js');
            hls = new window['Hls']({
                enableWorker: true,
                lowLatencyMode: true,
                backBufferLength: 90,
                maxBufferLength: 30,
                maxMaxBufferLength: 600,
                maxBufferSize: 60 * 1000 * 1000,
                maxBufferHole: 0.5,
                highBufferWatchdogPeriod: 2,
                nudgeOffset: 0.1,
                nudgeMaxRetry: 5,
                startLevel: -1,
                abrEwmaDefaultEstimate: 500000,
                abrBandWidthFactor: 0.95,
                abrBandWidthUpFactor: 0.7,
                fragLoadingTimeOut: 20000,
                fragLoadingMaxRetry: 6,
                levelLoadingTimeOut: 10000,
                manifestLoadingTimeOut: 10000,
            });
            
            hls.loadSource(src);
            hls.attachMedia(videoEl);
            
            hls.on(window['Hls'].Events.MANIFEST_PARSED, (event, data) => {
                console.log('[HLS] Manifest parsed, levels:', data.levels.length);
                isLoading = false;
                
                // Detecta tracks de áudio
                if (hls.audioTracks && hls.audioTracks.length > 0) {
                    audioTracks = hls.audioTracks.map((t, i) => ({
                        id: i,
                        name: t.name || `Áudio ${i + 1}`,
                        lang: t.lang || 'und'
                    }));
                    selectedAudioTrack = hls.audioTrack;
                    console.log('[HLS] Audio tracks:', audioTracks);
                }
                
                // Detecta tracks de legenda
                if (hls.subtitleTracks && hls.subtitleTracks.length > 0) {
                    subtitleTracks = hls.subtitleTracks.map((t, i) => ({
                        id: i,
                        name: t.name || `Legenda ${i + 1}`,
                        lang: t.lang || 'und'
                    }));
                    selectedSubtitleTrack = hls.subtitleTrack;
                    console.log('[HLS] Subtitle tracks:', subtitleTracks);
                }
                
                videoEl.play().catch(e => console.warn('[Player] Autoplay blocked:', e));
            });
            
            hls.on(window['Hls'].Events.LEVEL_SWITCHED, (event, data) => {
                const level = hls.levels[data.level];
                if (level) {
                    console.log(`[HLS] Quality: ${level.height}p @ ${Math.round(level.bitrate/1000)}kbps`);
                }
            });
            
            hls.on(window['Hls'].Events.ERROR, (event, data) => {
                console.error('[HLS] Error:', data.type, data.details);
                if (data.fatal) {
                    switch(data.type) {
                        case window['Hls'].ErrorTypes.NETWORK_ERROR:
                            console.log('[HLS] Network error, trying to recover...');
                            hls.startLoad();
                            break;
                        case window['Hls'].ErrorTypes.MEDIA_ERROR:
                            console.log('[HLS] Media error, trying to recover...');
                            hls.recoverMediaError();
                            break;
                        default:
                            error = 'Erro ao carregar vídeo';
                            isLoading = false;
                            break;
                    }
                }
            });
        } else if (isHLS && videoEl.canPlayType('application/vnd.apple.mpegurl')) {
            videoEl.src = src;
        } else {
            videoEl.src = src;
        }
    }
    
    function togglePlay() {
        if (!videoEl) return;
        
        if (videoEl.paused) {
            videoEl.play().catch(e => {
                console.error('[Player] Play error:', e);
                error = 'Erro ao reproduzir';
            });
            triggerPlayPauseAnim('▶');
        } else {
            videoEl.pause();
            triggerPlayPauseAnim('⏸');
        }
    }
    
    function triggerPlayPauseAnim(icon) {
        clearTimeout(animTimeout);
        playPauseAnimIcon = icon;
        showPlayPauseAnim = true;
        animTimeout = setTimeout(() => {
            showPlayPauseAnim = false;
        }, 400);
    }
    
    function seek(e) {
        if (!videoEl || !duration || !progressBarEl) return;
        const rect = progressBarEl.getBoundingClientRect();
        const percent = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
        videoEl.currentTime = percent * duration;
    }
    
    function handleProgressHover(e) {
        if (!progressBarEl || !duration) return;
        const rect = progressBarEl.getBoundingClientRect();
        const percent = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
        hoverTime = percent * duration;
        hoverX = e.clientX - rect.left;
        showHoverTime = true;
    }
    
    function handleProgressLeave() {
        showHoverTime = false;
    }
    
    function setVolume(e) {
        if (!videoEl) return;
        const rect = e.currentTarget.getBoundingClientRect();
        volume = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
        videoEl.volume = volume;
        isMuted = volume === 0;
    }
    
    function toggleMute() {
        if (!videoEl) return;
        isMuted = !isMuted;
        videoEl.muted = isMuted;
    }
    
    async function toggleFullscreen() {
        try {
            isFullscreen = !isFullscreen;
            await SetFullscreen(isFullscreen);
        } catch (err) {
            try {
                if (!document.fullscreenElement && containerEl) {
                    await containerEl.requestFullscreen();
                    isFullscreen = true;
                } else if (document.fullscreenElement) {
                    await document.exitFullscreen();
                    isFullscreen = false;
                }
            } catch (e) {
                console.error('[Player] Fullscreen error:', e);
            }
        }
    }
    
    function formatTime(seconds) {
        if (isNaN(seconds) || !isFinite(seconds)) return '0:00';
        const hrs = Math.floor(seconds / 3600);
        const mins = Math.floor((seconds % 3600) / 60);
        const secs = Math.floor(seconds % 60);
        if (hrs > 0) {
            return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        }
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }
    
    function showControlsTemporarily() {
        showControls = true;
        clearTimeout(controlsTimeout);
        controlsTimeout = setTimeout(() => {
            if (isPlaying && !isSeeking) showControls = false;
        }, 3000);
    }
    
    function handleKeydown(e) {
        if (!videoEl) return;
        
        switch(e.key) {
            case ' ':
            case 'k':
                e.preventDefault();
                togglePlay();
                break;
            case 'ArrowLeft':
                e.preventDefault();
                videoEl.currentTime = Math.max(0, videoEl.currentTime - 5);
                showControlsTemporarily();
                break;
            case 'ArrowRight':
                e.preventDefault();
                videoEl.currentTime = Math.min(duration, videoEl.currentTime + 5);
                showControlsTemporarily();
                break;
            case 'j':
                videoEl.currentTime = Math.max(0, videoEl.currentTime - 10);
                showControlsTemporarily();
                break;
            case 'l':
                videoEl.currentTime = Math.min(duration, videoEl.currentTime + 10);
                showControlsTemporarily();
                break;
            case 'ArrowUp':
                e.preventDefault();
                volume = Math.min(1, volume + 0.1);
                videoEl.volume = volume;
                showControlsTemporarily();
                break;
            case 'ArrowDown':
                e.preventDefault();
                volume = Math.max(0, volume - 0.1);
                videoEl.volume = volume;
                showControlsTemporarily();
                break;
            case 'f':
                toggleFullscreen();
                break;
            case 'm':
                toggleMute();
                break;
            case 'n':
                onNext();
                break;
            case 'p':
                onPrevious();
                break;
            case 's':
                if (showSkipButton) doSkip();
                break;
            case 'Escape':
                if (isFullscreen) {
                    SetFullscreen(false).catch(() => {});
                    isFullscreen = false;
                } else {
                    onClose();
                }
                break;
        }
    }
    
    function handleVideoClick(e) {
        // Prevent double-triggering with double-click
        if (e.detail === 1) {
            setTimeout(() => {
                if (!e.defaultPrevented) togglePlay();
            }, 200);
        }
    }
    
    function handleDoubleClick(e) {
        e.preventDefault();
        toggleFullscreen();
    }
    
    function skipForward() {
        if (!videoEl) return;
        videoEl.currentTime = Math.min(duration, videoEl.currentTime + 10);
        showControlsTemporarily();
    }
    
    function skipBackward() {
        if (!videoEl) return;
        videoEl.currentTime = Math.max(0, videoEl.currentTime - 10);
        showControlsTemporarily();
    }
    
    onMount(() => {
        if (!videoEl) return;
        
        // Optimize video rendering
        videoEl.style.willChange = 'transform';
        
        videoEl.addEventListener('loadedmetadata', () => {
            duration = videoEl.duration || 0;
            isLoading = false;
            console.log('[Player] Metadata loaded, duration:', duration);
        });
        
        videoEl.addEventListener('timeupdate', () => {
            if (!isSeeking) {
                currentTime = videoEl.currentTime || 0;
            }
            if (videoEl.buffered.length > 0) {
                buffered = videoEl.buffered.end(videoEl.buffered.length - 1);
            }
            checkSkipSection();
        });
        
        videoEl.addEventListener('progress', () => {
            if (videoEl.buffered.length > 0) {
                buffered = videoEl.buffered.end(videoEl.buffered.length - 1);
            }
        });
        
        videoEl.addEventListener('play', () => { 
            isPlaying = true; 
            showControlsTemporarily();
        });
        videoEl.addEventListener('pause', () => { 
            isPlaying = false; 
            showControls = true;
        });
        videoEl.addEventListener('ended', () => { 
            isPlaying = false;
            showControls = true;
        });
        videoEl.addEventListener('waiting', () => { isLoading = true; });
        videoEl.addEventListener('canplay', () => { 
            isLoading = false; 
            error = null;
        });
        videoEl.addEventListener('playing', () => { 
            isLoading = false; 
        });
        
        videoEl.addEventListener('error', () => {
            const codes = { 1: 'Abortado', 2: 'Rede', 3: 'Decodificação', 4: 'Formato não suportado' };
            error = codes[videoEl?.error?.code] || 'Erro desconhecido';
            isLoading = false;
        });
        
        initPlayer();
        
        document.addEventListener('keydown', handleKeydown);
        document.addEventListener('fullscreenchange', () => {
            isFullscreen = !!document.fullscreenElement;
        });
    });
    
    onDestroy(() => {
        if (hls) {
            hls.destroy();
            hls = null;
        }
        document.removeEventListener('keydown', handleKeydown);
        clearTimeout(controlsTimeout);
        clearTimeout(animTimeout);
        if (isFullscreen) {
            SetFullscreen(false).catch(() => {});
        }
    });
    
    // Track source changes
    let prevSrc = "";
    $: if (src && videoEl && src !== prevSrc) {
        console.log('[Player] Source changed:', src);
        prevSrc = src;
        isLoading = true;
        error = null;
        currentTime = 0;
        duration = 0;
        isPlaying = false;
        showSkipButton = false;
        currentSkipType = null;
        setTimeout(() => initPlayer(), 50);
    }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div 
    class="goanime-player" 
    class:fullscreen={isFullscreen}
    class:hide-cursor={isPlaying && !showControls}
    bind:this={containerEl}
    onmousemove={showControlsTemporarily}
>
    <!-- Ambient Background Glow -->
    <div class="ambient-glow"></div>
    
    <!-- Video Element -->
    <!-- svelte-ignore a11y_media_has_caption -->
    <video
        bind:this={videoEl}
        class="video-element"
        playsinline
        preload="auto"
        onclick={handleVideoClick}
        ondblclick={handleDoubleClick}
    ></video>
    
    <!-- Top Gradient & Header -->
    <div class="top-overlay" class:visible={showControls || !isPlaying}>
        <div class="header-content">
            <button type="button" class="btn-back" onclick={onClose} aria-label="Voltar">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M19 12H5M12 19l-7-7 7-7"/>
                </svg>
            </button>
            {#if animeCover}
                <div class="anime-cover">
                    <img src={animeCover} alt={title} />
                </div>
            {/if}
            <div class="title-info">
                <span class="anime-title">{title || 'GoAnime Player'}</span>
                {#if episodeTitle}
                    <span class="episode-title">{episodeTitle}</span>
                {/if}
            </div>
        </div>
    </div>
    
    <!-- Center Play/Pause Animation -->
    {#if showPlayPauseAnim}
        <div class="center-anim">
            <div class="anim-circle">
                <span>{playPauseAnimIcon}</span>
            </div>
        </div>
    {/if}
    
    <!-- Loading Spinner -->
    {#if isLoading}
        <div class="loading-overlay">
            <div class="loader">
                <div class="loader-ring"></div>
                <div class="loader-ring"></div>
                <div class="loader-ring"></div>
                <span class="loader-text">Carregando</span>
            </div>
        </div>
    {/if}
    
    <!-- Error State -->
    {#if error}
        <div class="error-overlay">
            <div class="error-content">
                <div class="error-icon">⚠️</div>
                <p class="error-message">{error}</p>
                <div class="error-actions">
                    <button onclick={() => { error = null; isLoading = true; initPlayer(); }} aria-label="Tentar novamente">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                        </svg>
                        Tentar novamente
                    </button>
                    <button onclick={onClose} aria-label="Voltar">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M19 12H5M12 19l-7-7 7-7"/>
                        </svg>
                        Voltar
                    </button>
                </div>
            </div>
        </div>
    {/if}
    
    <!-- Paused Overlay -->
    {#if !isPlaying && !isLoading && !error}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="paused-overlay" tabindex="0" role="button" aria-label="Play/Pause" onclick={togglePlay} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') togglePlay(); }}>
            <div class="big-play-btn">
                <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M8 5v14l11-7z"/>
                </svg>
            </div>
        </div>
    {/if}
    
    <!-- Skip Button -->
    {#if showSkipButton && currentSkipType}
        <button type="button" class="skip-btn" onclick={doSkip} aria-label="Pular abertura/ending">
            <span class="skip-content">
                <svg viewBox="0 0 24 24" fill="currentColor" class="skip-icon">
                    <path d="M5 4l10 8-10 8V4zM19 4v16h-2V4h2z"/>
                </svg>
                <span class="skip-label">
                    {currentSkipType === 'opening' ? 'PULAR ABERTURA' : 'PULAR ENCERRAMENTO'}
                </span>
            </span>
            <span class="skip-key">S</span>
        </button>
    {/if}
    
    <!-- Bottom Controls -->
    <div class="bottom-controls" class:visible={showControls || !isPlaying}>
        <!-- Progress Bar Container -->
        <div class="progress-container">
            <!-- Skip Markers -->
            {#if skipTimes && duration > 0}
                {#if skipTimes.hasOpening}
                    <div 
                        class="skip-marker opening"
                        style="left: {(skipTimes.openingStart / duration) * 100}%; width: {((skipTimes.openingEnd - skipTimes.openingStart) / duration) * 100}%"
                    ></div>
                {/if}
                {#if skipTimes.hasEnding}
                    <div 
                        class="skip-marker ending"
                        style="left: {(skipTimes.endingStart / duration) * 100}%; width: {((skipTimes.endingEnd - skipTimes.endingStart) / duration) * 100}%"
                    ></div>
                {/if}
            {/if}
            
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div 
                class="progress-bar"
                bind:this={progressBarEl}
                onclick={seek}
                tabindex="0"
                role="slider"
                aria-label="Barra de progresso"
                aria-valuenow={Math.round(currentTime)}
                aria-valuemin="0"
                aria-valuemax={Math.round(duration) || 100}
                onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') seek(e); }}
                onmousemove={handleProgressHover}
                onmouseleave={handleProgressLeave}
            >
                <div class="progress-buffered" style="width: {duration ? (buffered / duration) * 100 : 0}%"></div>
                <div class="progress-played" style="width: {duration ? (currentTime / duration) * 100 : 0}%"></div>
                <div class="progress-handle" style="left: {duration ? (currentTime / duration) * 100 : 0}%"></div>
                
                <!-- Hover Preview -->
                {#if showHoverTime}
                    <div class="hover-time" style="left: {hoverX}px">
                        {formatTime(hoverTime)}
                    </div>
                {/if}
            </div>
        </div>
        
        <!-- Control Buttons -->
        <div class="controls-row">
            <div class="controls-left">
                <!-- Previous -->
                <button type="button" class="ctrl-btn" onclick={onPrevious} title="Episódio anterior (P)">
                    <svg viewBox="0 0 24 24" fill="currentColor">
                        <path d="M6 6h2v12H6zm3.5 6l8.5 6V6z"/>
                    </svg>
                </button>
                
                <!-- Skip Back -->
                <button type="button" class="ctrl-btn" onclick={skipBackward} title="Voltar 10s (J)">
                    <svg viewBox="0 0 24 24" fill="currentColor">
                        <path d="M11.99 5V1l-5 5 5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6h-2c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/>
                    </svg>
                </button>
                
                <!-- Play/Pause -->
                <button type="button" class="ctrl-btn play-pause" onclick={togglePlay} title="Play/Pause (K)">
                    {#if isPlaying}
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
                        </svg>
                    {:else}
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M8 5v14l11-7z"/>
                        </svg>
                    {/if}
                </button>
                
                <!-- Skip Forward -->
                <button type="button" class="ctrl-btn" onclick={skipForward} title="Avançar 10s (L)">
                    <svg viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 5V1l5 5-5 5V7c-3.31 0-6 2.69-6 6s2.69 6 6 6 6-2.69 6-6h2c0 4.42-3.58 8-8 8s-8-3.58-8-8 3.58-8 8-8z"/>
                    </svg>
                </button>
                
                <!-- Next -->
                <button type="button" class="ctrl-btn" onclick={onNext} title="Próximo episódio (N)">
                    <svg viewBox="0 0 24 24" fill="currentColor">
                        <path d="M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z"/>
                    </svg>
                </button>
                
                <!-- Volume -->
                <div class="volume-control">
                    <button type="button" class="ctrl-btn" onclick={toggleMute} title="Mudo (M)">
                        {#if isMuted || volume === 0}
                            <svg viewBox="0 0 24 24" fill="currentColor">
                                <path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"/>
                            </svg>
                        {:else if volume < 0.5}
                            <svg viewBox="0 0 24 24" fill="currentColor">
                                <path d="M18.5 12c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM5 9v6h4l5 5V4L9 9H5z"/>
                            </svg>
                        {:else}
                            <svg viewBox="0 0 24 24" fill="currentColor">
                                <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
                            </svg>
                        {/if}
                    </button>
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <div class="volume-slider" onclick={setVolume} tabindex="0" role="slider" aria-label="Controle de volume" aria-valuenow={Math.round(volume * 100)} aria-valuemin="0" aria-valuemax="100" onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') setVolume(e); }}>
                        <div class="volume-track">
                            <div class="volume-fill" style="width: {isMuted ? 0 : volume * 100}%"></div>
                        </div>
                    </div>
                </div>
                
                <!-- Time -->
                <span class="time-display">
                    <span class="time-current">{formatTime(currentTime)}</span>
                    <span class="time-separator">/</span>
                    <span class="time-duration">{formatTime(duration)}</span>
                </span>
            </div>
            
            <div class="controls-right">
                <!-- Audio Track -->
                {#if audioTracks.length > 1}
                    <div class="track-menu-container">
                        <button type="button" class="ctrl-btn" onclick={toggleAudioMenu} title="Áudio">
                            <svg viewBox="0 0 24 24" fill="currentColor">
                                <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
                            </svg>
                            <span class="track-label">Áudio</span>
                        </button>
                        {#if showAudioMenu}
                            <div class="track-menu">
                                <div class="track-menu-title">Áudio</div>
                                {#each audioTracks as track}
                                    <button 
                                        type="button" 
                                        class="track-option" 
                                        class:active={selectedAudioTrack === track.id}
                                        onclick={() => selectAudioTrack(track.id)}
                                    >
                                        <span class="track-check">{selectedAudioTrack === track.id ? '✓' : ''}</span>
                                        <span class="track-name">{track.name}</span>
                                        <span class="track-lang">{getLanguageName(track.lang)}</span>
                                    </button>
                                {/each}
                            </div>
                        {/if}
                    </div>
                {/if}
                
                <!-- Subtitle Track -->
                {#if subtitleTracks.length > 0}
                    <div class="track-menu-container">
                        <button type="button" class="ctrl-btn" onclick={toggleSubtitleMenu} title="Legendas">
                            <svg viewBox="0 0 24 24" fill="currentColor">
                                <path d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z"/>
                            </svg>
                            <span class="track-label">CC</span>
                        </button>
                        {#if showSubtitleMenu}
                            <div class="track-menu">
                                <div class="track-menu-title">Legendas</div>
                                <button 
                                    type="button" 
                                    class="track-option" 
                                    class:active={selectedSubtitleTrack === -1}
                                    onclick={() => selectSubtitleTrack(-1)}
                                >
                                    <span class="track-check">{selectedSubtitleTrack === -1 ? '✓' : ''}</span>
                                    <span class="track-name">Desativado</span>
                                </button>
                                {#each subtitleTracks as track}
                                    <button 
                                        type="button" 
                                        class="track-option" 
                                        class:active={selectedSubtitleTrack === track.id}
                                        onclick={() => selectSubtitleTrack(track.id)}
                                    >
                                        <span class="track-check">{selectedSubtitleTrack === track.id ? '✓' : ''}</span>
                                        <span class="track-name">{track.name}</span>
                                        <span class="track-lang">{getLanguageName(track.lang)}</span>
                                    </button>
                                {/each}
                            </div>
                        {/if}
                    </div>
                {/if}
                
                <!-- Fullscreen -->
                <button type="button" class="ctrl-btn" onclick={toggleFullscreen} title="Tela cheia (F)">
                    {#if isFullscreen}
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"/>
                        </svg>
                    {:else}
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/>
                        </svg>
                    {/if}
                </button>
            </div>
        </div>
    </div>
</div>

<style>
    /* ========================================
       GOANIME PLAYER - MODERN UI
       ======================================== */
    
    .goanime-player {
        position: fixed;
        inset: 0;
        background: #000;
        z-index: 9999;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
    }
    
    .goanime-player.hide-cursor {
        cursor: none;
    }
    
    /* Ambient Glow Effect */
    .ambient-glow {
        position: absolute;
        inset: -50%;
        background: radial-gradient(ellipse at center, rgba(245, 87, 108, 0.06) 0%, transparent 70%);
        pointer-events: none;
        animation: ambientPulse 8s ease-in-out infinite;
    }
    
    @keyframes ambientPulse {
        0%, 100% { opacity: 0.5; transform: scale(1); }
        50% { opacity: 0.8; transform: scale(1.1); }
    }
    
    /* Video Element - Optimized for smooth playback */
    .video-element {
        width: 100%;
        height: 100%;
        object-fit: contain;
        background: #000;
        /* GPU acceleration for smooth playback */
        transform: translateZ(0);
        will-change: contents;
        /* Prevent layout shifts */
        contain: strict;
    }
    
    /* ========================================
       TOP OVERLAY / HEADER
       ======================================== */
    
    .top-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        padding: 20px 24px;
        background: linear-gradient(to bottom, 
            rgba(0, 0, 0, 0.9) 0%,
            rgba(0, 0, 0, 0.6) 50%,
            transparent 100%
        );
        opacity: 0;
        transform: translateY(-10px);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        z-index: 20;
        pointer-events: none;
    }
    
    .top-overlay.visible {
        opacity: 1;
        transform: translateY(0);
        pointer-events: auto;
    }
    
    .header-content {
        display: flex;
        align-items: center;
        gap: 16px;
    }
    
    .btn-back {
        width: 44px;
        height: 44px;
        border: none;
        background: rgba(255, 255, 255, 0.1);
        backdrop-filter: blur(10px);
        border-radius: 50%;
        color: #fff;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s ease;
        flex-shrink: 0;
    }
    
    .btn-back:hover {
        background: rgba(245, 87, 108, 0.8);
        transform: scale(1.1);
    }
    
    .btn-back svg {
        width: 22px;
        height: 22px;
    }
    
    .anime-cover {
        width: 50px;
        height: 70px;
        border-radius: 6px;
        overflow: hidden;
        flex-shrink: 0;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
    }
    
    .anime-cover img {
        width: 100%;
        height: 100%;
        object-fit: cover;
    }
    
    .title-info {
        display: flex;
        flex-direction: column;
        gap: 4px;
        min-width: 0;
    }
    
    .anime-title {
        font-size: 1.1rem;
        font-weight: 600;
        color: #fff;
        text-shadow: 0 2px 10px rgba(0, 0, 0, 0.5);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    
    .episode-title {
        font-size: 0.9rem;
        color: rgba(255, 255, 255, 0.7);
        text-shadow: 0 1px 5px rgba(0, 0, 0, 0.5);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    
    /* ========================================
       CENTER ANIMATIONS
       ======================================== */
    
    .center-anim {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 25;
        pointer-events: none;
    }
    
    .anim-circle {
        width: 80px;
        height: 80px;
        background: rgba(245, 87, 108, 0.9);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        animation: popIn 0.4s cubic-bezier(0.68, -0.55, 0.265, 1.55);
    }
    
    .anim-circle span {
        font-size: 2rem;
        color: #fff;
    }
    
    @keyframes popIn {
        0% { transform: scale(0.5); opacity: 0; }
        50% { transform: scale(1.2); opacity: 1; }
        100% { transform: scale(1); opacity: 0; }
    }
    
    /* ========================================
       LOADING STATE
       ======================================== */
    
    .loading-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 0, 0, 0.6);
        backdrop-filter: blur(8px);
        z-index: 30;
    }
    
    .loader {
        position: relative;
        width: 80px;
        height: 80px;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    
    .loader-ring {
        position: absolute;
        width: 100%;
        height: 100%;
        border-radius: 50%;
        border: 3px solid transparent;
        border-top-color: #f5576c;
        animation: loaderSpin 1.2s cubic-bezier(0.5, 0, 0.5, 1) infinite;
    }
    
    .loader-ring:nth-child(1) {
        animation-delay: -0.45s;
    }
    
    .loader-ring:nth-child(2) {
        width: 70%;
        height: 70%;
        border-top-color: #ff8a5b;
        animation-delay: -0.3s;
    }
    
    .loader-ring:nth-child(3) {
        width: 40%;
        height: 40%;
        border-top-color: #ffc371;
        animation-delay: -0.15s;
    }
    
    @keyframes loaderSpin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
    
    .loader-text {
        font-size: 0.7rem;
        color: rgba(255, 255, 255, 0.6);
        text-transform: uppercase;
        letter-spacing: 2px;
        position: absolute;
        bottom: -30px;
        animation: pulse 1.5s ease-in-out infinite;
    }
    
    @keyframes pulse {
        0%, 100% { opacity: 0.6; }
        50% { opacity: 1; }
    }
    
    /* ========================================
       ERROR STATE
       ======================================== */
    
    .error-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 0, 0, 0.85);
        backdrop-filter: blur(15px);
        z-index: 30;
    }
    
    .error-content {
        text-align: center;
        padding: 40px;
    }
    
    .error-icon {
        font-size: 4rem;
        margin-bottom: 20px;
        animation: shake 0.5s ease-in-out;
    }
    
    @keyframes shake {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-10px); }
        75% { transform: translateX(10px); }
    }
    
    .error-message {
        font-size: 1.2rem;
        color: #fff;
        margin-bottom: 24px;
    }
    
    .error-actions {
        display: flex;
        gap: 12px;
        justify-content: center;
    }
    
    .error-actions button {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 12px 24px;
        border: none;
        border-radius: 30px;
        font-size: 0.95rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;
    }
    
    .error-actions button:first-child {
        background: linear-gradient(135deg, #f5576c, #ff8a5b);
        color: #fff;
    }
    
    .error-actions button:last-child {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
    }
    
    .error-actions button:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 20px rgba(245, 87, 108, 0.3);
    }
    
    .error-actions button svg {
        width: 18px;
        height: 18px;
    }
    
    /* ========================================
       PAUSED OVERLAY
       ======================================== */
    
    .paused-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        z-index: 15;
    }
    
    .big-play-btn {
        width: 90px;
        height: 90px;
        background: linear-gradient(135deg, rgba(245, 87, 108, 0.95), rgba(255, 138, 91, 0.95));
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 10px 40px rgba(245, 87, 108, 0.4);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        animation: pulseGlow 2s ease-in-out infinite;
    }
    
    .big-play-btn:hover {
        transform: scale(1.1);
        box-shadow: 0 15px 50px rgba(245, 87, 108, 0.5);
    }
    
    .big-play-btn svg {
        width: 40px;
        height: 40px;
        color: #fff;
        margin-left: 5px;
    }
    
    @keyframes pulseGlow {
        0%, 100% { box-shadow: 0 10px 40px rgba(245, 87, 108, 0.4); }
        50% { box-shadow: 0 10px 60px rgba(245, 87, 108, 0.6); }
    }
    
    /* ========================================
       SKIP BUTTON
       ======================================== */
    
    .skip-btn {
        position: absolute;
        bottom: 120px;
        right: 24px;
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 14px 24px;
        background: rgba(0, 0, 0, 0.85);
        backdrop-filter: blur(20px);
        border: 2px solid rgba(255, 255, 255, 0.2);
        border-radius: 12px;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 600;
        cursor: pointer;
        z-index: 25;
        animation: slideInRight 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        transition: all 0.25s ease;
    }
    
    @keyframes slideInRight {
        from {
            opacity: 0;
            transform: translateX(40px);
        }
        to {
            opacity: 1;
            transform: translateX(0);
        }
    }
    
    .skip-btn:hover {
        background: linear-gradient(135deg, #f5576c, #ff8a5b);
        border-color: transparent;
        transform: scale(1.05);
        box-shadow: 0 8px 30px rgba(245, 87, 108, 0.4);
    }
    
    .skip-content {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    
    .skip-icon {
        width: 20px;
        height: 20px;
    }
    
    .skip-label {
        letter-spacing: 1px;
    }
    
    .skip-key {
        padding: 4px 10px;
        background: rgba(255, 255, 255, 0.15);
        border-radius: 6px;
        font-size: 0.8rem;
        font-weight: 700;
        letter-spacing: 0;
    }
    
    /* ========================================
       BOTTOM CONTROLS
       ======================================== */
    
    .bottom-controls {
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        padding: 0 24px 20px;
        background: linear-gradient(to top,
            rgba(0, 0, 0, 0.95) 0%,
            rgba(0, 0, 0, 0.7) 50%,
            transparent 100%
        );
        opacity: 0;
        transform: translateY(10px);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        z-index: 20;
        pointer-events: none;
    }
    
    .bottom-controls.visible {
        opacity: 1;
        transform: translateY(0);
        pointer-events: auto;
    }
    
    /* ========================================
       PROGRESS BAR
       ======================================== */
    
    .progress-container {
        position: relative;
        padding: 12px 0;
        margin-bottom: 8px;
    }
    
    .progress-bar {
        height: 5px;
        background: rgba(255, 255, 255, 0.2);
        border-radius: 10px;
        cursor: pointer;
        position: relative;
        transition: height 0.15s ease;
    }
    
    .progress-bar:hover {
        height: 8px;
    }
    
    .progress-buffered {
        position: absolute;
        top: 0;
        left: 0;
        height: 100%;
        background: rgba(255, 255, 255, 0.3);
        border-radius: 10px;
        transition: width 0.1s linear;
    }
    
    .progress-played {
        position: absolute;
        top: 0;
        left: 0;
        height: 100%;
        background: linear-gradient(90deg, #f5576c, #ff8a5b);
        border-radius: 10px;
        transition: width 0.1s linear;
    }
    
    .progress-handle {
        position: absolute;
        top: 50%;
        width: 16px;
        height: 16px;
        background: #fff;
        border-radius: 50%;
        transform: translate(-50%, -50%) scale(0);
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
        transition: transform 0.15s ease;
    }
    
    .progress-bar:hover .progress-handle {
        transform: translate(-50%, -50%) scale(1);
    }
    
    .hover-time {
        position: absolute;
        bottom: 20px;
        transform: translateX(-50%);
        padding: 6px 10px;
        background: rgba(0, 0, 0, 0.9);
        border-radius: 6px;
        font-size: 0.8rem;
        color: #fff;
        white-space: nowrap;
        pointer-events: none;
    }
    
    /* Skip Markers on Progress */
    .skip-marker {
        position: absolute;
        top: 0;
        height: 100%;
        border-radius: 10px;
        opacity: 0.6;
        pointer-events: none;
        z-index: 1;
    }
    
    .skip-marker.opening {
        background: linear-gradient(90deg, #f5576c, #ff6b8a);
    }
    
    .skip-marker.ending {
        background: linear-gradient(90deg, #5865F2, #7289da);
    }
    
    /* ========================================
       CONTROL BUTTONS
       ======================================== */
    
    .controls-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    
    .controls-left, .controls-right {
        display: flex;
        align-items: center;
        gap: 4px;
    }
    
    .ctrl-btn {
        width: 44px;
        height: 44px;
        border: none;
        background: transparent;
        color: #fff;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 50%;
        transition: all 0.2s ease;
    }
    
    .ctrl-btn:hover {
        background: rgba(255, 255, 255, 0.1);
        transform: scale(1.1);
    }
    
    .ctrl-btn svg {
        width: 24px;
        height: 24px;
    }
    
    .ctrl-btn.play-pause {
        width: 52px;
        height: 52px;
        background: rgba(255, 255, 255, 0.1);
        margin: 0 4px;
    }
    
    .ctrl-btn.play-pause:hover {
        background: rgba(245, 87, 108, 0.8);
    }
    
    .ctrl-btn.play-pause svg {
        width: 28px;
        height: 28px;
    }
    
    /* ========================================
       VOLUME CONTROL
       ======================================== */
    
    .volume-control {
        display: flex;
        align-items: center;
        gap: 4px;
        margin-left: 8px;
    }
    
    .volume-slider {
        width: 0;
        overflow: hidden;
        transition: width 0.2s ease;
        cursor: pointer;
    }
    
    .volume-control:hover .volume-slider {
        width: 80px;
    }
    
    .volume-track {
        width: 80px;
        height: 4px;
        background: rgba(255, 255, 255, 0.3);
        border-radius: 10px;
        overflow: hidden;
    }
    
    .volume-fill {
        height: 100%;
        background: #fff;
        border-radius: 10px;
        transition: width 0.1s ease;
    }
    
    /* ========================================
       TIME DISPLAY
       ======================================== */
    
    .time-display {
        margin-left: 12px;
        font-size: 0.85rem;
        color: rgba(255, 255, 255, 0.9);
        font-variant-numeric: tabular-nums;
    }
    
    .time-separator {
        margin: 0 4px;
        opacity: 0.5;
    }
    
    .time-duration {
        opacity: 0.7;
    }
    
    /* ========================================
       TRACK MENUS (AUDIO/SUBTITLE)
       ======================================== */
    
    .track-menu-container {
        position: relative;
    }
    
    .track-label {
        display: none;
        font-size: 0.75rem;
        margin-left: 4px;
    }
    
    .track-menu {
        position: absolute;
        bottom: 100%;
        right: 0;
        background: rgba(20, 20, 20, 0.95);
        border-radius: 8px;
        border: 1px solid rgba(255, 255, 255, 0.1);
        padding: 8px 0;
        min-width: 200px;
        max-height: 300px;
        overflow-y: auto;
        margin-bottom: 8px;
        backdrop-filter: blur(10px);
        z-index: 1000;
        animation: fadeInUp 0.2s ease-out;
    }
    
    @keyframes fadeInUp {
        from {
            opacity: 0;
            transform: translateY(10px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .track-menu-title {
        padding: 8px 16px;
        font-size: 0.85rem;
        font-weight: 600;
        color: rgba(255, 255, 255, 0.7);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        margin-bottom: 4px;
    }
    
    .track-option {
        display: flex;
        align-items: center;
        gap: 10px;
        width: 100%;
        padding: 10px 16px;
        background: none;
        border: none;
        color: rgba(255, 255, 255, 0.9);
        font-size: 0.9rem;
        cursor: pointer;
        transition: background 0.15s ease;
        text-align: left;
    }
    
    .track-option:hover {
        background: rgba(255, 255, 255, 0.1);
    }
    
    .track-option.active {
        background: rgba(245, 87, 108, 0.2);
    }
    
    .track-check {
        width: 20px;
        color: #f5576c;
        font-weight: bold;
    }
    
    .track-name {
        flex: 1;
    }
    
    .track-lang {
        font-size: 0.8rem;
        color: rgba(255, 255, 255, 0.5);
    }
    
    /* ========================================
       RESPONSIVE
       ======================================== */
    
    @media (max-width: 768px) {
        .top-overlay {
            padding: 16px;
        }
        
        .btn-back {
            width: 40px;
            height: 40px;
        }
        
        .anime-title {
            font-size: 1rem;
        }
        
        .bottom-controls {
            padding: 0 16px 16px;
        }
        
        .ctrl-btn {
            width: 40px;
            height: 40px;
        }
        
        .ctrl-btn.play-pause {
            width: 48px;
            height: 48px;
        }
        
        .skip-btn {
            bottom: 100px;
            right: 16px;
            padding: 12px 18px;
            font-size: 0.85rem;
        }
        
        .big-play-btn {
            width: 70px;
            height: 70px;
        }
        
        .big-play-btn svg {
            width: 32px;
            height: 32px;
        }
    }
</style>
