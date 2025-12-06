<script>
    /**
     * LoginScreen - Tela de login/criação de conta
     */
    import { AVATARS } from '../../constants/genres.js';
    
    /** @type {{ onLogin?: Function }} */
    let { onLogin = () => {} } = $props();
    
    let nomeInput = $state('');
    let avatarSelecionado = $state('avatar1.png');
    
    function criarConta() {
        if (!nomeInput.trim()) return;
        onLogin(nomeInput.trim(), avatarSelecionado);
    }
</script>

<div class="login-screen">
    <!-- Animated Background -->
    <div class="login-bg">
        <div class="bg-gradient"></div>
        <div class="bg-particles">
            {#each Array(20) as _, i}
                <div 
                    class="particle" 
                    style="--delay: {i * 0.5}s; --x: {Math.random() * 100}%; --duration: {15 + Math.random() * 20}s"
                ></div>
            {/each}
        </div>
    </div>
    
    <!-- Login Content -->
    <div class="login-content">
        <!-- Logo Section -->
        <div class="login-branding">
            <div class="logo-container">
                <div class="logo-icon">
                    <span class="logo-emoji">🎬</span>
                    <div class="logo-glow"></div>
                </div>
                <h1 class="logo-text">
                    <span class="logo-go">Go</span><span class="logo-anime">Anime</span>
                </h1>
            </div>
            <p class="login-tagline">Sua plataforma de anime favorita</p>
        </div>
        
        <!-- Login Card -->
        <div class="login-card-modern">
            <div class="card-header">
                <h2>Bem-vindo!</h2>
                <p>Crie seu perfil para começar</p>
            </div>
            
            <div class="card-body">
                <!-- Name Input -->
                <div class="input-group">
                    <label for="username">Seu nome</label>
                    <div class="input-wrapper">
                        <span class="input-icon">👤</span>
                        <input 
                            id="username"
                            type="text"
                            bind:value={nomeInput} 
                            placeholder="Digite seu nome" 
                            class="input-modern"
                            onkeydown={(e) => e.key === 'Enter' && criarConta()}
                        />
                    </div>
                </div>
                
                <!-- Avatar Selection -->
                <div class="avatar-group">
                    <span class="avatar-label" id="avatar-label">Escolha seu avatar</span>
                    <div class="avatar-grid" role="radiogroup" aria-labelledby="avatar-label">
                        {#each AVATARS as avatar}
                            <button 
                                type="button"
                                class="avatar-option {avatarSelecionado === avatar.id ? 'selected' : ''}"
                                onclick={() => avatarSelecionado = avatar.id}
                                title={avatar.label}
                            >
                                <span class="avatar-emoji">{avatar.emoji}</span>
                                {#if avatarSelecionado === avatar.id}
                                    <span class="avatar-check">✓</span>
                                {/if}
                            </button>
                        {/each}
                    </div>
                </div>
            </div>
            
            <div class="card-footer">
                <button type="button" class="btn-enter" onclick={criarConta} disabled={!nomeInput.trim()}>
                    <span>Entrar</span>
                    <span class="btn-arrow">→</span>
                </button>
            </div>
        </div>
        
        <!-- Features Preview -->
        <div class="login-features">
            <div class="feature">
                <span class="feature-icon">🔥</span>
                <span class="feature-text">Animes em alta qualidade</span>
            </div>
            <div class="feature">
                <span class="feature-icon">⚡</span>
                <span class="feature-text">Streaming rápido</span>
            </div>
            <div class="feature">
                <span class="feature-icon">📱</span>
                <span class="feature-text">100% gratuito</span>
            </div>
        </div>
    </div>
</div>

<style>
    .login-screen {
        position: fixed;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        background: #0a0b0f;
        overflow: hidden;
    }
    
    .login-bg {
        position: absolute;
        inset: 0;
    }
    
    .bg-gradient {
        position: absolute;
        inset: 0;
        background: radial-gradient(ellipse at 30% 20%, rgba(245, 87, 108, 0.15) 0%, transparent 50%),
                    radial-gradient(ellipse at 70% 80%, rgba(240, 147, 251, 0.1) 0%, transparent 50%);
    }
    
    .bg-particles {
        position: absolute;
        inset: 0;
        overflow: hidden;
    }
    
    .particle {
        position: absolute;
        width: 4px;
        height: 4px;
        background: rgba(245, 87, 108, 0.5);
        border-radius: 50%;
        left: var(--x);
        animation: float var(--duration) linear infinite;
        animation-delay: var(--delay);
        opacity: 0;
    }
    
    @keyframes float {
        0% { transform: translateY(100vh); opacity: 0; }
        10% { opacity: 0.5; }
        90% { opacity: 0.5; }
        100% { transform: translateY(-100vh); opacity: 0; }
    }
    
    .login-content {
        position: relative;
        z-index: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 40px;
        padding: 20px;
        max-width: 500px;
        width: 100%;
    }
    
    .login-branding {
        text-align: center;
    }
    
    .logo-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 10px;
    }
    
    .logo-icon {
        position: relative;
    }
    
    .logo-emoji {
        font-size: 4rem;
    }
    
    .logo-glow {
        position: absolute;
        inset: -20px;
        background: radial-gradient(circle, rgba(245, 87, 108, 0.3) 0%, transparent 70%);
        animation: pulse 3s ease-in-out infinite;
    }
    
    @keyframes pulse {
        0%, 100% { opacity: 0.5; transform: scale(1); }
        50% { opacity: 1; transform: scale(1.1); }
    }
    
    .logo-text {
        font-size: 2.5rem;
        font-weight: 800;
        margin: 0;
    }
    
    .logo-go {
        color: #fff;
    }
    
    .logo-anime {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .login-tagline {
        color: #666;
        font-size: 1.1rem;
        margin: 10px 0 0 0;
    }
    
    .login-card-modern {
        width: 100%;
        background: rgba(26, 29, 46, 0.8);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        padding: 35px;
        backdrop-filter: blur(10px);
    }
    
    .card-header {
        text-align: center;
        margin-bottom: 30px;
    }
    
    .card-header h2 {
        font-size: 1.6rem;
        font-weight: 700;
        color: #fff;
        margin: 0 0 8px 0;
    }
    
    .card-header p {
        color: #888;
        font-size: 0.95rem;
        margin: 0;
    }
    
    .card-body {
        display: flex;
        flex-direction: column;
        gap: 25px;
    }
    
    .input-group label {
        display: block;
        color: #aaa;
        font-size: 0.85rem;
        margin-bottom: 8px;
    }
    
    .input-wrapper {
        position: relative;
    }
    
    .input-icon {
        position: absolute;
        left: 16px;
        top: 50%;
        transform: translateY(-50%);
        font-size: 1.1rem;
    }
    
    .input-modern {
        width: 100%;
        padding: 14px 16px 14px 48px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        color: #fff;
        font-size: 1rem;
        outline: none;
        transition: all 0.2s;
        box-sizing: border-box;
    }
    
    .input-modern:focus {
        border-color: rgba(245, 87, 108, 0.5);
        background: rgba(255, 255, 255, 0.08);
    }
    
    .input-modern::placeholder {
        color: #555;
    }
    
    .avatar-group {
        text-align: center;
    }
    
    .avatar-label {
        display: block;
        margin-bottom: 15px;
        color: #aaa;
        font-size: 0.9rem;
        font-weight: 500;
    }
    
    .avatar-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 12px;
    }
    
    .avatar-option {
        aspect-ratio: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(255, 255, 255, 0.05);
        border: 2px solid transparent;
        border-radius: 16px;
        cursor: pointer;
        transition: all 0.2s;
        position: relative;
    }
    
    .avatar-option:hover {
        background: rgba(255, 255, 255, 0.1);
        transform: scale(1.05);
    }
    
    .avatar-option.selected {
        border-color: #f5576c;
        background: rgba(245, 87, 108, 0.1);
    }
    
    .avatar-emoji {
        font-size: 2rem;
    }
    
    .avatar-check {
        position: absolute;
        bottom: 4px;
        right: 4px;
        width: 20px;
        height: 20px;
        background: #f5576c;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.7rem;
        color: #fff;
    }
    
    .card-footer {
        margin-top: 30px;
    }
    
    .btn-enter {
        width: 100%;
        padding: 16px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border: none;
        border-radius: 12px;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 600;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        transition: all 0.3s;
    }
    
    .btn-enter:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 10px 30px rgba(245, 87, 108, 0.4);
    }
    
    .btn-enter:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    .btn-arrow {
        font-size: 1.2rem;
    }
    
    .login-features {
        display: flex;
        gap: 30px;
        flex-wrap: wrap;
        justify-content: center;
    }
    
    .feature {
        display: flex;
        align-items: center;
        gap: 8px;
        color: #666;
        font-size: 0.9rem;
    }
    
    .feature-icon {
        font-size: 1.2rem;
    }
</style>
