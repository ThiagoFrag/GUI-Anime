<script>
    import { onMount } from 'svelte';
    import { 
        InitTorBox, TorBoxGetUser, TorBoxSearchTorrents, 
        TorBoxGetInstantStream, TorBoxStreamAnimeEpisode 
    } from '../../wailsjs/go/main/App';

    export let show = false;
    export let onClose = () => {};

    let apiKey = '';
    let user = null;
    let loading = false;
    let error = '';
    let connected = false;
    let savedKey = '';

    onMount(async () => {
        // Carrega API key salva do localStorage
        savedKey = localStorage.getItem('torbox_api_key') || '';
        if (savedKey) {
            apiKey = savedKey;
            await connect();
        }
    });

    async function connect() {
        if (!apiKey.trim()) {
            error = 'Digite sua API key do TorBox';
            return;
        }

        loading = true;
        error = '';

        try {
            const result = await InitTorBox(apiKey);
            if (result) {
                user = await TorBoxGetUser();
                if (user) {
                    connected = true;
                    localStorage.setItem('torbox_api_key', apiKey);
                    error = '';
                } else {
                    error = 'API key inválida ou erro de conexão';
                }
            } else {
                error = 'Falha ao conectar com TorBox';
            }
        } catch (e) {
            error = 'Erro: ' + e.message;
        } finally {
            loading = false;
        }
    }

    function disconnect() {
        connected = false;
        user = null;
        localStorage.removeItem('torbox_api_key');
        apiKey = '';
    }

    function formatBytes(bytes) {
        if (!bytes) return '0 B';
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + sizes[i];
    }
</script>

{#if show}
<div class="modal-overlay" on:click={onClose}>
    <div class="modal-content" on:click|stopPropagation>
        <button class="close-btn" on:click={onClose}></button>
        
        <div class="modal-header">
            <img src="https://torbox.app/images/logo.svg" alt="TorBox" class="torbox-logo" />
            <h2> TorBox Streaming</h2>
            <p class="subtitle">Streaming instantâneo via torrent</p>
        </div>

        {#if !connected}
            <div class="connect-form">
                <p class="info">
                    O TorBox permite assistir anime via torrent com streaming instantâneo. 
                    Torrents em cache começam a tocar imediatamente!
                </p>

                <div class="input-group">
                    <label for="apikey">API Key</label>
                    <input 
                        type="password" 
                        id="apikey"
                        bind:value={apiKey} 
                        placeholder="Cole sua API key do TorBox"
                        disabled={loading}
                    />
                    <a href="https://torbox.app/settings" target="_blank" class="get-key">
                        Obter API key 
                    </a>
                </div>

                {#if error}
                    <div class="error">{error}</div>
                {/if}

                <button class="btn-connect" on:click={connect} disabled={loading}>
                    {#if loading}
                        <span class="spinner"></span> Conectando...
                    {:else}
                         Conectar
                    {/if}
                </button>
            </div>
        {:else}
            <div class="connected-info">
                <div class="user-info">
                    <div class="avatar"></div>
                    <div class="details">
                        <span class="email">{user?.email || 'Usuário'}</span>
                        <span class="plan {user?.plan_name?.toLowerCase()}">
                            {user?.plan_name || 'Free'}
                        </span>
                    </div>
                </div>

                <div class="stats">
                    <div class="stat">
                        <span class="value">{formatBytes(user?.total_downloaded)}</span>
                        <span class="label">Download Total</span>
                    </div>
                    <div class="stat">
                        <span class="value">{user?.is_subscribed ? '' : ''}</span>
                        <span class="label">Premium</span>
                    </div>
                </div>

                <div class="features">
                    <h4> Funcionalidades</h4>
                    <ul>
                        <li> Streaming instantâneo de torrents em cache</li>
                        <li> Busca automática no Nyaa.si</li>
                        <li> Qualidade até 4K (se disponível)</li>
                        <li> Sem limite de velocidade</li>
                    </ul>
                </div>

                <p class="usage-tip">
                     <strong>Como usar:</strong> Ao assistir um anime, clique em 
                    " TorBox" para buscar via torrent com streaming instantâneo.
                </p>

                <button class="btn-disconnect" on:click={disconnect}>
                     Desconectar
                </button>
            </div>
        {/if}
    </div>
</div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.85);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
        animation: fadeIn 0.2s ease;
    }

    .modal-content {
        background: linear-gradient(135deg, #1a1f3a 0%, #0d0f1a 100%);
        border-radius: 20px;
        padding: 30px;
        max-width: 480px;
        width: 90%;
        position: relative;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
        border: 1px solid rgba(255, 255, 255, 0.1);
    }

    .close-btn {
        position: absolute;
        top: 15px;
        right: 15px;
        background: none;
        border: none;
        color: #888;
        font-size: 24px;
        cursor: pointer;
        transition: color 0.2s;
    }

    .close-btn:hover {
        color: #fff;
    }

    .modal-header {
        text-align: center;
        margin-bottom: 25px;
    }

    .torbox-logo {
        width: 100px;
        margin-bottom: 15px;
    }

    .modal-header h2 {
        margin: 0 0 5px 0;
        color: #fff;
        font-size: 1.5rem;
    }

    .subtitle {
        color: #888;
        margin: 0;
        font-size: 0.9rem;
    }

    .info {
        color: #aaa;
        font-size: 0.9rem;
        line-height: 1.6;
        margin-bottom: 20px;
    }

    .input-group {
        margin-bottom: 20px;
    }

    .input-group label {
        display: block;
        color: #fff;
        margin-bottom: 8px;
        font-weight: 500;
    }

    .input-group input {
        width: 100%;
        padding: 12px 15px;
        background: #0d0f1a;
        border: 1px solid #333;
        border-radius: 10px;
        color: #fff;
        font-size: 14px;
        box-sizing: border-box;
    }

    .input-group input:focus {
        border-color: #f5576c;
        outline: none;
    }

    .get-key {
        display: inline-block;
        margin-top: 8px;
        color: #f5576c;
        font-size: 0.85rem;
        text-decoration: none;
    }

    .get-key:hover {
        text-decoration: underline;
    }

    .error {
        background: rgba(255, 87, 108, 0.2);
        color: #f5576c;
        padding: 10px 15px;
        border-radius: 8px;
        margin-bottom: 15px;
        font-size: 0.9rem;
    }

    .btn-connect, .btn-disconnect {
        width: 100%;
        padding: 14px;
        border: none;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-connect {
        background: linear-gradient(135deg, #f5576c, #f093fb);
        color: #fff;
    }

    .btn-connect:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 5px 20px rgba(245, 87, 108, 0.4);
    }

    .btn-connect:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .btn-disconnect {
        background: #333;
        color: #fff;
        margin-top: 20px;
    }

    .btn-disconnect:hover {
        background: #444;
    }

    .spinner {
        display: inline-block;
        width: 16px;
        height: 16px;
        border: 2px solid #fff;
        border-top-color: transparent;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-right: 8px;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    .connected-info {
        text-align: center;
    }

    .user-info {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 15px;
        margin-bottom: 25px;
    }

    .avatar {
        width: 60px;
        height: 60px;
        background: linear-gradient(135deg, #f5576c, #f093fb);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 28px;
    }

    .details {
        text-align: left;
    }

    .email {
        display: block;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 500;
    }

    .plan {
        display: inline-block;
        padding: 4px 12px;
        border-radius: 20px;
        font-size: 0.8rem;
        margin-top: 5px;
        background: #333;
        color: #aaa;
    }

    .plan.pro, .plan.standard, .plan.essential {
        background: linear-gradient(135deg, #f5576c, #f093fb);
        color: #fff;
    }

    .stats {
        display: flex;
        justify-content: center;
        gap: 40px;
        margin-bottom: 25px;
    }

    .stat {
        text-align: center;
    }

    .stat .value {
        display: block;
        font-size: 1.3rem;
        font-weight: 600;
        color: #f5576c;
    }

    .stat .label {
        color: #888;
        font-size: 0.85rem;
    }

    .features {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 20px;
        text-align: left;
    }

    .features h4 {
        margin: 0 0 15px 0;
        color: #fff;
    }

    .features ul {
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .features li {
        color: #aaa;
        padding: 5px 0;
        font-size: 0.9rem;
    }

    .usage-tip {
        background: rgba(245, 87, 108, 0.1);
        border: 1px solid rgba(245, 87, 108, 0.3);
        border-radius: 10px;
        padding: 15px;
        color: #fff;
        font-size: 0.9rem;
        text-align: left;
    }
</style>
