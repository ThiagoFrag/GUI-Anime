<script>
    /**
     * MangaCard - Card de mangá reutilizável
     * Exibe imagem, título e último capítulo
     */
    
    /** @type {{ manga: Object, isFavorite?: boolean, showFavorite?: boolean, onSelect?: Function, onToggleFavorite?: Function, onHover?: Function }} */
    let { 
        manga,
        isFavorite = false,
        showFavorite = true,
        onSelect = () => {},
        onToggleFavorite = () => {},
        onHover = () => {}
    } = $props();
    
    function handleClick() {
        onSelect(manga);
    }
    
    function handleFavorite(e) {
        e.stopPropagation();
        onToggleFavorite(manga);
    }
    
    function handleMouseEnter() {
        onHover(manga);
    }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div 
    class="manga-card" 
    onclick={handleClick}
    onmouseenter={handleMouseEnter}
>
    {#if manga.image}
        <img src={manga.image} alt={manga.title} loading="lazy" />
    {:else}
        <div class="no-image">📚</div>
    {/if}
    
    <div class="manga-info">
        <div class="manga-title">{manga.title}</div>
        {#if manga.latestChapter}
            <div class="manga-chapter">📖 {manga.latestChapter}</div>
        {/if}
    </div>
    
    {#if showFavorite}
        <button 
            type="button" 
            class="btn-fav {isFavorite ? 'active' : ''}"
            onclick={handleFavorite}
            title={isFavorite ? 'Remover dos favoritos' : 'Adicionar aos favoritos'}
        >
            {isFavorite ? '⭐' : '☆'}
        </button>
    {/if}
    
    <div class="card-overlay">
        <span class="read-icon">📖</span>
    </div>
</div>

<style>
    .manga-card {
        position: relative;
        border-radius: 12px;
        overflow: hidden;
        background: #1a1d2e;
        cursor: pointer;
        transition: all 0.3s ease;
        aspect-ratio: 2/3;
    }
    
    .manga-card:hover {
        transform: translateY(-8px) scale(1.02);
        box-shadow: 0 15px 40px rgba(0, 0, 0, 0.4);
    }
    
    .manga-card img {
        width: 100%;
        height: 100%;
        object-fit: cover;
    }
    
    .no-image {
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 3rem;
        background: linear-gradient(135deg, #1a1d2e 0%, #2a2d3e 100%);
    }
    
    .manga-info {
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        padding: 40px 12px 12px;
        background: linear-gradient(to top, rgba(0,0,0,0.9) 0%, rgba(0,0,0,0.5) 70%, transparent 100%);
    }
    
    .manga-title {
        font-size: 0.9rem;
        font-weight: 600;
        color: #fff;
        overflow: hidden;
        text-overflow: ellipsis;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        line-height: 1.3;
    }
    
    .manga-chapter {
        font-size: 0.75rem;
        color: #aaa;
        margin-top: 4px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    
    .btn-fav {
        position: absolute;
        top: 10px;
        right: 10px;
        background: rgba(0, 0, 0, 0.6);
        border: none;
        border-radius: 50%;
        width: 32px;
        height: 32px;
        font-size: 1rem;
        cursor: pointer;
        transition: all 0.2s;
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 2;
    }
    
    .btn-fav:hover {
        background: rgba(0, 0, 0, 0.8);
        transform: scale(1.1);
    }
    
    .btn-fav.active {
        background: rgba(255, 193, 7, 0.3);
    }
    
    .card-overlay {
        position: absolute;
        inset: 0;
        background: rgba(102, 51, 153, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.3s;
    }
    
    .manga-card:hover .card-overlay {
        opacity: 1;
    }
    
    .read-icon {
        font-size: 2.5rem;
        color: white;
        text-shadow: 0 2px 10px rgba(0,0,0,0.3);
    }
</style>
