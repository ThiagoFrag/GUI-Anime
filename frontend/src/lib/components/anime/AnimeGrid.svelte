<script>
    /**
     * AnimeGrid - Grid responsivo de cards de anime
     */
    import AnimeCard from './AnimeCard.svelte';
    
    /** @type {{ animes: Array, favorites?: Set|Array, showFavorite?: boolean, emptyMessage?: string, onSelect?: Function, onToggleFavorite?: Function, onHover?: Function }} */
    let { 
        animes = [],
        favorites = [],
        showFavorite = true,
        emptyMessage = 'Nenhum anime encontrado',
        onSelect = () => {},
        onToggleFavorite = () => {},
        onHover = () => {}
    } = $props();
    
    // Converte para Set se for array
    let favoritesSet = $derived(
        favorites instanceof Set ? favorites : new Set(favorites.map(f => f.URL || f.url))
    );
    
    function isFavorite(anime) {
        return favoritesSet.has(anime.URL || anime.url);
    }
</script>

{#if animes.length === 0}
    <div class="empty-state">
        <p>{emptyMessage}</p>
    </div>
{:else}
    <div class="anime-grid">
        {#each animes as anime (anime.URL || anime.url || anime.id || anime.title)}
            <AnimeCard 
                {anime}
                isFavorite={isFavorite(anime)}
                {showFavorite}
                {onSelect}
                {onToggleFavorite}
                {onHover}
            />
        {/each}
    </div>
{/if}

<style>
    .anime-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
        gap: 20px;
        padding: 20px 0;
    }
    
    .empty-state {
        text-align: center;
        padding: 60px 20px;
        color: #666;
    }
    
    .empty-state p {
        font-size: 1.1rem;
    }
    
    @media (min-width: 768px) {
        .anime-grid {
            grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
        }
    }
    
    @media (min-width: 1200px) {
        .anime-grid {
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 25px;
        }
    }
</style>
