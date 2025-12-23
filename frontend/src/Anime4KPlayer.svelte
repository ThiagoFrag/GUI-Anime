<script>
    import { onMount, onDestroy } from 'svelte';
    import { SetFullscreen, GetSettings, SaveSettings } from '../wailsjs/go/main/App.js';
    
    // Props
    export let src = "";
    export let title = "";
    export let episodeTitle = "";
    export let onClose = () => {};
    export let onNext = () => {};
    export let onPrevious = () => {};
    
    // Upscale settings - disabled by default for stability
    let upscaleEnabled = false;
    let upscaleStrength = 1.0; // 0.0 - 2.0
    let showSettings = false;
    
    // Elements
    let videoEl;
    let canvasEl;
    let containerEl;
    let gl;
    let animationId;
    let program;
    let texture;
    
    // Video state
    let isPlaying = false;
    let currentTime = 0;
    let duration = 0;
    let volume = 1;
    let isMuted = false;
    
    // Fill mode for video display
    let fillMode = 'contain'; // 'contain' or 'cover'
    let currentSettings = null;
    
    // Error handling
    let loadTimeout = null;
    let retryCount = 0;
    const MAX_RETRIES = 2;
    let isFullscreen = false;

    // Detect Google Video URLs (need special handling - no CORS)
    $: isGoogleVideo = src && (src.includes("googlevideo.com") || src.includes("googleusercontent.com"));
    
    // Frame counter for debug logging
    let frameCounter = 0;
    let showControls = true;
    let controlsTimeout;
    let isLoading = true;
    let error = null;
    
    // Anime4K Shader - Versão otimizada para WebGL
    // Baseado no algoritmo original Anime4K (Mode A - Fast)
    const vertexShaderSource = `
        attribute vec2 a_position;
        attribute vec2 a_texCoord;
        varying vec2 v_texCoord;
        void main() {
            gl_Position = vec4(a_position, 0.0, 1.0);
            v_texCoord = a_texCoord;
        }
    `;
    
    // Anime4K Restore Shader (CNN-based edge enhancement)
    const fragmentShaderSource = `
        precision highp float;
        
        uniform sampler2D u_image;
        uniform vec2 u_resolution;
        uniform float u_strength;
        uniform bool u_enabled;
        
        varying vec2 v_texCoord;
        
        // Luminance calculation (ITU-R BT.709)
        float getLuma(vec3 rgb) {
            return dot(rgb, vec3(0.2126, 0.7152, 0.0722));
        }
        
        // Sobel edge detection
        float edgeDetect(vec2 uv, vec2 pixelSize) {
            float tl = getLuma(texture2D(u_image, uv + vec2(-pixelSize.x, -pixelSize.y)).rgb);
            float t  = getLuma(texture2D(u_image, uv + vec2(0.0, -pixelSize.y)).rgb);
            float tr = getLuma(texture2D(u_image, uv + vec2(pixelSize.x, -pixelSize.y)).rgb);
            float l  = getLuma(texture2D(u_image, uv + vec2(-pixelSize.x, 0.0)).rgb);
            float r  = getLuma(texture2D(u_image, uv + vec2(pixelSize.x, 0.0)).rgb);
            float bl = getLuma(texture2D(u_image, uv + vec2(-pixelSize.x, pixelSize.y)).rgb);
            float b  = getLuma(texture2D(u_image, uv + vec2(0.0, pixelSize.y)).rgb);
            float br = getLuma(texture2D(u_image, uv + vec2(pixelSize.x, pixelSize.y)).rgb);
            
            float gx = -tl - 2.0*l - bl + tr + 2.0*r + br;
            float gy = -tl - 2.0*t - tr + bl + 2.0*b + br;
            
            return sqrt(gx*gx + gy*gy);
        }
        
        // Anime4K Line Reconstruction
        vec3 anime4kRestore(vec2 uv, vec2 pixelSize) {
            vec3 center = texture2D(u_image, uv).rgb;
            
            // Sample neighbors
            vec3 n = texture2D(u_image, uv + vec2(0.0, -pixelSize.y)).rgb;
            vec3 s = texture2D(u_image, uv + vec2(0.0, pixelSize.y)).rgb;
            vec3 e = texture2D(u_image, uv + vec2(pixelSize.x, 0.0)).rgb;
            vec3 w = texture2D(u_image, uv + vec2(-pixelSize.x, 0.0)).rgb;
            
            vec3 ne = texture2D(u_image, uv + vec2(pixelSize.x, -pixelSize.y)).rgb;
            vec3 nw = texture2D(u_image, uv + vec2(-pixelSize.x, -pixelSize.y)).rgb;
            vec3 se = texture2D(u_image, uv + vec2(pixelSize.x, pixelSize.y)).rgb;
            vec3 sw = texture2D(u_image, uv + vec2(-pixelSize.x, pixelSize.y)).rgb;
            
            // Calculate gradients
            float edge = edgeDetect(uv, pixelSize);
            
            // Line thinning / reconstruction based on gradient
            vec3 minC = min(min(min(n, s), min(e, w)), center);
            vec3 maxC = max(max(max(n, s), max(e, w)), center);
            
            // Compute sharpening based on local contrast
            vec3 avg = (n + s + e + w + ne + nw + se + sw) / 8.0;
            vec3 diff = center - avg;
            
            // Apply edge-aware sharpening
            float sharpenAmount = u_strength * edge * 2.0;
            vec3 sharpened = center + diff * sharpenAmount;
            
            // Clamp to prevent ringing artifacts
            sharpened = clamp(sharpened, minC - 0.1, maxC + 0.1);
            
            return sharpened;
        }
        
        // Thin line enhancement for anime
        vec3 enhanceLines(vec2 uv, vec2 pixelSize, vec3 color) {
            float luma = getLuma(color);
            
            // Sample in cross pattern
            float lumaN = getLuma(texture2D(u_image, uv + vec2(0.0, -pixelSize.y)).rgb);
            float lumaS = getLuma(texture2D(u_image, uv + vec2(0.0, pixelSize.y)).rgb);
            float lumaE = getLuma(texture2D(u_image, uv + vec2(pixelSize.x, 0.0)).rgb);
            float lumaW = getLuma(texture2D(u_image, uv + vec2(-pixelSize.x, 0.0)).rgb);
            
            // Detect thin lines (dark pixels surrounded by lighter ones)
            float avgNeighbor = (lumaN + lumaS + lumaE + lumaW) / 4.0;
            float lineFactor = max(0.0, avgNeighbor - luma);
            
            // Darken lines slightly for crispness
            vec3 enhanced = color - vec3(lineFactor * u_strength * 0.3);
            
            return max(enhanced, vec3(0.0));
        }
        
        void main() {
            if (!u_enabled) {
                gl_FragColor = texture2D(u_image, v_texCoord);
                return;
            }
            
            vec2 pixelSize = 1.0 / u_resolution;
            
            // Apply Anime4K restoration
            vec3 restored = anime4kRestore(v_texCoord, pixelSize);
            
            // Enhance thin lines
            vec3 enhanced = enhanceLines(v_texCoord, pixelSize, restored);
            
            // Slight saturation boost for anime colors
            float luma = getLuma(enhanced);
            vec3 saturated = mix(vec3(luma), enhanced, 1.0 + u_strength * 0.15);
            
            gl_FragColor = vec4(saturated, 1.0);
        }
    `;
    
    function initWebGL() {
        if (!canvasEl) return false;
        
        gl = canvasEl.getContext('webgl', { 
            premultipliedAlpha: false,
            antialias: false,
            preserveDrawingBuffer: true
        });
        
        if (!gl) {
            console.warn('[Anime4K] WebGL not supported, falling back to standard video');
            return false;
        }
        
        // Create shaders
        const vertexShader = gl.createShader(gl.VERTEX_SHADER);
        gl.shaderSource(vertexShader, vertexShaderSource);
        gl.compileShader(vertexShader);
        
        if (!gl.getShaderParameter(vertexShader, gl.COMPILE_STATUS)) {
            console.error('[Anime4K] Vertex shader error:', gl.getShaderInfoLog(vertexShader));
            return false;
        }
        
        const fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
        gl.shaderSource(fragmentShader, fragmentShaderSource);
        gl.compileShader(fragmentShader);
        
        if (!gl.getShaderParameter(fragmentShader, gl.COMPILE_STATUS)) {
            console.error('[Anime4K] Fragment shader error:', gl.getShaderInfoLog(fragmentShader));
            return false;
        }
        
        // Create program
        program = gl.createProgram();
        gl.attachShader(program, vertexShader);
        gl.attachShader(program, fragmentShader);
        gl.linkProgram(program);
        
        if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
            console.error('[Anime4K] Program link error:', gl.getProgramInfoLog(program));
            return false;
        }
        
        gl.useProgram(program);
        
        // Set up geometry (fullscreen quad)
        const positionBuffer = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([
            -1, -1,  1, -1,  -1, 1,
            -1,  1,  1, -1,   1, 1
        ]), gl.STATIC_DRAW);
        
        const positionLocation = gl.getAttribLocation(program, 'a_position');
        gl.enableVertexAttribArray(positionLocation);
        gl.vertexAttribPointer(positionLocation, 2, gl.FLOAT, false, 0, 0);
        
        // Set up texture coordinates
        const texCoordBuffer = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, texCoordBuffer);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([
            0, 1,  1, 1,  0, 0,
            0, 0,  1, 1,  1, 0
        ]), gl.STATIC_DRAW);
        
        const texCoordLocation = gl.getAttribLocation(program, 'a_texCoord');
        gl.enableVertexAttribArray(texCoordLocation);
        gl.vertexAttribPointer(texCoordLocation, 2, gl.FLOAT, false, 0, 0);
        
        // Create texture for video frames
        texture = gl.createTexture();
        gl.bindTexture(gl.TEXTURE_2D, texture);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
        
        console.log('[Anime4K] WebGL initialized successfully');
        return true;
    }
    
    function renderFrame() {
        // Skip if not ready or upscale disabled
        if (!upscaleEnabled || !gl || !videoEl || !canvasEl) {
            if (isPlaying && upscaleEnabled) {
                animationId = requestAnimationFrame(renderFrame);
            }
            return;
        }
        
        // Skip if video not playing
        if (videoEl.paused || videoEl.ended || videoEl.readyState < 2) {
            if (isPlaying) {
                animationId = requestAnimationFrame(renderFrame);
            }
            return;
        }

        // Simple frame counter for debug (less frequent)
        frameCounter++;
        if (frameCounter % 120 === 0) {
            console.debug('[Anime4K] frame=' + frameCounter);
        }

        // Update canvas size depending on fill mode
        try {
            const dpr = window.devicePixelRatio || 1;
            if (fillMode === 'cover' && containerEl) {
                // For 'cover' we want the canvas internal resolution to match the container display size
                const cw = Math.max(1, Math.floor(containerEl.clientWidth * dpr));
                const ch = Math.max(1, Math.floor(containerEl.clientHeight * dpr));
                if (canvasEl.width !== cw || canvasEl.height !== ch) {
                    canvasEl.width = cw;
                    canvasEl.height = ch;
                    gl.viewport(0, 0, canvasEl.width, canvasEl.height);
                }
            } else {
                // Default: match video native resolution for best quality
                const vw = Math.max(1, Math.floor(videoEl.videoWidth));
                const vh = Math.max(1, Math.floor(videoEl.videoHeight));
                if (canvasEl.width !== vw || canvasEl.height !== vh) {
                    canvasEl.width = vw;
                    canvasEl.height = vh;
                    gl.viewport(0, 0, canvasEl.width, canvasEl.height);
                }
            }
        } catch (e) {
            // Fallback to video resolution if anything goes wrong
            if (canvasEl.width !== videoEl.videoWidth || canvasEl.height !== videoEl.videoHeight) {
                canvasEl.width = videoEl.videoWidth;
                canvasEl.height = videoEl.videoHeight;
                gl.viewport(0, 0, canvasEl.width, canvasEl.height);
            }
        }
        
        // Upload video frame to texture
        gl.bindTexture(gl.TEXTURE_2D, texture);
        gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, videoEl);
        
        // Set uniforms
        const resolutionLocation = gl.getUniformLocation(program, 'u_resolution');
        gl.uniform2f(resolutionLocation, videoEl.videoWidth, videoEl.videoHeight);
        
        const strengthLocation = gl.getUniformLocation(program, 'u_strength');
        gl.uniform1f(strengthLocation, upscaleStrength);
        
        const enabledLocation = gl.getUniformLocation(program, 'u_enabled');
        gl.uniform1i(enabledLocation, upscaleEnabled ? 1 : 0);
        
        // Draw
        gl.drawArrays(gl.TRIANGLES, 0, 6);
        
        animationId = requestAnimationFrame(renderFrame);
    }
    
    function startRendering() {
        if (!animationId) {
            renderFrame();
        }
    }
    
    function stopRendering() {
        if (animationId) {
            cancelAnimationFrame(animationId);
            animationId = null;
        }
    }
    
    // Video controls
    function togglePlay() {
        if (!videoEl) return;
        if (videoEl.paused) {
            videoEl.play().catch(e => {
                console.error('[Anime4K] Play error:', e);
                error = 'Erro ao reproduzir o vídeo';
            });
        } else {
            videoEl.pause();
        }
    }
    
    function seek(e) {
        if (!videoEl || !duration) return;
        const rect = e.currentTarget.getBoundingClientRect();
        const percent = (e.clientX - rect.left) / rect.width;
        videoEl.currentTime = percent * duration;
    }
    
    function setVolume(e) {
        if (!videoEl) return;
        const rect = e.currentTarget.getBoundingClientRect();
        volume = (e.clientX - rect.left) / rect.width;
        videoEl.volume = Math.max(0, Math.min(1, volume));
        isMuted = volume === 0;
    }
    
    function toggleMute() {
        if (!videoEl) return;
        isMuted = !isMuted;
        videoEl.muted = isMuted;
    }
    
    async function toggleFullscreen() {
        try {
            // Usa a API do Wails para controlar a janela
            isFullscreen = !isFullscreen;
            await SetFullscreen(isFullscreen);
            console.log('[Anime4K] Fullscreen:', isFullscreen);
        } catch (err) {
            console.error('[Anime4K] Wails fullscreen error:', err);
            // Fallback: tenta a API padrão do navegador
            try {
                if (!document.fullscreenElement) {
                    const element = containerEl || document.querySelector('.player-container');
                    if (element && element.requestFullscreen) {
                        await element.requestFullscreen();
                        isFullscreen = true;
                    }
                } else {
                    await document.exitFullscreen();
                    isFullscreen = false;
                }
            } catch (e) {
                console.error('[Anime4K] Browser fullscreen fallback error:', e);
            }
        }
    }

    async function setFillMode(mode) {
        fillMode = mode === 'cover' ? 'cover' : 'contain';
        try {
            const s = currentSettings || (await GetSettings());
            // try to set multiple casing variants depending on bindings
            s.player_fill_mode = fillMode;
            s.PlayerFillMode = fillMode;
            s.playerFillMode = fillMode;
            await SaveSettings(s);
            currentSettings = s;
        } catch (e) {
            console.warn('[Anime4K] Failed to save settings:', e);
        }
    }
    
    function formatTime(seconds) {
        if (isNaN(seconds)) return '0:00';
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }
    
    function showControlsTemporarily() {
        showControls = true;
        clearTimeout(controlsTimeout);
        controlsTimeout = setTimeout(() => {
            if (isPlaying) showControls = false;
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
                if (videoEl) videoEl.currentTime = Math.max(0, (videoEl.currentTime || 0) - 10);
                break;
            case 'ArrowRight':
                if (videoEl) videoEl.currentTime = (videoEl.currentTime || 0) + 10;
                break;
            case 'ArrowUp':
                volume = Math.min(1, volume + 0.1);
                if (videoEl) videoEl.volume = volume;
                break;
            case 'ArrowDown':
                volume = Math.max(0, volume - 0.1);
                if (videoEl) videoEl.volume = volume;
                break;
            case 'f':
                toggleFullscreen();
                break;
            case 'm':
                toggleMute();
                break;
            case 'Escape':
                if (isFullscreen) {
                    document.exitFullscreen();
                } else {
                    onClose();
                }
                break;
        }
    }
    
    // Function to enable/disable upscaling
    function toggleUpscale(enabled) {
        upscaleEnabled = enabled;
        if (enabled && !gl && canvasEl) {
            // Initialize WebGL when first enabling
            const success = initWebGL();
            console.log('[Anime4K] WebGL initialized on enable:', success);
            if (success && isPlaying) {
                startRendering();
            }
        } else if (!enabled) {
            stopRendering();
        }
    }
    
    onMount(() => {
        // Don't initialize WebGL until upscale is enabled (starts disabled for stability)
        console.log('[Anime4K] Player mounted, upscale:', upscaleEnabled);
        
        if (videoEl) {
            videoEl.addEventListener('loadedmetadata', () => {
                if (videoEl) {
                    duration = videoEl.duration || 0;
                    isLoading = false;
                    // Cancela timeout de carregamento
                    if (loadTimeout) {
                        clearTimeout(loadTimeout);
                        loadTimeout = null;
                    }
                }
            });
            
            videoEl.addEventListener('timeupdate', () => {
                if (videoEl) {
                    currentTime = videoEl.currentTime || 0;
                }
            });
            
            videoEl.addEventListener('play', () => {
                isPlaying = true;
                // Only start WebGL rendering if gl is available
                if (gl && upscaleEnabled) {
                    startRendering();
                    console.log('[Anime4K] Started WebGL rendering');
                } else {
                    console.log('[Anime4K] Playing without WebGL (gl:', !!gl, 'upscale:', upscaleEnabled, ')');
                }
            });
            
            videoEl.addEventListener('pause', () => {
                isPlaying = false;
            });
            
            videoEl.addEventListener('ended', () => {
                isPlaying = false;
                stopRendering();
            });
            
            videoEl.addEventListener('error', (e) => {
                console.error('[Anime4K] Video error:', e);
                // Cancela timeout
                if (loadTimeout) {
                    clearTimeout(loadTimeout);
                    loadTimeout = null;
                }
                // Tentar recuperar informações do erro
                let errorMessage = 'Erro ao carregar o vídeo';
                if (videoEl && videoEl.error) {
                    switch (videoEl.error.code) {
                        case 1: errorMessage = 'Carregamento do vídeo abortado'; break;
                        case 2: errorMessage = 'Erro de rede ao carregar vídeo'; break;
                        case 3: errorMessage = 'Erro ao decodificar o vídeo'; break;
                        case 4: errorMessage = 'Formato de vídeo não suportado'; break;
                    }
                }
                error = errorMessage;
                isLoading = false;
            });
            
            videoEl.addEventListener('waiting', () => {
                isLoading = true;
            });
            
            videoEl.addEventListener('canplay', () => {
                isLoading = false;
                error = null; // Limpa erro se conseguir carregar
                retryCount = 0; // Reset retry count on success
                // Cancela timeout
                if (loadTimeout) {
                    clearTimeout(loadTimeout);
                    loadTimeout = null;
                }
            });
            
            // Timeout de 15 segundos para carregamento inicial
            loadTimeout = setTimeout(() => {
                if (isLoading && !error) {
                    console.warn('[Anime4K] Timeout ao carregar vídeo');
                    error = 'Tempo esgotado ao carregar vídeo. O servidor pode estar lento ou indisponível.';
                    isLoading = false;
                }
            }, 15000);
        }
        
        document.addEventListener('keydown', handleKeydown);
        document.addEventListener('fullscreenchange', () => {
            isFullscreen = !!document.fullscreenElement;
        });
    });
    
    onDestroy(() => {
        stopRendering();
        if (gl) {
            gl.deleteTexture(texture);
            gl.deleteProgram(program);
        }
        document.removeEventListener('keydown', handleKeydown);
        clearTimeout(controlsTimeout);
        // Limpa timeout de carregamento
        if (loadTimeout) {
            clearTimeout(loadTimeout);
            loadTimeout = null;
        }
        // Sai da tela cheia ao destruir o player
        if (isFullscreen) {
            SetFullscreen(false).catch(() => {});
        }
    });
    
    // Reactive: restart rendering when settings change
    $: if (gl && isPlaying) {
        // Settings changed, just continue rendering with new values
    }
</script>

<div 
    class="player-container" 
    class:fullscreen={isFullscreen}
    bind:this={containerEl}
    onmousemove={showControlsTemporarily}
    role="application"
    aria-label="Video Player with Anime4K upscaling"
>
    <!-- Header -->
    <div class="player-header" class:visible={showControls || !isPlaying}>
        <div class="header-info">
            <strong>{title || 'Reproduzindo...'}</strong>
            <span>{episodeTitle}</span>
        </div>
        <div class="header-actions">
            <button type="button" class="btn-settings" onclick={() => showSettings = !showSettings} title="Configurações Anime4K">
                ⚙️ Anime4K
            </button>
            <button type="button" class="btn-close" onclick={onClose}>✕ Fechar</button>
        </div>
    </div>
    
    <!-- Settings Panel -->
    {#if showSettings}
        <div class="settings-panel">
            <h3>🎨 Configurações Anime4K</h3>
            
            <div class="setting-row">
                <label>
                    <input type="checkbox" checked={upscaleEnabled} onchange={(e) => toggleUpscale(e.target.checked)} />
                    Ativar Upscaling Anime4K (experimental)
                </label>
                <p class="hint">⚠️ Pode causar lentidão em alguns dispositivos</p>
            </div>
            
            <div class="setting-row">
                <label>
                    Intensidade: {upscaleStrength.toFixed(1)}
                    <input 
                        type="range" 
                        min="0" 
                        max="2" 
                        step="0.1" 
                        bind:value={upscaleStrength}
                        disabled={!upscaleEnabled}
                    />
                </label>
                <span class="hint">
                    {upscaleStrength < 0.5 ? 'Sutil' : upscaleStrength < 1.2 ? 'Equilibrado' : 'Intenso'}
                </span>
            </div>
            
            <div class="settings-info">
                <p>✨ <strong>Anime4K</strong> melhora linhas finas e cores em tempo real usando WebGL.</p>
                <p>🎯 Funciona melhor com anime 720p ou 480p → exibido em tela cheia.</p>
            </div>
            
            <div class="setting-row">
                <span class="setting-label">Modo de tela:</span>
                <div class="toggle-group">
                    <button type="button" class:active={fillMode==='contain'} onclick={() => setFillMode('contain')} title="Ajustar (sem cortes)">Ajustar</button>
                    <button type="button" class:active={fillMode==='cover'} onclick={() => setFillMode('cover')} title="Preencher (pode recortar)">Preencher</button>
                </div>
            </div>

            <button type="button" class="btn-close-settings" onclick={() => showSettings = false}>
                Fechar
            </button>
        </div>
    {/if}
    
    <!-- Video Container -->
    <div class="video-container">
        <!-- Video element - always visible when upscale disabled -->
        <video
            bind:this={videoEl}
            src={src}
            crossorigin={isGoogleVideo ? undefined : "anonymous"}
            playsinline
            autoplay
            class="player-video"
            class:video-hidden={upscaleEnabled && gl}
            class:fit-contain={fillMode==='contain'}
            class:fit-cover={fillMode==='cover'}
            onclick={togglePlay}
        >
            <track kind="captions" />
        </video>
        
        <!-- Canvas with Anime4K shader output - only when enabled -->
        {#if upscaleEnabled && gl}
            <canvas 
                bind:this={canvasEl} 
                class="upscaled-canvas"
                class:fit-contain={fillMode==='contain'}
                class:fit-cover={fillMode==='cover'}
                onclick={togglePlay}
            ></canvas>
        {/if}
        
        <!-- Loading Spinner -->
        {#if isLoading}
            <div class="loading-overlay">
                <div class="spinner"></div>
                <p>Carregando vídeo...</p>
            </div>
        {/if}
        
        <!-- Error Message -->
        {#if error}
            <div class="error-overlay">
                <div class="error-icon">⚠️</div>
                <p class="error-title">{error}</p>
                <p class="error-hint">O vídeo pode estar temporariamente indisponível ou o formato não é suportado.</p>
                <div class="error-actions">
                    <button type="button" class="btn-retry" onclick={() => { error = null; isLoading = true; videoEl?.load(); }}>
                        🔄 Tentar Novamente
                    </button>
                    <button type="button" class="btn-close-error" onclick={onClose}>
                        ← Voltar
                    </button>
                </div>
            </div>
        {/if}
        
        <!-- Play/Pause Indicator -->
        {#if !isPlaying && !isLoading && !error}
                <div class="play-indicator" role="button" tabindex="0" onclick={togglePlay} onkeydown={(e) => (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') && togglePlay()}>
                <span class="play-icon">▶</span>
            </div>
        {/if}
        
        <!-- Anime4K Badge -->
        {#if upscaleEnabled && gl}
            <div class="anime4k-badge">
                <span>✨ Anime4K</span>
            </div>
        {/if}
    </div>
    
    <!-- Controls Bar -->
    <div class="controls-bar" class:visible={showControls || !isPlaying}>
        <!-- Progress Bar -->
        <div class="progress-container" onclick={seek} role="slider" aria-label="Video progress" tabindex="0" onkeydown={(e) => (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') && seek(e)}>
                <div class="progress-bar" tabindex="0" onkeydown={(e) => (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') && seek(e)}>
                <div class="progress-filled" style="width: {(currentTime / duration) * 100}%"></div>
                <div class="progress-handle" style="left: {(currentTime / duration) * 100}%"></div>
            </div>
        </div>
        
        <div class="controls-main">
            <!-- Left controls -->
            <div class="controls-left">
                <button type="button" class="btn-nav" onclick={onPrevious} title="Episódio anterior">
                    ⏮
                </button>
                <button type="button" class="btn-play" onclick={togglePlay} title={isPlaying ? 'Pausar' : 'Reproduzir'}>
                    {isPlaying ? '⏸' : '▶'}
                </button>
                <button type="button" class="btn-nav" onclick={onNext} title="Próximo episódio">
                    ⏭
                </button>
                
                <div class="volume-control">
                    <button type="button" class="btn-volume" onclick={toggleMute} title={isMuted ? 'Ativar som' : 'Mudo'}>
                        {isMuted || volume === 0 ? '🔇' : volume < 0.5 ? '🔉' : '🔊'}
                    </button>
                    <div class="volume-slider" onclick={setVolume} role="slider" aria-label="Volume" tabindex="0" onkeydown={(e) => (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') && setVolume(e)}>
                            <div class="volume-bar" tabindex="0" onkeydown={(e) => (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') && setVolume(e)}>
                            <div class="volume-filled" style="width: {isMuted ? 0 : volume * 100}%"></div>
                        </div>
                    </div>
                </div>
                
                <span class="time-display">
                    {formatTime(currentTime)} / {formatTime(duration)}
                </span>
            </div>
            
            <!-- Right controls -->
            <div class="controls-right">
                <button type="button" class="btn-quality" onclick={() => showSettings = !showSettings} title="Anime4K Settings">
                    {upscaleEnabled ? '✨ 4K' : '📺 SD'}
                </button>
                <button type="button" class="btn-fullscreen" onclick={toggleFullscreen} title={isFullscreen ? 'Sair da tela cheia' : 'Tela cheia'}>
                    {isFullscreen ? '⛶' : '⛶'}
                </button>
            </div>
        </div>
    </div>
</div>

<style>
    .player-container {
        position: fixed;
        inset: 0;
        background: #000;
        display: flex;
        flex-direction: column;
        z-index: 2000;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    }
    
    .player-container.fullscreen {
        /* Already fullscreen via Fullscreen API */
    }
    
    /* Header */
    .player-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 15px 20px;
        background: linear-gradient(to bottom, rgba(0,0,0,0.9) 0%, transparent 100%);
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        z-index: 10;
        opacity: 0;
        transition: opacity 0.3s;
    }
    
    .player-header.visible {
        opacity: 1;
    }
    
    .header-info {
        display: flex;
        flex-direction: column;
    }
    
    .header-info strong {
        color: #f5576c;
        font-size: 1.1rem;
    }
    
    .header-info span {
        color: #aaa;
        font-size: 0.9rem;
    }
    
    .header-actions {
        display: flex;
        gap: 10px;
    }
    
    .btn-settings, .btn-close {
        padding: 8px 16px;
        border-radius: 6px;
        border: 1px solid #555;
        background: rgba(0,0,0,0.6);
        color: #fff;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-settings:hover {
        background: rgba(245, 87, 108, 0.3);
        border-color: #f5576c;
    }
    
    .btn-close:hover {
        background: #f5576c;
    }
    
    /* Settings Panel */
    .settings-panel {
        position: absolute;
        top: 70px;
        right: 20px;
        width: 320px;
        background: rgba(20, 21, 30, 0.95);
        border: 1px solid #444;
        border-radius: 12px;
        padding: 20px;
        z-index: 20;
        backdrop-filter: blur(10px);
    }
    
    .settings-panel h3 {
        margin: 0 0 15px 0;
        color: #f5576c;
        font-size: 1.1rem;
    }
    
    .setting-row {
        margin-bottom: 15px;
    }
    
    .setting-row label,
    .setting-row .setting-label {
        display: flex;
        flex-direction: column;
        gap: 8px;
        color: #ddd;
        font-size: 0.95rem;
    }
    
    .setting-row input[type="checkbox"] {
        width: 18px;
        height: 18px;
        margin-right: 8px;
    }
    
    .setting-row input[type="range"] {
        width: 100%;
        accent-color: #f5576c;
    }
    
    .hint {
        font-size: 0.8rem;
        color: #888;
        text-align: right;
    }
    
    .settings-info {
        background: rgba(245, 87, 108, 0.1);
        border-radius: 8px;
        padding: 12px;
        margin-top: 15px;
    }
    
    .settings-info p {
        margin: 5px 0;
        font-size: 0.85rem;
        color: #bbb;
    }
    
    .btn-close-settings {
        width: 100%;
        padding: 10px;
        margin-top: 15px;
        background: #333;
        border: 1px solid #555;
        border-radius: 6px;
        color: #fff;
        cursor: pointer;
    }
    
    .btn-close-settings:hover {
        background: #444;
    }
    
    /* Video Container */
    .video-container {
        flex: 1;
        position: relative;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        background: #000;
        min-height: 200px;
        height: 100%;
    }
    
    /* Player video - main video element */
    .player-video {
        position: absolute;
        width: 100%;
        height: 100%;
        display: block;
        cursor: pointer;
        z-index: 1;
        background: #000;
    }
    .player-video.fit-contain { object-fit: contain; }
    .player-video.fit-cover { object-fit: cover; }
    
    /* Hide video when canvas is rendering */
    .player-video.video-hidden {
        opacity: 0;
        pointer-events: none;
    }

    /* Canvas for WebGL Anime4K output */
    .upscaled-canvas {
        position: absolute;
        cursor: pointer;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        display: block;
        z-index: 2;
        background: transparent;
    }
    .upscaled-canvas.fit-contain { object-fit: contain; }
    .upscaled-canvas.fit-cover { object-fit: cover; }

    .toggle-group { display: inline-flex; gap: 8px; margin-left: 10px; }
    .toggle-group > button { padding: 6px 12px; border: 1px solid #555; background: #222; color: #fff; border-radius: 6px; cursor: pointer; }
    .toggle-group > button.active { background: #f5576c; border-color: #f5576c; }
    
    /* Overlays */
    .loading-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        background: rgba(0,0,0,0.8);
        color: #fff;
    }
    
    .error-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        background: rgba(0,0,0,0.9);
        color: #fff;
        padding: 20px;
        text-align: center;
    }
    
    .error-icon {
        font-size: 4rem;
        margin-bottom: 15px;
    }
    
    .error-title {
        font-size: 1.3rem;
        font-weight: 600;
        color: #ff6b6b;
        margin-bottom: 10px;
    }
    
    .error-hint {
        font-size: 0.9rem;
        color: #aaa;
        max-width: 400px;
        margin-bottom: 20px;
    }
    
    .error-actions {
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
        justify-content: center;
    }
    
    .btn-retry {
        padding: 12px 25px;
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        border: none;
        border-radius: 8px;
        color: #fff;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-retry:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 20px rgba(245, 87, 108, 0.4);
    }
    
    .btn-close-error {
        padding: 12px 25px;
        background: rgba(255,255,255,0.1);
        border: 1px solid rgba(255,255,255,0.2);
        border-radius: 8px;
        color: #fff;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-close-error:hover {
        background: rgba(255,255,255,0.2);
    }
    
    .spinner {
        width: 50px;
        height: 50px;
        border: 4px solid rgba(255,255,255,0.2);
        border-top-color: #f5576c;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    .play-indicator {
        position: absolute;
        width: 80px;
        height: 80px;
        background: rgba(245, 87, 108, 0.8);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: transform 0.2s;
    }
    
    .play-indicator:hover {
        transform: scale(1.1);
    }
    
    .play-icon {
        font-size: 2.5rem;
        color: #fff;
        margin-left: 5px;
    }
    
    .anime4k-badge {
        position: absolute;
        top: 80px;
        left: 20px;
        padding: 6px 12px;
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        border-radius: 20px;
        font-size: 0.8rem;
        font-weight: bold;
        color: #fff;
        pointer-events: none;
    }
    
    /* Controls Bar */
    .controls-bar {
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        background: linear-gradient(to top, rgba(0,0,0,0.9) 0%, transparent 100%);
        padding: 40px 20px 15px;
        opacity: 0;
        transition: opacity 0.3s;
    }
    
    .controls-bar.visible {
        opacity: 1;
    }
    
    /* Progress Bar */
    .progress-container {
        cursor: pointer;
        padding: 10px 0;
    }
    
    .progress-bar {
        height: 5px;
        background: rgba(255,255,255,0.2);
        border-radius: 3px;
        position: relative;
    }
    
    .progress-filled {
        height: 100%;
        background: linear-gradient(90deg, #f093fb 0%, #f5576c 100%);
        border-radius: 3px;
        transition: width 0.1s;
    }
    
    .progress-handle {
        position: absolute;
        top: 50%;
        transform: translate(-50%, -50%) scale(0);
        width: 14px;
        height: 14px;
        background: #f5576c;
        border-radius: 50%;
        transition: transform 0.2s;
    }
    
    .progress-container:hover .progress-handle {
        transform: translate(-50%, -50%) scale(1);
    }
    
    /* Main Controls */
    .controls-main {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-top: 10px;
    }
    
    .controls-left, .controls-right {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    
    .btn-play, .btn-nav, .btn-volume, .btn-quality, .btn-fullscreen {
        width: 40px;
        height: 40px;
        border: none;
        background: transparent;
        color: #fff;
        font-size: 1.2rem;
        cursor: pointer;
        border-radius: 50%;
        transition: background 0.2s;
    }
    
    .btn-play {
        width: 50px;
        height: 50px;
        font-size: 1.5rem;
        background: rgba(255,255,255,0.1);
    }
    
    .btn-play:hover, .btn-nav:hover, .btn-volume:hover, .btn-fullscreen:hover {
        background: rgba(255,255,255,0.2);
    }
    
    .btn-quality {
        width: auto;
        padding: 8px 15px;
        font-size: 0.9rem;
        border-radius: 6px;
        background: rgba(245, 87, 108, 0.2);
        border: 1px solid rgba(245, 87, 108, 0.5);
    }
    
    .btn-quality:hover {
        background: rgba(245, 87, 108, 0.4);
    }
    
    /* Volume Control */
    .volume-control {
        display: flex;
        align-items: center;
        gap: 5px;
    }
    
    .volume-slider {
        width: 80px;
        cursor: pointer;
        padding: 10px 0;
    }
    
    .volume-bar {
        height: 4px;
        background: rgba(255,255,255,0.2);
        border-radius: 2px;
    }
    
    .volume-filled {
        height: 100%;
        background: #fff;
        border-radius: 2px;
    }
    
    .time-display {
        color: #aaa;
        font-size: 0.9rem;
        margin-left: 10px;
    }
    
    /* Responsive */
    @media (max-width: 600px) {
        .settings-panel {
            left: 10px;
            right: 10px;
            width: auto;
        }
        
        .volume-slider {
            display: none;
        }
        
        .time-display {
            font-size: 0.8rem;
        }
        
        /* Removed unused .btn-quality span rule to avoid Svelte warning */
    }
</style>
