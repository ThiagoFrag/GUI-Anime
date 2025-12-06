<script>
    import { onMount, onDestroy } from 'svelte';
    import SimplePlayer from './SimplePlayer.svelte';
    import { 
        GetCurrentUser, CreateUser, BuscarAnimes, BuscarAnimesMulti, GetTopAnimes, GetAnimeURL, 
        GetEpisodes, GetEpisodesForSource, PlayAnime, 
        GetStreamURLForEpisode, AssistirEpisodio, GetProxyURLForVideo,
        GetFavorites, AddToFavorites, RemoveFromFavorites, IsFavorite,
        GetWatchHistory, AddToWatchHistory, GetSettings, SaveSettings,
        ExportUserData, ImportUserData,
        GetTrendingAnimes, GetPopularAnimes, SearchAniList, GetAnimeHDImage,
        ClearEpisodesCache, ClearAllCache, GetCacheStats, ResetSourceFailures,
        GetDiscordStatus, SimulateDiscordConnect, DisconnectDiscord,
        GetDiscordRecommendations, SendDiscordRecommendation, LikeDiscordRecommendation,
        StartDiscordOAuth, GetDiscordUser, DisconnectDiscordUser,
        GetSkipTimes,
        // Discord Linking System
        GetDiscordLinkStatus, LinkDiscordWithCode, UnlinkDiscord,
        GetDiscordServerInvite, UpdateDiscordWatchingStatus, GetDiscordFriendsActivity,
        SetDiscordShowStatus, SetDiscordShareAnimes
    } from '../wailsjs/go/main/App';
    import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';

    // === GLOBAL STATES ===
    let usuario = null;
    let nomeInput = "";
    let avatarSelecionado = "avatar1.png";
    
    // Home - Enhanced with AniList
    let topAnimes = [];
    let trendingAnimes = [];
    let featuredAnime = null;
    let featuredIndex = 0; // Index for rotating featured anime
    let featuredInterval = null; // Interval for auto-rotation
    let resultadosBusca = [];
    let carregando = false;
    let termoBusca = "";
    
    // === NAVIGATION TABS ===
    let activeTab = 'anime'; // 'anime' | 'manga' | 'community' | 'friends'
    
    // === DISCORD INTEGRATION (Vinculação por Código) ===
    let discordLinked = false;
    let discordLinkInfo = null;  // { username, userId, avatar, linkedAt, showStatus, shareAnimes }
    let friendsActivity = [];    // Atividade dos amigos
    let loadingFriends = false;
    
    // === DISCORD LINKING UI ===
    let showLinkModal = false;
    let linkCode = "";
    let linkError = "";
    let linkLoading = false;
    let discordServerInvite = "";
    
    // === COMMUNITY ===
    let communityAnimes = [];
    let loadingCommunity = false;
    
    // Gêneros com termos de busca otimizados (animes populares de cada gênero)
    let selectedGenre = null;
    const animeGenres = [
        { id: 'action', name: 'Ação', icon: '⚔️', searchTerms: ['naruto', 'bleach', 'attack on titan', 'demon slayer'] },
        { id: 'adventure', name: 'Aventura', icon: '🗺️', searchTerms: ['one piece', 'hunter x hunter', 'made in abyss'] },
        { id: 'comedy', name: 'Comédia', icon: '😂', searchTerms: ['konosuba', 'gintama', 'kaguya-sama'] },
        { id: 'drama', name: 'Drama', icon: '🎭', searchTerms: ['your lie in april', 'clannad', 'violet evergarden'] },
        { id: 'fantasy', name: 'Fantasia', icon: '✨', searchTerms: ['frieren', 'mushoku tensei', 're:zero'] },
        { id: 'horror', name: 'Terror', icon: '👻', searchTerms: ['junji ito', 'another', 'parasyte', 'hellsing'] },
        { id: 'mystery', name: 'Mistério', icon: '🔍', searchTerms: ['death note', 'monster', 'steins gate'] },
        { id: 'romance', name: 'Romance', icon: '💕', searchTerms: ['toradora', 'horimiya', 'my dress up darling'] },
        { id: 'sci-fi', name: 'Sci-Fi', icon: '🚀', searchTerms: ['cyberpunk', 'psycho-pass', 'ghost in the shell'] },
        { id: 'slice-of-life', name: 'Slice of Life', icon: '🌸', searchTerms: ['bocchi', 'spy x family', 'k-on'] },
        { id: 'sports', name: 'Esportes', icon: '⚽', searchTerms: ['haikyuu', 'blue lock', 'kuroko no basket'] },
        { id: 'supernatural', name: 'Sobrenatural', icon: '👁️', searchTerms: ['jujutsu kaisen', 'mob psycho', 'noragami'] },
        { id: 'thriller', name: 'Thriller', icon: '😱', searchTerms: ['death note', 'terror', 'zankyou no terror'] },
        { id: 'isekai', name: 'Isekai', icon: '🌀', searchTerms: ['solo leveling', 'overlord', 'sword art online', 'that time'] },
        { id: 'mecha', name: 'Mecha', icon: '🤖', searchTerms: ['gundam', 'code geass', 'evangelion', 'gurren lagann'] },
        { id: 'shounen', name: 'Shounen', icon: '💪', searchTerms: ['dragon ball', 'my hero academia', 'black clover'] },
    ];
    
    // Episodes & Playback
    let selectedAnime = null;
    let episodes = [];
    let seasons = [];
    let selectedSeason = 1;
    let selectedEpisodeURL = "";
    let currentPlayingEpisodeTitle = "";
    
    // Source selection (para animes com múltiplas fontes)
    let availableSources = [];
    let selectedSource = null;
    let showSourceSelector = false;
    
    // UI States
    let episodeSelectionScreen = false;
    let loadingEpisodes = false;
    let playingEpisodeNatively = false;
    let appReady = false; // Para animação de entrada
    
    // Player
    let playerUrl = "";
    let originalStreamUrl = "";
    let videoEl = null;
    let hlsInstance = null;
    
    // Skip Intro/Outro (AniSkip)
    let currentSkipTimes = null;
    let currentMalID = 0;
    let currentEpisodeNumber = 1;
    
    // Cache para melhorar performance
    let episodeCache = new Map();
    let urlCache = new Map();
    let prefetchedAnimes = new Set(); // Animes que já foram prefetched
    
    // Status das fontes de vídeo (cache inteligente)
    let cacheStats = null;
    let showCacheStatus = false;
    
    // Splash screen / Loading
    let showSplash = true;
    let splashProgress = 0;
    let splashStatus = "Iniciando...";

    // === SCROLL & NAVIGATION ===
    let mainContentEl = null;
    let _searchTimeout = null;
    let _prefetchTimeout = null;

    function scheduleSearch(delay = 300) {
        if (_searchTimeout) clearTimeout(_searchTimeout);
        _searchTimeout = setTimeout(() => pesquisar(), delay);
    }
    
    function scrollToTop(smooth = true) {
        if (mainContentEl) {
            mainContentEl.scrollTo({ top: 0, behavior: smooth ? 'smooth' : 'instant' });
        }
    }

    // Prefetch de dados quando o usuário passa o mouse (melhora UX)
    function schedulePrefetch(anime) {
        if (_prefetchTimeout) clearTimeout(_prefetchTimeout);
        _prefetchTimeout = setTimeout(() => prefetchAnimeData(anime), 200);
    }

    async function prefetchAnimeData(anime) {
        if (!anime || !anime.Title) return;
        const key = anime.URL || anime.Title;
        if (prefetchedAnimes.has(key)) return;
        
        try {
            // Prefetch URL se não existir
            if (!anime.URL && !urlCache.has(anime.Title)) {
                const url = await GetAnimeURL(anime.Title);
                if (url) urlCache.set(anime.Title, url);
            }
            prefetchedAnimes.add(key);
            console.log('[prefetch] Dados pré-carregados para:', anime.Title);
        } catch (err) {
            // Silently fail - prefetch is optional
        }
    }

    // === FEATURED ANIME ROTATION ===
    function startFeaturedRotation() {
        if (featuredInterval) clearInterval(featuredInterval);
        featuredInterval = setInterval(() => {
            if (trendingAnimes.length > 0) {
                featuredIndex = (featuredIndex + 1) % Math.min(trendingAnimes.length, 10);
                featuredAnime = trendingAnimes[featuredIndex];
            }
        }, 8000); // Rotate every 8 seconds
    }
    
    function selectFeatured(index) {
        featuredIndex = index;
        featuredAnime = trendingAnimes[index];
        // Reset interval to give more time on manual selection
        startFeaturedRotation();
    }

    // === MENU DO UTILIZADOR ===
    let userMenuOpen = false;
    let currentView = 'home';
    let favorites = [];
    let watchHistory = [];
    let settings = {
        start_fullscreen: false,
        content_language: 'all',
        default_quality: 'auto',
        use_anime4k: true
    };
    let importJsonText = "";
    let exportJsonText = "";
    let showImportExport = false;

    onMount(() => {
        // Fast splash - mostra por tempo mínimo enquanto carrega
        showSplash = true;
        splashProgress = 0;
        splashStatus = "Iniciando...";
        
        // Registra event listeners do Discord
        EventsOn('discord:linked', handleDiscordLinked);
        EventsOn('discord:unlinked', handleDiscordUnlinked);
        
        // Progresso rápido (100ms intervals)
        const progressInterval = setInterval(() => {
            if (splashProgress < 95) {
                splashProgress += 20 + Math.random() * 10;
                if (splashProgress > 95) splashProgress = 95;
            }
        }, 100);
        
        // Tempo mínimo de splash (800ms) para UX suave
        const minSplashTime = Date.now() + 800;
        
        // Inicialização paralela ultra-rápida
        (async () => {
            console.log('[onMount] Iniciando carregamento paralelo...');
            
            // Carrega tudo em paralelo
            const [userResult, dataResult] = await Promise.allSettled([
                (async () => {
                    try {
                        const user = await GetCurrentUser();
                        if (user && user.username) {
                            usuario = user;
                            // Settings em background (não bloqueia)
                            loadUserSettings();
                        }
                    } catch (err) {
                        console.error('GetCurrentUser error:', err);
                    }
                })(),
                carregarDados()
            ]);
            
            console.log('[onMount] Carregamento concluído');
            console.log('[onMount] dataResult:', dataResult);
            console.log('[onMount] featuredAnime após carregamento:', featuredAnime);
            console.log('[onMount] trendingAnimes.length:', trendingAnimes.length);
            
            // Carrega estado do Discord em background
            loadDiscordState();
            
            splashProgress = 100;
            splashStatus = "Pronto!";
            clearInterval(progressInterval);
            
            // Espera tempo mínimo de splash para animação suave
            const remaining = minSplashTime - Date.now();
            if (remaining > 0) {
                await new Promise(r => setTimeout(r, remaining));
            }
            
            // Fade out rápido
            showSplash = false;
            appReady = true;
            
            console.log('[onMount] App pronto! featuredAnime:', featuredAnime?.title);
            
            document.addEventListener('click', handleClickOutside);
        })();

        return () => {
            document.removeEventListener('click', handleClickOutside);
            if (featuredInterval) clearInterval(featuredInterval);
            // Remove event listeners do Discord
            EventsOff('discord:linked');
            EventsOff('discord:unlinked');
        };
    });

    function handleClickOutside(e) {
        if (userMenuOpen && !e.target.closest('.user-menu-container')) {
            userMenuOpen = false;
        }
    }

    async function loadUserSettings() {
        try {
            const s = await GetSettings();
            if (s) {
                settings = {
                    start_fullscreen: s.start_fullscreen || false,
                    content_language: s.content_language || 'all',
                    default_quality: s.default_quality || 'auto',
                    use_anime4k: s.use_anime4k !== false
                };
            }
        } catch (err) {
            console.error('GetSettings error:', err);
        }
    }

    async function loadFavorites() {
        try {
            favorites = (await GetFavorites()) || [];
        } catch (err) {
            console.error('GetFavorites error:', err);
            favorites = [];
        }
    }

    async function loadWatchHistory() {
        try {
            watchHistory = (await GetWatchHistory()) || [];
        } catch (err) {
            console.error('GetWatchHistory error:', err);
            watchHistory = [];
        }
    }

    async function toggleFavorite(anime) {
        try {
            const isFav = await IsFavorite(anime.URL);
            if (isFav) {
                await RemoveFromFavorites(anime.URL);
            } else {
                await AddToFavorites(anime);
            }
            await loadFavorites();
        } catch (err) {
            console.error('toggleFavorite error:', err);
        }
    }

    async function saveUserSettings() {
        try {
            await SaveSettings(settings);
            alert('Configurações salvas!');
        } catch (err) {
            console.error('SaveSettings error:', err);
            alert('Erro ao salvar configurações');
        }
    }

    async function exportData() {
        try {
            exportJsonText = await ExportUserData();
            showImportExport = true;
        } catch (err) {
            console.error('ExportUserData error:', err);
            alert('Erro ao exportar dados');
        }
    }

    async function importData() {
        if (!importJsonText.trim()) {
            alert('Cole o JSON de importação');
            return;
        }
        try {
            await ImportUserData(importJsonText);
            const user = await GetCurrentUser();
            if (user) {
                usuario = user;
                await loadUserSettings();
            }
            alert('Dados importados com sucesso!');
            showImportExport = false;
            importJsonText = "";
        } catch (err) {
            console.error('ImportUserData error:', err);
            alert('Erro ao importar: ' + err);
        }
    }

    // === CACHE & SOURCE STATUS ===
    async function loadCacheStats() {
        try {
            cacheStats = await GetCacheStats();
            console.log('[CacheStats]', cacheStats);
        } catch (err) {
            console.error('GetCacheStats error:', err);
        }
    }

    async function resetSources() {
        try {
            await ResetSourceFailures();
            await loadCacheStats();
            alert('Fontes resetadas com sucesso!');
        } catch (err) {
            console.error('ResetSourceFailures error:', err);
        }
    }

    async function clearAllCacheAction() {
        try {
            await ClearAllCache();
            await loadCacheStats();
            alert('Cache limpo com sucesso!');
        } catch (err) {
            console.error('ClearAllCache error:', err);
        }
    }

    function copyExportData() {
        navigator.clipboard.writeText(exportJsonText);
        alert('Copiado para a área de transferência!');
    }

    // === DISCORD LINKING SYSTEM (Vinculação por Código) ===
    
    // Abre o modal de vinculação
    function openLinkModal() {
        showLinkModal = true;
        linkCode = "";
        linkError = "";
    }
    
    // Fecha o modal de vinculação
    function closeLinkModal() {
        showLinkModal = false;
        linkCode = "";
        linkError = "";
    }
    
    // Vincula conta usando código
    async function linkWithCode() {
        if (!linkCode.trim()) {
            linkError = "Digite o código gerado pelo bot!";
            return;
        }
        
        linkLoading = true;
        linkError = "";
        
        try {
            const result = await LinkDiscordWithCode(linkCode.trim());
            if (result.isLinked) {
                discordLinked = true;
                discordLinkInfo = result;
                showLinkModal = false;
                linkCode = "";
                // Carrega atividade dos amigos
                await loadFriendsActivity();
            }
        } catch (err) {
            console.error('Link error:', err);
            linkError = err.toString().replace('Error: ', '');
        } finally {
            linkLoading = false;
        }
    }
    
    // Desvincula conta
    async function unlinkDiscord() {
        if (!confirm("Deseja realmente desvincular sua conta Discord?")) return;
        
        try {
            await UnlinkDiscord();
            discordLinked = false;
            discordLinkInfo = null;
            friendsActivity = [];
        } catch (err) {
            console.error('Unlink error:', err);
        }
    }
    
    // Carrega estado do Discord (vinculação)
    async function loadDiscordState() {
        try {
            const status = await GetDiscordLinkStatus();
            discordLinked = status.isLinked;
            
            if (status.isLinked) {
                discordLinkInfo = status;
                await loadFriendsActivity();
            }
            
            // Obtém link do servidor
            discordServerInvite = await GetDiscordServerInvite();
        } catch (err) {
            console.error('Error loading Discord state:', err);
        }
    }
    
    // Carrega atividade dos amigos
    async function loadFriendsActivity() {
        if (!discordLinked) return;
        loadingFriends = true;
        
        try {
            const activities = await GetDiscordFriendsActivity();
            friendsActivity = activities || [];
        } catch (err) {
            console.error('Error loading friends activity:', err);
            friendsActivity = [];
        } finally {
            loadingFriends = false;
        }
    }
    
    // Atualiza status de "assistindo"
    async function updateWatchingStatus(animeTitle, episodeNum, animeImage, totalEpisodes = 0) {
        if (!discordLinked || !discordLinkInfo?.showStatus) return;
        
        try {
            await UpdateDiscordWatchingStatus(animeTitle, episodeNum, animeImage, totalEpisodes);
        } catch (err) {
            console.error('Error updating watching status:', err);
        }
    }
    
    // Toggle para mostrar status
    async function toggleShowStatus() {
        if (!discordLinkInfo) return;
        
        const newValue = !discordLinkInfo.showStatus;
        try {
            await SetDiscordShowStatus(newValue);
            discordLinkInfo.showStatus = newValue;
        } catch (err) {
            console.error('Error toggling show status:', err);
        }
    }
    
    // Toggle para compartilhar animes
    async function toggleShareAnimes() {
        if (!discordLinkInfo) return;
        
        const newValue = !discordLinkInfo.shareAnimes;
        try {
            await SetDiscordShareAnimes(newValue);
            discordLinkInfo.shareAnimes = newValue;
        } catch (err) {
            console.error('Error toggling share animes:', err);
        }
    }
    
    // Handler para evento de vinculação
    function handleDiscordLinked(data) {
        console.log('Discord linked:', data);
        discordLinked = true;
        loadDiscordState();
    }
    
    // Handler para evento de desvinculação
    function handleDiscordUnlinked() {
        discordLinked = false;
        discordLinkInfo = null;
        friendsActivity = [];
    }
    
    // Estado para modal de compartilhar
    let showShareModal = false;
    let shareAnime = null;
    let shareMessage = '';
    
    function openShareModal(anime) {
        shareAnime = anime;
        shareMessage = '';
        showShareModal = true;
    }
    
    async function sendRecommendation() {
        if (!shareAnime || !shareMessage.trim()) return;
        
        try {
            await SendDiscordRecommendation(
                shareAnime.title || shareAnime.Title,
                shareAnime.image || shareAnime.Image,
                shareAnime.score || 0,
                shareMessage
            );
            showShareModal = false;
            shareAnime = null;
            shareMessage = '';
            alert('Recomendação enviada! 🎉');
        } catch (err) {
            console.error('Send recommendation error:', err);
            alert('Erro ao enviar recomendação');
        }
    }
    
    function formatTimeAgo(timestamp) {
        const diff = Date.now() - timestamp;
        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);
        
        if (days > 0) return `${days}d atrás`;
        if (hours > 0) return `${hours}h atrás`;
        if (minutes > 0) return `${minutes}min atrás`;
        return 'agora';
    }
    
    // === TAB NAVIGATION ===
    function switchTab(tab) {
        activeTab = tab;
        if (tab === 'friends' && discordLinked) {
            loadFriendsActivity();
        }
    }

    function openView(view) {
        currentView = view;
        userMenuOpen = false;
        episodeSelectionScreen = false;
        playingEpisodeNatively = false;
        
        // Scroll suave para o topo
        setTimeout(() => scrollToTop(true), 50);
        
        if (view === 'favorites') {
            loadFavorites();
        } else if (view === 'history') {
            loadWatchHistory();
        } else if (view === 'settings') {
            loadUserSettings();
            loadCacheStats(); // Carrega status das fontes automaticamente
        }
    }

    async function carregarDados() {
        carregando = true;
        
        // Timeout de 8s para não travar se backend demorar
        const timeout = (promise, ms) => Promise.race([
            promise,
            new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), ms))
        ]);
        
        try {
            // Load em paralelo com timeout generoso
            const [trendingRes, topRes] = await Promise.allSettled([
                timeout(GetTrendingAnimes(15), 8000),
                timeout(GetTopAnimes(), 8000)
            ]);
            
            console.log('[carregarDados] trendingRes:', trendingRes);
            console.log('[carregarDados] topRes:', topRes);
            
            // AniList Trending (HD images, banners)
            if (trendingRes.status === 'fulfilled' && trendingRes.value?.length > 0) {
                trendingAnimes = trendingRes.value;
                console.log('[carregarDados] trendingAnimes:', trendingAnimes.length);
                // Featured anime - prefer with banner, fallback to first with image
                const animesWithBanners = trendingAnimes.filter(a => a.banner);
                console.log('[carregarDados] animesWithBanners:', animesWithBanners.length);
                if (animesWithBanners.length > 0) {
                    featuredAnime = animesWithBanners[0];
                    featuredIndex = 0;
                    startFeaturedRotation();
                    console.log('[carregarDados] featuredAnime set (with banner):', featuredAnime.title);
                } else if (trendingAnimes.length > 0 && trendingAnimes[0].image) {
                    // Fallback: usa primeiro anime com imagem, cria banner fake com gradient
                    featuredAnime = {...trendingAnimes[0], banner: trendingAnimes[0].image};
                    featuredIndex = 0;
                    console.log('[carregarDados] featuredAnime set (fallback):', featuredAnime.title);
                }
            } else {
                console.warn('[carregarDados] trendingRes failed or empty:', trendingRes);
            }
            
            // Top animes
            if (topRes.status === 'fulfilled' && topRes.value?.length > 0) {
                topAnimes = topRes.value;
                console.log('[carregarDados] topAnimes:', topAnimes.length);
            }
            
        } catch (err) {
            console.error('carregarDados error:', err);
        } finally {
            carregando = false;
        }
    }

    async function criarConta() {
        if (!nomeInput) return;
        try {
            usuario = await CreateUser(nomeInput, avatarSelecionado);
            await carregarDados();
        } catch (err) {
            console.error('CreateUser error:', err);
        }
    }

    async function pesquisar() {
        if (!termoBusca) return;
        selectedGenre = null; // Limpa gênero selecionado ao pesquisar
        carregando = true;
        try {
            const res = (await BuscarAnimes(termoBusca)) || [];
            resultadosBusca = Array.isArray(res) ? res : [];
        } catch (err) {
            console.error('BuscarAnimes error:', err);
            resultadosBusca = [];
        } finally {
            carregando = false;
        }
    }
    
    async function searchByGenre(genre) {
        selectedGenre = genre;
        termoBusca = genre.name;
        carregando = true;
        resultadosBusca = [];
        
        try {
            // Usa a função Go otimizada que busca TODOS os termos em paralelo no backend
            const searchTerms = genre.searchTerms || [genre.id];
            console.log(`[searchByGenre] Buscando ${searchTerms.length} termos via Go multithread...`);
            
            // Chama função Go que faz multithread internamente
            const results = await BuscarAnimesMulti(searchTerms);
            resultadosBusca = Array.isArray(results) ? results : [];
            
            console.log(`[searchByGenre] ${genre.name}: ${resultadosBusca.length} resultados (multithread Go)`);
        } catch (err) {
            console.error('searchByGenre error:', err);
            resultadosBusca = [];
        } finally {
            carregando = false;
        }
    }
    
    function clearGenreFilter() {
        selectedGenre = null;
        termoBusca = '';
        resultadosBusca = [];
    }

    async function openEpisodeSelection(anime) {
        // Fecha player se estiver ativo
        if (playingEpisodeNatively) {
            closePlayer();
        }
        
        // Limpa estado anterior
        selectedAnime = anime;
        episodes = [];
        seasons = [];
        selectedEpisodeURL = "";
        selectedSource = null;
        showSourceSelector = false;
        playerUrl = "";
        originalStreamUrl = "";
        currentPlayingEpisodeTitle = "";
        
        // Guarda as fontes disponíveis
        availableSources = anime.Sources || [];

        // Scroll para o topo ao abrir detalhes do anime
        setTimeout(() => scrollToTop(true), 50);

        // Verifica se tem múltiplas fontes
        if (anime.Sources && anime.Sources.length > 1) {
            // Mostra tela de seleção de fonte
            showSourceSelector = true;
            episodeSelectionScreen = true;
            loadingEpisodes = false;
            return;
        }

        // Se só tem uma fonte (ou nenhuma explícita), carrega direto
        await loadEpisodesFromSource(anime);
    }

    async function selectSource(source) {
        selectedSource = source;
        showSourceSelector = false;
        loadingEpisodes = true;
        
        try {
            const cacheKey = `${source.Name}:${source.URL}`;
            
            if (episodeCache.has(cacheKey)) {
                episodes = episodeCache.get(cacheKey);
            } else {
                const eps = await GetEpisodesForSource(source.URL, source.Name);
                episodes = Array.isArray(eps) ? eps : [];
                episodeCache.set(cacheKey, episodes);
            }
            
            // Atualiza URL do anime selecionado para a fonte escolhida
            selectedAnime.URL = source.URL;
            
            // Processa temporadas
            const s = new Set();
            episodes.forEach(e => s.add(e.Season || 1));
            seasons = Array.from(s).sort((a,b) => a-b);
            if (seasons.length > 0) selectedSeason = seasons[0];

        } catch (err) {
            console.error('[selectSource] Error:', err);
            alert('Erro ao carregar episódios da fonte: ' + source.Name);
        } finally {
            loadingEpisodes = false;
        }
    }

    async function loadEpisodesFromSource(anime) {
        episodeSelectionScreen = true;
        loadingEpisodes = true;

        try {
            // Cache de URL
            let seriesURL = anime.URL;
            if (!seriesURL) {
                if (urlCache.has(anime.Title)) {
                    seriesURL = urlCache.get(anime.Title);
                } else {
                    seriesURL = await GetAnimeURL(anime.Title);
                    urlCache.set(anime.Title, seriesURL);
                }
                anime.URL = seriesURL;
            }

            // Se tem uma fonte específica, usa ela
            if (anime.Sources && anime.Sources.length === 1) {
                selectedSource = anime.Sources[0];
                seriesURL = anime.Sources[0].URL;
            }

            // Cache de episódios
            const cacheKey = selectedSource ? `${selectedSource.Name}:${seriesURL}` : seriesURL;
            if (episodeCache.has(cacheKey)) {
                episodes = episodeCache.get(cacheKey);
            } else {
                let eps;
                if (selectedSource) {
                    eps = await GetEpisodesForSource(seriesURL, selectedSource.Name);
                } else {
                    eps = await GetEpisodes(seriesURL);
                }
                episodes = Array.isArray(eps) ? eps : [];
                episodeCache.set(cacheKey, episodes);
            }
            
            const s = new Set();
            episodes.forEach(e => s.add(e.Season || 1));
            seasons = Array.from(s).sort((a,b) => a-b);
            if (seasons.length > 0) selectedSeason = seasons[0];

        } catch (err) {
            console.error('[loadEpisodesFromSource] Error:', err);
            alert('Erro ao carregar episódios');
        } finally {
            loadingEpisodes = false;
        }
    }

    function closeEpisodeSelection() {
        episodeSelectionScreen = false;
        selectedAnime = null;
        episodes = [];
        selectedEpisodeURL = "";
        playingEpisodeNatively = false;
        playerUrl = "";
        showSourceSelector = false;
        selectedSource = null;
        availableSources = [];
        
        // Scroll suave de volta ao topo
        setTimeout(() => scrollToTop(true), 50);
    }

    // Força recarregar episódios (limpa cache local e do backend)
    async function forceReloadEpisodes() {
        if (!selectedAnime) return;
        
        loadingEpisodes = true;
        try {
            // Limpa cache do backend
            await ClearEpisodesCache();
            
            // Limpa cache local
            episodeCache.clear();
            
            // Recarrega
            if (selectedSource) {
                await selectSource(selectedSource);
            } else {
                await loadEpisodesFromSource(selectedAnime);
            }
            
            console.log('[forceReloadEpisodes] Recarregado com sucesso');
        } catch (err) {
            console.error('[forceReloadEpisodes] Error:', err);
            alert('Erro ao recarregar episódios');
        } finally {
            loadingEpisodes = false;
        }
    }

    async function playEpisode() {
        console.log('[playEpisode] CHAMADO - MPV');
        if (!selectedEpisodeURL) {
            alert('Selecione um episódio');
            return;
        }

        const currentEp = episodes.find(e => e.URL === selectedEpisodeURL);
        currentPlayingEpisodeTitle = currentEp 
            ? `Episódio ${currentEp.Number} - ${currentEp.Title || 'Sem título'}` 
            : 'Episódio';

        try {
            const animeURL = selectedAnime?.URL || '';
            if (!animeURL) {
                alert('Erro: URL do anime não encontrada');
                return;
            }
            
            // Toca direto no MPV
            await AssistirEpisodio(animeURL, selectedEpisodeURL, currentPlayingEpisodeTitle);
            
        } catch (err) {
            console.error('[playEpisode] Error:', err);
            alert('Erro ao reproduzir: ' + err);
        }
    }

    async function playEpisodeInBrowser() {
        console.log('[playEpisodeInBrowser] CHAMADO - NAVEGADOR');
        if (!selectedEpisodeURL) {
            alert('Selecione um episódio');
            return;
        }

        const currentEp = episodes.find(e => e.URL === selectedEpisodeURL);
        currentPlayingEpisodeTitle = currentEp 
            ? `Episódio ${currentEp.Number} - ${currentEp.Title || 'Sem título'}` 
            : 'Episódio';
        
        // Salva o número do episódio para os skip times
        currentEpisodeNumber = currentEp?.Number || 1;

        try {
            const animeURL = selectedAnime?.URL || '';
            if (!animeURL) {
                alert('Erro: URL do anime não encontrada');
                return;
            }
            
            // Busca MAL ID para AniSkip (em paralelo com o stream)
            const malIdPromise = fetchMalIdForAnime(selectedAnime);
            
            // Extrai URL do stream
            const streamURL = await GetStreamURLForEpisode(animeURL, selectedEpisodeURL);
            if (!streamURL) {
                alert('Não foi possível extrair o link do vídeo. Tente usar o MPV.');
                return;
            }
            
            console.log('[playEpisodeInBrowser] Stream URL:', streamURL);
            originalStreamUrl = streamURL;
            
            // Verifica se é SharePoint/OneDrive - avisa mas tenta mesmo assim
            const isSharePoint = streamURL.includes('sharepoint.com') || 
                                 streamURL.includes('microsoft.com') ||
                                 streamURL.includes('onedrive') ||
                                 streamURL.includes('download.aspx');
            
            if (isSharePoint) {
                console.log('[playEpisodeInBrowser] URL é SharePoint - pode ter problemas de CORS');
            }
            
            // Usa proxy para contornar CORS
            const proxyURL = await GetProxyURLForVideo(streamURL);
            console.log('[playEpisodeInBrowser] Proxy URL:', proxyURL);
            
            playerUrl = proxyURL;
            playingEpisodeNatively = true;
            
            // Aguarda o DOM atualizar e configura o player
            setTimeout(() => setupVideoPlayer(), 100);
            
            // Carrega skip times em background (não bloqueia o player)
            malIdPromise.then(malID => {
                if (malID > 0) {
                    currentMalID = malID;
                    loadSkipTimes(malID, currentEpisodeNumber);
                }
            });
            
        } catch (err) {
            console.error('[playEpisodeInBrowser] Error:', err);
            alert('Erro ao reproduzir no navegador: ' + err + '\n\nTente usar o MPV.');
        }
    }
    
    function setupVideoPlayer() {
        if (!videoEl || !playerUrl) return;
        
        console.log('[setupVideoPlayer] Configurando player para:', originalStreamUrl);
        
        // Limpa instância anterior do HLS
        if (hlsInstance) {
            hlsInstance.destroy();
            hlsInstance = null;
        }
        
        // Verifica se é HLS (m3u8)
        const isHLS = originalStreamUrl.includes('.m3u8') || originalStreamUrl.includes('m3u8');
        
        if (isHLS && window['Hls'] && window['Hls'].isSupported()) {
            console.log('[setupVideoPlayer] Usando HLS.js para stream m3u8');
            hlsInstance = new window['Hls']({
                debug: false,
                enableWorker: true,
                lowLatencyMode: false,
            });
            
            hlsInstance.loadSource(playerUrl);
            hlsInstance.attachMedia(videoEl);
            
            hlsInstance.on(window['Hls'].Events.MANIFEST_PARSED, () => {
                console.log('[HLS] Manifest parsed, iniciando reprodução');
                videoEl.play().catch(err => console.log('[HLS] Play error:', err));
            });
            
            hlsInstance.on(window['Hls'].Events.ERROR, (event, data) => {
                console.error('[HLS] Error:', data.type, data.details);
                if (data.fatal) {
                    switch (data.type) {
                        case window['Hls'].ErrorTypes.NETWORK_ERROR:
                            console.log('[HLS] Tentando recuperar erro de rede...');
                            hlsInstance.startLoad();
                            break;
                        case window['Hls'].ErrorTypes.MEDIA_ERROR:
                            console.log('[HLS] Tentando recuperar erro de mídia...');
                            hlsInstance.recoverMediaError();
                            break;
                        default:
                            console.error('[HLS] Erro fatal, destruindo instância');
                            hlsInstance.destroy();
                            break;
                    }
                }
            });
        } else if (videoEl.canPlayType('application/vnd.apple.mpegurl')) {
            // Safari suporta HLS nativamente
            console.log('[setupVideoPlayer] Usando HLS nativo (Safari)');
            videoEl.src = playerUrl;
            videoEl.play().catch(err => console.log('[Native HLS] Play error:', err));
        } else {
            // MP4 ou outros formatos
            console.log('[setupVideoPlayer] Usando player nativo para:', originalStreamUrl);
            videoEl.src = playerUrl;
            videoEl.load();
            videoEl.play().catch(err => console.log('[Native] Play error:', err));
        }
    }

    function closePlayer() {
        console.log('[closePlayer] Fechando player...');
        
        // Limpa HLS.js
        if (hlsInstance) {
            hlsInstance.destroy();
            hlsInstance = null;
        }
        
        if (videoEl) {
            try {
                videoEl.pause();
                videoEl.removeAttribute('src');
                videoEl.load();
            } catch (err) {
                console.log('[closePlayer] Error:', err);
            }
        }
        
        // Limpa estado do player
        playingEpisodeNatively = false;
        playerUrl = "";
        originalStreamUrl = "";
        currentPlayingEpisodeTitle = "";
        currentSkipTimes = null;
        currentMalID = 0;
        currentEpisodeNumber = 1;
        
        // Volta para tela de episódios (não limpa o anime selecionado)
        // Assim o usuário pode escolher outro episódio
    }

    // Busca MAL ID pelo título do anime (para AniSkip)
    async function fetchMalIdForAnime(anime) {
        // Se já tem MAL ID, retorna direto
        if (anime?.malId && anime.malId > 0) {
            console.log('[fetchMalIdForAnime] MAL ID já disponível:', anime.malId);
            return anime.malId;
        }
        
        // Tenta buscar no AniList pelo título
        const title = anime?.Title || anime?.title;
        if (!title) return 0;
        
        try {
            console.log('[fetchMalIdForAnime] Buscando MAL ID para:', title);
            const results = await SearchAniList(title, 1);
            if (results && results.length > 0 && results[0].malId > 0) {
                console.log('[fetchMalIdForAnime] MAL ID encontrado:', results[0].malId);
                return results[0].malId;
            }
        } catch (err) {
            console.log('[fetchMalIdForAnime] Erro ao buscar MAL ID:', err);
        }
        
        return 0;
    }

    // Busca skip times (abertura/encerramento) para o episódio
    async function loadSkipTimes(malID, episodeNumber) {
        if (!malID || malID <= 0 || !episodeNumber || episodeNumber <= 0) {
            console.log('[loadSkipTimes] MAL ID ou episódio inválido:', malID, episodeNumber);
            currentSkipTimes = null;
            return;
        }
        
        try {
            console.log('[loadSkipTimes] Buscando skip times: MAL ID =', malID, ', Ep =', episodeNumber);
            const skipTimes = await GetSkipTimes(malID, episodeNumber);
            if (skipTimes) {
                currentSkipTimes = skipTimes;
                console.log('[loadSkipTimes] Skip times carregados:', skipTimes);
            } else {
                currentSkipTimes = null;
            }
        } catch (err) {
            console.log('[loadSkipTimes] Erro ao buscar skip times:', err);
            currentSkipTimes = null;
        }
    }

    async function selectNextEpisode() {
        if (!filteredEpisodes || filteredEpisodes.length === 0) {
            console.log('[selectNextEpisode] Sem episódios disponíveis');
            return;
        }
        
        const currentIndex = filteredEpisodes.findIndex(e => e.URL === selectedEpisodeURL);
        console.log('[selectNextEpisode] Index atual:', currentIndex, 'de', filteredEpisodes.length);
        
        if (currentIndex >= 0 && currentIndex < filteredEpisodes.length - 1) {
            selectedEpisodeURL = filteredEpisodes[currentIndex + 1].URL;
            console.log('[selectNextEpisode] Próximo episódio:', selectedEpisodeURL);
            await playEpisodeInBrowser();
        } else {
            console.log('[selectNextEpisode] Já está no último episódio');
        }
    }

    async function selectPreviousEpisode() {
        if (!filteredEpisodes || filteredEpisodes.length === 0) {
            console.log('[selectPreviousEpisode] Sem episódios disponíveis');
            return;
        }
        
        const currentIndex = filteredEpisodes.findIndex(e => e.URL === selectedEpisodeURL);
        console.log('[selectPreviousEpisode] Index atual:', currentIndex);
        
        if (currentIndex > 0) {
            selectedEpisodeURL = filteredEpisodes[currentIndex - 1].URL;
            console.log('[selectPreviousEpisode] Episódio anterior:', selectedEpisodeURL);
            await playEpisodeInBrowser();
        } else {
            console.log('[selectPreviousEpisode] Já está no primeiro episódio');
        }
    }

    $: filteredEpisodes = selectedSeason ? episodes.filter(e => (e.Season || 1) === selectedSeason) : episodes;

</script>

<main>
    <!-- SPLASH SCREEN - Netflix Style -->
    {#if showSplash}
        <div class="splash-screen" class:fade-out={splashProgress >= 100}>
            <div class="splash-content">
                <!-- Logo Animation -->
                <div class="splash-logo">
                    <div class="splash-logo-icon">
                        <span class="splash-emoji">🎬</span>
                        <div class="splash-glow"></div>
                    </div>
                    <h1 class="splash-title">
                        <span class="splash-go">Go</span><span class="splash-anime">Anime</span>
                    </h1>
                </div>
                
                <!-- Loading Animation -->
                <div class="splash-loader">
                    <div class="loader-bar">
                        <div class="loader-progress" style="width: {splashProgress}%"></div>
                    </div>
                    <p class="loader-status">{splashStatus}</p>
                </div>
            </div>
        </div>
    {/if}

    <!-- Simple Video Player -->
    {#if playingEpisodeNatively}
        <SimplePlayer
            src={playerUrl}
            title={selectedAnime?.Title || 'Reproduzindo...'}
            episodeTitle={currentPlayingEpisodeTitle}
            skipTimes={currentSkipTimes}
            onClose={closePlayer}
            onNext={selectNextEpisode}
            onPrevious={selectPreviousEpisode}
        />
    {:else if !usuario}
        <!-- LOGIN SCREEN - MODERN FULLSCREEN -->
        <div class="login-screen">
            <!-- Animated Background -->
            <div class="login-bg">
                <div class="bg-gradient"></div>
                <div class="bg-particles">
                    {#each Array(20) as _, i}
                        <div class="particle" style="--delay: {i * 0.5}s; --x: {Math.random() * 100}%; --duration: {15 + Math.random() * 20}s"></div>
                    {/each}
                </div>
            </div>
            
            <!-- Login Content -->
            <div class="login-content">
                <!-- Logo Section -->
                <div class="login-branding">
                    <div class="logo-container">
                        <div class="logo-icon">
                            <span class="logo-emoji">🎬</span>
                            <div class="logo-glow"></div>
                        </div>
                        <h1 class="logo-text">
                            <span class="logo-go">Go</span><span class="logo-anime">Anime</span>
                        </h1>
                    </div>
                    <p class="login-tagline">Sua plataforma de anime favorita</p>
                </div>
                
                <!-- Login Card -->
                <div class="login-card-modern">
                    <div class="card-header">
                        <h2>Bem-vindo!</h2>
                        <p>Crie seu perfil para começar</p>
                    </div>
                    
                    <div class="card-body">
                        <!-- Name Input -->
                        <div class="input-group">
                            <label for="username">Seu nome</label>
                            <div class="input-wrapper">
                                <span class="input-icon">👤</span>
                                <input 
                                    id="username"
                                    type="text"
                                    bind:value={nomeInput} 
                                    placeholder="Digite seu nome" 
                                    class="input-modern"
                                    onkeydown={(e) => e.key === 'Enter' && criarConta()}
                                />
                            </div>
                        </div>
                        
                        <!-- Avatar Selection -->
                        <div class="avatar-group">
                            <span class="avatar-label" id="avatar-label">Escolha seu avatar</span>
                            <div class="avatar-grid" role="radiogroup" aria-labelledby="avatar-label">
                                {#each [
                                    { id: 'avatar1.png', emoji: '👤', label: 'Usuário' },
                                    { id: 'avatar2.png', emoji: '🦊', label: 'Raposa' },
                                    { id: 'avatar3.png', emoji: '🤖', label: 'Robô' },
                                    { id: 'avatar4.png', emoji: '🐱', label: 'Gato' },
                                    { id: 'avatar5.png', emoji: '🎮', label: 'Gamer' },
                                    { id: 'avatar6.png', emoji: '⚡', label: 'Energia' }
                                ] as avatar}
                                    <button 
                                        type="button"
                                        class="avatar-option {avatarSelecionado === avatar.id ? 'selected' : ''}"
                                        onclick={() => avatarSelecionado = avatar.id}
                                        title={avatar.label}
                                    >
                                        <span class="avatar-emoji">{avatar.emoji}</span>
                                        {#if avatarSelecionado === avatar.id}
                                            <span class="avatar-check">✓</span>
                                        {/if}
                                    </button>
                                {/each}
                            </div>
                        </div>
                    </div>
                    
                    <div class="card-footer">
                        <button type="button" class="btn-enter" onclick={criarConta} disabled={!nomeInput.trim()}>
                            <span>Entrar</span>
                            <span class="btn-arrow">→</span>
                        </button>
                    </div>
                </div>
                
                <!-- Features Preview -->
                <div class="login-features">
                    <div class="feature">
                        <span class="feature-icon">🔥</span>
                        <span class="feature-text">Animes em alta qualidade</span>
                    </div>
                    <div class="feature">
                        <span class="feature-icon">⚡</span>
                        <span class="feature-text">Streaming rápido</span>
                    </div>
                    <div class="feature">
                        <span class="feature-icon">📱</span>
                        <span class="feature-text">100% gratuito</span>
                    </div>
                </div>
            </div>
        </div>
    {:else}
        <!-- MAIN APP -->
        <div class="app">
            <!-- HEADER - Modern design -->
            <header class="header {episodeSelectionScreen || currentView !== 'home' ? '' : 'minimal'}">
                <div class="header-left">
                    <button type="button" class="btn-logo" onclick={() => { currentView = 'home'; episodeSelectionScreen = false; }}>
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
                        onclick={(e) => { e.stopPropagation(); userMenuOpen = !userMenuOpen; }}
                    >
                        <span class="user-avatar">👤</span>
                        <span class="user-name">{usuario.username}</span>
                        <span class="menu-arrow">{userMenuOpen ? '▲' : '▼'}</span>
                    </button>
                    
                    {#if userMenuOpen}
                        <div class="user-dropdown">
                            <button type="button" class="dropdown-item" onclick={() => openView('favorites')}>
                                ⭐ Favoritos
                            </button>
                            <button type="button" class="dropdown-item" onclick={() => openView('history')}>
                                🕐 Últimos Assistidos
                            </button>
                            <button type="button" class="dropdown-item" onclick={() => openView('settings')}>
                                ⚙️ Configurações
                            </button>
                        </div>
                    {/if}
                </div>
            </header>

            <!-- CONTENT -->
            <div class="main-content" bind:this={mainContentEl}>
                <!-- FAVORITES VIEW -->
                {#if currentView === 'favorites' && !episodeSelectionScreen}
                    <div class="user-view">
                        <h2>⭐ Meus Favoritos</h2>
                        
                        {#if favorites.length === 0}
                            <div class="empty-state">
                                <p>Você ainda não tem favoritos.</p>
                                <p>Clique no ⭐ em um anime para adicionar!</p>
                            </div>
                        {:else}
                            <div class="anime-grid">
                                {#each favorites as anime}
                                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                                    <div 
                                        class="anime-card" 
                                        onclick={() => openEpisodeSelection(anime)}
                                        onmouseenter={() => schedulePrefetch(anime)}
                                    >
                                        {#if anime.Image}
                                            <img src={anime.Image} alt={anime.Title} />
                                        {:else}
                                            <div class="no-image">📺</div>
                                        {/if}
                                        <div class="anime-title">{anime.Title}</div>
                                        <button 
                                            type="button" 
                                            class="btn-fav active"
                                            onclick={(e) => { e.stopPropagation(); toggleFavorite(anime); }}
                                        >⭐</button>
                                    </div>
                                {/each}
                            </div>
                        {/if}
                    </div>
                    
                <!-- HISTORY VIEW -->
                {:else if currentView === 'history' && !episodeSelectionScreen}
                    <div class="user-view">
                        <h2>🕐 Últimos Assistidos</h2>
                        
                        {#if watchHistory.length === 0}
                            <div class="empty-state">
                                <p>Você ainda não assistiu nenhum episódio.</p>
                                <p>Comece a assistir para ver o histórico aqui!</p>
                            </div>
                        {:else}
                                    <div class="history-list">
                                {#each watchHistory as item}
                                    <button type="button" class="history-item" onclick={() => {
                                        selectedAnime = { Title: item.anime_title, Image: item.anime_image, URL: item.anime_url };
                                        openEpisodeSelection(selectedAnime);
                                    }}>
                                        {#if item.anime_image}
                                            <img src={item.anime_image} alt={item.anime_title} class="history-thumb" />
                                        {:else}
                                            <div class="history-thumb no-image">📺</div>
                                        {/if}
                                        <div class="history-info">
                                            <div class="history-anime">{item.anime_title}</div>
                                            <div class="history-episode">Episódio {item.episode_num}</div>
                                            <div class="history-date">{new Date(item.watched_at).toLocaleDateString()}</div>
                                        </div>
                                    </button>
                                {/each}
                            </div>
                        {/if}
                    </div>
                    
                <!-- SETTINGS VIEW -->
                {:else if currentView === 'settings' && !episodeSelectionScreen}
                    <div class="user-view settings-view">
                        <h2>⚙️ Configurações</h2>
                        
                        <div class="settings-group">
                            <h3>Preferências</h3>
                            
                            <div class="setting-item">
                                <label>
                                    <input type="checkbox" bind:checked={settings.start_fullscreen} />
                                    Iniciar em tela cheia
                                </label>
                            </div>
                            
                            <div class="setting-item">
                                <label>
                                    Conteúdo preferido:
                                    <select bind:value={settings.content_language}>
                                        <option value="all">Todos (BR + EN)</option>
                                        <option value="br">Apenas Português (BR)</option>
                                        <option value="en">Apenas Inglês (EN)</option>
                                    </select>
                                </label>
                            </div>
                            
                            <div class="setting-item">
                                <label>
                                    <input type="checkbox" bind:checked={settings.use_anime4k} />
                                    Usar Anime4K (upscaling)
                                </label>
                            </div>
                            
                            <button type="button" class="btn-primary" onclick={saveUserSettings}>
                                💾 Salvar Configurações
                            </button>
                        </div>
                        
                        <div class="settings-group">
                            <h3>Backup & Restauração</h3>
                            <p class="settings-desc">Exporte seus dados para fazer backup ou importe para restaurar.</p>
                            
                            <div class="backup-buttons">
                                <button type="button" class="btn-secondary" onclick={exportData}>
                                    📤 Exportar Dados
                                </button>
                                <button type="button" class="btn-secondary" onclick={() => showImportExport = true}>
                                    📥 Importar Dados
                                </button>
                            </div>
                            
                            {#if showImportExport}
                                <div class="import-export-modal">
                                    <div class="modal-content">
                                        <h4>Importar / Exportar</h4>
                                        
                                        {#if exportJsonText}
                                            <div class="export-section">
                                                <p>Copie o JSON abaixo para fazer backup:</p>
                                                <textarea readonly value={exportJsonText}></textarea>
                                                <button type="button" class="btn-primary" onclick={copyExportData}>
                                                    📋 Copiar
                                                </button>
                                            </div>
                                        {/if}
                                        
                                        <div class="import-section">
                                            <p>Cole o JSON para importar:</p>
                                            <textarea bind:value={importJsonText} placeholder="Cole o JSON aqui..."></textarea>
                                            <button type="button" class="btn-primary" onclick={importData}>
                                                📥 Importar
                                            </button>
                                        </div>
                                        
                                        <button type="button" class="btn-close" onclick={() => { showImportExport = false; exportJsonText = ''; importJsonText = ''; }}>
                                            ✕ Fechar
                                        </button>
                                    </div>
                                </div>
                            {/if}
                        </div>
                        
                        <!-- STATUS DAS FONTES DE VÍDEO (SMART CACHE) -->
                        <div class="settings-group sources-status">
                            <h3>📡 Status das Fontes de Vídeo</h3>
                            <p class="settings-desc">Mostra o estado atual das fontes de streaming e cache inteligente.</p>
                            
                            <button type="button" class="btn-secondary" onclick={loadCacheStats}>
                                🔄 Atualizar Status
                            </button>
                            
                            {#if cacheStats}
                                <div class="cache-overview">
                                    <div class="cache-stat">
                                        <span class="stat-label">Streams em Cache:</span>
                                        <span class="stat-value">{cacheStats.totalStreams}</span>
                                    </div>
                                    <div class="cache-stat">
                                        <span class="stat-label">Total em Cache:</span>
                                        <span class="stat-value">{cacheStats.totalCache}</span>
                                    </div>
                                </div>
                                
                                <div class="sources-grid">
                                    {#each cacheStats.sources as source}
                                        <div class="source-status-card {source.isAvailable ? 'available' : 'unavailable'}">
                                            <div class="source-header">
                                                <span class="source-icon">{source.isAvailable ? '✅' : '⚠️'}</span>
                                                <span class="source-name">{source.name}</span>
                                            </div>
                                            <div class="source-details">
                                                {#if source.cachedUrls > 0}
                                                    <span class="cached-count">📦 {source.cachedUrls} URLs em cache</span>
                                                {/if}
                                                {#if source.failCount > 0}
                                                    <span class="fail-count">❌ {source.failCount} falhas</span>
                                                {/if}
                                                {#if source.retryAfter}
                                                    <span class="retry-time">⏰ Retry: {source.retryAfter}</span>
                                                {/if}
                                                {#if source.lastError}
                                                    <span class="last-error" title={source.lastError}>💬 {source.lastError.substring(0, 30)}...</span>
                                                {/if}
                                            </div>
                                        </div>
                                    {/each}
                                </div>
                                
                                <div class="cache-actions">
                                    <button type="button" class="btn-warning" onclick={resetSources}>
                                        🔄 Resetar Falhas
                                    </button>
                                    <button type="button" class="btn-danger" onclick={clearAllCacheAction}>
                                        🗑️ Limpar Todo Cache
                                    </button>
                                </div>
                            {:else}
                                <p class="no-stats">Clique em "Atualizar Status" para ver o estado das fontes.</p>
                            {/if}
                        </div>
                    </div>
                    
                {:else if episodeSelectionScreen}
                    <!-- ANIME DETAIL VIEW -->
                    <div class="anime-detail">
                        <button type="button" class="btn-back" onclick={closeEpisodeSelection}>
                            ← Voltar
                        </button>

                        <div class="anime-info">
                            {#if selectedAnime?.Image}
                                <img src={selectedAnime.Image} alt={selectedAnime.Title} class="anime-poster" />
                            {:else}
                                <div class="anime-poster no-poster">📺</div>
                            {/if}
                            <div class="anime-meta">
                                <h2>{selectedAnime?.Title}</h2>
                                
                                <!-- SELETOR DE FONTE (IDIOMA) -->
                                {#if showSourceSelector}
                                    <div class="source-selector">
                                        <h3>🌐 Escolha a Fonte / Idioma:</h3>
                                        <div class="source-buttons">
                                            {#each availableSources as source}
                                                <button 
                                                    type="button" 
                                                    class="source-btn {source.Language === 'en' ? 'english' : 'portuguese'}"
                                                    onclick={() => selectSource(source)}
                                                >
                                                    <span class="source-flag">
                                                        {source.Language === 'en' ? '🇺🇸' : '🇧🇷'}
                                                    </span>
                                                    <span class="source-name">{source.Name}</span>
                                                    <span class="source-lang">
                                                        {source.Language === 'en' ? 'Inglês (Legendado)' : 'Português (Dublado)'}
                                                    </span>
                                                </button>
                                            {/each}
                                        </div>
                                    </div>
                                {:else}
                                    <!-- Mostra fonte selecionada -->
                                    {#if selectedSource}
                                        <div class="current-source">
                                            <span class="source-badge {selectedSource.Language === 'en' ? 'english' : 'portuguese'}">
                                                {selectedSource.Language === 'en' ? '🇺🇸 Inglês' : '🇧🇷 Português'}
                                            </span>
                                            {#if availableSources.length > 1}
                                                <button type="button" class="btn-change-source" onclick={() => showSourceSelector = true}>
                                                    🔄 Trocar fonte
                                                </button>
                                            {/if}
                                            <button type="button" class="btn-reload" onclick={forceReloadEpisodes} title="Recarregar episódios">
                                                🔃 Recarregar
                                            </button>
                                        </div>
                                    {:else}
                                        <div class="current-source">
                                            <button type="button" class="btn-reload" onclick={forceReloadEpisodes} title="Recarregar episódios">
                                                🔃 Recarregar Episódios
                                            </button>
                                        </div>
                                    {/if}
                                    
                                    {#if seasons.length > 1}
                                        <div class="season-tabs">
                                            {#each seasons as season}
                                                <button 
                                                    type="button"
                                                    class="season-tab {selectedSeason === season ? 'active' : ''}"
                                                    onclick={() => selectedSeason = season}
                                                >
                                                    Temporada {season}
                                                </button>
                                            {/each}
                                        </div>
                                    {/if}
                                {/if}
                            </div>
                        </div>

                        {#if !showSourceSelector}
                            {#if loadingEpisodes}
                                <div class="loading">
                                    <div class="spinner"></div>
                                    <p>Carregando episódios...</p>
                                </div>
                            {:else}
                                <div class="episodes-grid" role="list">
                                    {#each filteredEpisodes as ep, index (`${ep.Number}-${index}`)}
                                        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
                                        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
                                        <div 
                                            class="episode-card {selectedEpisodeURL === ep.URL ? 'selected' : ''}"
                                            role="listitem"
                                            onclick={() => selectedEpisodeURL = ep.URL}
                                            onkeydown={(e) => e.key === 'Enter' && (selectedEpisodeURL = ep.URL)}
                                            tabindex="0"
                                        >
                                            <div class="episode-number">EP {ep.Number}</div>
                                            <div class="episode-title">{ep.Title || `Episódio ${ep.Number}`}</div>
                                            {#if ep.Source}
                                                <div class="episode-source">{ep.Source}</div>
                                            {/if}
                                            {#if selectedEpisodeURL === ep.URL}
                                                <div class="episode-actions">
                                                    <button type="button" class="btn-play-mpv primary" onclick={(e) => { e.stopPropagation(); playEpisode(); }} title="Recomendado - Funciona com todas as fontes">
                                                        🖥️ MPV (Recomendado)
                                                    </button>
                                                    <button type="button" class="btn-play-web" onclick={(e) => { e.stopPropagation(); playEpisodeInBrowser(); }} title="Pode não funcionar com algumas fontes">
                                                        ▶ Navegador
                                                    </button>
                                                </div>
                                            {/if}
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        {/if}
                    </div>
                {:else if currentView === 'home'}
                    <!-- HOME VIEW - NETFLIX/CRUNCHYROLL STYLE -->
                    <div class="home-view" class:ready={appReady}>
                        <!-- FEATURED HERO with ROTATION -->
                        {#if featuredAnime && featuredAnime.banner}
                            {#key featuredAnime.id || featuredAnime.title}
                            <div 
                                class="featured-hero" 
                                style="--banner-url: url({featuredAnime.banner}); --accent-color: {featuredAnime.color || '#f5576c'}"
                            >
                                <div class="featured-overlay"></div>
                                <div class="featured-content">
                                    <div class="featured-info">
                                        <div class="featured-badges">
                                            {#if featuredAnime.isAiring}
                                                <span class="badge airing">🔴 EM EXIBIÇÃO</span>
                                            {/if}
                                            {#if featuredAnime.score}
                                                <span class="badge score">⭐ {featuredAnime.score}%</span>
                                            {/if}
                                            {#if featuredAnime.episodes}
                                                <span class="badge episodes">{featuredAnime.episodes} eps</span>
                                            {/if}
                                        </div>
                                        <h1 class="featured-title">{featuredAnime.title}</h1>
                                        <p class="featured-meta">
                                            {featuredAnime.genres?.slice(0, 3).join(' • ') || ''}
                                            {#if featuredAnime.studio} • {featuredAnime.studio}{/if}
                                            {#if featuredAnime.year} • {featuredAnime.year}{/if}
                                        </p>
                                        {#if featuredAnime.description}
                                            <p class="featured-desc">
                                                {featuredAnime.description?.slice(0, 180)}{featuredAnime.description?.length > 180 ? '...' : ''}
                                            </p>
                                        {/if}
                                        <div class="featured-actions">
                                            <button type="button" class="btn-featured-play" onclick={() => {
                                                termoBusca = featuredAnime.title;
                                                pesquisar();
                                            }}>
                                                ▶ Assistir
                                            </button>
                                            {#if featuredAnime.trailerUrl}
                                                <a href={featuredAnime.trailerUrl} target="_blank" class="btn-featured-trailer">
                                                    🎬 Trailer
                                                </a>
                                            {/if}
                                        </div>
                                    </div>
                                    <div class="featured-poster">
                                        <img src={featuredAnime.image} alt={featuredAnime.title} loading="eager" />
                                    </div>
                                </div>
                                
                                <!-- Navigation Dots -->
                                <div class="featured-nav">
                                    {#each trendingAnimes.slice(0, 8) as anime, i}
                                        {#if anime.banner}
                                            <button 
                                                type="button"
                                                class="nav-dot {i === featuredIndex ? 'active' : ''}"
                                                onclick={() => selectFeatured(i)}
                                                title={anime.title}
                                            ></button>
                                        {/if}
                                    {/each}
                                </div>
                            </div>
                            {/key}
                        {:else if carregando}
                            <!-- Loading skeleton -->
                            <div class="featured-skeleton">
                                <div class="skeleton-shimmer"></div>
                            </div>
                        {:else}
                            <!-- Fallback Hero - Modern -->
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
                        
                        <!-- UNIFIED SEARCH & CONTENT AREA -->
                        <div class="content-area">
                            <!-- NAVIGATION TABS - Modern -->
                            <div class="nav-tabs-container">
                                <div class="nav-tabs">
                                    <button 
                                        type="button" 
                                        class="nav-tab {activeTab === 'anime' ? 'active' : ''}"
                                        onclick={() => switchTab('anime')}
                                    >
                                        <span class="tab-icon">🎬</span>
                                        <span class="tab-text">Animes</span>
                                    </button>
                                    <button 
                                        type="button" 
                                        class="nav-tab {activeTab === 'manga' ? 'active' : ''}"
                                        onclick={() => switchTab('manga')}
                                    >
                                        <span class="tab-icon">📚</span>
                                        <span class="tab-text">Mangás</span>
                                        <span class="tab-badge soon">Em breve</span>
                                    </button>
                                    <button 
                                        type="button" 
                                        class="nav-tab {activeTab === 'friends' ? 'active' : ''}"
                                        onclick={() => switchTab('friends')}
                                    >
                                        <span class="tab-icon">👥</span>
                                        <span class="tab-text">Amigos</span>
                                        {#if discordLinked && friendsActivity.length > 0}
                                            <span class="tab-badge notify">{friendsActivity.length}</span>
                                        {/if}
                                    </button>
                                    <button 
                                        type="button" 
                                        class="nav-tab {activeTab === 'community' ? 'active' : ''}"
                                        onclick={() => switchTab('community')}
                                    >
                                        <span class="tab-icon">🌐</span>
                                        <span class="tab-text">Comunidade</span>
                                    </button>
                                </div>
                            </div>
                            
                            <!-- TAB CONTENT -->
                            {#if activeTab === 'anime'}
                                <!-- SEARCH BAR - Always visible at top -->
                                <div class="search-bar-sticky">
                                    <div class="search-wrapper">
                                        <span class="search-icon">🔍</span>
                                        <input 
                                            type="text"
                                            bind:value={termoBusca}
                                            placeholder="Buscar anime... (ex: Frieren, One Piece, Naruto)"
                                            class="search-input"
                                            oninput={() => scheduleSearch(350)}
                                            onkeydown={e => e.key === 'Enter' && pesquisar()}
                                        />
                                        <button type="button" class="btn-search" onclick={pesquisar} disabled={carregando}>
                                            {carregando ? '⏳' : 'Buscar'}
                                        </button>
                                    </div>
                                    
                                    {#if resultadosBusca.length === 0 && !selectedGenre}
                                        <!-- Quick Search Pills -->
                                        <div class="quick-pills">
                                            <span class="pills-label">Popular:</span>
                                            <button type="button" class="pill" onclick={() => { termoBusca = 'Frieren'; pesquisar(); }}>Frieren</button>
                                            <button type="button" class="pill" onclick={() => { termoBusca = 'Jujutsu Kaisen'; pesquisar(); }}>Jujutsu</button>
                                            <button type="button" class="pill" onclick={() => { termoBusca = 'One Piece'; pesquisar(); }}>One Piece</button>
                                            <button type="button" class="pill" onclick={() => { termoBusca = 'Solo Leveling'; pesquisar(); }}>Solo Leveling</button>
                                    </div>
                                    
                                    <!-- GENRE CHIPS - Modern & Compact -->
                                    <div class="genre-chips-container">
                                        <span class="chips-label">Gêneros:</span>
                                        <div class="genre-chips">
                                            {#each animeGenres as genre}
                                                <button 
                                                    type="button" 
                                                    class="genre-chip"
                                                    onclick={() => searchByGenre(genre)}
                                                    title={genre.name}
                                                >
                                                    <span class="chip-icon">{genre.icon}</span>
                                                    <span class="chip-text">{genre.name}</span>
                                                </button>
                                            {/each}
                                        </div>
                                    </div>
                                {:else if resultadosBusca.length > 0}
                                    <div class="results-header">
                                        {#if selectedGenre}
                                            <span class="results-count">
                                                <span class="genre-badge">{selectedGenre.icon} {selectedGenre.name}</span>
                                                {resultadosBusca.length} resultados
                                            </span>
                                        {:else}
                                            <span class="results-count">{resultadosBusca.length} resultados para "{termoBusca}"</span>
                                        {/if}
                                        <button type="button" class="btn-clear-inline" onclick={clearGenreFilter}>
                                            ✕ Limpar
                                        </button>
                                    </div>
                                {/if}
                            </div>

                            {#if resultadosBusca.length > 0}
                                <!-- SEARCH RESULTS -->
                                <div class="anime-grid large">
                                    {#each resultadosBusca as anime (anime.Title)}
                                        <button 
                                            type="button" 
                                            class="anime-card" 
                                            onclick={() => openEpisodeSelection(anime)}
                                            onmouseenter={() => schedulePrefetch(anime)}
                                        >
                                            <div class="card-poster">
                                                {#if anime.Image}
                                                    <img src={anime.Image} alt={anime.Title} loading="lazy" />
                                                {:else}
                                                    <div class="no-image">📺</div>
                                                {/if}
                                                {#if anime.Sources && anime.Sources.length > 0}
                                                    <div class="source-badges">
                                                        {#each anime.Sources as src}
                                                            <span class="mini-badge {src.Language === 'en' ? 'en' : 'pt'}">
                                                                {src.Language === 'en' ? '🇺🇸' : '🇧🇷'}
                                                            </span>
                                                        {/each}
                                                    </div>
                                                {/if}
                                                <div class="card-overlay">
                                                    <span class="play-icon">▶</span>
                                                </div>
                                            </div>
                                            <div class="card-info">
                                                <div class="card-title">{anime.Title}</div>
                                                {#if anime.Source}
                                                    <div class="card-source">{anime.Source}</div>
                                                {/if}
                                            </div>
                                        </button>
                                    {/each}
                                </div>
                            {:else}
                            <!-- TRENDING SECTION (AniList HD) -->
                            {#if trendingAnimes.length > 0}
                                <div class="content-section">
                                    <h2 class="section-title">
                                        <span class="fire-icon">🔥</span> 
                                        Em Alta Agora
                                        <span class="title-badge anilist">AniList HD</span>
                                    </h2>
                                    <div class="anime-row">
                                        {#each trendingAnimes.slice(0, 10) as anime}
                                            <button type="button" class="anime-card-hd" onclick={() => { termoBusca = anime.title; pesquisar(); }}>
                                                <div class="card-poster-hd" style="--card-color: {anime.color || '#1a1f3a'}">
                                                    <img src={anime.image} alt={anime.title} loading="lazy" />
                                                    <div class="card-badges-hd">
                                                        {#if anime.isAiring}
                                                            <span class="badge-mini airing">🔴</span>
                                                        {/if}
                                                        <span class="badge-mini score">⭐{anime.score}</span>
                                                    </div>
                                                    <div class="card-overlay-hd">
                                                        <span class="play-icon">▶</span>
                                                    </div>
                                                </div>
                                                <div class="card-info-hd">
                                                    <div class="card-title-hd">{anime.title}</div>
                                                    <div class="card-meta-hd">
                                                        {anime.episodes ? `${anime.episodes} eps` : 'Em exibição'}
                                                        {#if anime.studio} • {anime.studio}{/if}
                                                    </div>
                                                </div>
                                            </button>
                                        {/each}
                                    </div>
                                </div>
                            {/if}
                            
                            <!-- POPULAR SECTION (Streaming Sources) -->
                            <div class="content-section">
                                <h2 class="section-title">
                                    <span class="fire-icon">📺</span> 
                                    Disponíveis para Assistir
                                    <span class="title-badge sources">AllAnime + AnimeFire</span>
                                </h2>
                                {#if carregando}
                                    <div class="loading-grid">
                                        {#each Array(12) as _, i}
                                            <div class="skeleton-card">
                                                <div class="skeleton-poster"></div>
                                                <div class="skeleton-title"></div>
                                            </div>
                                        {/each}
                                    </div>
                                {:else}
                                    <div class="anime-grid large">
                                        {#each topAnimes as anime (anime.Title)}
                                            <button 
                                                type="button" 
                                                class="anime-card" 
                                                onclick={() => openEpisodeSelection(anime)}
                                                onmouseenter={() => schedulePrefetch(anime)}
                                            >
                                                <div class="card-poster">
                                                    {#if anime.Image}
                                                        <img src={anime.Image} alt={anime.Title} loading="lazy" />
                                                    {:else}
                                                        <div class="no-image">📺</div>
                                                    {/if}
                                                    {#if anime.Source}
                                                        <div class="source-badges">
                                                            <span class="mini-badge {anime.Source === 'AllAnime' ? 'en' : 'pt'}">
                                                                {anime.Source === 'AllAnime' ? '🇺🇸' : '🇧🇷'}
                                                            </span>
                                                        </div>
                                                    {/if}
                                                    <div class="card-overlay">
                                                        <span class="play-icon">▶</span>
                                                    </div>
                                                </div>
                                                <div class="card-info">
                                                    <div class="card-title">{anime.Title}</div>
                                                    {#if anime.Source}
                                                        <div class="card-source">{anime.Source}</div>
                                                    {/if}
                                                </div>
                                            </button>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                            {/if}
                            
                            <!-- MANGA TAB -->
                            {:else if activeTab === 'manga'}
                                <div class="tab-content manga-coming-soon">
                                    <div class="coming-soon-card">
                                        <div class="coming-soon-icon">📚</div>
                                        <h2>Mangás em Breve!</h2>
                                        <p>Estamos trabalhando para trazer seus mangás favoritos para você.</p>
                                        <div class="coming-soon-features">
                                            <div class="feature">
                                                <span class="feature-icon">📖</span>
                                                <span>Leitura Online</span>
                                            </div>
                                            <div class="feature">
                                                <span class="feature-icon">⬇️</span>
                                                <span>Download Offline</span>
                                            </div>
                                            <div class="feature">
                                                <span class="feature-icon">🔖</span>
                                                <span>Marcadores</span>
                                            </div>
                                        </div>
                                        <button type="button" class="btn-notify" onclick={() => alert('Você será notificado quando disponível!')}>
                                            🔔 Notifique-me
                                        </button>
                                    </div>
                                </div>
                            
                            <!-- FRIENDS TAB (Discord Linking) -->
                            {:else if activeTab === 'friends'}
                                <div class="tab-content friends-tab">
                                    {#if !discordLinked}
                                        <!-- Não vinculado - Mostrar opções de conexão -->
                                        <div class="discord-connect-container">
                                            <div class="discord-connect-card">
                                                <div class="discord-glow"></div>
                                                <div class="discord-content">
                                                    <div class="discord-logo-animated">
                                                        <div class="logo-ring"></div>
                                                        <div class="logo-ring ring-2"></div>
                                                        <svg viewBox="0 0 24 24" width="64" height="64" class="discord-svg">
                                                            <path fill="#fff" d="M19.27 5.33C17.94 4.71 16.5 4.26 15 4a.09.09 0 0 0-.07.03c-.18.33-.39.76-.53 1.09a16.09 16.09 0 0 0-4.8 0c-.14-.34-.35-.76-.54-1.09c-.01-.02-.04-.03-.07-.03c-1.5.26-2.93.71-4.27 1.33c-.01 0-.02.01-.03.02c-2.72 4.07-3.47 8.03-3.1 11.95c0 .02.01.04.03.05c1.8 1.32 3.53 2.12 5.24 2.65c.03.01.06 0 .07-.02c.4-.55.76-1.13 1.07-1.74c.02-.04 0-.08-.04-.09c-.57-.22-1.11-.48-1.64-.78c-.04-.02-.04-.08-.01-.11c.11-.08.22-.17.33-.25c.02-.02.05-.02.07-.01c3.44 1.57 7.15 1.57 10.55 0c.02-.01.05-.01.07.01c.11.09.22.17.33.26c.04.03.04.09-.01.11c-.52.31-1.07.56-1.64.78c-.04.01-.05.06-.04.09c.32.61.68 1.19 1.07 1.74c.03.01.06.02.09.01c1.72-.53 3.45-1.33 5.25-2.65c.02-.01.03-.03.03-.05c.44-4.53-.73-8.46-3.1-11.95c-.01-.01-.02-.02-.04-.02zM8.52 14.91c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.84 2.12-1.89 2.12zm6.97 0c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.83 2.12-1.89 2.12z"/>
                                                        </svg>
                                                    </div>
                                                    
                                                    <h2 class="discord-title">Conecte com Amigos</h2>
                                                    <p class="discord-subtitle">Compartilhe o que você está assistindo com amigos do Discord!</p>
                                                    
                                                    <div class="discord-features">
                                                        <div class="feature-item">
                                                            <span class="feature-icon">📺</span>
                                                            <div class="feature-text">
                                                                <strong>Status Ao Vivo</strong>
                                                                <span>Amigos veem o que você está assistindo</span>
                                                            </div>
                                                        </div>
                                                        <div class="feature-item">
                                                            <span class="feature-icon">💬</span>
                                                            <div class="feature-text">
                                                                <strong>Recomendações</strong>
                                                                <span>Envie animes para seus amigos</span>
                                                            </div>
                                                        </div>
                                                        <div class="feature-item">
                                                            <span class="feature-icon">🏆</span>
                                                            <div class="feature-text">
                                                                <strong>Conquistas</strong>
                                                                <span>Compartilhe seu progresso</span>
                                                            </div>
                                                        </div>
                                                    </div>
                                                    
                                                    <div class="link-steps">
                                                        <div class="link-step">
                                                            <span class="step-num">1</span>
                                                            <span>Entre no servidor GoAnime no Discord</span>
                                                        </div>
                                                        <div class="link-step">
                                                            <span class="step-num">2</span>
                                                            <span>Use o comando <code>/vincular</code> no bot</span>
                                                        </div>
                                                        <div class="link-step">
                                                            <span class="step-num">3</span>
                                                            <span>Cole o código gerado abaixo</span>
                                                        </div>
                                                    </div>
                                                    
                                                    <div class="link-actions">
                                                        <a href={discordServerInvite || "https://discord.gg/goanime"} target="_blank" rel="noopener" class="btn-discord-join">
                                                            <svg viewBox="0 0 24 24" width="20" height="20">
                                                                <path fill="currentColor" d="M19.27 5.33C17.94 4.71 16.5 4.26 15 4a.09.09 0 0 0-.07.03c-.18.33-.39.76-.53 1.09a16.09 16.09 0 0 0-4.8 0c-.14-.34-.35-.76-.54-1.09c-.01-.02-.04-.03-.07-.03c-1.5.26-2.93.71-4.27 1.33c-.01 0-.02.01-.03.02c-2.72 4.07-3.47 8.03-3.1 11.95c0 .02.01.04.03.05c1.8 1.32 3.53 2.12 5.24 2.65c.03.01.06 0 .07-.02c.4-.55.76-1.13 1.07-1.74c.02-.04 0-.08-.04-.09c-.57-.22-1.11-.48-1.64-.78c-.04-.02-.04-.08-.01-.11c.11-.08.22-.17.33-.25c.02-.02.05-.02.07-.01c3.44 1.57 7.15 1.57 10.55 0c.02-.01.05-.01.07.01c.11.09.22.17.33.26c.04.03.04.09-.01.11c-.52.31-1.07.56-1.64.78c-.04.01-.05.06-.04.09c.32.61.68 1.19 1.07 1.74c.03.01.06.02.09.01c1.72-.53 3.45-1.33 5.25-2.65c.02-.01.03-.03.03-.05c.44-4.53-.73-8.46-3.1-11.95c-.01-.01-.02-.02-.04-.02zM8.52 14.91c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.84 2.12-1.89 2.12zm6.97 0c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.83 2.12-1.89 2.12z"/>
                                                            </svg>
                                                            Entrar no Servidor
                                                        </a>
                                                        <button type="button" class="btn-discord-link" onclick={openLinkModal}>
                                                            🔗 Vincular Conta
                                                        </button>
                                                    </div>
                                                    
                                                    <p class="discord-privacy">🔒 Sem login necessário - apenas um código!</p>
                                                </div>
                                            </div>
                                        </div>
                                    {:else}
                                        <!-- Vinculado - Mostrar perfil e amigos -->
                                        <div class="discord-profile-section">
                                            <div class="discord-profile-card">
                                                <div class="profile-background"></div>
                                                <div class="profile-content">
                                                    <div class="profile-avatar-wrapper">
                                                        {#if discordLinkInfo?.avatar}
                                                            <img src={discordLinkInfo.avatar} alt={discordLinkInfo.username} class="profile-avatar" />
                                                        {:else}
                                                            <div class="profile-avatar-placeholder">
                                                                <svg viewBox="0 0 24 24" width="40" height="40">
                                                                    <path fill="#fff" d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
                                                                </svg>
                                                            </div>
                                                        {/if}
                                                        <span class="online-badge"></span>
                                                    </div>
                                                    <div class="profile-details">
                                                        <span class="profile-name">{discordLinkInfo?.username || 'Usuário'}</span>
                                                        <span class="profile-tag">Vinculado em {discordLinkInfo?.linkedAt}</span>
                                                    </div>
                                                    <button type="button" class="btn-disconnect" onclick={unlinkDiscord}>
                                                        <svg viewBox="0 0 24 24" width="16" height="16">
                                                            <path fill="currentColor" d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                                                        </svg>
                                                        Desvincular
                                                    </button>
                                                </div>
                                            </div>
                                            
                                            <!-- Configurações de compartilhamento -->
                                            <div class="share-settings">
                                                <h3>⚙️ Configurações de Compartilhamento</h3>
                                                <div class="setting-row">
                                                    <div class="setting-info">
                                                        <span class="setting-title">📺 Mostrar o que estou assistindo</span>
                                                        <span class="setting-desc">Amigos veem quando você está assistindo</span>
                                                    </div>
                                                    <button 
                                                        type="button" 
                                                        class="toggle-btn {discordLinkInfo?.showStatus ? 'active' : ''}"
                                                        onclick={toggleShowStatus}
                                                        aria-label="Ativar/desativar mostrar status"
                                                    >
                                                        <span class="toggle-slider"></span>
                                                    </button>
                                                </div>
                                                <div class="setting-row">
                                                    <div class="setting-info">
                                                        <span class="setting-title">📤 Compartilhar recomendações</span>
                                                        <span class="setting-desc">Enviar animes para o servidor</span>
                                                    </div>
                                                    <button 
                                                        type="button" 
                                                        class="toggle-btn {discordLinkInfo?.shareAnimes ? 'active' : ''}"
                                                        onclick={toggleShareAnimes}
                                                        aria-label="Ativar/desativar compartilhamento"
                                                    >
                                                        <span class="toggle-slider"></span>
                                                    </button>
                                                </div>
                                            </div>
                                        </div>
                                        
                                        <!-- Atividade dos amigos -->
                                        <div class="section-header">
                                            <h2><span class="section-icon">👥</span> Amigos Assistindo</h2>
                                            <p>Veja o que seus amigos estão assistindo agora</p>
                                            <button type="button" class="btn-refresh" onclick={loadFriendsActivity}>
                                                🔄 Atualizar
                                            </button>
                                        </div>
                                        
                                        {#if loadingFriends}
                                            <div class="friends-loading">
                                                <div class="spinner"></div>
                                                <span>Carregando amigos...</span>
                                            </div>
                                        {:else if friendsActivity.length === 0}
                                            <div class="no-friends-activity">
                                                <span class="empty-icon">🎮</span>
                                                <p>Nenhum amigo assistindo agora</p>
                                                <span class="empty-hint">Quando amigos do servidor estiverem assistindo, eles aparecerão aqui!</span>
                                                <a href={discordServerInvite || "https://discord.gg/goanime"} target="_blank" rel="noopener" class="invite-friends-link">
                                                    Convidar amigos para o servidor →
                                                </a>
                                            </div>
                                        {:else}
                                            <div class="friends-activity-list">
                                                {#each friendsActivity as friend}
                                                    <div class="friend-activity-card">
                                                        <div class="friend-avatar-section">
                                                            {#if friend.avatar}
                                                                <img src={friend.avatar} alt={friend.username} class="friend-avatar" />
                                                            {:else}
                                                                <div class="friend-avatar-placeholder">👤</div>
                                                            {/if}
                                                            {#if friend.isOnline}
                                                                <span class="online-dot"></span>
                                                            {/if}
                                                        </div>
                                                        <div class="friend-info">
                                                            <span class="friend-name">{friend.username}</span>
                                                            <span class="friend-watching">
                                                                Assistindo <strong>{friend.animeTitle}</strong>
                                                            </span>
                                                            <span class="friend-episode">Episódio {friend.episodeNum}</span>
                                                        </div>
                                                        {#if friend.animeImage}
                                                            <img src={friend.animeImage} alt={friend.animeTitle} class="friend-anime-thumb" />
                                                        {/if}
                                                    </div>
                                                {/each}
                                            </div>
                                        {/if}
                                    {/if}
                                </div>
                            
                            <!-- COMMUNITY TAB -->
                            {:else if activeTab === 'community'}
                                <div class="tab-content community-tab">
                                    <div class="community-header">
                                        <h2>🌐 Comunidade GoAnime</h2>
                                        <p>Conecte-se com outros fãs de anime!</p>
                                    </div>
                                    
                                    <div class="community-links">
                                        <a href="https://discord.gg/goanime" target="_blank" class="community-card discord">
                                            <div class="community-icon">
                                                <svg viewBox="0 0 24 24" width="40" height="40">
                                                    <path fill="currentColor" d="M19.27 5.33C17.94 4.71 16.5 4.26 15 4a.09.09 0 0 0-.07.03c-.18.33-.39.76-.53 1.09a16.09 16.09 0 0 0-4.8 0c-.14-.34-.35-.76-.54-1.09c-.01-.02-.04-.03-.07-.03c-1.5.26-2.93.71-4.27 1.33c-.01 0-.02.01-.03.02c-2.72 4.07-3.47 8.03-3.1 11.95c0 .02.01.04.03.05c1.8 1.32 3.53 2.12 5.24 2.65c.03.01.06 0 .07-.02c.4-.55.76-1.13 1.07-1.74c.02-.04 0-.08-.04-.09c-.57-.22-1.11-.48-1.64-.78c-.04-.02-.04-.08-.01-.11c.11-.08.22-.17.33-.25c.02-.02.05-.02.07-.01c3.44 1.57 7.15 1.57 10.55 0c.02-.01.05-.01.07.01c.11.09.22.17.33.26c.04.03.04.09-.01.11c-.52.31-1.07.56-1.64.78c-.04.01-.05.06-.04.09c.32.61.68 1.19 1.07 1.74c.03.01.06.02.09.01c1.72-.53 3.45-1.33 5.25-2.65c.02-.01.03-.03.03-.05c.44-4.53-.73-8.46-3.1-11.95c-.01-.01-.02-.02-.04-.02zM8.52 14.91c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.84 2.12-1.89 2.12zm6.97 0c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.83 2.12-1.89 2.12z"/>
                                                </svg>
                                            </div>
                                            <div class="community-info">
                                                <h3>Discord</h3>
                                                <p>Junte-se à nossa comunidade no Discord</p>
                                                <span class="member-count">🟢 2.5k+ membros online</span>
                                            </div>
                                        </a>
                                        
                                        <a href="https://github.com/alvarorichard/GoAnime" target="_blank" class="community-card github">
                                            <div class="community-icon">
                                                <svg viewBox="0 0 24 24" width="40" height="40">
                                                    <path fill="currentColor" d="M12 2A10 10 0 0 0 2 12c0 4.42 2.87 8.17 6.84 9.5c.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34c-.46-1.16-1.11-1.47-1.11-1.47c-.91-.62.07-.6.07-.6c1 .07 1.53 1.03 1.53 1.03c.87 1.52 2.34 1.07 2.91.83c.09-.65.35-1.09.63-1.34c-2.22-.25-4.55-1.11-4.55-4.92c0-1.11.38-2 1.03-2.71c-.1-.25-.45-1.29.1-2.64c0 0 .84-.27 2.75 1.02c.79-.22 1.65-.33 2.5-.33c.85 0 1.71.11 2.5.33c1.91-1.29 2.75-1.02 2.75-1.02c.55 1.35.2 2.39.1 2.64c.65.71 1.03 1.6 1.03 2.71c0 3.82-2.34 4.66-4.57 4.91c.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0 0 12 2z"/>
                                                </svg>
                                            </div>
                                            <div class="community-info">
                                                <h3>GitHub</h3>
                                                <p>Contribua com o projeto open source</p>
                                                <span class="star-count">⭐ 500+ stars</span>
                                            </div>
                                        </a>
                                        
                                        <a href="https://twitter.com/goanime" target="_blank" class="community-card twitter">
                                            <div class="community-icon">
                                                <svg viewBox="0 0 24 24" width="40" height="40">
                                                    <path fill="currentColor" d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
                                                </svg>
                                            </div>
                                            <div class="community-info">
                                                <h3>X (Twitter)</h3>
                                                <p>Siga para atualizações e novidades</p>
                                                <span class="follower-count">📢 Novidades diárias</span>
                                            </div>
                                        </a>
                                    </div>
                                    
                                    <div class="community-stats">
                                        <div class="stat-card">
                                            <span class="stat-value">50K+</span>
                                            <span class="stat-label">Usuários</span>
                                        </div>
                                        <div class="stat-card">
                                            <span class="stat-value">10K+</span>
                                            <span class="stat-label">Animes</span>
                                        </div>
                                        <div class="stat-card">
                                            <span class="stat-value">1M+</span>
                                            <span class="stat-label">Episódios Assistidos</span>
                                        </div>
                                    </div>
                                </div>
                            {/if}
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    {/if}
    
    <!-- SHARE RECOMMENDATION MODAL -->
    {#if showShareModal && shareAnime}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div class="modal-overlay" onclick={() => showShareModal = false}>
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
            <div class="share-modal" onclick={(e) => e.stopPropagation()}>
                <button type="button" class="modal-close" onclick={() => showShareModal = false}>✕</button>
                
                <div class="share-modal-header">
                    <h2>📤 Compartilhar Recomendação</h2>
                    <p>Envie para seus amigos do Discord</p>
                </div>
                
                <div class="share-anime-preview">
                    <img src={shareAnime.image || shareAnime.Image} alt={shareAnime.title || shareAnime.Title} />
                    <div class="share-anime-info">
                        <h3>{shareAnime.title || shareAnime.Title}</h3>
                        {#if shareAnime.score}
                            <span class="share-score">⭐ {shareAnime.score}</span>
                        {/if}
                    </div>
                </div>
                
                <div class="share-message-input">
                    <label for="shareMessage">Sua mensagem:</label>
                    <textarea 
                        id="shareMessage"
                        bind:value={shareMessage}
                        placeholder="Por que você recomenda esse anime? (ex: A história é incrível! 🔥)"
                        rows="3"
                    ></textarea>
                </div>
                
                <div class="share-modal-actions">
                    <button type="button" class="btn-cancel" onclick={() => showShareModal = false}>
                        Cancelar
                    </button>
                    <button 
                        type="button" 
                        class="btn-send-share" 
                        onclick={sendRecommendation}
                        disabled={!shareMessage.trim()}
                    >
                        <svg viewBox="0 0 24 24" width="20" height="20">
                            <path fill="currentColor" d="M19.27 5.33C17.94 4.71 16.5 4.26 15 4a.09.09 0 0 0-.07.03c-.18.33-.39.76-.53 1.09a16.09 16.09 0 0 0-4.8 0c-.14-.34-.35-.76-.54-1.09c-.01-.02-.04-.03-.07-.03c-1.5.26-2.93.71-4.27 1.33c-.01 0-.02.01-.03.02c-2.72 4.07-3.47 8.03-3.1 11.95c0 .02.01.04.03.05c1.8 1.32 3.53 2.12 5.24 2.65c.03.01.06 0 .07-.02c.4-.55.76-1.13 1.07-1.74c.02-.04 0-.08-.04-.09c-.57-.22-1.11-.48-1.64-.78c-.04-.02-.04-.08-.01-.11c.11-.08.22-.17.33-.25c.02-.02.05-.02.07-.01c3.44 1.57 7.15 1.57 10.55 0c.02-.01.05-.01.07.01c.11.09.22.17.33.26c.04.03.04.09-.01.11c-.52.31-1.07.56-1.64.78c-.04.01-.05.06-.04.09c.32.61.68 1.19 1.07 1.74c.03.01.06.02.09.01c1.72-.53 3.45-1.33 5.25-2.65c.02-.01.03-.03.03-.05c.44-4.53-.73-8.46-3.1-11.95c-.01-.01-.02-.02-.04-.02zM8.52 14.91c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.84 2.12-1.89 2.12zm6.97 0c-1.03 0-1.89-.95-1.89-2.12s.84-2.12 1.89-2.12c1.06 0 1.9.96 1.89 2.12c0 1.17-.83 2.12-1.89 2.12z"/>
                        </svg>
                        Enviar no Discord
                    </button>
                </div>
            </div>
        </div>
    {/if}
    
    <!-- DISCORD LINK CODE MODAL -->
    {#if showLinkModal}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div class="modal-overlay" onclick={closeLinkModal} role="presentation">
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
            <div class="discord-link-modal" onclick={(e) => e.stopPropagation()}>
                <button type="button" class="modal-close" onclick={closeLinkModal}>✕</button>
                
                <div class="link-modal-header">
                    <div class="link-icon">🔗</div>
                    <h2>Vincular Conta Discord</h2>
                    <p>Cole o código gerado pelo bot no servidor</p>
                </div>
                
                <div class="link-code-input">
                    <label for="linkCodeInput">Código de Vinculação</label>
                    <input 
                        type="text" 
                        id="linkCodeInput" 
                        bind:value={linkCode}
                        placeholder="ANIME-XXXXXXXX"
                        class="code-input"
                        onkeydown={(e) => e.key === 'Enter' && linkWithCode()}
                    />
                    {#if linkError}
                        <span class="link-error">❌ {linkError}</span>
                    {/if}
                </div>
                
                <div class="link-help">
                    <p>💡 <strong>Como obter o código?</strong></p>
                    <ol>
                        <li>Entre no <a href={discordServerInvite || "https://discord.gg/goanime"} target="_blank" rel="noopener">servidor GoAnime</a></li>
                        <li>Vá no canal <code>#vincular</code></li>
                        <li>Digite <code>/vincular</code></li>
                        <li>Cole o código aqui!</li>
                    </ol>
                </div>
                
                <div class="link-modal-actions">
                    <button type="button" class="btn-cancel" onclick={closeLinkModal}>
                        Cancelar
                    </button>
                    <button 
                        type="button" 
                        class="btn-link-confirm" 
                        onclick={linkWithCode}
                        disabled={linkLoading || !linkCode.trim()}
                    >
                        {#if linkLoading}
                            <span class="spinner-small"></span>
                            Vinculando...
                        {:else}
                            ✓ Vincular Conta
                        {/if}
                    </button>
                </div>
            </div>
        </div>
    {/if}
</main>

<style>
    * { box-sizing: border-box; }
    
    :global(body) {
        margin: 0;
        padding: 0;
        background: #0a0e27;
        color: #fff;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif;
    }

    main {
        width: 100%;
        height: 100vh;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    /* ============================================
       SPLASH SCREEN - Netflix Style
       ============================================ */
    .splash-screen {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: linear-gradient(135deg, #0a0e27 0%, #1a1a2e 50%, #16213e 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9999;
        transition: opacity 0.3s ease-out;
        will-change: opacity;
    }
    
    .splash-screen.fade-out {
        opacity: 0;
        pointer-events: none;
    }
    
    .splash-content {
        text-align: center;
        position: relative;
        z-index: 2;
        will-change: transform;
    }
    
    .splash-logo {
        margin-bottom: 30px;
        animation: splash-pulse 1.5s ease-in-out infinite;
    }
    
    @keyframes splash-pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.03); }
    }
    
    .splash-logo-icon {
        position: relative;
        display: inline-block;
        margin-bottom: 10px;
    }
    
    .splash-emoji {
        font-size: 64px;
        display: block;
        animation: splash-bounce 0.8s ease-in-out infinite;
    }
    
    @keyframes splash-bounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-8px); }
    }
    
    .splash-glow {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        width: 100px;
        height: 100px;
        background: radial-gradient(circle, rgba(245, 87, 108, 0.4) 0%, transparent 70%);
        border-radius: 50%;
        animation: splash-glow-pulse 1.5s ease-in-out infinite;
    }
    
    @keyframes splash-glow-pulse {
        0%, 100% { opacity: 0.5; transform: translate(-50%, -50%) scale(1); }
        50% { opacity: 1; transform: translate(-50%, -50%) scale(1.2); }
    }
    
    .splash-title {
        font-size: 48px;
        font-weight: 800;
        margin: 0;
        letter-spacing: -1px;
    }
    
    .splash-go {
        color: #f5576c;
        text-shadow: 0 0 30px rgba(245, 87, 108, 0.5);
    }
    
    .splash-anime {
        color: #fff;
    }
    
    .splash-loader {
        width: 280px;
        margin: 0 auto;
    }
    
    .loader-bar {
        height: 4px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 2px;
        overflow: hidden;
        margin-bottom: 15px;
    }
    
    .loader-progress {
        height: 100%;
        background: linear-gradient(90deg, #f5576c 0%, #ff6b9d 100%);
        border-radius: 2px;
        transition: width 0.15s ease-out;
        will-change: width;
    }
    
    .loader-status {
        font-size: 13px;
        color: rgba(255, 255, 255, 0.6);
        margin: 0;
    }

    /* LOADING SPINNER */
    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    .spinner {
        width: 40px;
        height: 40px;
        border: 4px solid rgba(255,255,255,0.1);
        border-top-color: #f5576c;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin: 0 auto 15px;
    }

    /* LOGIN */
    /* Legacy login styles removed - using modern fullscreen design */

    /* ============================================
       LOGIN SCREEN - MODERN FULLSCREEN DESIGN
       ============================================ */
    .login-screen {
        position: relative;
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
    }

    /* Animated Background */
    .login-bg {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 0;
    }

    .bg-gradient {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: 
            radial-gradient(ellipse at 20% 20%, rgba(245, 87, 108, 0.15) 0%, transparent 50%),
            radial-gradient(ellipse at 80% 80%, rgba(79, 172, 254, 0.15) 0%, transparent 50%),
            radial-gradient(ellipse at 50% 50%, rgba(240, 147, 251, 0.1) 0%, transparent 60%),
            linear-gradient(180deg, #0a0e27 0%, #131832 50%, #0a0e27 100%);
    }

    .bg-particles {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        overflow: hidden;
    }

    .particle {
        position: absolute;
        width: 4px;
        height: 4px;
        background: rgba(245, 87, 108, 0.5);
        border-radius: 50%;
        left: var(--x);
        animation: float var(--duration) ease-in-out infinite;
        animation-delay: var(--delay);
        opacity: 0.6;
    }

    @keyframes float {
        0%, 100% {
            transform: translateY(100vh) scale(0);
            opacity: 0;
        }
        10% {
            opacity: 0.6;
        }
        90% {
            opacity: 0.6;
        }
        100% {
            transform: translateY(-100px) scale(1);
            opacity: 0;
        }
    }

    /* Login Content */
    .login-content {
        position: relative;
        z-index: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 30px;
        padding: 40px;
        width: 100%;
        max-width: 500px;
    }

    /* Branding */
    .login-branding {
        text-align: center;
    }

    .logo-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 15px;
        margin-bottom: 15px;
    }

    .logo-icon {
        position: relative;
        width: 100px;
        height: 100px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .logo-emoji {
        font-size: 4rem;
        z-index: 1;
        animation: logo-bounce 3s ease-in-out infinite;
    }

    @keyframes logo-bounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-10px); }
    }

    .logo-glow {
        position: absolute;
        width: 100%;
        height: 100%;
        background: radial-gradient(circle, rgba(245, 87, 108, 0.4) 0%, transparent 70%);
        border-radius: 50%;
        animation: glow-pulse 2s ease-in-out infinite;
    }

    @keyframes glow-pulse {
        0%, 100% { transform: scale(1); opacity: 0.5; }
        50% { transform: scale(1.2); opacity: 0.8; }
    }

    .logo-text {
        font-size: 3rem;
        font-weight: 800;
        margin: 0;
        letter-spacing: -1px;
    }

    .logo-go {
        color: #fff;
    }

    .logo-anime {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 50%, #4facfe 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .login-tagline {
        font-size: 1.1rem;
        color: #888;
        margin: 0;
    }

    /* Login Card Modern */
    .login-card-modern {
        width: 100%;
        background: rgba(26, 31, 58, 0.8);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        backdrop-filter: blur(20px);
        overflow: hidden;
        box-shadow: 
            0 25px 50px rgba(0, 0, 0, 0.5),
            0 0 100px rgba(245, 87, 108, 0.1);
    }

    .card-header {
        padding: 25px 30px;
        background: rgba(0, 0, 0, 0.2);
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .card-header h2 {
        margin: 0 0 5px 0;
        font-size: 1.5rem;
        font-weight: 600;
    }

    .card-header p {
        margin: 0;
        color: #888;
        font-size: 0.95rem;
    }

    .card-body {
        padding: 30px;
    }

    .input-group {
        margin-bottom: 25px;
    }

    .input-group label {
        display: block;
        margin-bottom: 10px;
        color: #aaa;
        font-size: 0.9rem;
        font-weight: 500;
    }

    .input-wrapper {
        display: flex;
        align-items: center;
        background: rgba(10, 14, 39, 0.8);
        border: 2px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        padding: 4px;
        transition: all 0.3s;
    }

    .input-wrapper:focus-within {
        border-color: #f5576c;
        box-shadow: 0 0 20px rgba(245, 87, 108, 0.2);
    }

    .input-icon {
        padding: 0 15px;
        font-size: 1.2rem;
    }

    .input-modern {
        flex: 1;
        padding: 14px 10px;
        background: transparent;
        border: none;
        color: #fff;
        font-size: 1rem;
    }

    .input-modern:focus {
        outline: none;
    }

    .input-modern::placeholder {
        color: #555;
    }

    .avatar-label {
        display: block;
        margin-bottom: 15px;
        color: #aaa;
        font-size: 0.9rem;
        font-weight: 500;
    }

    .avatar-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 12px;
    }

    .avatar-option {
        aspect-ratio: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(10, 14, 39, 0.8);
        border: 2px solid rgba(255, 255, 255, 0.1);
        border-radius: 16px;
        cursor: pointer;
        transition: all 0.3s;
        position: relative;
        overflow: hidden;
    }

    .avatar-option:hover {
        border-color: rgba(245, 87, 108, 0.5);
        transform: translateY(-3px);
        background: rgba(245, 87, 108, 0.1);
    }

    .avatar-option.selected {
        border-color: #f5576c;
        background: rgba(245, 87, 108, 0.2);
        box-shadow: 0 0 25px rgba(245, 87, 108, 0.3);
    }

    .avatar-emoji {
        font-size: 2rem;
        transition: transform 0.3s;
    }

    .avatar-option:hover .avatar-emoji {
        transform: scale(1.1);
    }

    .avatar-check {
        position: absolute;
        bottom: 5px;
        right: 5px;
        width: 20px;
        height: 20px;
        background: #f5576c;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.7rem;
        font-weight: bold;
        animation: pop 0.3s ease-out;
    }

    @keyframes pop {
        0% { transform: scale(0); }
        50% { transform: scale(1.2); }
        100% { transform: scale(1); }
    }

    .card-footer {
        padding: 20px 30px 30px;
    }

    .btn-enter {
        width: 100%;
        padding: 16px 30px;
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        border: none;
        border-radius: 12px;
        color: white;
        font-size: 1.1rem;
        font-weight: 600;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        transition: all 0.3s;
        box-shadow: 0 10px 30px rgba(245, 87, 108, 0.3);
    }

    .btn-enter:hover:not(:disabled) {
        transform: translateY(-3px);
        box-shadow: 0 15px 40px rgba(245, 87, 108, 0.4);
    }

    .btn-enter:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .btn-arrow {
        font-size: 1.3rem;
        transition: transform 0.3s;
    }

    .btn-enter:hover:not(:disabled) .btn-arrow {
        transform: translateX(5px);
    }

    /* Features */
    .login-features {
        display: flex;
        gap: 30px;
        flex-wrap: wrap;
        justify-content: center;
    }

    .feature {
        display: flex;
        align-items: center;
        gap: 10px;
        color: #888;
        font-size: 0.9rem;
    }

    .feature-icon {
        font-size: 1.2rem;
    }

    /* APP */
    .app {
        display: flex;
        flex-direction: column;
        height: 100%;
    }

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

    :global(.header h1) {
        margin: 0;
        font-size: 1.5rem;
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
        background: rgba(26, 31, 58, 1);
        border-color: rgba(245, 87, 108, 0.3);
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
        display: block;
        width: 100%;
        padding: 14px 20px;
        background: transparent;
        border: none;
        color: #fff;
        font-size: 0.95rem;
        text-align: left;
        cursor: pointer;
        transition: background 0.2s;
    }

    .dropdown-item:hover {
        background: rgba(245, 87, 108, 0.2);
    }

    .btn-logo {
        display: flex;
        align-items: center;
        gap: 10px;
        background: none;
        border: none;
        color: #fff;
        cursor: pointer;
        padding: 8px 12px;
        border-radius: 10px;
        transition: all 0.3s;
    }

    .btn-logo:hover {
        background: rgba(255, 255, 255, 0.05);
    }

    .logo-icon-small {
        font-size: 1.5rem;
    }

    .logo-text-small {
        font-size: 1.3rem;
        font-weight: 700;
        letter-spacing: -0.5px;
    }

    .logo-text-small .go {
        color: #fff;
    }

    .logo-text-small .anime {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    /* ============================================
       HERO SECTION - MODERN FULLSCREEN
       ============================================ */
    .hero-section-modern {
        position: relative;
        width: 100%;
        min-height: 50vh;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        padding: 60px 20px;
    }

    .hero-bg-effects {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 0;
    }

    .hero-gradient {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: 
            radial-gradient(ellipse at 30% 0%, rgba(245, 87, 108, 0.2) 0%, transparent 50%),
            radial-gradient(ellipse at 70% 100%, rgba(79, 172, 254, 0.15) 0%, transparent 50%),
            linear-gradient(180deg, rgba(10, 14, 39, 0) 0%, #0a0e27 100%);
    }

    .hero-grid-pattern {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-image: 
            linear-gradient(rgba(255, 255, 255, 0.02) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
        background-size: 50px 50px;
        mask-image: radial-gradient(ellipse at center, black 0%, transparent 70%);
        -webkit-mask-image: radial-gradient(ellipse at center, black 0%, transparent 70%);
    }

    .hero-content-centered {
        position: relative;
        z-index: 1;
        text-align: center;
        animation: fadeInUp 0.8s ease-out;
    }

    @keyframes fadeInUp {
        from {
            opacity: 0;
            transform: translateY(30px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .hero-logo {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 10px;
        margin-bottom: 20px;
    }

    .hero-emoji {
        font-size: clamp(3rem, 8vw, 5rem);
        animation: hero-float 3s ease-in-out infinite;
    }

    @keyframes hero-float {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-15px); }
    }

    .hero-brand {
        font-size: clamp(2.5rem, 8vw, 4.5rem);
        font-weight: 800;
        margin: 0;
        letter-spacing: -2px;
    }

    .brand-go {
        color: #fff;
    }

    .brand-anime {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 50%, #4facfe 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        background-size: 200% 200%;
        animation: gradient-shift 5s ease infinite;
    }

    @keyframes gradient-shift {
        0%, 100% { background-position: 0% 50%; }
        50% { background-position: 100% 50%; }
    }

    .hero-tagline {
        font-size: clamp(1rem, 3vw, 1.4rem);
        color: #888;
        margin: 0 0 30px 0;
    }

    .hero-stats {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 25px;
        flex-wrap: wrap;
    }

    .stat {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 5px;
    }

    .stat-number {
        font-size: clamp(1.5rem, 4vw, 2rem);
        font-weight: 700;
        background: linear-gradient(135deg, #f5576c, #f093fb);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .stat-label {
        font-size: 0.85rem;
        color: #666;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .stat-divider {
        width: 1px;
        height: 40px;
        background: linear-gradient(180deg, transparent, rgba(255, 255, 255, 0.2), transparent);
    }

    /* USER VIEWS (Favorites, History, Settings) */
    .user-view {
        max-width: 1400px;
        margin: 0 auto;
        padding: clamp(30px, 5vw, 60px) clamp(20px, 4vw, 40px);
        width: 100%;
        min-height: 100%;
    }

    .user-view h2 {
        font-size: clamp(1.5rem, 3vw, 2rem);
        margin-bottom: 30px;
        padding-bottom: 15px;
        border-bottom: 2px solid rgba(245, 87, 108, 0.3);
        background: linear-gradient(90deg, #f5576c, #f093fb);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .empty-state {
        text-align: center;
        padding: 80px 20px;
        color: #888;
        background: rgba(26, 31, 58, 0.4);
        border-radius: 16px;
        border: 1px dashed rgba(255, 255, 255, 0.1);
    }

    .empty-state p {
        margin: 10px 0;
        font-size: 1.1rem;
    }

    /* FAVORITES GRID */
    .user-view .anime-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
        gap: 24px;
        padding-bottom: 40px;
    }

    .user-view .anime-card {
        position: relative;
        background: linear-gradient(145deg, rgba(26, 31, 58, 0.9), rgba(20, 24, 45, 0.95));
        border-radius: 12px;
        overflow: hidden;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        border: 1px solid rgba(255, 255, 255, 0.08);
    }

    .user-view .anime-card:hover {
        transform: translateY(-8px);
        box-shadow: 0 15px 40px rgba(245, 87, 108, 0.2);
        border-color: rgba(245, 87, 108, 0.3);
    }

    .user-view .anime-card img {
        width: 100%;
        aspect-ratio: 3/4;
        object-fit: cover;
        transition: transform 0.3s ease;
    }

    .user-view .anime-card:hover img {
        transform: scale(1.05);
    }

    .user-view .anime-card .no-image {
        width: 100%;
        aspect-ratio: 3/4;
        display: flex;
        align-items: center;
        justify-content: center;
        background: linear-gradient(135deg, #1a1f3a, #252a4d);
        font-size: 3rem;
    }

    .user-view .anime-card .anime-title {
        padding: 14px;
        font-size: 0.9rem;
        font-weight: 500;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        background: rgba(0, 0, 0, 0.3);
    }

    .btn-fav {
        position: absolute;
        top: 10px;
        right: 10px;
        background: rgba(0, 0, 0, 0.6);
        border: none;
        border-radius: 50%;
        width: 36px;
        height: 36px;
        font-size: 1.2rem;
        cursor: pointer;
        transition: all 0.2s;
        opacity: 0.5;
    }

    .btn-fav:hover,
    .btn-fav.active {
        opacity: 1;
        background: rgba(245, 87, 108, 0.8);
    }

    /* HISTORY LIST */
    .history-list {
        display: flex;
        flex-direction: column;
        gap: 15px;
    }

    .history-item {
        display: flex;
        align-items: center;
        gap: 20px;
        background: #1a1f3a;
        padding: 15px;
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .history-item:hover {
        background: #252a4d;
    }

    .history-thumb {
        width: 80px;
        height: 60px;
        object-fit: cover;
        border-radius: 8px;
        flex-shrink: 0;
    }

    .history-thumb.no-image {
        display: flex;
        align-items: center;
        justify-content: center;
        background: #252a4d;
        font-size: 1.5rem;
    }

    .history-info {
        flex: 1;
    }

    .history-anime {
        font-weight: 600;
        margin-bottom: 5px;
    }

    .history-episode {
        color: #f5576c;
        font-size: 0.9rem;
    }

    .history-date {
        color: #888;
        font-size: 0.8rem;
        margin-top: 5px;
    }

    /* SETTINGS */
    .settings-view {
        max-width: 600px;
    }

    .settings-group {
        background: #1a1f3a;
        border-radius: 16px;
        padding: 25px;
        margin-bottom: 25px;
    }

    .settings-group h3 {
        margin: 0 0 20px 0;
        font-size: 1.2rem;
        color: #f5576c;
    }

    .settings-desc {
        color: #888;
        font-size: 0.9rem;
        margin-bottom: 20px;
    }

    .setting-item {
        margin-bottom: 20px;
    }

    .setting-item label {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 1rem;
        cursor: pointer;
    }

    .setting-item input[type="checkbox"] {
        width: 20px;
        height: 20px;
        accent-color: #f5576c;
    }

    .setting-item select {
        width: 100%;
        padding: 12px;
        background: #0a0e27;
        border: 1px solid #444;
        border-radius: 8px;
        color: #fff;
        font-size: 1rem;
        margin-top: 8px;
    }

    .backup-buttons {
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
    }

    .btn-secondary {
        padding: 12px 24px;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid #444;
        border-radius: 8px;
        color: #fff;
        font-size: 0.95rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-secondary:hover {
        background: rgba(255, 255, 255, 0.2);
        border-color: #666;
    }

    /* IMPORT/EXPORT MODAL */
    .import-export-modal {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 2000;
    }

    .modal-content {
        background: #1a1f3a;
        border-radius: 16px;
        padding: 30px;
        max-width: 500px;
        width: 90%;
        max-height: 80vh;
        overflow-y: auto;
    }

    .modal-content h4 {
        margin: 0 0 20px 0;
        font-size: 1.3rem;
    }

    .modal-content textarea {
        width: 100%;
        height: 150px;
        padding: 15px;
        background: #0a0e27;
        border: 1px solid #444;
        border-radius: 8px;
        color: #fff;
        font-family: monospace;
        font-size: 0.85rem;
        resize: vertical;
        margin-bottom: 15px;
    }

    .modal-content p {
        color: #ccc;
        margin-bottom: 10px;
    }

    .export-section,
    .import-section {
        margin-bottom: 25px;
    }

    .btn-close {
        display: block;
        width: 100%;
        padding: 12px;
        margin-top: 15px;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid #444;
        border-radius: 8px;
        color: #fff;
        cursor: pointer;
    }

    .btn-close:hover {
        background: rgba(255, 255, 255, 0.15);
    }

    /* === SOURCES STATUS SECTION === */
    .sources-status {
        border: 1px solid rgba(245, 87, 108, 0.3);
    }

    .cache-overview {
        display: flex;
        gap: 20px;
        margin: 15px 0;
        flex-wrap: wrap;
    }

    .cache-stat {
        background: rgba(0, 0, 0, 0.3);
        padding: 12px 20px;
        border-radius: 8px;
        display: flex;
        gap: 10px;
        align-items: center;
    }

    .stat-label {
        color: #888;
        font-size: 0.9rem;
    }

    .stat-value {
        color: #4dff88;
        font-weight: bold;
        font-size: 1.1rem;
    }

    .sources-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 15px;
        margin: 20px 0;
    }

    .source-status-card {
        background: rgba(0, 0, 0, 0.3);
        border-radius: 12px;
        padding: 15px;
        border: 2px solid transparent;
        transition: all 0.3s;
    }

    .source-status-card.available {
        border-color: rgba(77, 255, 136, 0.3);
    }

    .source-status-card.unavailable {
        border-color: rgba(255, 77, 77, 0.3);
        background: rgba(255, 77, 77, 0.1);
    }

    .source-status-card .source-header {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 10px;
    }

    .source-status-card .source-icon {
        font-size: 1.2rem;
    }

    .source-status-card .source-name {
        font-weight: bold;
        font-size: 1rem;
    }

    .source-status-card .source-details {
        display: flex;
        flex-direction: column;
        gap: 5px;
        font-size: 0.85rem;
    }

    .source-status-card .cached-count {
        color: #4dff88;
    }

    .source-status-card .fail-count {
        color: #ff6b6b;
    }

    .source-status-card .retry-time {
        color: #ffa500;
    }

    .source-status-card .last-error {
        color: #888;
        font-size: 0.8rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .cache-actions {
        display: flex;
        gap: 15px;
        margin-top: 20px;
        flex-wrap: wrap;
    }

    .btn-warning {
        padding: 10px 20px;
        background: rgba(255, 165, 0, 0.2);
        border: 1px solid #ffa500;
        border-radius: 8px;
        color: #ffa500;
        font-size: 0.9rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-warning:hover {
        background: rgba(255, 165, 0, 0.3);
    }

    .btn-danger {
        padding: 10px 20px;
        background: rgba(255, 77, 77, 0.2);
        border: 1px solid #ff4d4d;
        border-radius: 8px;
        color: #ff4d4d;
        font-size: 0.9rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-danger:hover {
        background: rgba(255, 77, 77, 0.3);
    }

    .no-stats {
        color: #888;
        font-style: italic;
        text-align: center;
        padding: 20px;
    }

    .user-avatar {
        font-size: 1.2rem;
    }

    .user-name {
        font-weight: 500;
    }

    .main-content {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
        padding: 0;
        display: flex;
        flex-direction: column;
        scroll-behavior: smooth;
        /* GPU acceleration for smooth scrolling */
        will-change: scroll-position;
        -webkit-overflow-scrolling: touch;
        /* Custom scrollbar */
        scrollbar-width: thin;
        scrollbar-color: rgba(245, 87, 108, 0.5) rgba(10, 14, 39, 0.8);
    }

    .main-content::-webkit-scrollbar {
        width: 6px;
    }

    .main-content::-webkit-scrollbar-track {
        background: transparent;
    }

    .main-content::-webkit-scrollbar-thumb {
        background: rgba(245, 87, 108, 0.4);
        border-radius: 3px;
    }

    .main-content::-webkit-scrollbar-thumb:hover {
        background: rgba(245, 87, 108, 0.7);
    }

    /* VIEW TRANSITIONS */
    .user-view,
    .anime-detail {
        animation: fadeSlideIn 0.25s ease-out;
    }

    @keyframes fadeSlideIn {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    /* HOME VIEW - RESPONSIVE FULLSCREEN */
    .home-view {
        width: 100%;
        min-height: 100%;
        display: flex;
        flex-direction: column;
    }

    /* Legacy hero styles removed - using modern hero-section-modern */

    @keyframes shimmer {
        0%, 100% { filter: brightness(1); }
        50% { filter: brightness(1.2); }
    }

    :global(.search-box) {
        margin-bottom: 20px;
    }

    .search-wrapper {
        display: flex;
        align-items: center;
        background: rgba(26, 31, 58, 0.8);
        border: 2px solid #444;
        border-radius: 50px;
        padding: 5px;
        max-width: 700px;
        margin: 0 auto;
        transition: all 0.3s;
    }

    .search-wrapper:focus-within {
        border-color: #f5576c;
        box-shadow: 0 0 20px rgba(245, 87, 108, 0.3);
    }

    .search-icon {
        padding: 0 15px;
        font-size: 1.2rem;
    }

    .search-input {
        flex: 1;
        padding: 14px 10px;
        background: transparent;
        border: none;
        color: #fff;
        font-size: 1rem;
    }

    .search-input:focus {
        outline: none;
    }

    .search-input::placeholder {
        color: #666;
    }

    .btn-search {
        padding: 14px 35px;
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        border: none;
        border-radius: 50px;
        color: white;
        font-weight: bold;
        cursor: pointer;
        transition: all 0.3s;
    }

    .btn-search:hover:not(:disabled) {
        transform: scale(1.05);
        box-shadow: 0 4px 15px rgba(245, 87, 108, 0.5);
    }

    /* QUICK PILLS (Popular searches) */
    .quick-pills {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        margin-top: 15px;
        flex-wrap: wrap;
    }

    .pills-label {
        color: #666;
        font-size: 0.85rem;
        margin-right: 4px;
    }

    .pill {
        padding: 6px 14px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        color: #aaa;
        font-size: 0.8rem;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .pill:hover {
        background: rgba(245, 87, 108, 0.15);
        border-color: rgba(245, 87, 108, 0.3);
        color: #fff;
    }

    /* GENRE CHIPS - Modern & Compact */
    .genre-chips-container {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        margin-top: 20px;
        flex-wrap: wrap;
        max-width: 900px;
        margin-left: auto;
        margin-right: auto;
    }

    .chips-label {
        color: #666;
        font-size: 0.85rem;
        flex-shrink: 0;
    }

    .genre-chips {
        display: flex;
        align-items: center;
        gap: 6px;
        flex-wrap: wrap;
        justify-content: center;
    }

    .genre-chip {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 5px 12px;
        background: transparent;
        border: 1px solid rgba(255, 255, 255, 0.12);
        border-radius: 18px;
        color: #888;
        font-size: 0.78rem;
        cursor: pointer;
        transition: all 0.2s ease;
        white-space: nowrap;
    }

    .genre-chip:hover {
        background: rgba(245, 87, 108, 0.12);
        border-color: rgba(245, 87, 108, 0.35);
        color: #f5576c;
        transform: translateY(-1px);
    }

    .genre-chip .chip-icon {
        font-size: 0.9rem;
        line-height: 1;
    }

    .genre-chip .chip-text {
        font-weight: 500;
    }

    .genre-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        background: rgba(245, 87, 108, 0.2);
        border: 1px solid rgba(245, 87, 108, 0.4);
        border-radius: 20px;
        font-size: 0.9rem;
        margin-right: 10px;
    }

    /* Responsive genre chips */
    @media (max-width: 600px) {
        .genre-chips-container {
            flex-direction: column;
            gap: 8px;
        }
        
        .genre-chips {
            gap: 5px;
        }
        
        .genre-chip {
            padding: 4px 10px;
            font-size: 0.72rem;
        }
        
        .genre-chip .chip-icon {
            font-size: 0.8rem;
        }
        
        .quick-pills {
            gap: 6px;
        }
        
        .pill {
            padding: 5px 10px;
            font-size: 0.75rem;
        }
    }

    /* SECTIONS - Responsivo */
    :global(.results-section),
    :global(.popular-section) {
        padding: clamp(15px, 4vh, 40px) clamp(15px, 4vw, 50px);
        flex: 1;
        overflow-y: auto;
    }

    :global(.section-header) {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 25px;
    }

    .section-title {
        margin: 0 0 clamp(15px, 3vh, 30px) 0;
        font-size: clamp(1.2rem, 3vw, 2rem);
        display: flex;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap;
    }

    .fire-icon {
        animation: pulse 1.5s ease-in-out infinite;
    }

    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }

    .title-badge {
        font-size: 0.7rem;
        padding: 4px 10px;
        background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
        border-radius: 20px;
        font-weight: 600;
    }

    :global(.btn-clear) {
        padding: 10px 20px;
        background: rgba(245, 87, 108, 0.1);
        border: 1px solid #f5576c;
        border-radius: 25px;
        color: #f5576c;
        cursor: pointer;
        transition: all 0.2s;
        font-weight: 600;
    }

    :global(.btn-clear:hover) {
        background: #f5576c;
        color: white;
    }

    /* ANIME GRID - Responsivo */
    .anime-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(clamp(140px, 15vw, 200px), 1fr));
        gap: clamp(12px, 2vw, 25px);
    }

    .anime-grid.large {
        grid-template-columns: repeat(auto-fill, minmax(clamp(150px, 18vw, 220px), 1fr));
        gap: clamp(15px, 2.5vw, 35px);
    }

    /* SKELETON LOADING - Responsivo */
    .loading-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(clamp(150px, 18vw, 220px), 1fr));
        gap: clamp(15px, 2.5vw, 35px);
    }

    .skeleton-card {
        background: #1a1f3a;
        border-radius: 12px;
        overflow: hidden;
    }

    .skeleton-poster {
        aspect-ratio: 2/3;
        background: linear-gradient(90deg, #1a1f3a 0%, #252a4d 50%, #1a1f3a 100%);
        background-size: 200% 100%;
        animation: skeleton-loading 1.5s infinite;
    }

    .skeleton-title {
        height: 20px;
        margin: 15px;
        background: linear-gradient(90deg, #1a1f3a 0%, #252a4d 50%, #1a1f3a 100%);
        background-size: 200% 100%;
        animation: skeleton-loading 1.5s infinite;
        border-radius: 4px;
    }

    @keyframes skeleton-loading {
        0% { background-position: 200% 0; }
        100% { background-position: -200% 0; }
    }

    /* ANIME CARD - Optimized for performance */
    .anime-card {
        background: #1a1f3a;
        border-radius: 12px;
        overflow: hidden;
        cursor: pointer;
        position: relative;
        border: none;
        padding: 0;
        text-align: left;
        color: inherit;
        font-family: inherit;
        width: 100%;
        will-change: transform;
        transform: translateZ(0); /* GPU acceleration */
        transition: transform 0.2s ease, box-shadow 0.2s ease;
    }

    .anime-card:hover {
        transform: translateY(-8px) translateZ(0);
        box-shadow: 0 15px 35px rgba(0, 0, 0, 0.4);
    }

    .anime-card:hover .card-overlay {
        opacity: 1;
    }
    
    .anime-card:focus {
        outline: 2px solid #f5576c;
        outline-offset: 2px;
    }

    .card-poster {
        width: 100%;
        aspect-ratio: 2/3;
        background: #0a0e27;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        position: relative;
    }

    .card-poster img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transition: transform 0.3s ease;
    }
    
    .anime-card:hover .card-poster img {
        transform: scale(1.05);
    }

    .source-badges {
        position: absolute;
        top: 8px;
        right: 8px;
        display: flex;
        gap: 4px;
    }

    .mini-badge {
        font-size: 1rem;
        padding: 4px;
        background: rgba(0, 0, 0, 0.7);
        border-radius: 4px;
        backdrop-filter: blur(4px);
    }

    .mini-badge.en {
        box-shadow: 0 0 0 1px rgba(74, 144, 217, 0.5);
    }

    .mini-badge.pt {
        box-shadow: 0 0 0 1px rgba(34, 197, 94, 0.5);
    }

    .no-image {
        font-size: 3rem;
    }

    .card-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.3s;
    }

    .play-icon {
        font-size: 3rem;
        color: white;
        text-shadow: 0 0 20px rgba(245, 87, 108, 0.8);
    }

    .card-info {
        padding: 15px;
    }

    .card-title {
        font-size: 0.95rem;
        font-weight: 600;
        line-height: 1.3;
        display: -webkit-box;
        line-clamp: 2;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
        margin-bottom: 5px;
    }

    .card-source {
        font-size: 0.75rem;
        color: #888;
        display: flex;
        align-items: center;
        gap: 5px;
    }

    .card-source::before {
        content: '📡';
        font-size: 0.7rem;
    }

    /* ANIME DETAIL VIEW - EXPANSIVE */
    .anime-detail {
        max-width: 1400px;
        width: 100%;
        margin: 0 auto;
        padding: clamp(20px, 4vw, 40px);
        min-height: 100%;
    }

    .btn-back {
        padding: 12px 24px;
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 12px;
        color: #f5576c;
        cursor: pointer;
        margin-bottom: 24px;
        font-weight: 600;
        transition: all 0.3s ease;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        backdrop-filter: blur(10px);
    }

    .btn-back:hover {
        background: rgba(245, 87, 108, 0.15);
        transform: translateX(-6px);
        border-color: rgba(245, 87, 108, 0.3);
        box-shadow: 0 4px 20px rgba(245, 87, 108, 0.2);
    }

    .anime-info {
        display: flex;
        gap: clamp(20px, 4vw, 40px);
        margin-bottom: 30px;
        background: rgba(26, 31, 58, 0.6);
        padding: clamp(20px, 3vw, 30px);
        border-radius: 16px;
        border: 1px solid rgba(255, 255, 255, 0.08);
        backdrop-filter: blur(10px);
        border-radius: 12px;
    }

    .anime-poster {
        width: 200px;
        height: 300px;
        object-fit: cover;
        border-radius: 8px;
        flex-shrink: 0;
    }

    .no-poster {
        background: #0a0e27;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 4rem;
    }

    .anime-meta {
        flex: 1;
    }

    .anime-meta h2 {
        margin: 0 0 20px 0;
        font-size: 2rem;
    }

    /* SOURCE SELECTOR */
    .source-selector {
        margin: 20px 0;
        padding: 20px;
        background: rgba(26, 31, 58, 0.8);
        border-radius: 12px;
        border: 2px solid #444;
    }

    .source-selector h3 {
        margin: 0 0 15px 0;
        font-size: 1.2rem;
        color: #f5576c;
    }

    .source-buttons {
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
    }

    .source-btn {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 20px 30px;
        border: 2px solid #444;
        border-radius: 12px;
        background: rgba(26, 31, 58, 0.9);
        color: white;
        cursor: pointer;
        transition: all 0.3s;
        min-width: 180px;
    }

    .source-btn:hover {
        transform: translateY(-4px);
        box-shadow: 0 8px 20px rgba(0, 0, 0, 0.3);
    }

    .source-btn.english {
        border-color: #4a90d9;
    }

    .source-btn.english:hover {
        background: rgba(74, 144, 217, 0.2);
        border-color: #6ab0ff;
    }

    .source-btn.portuguese {
        border-color: #22c55e;
    }

    .source-btn.portuguese:hover {
        background: rgba(34, 197, 94, 0.2);
        border-color: #4ade80;
    }

    .source-flag {
        font-size: 2.5rem;
    }

    .source-name {
        font-weight: bold;
        font-size: 1.1rem;
    }

    .source-lang {
        font-size: 0.85rem;
        color: #aaa;
    }

    .current-source {
        display: flex;
        align-items: center;
        gap: 15px;
        margin-bottom: 15px;
    }

    .source-badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 8px 16px;
        border-radius: 20px;
        font-weight: 600;
        font-size: 0.9rem;
    }

    .source-badge.english {
        background: rgba(74, 144, 217, 0.2);
        border: 1px solid #4a90d9;
        color: #6ab0ff;
    }

    .source-badge.portuguese {
        background: rgba(34, 197, 94, 0.2);
        border: 1px solid #22c55e;
        color: #4ade80;
    }

    .btn-change-source {
        padding: 6px 12px;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid #555;
        border-radius: 6px;
        color: #aaa;
        cursor: pointer;
        font-size: 0.85rem;
        transition: all 0.2s;
    }

    .btn-change-source:hover {
        background: rgba(245, 87, 108, 0.2);
        border-color: #f5576c;
        color: white;
    }

    .btn-reload {
        padding: 6px 12px;
        background: rgba(59, 130, 246, 0.1);
        border: 1px solid #3b82f6;
        border-radius: 6px;
        color: #60a5fa;
        cursor: pointer;
        font-size: 0.85rem;
        transition: all 0.2s;
    }

    .btn-reload:hover {
        background: rgba(59, 130, 246, 0.3);
        color: white;
    }

    .season-tabs {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
    }

    .season-tab {
        padding: 10px 20px;
        background: rgba(255, 255, 255, 0.05);
        border: 2px solid #444;
        border-radius: 8px;
        color: #aaa;
        cursor: pointer;
        font-weight: 600;
        transition: all 0.2s;
    }

    .season-tab:hover {
        border-color: #f5576c;
        color: white;
    }

    .season-tab.active {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        border-color: #f5576c;
        color: white;
    }

    .episodes-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 16px;
        padding-bottom: 40px;
        animation: fadeSlideIn 0.4s ease-out 0.1s both;
    }

    .episode-card {
        background: linear-gradient(145deg, rgba(26, 31, 58, 0.9), rgba(20, 24, 45, 0.95));
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        padding: 18px;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        text-align: left;
        color: inherit;
        font-family: inherit;
        width: 100%;
        backdrop-filter: blur(10px);
        position: relative;
        overflow: hidden;
    }

    .episode-card::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 3px;
        background: linear-gradient(90deg, #f5576c, #f093fb);
        transform: scaleX(0);
        transform-origin: left;
        transition: transform 0.3s ease;
    }

    .episode-card:hover::before {
        transform: scaleX(1);
    }

    .episode-card:hover {
        border-color: rgba(245, 87, 108, 0.4);
        transform: translateY(-4px);
        box-shadow: 0 10px 30px rgba(245, 87, 108, 0.15);
    }
    
    .episode-card:focus {
        outline: 2px solid #f5576c;
        outline-offset: 2px;
    }

    .episode-card.selected {
        border-color: #f5576c;
        background: linear-gradient(135deg, rgba(240, 147, 251, 0.15) 0%, rgba(245, 87, 108, 0.15) 100%);
        box-shadow: 0 0 20px rgba(245, 87, 108, 0.2);
    }

    .episode-card.selected::before {
        transform: scaleX(1);
    }

    .episode-number {
        font-size: 0.85rem;
        color: #f5576c;
        font-weight: 700;
        margin-bottom: 6px;
        letter-spacing: 0.5px;
    }

    .episode-title {
        font-size: 1rem;
        font-weight: 600;
        margin-bottom: 10px;
        color: #fff;
        line-height: 1.3;
    }

    .episode-actions {
        display: flex;
        gap: 10px;
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid rgba(255, 255, 255, 0.1);
        animation: fadeSlideIn 0.3s ease-out;
    }

    .btn-play-mpv,
    .btn-play-web {
        flex: 1;
        padding: 12px;
        border: none;
        border-radius: 8px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s ease;
        font-size: 0.9rem;
    }

    .btn-play-mpv {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        color: white;
        box-shadow: 0 4px 15px rgba(245, 87, 108, 0.3);
    }

    .btn-play-mpv.primary {
        flex: 1.5;
        background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
        box-shadow: 0 4px 15px rgba(34, 197, 94, 0.3);
    }

    .btn-play-web {
        background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        color: white;
        box-shadow: 0 4px 15px rgba(79, 172, 254, 0.3);
    }

    .btn-play-mpv:hover {
        transform: translateY(-2px);
        box-shadow: 0 6px 20px rgba(245, 87, 108, 0.4);
    }

    .btn-play-mpv.primary:hover {
        box-shadow: 0 6px 20px rgba(34, 197, 94, 0.4);
    }

    .btn-play-web:hover {
        transform: translateY(-2px);
        box-shadow: 0 6px 20px rgba(79, 172, 254, 0.4);
    }

    .episode-source {
        font-size: 0.7rem;
        color: #888;
        margin-top: 2px;
    }

    .loading {
        text-align: center;
        padding: 60px 20px;
        color: #bbb;
    }

    /* ===== MEDIA QUERIES para diferentes tamanhos ===== */
    
    /* Telas muito pequenas (< 480px) */
    @media (max-width: 480px) {
        .search-wrapper {
            flex-direction: column;
            border-radius: 16px;
            gap: 0;
        }
        
        .search-input {
            text-align: center;
        }
        
        .btn-search {
            width: 100%;
            border-radius: 12px;
            margin-top: 5px;
        }
        
        .card-info {
            padding: 10px;
        }
        
        .card-title {
            font-size: 0.85rem;
        }
    }
    
    /* Telas pequenas (480px - 768px) */
    @media (min-width: 481px) and (max-width: 768px) {
        .anime-grid,
        .anime-grid.large {
            grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
            gap: 15px;
        }
    }
    
    /* Tela cheia / Grandes (> 1200px) */
    @media (min-width: 1200px) {
        .search-wrapper {
            max-width: 900px;
        }
        
        .search-input {
            font-size: 1.15rem;
            padding: 18px 15px;
        }
        
        .btn-search {
            padding: 18px 50px;
            font-size: 1.1rem;
        }
        
        .anime-grid.large {
            grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
            gap: 40px;
        }
        
        :global(.results-section),
        :global(.popular-section) {
            padding: 50px 80px;
        }
        
        .section-title {
            font-size: 2.2rem;
        }
    }
    
    /* Modo Tela Cheia Extra Large (> 1600px) */
    @media (min-width: 1600px) {
        .search-wrapper {
            max-width: 1000px;
        }
        
        .anime-grid.large {
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 50px;
        }
        
        :global(.results-section),
        :global(.popular-section) {
            padding: clamp(15px, 4vh, 40px) clamp(15px, 4vw, 50px);
            flex: 1;
            overflow-y: auto;
        }
        .card-title {
            font-size: 1.1rem;
        }
        
        .play-icon {
            font-size: 4rem;
        }
    }

    /* ============================================
       FEATURED HERO (AniList Banner) - Optimized
       ============================================ */
    .featured-hero {
        position: relative;
        min-height: 55vh;
        max-height: 65vh;
        background-image: var(--banner-url);
        background-size: cover;
        background-position: center top;
        display: flex;
        align-items: flex-end;
        animation: fadeInHero 0.6s ease-out;
    }
    
    @keyframes fadeInHero {
        from { opacity: 0; transform: scale(1.02); }
        to { opacity: 1; transform: scale(1); }
    }

    .featured-overlay {
        position: absolute;
        inset: 0;
        background: linear-gradient(
            to top,
            rgba(10, 14, 39, 1) 0%,
            rgba(10, 14, 39, 0.8) 30%,
            rgba(10, 14, 39, 0.4) 60%,
            rgba(10, 14, 39, 0.2) 100%
        );
    }

    .featured-content {
        position: relative;
        z-index: 2;
        width: 100%;
        max-width: 1400px;
        margin: 0 auto;
        padding: 40px 60px;
        display: flex;
        gap: 40px;
        align-items: flex-end;
    }

    .featured-info {
        flex: 1;
    }

    .featured-badges {
        display: flex;
        gap: 10px;
        margin-bottom: 15px;
    }

    .badge {
        padding: 6px 12px;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 600;
        text-transform: uppercase;
    }

    .badge.airing {
        background: rgba(255, 0, 0, 0.3);
        color: #ff4444;
        border: 1px solid #ff4444;
    }

    .badge.score {
        background: rgba(255, 193, 7, 0.2);
        color: #ffc107;
    }

    .badge.episodes {
        background: rgba(255, 255, 255, 0.1);
        color: #ccc;
    }

    .featured-title {
        font-size: clamp(2rem, 4vw, 3.5rem);
        font-weight: 700;
        margin: 0 0 10px 0;
        text-shadow: 2px 2px 20px rgba(0, 0, 0, 0.8);
    }

    .featured-meta {
        color: #aaa;
        font-size: 0.95rem;
        margin-bottom: 15px;
    }

    .featured-desc {
        color: #ccc;
        font-size: 1rem;
        line-height: 1.6;
        max-width: 600px;
        margin-bottom: 25px;
    }

    .featured-actions {
        display: flex;
        gap: 15px;
    }

    .btn-featured-play {
        padding: 14px 32px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border: none;
        border-radius: 8px;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s;
    }

    .btn-featured-play:hover {
        transform: scale(1.05);
        box-shadow: 0 5px 30px rgba(245, 87, 108, 0.4);
    }

    .btn-featured-trailer {
        padding: 14px 28px;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.3);
        border-radius: 8px;
        color: #fff;
        font-size: 1rem;
        text-decoration: none;
        transition: all 0.3s;
    }

    .btn-featured-trailer:hover {
        background: rgba(255, 255, 255, 0.2);
    }

    .featured-poster {
        width: 200px;
        flex-shrink: 0;
    }

    .featured-poster img {
        width: 100%;
        border-radius: 12px;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
    }

    /* Featured Navigation Dots */
    .featured-nav {
        position: absolute;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%);
        display: flex;
        gap: 8px;
        z-index: 10;
    }
    
    .nav-dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.3);
        border: none;
        cursor: pointer;
        transition: all 0.3s;
        padding: 0;
    }
    
    .nav-dot:hover {
        background: rgba(255, 255, 255, 0.6);
    }
    
    .nav-dot.active {
        background: #f5576c;
        width: 24px;
        border-radius: 5px;
    }

    /* Featured Skeleton Loading */
    .featured-skeleton {
        min-height: 50vh;
        background: linear-gradient(135deg, #1a1f3a 0%, #0a0e27 100%);
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

    /* Home View Animations */
    .home-view {
        opacity: 0;
        transform: translateY(10px);
        transition: opacity 0.4s ease, transform 0.4s ease;
    }
    
    .home-view.ready {
        opacity: 1;
        transform: translateY(0);
    }

    @media (max-width: 768px) {
        .featured-hero {
            min-height: 50vh;
        }
        
        .featured-content {
            padding: 30px;
            flex-direction: column-reverse;
            align-items: flex-start;
        }
        
        .featured-poster {
            width: 120px;
        }
        
        .featured-desc {
            display: none;
        }
        
        .featured-nav {
            bottom: 15px;
        }
    }

    /* ============================================
       UNIFIED CONTENT AREA (Busca + Resultados)
       ============================================ */
    .content-area {
        padding: 30px clamp(20px, 5vw, 60px);
        padding-bottom: 60px;
    }

    .search-bar-sticky {
        /* Não é mais sticky - desce junto com o conteúdo */
        padding: 0 0 30px 0;
        margin-bottom: 0;
    }

    .search-bar-sticky .search-wrapper {
        display: flex;
        align-items: center;
        background: rgba(26, 31, 58, 0.8);
        border: 2px solid #444;
        border-radius: 50px;
        padding: 5px;
        max-width: 800px;
        margin: 0 auto;
        transition: all 0.3s;
    }

    .search-bar-sticky .search-wrapper:focus-within {
        border-color: #f5576c;
        box-shadow: 0 0 20px rgba(245, 87, 108, 0.3);
    }

    .results-header {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 15px;
        margin-top: 15px;
        padding: 10px 0;
    }

    .results-count {
        color: #aaa;
        font-size: 0.9rem;
    }

    :global(.btn-clear-inline) {
        background: rgba(255, 100, 100, 0.2);
        border: 1px solid rgba(255, 100, 100, 0.4);
        color: #ff6b6b;
        padding: 6px 15px;
        border-radius: 20px;
        cursor: pointer;
        font-size: 0.85rem;
        transition: all 0.2s;
    }

    :global(.btn-clear-inline:hover) {
        background: rgba(255, 100, 100, 0.3);
    }

    /* SEARCH SECTION - Antigo (manter compatibilidade) */
    :global(.search-section) {
        background: rgba(10, 14, 39, 0.95);
        padding: 30px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    :global(.search-section .search-box) {
        max-width: 800px;
        margin: 0 auto;
    }

    /* ============================================
       CONTENT SECTIONS
       ============================================ */
    .content-section {
        padding: 40px 0;
    }

    .title-badge.anilist {
        background: linear-gradient(135deg, #02a9ff 0%, #0084ff 100%);
    }

    .title-badge.sources {
        background: linear-gradient(135deg, #4caf50 0%, #8bc34a 100%);
    }

    /* ============================================
       ANIME ROW (Horizontal Scroll with Arrows)
       ============================================ */
    .anime-row {
        display: flex;
        gap: 20px;
        overflow-x: auto;
        padding: 10px 0 20px;
        scroll-snap-type: x mandatory;
        scroll-behavior: smooth;
        /* Fade edges */
        mask-image: linear-gradient(90deg, transparent 0%, black 3%, black 97%, transparent 100%);
        -webkit-mask-image: linear-gradient(90deg, transparent 0%, black 3%, black 97%, transparent 100%);
    }

    .anime-row::-webkit-scrollbar {
        height: 8px;
    }

    .anime-row::-webkit-scrollbar-track {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 4px;
    }

    .anime-row::-webkit-scrollbar-thumb {
        background: linear-gradient(90deg, rgba(245, 87, 108, 0.5), rgba(240, 147, 251, 0.5));
        border-radius: 4px;
    }

    .anime-row::-webkit-scrollbar-thumb:hover {
        background: linear-gradient(90deg, rgba(245, 87, 108, 0.8), rgba(240, 147, 251, 0.8));
    }

    /* ============================================
       ANIME CARD HD (AniList Style)
       ============================================ */
    .anime-card-hd {
        flex-shrink: 0;
        width: 200px;
        cursor: pointer;
        scroll-snap-align: start;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .anime-card-hd:hover {
        transform: translateY(-10px) scale(1.02);
    }

    .card-poster-hd {
        position: relative;
        aspect-ratio: 3/4;
        border-radius: 12px;
        overflow: hidden;
        background: var(--card-color);
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
        transition: box-shadow 0.3s ease;
    }

    .anime-card-hd:hover .card-poster-hd {
        box-shadow: 0 10px 40px rgba(245, 87, 108, 0.3);
    }

    .card-poster-hd img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transition: transform 0.4s ease;
    }

    .anime-card-hd:hover .card-poster-hd img {
        transform: scale(1.08);
    }

    .card-badges-hd {
        position: absolute;
        top: 10px;
        left: 10px;
        display: flex;
        gap: 6px;
    }

    .badge-mini {
        padding: 4px 8px;
        border-radius: 4px;
        font-size: 0.7rem;
        font-weight: 600;
        background: rgba(0, 0, 0, 0.7);
        backdrop-filter: blur(4px);
    }

    .badge-mini.airing {
        animation: pulse 2s infinite;
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; }
    }

    .badge-mini.score {
        color: #ffc107;
    }

    .card-overlay-hd {
        position: absolute;
        inset: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.3s;
    }

    .anime-card-hd:hover .card-overlay-hd {
        opacity: 1;
    }

    .card-info-hd {
        padding: 12px 4px;
    }

    .card-title-hd {
        font-size: 0.95rem;
        font-weight: 600;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        margin-bottom: 4px;
    }

    .card-meta-hd {
        font-size: 0.8rem;
        color: #888;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    /* Responsive for HD Cards */
    @media (max-width: 768px) {
        .anime-card-hd {
            width: 150px;
        }
    }

    @media (min-width: 1600px) {
        .anime-card-hd {
            width: 240px;
        }
        
        .card-title-hd {
            font-size: 1.05rem;
        }
    }

    /* ============================================
       RESPONSIVE LOGIN SCREEN
       ============================================ */
    @media (max-width: 600px) {
        .login-content {
            padding: 20px;
        }
        
        .logo-emoji {
            font-size: 3rem;
        }
        
        .logo-text {
            font-size: 2.2rem;
        }
        
        .login-tagline {
            font-size: 0.95rem;
        }
        
        .login-card-modern {
            border-radius: 16px;
        }
        
        .card-header,
        .card-body,
        .card-footer {
            padding: 20px;
        }
        
        .avatar-grid {
            grid-template-columns: repeat(3, 1fr);
            gap: 10px;
        }
        
        .avatar-emoji {
            font-size: 1.6rem;
        }
        
        .login-features {
            flex-direction: column;
            gap: 15px;
        }
    }
    
    @media (min-width: 1400px) {
        .login-content {
            max-width: 550px;
        }
        
        .logo-emoji {
            font-size: 5rem;
        }
        
        .logo-text {
            font-size: 4rem;
        }
        
        .login-card-modern {
            border-radius: 24px;
        }
        
        .card-header {
            padding: 30px 40px;
        }
        
        .card-body {
            padding: 40px;
        }
        
        .avatar-emoji {
            font-size: 2.5rem;
        }
    }
    
    /* ============================================
       RESPONSIVE HERO SECTION MODERN
       ============================================ */
    @media (max-width: 600px) {
        .hero-section-modern {
            min-height: 40vh;
            padding: 40px 15px;
        }
        
        .hero-stats {
            gap: 15px;
        }
        
        .stat-number {
            font-size: 1.3rem;
        }
        
        .stat-label {
            font-size: 0.75rem;
        }
        
        .stat-divider {
            height: 30px;
        }
    }
    
    @media (min-width: 1400px) {
        .hero-section-modern {
            min-height: 55vh;
            padding: 80px 40px;
        }
        
        .hero-stats {
            gap: 40px;
        }
        
        .stat-number {
            font-size: 2.5rem;
        }
        
        .stat-label {
            font-size: 1rem;
        }
    }
    
    /* ============================================
       FULLSCREEN APP LAYOUT
       ============================================ */
    .app {
        display: flex;
        flex-direction: column;
        height: 100%;
        width: 100%;
    }
    
    .main-content {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
    }
    
    .home-view {
        min-height: 100%;
    }

    /* ============================================
       NAVIGATION TABS - Modern Style
       ============================================ */
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
        background: rgba(255, 193, 7, 0.2);
        color: #ffc107;
    }
    
    .tab-badge.notify {
        background: #f5576c;
        color: #fff;
        animation: pulse 2s infinite;
    }
    
    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }

    /* ============================================
       TAB CONTENT STYLES
       ============================================ */
    .tab-content {
        padding: 30px 20px;
        max-width: 1200px;
        margin: 0 auto;
    }
    
    /* Manga Coming Soon */
    .manga-coming-soon {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 400px;
    }
    
    .coming-soon-card {
        text-align: center;
        padding: 60px 40px;
        background: rgba(20, 25, 50, 0.8);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        max-width: 500px;
    }
    
    .coming-soon-icon {
        font-size: 4rem;
        margin-bottom: 20px;
    }
    
    .coming-soon-card h2 {
        margin: 0 0 10px;
        color: #fff;
        font-size: 1.8rem;
    }
    
    .coming-soon-card p {
        color: #888;
        margin-bottom: 30px;
    }
    
    .coming-soon-features {
        display: flex;
        justify-content: center;
        gap: 30px;
        margin-bottom: 30px;
    }
    
    .coming-soon-features .feature {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        color: #aaa;
        font-size: 0.9rem;
    }
    
    .coming-soon-features .feature-icon {
        font-size: 1.5rem;
    }
    
    .btn-notify {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border: none;
        padding: 12px 30px;
        border-radius: 25px;
        color: #fff;
        font-size: 1rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.25s;
    }
    
    .btn-notify:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 20px rgba(245, 87, 108, 0.4);
    }

    /* ============================================
       FRIENDS TAB (Discord Recommendations)
       ============================================ */
    .friends-tab {
        max-width: 900px;
    }
    
    /* Discord Connect Container */
    .discord-connect-container {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 60vh;
        padding: 20px;
    }
    
    .discord-connect-card {
        position: relative;
        background: rgba(20, 25, 50, 0.9);
        border: 1px solid rgba(88, 101, 242, 0.3);
        border-radius: 24px;
        padding: 50px 40px;
        max-width: 450px;
        text-align: center;
        overflow: hidden;
        backdrop-filter: blur(10px);
    }
    
    .discord-glow {
        position: absolute;
        top: -50%;
        left: -50%;
        width: 200%;
        height: 200%;
        background: radial-gradient(circle at center, rgba(88, 101, 242, 0.15) 0%, transparent 50%);
        animation: rotate-glow 10s linear infinite;
        pointer-events: none;
    }
    
    @keyframes rotate-glow {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }
    
    .discord-content {
        position: relative;
        z-index: 1;
    }
    
    .discord-logo-animated {
        position: relative;
        display: inline-flex;
        justify-content: center;
        align-items: center;
        width: 120px;
        height: 120px;
        margin-bottom: 25px;
    }
    
    .logo-ring {
        position: absolute;
        width: 100%;
        height: 100%;
        border: 2px solid rgba(88, 101, 242, 0.3);
        border-radius: 50%;
        animation: pulse-ring 2s ease-out infinite;
    }
    
    .logo-ring.ring-2 {
        animation-delay: 1s;
    }
    
    @keyframes pulse-ring {
        0% { transform: scale(0.8); opacity: 1; }
        100% { transform: scale(1.5); opacity: 0; }
    }
    
    .discord-svg {
        background: linear-gradient(135deg, #5865F2 0%, #7289da 100%);
        border-radius: 50%;
        padding: 15px;
        box-shadow: 0 8px 30px rgba(88, 101, 242, 0.4);
    }
    
    .discord-title {
        color: #fff;
        font-size: 1.8rem;
        font-weight: 700;
        margin: 0 0 10px;
    }
    
    .discord-subtitle {
        color: #8b8fa3;
        font-size: 1rem;
        margin: 0 0 30px;
    }
    
    .discord-features {
        display: flex;
        flex-direction: column;
        gap: 16px;
        margin-bottom: 35px;
    }
    
    .feature-item {
        display: flex;
        align-items: center;
        gap: 15px;
        padding: 12px 20px;
        background: rgba(255, 255, 255, 0.03);
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 12px;
        text-align: left;
        transition: all 0.25s;
    }
    
    .feature-item:hover {
        background: rgba(88, 101, 242, 0.1);
        border-color: rgba(88, 101, 242, 0.2);
        transform: translateX(5px);
    }
    
    .feature-icon {
        font-size: 1.5rem;
    }
    
    .feature-text {
        display: flex;
        flex-direction: column;
    }
    
    .feature-text strong {
        color: #fff;
        font-size: 0.95rem;
    }
    
    .feature-text span {
        color: #666;
        font-size: 0.85rem;
    }

    .discord-privacy {
        color: #555;
        font-size: 0.8rem;
        margin: 20px 0 0;
    }
    
    .spinner-small {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    /* Discord Linking Modal */
    .discord-link-modal {
        position: relative;
        background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
        border: 1px solid rgba(88, 101, 242, 0.3);
        border-radius: 20px;
        width: 90%;
        max-width: 500px;
        padding: 30px;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
    }
    
    .link-modal-header {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
        gap: 10px;
        margin-bottom: 25px;
    }
    
    .link-modal-header .link-icon {
        font-size: 40px;
    }
    
    .link-modal-header h2 {
        color: #fff;
        font-size: 1.4rem;
        margin: 0;
    }
    
    .link-modal-header p {
        color: #8b8b8b;
        font-size: 0.9rem;
        margin: 0;
    }
    
    .link-code-input {
        margin-bottom: 20px;
    }
    
    .link-code-input label {
        display: block;
        color: #8b8b8b;
        font-size: 0.85rem;
        margin-bottom: 8px;
    }
    
    .link-code-input .code-input {
        width: 100%;
        padding: 14px 16px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(88, 101, 242, 0.3);
        border-radius: 10px;
        color: #fff;
        font-size: 1.1rem;
        font-family: monospace;
        text-align: center;
        letter-spacing: 2px;
        text-transform: uppercase;
        box-sizing: border-box;
    }
    
    .link-code-input .code-input:focus {
        outline: none;
        border-color: #5865F2;
        box-shadow: 0 0 15px rgba(88, 101, 242, 0.3);
    }
    
    .link-code-input .code-input::placeholder {
        color: #555;
        letter-spacing: 1px;
    }
    
    .link-code-input .link-error {
        display: block;
        color: #ed4245;
        font-size: 0.85rem;
        margin-top: 8px;
    }
    
    .link-help {
        background: rgba(88, 101, 242, 0.1);
        border: 1px solid rgba(88, 101, 242, 0.2);
        border-radius: 12px;
        padding: 15px;
        margin-bottom: 20px;
    }
    
    .link-help p {
        color: #fff;
        font-size: 0.9rem;
        margin: 0 0 10px 0;
    }
    
    .link-help ol {
        margin: 0;
        padding-left: 20px;
        color: #8b8b8b;
        font-size: 0.85rem;
    }
    
    .link-help li {
        margin-bottom: 6px;
    }
    
    .link-help a {
        color: #5865F2;
        text-decoration: none;
    }
    
    .link-help a:hover {
        text-decoration: underline;
    }
    
    .link-help code {
        background: rgba(0, 0, 0, 0.3);
        padding: 2px 6px;
        border-radius: 4px;
        font-family: monospace;
        color: #7289da;
    }
    
    .link-modal-actions {
        display: flex;
        gap: 12px;
    }
    
    .link-modal-actions .btn-cancel {
        flex: 1;
        padding: 14px 20px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 12px;
        color: #888;
        font-size: 0.95rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .link-modal-actions .btn-cancel:hover {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
    }
    
    .btn-link-confirm {
        flex: 2;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 14px 24px;
        background: linear-gradient(135deg, #5865F2 0%, #7289da 100%);
        border: none;
        border-radius: 12px;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s;
    }
    
    .btn-link-confirm:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 6px 25px rgba(88, 101, 242, 0.4);
    }
    
    .btn-link-confirm:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    /* Friend Activity Card */
    .friend-activity-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 16px;
        background: rgba(20, 25, 50, 0.6);
        border: 1px solid rgba(88, 101, 242, 0.15);
        border-radius: 12px;
        transition: all 0.3s ease;
    }
    
    .friend-activity-card:hover {
        background: rgba(88, 101, 242, 0.1);
        border-color: rgba(88, 101, 242, 0.3);
        transform: translateX(4px);
    }
    
    .friend-avatar-section {
        position: relative;
        flex-shrink: 0;
    }
    
    .friend-avatar {
        width: 45px;
        height: 45px;
        border-radius: 50%;
        border: 2px solid rgba(88, 101, 242, 0.3);
    }
    
    .online-dot {
        position: absolute;
        bottom: 2px;
        right: 2px;
        width: 12px;
        height: 12px;
        background: #3ba55c;
        border: 2px solid #1a1a2e;
        border-radius: 50%;
    }
    
    .friend-info {
        flex: 1;
        min-width: 0;
    }
    
    .friend-name {
        color: #fff;
        font-weight: 600;
        font-size: 0.95rem;
        margin-bottom: 4px;
    }
    
    .friend-watching {
        color: #8b8b8b;
        font-size: 0.8rem;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    
    .friend-watching strong {
        color: #5865F2;
    }
    
    .friend-anime-thumb {
        width: 50px;
        height: 70px;
        object-fit: cover;
        border-radius: 6px;
        border: 1px solid rgba(88, 101, 242, 0.2);
        flex-shrink: 0;
    }
    
    /* Link Steps */
    .link-steps {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 20px;
    }
    
    .link-step {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 16px;
        background: rgba(88, 101, 242, 0.08);
        border-radius: 10px;
    }
    
    .link-step .step-num {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 28px;
        height: 28px;
        background: linear-gradient(135deg, #5865F2 0%, #7289da 100%);
        border-radius: 50%;
        color: #fff;
        font-weight: bold;
        font-size: 0.85rem;
        flex-shrink: 0;
    }
    
    .link-step span:last-child {
        color: #ccc;
        font-size: 0.9rem;
    }
    
    .link-step code {
        background: rgba(0, 0, 0, 0.3);
        padding: 2px 6px;
        border-radius: 4px;
        font-family: monospace;
        color: #7289da;
    }
    
    /* Link Actions */
    .link-actions {
        display: flex;
        gap: 12px;
        margin-bottom: 15px;
    }
    
    .btn-discord-join {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        flex: 1;
        padding: 14px 20px;
        background: rgba(88, 101, 242, 0.15);
        border: 1px solid rgba(88, 101, 242, 0.3);
        border-radius: 10px;
        color: #5865F2;
        font-size: 0.9rem;
        font-weight: 500;
        text-decoration: none;
        transition: all 0.3s ease;
    }
    
    .btn-discord-join:hover {
        background: rgba(88, 101, 242, 0.25);
        border-color: rgba(88, 101, 242, 0.5);
    }

    /* Discord Link Button */
    .btn-discord-link {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        padding: 16px 32px;
        background: linear-gradient(135deg, #5865F2 0%, #7289da 100%);
        border: none;
        border-radius: 12px;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
        width: 100%;
        max-width: 300px;
    }
    
    .btn-discord-link:hover {
        transform: translateY(-3px);
        box-shadow: 0 8px 30px rgba(88, 101, 242, 0.4);
    }

    /* Share Settings */
    .share-settings {
        background: rgba(20, 25, 50, 0.6);
        border: 1px solid rgba(88, 101, 242, 0.15);
        border-radius: 16px;
        padding: 20px;
        margin-bottom: 30px;
    }
    
    .share-settings h3 {
        color: #fff;
        font-size: 1rem;
        margin: 0 0 15px 0;
    }
    
    .setting-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 12px 0;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }
    
    .setting-row:last-child {
        border-bottom: none;
    }
    
    .setting-info {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    
    .setting-title {
        color: #fff;
        font-size: 0.9rem;
    }
    
    .setting-desc {
        color: #8b8b8b;
        font-size: 0.8rem;
    }
    
    .toggle-btn {
        position: relative;
        width: 50px;
        height: 26px;
        background: rgba(0, 0, 0, 0.3);
        border: none;
        border-radius: 13px;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .toggle-btn.active {
        background: #5865F2;
    }
    
    .toggle-btn .toggle-slider {
        position: absolute;
        top: 3px;
        left: 3px;
        width: 20px;
        height: 20px;
        background: #fff;
        border-radius: 50%;
        transition: all 0.3s ease;
    }
    
    .toggle-btn.active .toggle-slider {
        left: 27px;
    }
    
    /* Friends Activity */
    .section-header {
        margin-bottom: 20px;
    }
    
    .section-header h2 {
        color: #fff;
        font-size: 1.2rem;
        margin: 0 0 5px 0;
    }
    
    .section-header p {
        color: #8b8b8b;
        font-size: 0.85rem;
        margin: 0 0 15px 0;
    }
    
    .btn-refresh {
        padding: 8px 16px;
        background: rgba(88, 101, 242, 0.15);
        border: 1px solid rgba(88, 101, 242, 0.3);
        border-radius: 8px;
        color: #5865F2;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .btn-refresh:hover {
        background: rgba(88, 101, 242, 0.25);
    }
    
    .friends-loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 15px;
        padding: 40px;
    }
    
    .friends-loading span {
        color: #8b8b8b;
        font-size: 0.9rem;
    }
    
    .no-friends-activity {
        text-align: center;
        padding: 40px 20px;
        background: rgba(20, 25, 50, 0.4);
        border-radius: 16px;
    }
    
    .no-friends-activity .empty-icon {
        font-size: 48px;
        margin-bottom: 15px;
    }
    
    .no-friends-activity p {
        color: #fff;
        font-size: 1rem;
        margin: 0 0 8px 0;
    }
    
    .no-friends-activity .empty-hint {
        color: #8b8b8b;
        font-size: 0.85rem;
        margin-bottom: 20px;
    }
    
    .invite-friends-link {
        display: inline-block;
        color: #5865F2;
        font-size: 0.9rem;
        text-decoration: none;
        transition: all 0.3s ease;
    }
    
    .invite-friends-link:hover {
        text-decoration: underline;
    }
    
    .friends-activity-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    
    .friend-avatar-placeholder {
        width: 45px;
        height: 45px;
        border-radius: 50%;
        background: rgba(88, 101, 242, 0.2);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 20px;
    }
    
    .friend-episode {
        color: #666;
        font-size: 0.75rem;
    }
    
    /* Discord Disconnect Button */
    .btn-disconnect {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 10px 16px;
        background: rgba(237, 66, 69, 0.15);
        border: 1px solid rgba(237, 66, 69, 0.3);
        border-radius: 10px;
        color: #ed4245;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .btn-disconnect:hover {
        background: rgba(237, 66, 69, 0.25);
        border-color: rgba(237, 66, 69, 0.5);
    }

    /* Discord Profile Section */
    .discord-profile-section {
        margin-bottom: 40px;
    }
    
    .discord-profile-card {
        position: relative;
        background: rgba(20, 25, 50, 0.9);
        border: 1px solid rgba(88, 101, 242, 0.3);
        border-radius: 20px;
        overflow: hidden;
        margin-bottom: 20px;
    }
    
    .profile-background {
        height: 80px;
        background: linear-gradient(135deg, #5865F2 0%, #7289da 50%, #5865F2 100%);
        background-size: 200% 200%;
        animation: gradient-shift 5s ease infinite;
    }
    
    @keyframes gradient-shift {
        0%, 100% { background-position: 0% 50%; }
        50% { background-position: 100% 50%; }
    }
    
    .profile-content {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 0 25px 20px;
        margin-top: -40px;
    }
    
    .profile-avatar-wrapper {
        position: relative;
        flex-shrink: 0;
    }
    
    .profile-avatar {
        width: 80px;
        height: 80px;
        border-radius: 50%;
        border: 4px solid #1a1a2e;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
    }
    
    .online-badge {
        position: absolute;
        bottom: 5px;
        right: 5px;
        width: 18px;
        height: 18px;
        background: #3ba55c;
        border: 3px solid #1a1a2e;
        border-radius: 50%;
        animation: pulse-online 2s infinite;
    }
    
    @keyframes pulse-online {
        0%, 100% { box-shadow: 0 0 0 0 rgba(59, 165, 92, 0.4); }
        50% { box-shadow: 0 0 0 6px rgba(59, 165, 92, 0); }
    }
    
    .profile-details {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    
    .profile-name {
        color: #fff;
        font-size: 1.4rem;
        font-weight: 700;
    }
    
    .profile-tag {
        color: #3ba55c;
        font-size: 0.9rem;
        font-weight: 500;
    }
    
    .btn-disconnect {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 10px 18px;
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.2);
        border-radius: 10px;
        color: #ef4444;
        font-size: 0.85rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.25s;
    }
    
    .btn-disconnect:hover {
        background: rgba(239, 68, 68, 0.2);
        border-color: rgba(239, 68, 68, 0.4);
    }
    
    /* Section Header */
    .section-header {
        margin-bottom: 25px;
    }
    
    .section-header h2 {
        display: flex;
        align-items: center;
        gap: 10px;
        color: #fff;
        font-size: 1.4rem;
        font-weight: 600;
        margin: 0 0 8px;
    }
    
    .section-icon {
        font-size: 1.2rem;
    }
    
    .section-header p {
        color: #666;
        margin: 0;
        font-size: 0.95rem;
    }

    /* Loading */
    .friends-loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 20px;
        padding: 60px 20px;
        color: #888;
    }
    
    .spinner {
        width: 50px;
        height: 50px;
        border: 3px solid rgba(88, 101, 242, 0.2);
        border-top-color: #5865F2;
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    /* ============================================
       COMMUNITY TAB
       ============================================ */
    .community-tab {
        max-width: 900px;
    }
    
    .community-header {
        text-align: center;
        margin-bottom: 40px;
    }
    
    .community-header h2 {
        color: #fff;
        font-size: 1.8rem;
        margin: 0 0 8px;
    }
    
    .community-header p {
        color: #888;
        margin: 0;
    }
    
    .community-links {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: 20px;
        margin-bottom: 40px;
    }
    
    .community-card {
        display: flex;
        align-items: center;
        gap: 20px;
        padding: 25px;
        background: rgba(20, 25, 50, 0.8);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 15px;
        text-decoration: none;
        transition: all 0.25s;
    }
    
    .community-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
    }
    
    .community-card.discord {
        border-color: rgba(88, 101, 242, 0.3);
    }
    
    .community-card.discord:hover {
        border-color: #5865F2;
        box-shadow: 0 10px 30px rgba(88, 101, 242, 0.2);
    }
    
    .community-card.github {
        border-color: rgba(255, 255, 255, 0.2);
    }
    
    .community-card.github:hover {
        border-color: #fff;
    }
    
    .community-card.twitter {
        border-color: rgba(29, 155, 240, 0.3);
    }
    
    .community-card.twitter:hover {
        border-color: #1d9bf0;
        box-shadow: 0 10px 30px rgba(29, 155, 240, 0.2);
    }
    
    .community-icon {
        flex-shrink: 0;
    }
    
    .community-info h3 {
        color: #fff;
        margin: 0 0 5px;
        font-size: 1.2rem;
    }
    
    .community-info p {
        color: #888;
        margin: 0 0 8px;
        font-size: 0.9rem;
    }
    
    .member-count, .star-count, .follower-count {
        font-size: 0.85rem;
        color: #5865F2;
    }
    
    .star-count {
        color: #ffc107;
    }
    
    .follower-count {
        color: #1d9bf0;
    }
    
    .community-stats {
        display: flex;
        justify-content: center;
        gap: 40px;
    }
    
    .community-stats .stat-card {
        text-align: center;
        padding: 25px 40px;
        background: rgba(20, 25, 50, 0.8);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 15px;
    }
    
    .community-stats .stat-value {
        display: block;
        font-size: 2rem;
        font-weight: 700;
        color: #fff;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .community-stats .stat-label {
        color: #888;
        font-size: 0.9rem;
    }

    /* Responsive for tabs */
    @media (max-width: 768px) {
        .nav-tabs-container {
            flex-direction: column;
            gap: 15px;
        }
        
        .nav-tabs {
            flex-wrap: wrap;
            justify-content: center;
        }
        
        .nav-tab {
            padding: 8px 15px;
            font-size: 0.85rem;
        }
        
        .tab-text {
            display: none;
        }
        
        .nav-tab.active .tab-text {
            display: inline;
        }
        
        .community-links {
            grid-template-columns: 1fr;
        }
        
        .community-stats {
            flex-direction: column;
            gap: 15px;
        }
    }

    /* ============================================
       SHARE MODAL
       ============================================ */
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        backdrop-filter: blur(5px);
    }
    
    .share-modal {
        background: linear-gradient(135deg, #1a1f3a 0%, #0d1025 100%);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        padding: 30px;
        max-width: 500px;
        width: 90%;
        position: relative;
        animation: modalSlideIn 0.3s ease;
    }
    
    @keyframes modalSlideIn {
        from {
            opacity: 0;
            transform: translateY(-20px) scale(0.95);
        }
        to {
            opacity: 1;
            transform: translateY(0) scale(1);
        }
    }
    
    .modal-close {
        position: absolute;
        top: 15px;
        right: 15px;
        background: rgba(255, 255, 255, 0.1);
        border: none;
        color: #fff;
        width: 32px;
        height: 32px;
        border-radius: 50%;
        cursor: pointer;
        font-size: 1rem;
        transition: all 0.2s;
    }
    
    .modal-close:hover {
        background: rgba(255, 100, 100, 0.3);
    }
    
    .share-modal-header {
        text-align: center;
        margin-bottom: 25px;
    }
    
    .share-modal-header h2 {
        color: #fff;
        margin: 0 0 8px;
        font-size: 1.5rem;
    }
    
    .share-modal-header p {
        color: #888;
        margin: 0;
    }
    
    .share-anime-preview {
        display: flex;
        gap: 15px;
        padding: 15px;
        background: rgba(255, 255, 255, 0.05);
        border-radius: 12px;
        margin-bottom: 20px;
    }
    
    .share-anime-preview img {
        width: 80px;
        height: 115px;
        object-fit: cover;
        border-radius: 8px;
    }
    
    .share-anime-info {
        display: flex;
        flex-direction: column;
        justify-content: center;
    }
    
    .share-anime-info h3 {
        color: #fff;
        margin: 0 0 8px;
        font-size: 1.1rem;
    }
    
    .share-score {
        color: #ffc107;
        font-size: 0.9rem;
    }
    
    .share-message-input {
        margin-bottom: 20px;
    }
    
    .share-message-input label {
        display: block;
        color: #aaa;
        margin-bottom: 8px;
        font-size: 0.9rem;
    }
    
    .share-message-input textarea {
        width: 100%;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 10px;
        padding: 12px;
        color: #fff;
        font-size: 0.95rem;
        resize: none;
        font-family: inherit;
    }
    
    .share-message-input textarea:focus {
        outline: none;
        border-color: #5865F2;
    }
    
    .share-message-input textarea::placeholder {
        color: #666;
    }
    
    .share-modal-actions {
        display: flex;
        gap: 12px;
    }
    
    .btn-cancel {
        flex: 1;
        padding: 12px 20px;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        border-radius: 25px;
        color: #fff;
        font-size: 0.95rem;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-cancel:hover {
        background: rgba(255, 255, 255, 0.15);
    }
    
    .btn-send-share {
        flex: 2;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 12px 20px;
        background: #5865F2;
        border: none;
        border-radius: 25px;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-send-share:hover:not(:disabled) {
        background: #4752c4;
        transform: translateY(-2px);
        box-shadow: 0 5px 20px rgba(88, 101, 242, 0.4);
    }
    
    .btn-send-share:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
