<script>
    import { onMount } from 'svelte';
    import { GetPlayer4KModes, PlayWithPlayer4K, StopPlayer4K, IsPlayer4KAvailable, GetStreamURLForEpisode } from '../wailsjs/go/main/App.js';

    // Props
    export let videoUrl = "";
    export let animeUrl = "";
    export let episodeUrl = "";
    export let isAnime = true;

    // State
    let available = false;
    let modes = [];
    let selectedMode = "medium";
    let useAnimeShaders = true;
    let showModeSelector = false;
    let isPlaying = false;
    let isLoading = false;
    let error = null;

    onMount(async () => {
        try {
            available = await IsPlayer4KAvailable();
            if (available) {
                modes = await GetPlayer4KModes();
            }
        } catch (e) {
            console.error('[Player4K] Erro ao verificar disponibilidade:', e);
            available = false;
        }
    });

    async function playWithUpscaling() {
        let urlToPlay = videoUrl;

        // Se não tem URL direta, tenta buscar do episódio
        if (!urlToPlay && episodeUrl && animeUrl) {
            try {
                isLoading = true;
                error = null;
                console.log('[Player4K] Buscando URL do stream...');
                urlToPlay = await GetStreamURLForEpisode(animeUrl, episodeUrl);
            } catch (e) {
                error = "Erro ao obter URL do vídeo: " + (e.message || e);
                isLoading = false;
                return;
            }
        }

        if (!urlToPlay) {
            error = "Nenhum vídeo selecionado";
            isLoading = false;
            return;
        }

        try {
            error = null;
            isLoading = true;
            console.log('[Player4K] Iniciando com URL:', urlToPlay);
            await PlayWithPlayer4K(urlToPlay, selectedMode, useAnimeShaders && isAnime);
            showModeSelector = false;
            isPlaying = true;
        } catch (e) {
            error = e.message || "Erro ao iniciar player";
            isPlaying = false;
        } finally {
            isLoading = false;
        }
    }

    async function stopPlayer() {
        try {
            await StopPlayer4K();
            isPlaying = false;
        } catch (e) {
            console.error('[Player4K] Erro ao parar:', e);
        }
    }

    function toggleModeSelector() {
        showModeSelector = !showModeSelector;
    }

    function selectMode(mode) {
        selectedMode = mode;
    }

    function getModeIcon(modeId) {
        const mode = modes.find(m => m.id === modeId);
        return mode?.icon || '🎬';
    }

    function getModeName(modeId) {
        const mode = modes.find(m => m.id === modeId);
        return mode?.name || 'Equilibrado';
    }
</script>

{#if available}
    <div class="player4k-container">
        <!-- Botão Principal -->
        <div class="player4k-button-group">
            <button 
                class="player4k-btn"
                class:playing={isPlaying}
                class:loading={isLoading}
                onclick={isPlaying ? stopPlayer : playWithUpscaling}
                disabled={isLoading}
                title={isPlaying ? "Parar Player4K" : "Reproduzir com Upscaling AI"}
            >
                <span class="btn-icon">
                    {#if isLoading}
                        ⏳
                    {:else if isPlaying}
                        ⏹️
                    {:else}
                        🎬
                    {/if}
                </span>
                <span class="btn-text">
                    {#if isLoading}
                        Carregando...
                    {:else if isPlaying}
                        Parar 4K
                    {:else}
                        Player 4K
                    {/if}
                </span>
            </button>

            <!-- Botão de Configurações -->
            <button 
                class="player4k-settings-btn"
                onclick={toggleModeSelector}
                title="Configurar qualidade"
            >
                ⚙️
            </button>
        </div>

        <!-- Seletor de Modo (Dropdown) -->
        {#if showModeSelector}
            <div class="mode-selector">
                <div class="mode-header">
                    <span class="mode-title">🎮 Modo de Qualidade</span>
                    <button class="close-btn" onclick={() => showModeSelector = false}>✕</button>
                </div>

                <div class="mode-options">
                    {#each modes as mode}
                        <button 
                            class="mode-option"
                            class:selected={selectedMode === mode.id}
                            onclick={() => selectMode(mode.id)}
                        >
                            <span class="mode-icon">{mode.icon}</span>
                            <div class="mode-info">
                                <span class="mode-name">{mode.name}</span>
                                <span class="mode-desc">{mode.description}</span>
                                <span class="mode-gpu">GPU: {mode.gpuRequired}</span>
                            </div>
                            {#if selectedMode === mode.id}
                                <span class="check">✓</span>
                            {/if}
                        </button>
                    {/each}
                </div>

                <!-- Toggle Anime4K -->
                {#if isAnime}
                    <div class="anime-toggle">
                        <label class="toggle-label">
                            <input 
                                type="checkbox" 
                                bind:checked={useAnimeShaders}
                            />
                            <span class="toggle-text">🎌 Shaders Anime4K</span>
                            <span class="toggle-hint">Otimizado para anime</span>
                        </label>
                    </div>
                {/if}

                <!-- Botão Play no Dropdown -->
                <button class="play-btn" onclick={playWithUpscaling}>
                    <span>▶ Reproduzir com {getModeName(selectedMode)}</span>
                </button>
            </div>
        {/if}

        <!-- Erro -->
        {#if error}
            <div class="player4k-error">
                ⚠️ {error}
            </div>
        {/if}
    </div>
{/if}

<style>
    .player4k-container {
        position: relative;
        display: inline-block;
    }

    .player4k-button-group {
        display: flex;
        gap: 4px;
    }

    .player4k-btn {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        border: none;
        border-radius: 8px 0 0 8px;
        color: white;
        font-weight: 600;
        font-size: 14px;
        cursor: pointer;
        transition: all 0.2s ease;
        box-shadow: 0 2px 10px rgba(99, 102, 241, 0.3);
    }

    .player4k-btn:hover {
        background: linear-gradient(135deg, #5558e3, #7c4fe8);
        transform: translateY(-1px);
        box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);
    }

    .player4k-btn.playing {
        background: linear-gradient(135deg, #ef4444, #f97316);
        box-shadow: 0 2px 10px rgba(239, 68, 68, 0.3);
    }

    .player4k-btn.playing:hover {
        background: linear-gradient(135deg, #dc2626, #ea580c);
        box-shadow: 0 4px 15px rgba(239, 68, 68, 0.4);
    }

    .player4k-btn.loading {
        background: linear-gradient(135deg, #f59e0b, #d97706);
        cursor: wait;
        opacity: 0.9;
    }

    .player4k-btn:disabled {
        cursor: wait;
        transform: none;
    }

    .btn-icon {
        font-size: 18px;
    }

    .player4k-settings-btn {
        padding: 10px 12px;
        background: linear-gradient(135deg, #4f46e5, #7c3aed);
        border: none;
        border-radius: 0 8px 8px 0;
        color: white;
        font-size: 16px;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .player4k-settings-btn:hover {
        background: linear-gradient(135deg, #4338ca, #6d28d9);
    }

    .mode-selector {
        position: absolute;
        top: 100%;
        right: 0;
        margin-top: 8px;
        background: #1e1e2e;
        border-radius: 12px;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
        min-width: 320px;
        z-index: 1000;
        overflow: hidden;
        border: 1px solid rgba(255, 255, 255, 0.1);
    }

    .mode-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        background: rgba(255, 255, 255, 0.03);
    }

    .mode-title {
        font-weight: 600;
        color: #fff;
    }

    .close-btn {
        background: none;
        border: none;
        color: #888;
        font-size: 18px;
        cursor: pointer;
        padding: 4px 8px;
        border-radius: 4px;
    }

    .close-btn:hover {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
    }

    .mode-options {
        padding: 8px;
    }

    .mode-option {
        display: flex;
        align-items: center;
        gap: 12px;
        width: 100%;
        padding: 12px;
        background: transparent;
        border: 2px solid transparent;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s ease;
        text-align: left;
        color: #ccc;
    }

    .mode-option:hover {
        background: rgba(99, 102, 241, 0.1);
        border-color: rgba(99, 102, 241, 0.3);
    }

    .mode-option.selected {
        background: rgba(99, 102, 241, 0.15);
        border-color: #6366f1;
    }

    .mode-icon {
        font-size: 28px;
        width: 40px;
        text-align: center;
    }

    .mode-info {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .mode-name {
        font-weight: 600;
        color: #fff;
        font-size: 14px;
    }

    .mode-desc {
        font-size: 12px;
        color: #888;
    }

    .mode-gpu {
        font-size: 11px;
        color: #666;
    }

    .check {
        color: #6366f1;
        font-weight: bold;
        font-size: 18px;
    }

    .anime-toggle {
        padding: 12px 16px;
        border-top: 1px solid rgba(255, 255, 255, 0.1);
    }

    .toggle-label {
        display: flex;
        align-items: center;
        gap: 10px;
        cursor: pointer;
    }

    .toggle-label input[type="checkbox"] {
        width: 18px;
        height: 18px;
        accent-color: #6366f1;
    }

    .toggle-text {
        font-size: 14px;
        color: #fff;
    }

    .toggle-hint {
        font-size: 11px;
        color: #666;
        margin-left: auto;
    }

    .play-btn {
        width: calc(100% - 16px);
        margin: 8px;
        padding: 14px;
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        border: none;
        border-radius: 8px;
        color: white;
        font-weight: 600;
        font-size: 14px;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .play-btn:hover {
        background: linear-gradient(135deg, #5558e3, #7c4fe8);
        transform: scale(1.02);
    }

    .player4k-error {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        margin-top: 8px;
        padding: 10px;
        background: rgba(239, 68, 68, 0.2);
        border: 1px solid rgba(239, 68, 68, 0.5);
        border-radius: 8px;
        color: #fca5a5;
        font-size: 12px;
        text-align: center;
    }
</style>
