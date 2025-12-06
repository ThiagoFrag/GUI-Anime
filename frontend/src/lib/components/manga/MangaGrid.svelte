<script>
    /**
     * MangaGrid - Grid responsivo de cards de mangá
     */
    import MangaCard from './MangaCard.svelte';
    
    /** @type {{ mangas: Array, favorites?: Set|Array, showFavorite?: boolean, emptyMessage?: string, onSelect?: Function, onToggleFavorite?: Function, onHover?: Function }} */
    let { 
        mangas = [],
        favorites = [],
        showFavorite = true,
        emptyMessage = 'Nenhum mangá encontrado',
        onSelect = () => {},
        onToggleFavorite = () => {},
        onHover = () => {}
    } = $props();
    
    // Converte para Set se for array
    let favoritesSet = $derived(
        favorites instanceof Set ? favorites : new Set(favorites.map(f => f.url || f.URL))
    );
    
    function isFavorite(manga) {
        return favoritesSet.has(manga.url || manga.URL);
    }
</script>

{#if mangas.length === 0}
    <div class="empty-state">
        <p>{emptyMessage}</p>
    </div>
{:else}
    <div class="manga-grid">
        {#each mangas as manga (manga.url || manga.URL || manga.id || manga.title)}
            <MangaCard 
                {manga}
                isFavorite={isFavorite(manga)}
                {showFavorite}
                {onSelect}
                {onToggleFavorite}
                {onHover}
            />
        {/each}
    </div>
{/if}

<style>
    .manga-grid {
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
        .manga-grid {
            grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
        }
    }
    
    @media (min-width: 1200px) {
        .manga-grid {
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 25px;
        }
    }
</style>
