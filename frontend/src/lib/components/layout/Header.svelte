<script>
    /**
     * Header - Barra de navegação principal do app
     * Exibe logo, título e menu do usuário
     */
    
    /** @type {{ user: Object, minimal?: boolean, onHome?: Function, onFavorites?: Function, onHistory?: Function, onSettings?: Function }} */
    let { 
        user, 
        minimal = false,
        onHome = () => {},
        onFavorites = () => {},
        onHistory = () => {},
        onSettings = () => {}
    } = $props();
    
    let userMenuOpen = $state(false);
    
    function handleClickOutside(e) {
        if (userMenuOpen && !e.target.closest('.user-menu-container')) {
            userMenuOpen = false;
        }
    }
    
    function toggleMenu(e) {
        e.stopPropagation();
        userMenuOpen = !userMenuOpen;
    }
    
    function selectOption(handler) {
        userMenuOpen = false;
        handler();
    }
    
    $effect(() => {
        document.addEventListener('click', handleClickOutside);
        return () => document.removeEventListener('click', handleClickOutside);
    });
</script>

<header class="header {minimal ? 'minimal' : ''}">
    <div class="header-left">
        <button type="button" class="btn-logo" onclick={onHome}>
            <span class="logo-icon-small">🎬</span>
            <span class="logo-text-small">
                <span class="go">Go</span><span class="anime">Anime</span>
            </span>
        </button>
    </div>
    
    <!-- USER MENU -->
    <div class="user-menu-container">
        <button 
            type="button" 
            class="user-section" 
            onclick={toggleMenu}
        >
            <span class="user-avatar">👤</span>
            <span class="user-name">{user?.username || 'Usuário'}</span>
            <span class="menu-arrow">{userMenuOpen ? '▲' : '▼'}</span>
        </button>
        
        {#if userMenuOpen}
            <div class="user-dropdown">
                <button type="button" class="dropdown-item" onclick={() => selectOption(onFavorites)}>
                    ⭐ Favoritos
                </button>
                <button type="button" class="dropdown-item" onclick={() => selectOption(onHistory)}>
                    🕐 Últimos Assistidos
                </button>
                <button type="button" class="dropdown-item" onclick={() => selectOption(onSettings)}>
                    ⚙️ Configurações
                </button>
            </div>
        {/if}
    </div>
</header>

<style>
    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 15px 30px;
        background: #14151a;
        border-bottom: 1px solid #333;
        z-index: 100;
    }

    .header.minimal {
        position: absolute;
        top: 0;
        right: 0;
        background: transparent;
        border: none;
        padding: 20px 40px;
    }

    .header-left {
        display: flex;
        align-items: center;
    }
    
    .btn-logo {
        display: flex;
        align-items: center;
        gap: 8px;
        background: none;
        border: none;
        cursor: pointer;
        padding: 5px 10px;
        border-radius: 8px;
        transition: background 0.2s;
    }
    
    .btn-logo:hover {
        background: rgba(255, 255, 255, 0.1);
    }
    
    .logo-icon-small {
        font-size: 1.5rem;
    }
    
    .logo-text-small {
        font-size: 1.2rem;
        font-weight: 700;
    }
    
    .logo-text-small .go {
        color: #fff;
    }
    
    .logo-text-small .anime {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .user-section {
        color: #fff;
        display: flex;
        align-items: center;
        gap: 10px;
        background: #1a1f3a;
        padding: 8px 16px;
        border-radius: 25px;
        cursor: pointer;
        border: 1px solid rgba(255, 255, 255, 0.15);
        transition: all 0.2s;
    }

    .user-section:hover {
        background: #252b4d;
        border-color: rgba(245, 87, 108, 0.3);
    }
    
    .user-avatar {
        font-size: 1.2rem;
    }
    
    .user-name {
        font-size: 0.95rem;
        font-weight: 500;
    }

    .menu-arrow {
        font-size: 0.7rem;
        opacity: 0.7;
    }

    .user-menu-container {
        position: relative;
    }

    .user-dropdown {
        position: absolute;
        top: calc(100% + 10px);
        right: 0;
        background: rgba(26, 31, 58, 0.98);
        border: 1px solid #333;
        border-radius: 12px;
        min-width: 200px;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
        overflow: hidden;
        z-index: 1000;
        animation: slideDown 0.2s ease-out;
    }

    @keyframes slideDown {
        from { opacity: 0; transform: translateY(-10px); }
        to { opacity: 1; transform: translateY(0); }
    }

    .dropdown-item {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 12px 20px;
        color: #fff;
        background: transparent;
        border: none;
        width: 100%;
        text-align: left;
        cursor: pointer;
        transition: background 0.2s;
        font-size: 0.95rem;
    }

    .dropdown-item:hover {
        background: rgba(245, 87, 108, 0.2);
    }
</style>
