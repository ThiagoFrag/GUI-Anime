<script>
    /**
     * LoadingSpinner - Indicador de carregamento
     */
    
    /** @type {{ message?: string, size?: 'small'|'medium'|'large' }} */
    let { 
        message = '',
        size = 'medium'
    } = $props();
</script>

<div class="loading {size}">
    <div class="spinner"></div>
    {#if message}
        <p>{message}</p>
    {/if}
</div>

<style>
    .loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 40px;
        gap: 15px;
    }
    
    .spinner {
        border: 3px solid rgba(255, 255, 255, 0.1);
        border-top-color: #f5576c;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    .small .spinner {
        width: 24px;
        height: 24px;
        border-width: 2px;
    }
    
    .medium .spinner {
        width: 40px;
        height: 40px;
    }
    
    .large .spinner {
        width: 60px;
        height: 60px;
        border-width: 4px;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    .loading p {
        color: #888;
        font-size: 0.95rem;
        margin: 0;
    }
</style>
