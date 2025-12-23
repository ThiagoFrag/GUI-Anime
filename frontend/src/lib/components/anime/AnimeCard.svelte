<script>
    /**
     * AnimeCard - Card de anime reutilizável
     * Exibe imagem, título e badge de favorito
     */
    
    /** @type {{ anime: Object, isFavorite?: boolean, showFavorite?: boolean, onSelect?: Function, onToggleFavorite?: Function, onHover?: Function }} */
    let { 
        anime,
        isFavorite = false,
        showFavorite = true,
        onSelect = () => {},
        onToggleFavorite = () => {},
        onHover = () => {}
    } = $props();
    
    function handleClick() {
        onSelect(anime);
    }
    
    function handleFavorite(e) {
        e.stopPropagation();
        onToggleFavorite(anime);
    }
    
    function handleMouseEnter() {
        onHover(anime);
    }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div 
    class="anime-card" 
    onclick={handleClick}
    onmouseenter={handleMouseEnter}
>
    {#if anime.Image || anime.image}
        <img src={anime.Image || anime.image} alt={anime.Title || anime.title} loading="lazy" />
    {:else}
        <div class="no-image">📺</div>
    {/if}
    
    <div class="anime-info">
        <div class="anime-title">{anime.Title || anime.title}</div>
        {#if anime.score}
            <div class="anime-score">⭐ {anime.score}%</div>
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
        <span class="play-icon">▶</span>
    </div>
</div>

<style>
    .anime-card {
        position: relative;
        border-radius: 12px;
        overflow: hidden;
        background: #1a1d2e;
        cursor: pointer;
        transition: all 0.3s ease;
        aspect-ratio: 2/3;
    }
    
    .anime-card:hover {
        transform: translateY(-8px) scale(1.02);
        box-shadow: 0 15px 40px rgba(0, 0, 0, 0.4);
    }
    
    .anime-card img {
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
    
    .anime-info {
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        padding: 40px 12px 12px;
        background: linear-gradient(to top, rgba(0,0,0,0.9) 0%, transparent 100%);
    }
    
    .anime-title {
        font-size: 0.9rem;
        font-weight: 600;
        color: #fff;
        line-height: 1.3;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
    }
    
    .anime-score {
        font-size: 0.75rem;
        color: #ffd700;
        margin-top: 4px;
    }
    
    .btn-fav {
        position: absolute;
        top: 8px;
        right: 8px;
        width: 32px;
        height: 32px;
        background: rgba(0, 0, 0, 0.6);
        border: none;
        border-radius: 50%;
        font-size: 1rem;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
        z-index: 2;
    }
    
    .btn-fav:hover {
        background: rgba(245, 87, 108, 0.8);
        transform: scale(1.1);
    }
    
    .btn-fav.active {
        background: rgba(245, 87, 108, 0.9);
    }
    
    .card-overlay {
        position: absolute;
        inset: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.3s;
    }
    
    .anime-card:hover .card-overlay {
        opacity: 1;
    }
    
    .play-icon {
        width: 50px;
        height: 50px;
        background: rgba(245, 87, 108, 0.9);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1.2rem;
        color: #fff;
        transform: scale(0.8);
        transition: transform 0.3s;
    }
    
    .anime-card:hover .play-icon {
        transform: scale(1);
    }
</style>
