<script>
    /**
     * SplashScreen - Tela de carregamento inicial estilo Netflix
     */
    
    /** @type {{ progress?: number, status?: string, visible?: boolean }} */
    let { 
        progress = 0,
        status = 'Iniciando...',
        visible = true
    } = $props();
</script>

{#if visible}
    <div class="splash-screen" class:fade-out={progress >= 100}>
        <div class="splash-content">
            <!-- Logo Animation -->
            <div class="splash-logo">
                <div class="splash-logo-icon">
                    <span class="splash-emoji">🎬</span>
                    <div class="splash-glow"></div>
                </div>
                <h1 class="splash-title">
                    <span class="splash-go">Go</span><span class="splash-anime">Anime</span>
                </h1>
            </div>
            
            <!-- Loading Animation -->
            <div class="splash-loader">
                <div class="loader-bar">
                    <div class="loader-progress" style="width: {progress}%"></div>
                </div>
                <p class="loader-status">{status}</p>
            </div>
        </div>
    </div>
{/if}

<style>
    .splash-screen {
        position: fixed;
        inset: 0;
        background: linear-gradient(135deg, #0a0b0f 0%, #1a1d2e 50%, #0a0b0f 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9999;
        transition: opacity 0.5s ease-out;
    }
    
    .splash-screen.fade-out {
        opacity: 0;
        pointer-events: none;
    }
    
    .splash-content {
        text-align: center;
    }
    
    .splash-logo {
        margin-bottom: 40px;
    }
    
    .splash-logo-icon {
        position: relative;
        display: inline-block;
        margin-bottom: 20px;
    }
    
    .splash-emoji {
        font-size: 5rem;
        animation: pulse 2s ease-in-out infinite;
    }
    
    .splash-glow {
        position: absolute;
        inset: -20px;
        background: radial-gradient(circle, rgba(245, 87, 108, 0.3) 0%, transparent 70%);
        animation: glow 2s ease-in-out infinite;
        pointer-events: none;
    }
    
    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }
    
    @keyframes glow {
        0%, 100% { opacity: 0.5; transform: scale(1); }
        50% { opacity: 1; transform: scale(1.2); }
    }
    
    .splash-title {
        font-size: 3rem;
        font-weight: 800;
        margin: 0;
    }
    
    .splash-go {
        color: #fff;
    }
    
    .splash-anime {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .splash-loader {
        width: 280px;
        margin: 0 auto;
    }
    
    .loader-bar {
        height: 4px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 4px;
        overflow: hidden;
        margin-bottom: 12px;
    }
    
    .loader-progress {
        height: 100%;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border-radius: 4px;
        transition: width 0.3s ease-out;
    }
    
    .loader-status {
        color: #666;
        font-size: 0.9rem;
        margin: 0;
    }
</style>
