<script>
    /**
     * SearchBar - Barra de busca com busca inteligente
     */
    import { debounce } from '../../utils/helpers.js';
    
    /** @type {{ value?: string, placeholder?: string, loading?: boolean, onSearch?: Function, onClear?: Function }} */
    let { 
        value = $bindable(''),
        placeholder = 'Buscar animes...',
        loading = false,
        onSearch = () => {},
        onClear = () => {}
    } = $props();
    
    const debouncedSearch = debounce(() => {
        if (value.trim()) {
            onSearch(value);
        }
    }, 300);
    
    function handleInput() {
        debouncedSearch();
    }
    
    function handleKeydown(e) {
        if (e.key === 'Enter' && value.trim()) {
            onSearch(value);
        }
    }
    
    function clear() {
        value = '';
        onClear();
    }
</script>

<div class="search-bar">
    <div class="search-input-wrapper">
        <span class="search-icon">🔍</span>
        <input 
            type="text"
            bind:value={value}
            {placeholder}
            class="search-input"
            oninput={handleInput}
            onkeydown={handleKeydown}
        />
        {#if value}
            <button type="button" class="search-clear" onclick={clear}>
                ✕
            </button>
        {/if}
        {#if loading}
            <div class="search-spinner"></div>
        {/if}
    </div>
</div>

<style>
    .search-bar {
        width: 100%;
        max-width: 600px;
        margin: 0 auto;
    }
    
    .search-input-wrapper {
        position: relative;
        display: flex;
        align-items: center;
    }
    
    .search-icon {
        position: absolute;
        left: 16px;
        font-size: 1.1rem;
        opacity: 0.6;
        pointer-events: none;
    }
    
    .search-input {
        width: 100%;
        padding: 14px 50px 14px 48px;
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 30px;
        color: #fff;
        font-size: 1rem;
        outline: none;
        transition: all 0.25s ease;
    }
    
    .search-input::placeholder {
        color: #666;
    }
    
    .search-input:focus {
        background: rgba(255, 255, 255, 0.12);
        border-color: rgba(245, 87, 108, 0.5);
        box-shadow: 0 0 20px rgba(245, 87, 108, 0.2);
    }
    
    .search-clear {
        position: absolute;
        right: 16px;
        width: 24px;
        height: 24px;
        background: rgba(255, 255, 255, 0.2);
        border: none;
        border-radius: 50%;
        color: #fff;
        font-size: 0.75rem;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: background 0.2s;
    }
    
    .search-clear:hover {
        background: rgba(245, 87, 108, 0.5);
    }
    
    .search-spinner {
        position: absolute;
        right: 50px;
        width: 18px;
        height: 18px;
        border: 2px solid rgba(255, 255, 255, 0.2);
        border-top-color: #f5576c;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
</style>
