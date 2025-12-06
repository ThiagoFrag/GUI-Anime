<script>
    /**
     * GenreFilter - Filtro por gêneros de anime
     */
    import { ANIME_GENRES } from '../../constants/genres.js';
    
    /** @type {{ selectedGenre?: Object|null, onSelectGenre?: Function, onClear?: Function }} */
    let { 
        selectedGenre = null,
        onSelectGenre = () => {},
        onClear = () => {}
    } = $props();
</script>

<div class="genre-filter">
    <div class="genre-header">
        <h3>🎭 Gêneros</h3>
        {#if selectedGenre}
            <button type="button" class="btn-clear" onclick={onClear}>
                ✕ Limpar filtro
            </button>
        {/if}
    </div>
    
    <div class="genre-grid">
        {#each ANIME_GENRES as genre}
            <button 
                type="button"
                class="genre-chip {selectedGenre?.id === genre.id ? 'active' : ''}"
                onclick={() => onSelectGenre(genre)}
            >
                <span class="genre-icon">{genre.icon}</span>
                <span class="genre-name">{genre.name}</span>
            </button>
        {/each}
    </div>
</div>

<style>
    .genre-filter {
        padding: 20px 0;
    }
    
    .genre-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 15px;
    }
    
    .genre-header h3 {
        font-size: 1.1rem;
        font-weight: 600;
        color: #fff;
        margin: 0;
    }
    
    .btn-clear {
        background: rgba(245, 87, 108, 0.2);
        border: 1px solid rgba(245, 87, 108, 0.3);
        color: #f5576c;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 0.8rem;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-clear:hover {
        background: rgba(245, 87, 108, 0.3);
    }
    
    .genre-grid {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }
    
    .genre-chip {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 8px 14px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        color: #aaa;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .genre-chip:hover {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
        transform: translateY(-2px);
    }
    
    .genre-chip.active {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border-color: transparent;
        color: #fff;
    }
    
    .genre-icon {
        font-size: 1rem;
    }
    
    .genre-name {
        font-weight: 500;
    }
</style>
