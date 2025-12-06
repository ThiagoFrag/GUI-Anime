<script>
    /**
     * FeaturedHero - Seção hero com anime em destaque (estilo Netflix)
     */
    
    /** @type {{ anime: Object|null, trendingAnimes?: Array, currentIndex?: number, loading?: boolean, onWatch?: Function, onSelectFeatured?: Function }} */
    let { 
        anime = null,
        trendingAnimes = [],
        currentIndex = 0,
        loading = false,
        onWatch = () => {},
        onSelectFeatured = () => {}
    } = $props();
    
    // Filtra animes com banner para os dots de navegação
    let animesWithBanners = $derived(
        trendingAnimes.slice(0, 8).filter(a => a.banner)
    );
</script>

{#if anime && anime.banner}
    {#key anime.id || anime.title}
        <div 
            class="featured-hero" 
            style="--banner-url: url({anime.banner}); --accent-color: {anime.color || '#f5576c'}"
        >
            <div class="featured-overlay"></div>
            <div class="featured-content">
                <div class="featured-info">
                    <div class="featured-badges">
                        {#if anime.isAiring}
                            <span class="badge airing">🔴 EM EXIBIÇÃO</span>
                        {/if}
                        {#if anime.score}
                            <span class="badge score">⭐ {anime.score}%</span>
                        {/if}
                        {#if anime.episodes}
                            <span class="badge episodes">{anime.episodes} eps</span>
                        {/if}
                    </div>
                    <h1 class="featured-title">{anime.title}</h1>
                    <p class="featured-meta">
                        {anime.genres?.slice(0, 3).join(' • ') || ''}
                        {#if anime.studio} • {anime.studio}{/if}
                        {#if anime.year} • {anime.year}{/if}
                    </p>
                    {#if anime.description}
                        <p class="featured-desc">
                            {anime.description?.slice(0, 180)}{anime.description?.length > 180 ? '...' : ''}
                        </p>
                    {/if}
                    <div class="featured-actions">
                        <button type="button" class="btn-featured-play" onclick={() => onWatch(anime)}>
                            ▶ Assistir
                        </button>
                        {#if anime.trailerUrl}
                            <a href={anime.trailerUrl} target="_blank" rel="noopener" class="btn-featured-trailer">
                                🎬 Trailer
                            </a>
                        {/if}
                    </div>
                </div>
                <div class="featured-poster">
                    <img src={anime.image} alt={anime.title} loading="eager" />
                </div>
            </div>
            
            <!-- Navigation Dots -->
            {#if animesWithBanners.length > 1}
                <div class="featured-nav">
                    {#each animesWithBanners as navAnime, i}
                        <button 
                            type="button"
                            class="nav-dot {i === currentIndex ? 'active' : ''}"
                            onclick={() => onSelectFeatured(i)}
                            title={navAnime.title}
                        ></button>
                    {/each}
                </div>
            {/if}
        </div>
    {/key}
{:else if loading}
    <!-- Loading skeleton -->
    <div class="featured-skeleton">
        <div class="skeleton-shimmer"></div>
    </div>
{:else}
    <!-- Fallback Hero -->
    <div class="hero-section-modern">
        <div class="hero-bg-effects">
            <div class="hero-gradient"></div>
            <div class="hero-grid-pattern"></div>
        </div>
        <div class="hero-content-centered">
            <div class="hero-logo">
                <span class="hero-emoji">🎬</span>
                <h1 class="hero-brand">
                    <span class="brand-go">Go</span><span class="brand-anime">Anime</span>
                </h1>
            </div>
            <p class="hero-tagline">Assista seus animes favoritos em HD</p>
            <div class="hero-stats">
                <div class="stat">
                    <span class="stat-number">10K+</span>
                    <span class="stat-label">Animes</span>
                </div>
                <div class="stat-divider"></div>
                <div class="stat">
                    <span class="stat-number">HD</span>
                    <span class="stat-label">Qualidade</span>
                </div>
                <div class="stat-divider"></div>
                <div class="stat">
                    <span class="stat-number">24/7</span>
                    <span class="stat-label">Disponível</span>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    .featured-hero {
        position: relative;
        min-height: 500px;
        background-image: var(--banner-url);
        background-size: cover;
        background-position: center 20%;
        display: flex;
        align-items: flex-end;
        padding: 60px 40px;
    }
    
    .featured-overlay {
        position: absolute;
        inset: 0;
        background: linear-gradient(
            to right,
            rgba(10, 11, 15, 0.95) 0%,
            rgba(10, 11, 15, 0.7) 50%,
            rgba(10, 11, 15, 0.3) 100%
        );
    }
    
    .featured-content {
        position: relative;
        z-index: 1;
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        width: 100%;
        max-width: 1400px;
        margin: 0 auto;
        gap: 40px;
    }
    
    .featured-info {
        flex: 1;
        max-width: 600px;
    }
    
    .featured-badges {
        display: flex;
        gap: 10px;
        margin-bottom: 15px;
    }
    
    .badge {
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 0.75rem;
        font-weight: 600;
        text-transform: uppercase;
    }
    
    .badge.airing {
        background: rgba(255, 0, 0, 0.3);
        color: #ff6b6b;
        border: 1px solid rgba(255, 0, 0, 0.3);
    }
    
    .badge.score {
        background: rgba(255, 215, 0, 0.2);
        color: #ffd700;
    }
    
    .badge.episodes {
        background: rgba(255, 255, 255, 0.1);
        color: #aaa;
    }
    
    .featured-title {
        font-size: 2.8rem;
        font-weight: 800;
        margin: 0 0 10px 0;
        line-height: 1.1;
        color: #fff;
    }
    
    .featured-meta {
        color: #888;
        font-size: 1rem;
        margin: 0 0 15px 0;
    }
    
    .featured-desc {
        color: #aaa;
        font-size: 0.95rem;
        line-height: 1.6;
        margin: 0 0 25px 0;
    }
    
    .featured-actions {
        display: flex;
        gap: 15px;
    }
    
    .btn-featured-play {
        padding: 14px 35px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border: none;
        border-radius: 30px;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s;
    }
    
    .btn-featured-play:hover {
        transform: translateY(-3px);
        box-shadow: 0 10px 30px rgba(245, 87, 108, 0.4);
    }
    
    .btn-featured-trailer {
        padding: 14px 25px;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        border-radius: 30px;
        color: #fff;
        font-size: 1rem;
        text-decoration: none;
        transition: all 0.3s;
    }
    
    .btn-featured-trailer:hover {
        background: rgba(255, 255, 255, 0.2);
    }
    
    .featured-poster {
        flex-shrink: 0;
        width: 280px;
        border-radius: 15px;
        overflow: hidden;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
    }
    
    .featured-poster img {
        width: 100%;
        height: auto;
        display: block;
    }
    
    .featured-nav {
        position: absolute;
        bottom: 30px;
        left: 50%;
        transform: translateX(-50%);
        display: flex;
        gap: 8px;
    }
    
    .nav-dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.3);
        border: none;
        cursor: pointer;
        transition: all 0.3s;
    }
    
    .nav-dot:hover {
        background: rgba(255, 255, 255, 0.5);
    }
    
    .nav-dot.active {
        background: #f5576c;
        width: 30px;
        border-radius: 5px;
    }
    
    /* Skeleton */
    .featured-skeleton {
        height: 500px;
        background: #1a1d2e;
        position: relative;
        overflow: hidden;
    }
    
    .skeleton-shimmer {
        position: absolute;
        inset: 0;
        background: linear-gradient(
            90deg,
            transparent 0%,
            rgba(255, 255, 255, 0.05) 50%,
            transparent 100%
        );
        animation: shimmer 1.5s infinite;
    }
    
    @keyframes shimmer {
        0% { transform: translateX(-100%); }
        100% { transform: translateX(100%); }
    }
    
    /* Fallback Hero */
    .hero-section-modern {
        position: relative;
        min-height: 400px;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
    }
    
    .hero-bg-effects {
        position: absolute;
        inset: 0;
    }
    
    .hero-gradient {
        position: absolute;
        inset: 0;
        background: radial-gradient(ellipse at center, rgba(245, 87, 108, 0.15) 0%, transparent 70%);
    }
    
    .hero-grid-pattern {
        position: absolute;
        inset: 0;
        background-image: 
            linear-gradient(rgba(255, 255, 255, 0.02) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
        background-size: 50px 50px;
    }
    
    .hero-content-centered {
        position: relative;
        z-index: 1;
        text-align: center;
    }
    
    .hero-logo {
        margin-bottom: 20px;
    }
    
    .hero-emoji {
        font-size: 4rem;
        display: block;
        margin-bottom: 10px;
    }
    
    .hero-brand {
        font-size: 3rem;
        font-weight: 800;
        margin: 0;
    }
    
    .brand-go {
        color: #fff;
    }
    
    .brand-anime {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .hero-tagline {
        color: #888;
        font-size: 1.2rem;
        margin: 0 0 30px 0;
    }
    
    .hero-stats {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 30px;
    }
    
    .stat {
        text-align: center;
    }
    
    .stat-number {
        display: block;
        font-size: 1.8rem;
        font-weight: 700;
        color: #fff;
    }
    
    .stat-label {
        font-size: 0.85rem;
        color: #666;
    }
    
    .stat-divider {
        width: 1px;
        height: 40px;
        background: rgba(255, 255, 255, 0.1);
    }
    
    @media (max-width: 768px) {
        .featured-hero {
            min-height: 400px;
            padding: 30px 20px;
        }
        
        .featured-content {
            flex-direction: column-reverse;
            align-items: center;
            text-align: center;
        }
        
        .featured-poster {
            width: 180px;
        }
        
        .featured-title {
            font-size: 1.8rem;
        }
        
        .featured-badges {
            justify-content: center;
        }
        
        .featured-actions {
            justify-content: center;
        }
    }
</style>
