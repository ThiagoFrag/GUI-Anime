<script>
    /**
     * MangaDetails - Exibe detalhes e capítulos de um mangá
     */
    import { getMangaDetails, getMangaChapters } from '../../services/manga.js';
    import LoadingSpinner from '../ui/LoadingSpinner.svelte';
    import MangaReader from './MangaReader.svelte';
    
    /** @type {{ manga: Object, onClose?: Function }} */
    let { 
        manga,
        onClose = /** @type {(e?: MouseEvent) => void} */ (() => {})
    } = $props();
    
    let details = $state(null);
    let chapters = $state([]);
    let loading = $state(true);
    let error = $state('');
    let selectedChapter = $state(null);
    let chaptersExpanded = $state(false);
    
    $effect(() => {
        if (manga?.url) {
            loadMangaData();
        }
    });
    
    async function loadMangaData() {
        loading = true;
        error = '';
        
        try {
            const [detailsRes, chaptersRes] = await Promise.all([
                getMangaDetails(manga.url),
                getMangaChapters(manga.url)
            ]);
            
            details = detailsRes || manga;
            chapters = chaptersRes || [];
        } catch (e) {
            error = `Erro ao carregar: ${e.message}`;
            details = manga;
        } finally {
            loading = false;
        }
    }
    
    function openChapter(chapter) {
        selectedChapter = chapter;
    }
    
    function closeReader() {
        selectedChapter = null;
    }
    
    function nextChapter() {
        const currentIndex = chapters.findIndex(c => c.url === selectedChapter.url);
        if (currentIndex > 0) {
            selectedChapter = chapters[currentIndex - 1];
        }
    }
    
    function prevChapter() {
        const currentIndex = chapters.findIndex(c => c.url === selectedChapter.url);
        if (currentIndex < chapters.length - 1) {
            selectedChapter = chapters[currentIndex + 1];
        }
    }
    
    function toggleChapters() {
        chaptersExpanded = !chaptersExpanded;
    }
    
    let displayedChapters = $derived(
        chaptersExpanded ? chapters : chapters.slice(0, 20)
    );
</script>

{#if selectedChapter}
    <MangaReader 
        chapter={selectedChapter}
        onClose={closeReader}
        onNextChapter={nextChapter}
        onPrevChapter={prevChapter}
    />
{:else}
    <div class="manga-details">
        <header class="details-header">
            <button class="btn-back" onclick={(e) => onClose(e)}>
                ← Voltar
            </button>
        </header>
        
        {#if loading}
            <div class="loading-container">
                <LoadingSpinner />
                <p>Carregando detalhes...</p>
            </div>
        {:else if error}
            <div class="error-container">
                <p>{error}</p>
            </div>
        {:else}
            <div class="details-content">
                <div class="manga-hero">
                    <img 
                        src={details?.image || manga.image} 
                        alt={details?.title || manga.title}
                        class="manga-cover"
                    />
                    
                    <div class="manga-info">
                        <h1 class="manga-title">{details?.title || manga.title}</h1>
                        
                        {#if details?.status}
                            <span class="manga-status">{details.status}</span>
                        {/if}
                        
                        {#if details?.genres?.length}
                            <div class="manga-genres">
                                {#each details.genres as genre}
                                    <span class="genre-tag">{genre}</span>
                                {/each}
                            </div>
                        {/if}
                        
                        {#if details?.description}
                            <p class="manga-description">{details.description}</p>
                        {/if}
                        
                        <div class="manga-stats">
                            <span>📚 {chapters.length} capítulos</span>
                        </div>
                    </div>
                </div>
                
                <section class="chapters-section">
                    <h2>Capítulos</h2>
                    
                    {#if chapters.length === 0}
                        <p class="no-chapters">Nenhum capítulo encontrado</p>
                    {:else}
                        <div class="chapters-list">
                            {#each displayedChapters as chapter}
                                <button 
                                    class="chapter-item"
                                    onclick={() => openChapter(chapter)}
                                >
                                    <span class="chapter-number">Cap. {chapter.number || '?'}</span>
                                    <span class="chapter-title">{chapter.title}</span>
                                    {#if chapter.date}
                                        <span class="chapter-date">{chapter.date}</span>
                                    {/if}
                                </button>
                            {/each}
                        </div>
                        
                        {#if chapters.length > 20}
                            <button class="btn-show-more" onclick={toggleChapters}>
                                {chaptersExpanded ? 'Mostrar menos' : `Mostrar todos (${chapters.length})`}
                            </button>
                        {/if}
                    {/if}
                </section>
            </div>
        {/if}
    </div>
{/if}

<style>
    .manga-details {
        background: #0d0d14;
        min-height: 100vh;
    }
    
    .details-header {
        padding: 16px 20px;
        background: rgba(20, 20, 30, 0.9);
        position: sticky;
        top: 0;
        z-index: 100;
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
    
    .loading-container,
    .error-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 100px 20px;
        color: #888;
        gap: 20px;
    }
    
    .details-content {
        max-width: 1200px;
        margin: 0 auto;
        padding: 20px;
    }
    
    .manga-hero {
        display: flex;
        gap: 30px;
        margin-bottom: 40px;
    }
    
    .manga-cover {
        width: 220px;
        height: auto;
        border-radius: 12px;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
        object-fit: cover;
        aspect-ratio: 2/3;
    }
    
    .manga-info {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    
    .manga-title {
        font-size: 2rem;
        font-weight: 700;
        color: #fff;
        margin: 0;
    }
    
    .manga-status {
        display: inline-block;
        background: #663399;
        color: #fff;
        padding: 4px 12px;
        border-radius: 20px;
        font-size: 0.85rem;
        width: fit-content;
    }
    
    .manga-genres {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }
    
    .genre-tag {
        background: rgba(255, 255, 255, 0.1);
        color: #ccc;
        padding: 4px 10px;
        border-radius: 15px;
        font-size: 0.8rem;
    }
    
    .manga-description {
        color: #aaa;
        font-size: 0.95rem;
        line-height: 1.6;
        max-width: 600px;
    }
    
    .manga-stats {
        color: #888;
        font-size: 0.9rem;
    }
    
    .chapters-section h2 {
        color: #fff;
        font-size: 1.4rem;
        margin-bottom: 20px;
    }
    
    .no-chapters {
        color: #666;
        text-align: center;
        padding: 40px;
    }
    
    .chapters-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    
    .chapter-item {
        display: flex;
        align-items: center;
        gap: 15px;
        padding: 14px 18px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid transparent;
        border-radius: 10px;
        cursor: pointer;
        transition: all 0.2s;
        text-align: left;
        color: #fff;
    }
    
    .chapter-item:hover {
        background: rgba(102, 51, 153, 0.2);
        border-color: #663399;
    }
    
    .chapter-number {
        font-weight: 600;
        color: #663399;
        min-width: 80px;
    }
    
    .chapter-title {
        flex: 1;
        color: #ddd;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    
    .chapter-date {
        color: #666;
        font-size: 0.85rem;
    }
    
    .btn-show-more {
        width: 100%;
        margin-top: 15px;
        padding: 12px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid #444;
        border-radius: 10px;
        color: #aaa;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-show-more:hover {
        background: rgba(255, 255, 255, 0.1);
        border-color: #666;
    }
    
    @media (max-width: 768px) {
        .manga-hero {
            flex-direction: column;
            align-items: center;
            text-align: center;
        }
        
        .manga-cover {
            width: 180px;
        }
        
        .manga-info {
            align-items: center;
        }
        
        .manga-title {
            font-size: 1.5rem;
        }
    }
</style>
