<script>
    import { onMount } from 'svelte';
    import {
        GetTorrentSources, SearchTorrents,
        TorBoxGetTorrentFiles, TorBoxGetFileStreamURL, InitTorBox
    } from '../../../../wailsjs/go/main/App';

    export let show = false;
    export let onClose = () => {};
    export let animeTitle = '';
    export let onPlayStream = (url, title) => {};

    let sources = [];
    let selectedSource = 'all';
    let searchQuery = '';
    let results = [];
    let loading = false;
    let error = '';
    let expandedTorrent = null;
    let torrentFiles = [];
    let loadingFiles = false;

    // TorBox API key
    let torboxApiKey = '';
    let torboxConnected = false;

    onMount(async () => {
        // Carrega API key salva
        torboxApiKey = localStorage.getItem('torbox_api_key') || '';
        if (torboxApiKey) {
            torboxConnected = await InitTorBox(torboxApiKey);
        }

        // Carrega fontes disponÃ­veis
        try {
            sources = await GetTorrentSources();
        } catch (e) {
            console.error('Erro ao carregar fontes:', e);
        }
    });

    $: if (show && animeTitle && searchQuery !== animeTitle) {
        searchQuery = animeTitle;
        search();
    }

    async function search() {
        if (!searchQuery.trim()) return;

        loading = true;
        error = '';
        results = [];
        expandedTorrent = null;

        try {
            results = await SearchTorrents(searchQuery, selectedSource);
            if (results.length === 0) {
                error = 'Nenhum torrent encontrado';
            }
        } catch (e) {
            error = 'Erro na busca: ' + e.message;
        } finally {
            loading = false;
        }
    }

    function changeSource(sourceId) {
        selectedSource = sourceId;
        if (searchQuery) {
            search();
        }
    }

    let expandedTorrentInfo = null;  // Info do torrent expandido

    async function expandTorrent(torrent) {
        if (expandedTorrent?.hash === torrent.hash) {
            expandedTorrent = null;
            torrentFiles = [];
            expandedTorrentInfo = null;
            return;
        }

        if (!torboxConnected) {
            error = 'Conecte ao TorBox para ver os arquivos';
            return;
        }

        expandedTorrent = torrent;
        loadingFiles = true;
        torrentFiles = [];
        expandedTorrentInfo = null;

        try {
            const info = await TorBoxGetTorrentFiles(torrent.magnet, torrent.hash);
            if (info) {
                expandedTorrentInfo = info;
                if (info.files && info.files.length > 0) {
                    torrentFiles = info.files;
                } else {
                    // Torrent ainda sem arquivos - mostra status
                    error = `Torrent carregando... Status: ${info.status || 'aguardando'}, Progresso: ${(info.progress || 0).toFixed(1)}%`;
                }
            } else {
                error = 'Não foi possível obter arquivos do torrent';
            }
        } catch (e) {
            error = 'Erro ao carregar arquivos: ' + e.message;
        } finally {
            loadingFiles = false;
        }
    }

    async function playFile(file) {
        if (!file.torrentId || !file.id) {
            error = 'Arquivo invÃ¡lido';
            return;
        }

        try {
            const streamUrl = await TorBoxGetFileStreamURL(file.torrentId, file.id);
            if (streamUrl) {
                onPlayStream(streamUrl, file.shortName || file.name);
                onClose();
            } else {
                error = 'NÃ£o foi possÃ­vel obter URL de streaming';
            }
        } catch (e) {
            error = 'Erro ao reproduzir: ' + e.message;
        }
    }

    function copyMagnet(magnet) {
        navigator.clipboard.writeText(magnet);
    }

    function formatSize(size) {
        return size || 'N/A';
    }
</script>

{#if show}
<div class="modal-overlay" on:click={onClose}>
    <div class="modal-content" on:click|stopPropagation>
        <button class="close-btn" on:click={onClose}>âœ•</button>

        <div class="modal-header">
            <h2>ðŸ§² Buscar Torrents</h2>
            <p class="subtitle">Encontre torrents de anime em mÃºltiplas fontes</p>
        </div>

        <!-- Barra de busca -->
        <div class="search-bar">
            <input
                type="text"
                bind:value={searchQuery}
                placeholder="Buscar anime..."
                on:keypress={(e) => e.key === 'Enter' && search()}
            />
            <button class="btn-search" on:click={search} disabled={loading}>
                {loading ? 'â³' : 'ðŸ”'}
            </button>
        </div>

        <!-- Filtros de fonte -->
        <div class="source-filters">
            {#each sources as source}
                <button
                    class="source-btn {selectedSource === source.id ? 'active' : ''} {source.isBr ? 'br' : ''}"
                    on:click={() => changeSource(source.id)}
                    disabled={!source.available}
                    title={source.description}
                >
                    {#if source.isBr}ðŸ‡§ðŸ‡·{/if}
                    {source.name}
                    {#if !source.available}
                        <span class="offline">âš ï¸</span>
                    {/if}
                </button>
            {/each}
        </div>

        <!-- Status TorBox -->
        {#if !torboxConnected}
            <div class="torbox-warning">
                âš ï¸ TorBox nÃ£o conectado - Configure nas configuraÃ§Ãµes para streaming direto
            </div>
        {/if}

        <!-- Erro -->
        {#if error}
            <div class="error">{error}</div>
        {/if}

        <!-- Resultados -->
        <div class="results-container">
            {#if loading}
                <div class="loading">
                    <div class="spinner"></div>
                    <p>Buscando em {selectedSource === 'all' ? 'todas as fontes' : sources.find(s => s.id === selectedSource)?.name}...</p>
                </div>
            {:else if results.length > 0}
                <div class="results-header">
                    <span>{results.length} resultados</span>
                    <span class="legend">
                        <span class="badge br">ðŸ‡§ðŸ‡· BR</span>
                        <span class="badge dual">ðŸ”Š Dual</span>
                    </span>
                </div>
                <div class="results-list">
                    {#each results as torrent}
                        <div class="torrent-item {expandedTorrent?.hash === torrent.hash ? 'expanded' : ''}">
                            <div class="torrent-main" on:click={() => expandTorrent(torrent)}>
                                <div class="torrent-info">
                                    <div class="torrent-title">
                                        {#if torrent.isBr}<span class="flag">ðŸ‡§ðŸ‡·</span>{/if}
                                        {torrent.title}
                                    </div>
                                    <div class="torrent-meta">
                                        <span class="source {torrent.source.toLowerCase().replace(/[^a-z]/g, '')}">{torrent.source}</span>
                                        {#if torrent.quality}<span class="quality">{torrent.quality}</span>{/if}
                                        <span class="size">{formatSize(torrent.size)}</span>
                                        <span class="seeders">ðŸŒ± {torrent.seeders}</span>
                                        {#if torrent.dualAudio}<span class="dual">ðŸ”Š Dual</span>{/if}
                                    </div>
                                </div>
                                <div class="torrent-actions">
                                    <button class="btn-icon" on:click|stopPropagation={() => copyMagnet(torrent.magnet)} title="Copiar magnet">
                                        ðŸ“‹
                                    </button>
                                    <button class="btn-expand" title="Ver arquivos">
                                        {expandedTorrent?.hash === torrent.hash ? 'â–²' : 'â–¼'}
                                    </button>
                                </div>
                            </div>

                            <!-- Arquivos expandidos -->
                            {#if expandedTorrent?.hash === torrent.hash}
                                <div class="torrent-files">
                                    {#if loadingFiles}
                                        <div class="loading-files">
                                            <div class="spinner-small"></div>
                                            <span>Carregando arquivos...</span>
                                        </div>
                                    {:else if torrentFiles.length > 0}
                                        <div class="files-list">
                                            {#each torrentFiles as file}
                                                <div class="file-item">
                                                    <div class="file-info">
                                                        <span class="file-episode">EP {file.episode || '?'}</span>
                                                        <span class="file-name">{file.shortName || file.name}</span>
                                                        <span class="file-size">{file.sizeStr}</span>
                                                    </div>
                                                    <button class="btn-play" on:click={() => playFile(file)}>
                                                        â–¶ï¸ Play
                                                    </button>
                                                </div>
                                            {/each}
                                        </div>
                                    {:else}
                                        <div class="no-files">Nenhum arquivo de vÃ­deo encontrado</div>
                                    {/if}
                                </div>
                            {/if}
                        </div>
                    {/each}
                </div>
            {:else if !loading && searchQuery}
                <div class="no-results">
                    <p>Nenhum torrent encontrado para "{searchQuery}"</p>
                    <p class="tip">Tente buscar pelo nome em inglÃªs ou japonÃªs</p>
                </div>
            {/if}
        </div>
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
        background: rgba(0, 0, 0, 0.9);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
        animation: fadeIn 0.2s ease;
    }

    .modal-content {
        background: linear-gradient(135deg, #1a1f3a 0%, #0d0f1a 100%);
        border-radius: 20px;
        padding: 25px;
        max-width: 800px;
        width: 95%;
        max-height: 90vh;
        overflow: hidden;
        display: flex;
        flex-direction: column;
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
        font-size: 20px;
        cursor: pointer;
        transition: color 0.2s;
        z-index: 10;
    }

    .close-btn:hover {
        color: #fff;
    }

    .modal-header {
        text-align: center;
        margin-bottom: 20px;
    }

    .modal-header h2 {
        margin: 0 0 5px 0;
        color: #fff;
        font-size: 1.4rem;
    }

    .subtitle {
        color: #888;
        margin: 0;
        font-size: 0.85rem;
    }

    .search-bar {
        display: flex;
        gap: 10px;
        margin-bottom: 15px;
    }

    .search-bar input {
        flex: 1;
        padding: 12px 15px;
        background: #0d0f1a;
        border: 1px solid #333;
        border-radius: 10px;
        color: #fff;
        font-size: 14px;
    }

    .search-bar input:focus {
        border-color: #f5576c;
        outline: none;
    }

    .btn-search {
        padding: 12px 20px;
        background: linear-gradient(135deg, #f5576c, #f093fb);
        border: none;
        border-radius: 10px;
        color: #fff;
        font-size: 16px;
        cursor: pointer;
        transition: transform 0.2s;
    }

    .btn-search:hover:not(:disabled) {
        transform: scale(1.05);
    }

    .btn-search:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .source-filters {
        display: flex;
        gap: 8px;
        margin-bottom: 15px;
        flex-wrap: wrap;
    }

    .source-btn {
        padding: 8px 16px;
        background: #1a1f3a;
        border: 1px solid #333;
        border-radius: 20px;
        color: #aaa;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .source-btn:hover:not(:disabled) {
        border-color: #f5576c;
        color: #fff;
    }

    .source-btn.active {
        background: linear-gradient(135deg, #f5576c, #f093fb);
        border-color: transparent;
        color: #fff;
    }

    .source-btn.br {
        border-color: #22c55e;
    }

    .source-btn.br.active {
        background: linear-gradient(135deg, #22c55e, #16a34a);
    }

    .source-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .offline {
        margin-left: 4px;
    }

    .torbox-warning {
        background: rgba(245, 158, 11, 0.2);
        border: 1px solid rgba(245, 158, 11, 0.5);
        color: #fbbf24;
        padding: 10px 15px;
        border-radius: 8px;
        font-size: 0.85rem;
        margin-bottom: 15px;
    }

    .error {
        background: rgba(239, 68, 68, 0.2);
        border: 1px solid rgba(239, 68, 68, 0.5);
        color: #ef4444;
        padding: 10px 15px;
        border-radius: 8px;
        font-size: 0.85rem;
        margin-bottom: 15px;
    }

    .results-container {
        flex: 1;
        overflow-y: auto;
        min-height: 200px;
    }

    .results-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
        color: #888;
        font-size: 0.85rem;
    }

    .legend {
        display: flex;
        gap: 10px;
    }

    .badge {
        padding: 2px 8px;
        border-radius: 10px;
        font-size: 0.75rem;
    }

    .badge.br {
        background: rgba(34, 197, 94, 0.2);
        color: #22c55e;
    }

    .badge.dual {
        background: rgba(59, 130, 246, 0.2);
        color: #3b82f6;
    }

    .loading {
        text-align: center;
        padding: 40px;
        color: #888;
    }

    .spinner {
        width: 40px;
        height: 40px;
        border: 3px solid #333;
        border-top-color: #f5576c;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin: 0 auto 15px;
    }

    .spinner-small {
        width: 20px;
        height: 20px;
        border: 2px solid #333;
        border-top-color: #f5576c;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    .results-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .torrent-item {
        background: rgba(255, 255, 255, 0.03);
        border: 1px solid #333;
        border-radius: 10px;
        overflow: hidden;
        transition: border-color 0.2s;
    }

    .torrent-item:hover {
        border-color: #555;
    }

    .torrent-item.expanded {
        border-color: #f5576c;
    }

    .torrent-main {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 15px;
        cursor: pointer;
    }

    .torrent-info {
        flex: 1;
        min-width: 0;
    }

    .torrent-title {
        color: #fff;
        font-size: 0.9rem;
        margin-bottom: 5px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .flag {
        margin-right: 5px;
    }

    .torrent-meta {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
        align-items: center;
    }

    .torrent-meta span {
        font-size: 0.75rem;
        padding: 2px 8px;
        border-radius: 4px;
        background: rgba(255, 255, 255, 0.1);
        color: #aaa;
    }

    .torrent-meta .source {
        background: rgba(139, 92, 246, 0.2);
        color: #a78bfa;
    }

    .torrent-meta .source.redetorrent {
        background: rgba(34, 197, 94, 0.2);
        color: #22c55e;
    }

    .torrent-meta .source.nyaa {
        background: rgba(59, 130, 246, 0.2);
        color: #3b82f6;
    }

    .torrent-meta .quality {
        background: rgba(245, 87, 108, 0.2);
        color: #f5576c;
    }

    .torrent-meta .seeders {
        background: rgba(34, 197, 94, 0.2);
        color: #22c55e;
    }

    .torrent-meta .dual {
        background: rgba(59, 130, 246, 0.2);
        color: #3b82f6;
    }

    .torrent-actions {
        display: flex;
        gap: 8px;
        align-items: center;
    }

    .btn-icon {
        background: none;
        border: none;
        color: #888;
        font-size: 16px;
        cursor: pointer;
        padding: 5px;
        transition: color 0.2s;
    }

    .btn-icon:hover {
        color: #fff;
    }

    .btn-expand {
        background: rgba(255, 255, 255, 0.1);
        border: none;
        color: #888;
        padding: 5px 10px;
        border-radius: 5px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-expand:hover {
        background: rgba(255, 255, 255, 0.2);
        color: #fff;
    }

    .torrent-files {
        border-top: 1px solid #333;
        padding: 15px;
        background: rgba(0, 0, 0, 0.3);
    }

    .loading-files {
        display: flex;
        align-items: center;
        gap: 10px;
        color: #888;
        font-size: 0.85rem;
    }

    .files-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
        max-height: 300px;
        overflow-y: auto;
    }

    .file-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px 12px;
        background: rgba(255, 255, 255, 0.05);
        border-radius: 8px;
    }

    .file-info {
        display: flex;
        gap: 10px;
        align-items: center;
        flex: 1;
        min-width: 0;
    }

    .file-episode {
        background: linear-gradient(135deg, #f5576c, #f093fb);
        color: #fff;
        padding: 3px 8px;
        border-radius: 5px;
        font-size: 0.75rem;
        font-weight: 600;
        white-space: nowrap;
    }

    .file-name {
        color: #ddd;
        font-size: 0.85rem;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        flex: 1;
    }

    .file-size {
        color: #888;
        font-size: 0.75rem;
        white-space: nowrap;
    }

    .btn-play {
        background: linear-gradient(135deg, #22c55e, #16a34a);
        border: none;
        color: #fff;
        padding: 6px 12px;
        border-radius: 6px;
        font-size: 0.8rem;
        cursor: pointer;
        transition: transform 0.2s;
        white-space: nowrap;
    }

    .btn-play:hover {
        transform: scale(1.05);
    }

    .no-files {
        color: #888;
        text-align: center;
        padding: 20px;
    }

    .no-results {
        text-align: center;
        padding: 40px;
        color: #888;
    }

    .no-results .tip {
        font-size: 0.85rem;
        color: #666;
        margin-top: 10px;
    }
</style>
