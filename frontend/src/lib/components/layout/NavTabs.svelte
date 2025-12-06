<script>
    /**
     * NavTabs - Barra de navegação por abas (Animes, Mangás, Amigos, Comunidade)
     */
    
    /** @type {{ activeTab?: string, discordLinked?: boolean, friendsCount?: number, onTabChange?: Function }} */
    let { 
        activeTab = 'anime',
        discordLinked = false,
        friendsCount = 0,
        onTabChange = () => {}
    } = $props();
    
    const tabs = [
        { id: 'anime', icon: '🎬', label: 'Animes' },
        { id: 'manga', icon: '📚', label: 'Mangás', badge: 'Em breve', badgeType: 'soon' },
        { id: 'friends', icon: '👥', label: 'Amigos' },
        { id: 'community', icon: '🌐', label: 'Comunidade', badge: 'Em breve', badgeType: 'soon' }
    ];
    
    function switchTab(tabId) {
        onTabChange(tabId);
    }
</script>

<div class="nav-tabs-container">
    <div class="nav-tabs">
        {#each tabs as tab}
            <button 
                type="button" 
                class="nav-tab {activeTab === tab.id ? 'active' : ''}"
                onclick={() => switchTab(tab.id)}
            >
                <span class="tab-icon">{tab.icon}</span>
                <span class="tab-text">{tab.label}</span>
                {#if tab.badge}
                    <span class="tab-badge {tab.badgeType}">{tab.badge}</span>
                {:else if tab.id === 'friends' && discordLinked && friendsCount > 0}
                    <span class="tab-badge notify">{friendsCount}</span>
                {/if}
            </button>
        {/each}
    </div>
</div>

<style>
    .nav-tabs-container {
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 15px 20px;
        background: #0f142d;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        position: sticky;
        top: 0;
        z-index: 50;
    }
    
    .nav-tabs {
        display: flex;
        gap: 8px;
    }
    
    .nav-tab {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 20px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 25px;
        color: #888;
        font-size: 0.95rem;
        cursor: pointer;
        transition: all 0.25s ease;
    }
    
    .nav-tab:hover {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
        transform: translateY(-2px);
    }
    
    .nav-tab.active {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border-color: transparent;
        color: #fff;
        box-shadow: 0 4px 15px rgba(245, 87, 108, 0.4);
    }
    
    .tab-icon {
        font-size: 1.1rem;
    }
    
    .tab-text {
        font-weight: 500;
    }
    
    .tab-badge {
        font-size: 0.7rem;
        padding: 2px 8px;
        border-radius: 10px;
        font-weight: 600;
    }
    
    .tab-badge.soon {
        background: rgba(255, 255, 255, 0.15);
        color: #888;
    }
    
    .tab-badge.notify {
        background: #f5576c;
        color: #fff;
    }
    
    @media (max-width: 768px) {
        .nav-tabs-container {
            padding: 10px;
        }
        
        .nav-tabs {
            gap: 4px;
        }
        
        .nav-tab {
            padding: 8px 12px;
            font-size: 0.85rem;
        }
        
        .tab-text {
            display: none;
        }
    }
</style>
