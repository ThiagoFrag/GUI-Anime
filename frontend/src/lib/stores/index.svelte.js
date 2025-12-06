/**
 * Stores centralizados para gerenciamento de estado global
 * Usando Svelte 5 runes ($state)
 */

// === USER STORE ===
let _user = $state(null);
let _isLoggedIn = $state(false);

export const userStore = {
    get user() { return _user; },
    get isLoggedIn() { return _isLoggedIn; },
    
    setUser(user) {
        _user = user;
        _isLoggedIn = !!user?.username;
    },
    
    clear() {
        _user = null;
        _isLoggedIn = false;
    }
};

// === SETTINGS STORE ===
let _settings = $state({
    start_fullscreen: false,
    content_language: 'all',
    default_quality: 'auto',
    use_anime4k: true
});

export const settingsStore = {
    get settings() { return _settings; },
    
    setSettings(settings) {
        _settings = { ..._settings, ...settings };
    },
    
    reset() {
        _settings = {
            start_fullscreen: false,
            content_language: 'all',
            default_quality: 'auto',
            use_anime4k: true
        };
    }
};

// === DISCORD STORE ===
let _discordLinked = $state(false);
let _discordLinkInfo = $state(null);
let _friendsActivity = $state([]);
let _loadingFriends = $state(false);
let _serverInvite = $state('');

export const discordStore = {
    get isLinked() { return _discordLinked; },
    get linkInfo() { return _discordLinkInfo; },
    get friendsActivity() { return _friendsActivity; },
    get loadingFriends() { return _loadingFriends; },
    get serverInvite() { return _serverInvite; },
    
    setLinked(linked, info = null) {
        _discordLinked = linked;
        _discordLinkInfo = info;
    },
    
    setFriendsActivity(activity) {
        _friendsActivity = activity || [];
    },
    
    setLoadingFriends(loading) {
        _loadingFriends = loading;
    },
    
    setServerInvite(invite) {
        _serverInvite = invite;
    },
    
    updateShowStatus(value) {
        if (_discordLinkInfo) {
            _discordLinkInfo = { ..._discordLinkInfo, showStatus: value };
        }
    },
    
    updateShareAnimes(value) {
        if (_discordLinkInfo) {
            _discordLinkInfo = { ..._discordLinkInfo, shareAnimes: value };
        }
    },
    
    clear() {
        _discordLinked = false;
        _discordLinkInfo = null;
        _friendsActivity = [];
    }
};

// === UI STORE ===
let _currentView = $state('home');
let _activeTab = $state('anime');
let _showSplash = $state(true);
let _splashProgress = $state(0);
let _splashStatus = $state('Iniciando...');
let _appReady = $state(false);
let _userMenuOpen = $state(false);
let _carregando = $state(false);

export const uiStore = {
    get currentView() { return _currentView; },
    get activeTab() { return _activeTab; },
    get showSplash() { return _showSplash; },
    get splashProgress() { return _splashProgress; },
    get splashStatus() { return _splashStatus; },
    get appReady() { return _appReady; },
    get userMenuOpen() { return _userMenuOpen; },
    get carregando() { return _carregando; },
    
    setCurrentView(view) {
        _currentView = view;
        _userMenuOpen = false;
    },
    
    setActiveTab(tab) {
        _activeTab = tab;
    },
    
    setSplash(show, progress = 0, status = '') {
        _showSplash = show;
        if (progress !== undefined) _splashProgress = progress;
        if (status) _splashStatus = status;
    },
    
    updateSplashProgress(progress, status = '') {
        _splashProgress = progress;
        if (status) _splashStatus = status;
    },
    
    setAppReady(ready) {
        _appReady = ready;
    },
    
    toggleUserMenu() {
        _userMenuOpen = !_userMenuOpen;
    },
    
    closeUserMenu() {
        _userMenuOpen = false;
    },
    
    setCarregando(loading) {
        _carregando = loading;
    }
};

// === ANIME STORE ===
let _topAnimes = $state([]);
let _trendingAnimes = $state([]);
let _featuredAnime = $state(null);
let _featuredIndex = $state(0);
let _resultadosBusca = $state([]);
let _selectedGenre = $state(null);
let _termoBusca = $state('');

export const animeStore = {
    get topAnimes() { return _topAnimes; },
    get trendingAnimes() { return _trendingAnimes; },
    get featuredAnime() { return _featuredAnime; },
    get featuredIndex() { return _featuredIndex; },
    get resultadosBusca() { return _resultadosBusca; },
    get selectedGenre() { return _selectedGenre; },
    get termoBusca() { return _termoBusca; },
    
    setTopAnimes(animes) {
        _topAnimes = animes || [];
    },
    
    setTrendingAnimes(animes) {
        _trendingAnimes = animes || [];
        // Atualiza featured automaticamente
        if (animes?.length > 0) {
            const withBanners = animes.filter(a => a.banner);
            if (withBanners.length > 0) {
                _featuredAnime = withBanners[0];
                _featuredIndex = 0;
            } else if (animes[0]?.image) {
                _featuredAnime = { ...animes[0], banner: animes[0].image };
                _featuredIndex = 0;
            }
        }
    },
    
    setFeatured(anime, index = 0) {
        _featuredAnime = anime;
        _featuredIndex = index;
    },
    
    nextFeatured() {
        if (_trendingAnimes.length > 0) {
            _featuredIndex = (_featuredIndex + 1) % Math.min(_trendingAnimes.length, 10);
            _featuredAnime = _trendingAnimes[_featuredIndex];
        }
    },
    
    setResultadosBusca(results) {
        _resultadosBusca = Array.isArray(results) ? results : [];
    },
    
    setSelectedGenre(genre) {
        _selectedGenre = genre;
    },
    
    setTermoBusca(termo) {
        _termoBusca = termo;
    },
    
    clearSearch() {
        _selectedGenre = null;
        _termoBusca = '';
        _resultadosBusca = [];
    }
};

// === PLAYER STORE ===
let _selectedAnime = $state(null);
let _episodes = $state([]);
let _seasons = $state([]);
let _selectedSeason = $state(1);
let _selectedEpisodeURL = $state('');
let _currentPlayingEpisodeTitle = $state('');
let _playingEpisodeNatively = $state(false);
let _playerUrl = $state('');
let _originalStreamUrl = $state('');
let _episodeSelectionScreen = $state(false);
let _loadingEpisodes = $state(false);
let _availableSources = $state([]);
let _selectedSource = $state(null);
let _showSourceSelector = $state(false);
let _currentSkipTimes = $state(null);
let _currentMalID = $state(0);
let _currentEpisodeNumber = $state(1);

export const playerStore = {
    get selectedAnime() { return _selectedAnime; },
    get episodes() { return _episodes; },
    get seasons() { return _seasons; },
    get selectedSeason() { return _selectedSeason; },
    get selectedEpisodeURL() { return _selectedEpisodeURL; },
    get currentPlayingEpisodeTitle() { return _currentPlayingEpisodeTitle; },
    get playingEpisodeNatively() { return _playingEpisodeNatively; },
    get playerUrl() { return _playerUrl; },
    get originalStreamUrl() { return _originalStreamUrl; },
    get episodeSelectionScreen() { return _episodeSelectionScreen; },
    get loadingEpisodes() { return _loadingEpisodes; },
    get availableSources() { return _availableSources; },
    get selectedSource() { return _selectedSource; },
    get showSourceSelector() { return _showSourceSelector; },
    get currentSkipTimes() { return _currentSkipTimes; },
    get currentMalID() { return _currentMalID; },
    get currentEpisodeNumber() { return _currentEpisodeNumber; },
    
    get filteredEpisodes() {
        return _selectedSeason 
            ? _episodes.filter(e => (e.Season || 1) === _selectedSeason)
            : _episodes;
    },
    
    setSelectedAnime(anime) {
        _selectedAnime = anime;
        _availableSources = anime?.Sources || [];
    },
    
    setEpisodes(eps) {
        _episodes = Array.isArray(eps) ? eps : [];
        // Calcula temporadas
        const s = new Set();
        _episodes.forEach(e => s.add(e.Season || 1));
        _seasons = Array.from(s).sort((a, b) => a - b);
        if (_seasons.length > 0) _selectedSeason = _seasons[0];
    },
    
    setSelectedSeason(season) {
        _selectedSeason = season;
    },
    
    setSelectedEpisodeURL(url) {
        _selectedEpisodeURL = url;
    },
    
    setPlaying(playing, url = '', originalUrl = '', title = '') {
        _playingEpisodeNatively = playing;
        _playerUrl = url;
        _originalStreamUrl = originalUrl;
        _currentPlayingEpisodeTitle = title;
    },
    
    setEpisodeSelectionScreen(show) {
        _episodeSelectionScreen = show;
    },
    
    setLoadingEpisodes(loading) {
        _loadingEpisodes = loading;
    },
    
    setSource(source) {
        _selectedSource = source;
        _showSourceSelector = false;
    },
    
    setShowSourceSelector(show) {
        _showSourceSelector = show;
    },
    
    setSkipTimes(times, malID = 0, epNumber = 1) {
        _currentSkipTimes = times;
        _currentMalID = malID;
        _currentEpisodeNumber = epNumber;
    },
    
    reset() {
        _selectedAnime = null;
        _episodes = [];
        _seasons = [];
        _selectedSeason = 1;
        _selectedEpisodeURL = '';
        _currentPlayingEpisodeTitle = '';
        _playingEpisodeNatively = false;
        _playerUrl = '';
        _originalStreamUrl = '';
        _episodeSelectionScreen = false;
        _loadingEpisodes = false;
        _availableSources = [];
        _selectedSource = null;
        _showSourceSelector = false;
        _currentSkipTimes = null;
        _currentMalID = 0;
        _currentEpisodeNumber = 1;
    },
    
    closePlayer() {
        _playingEpisodeNatively = false;
        _playerUrl = '';
        _originalStreamUrl = '';
        _currentPlayingEpisodeTitle = '';
        _currentSkipTimes = null;
    }
};

// === FAVORITES & HISTORY STORE ===
let _favorites = $state([]);
let _watchHistory = $state([]);

export const libraryStore = {
    get favorites() { return _favorites; },
    get watchHistory() { return _watchHistory; },
    
    setFavorites(favs) {
        _favorites = favs || [];
    },
    
    setWatchHistory(history) {
        _watchHistory = history || [];
    }
};

// === CACHE STORE ===
let _cacheStats = $state(null);
let _episodeCache = new Map();
let _urlCache = new Map();
let _prefetchedAnimes = new Set();

export const cacheStore = {
    get stats() { return _cacheStats; },
    get episodeCache() { return _episodeCache; },
    get urlCache() { return _urlCache; },
    get prefetchedAnimes() { return _prefetchedAnimes; },
    
    setStats(stats) {
        _cacheStats = stats;
    },
    
    cacheEpisodes(key, episodes) {
        _episodeCache.set(key, episodes);
    },
    
    getCachedEpisodes(key) {
        return _episodeCache.get(key);
    },
    
    cacheUrl(key, url) {
        _urlCache.set(key, url);
    },
    
    getCachedUrl(key) {
        return _urlCache.get(key);
    },
    
    markPrefetched(key) {
        _prefetchedAnimes.add(key);
    },
    
    isPrefetched(key) {
        return _prefetchedAnimes.has(key);
    },
    
    clearAll() {
        _episodeCache.clear();
        _urlCache.clear();
        _prefetchedAnimes.clear();
        _cacheStats = null;
    }
};
