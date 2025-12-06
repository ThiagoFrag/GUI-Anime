<script>
    /**
     * MangaReader - Leitor de mangá
     * Exibe as páginas de um capítulo de mangá
     */
    import { getChapterPages } from '../../services/manga.js';
    import LoadingSpinner from '../ui/LoadingSpinner.svelte';
    
    /** @type {{ chapter: Object, onClose?: Function, onNextChapter?: Function, onPrevChapter?: Function }} */
    let { 
        chapter,
        onClose = /** @type {(e?: MouseEvent) => void} */ (() => {}),
        onNextChapter = null,
        onPrevChapter = null
    } = $props();
    
    let pages = $state([]);
    let loading = $state(true);
    let error = $state('');
    let currentPage = $state(0);
    let viewMode = $state('scroll'); // 'scroll' ou 'page'
    let zoom = $state(100);
    
    $effect(() => {
        if (chapter?.url) {
            loadPages();
        }
    });
    
    async function loadPages() {
        loading = true;
        error = '';
        currentPage = 0;
        
        try {
            pages = await getChapterPages(chapter.url);
            if (pages.length === 0) {
                error = 'Nenhuma página encontrada neste capítulo';
            }
        } catch (e) {
            error = `Erro ao carregar páginas: ${e.message}`;
        } finally {
            loading = false;
        }
    }
    
    function nextPage() {
        if (currentPage < pages.length - 1) {
            currentPage++;
        } else if (onNextChapter) {
            onNextChapter();
        }
    }
    
    function prevPage() {
        if (currentPage > 0) {
            currentPage--;
        } else if (onPrevChapter) {
            onPrevChapter();
        }
    }
    
    function goToPage(index) {
        currentPage = index;
    }
    
    function toggleViewMode() {
        viewMode = viewMode === 'scroll' ? 'page' : 'scroll';
    }
    
    function handleKeydown(e) {
        if (viewMode === 'page') {
            if (e.key === 'ArrowRight' || e.key === ' ') {
                nextPage();
            } else if (e.key === 'ArrowLeft') {
                prevPage();
            }
        }
        if (e.key === 'Escape') {
            onClose();
        }
    }
    
    function zoomIn() {
        zoom = Math.min(zoom + 25, 200);
    }
    
    function zoomOut() {
        zoom = Math.max(zoom - 25, 50);
    }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="manga-reader">
    <header class="reader-header">
        <button class="btn-back" onclick={(e) => onClose(e)}>
            ← Voltar
        </button>
        
        <div class="chapter-info">
            <span class="manga-name">{chapter.mangaName || 'Mangá'}</span>
            <span class="chapter-number">Capítulo {chapter.number || chapter.title}</span>
        </div>
        
        <div class="reader-controls">
            <button class="btn-control" onclick={zoomOut} title="Diminuir zoom">
                🔍-
            </button>
            <span class="zoom-level">{zoom}%</span>
            <button class="btn-control" onclick={zoomIn} title="Aumentar zoom">
                🔍+
            </button>
            
            <button class="btn-control" onclick={toggleViewMode} title="Alternar modo de visualização">
                {viewMode === 'scroll' ? '📄' : '📜'}
            </button>
        </div>
    </header>
    
    {#if loading}
        <div class="loading-container">
            <LoadingSpinner />
            <p>Carregando páginas...</p>
        </div>
    {:else if error}
        <div class="error-container">
            <p>{error}</p>
            <button onclick={loadPages}>Tentar novamente</button>
        </div>
    {:else if viewMode === 'scroll'}
        <div class="scroll-view">
            {#each pages as page, i}
                <img 
                    src={page.url} 
                    alt="Página {page.number || i + 1}"
                    loading="lazy"
                    style="width: {zoom}%"
                />
            {/each}
        </div>
    {:else}
        <div class="page-view">
            {#if pages[currentPage]}
                <img 
                    src={pages[currentPage].url} 
                    alt="Página {currentPage + 1}"
                    style="max-width: {zoom}%; max-height: 90vh"
                />
            {/if}
            
            <div class="page-navigation">
                <button 
                    class="nav-btn prev" 
                    onclick={prevPage}
                    disabled={currentPage === 0 && !onPrevChapter}
                >
                    ◀ Anterior
                </button>
                
                <span class="page-counter">
                    {currentPage + 1} / {pages.length}
                </span>
                
                <button 
                    class="nav-btn next" 
                    onclick={nextPage}
                    disabled={currentPage === pages.length - 1 && !onNextChapter}
                >
                    Próxima ▶
                </button>
            </div>
        </div>
    {/if}
    
    {#if !loading && pages.length > 0}
        <div class="page-thumbnails">
            {#each pages as page, i}
                <button 
                    class="thumbnail {currentPage === i ? 'active' : ''}"
                    onclick={() => goToPage(i)}
                    title="Página {i + 1}"
                >
                    {i + 1}
                </button>
            {/each}
        </div>
    {/if}
</div>

<style>
    .manga-reader {
        position: fixed;
        inset: 0;
        background: #0a0a0f;
        z-index: 1000;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }
    
    .reader-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 12px 20px;
        background: rgba(20, 20, 30, 0.95);
        border-bottom: 1px solid #333;
        gap: 20px;
    }
    
    .btn-back {
        background: transparent;
        border: 1px solid #555;
        color: #fff;
        padding: 8px 16px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 0.9rem;
        transition: all 0.2s;
    }
    
    .btn-back:hover {
        background: #333;
        border-color: #777;
    }
    
    .chapter-info {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
    }
    
    .manga-name {
        font-size: 1rem;
        font-weight: 600;
        color: #fff;
    }
    
    .chapter-number {
        font-size: 0.85rem;
        color: #888;
    }
    
    .reader-controls {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    
    .btn-control {
        background: transparent;
        border: 1px solid #555;
        color: #fff;
        padding: 6px 12px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 1rem;
        transition: all 0.2s;
    }
    
    .btn-control:hover {
        background: #333;
    }
    
    .zoom-level {
        color: #888;
        font-size: 0.85rem;
        min-width: 50px;
        text-align: center;
    }
    
    .loading-container,
    .error-container {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        color: #888;
        gap: 20px;
    }
    
    .error-container button {
        background: #663399;
        border: none;
        color: #fff;
        padding: 10px 20px;
        border-radius: 8px;
        cursor: pointer;
    }
    
    .scroll-view {
        flex: 1;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 20px;
        gap: 10px;
    }
    
    .scroll-view img {
        max-width: 100%;
        height: auto;
        display: block;
    }
    
    .page-view {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 20px;
        gap: 20px;
    }
    
    .page-view img {
        object-fit: contain;
    }
    
    .page-navigation {
        display: flex;
        align-items: center;
        gap: 30px;
    }
    
    .nav-btn {
        background: #663399;
        border: none;
        color: #fff;
        padding: 12px 24px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 1rem;
        transition: all 0.2s;
    }
    
    .nav-btn:hover:not(:disabled) {
        background: #7744aa;
        transform: scale(1.05);
    }
    
    .nav-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }
    
    .page-counter {
        color: #fff;
        font-size: 1.1rem;
        font-weight: 500;
    }
    
    .page-thumbnails {
        display: flex;
        gap: 6px;
        padding: 10px 20px;
        background: rgba(20, 20, 30, 0.95);
        border-top: 1px solid #333;
        overflow-x: auto;
        justify-content: center;
        flex-wrap: wrap;
    }
    
    .thumbnail {
        width: 32px;
        height: 32px;
        border-radius: 4px;
        border: 1px solid #444;
        background: #222;
        color: #aaa;
        font-size: 0.75rem;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .thumbnail:hover {
        border-color: #666;
        background: #333;
    }
    
    .thumbnail.active {
        border-color: #663399;
        background: #663399;
        color: #fff;
    }
</style>
