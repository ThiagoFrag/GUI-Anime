<script>
    /**
     * DiscordLinkModal - Modal para vincular conta Discord
     */
    
    /** @type {{ visible?: boolean, serverInvite?: string, loading?: boolean, error?: string, onLink?: Function, onClose?: Function }} */
    let { 
        visible = false,
        serverInvite = '',
        loading = false,
        error = '',
        onLink = () => {},
        onClose = () => {}
    } = $props();
    
    let linkCode = $state('');
    
    async function handleSubmit() {
        if (!linkCode.trim()) return;
        await onLink(linkCode.trim());
        linkCode = '';
    }
    
    function handleKeydown(e) {
        if (e.key === 'Escape') {
            onClose();
        } else if (e.key === 'Enter' && linkCode.trim()) {
            handleSubmit();
        }
    }
    
    function handleOverlayClick() {
        onClose();
    }
</script>

{#if visible}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div 
        class="modal-overlay" 
        onclick={handleOverlayClick}
        onkeydown={handleKeydown}
        role="dialog"
        aria-modal="true"
        aria-labelledby="link-modal-title"
        tabindex="-1"
    >
        <div class="discord-link-modal" onclick={(e) => e.stopPropagation()}>
            <div class="link-modal-header">
                <h2 id="link-modal-title">🔗 Vincular Discord</h2>
                <button type="button" class="btn-close-modal" onclick={onClose}>✕</button>
            </div>
            
            <div class="link-modal-body">
                <div class="link-steps">
                    <div class="step">
                        <span class="step-number">1</span>
                        <div class="step-content">
                            <h4>Entre no servidor Discord</h4>
                            <p>Junte-se à nossa comunidade para vincular sua conta.</p>
                            <a 
                                href={serverInvite || 'https://discord.gg/goanime'} 
                                target="_blank" 
                                rel="noopener" 
                                class="btn-discord-invite"
                            >
                                <span class="discord-icon">💬</span>
                                Entrar no Servidor
                            </a>
                        </div>
                    </div>
                    
                    <div class="step">
                        <span class="step-number">2</span>
                        <div class="step-content">
                            <h4>Gere seu código</h4>
                            <p>Use o comando <code>/vincular</code> no Discord para receber seu código único.</p>
                        </div>
                    </div>
                    
                    <div class="step">
                        <span class="step-number">3</span>
                        <div class="step-content">
                            <h4>Cole o código abaixo</h4>
                            <input 
                                type="text"
                                bind:value={linkCode}
                                placeholder="ANIME-XXXXXXXX"
                                class="code-input"
                                disabled={loading}
                            />
                        </div>
                    </div>
                </div>
                
                {#if error}
                    <div class="link-error">
                        ⚠️ {error}
                    </div>
                {/if}
            </div>
            
            <div class="link-modal-footer">
                <button 
                    type="button" 
                    class="btn-link-confirm"
                    onclick={handleSubmit}
                    disabled={!linkCode.trim() || loading}
                >
                    {#if loading}
                        <span class="spinner-small"></span>
                        Vinculando...
                    {:else}
                        🔗 Vincular Conta
                    {/if}
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        padding: 20px;
    }
    
    .discord-link-modal {
        width: 100%;
        max-width: 480px;
        background: linear-gradient(180deg, #1a1d2e 0%, #0f1118 100%);
        border: 1px solid rgba(88, 101, 242, 0.3);
        border-radius: 16px;
        overflow: hidden;
    }
    
    .link-modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 20px 25px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    }
    
    .link-modal-header h2 {
        margin: 0;
        font-size: 1.3rem;
        color: #fff;
    }
    
    .btn-close-modal {
        width: 32px;
        height: 32px;
        background: rgba(255, 255, 255, 0.1);
        border: none;
        border-radius: 8px;
        color: #888;
        font-size: 1rem;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-close-modal:hover {
        background: rgba(245, 87, 108, 0.3);
        color: #fff;
    }
    
    .link-modal-body {
        padding: 25px;
    }
    
    .link-steps {
        display: flex;
        flex-direction: column;
        gap: 25px;
    }
    
    .step {
        display: flex;
        gap: 15px;
    }
    
    .step-number {
        flex-shrink: 0;
        width: 32px;
        height: 32px;
        background: linear-gradient(135deg, #5865f2 0%, #7289da 100%);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        color: #fff;
        font-size: 0.9rem;
    }
    
    .step-content {
        flex: 1;
    }
    
    .step-content h4 {
        margin: 0 0 5px 0;
        font-size: 1rem;
        color: #fff;
    }
    
    .step-content p {
        margin: 0 0 12px 0;
        color: #888;
        font-size: 0.9rem;
        line-height: 1.5;
    }
    
    .step-content code {
        background: rgba(88, 101, 242, 0.2);
        color: #7289da;
        padding: 2px 8px;
        border-radius: 4px;
        font-family: monospace;
    }
    
    .btn-discord-invite {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 18px;
        background: #5865f2;
        border: none;
        border-radius: 8px;
        color: #fff;
        font-size: 0.9rem;
        font-weight: 600;
        text-decoration: none;
        transition: all 0.2s;
    }
    
    .btn-discord-invite:hover {
        background: #4752c4;
        transform: translateY(-2px);
    }
    
    .discord-icon {
        font-size: 1.1rem;
    }
    
    .code-input {
        width: 100%;
        padding: 14px 16px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 10px;
        color: #fff;
        font-size: 1.1rem;
        font-family: monospace;
        text-transform: uppercase;
        letter-spacing: 2px;
        text-align: center;
        outline: none;
        transition: all 0.2s;
        box-sizing: border-box;
    }
    
    .code-input:focus {
        border-color: #5865f2;
        background: rgba(88, 101, 242, 0.1);
    }
    
    .code-input::placeholder {
        color: #555;
        letter-spacing: 2px;
    }
    
    .code-input:disabled {
        opacity: 0.6;
    }
    
    .link-error {
        margin-top: 15px;
        padding: 12px 15px;
        background: rgba(239, 68, 68, 0.15);
        border: 1px solid rgba(239, 68, 68, 0.3);
        border-radius: 8px;
        color: #ef4444;
        font-size: 0.9rem;
    }
    
    .link-modal-footer {
        padding: 20px 25px;
        border-top: 1px solid rgba(255, 255, 255, 0.1);
    }
    
    .btn-link-confirm {
        width: 100%;
        padding: 14px;
        background: linear-gradient(135deg, #5865f2 0%, #7289da 100%);
        border: none;
        border-radius: 10px;
        color: #fff;
        font-size: 1rem;
        font-weight: 600;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        transition: all 0.2s;
    }
    
    .btn-link-confirm:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 8px 25px rgba(88, 101, 242, 0.4);
    }
    
    .btn-link-confirm:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    .spinner-small {
        width: 18px;
        height: 18px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
</style>
