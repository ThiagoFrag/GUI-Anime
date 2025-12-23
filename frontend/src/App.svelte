<script>
    import { onMount, onDestroy, tick } from 'svelte';
    import SimplePlayer from './SimplePlayer.svelte';
    import Player4KButton from './Player4KButton.svelte';
    import { 
        GetCurrentUser, CreateUser, BuscarAnimes, BuscarAnimesMulti, GetTopAnimes, GetAnimeURL, 
        GetEpisodes, GetEpisodesForSource, PlayAnime, IsMPVInstalled,
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
        SetDiscordShowStatus, SetDiscordShareAnimes,
        // Manga System
        GetPopularMangas, GetLatestMangas, SearchMangas, GetMangaDetails, GetMangaChapters, GetChapterPages, GetMangasByGenre, GetAllMangasComplete,
        GetFeaturedMangas, GetAllMangasSafe, GetAllMangasAdult, GetPopularMangasSafe, GetPopularMangasAdult,
        // Manga Multiple Sources
        GetMangaSourcesInfo, GetFeaturedMangasFromSource, GetPopularMangasFromSource, GetLatestMangasFromSource, 
        SearchMangasFromSource, GetAllMangasFromSourceComplete, GetPopularMangasAllSources,
        GetMergedMangasWithBestSource,
        // Manga Source Management
        GetAllMangaSources, GetEnabledMangaSources, ToggleMangaSource, IsMangaSourceEnabled,
        GetMangaSourcesByLanguage, GetAvailableLanguages, ResetMangaSources,
        // Anime Multiple Sources
        GetAnimeSourcesInfo,
        // Utilitários
        GetAnimePoster, GetAnimePostersMulti,
        // Remote VPS API
        RemoteSearchAnimes, RemoteSearchTorrents, RemoteGetStreamLink, RemoteGetStreamLinkWithTorrent, RemoteGetTorrentFiles,
        RemoteGetEpisodes, RemoteGetRecentReleases, RemoteHealthCheck, InitRemoteConnection,
        // TorBox LOCAL (streaming direto, não passa pela VPS)
        IsTorBoxConfigured, TorBoxGetFilesFromMagnet, TorBoxGetStreamLinkLocal,
        // VPS Player API - Pipeline TorBox -> GoFile
        VPSStartPipeline, VPSSearchStream, VPSGetStreamURL, VPSCheckHealth, VPSGetEpisodeGoFile,
        // Player4K - Upscaling AI
        PlayWithPlayer4K, PlayWithPlayer4KTitle, PlayWithPlayer4KTitleSub, GetPlayer4KModes, IsPlayer4KAvailable,
        // Social / Friends System (Novo)
        HasSocialProfile, GetSocialProfile, CreateSocialProfile, UpdateSocialUsername,
        RegenerateSocialShareCode, AddFriendByCode, RemoveFriend, GetFriendsList,
        GetFriendsActivity, UpdateSocialWatchingStatus, ClearSocialWatchingStatus,
        SetSocialShowStatus, SetSocialShareAnimes, DeleteSocialProfile,
        GetSocialConnectionStatus, SyncSocialWithServer,
        // Seeding / Semeamento Comunitário
        StartSeeding, StopSeeding, ToggleSeeding, GetSeedingStats, IsSeedingRunning,
        // Autenticação
        AuthRegister, AuthLogin, AuthLoginAsGuest, AuthLogout, AuthGetSession, AuthIsLoggedIn, AuthIsGuest,
        AuthSetSeedingEnabled, AuthGetSeedingEnabled,
        // Episode Parser V2 - Agrupamento robusto
        ParseEpisodeFilenamesV2
    } from '../wailsjs/go/main/App';
    import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';

    // === AUTHENTICATION ===
    let authMode = 'login'; // 'login' | 'register' | 'guest'
    let loginUsername = '';
    let loginPassword = '';
    let registerUsername = '';
    let registerEmail = '';
    let registerPassword = '';
    let registerConfirmPassword = '';
    let authError = '';
    let authLoading = false;
    let isGuest = false;

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
    let groupedResults = {}; // Resultados agrupados por série
    let expandedGroups = {}; // Controla quais grupos estão expandidos
    let carregando = false;
    let termoBusca = "";
    
    // === NAVIGATION TABS ===
    let activeTab = 'anime'; // 'anime' | 'manga' | 'community' | 'friends'
    
    // === SOCIAL / FRIENDS SYSTEM (Novo - Sem Discord) ===
    let socialProfile = null;       // Perfil do usuário { userId, username, shareCode, ... }
    let socialFriends = [];         // Lista de amigos
    let friendsActivity = [];       // Atividade dos amigos
    let loadingFriends = false;
    let hasProfile = false;
    let socialConnected = false;    // Status de conexão com o servidor
    let socialConnectionMsg = "";   // Mensagem de status da conexão
    
    // === SOCIAL UI ===
    let showCreateProfileModal = false;
    let newUsername = "";
    let showAddFriendModal = false;
    let friendCode = "";
    let socialError = "";
    let socialLoading = false;
    
    // === SEEDING / SEMEAMENTO COMUNITÁRIO ===
    let seedingStats = null;        // Estatísticas de semeamento
    let seedingRunning = false;     // Se está rodando
    let seedingInterval = null;     // Intervalo para atualizar stats
    
    // === DISCORD INTEGRATION (Mantido para compatibilidade - será removido) ===
    let discordLinked = false;
    let discordLinkInfo = null;
    
    // === DISCORD LINKING UI (Legacy) ===
    let showLinkModal = false;
    let linkCode = "";
    let linkError = "";
    let linkLoading = false;
    let discordServerInvite = "";
    
    // === COMMUNITY ===
    let communityAnimes = [];
    let loadingCommunity = false;
    
    // === MANGA SYSTEM ===
    let allMangas = [];  // TODOS os mangás do site (SFW apenas por padrão)
    let featuredMangas = [];  // Mangás em destaque (24 mangás SFW)
    let popularMangas = [];
    let latestMangas = [];
    let adultMangas = [];  // Mangás +18 (separados)
    let showAdultContent = false;  // Toggle para exibir conteúdo adulto
    let mangaSearchResults = [];
    let loadingMangas = false;
    let selectedManga = null;
    let mangaChapters = [];
    let selectedChapter = null;
    let chapterPages = [];
    let loadingChapterPages = false;
    let mangaSearchTerm = "";
    let selectedMangaGenre = null;
    
    // === MANGA SOURCES ===
    let mangaSources = [];  // Lista de fontes disponíveis
    let selectedMangaSource = 'all';  // Fonte selecionada ('all', 'mangalivre.to', 'mangalivre.blog')
    let showMangaSourcesModal = false;  // Modal de gerenciamento de fontes
    let allMangaSources = [];  // Todas as fontes disponíveis (para o modal)
    let mangaSourcesLoading = false;  // Loading state para o modal
    let selectedMangaSourceLanguage = 'all';  // Filtro de idioma no modal
    const mangaGenres = [
        { id: 'acao', name: 'Ação', icon: '⚔️' },
        { id: 'aventura', name: 'Aventura', icon: '🗺️' },
        { id: 'comedia', name: 'Comédia', icon: '😂' },
        { id: 'drama', name: 'Drama', icon: '🎭' },
        { id: 'fantasia', name: 'Fantasia', icon: '✨' },
        { id: 'romance', name: 'Romance', icon: '💕' },
        { id: 'shounen', name: 'Shounen', icon: '💪' },
        { id: 'seinen', name: 'Seinen', icon: '🎯' },
        { id: 'isekai', name: 'Isekai', icon: '🌀' },
        { id: 'slice-of-life', name: 'Slice of Life', icon: '🌸' },
        { id: 'sobrenatural', name: 'Sobrenatural', icon: '👁️' },
        { id: 'manhwa', name: 'Manhwa', icon: '🇰🇷' },
    ];

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
    
    // === ANIME SOURCES ===
    let animeSources = [];  // Lista de fontes disponíveis
    let selectedAnimeSource = 'all';  // Fonte selecionada ('all', 'enime', 'consumet', 'vps')
    
    // === VPS REMOTE API (inclui TorBox) ===
    let vpsConnected = false;  // Se a conexão com a VPS está estabelecida
    let vpsRecentReleases = [];  // Lançamentos recentes da VPS
    
    // Episodes & Playback
    let selectedAnime = null;
    let episodes = [];
    let seasons = [];
    let selectedSeason = 1;
    let selectedEpisodeURL = "";
    let currentPlayingEpisodeTitle = "";
    
    // VPS TorBox files
    let vpsTorrentFiles = [];
    let vpsTorrentInfo = null;
    
    // === EPISÓDIOS AGRUPADOS V2 ===
    // Estrutura: { numero: 1, titulo_limpo: "...", versoes: [...] }
    let groupedEpisodes = [];
    let selectedGroupedEpisode = null;  // Episódio agrupado selecionado (para mostrar versões)
    let selectedVersion = null;         // Versão selecionada dentro do episódio
    
    // VPS Pipeline states
    let vpsHealthy = false;
    let pipelineProcessing = false;
    let pipelineStatus = '';
    let lastPipelineJob = null;
    
    // === PLAYER MODAL AVANÇADO ===
    let showPlayerModal = false;
    let playerModalEpisode = null;
    let selectedUpscaleMode = 'medium';  // low, medium, high
    let showShareOptions = false;
    let generatingShareLink = false;
    let shareLink = '';
    let showQualityInfo = false;
    
    // === LEGENDAS ===
    let showSubtitleOptions = false;
    let availableSubtitles = [];
    let selectedSubtitle = null;
    let subtitleUrl = '';
    let loadingSubtitles = false;
    
    // Modos de upscaling disponíveis
    const upscaleModes = [
        { id: 'low', name: 'Rápido', icon: '⚡', desc: 'Menor uso de GPU, ideal para PCs mais fracos', gpu: 'GTX 1050+' },
        { id: 'medium', name: 'Balanceado', icon: '⚖️', desc: 'Equilíbrio entre qualidade e performance', gpu: 'GTX 1060+' },
        { id: 'high', name: 'Máxima', icon: '✨', desc: 'Melhor qualidade, requer GPU potente', gpu: 'RTX 2060+' }
    ];
    
    // ===== FUNÇÕES DE AGRUPAMENTO POR SÉRIE/TEMPORADA =====
    
    // Normaliza caracteres especiais para comparação
    function normalizeForComparison(str) {
        if (!str) return '';
        return str
            .normalize('NFD').replace(/[\u0300-\u036f]/g, '') // Remove acentos
            .toLowerCase()
            .replace(/[^\w\s]/g, ' ')
            .replace(/\s+/g, ' ')
            .trim();
    }
    
    // Extrai nome base do anime removendo temporada, episódio, etc
    function extractBaseAnimeName(title) {
        if (!title) return 'Desconhecido';
        
        return title
            // 1. Remove tags técnicas e colchetes/parênteses primeiro
            .replace(/\[[^\]]*\]|\([^)]*\)|\{[^}]*\}/gi, '')
            // 2. CORREÇÃO CRÍTICA: Remove episódio mesmo colado (ex: AnimeE01 ou Anime - 01)
            .replace(/(?:S\d+)?E\d+/gi, '') 
            .replace(/\s+-\s+\d+.*$/i, '')
            // 3. Remove temporadas (S01, Season 1, 1ª Temporada, etc)
            .replace(/\s*[-–—:]?\s*(?:Season|S)\s*\d+/gi, '')
            .replace(/\s*\d+[ªº°]?\s*Temporada[s]?/gi, '')
            .replace(/\s*Temporada[s]?\s*\d+/gi, '')
            // 4. Remove metadados comuns brasileiros e técnicos
            .replace(/\s+(?:1080p|720p|480p|2160p|4K|CR|WEB-DL|WEB|DUAL|AAC2?\.?0?|H\.?264|H\.?265|HEVC|x264|x265|VARYG|AMZN|DDP2?\.?0?|MULTi|Rede\s*Torrent|Comando\s*Torrent|BluRay|BD|10-?bit|FLAC|Atmos|Multi-Subs|Dual-Audio).*/gi, '')
            // 5. Remove subtítulos de episódio (ex: "Battle Without a Quirk")
            .replace(/\s+(?:Battle|Wrench|Historys|Greatest|Villain|Open|Agora|Vez)[^-]*/gi, '')
            // 6. Limpa espaços duplos e pontuação no fim
            .replace(/\s+/g, ' ')
            .replace(/[-–—_:]+$/, '')
            .trim() || 'Desconhecido';
    }
    
    // Extrai informação de temporada do título - MELHORADO PARA PT-BR
    function extractSeasonInfo(title) {
        if (!title) return { season: 0, label: '' };
        
        // Padrões brasileiros (Rede Torrent) - PRIORIDADE
        const brPatterns = [
            // "Todas as Temporadas" - marca como especial (PRIMEIRO!)
            { regex: /Todas\s*(?:as)?\s*Temporada[s]?/i, group: 0, value: 0, isAll: true },
            // "Serie Completa", "Completo" (sem temporada especificada = todas)
            { regex: /(?:Serie\s*)?Complet[oa]\s*$/i, group: 0, value: 0, isAll: true },
            // "1ª a 5ª Temporada" - range de temporadas
            { regex: /(\d+)[ªº°]?\s*(?:a|até|ate|-|–)\s*(\d+)[ªº°]?\s*Temporada/i, group: 1, isRange: true, rangeGroup: 2 },
            // "1ª Temporada", "2ª Temporada", "1º Temporada"
            { regex: /(\d+)[ªº°]\s*Temporada/i, group: 1 },
            // "Temporada 1", "Temporada 2"
            { regex: /Temporada\s*(\d+)/i, group: 1 },
            // "T1", "T2" (abreviação comum)
            { regex: /\bT(\d+)\b(?!\w)/i, group: 1 },
        ];
        
        // Testa padrões brasileiros primeiro
        for (const pattern of brPatterns) {
            const match = title.match(pattern.regex);
            if (match) {
                if (pattern.isAll) {
                    return { season: 0, label: 'Todas Temporadas', isAll: true };
                }
                if (pattern.isRange) {
                    // Range de temporadas (1ª a 5ª)
                    const start = parseInt(match[pattern.group]);
                    const end = parseInt(match[pattern.rangeGroup]);
                    return { season: start, label: `Temporadas ${start}-${end}`, isRange: true, rangeEnd: end };
                }
                const num = parseInt(match[pattern.group]);
                return { season: num, label: `Temporada ${num}` };
            }
        }
        
        // Padrões em inglês
        const enPatterns = [
            // "Complete Series"
            { regex: /Complete\s*Series/i, group: 0, value: 0, isAll: true },
            // "Season 1", "Season 2"
            { regex: /Season\s*(\d+)/i, group: 1 },
            // "S01", "S1", "S02" (mas não S1080p)
            { regex: /\bS(\d{1,2})\b(?![\d])/i, group: 1 },
            // "1st Season", "2nd Season"
            { regex: /(\d+)(?:st|nd|rd|th)\s+Season/i, group: 1 },
            // "Part 1", "Part 2", "Parte 1"
            { regex: /Part[e]?\s*(\d+)/i, group: 1 },
            // "Cour 1", "Cour 2"
            { regex: /Cour\s*(\d+)/i, group: 1 },
        ];
        
        for (const pattern of enPatterns) {
            const match = title.match(pattern.regex);
            if (match) {
                if (pattern.isAll) {
                    return { season: 0, label: 'Todas Temporadas', isAll: true };
                }
                const num = parseInt(match[pattern.group]);
                return { season: num, label: `Temporada ${num}` };
            }
        }
        
        // Detecta numerais romanos no título (II, III, IV, etc.)
        const romanMatch = title.match(/\b(II|III|IV|V|VI|VII|VIII|IX|X)\b/);
        if (romanMatch) {
            const romanMap = { 'II': 2, 'III': 3, 'IV': 4, 'V': 5, 'VI': 6, 'VII': 7, 'VIII': 8, 'IX': 9, 'X': 10 };
            const num = romanMap[romanMatch[1]];
            if (num) return { season: num, label: `Temporada ${num}` };
        }
        
        // Detecta número no final do nome do anime (Mob Psycho 100 II → não, mas "Anime 2" sim)
        // Só se não for um número grande como "100" ou ano como "2023"
        const trailingNum = title.match(/\s(\d{1})\s*(?:$|[-–—])/);
        if (trailingNum) {
            const num = parseInt(trailingNum[1]);
            if (num >= 2 && num <= 9) {
                return { season: num, label: `Temporada ${num}` };
            }
        }
        
        return { season: 1, label: 'Temporada 1' };
    }
    
    // Calcula similaridade entre dois strings (0-1)
    function stringSimilarity(str1, str2) {
        const s1 = normalizeForComparison(str1);
        const s2 = normalizeForComparison(str2);
        
        if (s1 === s2) return 1;
        if (!s1 || !s2) return 0;
        
        // Se um contém o outro, alta similaridade
        if (s1.includes(s2) || s2.includes(s1)) {
            const longer = s1.length > s2.length ? s1 : s2;
            const shorter = s1.length > s2.length ? s2 : s1;
            return shorter.length / longer.length;
        }
        
        // Levenshtein simplificado para strings curtos
        const words1 = s1.split(' ');
        const words2 = s2.split(' ');
        
        let matches = 0;
        for (const w1 of words1) {
            if (words2.some(w2 => w1 === w2 || (w1.length > 3 && w2.includes(w1)) || (w2.length > 3 && w1.includes(w2)))) {
                matches++;
            }
        }
        
        return matches / Math.max(words1.length, words2.length);
    }
    
    // Encontra o melhor grupo existente para um nome base
    function findBestGroup(groups, baseName) {
        const normalized = normalizeForComparison(baseName);
        
        // Primeiro, busca match exato
        if (groups[baseName]) return baseName;
        
        // Busca por nome normalizado
        for (const groupName of Object.keys(groups)) {
            if (normalizeForComparison(groupName) === normalized) {
                return groupName;
            }
        }
        
        // Busca por alta similaridade (>= 80%)
        let bestMatch = null;
        let bestSimilarity = 0;
        
        for (const groupName of Object.keys(groups)) {
            const similarity = stringSimilarity(baseName, groupName);
            if (similarity >= 0.8 && similarity > bestSimilarity) {
                bestSimilarity = similarity;
                bestMatch = groupName;
            }
        }
        
        return bestMatch;
    }
    
    // Agrupa resultados por série e fonte, priorizando Rede Torrent
    function groupResultsBySeries(results) {
        const groups = {};
        
        results.forEach(item => {
            const baseName = extractBaseAnimeName(item.CleanTitle || item.Title);
            const seasonInfo = extractSeasonInfo(item.Title);
            const source = item._vps_torrent?.source || 'Nyaa';
            const isRedeTorrent = source.toLowerCase().includes('redetorrent') || 
                                  source.toLowerCase().includes('comando') ||
                                  item._vps_torrent?.is_brazilian;
            
            // Tenta encontrar grupo existente similar
            const existingGroup = findBestGroup(groups, baseName);
            const groupKey = existingGroup || baseName;
            
            // Cria grupo se não existe
            if (!groups[groupKey]) {
                groups[groupKey] = {
                    name: groupKey,
                    alternateNames: [baseName],
                    image: item.Image || '',
                    seasons: {},
                    totalItems: 0,
                    hasRedeTorrent: false,
                    hasAllSeasons: false
                };
            } else if (!groups[groupKey].alternateNames.includes(baseName)) {
                groups[groupKey].alternateNames.push(baseName);
            }
            
            // Atualiza imagem se ainda não tem
            if (!groups[groupKey].image && item.Image) {
                groups[groupKey].image = item.Image;
            }
            
            // Se é "Todas as Temporadas" ou range, usa key especial
            let seasonKey = seasonInfo.isAll ? 0 : (seasonInfo.season || 1);
            let seasonLabel = seasonInfo.label || `Temporada ${seasonKey}`;
            
            if (seasonInfo.isAll) {
                seasonLabel = '📦 Todas Temporadas (Completo)';
                groups[groupKey].hasAllSeasons = true;
            } else if (seasonInfo.isRange) {
                seasonLabel = `📚 ${seasonInfo.label}`;
            }
            
            // Cria temporada se não existe
            if (!groups[groupKey].seasons[seasonKey]) {
                groups[groupKey].seasons[seasonKey] = {
                    label: seasonLabel,
                    isAll: seasonInfo.isAll || false,
                    isRange: seasonInfo.isRange || false,
                    sources: {
                        redeTorrent: [],
                        nyaa: [],
                        other: []
                    }
                };
            }
            
            // Adiciona item com info extra
            const enrichedItem = {
                ...item,
                _seasonInfo: seasonInfo,
                _isRedeTorrent: isRedeTorrent
            };
            
            // Adiciona item à fonte apropriada
            if (isRedeTorrent) {
                groups[groupKey].seasons[seasonKey].sources.redeTorrent.push(enrichedItem);
                groups[groupKey].hasRedeTorrent = true;
            } else if (source.toLowerCase().includes('nyaa')) {
                groups[groupKey].seasons[seasonKey].sources.nyaa.push(enrichedItem);
            } else {
                groups[groupKey].seasons[seasonKey].sources.other.push(enrichedItem);
            }
            
            groups[groupKey].totalItems++;
        });
        
        // Ordena grupos: primeiro os que têm Rede Torrent
        const sortedGroups = Object.entries(groups)
            .sort((a, b) => {
                // Prioriza grupos com Rede Torrent
                if (a[1].hasRedeTorrent && !b[1].hasRedeTorrent) return -1;
                if (!a[1].hasRedeTorrent && b[1].hasRedeTorrent) return 1;
                // Depois por quantidade de itens
                return b[1].totalItems - a[1].totalItems;
            })
            .reduce((acc, [key, value]) => {
                acc[key] = value;
                return acc;
            }, {});
        
        return sortedGroups;
    }
    
    // Toggle expansão de grupo
    function toggleGroup(groupName) {
        expandedGroups = {
            ...expandedGroups,
            [groupName]: !expandedGroups[groupName]
        };
    }
    
    // Reactive: atualiza grupos quando resultados mudam
    $: {
        if (resultadosBusca.length > 0 && selectedAnimeSource === 'vps') {
            groupedResults = groupResultsBySeries(resultadosBusca);
            // Expande primeiro grupo por padrão
            const firstGroup = Object.keys(groupedResults)[0];
            if (firstGroup && Object.keys(expandedGroups).length === 0) {
                expandedGroups = { [firstGroup]: true };
            }
        } else {
            groupedResults = {};
        }
    }
    
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
        use_anime4k: true,
        // Seeding / Contribuição
        seeding_enabled: false,
        seeding_max_cpu: 50,
        seeding_max_bandwidth: 10,
        seeding_only_wifi: false,
        seeding_schedule: 'always',
        seeding_contributed: 0
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
            
            // Carrega estado do Sistema Social (novo)
            loadSocialState();
            
            // Carrega estado do Discord em background (legacy)
            loadDiscordState();
            
            // Carrega fontes de anime
            await loadAnimeSources();
            console.log('[onMount] animeSources após carregar:', animeSources);
            
            // Inicializa conexão com VPS API (inclui TorBox)
            try {
                const connected = await InitRemoteConnection('');
                vpsConnected = connected;
                console.log('[VPS API] Conexão estabelecida:', connected);
                
                if (connected) {
                    // Remove fontes antigas e usa apenas VPS
                    animeSources = [{ 
                        id: 'vps', 
                        name: 'VPS Server', 
                        icon: '🌐',
                        description: 'Servidor dedicado com TorBox integrado'
                    }];
                    // Define VPS como fonte padrão
                    selectedAnimeSource = 'vps';
                }
                
                // Verifica se o Player API do VPS está online (porta 3002)
                checkVpsHealth();
            } catch (err) {
                console.error('[VPS API] Erro ao conectar:', err);
            }
            
            // Inicializa estado do seeding
            try {
                // Primeiro verifica se o seeding deveria estar rodando baseado nas configurações
                const s = await GetSettings();
                if (s && s.seeding_enabled) {
                    // Tenta iniciar o seeding se não estiver rodando
                    const alreadyRunning = await IsSeedingRunning();
                    if (!alreadyRunning) {
                        console.log('[Seeding] Iniciando automaticamente (configuração habilitada)...');
                        await ToggleSeeding(true);
                    }
                }
                
                seedingRunning = await IsSeedingRunning();
                if (seedingRunning) {
                    seedingStats = await GetSeedingStats();
                    // Inicia intervalo para atualizar stats a cada 30s
                    seedingInterval = setInterval(async () => {
                        try {
                            seedingStats = await GetSeedingStats();
                            seedingRunning = await IsSeedingRunning();
                        } catch (e) {
                            console.error('[Seeding] Erro ao atualizar stats:', e);
                        }
                    }, 30000);
                }
                console.log('[Seeding] Estado inicial:', seedingRunning ? 'Ativo' : 'Inativo');
            } catch (err) {
                console.error('[Seeding] Erro ao verificar estado:', err);
            }
            
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
            if (seedingInterval) clearInterval(seedingInterval);
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
                    use_anime4k: s.use_anime4k !== false,
                    // Seeding / Contribuição
                    seeding_enabled: s.seeding_enabled || false,
                    seeding_max_cpu: s.seeding_max_cpu || 50,
                    seeding_max_bandwidth: s.seeding_max_bandwidth || 10,
                    seeding_only_wifi: s.seeding_only_wifi || false,
                    seeding_schedule: s.seeding_schedule || 'always',
                    seeding_contributed: s.seeding_contributed || 0
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
            
            // Ativa/desativa o semeamento conforme configuração
            try {
                await ToggleSeeding(settings.seeding_enabled);
                seedingRunning = await IsSeedingRunning();
                
                // Gerencia o intervalo de atualização de stats
                if (settings.seeding_enabled && !seedingInterval) {
                    // Busca stats iniciais
                    seedingStats = await GetSeedingStats();
                    // Inicia intervalo para atualizar stats a cada 30s
                    seedingInterval = setInterval(async () => {
                        try {
                            seedingStats = await GetSeedingStats();
                            seedingRunning = await IsSeedingRunning();
                        } catch (e) {
                            console.error('[Seeding] Erro ao atualizar stats:', e);
                        }
                    }, 30000);
                } else if (!settings.seeding_enabled && seedingInterval) {
                    // Para o intervalo quando desativado
                    clearInterval(seedingInterval);
                    seedingInterval = null;
                }
            } catch (seedErr) {
                console.warn('Seeding toggle error:', seedErr);
            }
            
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
    
    // === SOCIAL / FRIENDS SYSTEM FUNCTIONS ===
    
    // Verifica status da conexão com o servidor social
    async function checkSocialConnection() {
        try {
            const result = await GetSocialConnectionStatus();
            // A função retorna (bool, string) - Wails converte para array ou valor único
            if (typeof result === 'boolean') {
                socialConnected = result;
                socialConnectionMsg = result ? "Conectado" : "Desconectado";
            } else if (Array.isArray(result)) {
                socialConnected = result[0] === true;
                socialConnectionMsg = result[1] || "";
            } else {
                socialConnected = !!result;
                socialConnectionMsg = String(result || "");
            }
        } catch (err) {
            socialConnected = false;
            socialConnectionMsg = "Erro ao verificar conexão";
        }
    }
    
    // Carrega o estado inicial do sistema social
    async function loadSocialState() {
        try {
            // Verifica conexão primeiro
            await checkSocialConnection();
            
            hasProfile = await HasSocialProfile();
            
            if (hasProfile) {
                socialProfile = await GetSocialProfile();
                socialFriends = await GetFriendsList() || [];
            }
        } catch (err) {
            console.error('Error loading social state:', err);
        }
    }
    
    // Cria um novo perfil social
    async function createProfile() {
        if (!newUsername.trim()) {
            socialError = "Digite um nome de usuário";
            return;
        }
        
        socialLoading = true;
        socialError = "";
        
        try {
            socialProfile = await CreateSocialProfile(newUsername.trim());
            hasProfile = true;
            showCreateProfileModal = false;
            newUsername = "";
        } catch (err) {
            socialError = err.message || "Erro ao criar perfil";
        } finally {
            socialLoading = false;
        }
    }
    
    // Adiciona um amigo pelo código
    async function addFriend() {
        if (!friendCode.trim()) {
            socialError = "Digite o código do amigo";
            return;
        }
        
        socialLoading = true;
        socialError = "";
        
        try {
            const friend = await AddFriendByCode(friendCode.trim().toUpperCase());
            socialFriends = [...socialFriends, friend];
            showAddFriendModal = false;
            friendCode = "";
        } catch (err) {
            socialError = err.message || "Código inválido ou usuário não encontrado";
        } finally {
            socialLoading = false;
        }
    }
    
    // Remove um amigo
    async function removeFriendById(userId) {
        try {
            await RemoveFriend(userId);
            socialFriends = socialFriends.filter(f => f.userId !== userId);
        } catch (err) {
            console.error('Error removing friend:', err);
        }
    }
    
    // Carrega atividade dos amigos (novo sistema)
    async function loadSocialFriendsActivity() {
        if (!hasProfile) return;
        loadingFriends = true;
        
        try {
            const activities = await GetFriendsActivity();
            friendsActivity = activities || [];
        } catch (err) {
            console.error('Error loading friends activity:', err);
            friendsActivity = [];
        } finally {
            loadingFriends = false;
        }
    }
    
    // Gera novo código de compartilhamento
    async function regenerateCode() {
        try {
            const newCode = await RegenerateSocialShareCode();
            if (socialProfile) {
                socialProfile.shareCode = newCode;
            }
        } catch (err) {
            console.error('Error regenerating code:', err);
        }
    }
    
    // Copia código para clipboard
    async function copyShareCode() {
        if (socialProfile?.shareCode) {
            try {
                await navigator.clipboard.writeText(socialProfile.shareCode);
                // Feedback visual - altera temporariamente o texto
                const originalCode = socialProfile.shareCode;
                socialProfile = { ...socialProfile, shareCode: '✓ Copiado!' };
                setTimeout(() => {
                    socialProfile = { ...socialProfile, shareCode: originalCode };
                }, 1500);
            } catch (err) {
                console.error('Error copying code:', err);
            }
        }
    }
    
    // Toggle para mostrar status (social)
    async function toggleSocialShowStatus() {
        if (!socialProfile) return;
        
        const newValue = !socialProfile.showStatus;
        try {
            await SetSocialShowStatus(newValue);
            socialProfile.showStatus = newValue;
        } catch (err) {
            console.error('Error toggling show status:', err);
        }
    }
    
    // Toggle para compartilhar animes (social)
    async function toggleSocialShareAnimes() {
        if (!socialProfile) return;
        
        const newValue = !socialProfile.shareAnimes;
        try {
            await SetSocialShareAnimes(newValue);
            socialProfile.shareAnimes = newValue;
        } catch (err) {
            console.error('Error toggling share animes:', err);
        }
    }
    
    // Atualiza status de visualização (social)
    async function updateSocialWatchingStatus(animeTitle, episodeNum, animeImage, totalEpisodes = 0) {
        if (!hasProfile || !socialProfile?.showStatus) return;
        
        try {
            await UpdateSocialWatchingStatus(animeTitle, episodeNum, animeImage, totalEpisodes);
        } catch (err) {
            console.error('Error updating social watching status:', err);
        }
    }
    
    // Limpa status de visualização
    async function clearSocialWatchingStatus() {
        if (!hasProfile) return;
        
        try {
            await ClearSocialWatchingStatus();
        } catch (err) {
            console.error('Error clearing social watching status:', err);
        }
    }
    
    // Deleta perfil social
    async function deleteSocialProfile() {
        if (confirm("Tem certeza que deseja apagar seu perfil? Todos os seus amigos serão removidos.")) {
            try {
                await DeleteSocialProfile();
                socialProfile = null;
                socialFriends = [];
                friendsActivity = [];
                hasProfile = false;
            } catch (err) {
                console.error('Error deleting profile:', err);
            }
        }
    }

    // === ANIME SOURCES FUNCTIONS ===
    
    // Carrega as fontes de anime disponíveis
    async function loadAnimeSources() {
        try {
            console.log('[Anime] Carregando fontes...');
            const sources = await GetAnimeSourcesInfo();
            console.log('[Anime] Resposta do backend:', sources);
            animeSources = sources || [];
            console.log('[Anime] Fontes disponíveis:', animeSources.map(s => s.name));
        } catch (err) {
            console.error('[Anime] Erro ao carregar fontes:', err);
            // Fallback para fontes padrão
            animeSources = [
                { id: 'enime', name: 'Enime', description: 'Fonte rápida', language: 'en', priority: 1 },
                { id: 'consumet', name: 'Consumet', description: 'Fonte confiável', language: 'en', priority: 2 }
            ];
        }
    }
    
    // Altera a fonte de anime selecionada
    async function changeAnimeSource(sourceId) {
        selectedAnimeSource = sourceId;
        console.log('[Anime] Fonte alterada para:', sourceId);
        
        // Limpa resultados anteriores ao mudar fonte
        resultadosBusca = [];
        selectedGenre = null;
    }

    // === MANGA FUNCTIONS ===
    
    // Carrega as fontes de mangá disponíveis
    async function loadMangaSources() {
        try {
            const sources = await GetMangaSourcesInfo();
            mangaSources = sources || [];
            console.log('[Manga] Fontes disponíveis:', mangaSources.map(s => s.name));
        } catch (err) {
            console.error('[Manga] Erro ao carregar fontes:', err);
            mangaSources = [];
        }
    }

    // Abre o modal de gerenciamento de fontes
    async function openMangaSourcesModal() {
        showMangaSourcesModal = true;
        mangaSourcesLoading = true;
        try {
            allMangaSources = await GetAllMangaSources();
            console.log('[Manga] Fontes carregadas para gerenciamento:', allMangaSources.length);
        } catch (err) {
            console.error('[Manga] Erro ao carregar fontes:', err);
            allMangaSources = [];
        } finally {
            mangaSourcesLoading = false;
        }
    }

    // Fecha o modal de gerenciamento de fontes
    function closeMangaSourcesModal() {
        showMangaSourcesModal = false;
        // Recarrega as fontes após fechar o modal
        loadMangaSources();
    }

    // Alterna o estado de uma fonte (habilitar/desabilitar)
    async function toggleMangaSourceEnabled(sourceId, currentEnabled) {
        try {
            await ToggleMangaSource(sourceId, !currentEnabled);
            // Atualiza o estado local
            allMangaSources = allMangaSources.map(s => 
                s.id === sourceId ? { ...s, enabled: !currentEnabled } : s
            );
            console.log(`[Manga] Fonte ${sourceId} ${!currentEnabled ? 'habilitada' : 'desabilitada'}`);
        } catch (err) {
            console.error('[Manga] Erro ao alternar fonte:', err);
        }
    }

    // Reseta as fontes para o padrão
    async function resetMangaSourcesToDefault() {
        try {
            await ResetMangaSources();
            allMangaSources = await GetAllMangaSources();
            console.log('[Manga] Fontes resetadas para padrão');
        } catch (err) {
            console.error('[Manga] Erro ao resetar fontes:', err);
        }
    }

    // Filtra fontes por idioma
    function getFilteredMangaSources() {
        if (selectedMangaSourceLanguage === 'all') {
            return allMangaSources;
        }
        return allMangaSources.filter(s => s.language === selectedMangaSourceLanguage);
    }
    
    // Altera a fonte de mangá selecionada
    async function changeMangaSource(sourceId) {
        selectedMangaSource = sourceId;
        console.log('[Manga] Fonte alterada para:', sourceId);
        
        // Limpa dados anteriores
        featuredMangas = [];
        allMangas = [];
        adultMangas = [];
        
        // Recarrega com a nova fonte
        await loadMangaData();
    }
    
    // Carrega mangás em destaque (apenas SFW por padrão)
    async function loadMangaData() {
        loadingMangas = true;
        try {
            // Carrega fontes se ainda não carregou
            if (mangaSources.length === 0) {
                await loadMangaSources();
            }
            
            const sourceToUse = selectedMangaSource === 'all' ? '' : selectedMangaSource;
            console.log('[Manga] Carregando mangás em destaque da fonte:', sourceToUse || 'todas');
            
            // Busca mangás em destaque (24 mangás populares SFW)
            const featured = await GetFeaturedMangasFromSource(24, sourceToUse);
            featuredMangas = featured || [];
            
            // Popular = os mesmos em destaque
            popularMangas = featuredMangas.slice(0, 12);
            
            // Últimos = últimas atualizações
            let latest;
            if (sourceToUse) {
                latest = await GetLatestMangasFromSource(sourceToUse);
            } else {
                latest = await GetLatestMangas();
            }
            latestMangas = (latest || []).slice(0, 12);
            
            // allMangas fica vazio até o usuário clicar em "Ver Todos"
            allMangas = [];
            
            console.log('[Manga] Mangás em destaque carregados:', featuredMangas.length);
        } catch (err) {
            console.error('[Manga] Erro ao carregar:', err);
            featuredMangas = [];
            popularMangas = [];
            latestMangas = [];
        } finally {
            loadingMangas = false;
        }
    }
    
    // Carrega TODOS os mangás (SFW) quando usuário clicar em "Ver Todos"
    async function loadAllMangasSafe() {
        loadingMangas = true;
        try {
            const sourceToUse = selectedMangaSource === 'all' ? '' : selectedMangaSource;
            console.log('[Manga] Carregando todos os mangás (SFW) da fonte:', sourceToUse || 'todas (merge inteligente)');
            
            let result;
            if (sourceToUse) {
                result = await GetAllMangasFromSourceComplete(sourceToUse);
                // Filtra adultos no frontend já que GetAllMangasFromSourceComplete não filtra
                result = (result || []).filter(m => !isAdultMangaClient(m.genres));
            } else {
                // Usa merge inteligente: combina fontes escolhendo a versão com mais capítulos
                console.log('[Manga] Usando merge inteligente de todas as fontes...');
                result = await GetMergedMangasWithBestSource();
                result = (result || []).filter(m => !isAdultMangaClient(m.genres));
            }
            allMangas = result || [];
            console.log('[Manga] Total de mangás SFW:', allMangas.length);
        } catch (err) {
            console.error('[Manga] Erro ao carregar todos:', err);
            allMangas = [];
        } finally {
            loadingMangas = false;
        }
    }
    
    // Helper para verificar se é conteúdo hentai no frontend
    function isAdultMangaClient(genres) {
        if (!genres || !Array.isArray(genres)) return false;
        // Apenas hentai e conteúdo explícito
        const adultGenres = ['hentai', '+18', 'r18', 'r-18'];
        return genres.some(g => adultGenres.some(a => g.toLowerCase().includes(a)));
    }
    
    // Carrega conteúdo adulto (+18)
    async function loadAdultMangas() {
        if (adultMangas.length > 0) return; // Já carregado
        
        loadingMangas = true;
        try {
            console.log('[Manga] Carregando mangás adultos (+18)...');
            const result = await GetAllMangasAdult();
            adultMangas = result || [];
            console.log('[Manga] Total de mangás +18:', adultMangas.length);
        } catch (err) {
            console.error('[Manga] Erro ao carregar adultos:', err);
            adultMangas = [];
        } finally {
            loadingMangas = false;
        }
    }
    
    // Toggle para exibir conteúdo adulto
    async function toggleAdultContent() {
        showAdultContent = !showAdultContent;
        if (showAdultContent && adultMangas.length === 0) {
            await loadAdultMangas();
        }
    }
    
    // Busca mangás
    async function searchManga() {
        if (!mangaSearchTerm.trim()) {
            mangaSearchResults = [];
            return;
        }
        
        loadingMangas = true;
        try {
            mangaSearchResults = await SearchMangas(mangaSearchTerm) || [];
            console.log('[Manga] Busca retornou:', mangaSearchResults.length, 'resultados');
        } catch (err) {
            console.error('[Manga] Erro na busca:', err);
            mangaSearchResults = [];
        } finally {
            loadingMangas = false;
        }
    }
    
    // Seleciona um mangá para ver detalhes
    async function selectManga(manga) {
        selectedManga = manga;
        mangaChapters = [];
        loadingMangas = true;
        
        try {
            // Carrega detalhes e capítulos
            const [details, chapters] = await Promise.all([
                GetMangaDetails(manga.url),
                GetMangaChapters(manga.url)
            ]);
            
            if (details) {
                selectedManga = { ...manga, ...details };
            }
            mangaChapters = chapters || [];
            console.log('[Manga] Capítulos carregados:', mangaChapters.length);
        } catch (err) {
            console.error('[Manga] Erro ao carregar detalhes:', err);
        } finally {
            loadingMangas = false;
        }
    }
    
    // Seleciona um capítulo para ler
    async function selectChapter(chapter) {
        selectedChapter = chapter;
        chapterPages = [];
        loadingChapterPages = true;
        
        try {
            chapterPages = await GetChapterPages(chapter.url) || [];
            console.log('[Manga] Páginas carregadas:', chapterPages.length);
        } catch (err) {
            console.error('[Manga] Erro ao carregar páginas:', err);
            chapterPages = [];
        } finally {
            loadingChapterPages = false;
        }
    }
    
    // Volta do leitor de mangá
    function closeMangaReader() {
        selectedChapter = null;
        chapterPages = [];
    }
    
    // Volta dos detalhes do mangá
    function closeMangaDetails() {
        selectedManga = null;
        mangaChapters = [];
    }
    
    // Próximo capítulo
    function nextChapter() {
        if (!selectedChapter || mangaChapters.length === 0) return;
        const currentIndex = mangaChapters.findIndex(c => c.url === selectedChapter.url);
        if (currentIndex > 0) {
            selectChapter(mangaChapters[currentIndex - 1]);
        }
    }
    
    // Capítulo anterior
    function prevChapter() {
        if (!selectedChapter || mangaChapters.length === 0) return;
        const currentIndex = mangaChapters.findIndex(c => c.url === selectedChapter.url);
        if (currentIndex < mangaChapters.length - 1) {
            selectChapter(mangaChapters[currentIndex + 1]);
        }
    }
    
    // Busca mangás por gênero
    async function loadMangasByGenre(genre) {
        selectedMangaGenre = genre;
        loadingMangas = true;
        mangaSearchResults = [];
        
        try {
            mangaSearchResults = await GetMangasByGenre(genre.name) || [];
            console.log('[Manga] Gênero', genre.name, ':', mangaSearchResults.length, 'resultados');
        } catch (err) {
            console.error('[Manga] Erro ao buscar gênero:', err);
        } finally {
            loadingMangas = false;
        }
    }
    
    // Limpa filtro de gênero
    function clearMangaGenre() {
        selectedMangaGenre = null;
        mangaSearchResults = [];
    }
    
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
    
    function formatBytes(bytes) {
        if (!bytes || bytes === 0) return '0 B';
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`;
    }
    
    // === TAB NAVIGATION ===
    function switchTab(tab) {
        activeTab = tab;
        if (tab === 'friends' && discordLinked) {
            loadFriendsActivity();
        }
        if (tab === 'manga' && allMangas.length === 0) {
            loadMangaData();
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

    // === AUTENTICAÇÃO ===
    async function handleLogin() {
        if (!loginUsername || !loginPassword) {
            authError = 'Preencha usuário e senha';
            return;
        }
        authLoading = true;
        authError = '';
        try {
            const session = await AuthLogin(loginUsername, loginPassword);
            if (session) {
                usuario = await GetCurrentUser();
                isGuest = false;
                await carregarDados();
            }
        } catch (err) {
            console.error('Login error:', err);
            authError = err.toString().replace('Error: ', '');
        } finally {
            authLoading = false;
        }
    }

    async function handleRegister() {
        if (!registerUsername || !registerPassword) {
            authError = 'Preencha todos os campos obrigatórios';
            return;
        }
        if (registerPassword !== registerConfirmPassword) {
            authError = 'As senhas não conferem';
            return;
        }
        if (registerPassword.length < 6) {
            authError = 'A senha deve ter pelo menos 6 caracteres';
            return;
        }
        authLoading = true;
        authError = '';
        try {
            const session = await AuthRegister(registerUsername, registerEmail, registerPassword, avatarSelecionado);
            if (session) {
                usuario = await GetCurrentUser();
                isGuest = false;
                await carregarDados();
            }
        } catch (err) {
            console.error('Register error:', err);
            authError = err.toString().replace('Error: ', '');
        } finally {
            authLoading = false;
        }
    }

    async function handleGuestLogin() {
        authLoading = true;
        authError = '';
        try {
            const session = await AuthLoginAsGuest();
            if (session) {
                usuario = { username: 'Visitante', avatar: 'guest.png' };
                isGuest = true;
                await carregarDados();
            }
        } catch (err) {
            console.error('Guest login error:', err);
            authError = 'Erro ao entrar como visitante';
        } finally {
            authLoading = false;
        }
    }

    async function handleLogout() {
        try {
            await AuthLogout();
            usuario = null;
            isGuest = false;
            authMode = 'login';
        } catch (err) {
            console.error('Logout error:', err);
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
            // Se VPS selecionado, usa API da VPS (inclui TorBox)
            if (selectedAnimeSource === 'vps') {
                console.log('[Pesquisar] Usando VPS API para buscar:', termoBusca);
                
                // Primeiro tenta buscar torrents via VPS (que usa TorBox interno)
                const torrentResults = await RemoteSearchTorrents(termoBusca, false);
                console.log('[Pesquisar] VPS torrents retornou:', torrentResults);
                
                if (Array.isArray(torrentResults) && torrentResults.length > 0) {
                    // Backend já agrupa e retorna clean_title - usar diretamente!
                    const mapped = torrentResults.map((t, i) => {
                        // Pega o título completo do torrent
                        let fullTitle = t.title || `Torrent ${i+1}`;
                        
                        // Remove HTML se houver
                        if (fullTitle.includes('<')) {
                            fullTitle = fullTitle.replace(/<[^>]*>/g, '').trim();
                        }
                        
                        // Usa CleanTitle do backend (já calculado para busca de imagem)
                        let cleanTitle = t.clean_title || fullTitle
                            .replace(/\[[^\]]*\]/g, '')
                            .replace(/\([^)]*\)/g, '')
                            .replace(/\{[^}]*\}/g, '')
                            .replace(/\s+/g, ' ')
                            .trim();
                        
                        // Se ficou muito curto, usa o termo de busca
                        if (cleanTitle.length < 3) {
                            cleanTitle = termoBusca;
                        }
                        
                        // Identificar idioma pela fonte
                        const source = (t.source || 'nyaa.si').toLowerCase();
                        let language = 'multi';
                        let sourceName = t.source || 'nyaa.si';
                        
                        if (source.includes('redetorrent') || source.includes('comando')) {
                            language = 'pt-BR';
                            sourceName = `🇧🇷 ${sourceName}`;
                        } else if (source.includes('nyaa')) {
                            language = 'en';
                            sourceName = `🇺🇸 ${sourceName}`;
                        }
                        
                        // is_brazilian também indica BR
                        if (t.is_brazilian) {
                            language = 'pt-BR';
                        }
                        
                        // Conta variantes se existirem
                        const variantCount = (t.variants && t.variants.length) || 0;
                        
                        return {
                            Title: fullTitle,
                            CleanTitle: cleanTitle,
                            Image: '',
                            URL: t.magnet || t.hash || '',
                            Hash: t.hash || '',
                            Seeds: t.seeds || 0,
                            Size: t.size || '',
                            VariantCount: variantCount,
                            Sources: [{
                                Name: `VPS • ${sourceName}`,
                                URL: t.magnet || '',
                                Language: language
                            }],
                            _vps_torrent: t,
                            _variants: t.variants || []
                        };
                    });
                    // ORDENA a lista principal por episódio (Decrescente: mais novos primeiro)
                    resultadosBusca = [...mapped].sort((a, b) => {
                        // Extrai número de episódio do título para ordenação
                        const epA = extractEpisodeNumber({ name: a.Title }, 0);
                        const epB = extractEpisodeNumber({ name: b.Title }, 0);
                        return epB - epA;  // Decrescente
                    });
                    await tick();
                    
                    // Busca capas usando CleanTitles ÚNICOS (muito menos downloads!)
                    const uniqueCleanTitles = [...new Set(mapped.map(a => a.CleanTitle))];
                    console.log(`[Pesquisar] Buscando ${uniqueCleanTitles.length} capas únicas (de ${mapped.length} resultados)`);
                    
                    GetAnimePostersMulti(uniqueCleanTitles).then(posterMap => {
                        if (posterMap) {
                            resultadosBusca = resultadosBusca.map(anime => ({
                                ...anime,
                                Image: posterMap[anime.CleanTitle] || posterMap[anime.Title] || anime.Image
                            }));
                        }
                    }).catch(() => {});
                } else {
                    resultadosBusca = [];
                }
            } else {
                // Busca normal em todas as fontes
                const res = (await BuscarAnimes(termoBusca)) || [];
                resultadosBusca = Array.isArray(res) ? res : [];
            }
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
        
        // ===== VPS TORBOX - MOSTRA ARQUIVOS PARA SELEÇÃO =====
        if (anime._vps_torrent) {
            console.log('[TorBox] Abrindo seleção de arquivos:', anime._vps_torrent.title);
            
            selectedAnime = anime;
            episodeSelectionScreen = true;
            loadingEpisodes = true;
            episodes = [];
            seasons = [];
            vpsTorrentFiles = [];
            vpsTorrentInfo = null;
            
            // Scroll para o topo
            setTimeout(() => scrollToTop(true), 50);
            
            // Busca a capa do anime usando Jikan/AniList (em paralelo)
            if (!anime.Image && !anime.CoverImage) {
                GetAnimePoster(anime.Title || anime._vps_torrent.title).then(poster => {
                    if (poster) {
                        console.log('[TorBox] Capa encontrada:', poster);
                        selectedAnime = { ...selectedAnime, Image: poster, CoverImage: poster };
                    }
                }).catch(e => console.warn('[TorBox] Erro ao buscar capa:', e));
            }
            
            try {
                console.log('[TorBox] DEBUG _vps_torrent:', JSON.stringify(anime._vps_torrent, null, 2));
                console.log('[TorBox] Magnet:', anime._vps_torrent.magnet);
                console.log('[TorBox] Hash:', anime._vps_torrent.hash);
                
                // PRIORIDADE: TorBox LOCAL (não sobrecarrega VPS)
                // Se usuário tem TorBox configurado localmente, usa direto
                const hasTorBoxLocal = await IsTorBoxConfigured();
                let torrentInfo = null;
                
                if (hasTorBoxLocal) {
                    console.log('[TorBox LOCAL] Usuário tem TorBox configurado, usando streaming direto');
                    torrentInfo = await TorBoxGetFilesFromMagnet(anime._vps_torrent.magnet);
                    if (torrentInfo) {
                        torrentInfo._useLocalTorBox = true; // Marca para usar streaming local
                    }
                }
                
                // Fallback: VPS (usa banda do servidor - mais lento)
                if (!torrentInfo) {
                    console.log('[TorBox VPS] Usando VPS como fallback (configure sua API key TorBox para melhor performance)');
                    torrentInfo = await RemoteGetTorrentFiles(anime._vps_torrent.magnet, anime._vps_torrent.hash);
                }
                
                if (torrentInfo && torrentInfo.files && torrentInfo.files.length > 0) {
                    console.log('[TorBox] Arquivos encontrados:', torrentInfo.files.length);
                    
                    vpsTorrentInfo = torrentInfo;
                    vpsTorrentFiles = torrentInfo.files;
                    
                    // ===== AGRUPAMENTO ESTRITO POR EPISÓDIO =====
                    // Dicionário para agrupar arquivos pelo número do episódio
                    const groups = {};
                    
                    torrentInfo.files.forEach((f, idx) => {
                        const epNum = extractEpisodeNumber(f, idx);
                        const seasonNum = f.season || 1;
                        
                        if (!groups[epNum]) {
                            groups[epNum] = {
                                Episode: epNum,
                                Title: extractBaseAnimeName(f.short_name || f.name || ''),
                                Season: seasonNum,
                                files: [],  // Arquivos deste episódio
                                URL: '',
                                StreamURL: '',
                                Source: 'TorBox'
                            };
                        }
                        // Adiciona o arquivo APENAS ao grupo do seu episódio
                        groups[epNum].files.push(f);
                    });
                    
                    // Converte para array e ORDENA numericamente (1, 2, 3...)
                    episodes = Object.values(groups)
                        .sort((a, b) => a.Episode - b.Episode)
                        .map(g => ({
                            ...g,
                            _isGrouped: true,
                            _vpsTorrentFile: g.files[0] || null,  // Primeiro arquivo como padrão
                            _versoes: g.files.map(f => ({
                                nome_original: f.name || f.short_name,
                                qualidade: detectQuality(f.name || f.short_name || ''),
                                tamanho: f.size_str || f.size || '',
                                _vpsTorrentFile: f
                            }))
                        }));
                    
                    console.log('[TorBox] Episódios agrupados e ordenados:', episodes.length);
                    
                    // Extrai seasons únicas dos episódios agrupados
                    const uniqueSeasons = [...new Set(episodes.map(e => e.Season))].sort((a, b) => a - b);
                    seasons = uniqueSeasons.length > 0 ? uniqueSeasons : [1];
                    selectedSeason = seasons[0];
                    
                } else {
                    console.error('[TorBox] Nenhum arquivo de vídeo encontrado. TorrentInfo:', torrentInfo);
                    const status = torrentInfo?.status || 'unknown';
                    const progress = torrentInfo?.progress || 0;
                    
                    if (status === 'metaDL' || status === 'downloading') {
                        alert(`Torrent ainda está baixando metadados/arquivos.\n\nStatus: ${status}\nProgresso: ${progress.toFixed(1)}%\n\nTente novamente em alguns segundos.`);
                    } else {
                        alert(`Nenhum arquivo de vídeo encontrado neste torrent.\n\nStatus: ${status}\nProgress: ${progress.toFixed(1)}%`);
                    }
                }
            } catch (err) {
                console.error('[VPS TorBox] Erro ao obter arquivos:', err);
                alert('Erro ao conectar com VPS TorBox: ' + err.message);
            } finally {
                loadingEpisodes = false;
            }
            return;
        }
        
        // ===== FLUXO NORMAL (NÃO TORBOX) =====
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
            // DETECTA FONTE VPS/TORBOX
            // Se a fonte é VPS ou a URL é magnet/hash, usa fluxo TorBox
            if (source.Name?.includes('VPS') || source.URL?.startsWith('magnet:') || source.URL?.match(/^[a-f0-9]{40}$/i)) {
                console.log('[selectSource] Detectada fonte VPS/TorBox, usando RemoteGetTorrentFiles');
                
                // Extrai hash do magnet ou usa URL como hash
                let hash = '';
                let magnet = source.URL;
                
                if (source.URL.startsWith('magnet:')) {
                    const hashMatch = source.URL.match(/btih:([a-f0-9]+)/i);
                    hash = hashMatch ? hashMatch[1].toLowerCase() : '';
                } else {
                    hash = source.URL.toLowerCase();
                    magnet = '';
                }
                
                // Busca arquivos via RemoteGetTorrentFiles
                const torrentInfo = await RemoteGetTorrentFiles(magnet, hash);
                
                if (torrentInfo && torrentInfo.files && torrentInfo.files.length > 0) {
                    console.log('[selectSource] Arquivos TorBox encontrados:', torrentInfo.files.length);
                    
                    vpsTorrentInfo = torrentInfo;
                    vpsTorrentFiles = torrentInfo.files;
                    
                    // ===== AGRUPAMENTO ESTRITO POR EPISÓDIO =====
                    const groups = {};
                    
                    torrentInfo.files.forEach((f, idx) => {
                        const epNum = extractEpisodeNumber(f, idx);
                        const seasonNum = f.season || 1;
                        
                        if (!groups[epNum]) {
                            groups[epNum] = {
                                Episode: epNum,
                                Title: extractBaseAnimeName(f.short_name || f.name || ''),
                                Season: seasonNum,
                                files: [],
                                URL: '',
                                StreamURL: '',
                                Source: 'TorBox'
                            };
                        }
                        groups[epNum].files.push(f);
                    });
                    
                    episodes = Object.values(groups)
                        .sort((a, b) => a.Episode - b.Episode)
                        .map(g => ({
                            ...g,
                            _isGrouped: true,
                            _vpsTorrentFile: g.files[0] || null,
                            _versoes: g.files.map(f => ({
                                nome_original: f.name || f.short_name,
                                qualidade: detectQuality(f.name || f.short_name || ''),
                                tamanho: f.size_str || f.size || '',
                                _vpsTorrentFile: f
                            }))
                        }));
                    
                    console.log('[selectSource] Episódios agrupados:', episodes.length);
                    
                    const uniqueSeasons = [...new Set(episodes.map(e => e.Season))].sort((a, b) => a - b);
                    seasons = uniqueSeasons.length > 0 ? uniqueSeasons : [1];
                    selectedSeason = seasons[0];
                } else {
                    console.error('[selectSource] Nenhum arquivo TorBox encontrado');
                    alert('Nenhum arquivo de vídeo encontrado neste torrent.');
                    episodes = [];
                }
                
                loadingEpisodes = false;
                return;
            }
            
            // FLUXO NORMAL (fontes de streaming tradicionais)
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

            // DETECTA FONTE VPS/TORBOX
            const sourceIsVPS = selectedSource?.Name?.includes('VPS') || 
                               seriesURL?.startsWith('magnet:') || 
                               seriesURL?.match(/^[a-f0-9]{40}$/i);
            
            if (sourceIsVPS) {
                console.log('[loadEpisodesFromSource] Detectada fonte VPS/TorBox');
                
                // Extrai hash do magnet ou usa URL como hash
                let hash = '';
                let magnet = seriesURL;
                
                if (seriesURL.startsWith('magnet:')) {
                    const hashMatch = seriesURL.match(/btih:([a-f0-9]+)/i);
                    hash = hashMatch ? hashMatch[1].toLowerCase() : '';
                } else {
                    hash = seriesURL.toLowerCase();
                    magnet = '';
                }
                
                // Busca arquivos via RemoteGetTorrentFiles
                const torrentInfo = await RemoteGetTorrentFiles(magnet, hash);
                
                if (torrentInfo && torrentInfo.files && torrentInfo.files.length > 0) {
                    console.log('[loadEpisodesFromSource] Arquivos TorBox encontrados:', torrentInfo.files.length);
                    
                    vpsTorrentInfo = torrentInfo;
                    vpsTorrentFiles = torrentInfo.files;
                    
                    // ===== AGRUPAMENTO ESTRITO POR EPISÓDIO =====
                    const groups = {};
                    
                    torrentInfo.files.forEach((f, idx) => {
                        const epNum = extractEpisodeNumber(f, idx);
                        const seasonNum = f.season || 1;
                        
                        if (!groups[epNum]) {
                            groups[epNum] = {
                                Episode: epNum,
                                Title: extractBaseAnimeName(f.short_name || f.name || ''),
                                Season: seasonNum,
                                files: [],
                                URL: '',
                                StreamURL: '',
                                Source: 'TorBox'
                            };
                        }
                        groups[epNum].files.push(f);
                    });
                    
                    episodes = Object.values(groups)
                        .sort((a, b) => a.Episode - b.Episode)
                        .map(g => ({
                            ...g,
                            _isGrouped: true,
                            _vpsTorrentFile: g.files[0] || null,
                            _versoes: g.files.map(f => ({
                                nome_original: f.name || f.short_name,
                                qualidade: detectQuality(f.name || f.short_name || ''),
                                tamanho: f.size_str || f.size || '',
                                _vpsTorrentFile: f
                            }))
                        }));
                    
                    console.log('[loadEpisodesFromSource] Episódios agrupados:', episodes.length);
                    
                    const uniqueSeasons = [...new Set(episodes.map(e => e.Season))].sort((a, b) => a - b);
                    seasons = uniqueSeasons.length > 0 ? uniqueSeasons : [1];
                    selectedSeason = seasons[0];
                } else {
                    console.error('[loadEpisodesFromSource] Nenhum arquivo TorBox encontrado');
                    alert('Nenhum arquivo de vídeo encontrado neste torrent.');
                    episodes = [];
                }
                
                loadingEpisodes = false;
                return;
            }

            // FLUXO NORMAL - Cache de episódios
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
    
    // ===== TORBOX: Reproduzir arquivo selecionado =====
    // Prioriza TorBox LOCAL (streaming direto) para não sobrecarregar VPS
    async function playVpsTorrentFile(episode) {
        if (!episode._vpsTorrentFile) {
            alert('Erro: Informações do arquivo não encontradas');
            return;
        }
        
        const file = episode._vpsTorrentFile;
        console.log('[TorBox] Reproduzindo arquivo:', file.name);
        
        try {
            let streamUrl = '';
            
            // PRIORIDADE: TorBox LOCAL (streaming direto, não usa banda do servidor)
            // Usa torrentId (camelCase do JSON)
            const torrentId = file.torrentId || file.TorrentID || file.torrent_id;
            if (vpsTorrentInfo?._useLocalTorBox && torrentId) {
                console.log('[TorBox LOCAL] Usando streaming direto (não passa pela VPS)');
                console.log('[TorBox LOCAL] torrentId:', torrentId, 'fileId:', file.id);
                streamUrl = await TorBoxGetStreamLinkLocal(torrentId, file.id);
            }
            
            // Fallback: VPS (usa banda do servidor)
            if (!streamUrl) {
                console.log('[TorBox VPS] Usando VPS como fallback');
                // Usa torrentId diretamente se disponível para evitar chamada extra
                const torrentIdForStream = file.torrentId || file.TorrentID || file.torrent_id || 0;
                let streamLink;
                if (torrentIdForStream > 0) {
                    streamLink = await RemoteGetStreamLinkWithTorrent(vpsTorrentInfo?.hash || '', file.id, torrentIdForStream);
                } else {
                    streamLink = await RemoteGetStreamLink(vpsTorrentInfo?.hash || '', file.id);
                }
                streamUrl = streamLink?.direct_url || '';
            }
            
            if (!streamUrl) {
                alert('Erro: Não foi possível obter URL de streaming');
                return;
            }
            
            console.log('[TorBox] Stream URL obtida:', streamUrl);
            
            // Configura o player
            playerUrl = streamUrl;
            originalStreamUrl = streamUrl;
            playingEpisodeNatively = true;
            currentPlayingEpisodeTitle = `Ep ${file.episode || 1} - ${file.shortName || file.short_name || file.name}`;
            selectedEpisodeURL = `torrent:${file.id}`;
            
            // Configura o player
            setTimeout(() => setupVideoPlayer(), 100);
            
        } catch (err) {
            console.error('[TorBox] Erro ao reproduzir:', err);
            alert('Erro ao reproduzir arquivo TorBox: ' + err.message);
        }
    }
    
    // ===== VPS TORBOX: Reproduzir com Player HTML5 Interno =====
    async function playVpsTorrentFileInBrowser(episode) {
        if (!episode || !episode._vpsTorrentFile) {
            console.error('[TorBox Browser] Erro: Informações do arquivo não encontradas');
            alert('Erro: Informações do arquivo não encontradas');
            return;
        }
        
        const file = episode._vpsTorrentFile;
        console.log('[TorBox Browser] Reproduzindo no player interno:', file.name);
        
        try {
            let streamUrl = '';
            
            // Obtém URL de streaming
            const torrentId = file.torrentId || file.TorrentID || file.torrent_id;
            
            if (vpsTorrentInfo?._useLocalTorBox && torrentId) {
                console.log('[TorBox Browser] Usando TorBox LOCAL');
                try {
                    streamUrl = await TorBoxGetStreamLinkLocal(torrentId, file.id);
                } catch (localErr) {
                    console.error('[TorBox Browser] Erro local:', localErr);
                }
            }
            
            // Fallback: VPS
            if (!streamUrl) {
                console.log('[TorBox Browser] Usando VPS...');
                const hash = vpsTorrentInfo?.hash || '';
                let streamLink;
                
                if (torrentId > 0) {
                    streamLink = await RemoteGetStreamLinkWithTorrent(hash, file.id, torrentId);
                } else {
                    streamLink = await RemoteGetStreamLink(hash, file.id);
                }
                streamUrl = streamLink?.direct_url || '';
            }
            
            if (!streamUrl) {
                alert('Erro: Não foi possível obter URL de streaming');
                return;
            }
            
            console.log('[TorBox Browser] ✓ Stream URL:', streamUrl);
            
            // Configura o player HTML5 interno
            const animeName = selectedAnime?.Title || 'Anime';
            const episodeInfo = file.shortName || file.short_name || file.name || `Ep ${file.episode || 1}`;
            
            currentPlayingEpisodeTitle = `${animeName} - ${episodeInfo}`;
            playerUrl = streamUrl;
            playingEpisodeNatively = true;
            
            // Carrega skip times se disponível
            if (selectedAnime?.malId && file.episode) {
                loadSkipTimes(selectedAnime.malId, file.episode);
            }
            
            console.log('[TorBox Browser] ✓ Player interno iniciado!');
            
        } catch (err) {
            console.error('[TorBox Browser] Erro:', err);
            alert('Erro ao reproduzir: ' + (err.message || err));
        }
    }
    
    // ===== VPS TORBOX: Reproduzir com Player4K (Upscaling AI) =====
    async function playVpsTorrentFileWith4K(episode, subtitleUrlParam = '') {
        if (!episode || !episode._vpsTorrentFile) {
            console.error('[TorBox 4K] Erro: Informações do arquivo não encontradas', episode);
            alert('Erro: Informações do arquivo não encontradas');
            return;
        }
        
        const file = episode._vpsTorrentFile;
        console.log('[TorBox 4K] Reproduzindo com upscaling:', file.name);
        console.log('[TorBox 4K] Modo:', selectedUpscaleMode);
        console.log('[TorBox 4K] Legenda:', subtitleUrlParam || 'nenhuma');
        console.log('[TorBox 4K] File data:', JSON.stringify(file, null, 2));
        
        try {
            let streamUrl = '';
            
            // PRIORIDADE: TorBox LOCAL (streaming direto)
            const torrentId = file.torrentId || file.TorrentID || file.torrent_id;
            console.log('[TorBox 4K] torrentId:', torrentId);
            console.log('[TorBox 4K] vpsTorrentInfo:', vpsTorrentInfo);
            
            if (vpsTorrentInfo?._useLocalTorBox && torrentId) {
                console.log('[TorBox LOCAL 4K] Usando streaming direto');
                try {
                    streamUrl = await TorBoxGetStreamLinkLocal(torrentId, file.id);
                    console.log('[TorBox LOCAL 4K] URL:', streamUrl);
                } catch (localErr) {
                    console.error('[TorBox LOCAL 4K] Erro:', localErr);
                }
            }
            
            // Fallback: VPS
            if (!streamUrl) {
                console.log('[TorBox 4K] Tentando via VPS...');
                const torrentIdForStream = file.torrentId || file.TorrentID || file.torrent_id || 0;
                const hash = vpsTorrentInfo?.hash || '';
                console.log('[TorBox 4K] VPS params - hash:', hash, 'fileId:', file.id, 'torrentId:', torrentIdForStream);
                
                let streamLink;
                try {
                    if (torrentIdForStream > 0) {
                        streamLink = await RemoteGetStreamLinkWithTorrent(hash, file.id, torrentIdForStream);
                    } else {
                        streamLink = await RemoteGetStreamLink(hash, file.id);
                    }
                    console.log('[TorBox 4K] VPS streamLink:', streamLink);
                    streamUrl = streamLink?.direct_url || '';
                } catch (vpsErr) {
                    console.error('[TorBox 4K] Erro VPS:', vpsErr);
                }
            }
            
            if (!streamUrl) {
                console.error('[TorBox 4K] Não foi possível obter URL de streaming');
                alert('Erro: Não foi possível obter URL de streaming. Verifique o console para detalhes.');
                return;
            }
            
            console.log('[TorBox 4K] ✓ Stream URL obtida:', streamUrl);
            
            // Monta o título para o player
            const animeName = selectedAnime?.Title || 'Anime';
            const episodeInfo = file.shortName || file.short_name || file.name || `Ep ${file.episode || 1}`;
            const playerTitle = `${animeName} - ${episodeInfo}`;
            console.log('[TorBox 4K] Título:', playerTitle);
            
            // Tenta usar Player4K primeiro, depois MPV, e por último player interno
            let usedExternalPlayer = false;
            const mode = selectedUpscaleMode || 'medium';
            
            // 1. Tenta Player4K (nosso player com shaders)
            try {
                const player4kAvailable = await IsPlayer4KAvailable();
                console.log('[TorBox] Player4K disponível:', player4kAvailable);
                
                if (player4kAvailable) {
                    console.log('[TorBox] Iniciando Player4K com modo:', mode);
                    
                    if (subtitleUrlParam) {
                        await PlayWithPlayer4KTitleSub(streamUrl, playerTitle, mode, subtitleUrlParam);
                    } else {
                        await PlayWithPlayer4KTitle(streamUrl, playerTitle, mode);
                    }
                    usedExternalPlayer = true;
                    console.log('[TorBox] ✓ Player4K iniciado com sucesso!');
                }
            } catch (p4kErr) {
                console.warn('[TorBox] Player4K não disponível ou erro:', p4kErr);
            }
            
            // 2. Fallback: MPV simples
            if (!usedExternalPlayer) {
                try {
                    const mpvAvailable = await IsMPVInstalled();
                    console.log('[TorBox] MPV disponível:', mpvAvailable);
                    
                    if (mpvAvailable) {
                        console.log('[TorBox] Iniciando MPV externo...');
                        await PlayAnime(streamUrl);
                        usedExternalPlayer = true;
                        console.log('[TorBox] ✓ MPV iniciado com sucesso!');
                    }
                } catch (mpvErr) {
                    console.warn('[TorBox] MPV não disponível ou erro:', mpvErr);
                }
            }
            
            // 3. Fallback final: Player HTML5 interno
            if (!usedExternalPlayer) {
                console.log('[TorBox] Usando player HTML5 interno...');
                
                // Configura o player interno
                currentPlayingEpisodeTitle = playerTitle;
                playerUrl = streamUrl;
                playingEpisodeNatively = true;
                
                // Carrega skip times se disponível
                if (selectedAnime?.malId && file.episode) {
                    loadSkipTimes(selectedAnime.malId, file.episode);
                }
                
                console.log('[TorBox] ✓ Player interno iniciado!');
            }
            
        } catch (err) {
            console.error('[TorBox 4K] Erro ao reproduzir:', err);
            alert('Erro ao reproduzir com Player4K: ' + (err.message || err));
        }
    }
    
    // ===== PLAYER MODAL AVANÇADO =====
    function openPlayerModal(episode) {
        playerModalEpisode = episode;
        showPlayerModal = true;
        showShareOptions = false;
        showSubtitleOptions = false;
        subtitleUrl = '';
        selectedSubtitle = null;
        shareLink = '';
    }
    
    function closePlayerModal() {
        showPlayerModal = false;
        playerModalEpisode = null;
        showShareOptions = false;
        showSubtitleOptions = false;
        shareLink = '';
    }
    
    // Reproduzir com modo de upscale selecionado
    async function playWithSelectedMode() {
        console.log('[playWithSelectedMode] Iniciando...');
        console.log('[playWithSelectedMode] playerModalEpisode:', playerModalEpisode);
        
        if (!playerModalEpisode) {
            console.error('[playWithSelectedMode] playerModalEpisode is null');
            alert('Erro: Episódio não encontrado');
            return;
        }
        
        // Guarda referências ANTES de fechar o modal
        const episodeToPlay = playerModalEpisode;
        const currentSubtitleUrl = subtitleUrl;
        
        console.log('[playWithSelectedMode] Episode to play:', episodeToPlay);
        console.log('[playWithSelectedMode] _vpsTorrentFile:', episodeToPlay?._vpsTorrentFile);
        console.log('[playWithSelectedMode] Subtitle URL:', currentSubtitleUrl);
        
        // Fecha o modal (isso vai limpar playerModalEpisode)
        closePlayerModal();
        
        // Chama a função de reprodução com a referência salva
        try {
            await playVpsTorrentFileWith4K(episodeToPlay, currentSubtitleUrl);
        } catch (err) {
            console.error('[playWithSelectedMode] Erro ao reproduzir:', err);
            alert('Erro ao reproduzir: ' + (err.message || err));
        }
    }
    
    // Gerar link de compartilhamento
    async function generateShareLink() {
        if (!playerModalEpisode?._vpsTorrentFile) return;
        
        generatingShareLink = true;
        showShareOptions = true;
        
        try {
            const file = playerModalEpisode._vpsTorrentFile;
            const animeName = selectedAnime?.Title || 'Anime';
            const episodeInfo = file.shortName || file.short_name || file.name || `Ep ${file.episode || 1}`;
            
            // Monta link para compartilhar
            // Formato: goanime://play?anime=XXX&episode=XXX&torrent=XXX&file=XXX
            const shareData = {
                anime: animeName,
                episode: episodeInfo,
                hash: vpsTorrentInfo?.hash || '',
                fileId: file.id,
                torrentId: file.torrentId || file.TorrentID || file.torrent_id || 0
            };
            
            // Codifica em base64 para um link mais limpo
            const encoded = btoa(JSON.stringify(shareData));
            shareLink = `goanime://share/${encoded}`;
            
        } catch (err) {
            console.error('[Share] Erro:', err);
            shareLink = 'Erro ao gerar link';
        } finally {
            generatingShareLink = false;
        }
    }
    
    // Copiar link para área de transferência
    async function copyShareLink() {
        if (!shareLink) return;
        try {
            await navigator.clipboard.writeText(shareLink);
            alert('Link copiado! 📋\n\nEnvie para seu amigo e ele poderá assistir diretamente no GoAnime!');
        } catch (err) {
            // Fallback
            prompt('Copie o link abaixo:', shareLink);
        }
    }
    
    // Extrair informações de qualidade do arquivo
    function getFileQualityInfo(file) {
        if (!file) return null;
        
        const name = (file.name || file.short_name || '').toLowerCase();
        const info = {
            resolution: '720p',
            codec: 'H.264',
            audio: 'AAC',
            dualAudio: false,
            hdr: false,
            size: file.size_str || formatBytes(file.size)
        };
        
        // Detectar resolução
        if (name.includes('2160p') || name.includes('4k')) info.resolution = '4K';
        else if (name.includes('1080p')) info.resolution = '1080p';
        else if (name.includes('720p')) info.resolution = '720p';
        else if (name.includes('480p')) info.resolution = '480p';
        
        // Detectar codec
        if (name.includes('hevc') || name.includes('x265') || name.includes('h265')) info.codec = 'HEVC/H.265';
        else if (name.includes('av1')) info.codec = 'AV1';
        else if (name.includes('x264') || name.includes('h264')) info.codec = 'H.264';
        
        // Detectar áudio
        if (name.includes('dual') || name.includes('multi')) info.dualAudio = true;
        if (name.includes('flac')) info.audio = 'FLAC';
        else if (name.includes('dts')) info.audio = 'DTS';
        else if (name.includes('ac3') || name.includes('dolby')) info.audio = 'Dolby';
        
        // HDR
        if (name.includes('hdr') || name.includes('dolby vision') || name.includes('dv')) info.hdr = true;
        
        return info;
    }
    
    // Extrair número do episódio do nome do arquivo (evita confundir com resolução 1080p, 720p, etc)
    function extractEpisodeNumber(file, index = 0) {
        if (!file) return index + 1;
        
        const name = file.name || file.short_name || '';
        
        // Primeiro remove padrões de resolução para evitar confusão
        const cleanName = name
            .replace(/\b(2160|1080|720|480|360)p?\b/gi, '')
            .replace(/\b(4K|UHD|FHD|HD)\b/gi, '')
            .replace(/\bx264\b/gi, '')
            .replace(/\bx265\b/gi, '')
            .replace(/\bHEVC\b/gi, '')
            .replace(/\b10bit\b/gi, '')
            .replace(/\bFLAC\b/gi, '');
        
        // Padrões brasileiros/portugueses
        // "Naruto - 01", "Episodio 01", "EP01", "E01", "Ep. 01"
        const patterns = [
            /[-–]\s*(\d{1,4})(?:\s|\.|\[|$)/,           // "- 01" ou "– 01"
            /\bE(?:p(?:isodio|isode)?\.?\s*)?(\d{1,4})\b/i,  // "E01", "Ep01", "Ep 01", "Episodio 01"
            /\b(?:EP|Ep|ep)\.?\s*(\d{1,4})\b/,          // "EP01", "Ep.01"
            /\bS\d{1,2}E(\d{1,4})\b/i,                  // "S01E01"
            /[\[\(](\d{1,4})[\]\)]/,                    // "[01]" ou "(01)"
            /\s(\d{1,4})(?:\s*[\[\(]|\.mkv|\.mp4)/i,    // " 01 [" ou " 01.mkv"
            // NOVO: Captura episódios "colados" como GachiakutaE22, ThingE11
            /[a-zA-Z]E(\d{1,4})(?:\b|$)/,               // "GachiakutaE22" → 22
        ];
        
        for (const pattern of patterns) {
            const match = cleanName.match(pattern);
            if (match) {
                const num = parseInt(match[1], 10);
                // Validar que é um número de episódio razoável (1-9999)
                if (num > 0 && num < 10000) {
                    return num;
                }
            }
        }
        
        // Se o backend já extraiu corretamente, usar
        if (file.episode && file.episode > 0 && file.episode < 10000) {
            // Verificar se não é resolução
            if (file.episode !== 1080 && file.episode !== 720 && file.episode !== 480 && file.episode !== 2160) {
                return file.episode;
            }
        }
        
        // Fallback: usar índice + 1
        return index + 1;
    }
    
    // ===== EPISODE PARSER V2: Agrupa arquivos por episódio =====
    // Retorna estrutura: [{ numero, titulo_limpo, versoes: [{nome_original, qualidade, tamanho, ...}] }]
    async function parseAndGroupEpisodes(files) {
        if (!files || files.length === 0) return [];
        
        try {
            // Extrai nomes dos arquivos para enviar ao backend
            const filenames = files.map(f => f.name || f.short_name || f.shortName || '');
            
            // Chama o parser robusto do backend (Go)
            const result = await ParseEpisodeFilenamesV2(filenames);
            
            if (!result || !result.episodios) {
                console.warn('[ParseV2] Backend retornou resultado inválido, usando fallback');
                return fallbackGrouping(files);
            }
            
            console.log('[ParseV2] Resultado do backend:', result);
            
            // Converte para o formato esperado pelo frontend
            // Mapeia cada episódio do backend com os arquivos originais
            const grouped = result.episodios.map(ep => {
                // Encontra os arquivos originais que correspondem a este episódio
                const matchingFiles = ep.arquivos_disponiveis.map(arq => {
                    // Encontra o arquivo original pelo nome
                    const originalFile = files.find(f => 
                        (f.name === arq.nome_original) || 
                        (f.short_name === arq.nome_original) ||
                        (f.shortName === arq.nome_original)
                    );
                    
                    return {
                        nome_original: arq.nome_original,
                        qualidade: arq.qualidade || detectQuality(arq.nome_original),
                        subgrupo: arq.subgrupo || '',
                        tags: arq.tags || [],
                        // Mantém referência ao arquivo original do TorBox
                        _vpsTorrentFile: originalFile || null,
                        tamanho: originalFile?.size_str || originalFile?.size || '',
                        id: originalFile?.id || null
                    };
                });
                
                return {
                    numero: ep.id_episodio,
                    temporada: ep.temporada || 1,
                    titulo_limpo: ep.titulo_exibicao_limpo || result.nome_anime,
                    titulo_episodio: ep.titulo_episodio_completo || '',
                    versoes: matchingFiles
                };
            });
            
            // Ordena por número de episódio
            grouped.sort((a, b) => a.numero - b.numero);
            
            console.log('[ParseV2] Episódios agrupados:', grouped.length);
            return grouped;
            
        } catch (err) {
            console.error('[ParseV2] Erro ao chamar backend:', err);
            return fallbackGrouping(files);
        }
    }
    
    // Fallback: agrupa localmente se o backend falhar
    function fallbackGrouping(files) {
        const groups = {};
        
        files.forEach((file, idx) => {
            const epNum = extractEpisodeNumber(file, idx);
            
            if (!groups[epNum]) {
                groups[epNum] = {
                    numero: epNum,
                    temporada: file.season || 1,
                    titulo_limpo: cleanFilenameForDisplay(file.name || file.short_name || ''),
                    titulo_episodio: '',
                    versoes: []
                };
            }
            
            groups[epNum].versoes.push({
                nome_original: file.name || file.short_name || '',
                qualidade: detectQuality(file.name || ''),
                subgrupo: '',
                tags: [],
                _vpsTorrentFile: file,
                tamanho: file.size_str || file.size || '',
                id: file.id
            });
        });
        
        return Object.values(groups).sort((a, b) => a.numero - b.numero);
    }
    
    // Limpa nome de arquivo para exibição (remove tags técnicas)
    function cleanFilenameForDisplay(filename) {
        if (!filename) return '';
        
        return filename
            // Remove extensão
            .replace(/\.(mkv|mp4|avi|webm)$/i, '')
            // Remove colchetes com conteúdo técnico
            .replace(/\[(?:SubsPlease|Erai-raws|ASW|HorribleSubs|1080p|720p|480p|HEVC|x264|x265|AAC|FLAC|10bit|WEB-DL|WEBRip|BluRay|BD|BDRip|MULTI|DUAL|CR|VARYG|ToonsHub)[^\]]*\]/gi, '')
            // Remove parênteses com conteúdo técnico  
            .replace(/\((?:1080p|720p|480p|HEVC|x264|x265|AAC|FLAC|10bit|WEB-DL|WEBRip|BluRay|BD|BDRip|MULTI|DUAL|CR|VARYG)[^\)]*\)/gi, '')
            // Remove tags soltas
            .replace(/\b(?:1080p|720p|480p|2160p|4K|UHD|HEVC|x264|x265|H\.?264|H\.?265|AAC|FLAC|AC3|DTS|10bit|WEB-DL|WEBRip|BluRay|BD|BDRip|REMUX|REPACK|PROPER|MULTI|DUAL|CR|VARYG|2\s*0|5\s*1)\b/gi, '')
            // Remove underscores e hífens múltiplos
            .replace(/_/g, ' ')
            .replace(/[-–—]+/g, ' - ')
            // Limpa espaços
            .replace(/\s+/g, ' ')
            .replace(/^\s*[-–—]\s*/, '')
            .replace(/\s*[-–—]\s*$/, '')
            .trim();
    }
    
    // Detecta qualidade do arquivo pelo nome
    function detectQuality(filename) {
        if (!filename) return 'SD';
        const name = filename.toUpperCase();
        if (name.includes('2160P') || name.includes('4K') || name.includes('UHD')) return '4K';
        if (name.includes('1080P') || name.includes('FHD')) return '1080p';
        if (name.includes('720P') || name.includes('HD')) return '720p';
        if (name.includes('480P')) return '480p';
        return 'SD';
    }
    
    // Limpa título do episódio para exibição no card
    // Remove todo "lixo" técnico deixando apenas o nome legível do episódio
    function cleanEpisodeTitle(fileName, epNum, animeName = '') {
        if (!fileName) return `Episódio ${epNum}`;

        let clean = fileName
            // Remove o nome base do anime (se fornecido)
            .replace(new RegExp(animeName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi'), '')
            // Remove padrões de temporada/episódio comuns
            .replace(/S\d+E\d+/gi, '')
            .replace(new RegExp(`E0?${epNum}\\b`, 'gi'), '')
            .replace(new RegExp(`\\b0?${epNum}\\b`, 'g'), '')
            // Remove colchetes e parênteses com conteúdo
            .replace(/\[.*?\]/g, '')
            .replace(/\(.*?\)/g, '')
            // Remove tags técnicas extensivas
            .replace(/\b(?:1080p|720p|480p|2160p|4K|UHD|FHD|HD|SD)\b/gi, '')
            .replace(/\b(?:WEB-DL|WEBRip|BluRay|BD|BDRip|HDTV|HDRip|DVDRip|REMUX|REPACK|PROPER)\b/gi, '')
            .replace(/\b(?:x264|x265|H\.?264|H\.?265|HEVC|AVC|AV1|10bit|10-bit|8bit)\b/gi, '')
            .replace(/\b(?:AAC|AAC2\.0|FLAC|AC3|DTS|EAC3|OPUS|MP3|DDP2\.0|Atmos|5\.1|2\.0)\b/gi, '')
            .replace(/\b(?:CR|DUAL|MULTi|VARYG|AMZN|NF|DSNP|HULU|ATVP|SubsPlease|Erai-raws|ToonsHub|HorribleSubs|ASW)\b/gi, '')
            .replace(/\b(?:INTERNAL|PROPER|REPACK|v\d+)\b/gi, '')
            // Remove extensões
            .replace(/\.(?:mkv|mp4|avi|webm)$/gi, '')
            // Limpa hífens, pontos, underscores e espaços extras
            .replace(/[\-._]+/g, ' ')
            .replace(/\s+/g, ' ')
            .replace(/^\s*[-–—]\s*/, '')
            .replace(/\s*[-–—]\s*$/, '')
            .trim();

        // Se ficou vazio ou muito curto, retorna "Episódio X"
        return clean.length > 2 ? clean : `Episódio ${epNum}`;
    }
    
    // Extrai o melhor título de exibição para o card
    function getEpisodeDisplayTitle(ep) {
        // Se tem título do episódio específico do parser V2
        if (ep._groupedData?.titulo_episodio && ep._groupedData.titulo_episodio.length > 3) {
            return ep._groupedData.titulo_episodio;
        }
        
        // Se tem título limpo do anime
        if (ep.Title && ep.Title.length > 3) {
            return ep.Title;
        }
        
        // Fallback: limpa o nome do arquivo
        const fileName = ep._vpsTorrentFile?.short_name || ep._vpsTorrentFile?.name || '';
        const animeName = selectedAnime?.Title || '';
        return cleanEpisodeTitle(fileName, ep.Episode, animeName);
    }

    // ===== VPS PIPELINE: Processar episódio (Download -> Remux -> GoFile) =====
    async function processVpsPipeline(episode) {
        if (!episode._vpsTorrentFile) {
            alert('Erro: Informações do arquivo não encontradas');
            return;
        }
        
        const file = episode._vpsTorrentFile;
        const animeName = selectedAnime?.Title || 'Anime';
        const episodeNum = file.episode || episode.Number || 1;
        
        // Monta query no formato "Anime S01E01"
        const query = `${animeName} S01E${String(episodeNum).padStart(2, '0')}`;
        
        console.log('[VPS Pipeline] Iniciando processamento:', query);
        pipelineProcessing = true;
        pipelineStatus = `Processando: ${query}...`;
        
        try {
            // Chama o pipeline no VPS
            const result = await VPSStartPipeline(query, true, true);
            
            if (result && result.status === 'started') {
                pipelineStatus = `✅ Pipeline iniciado! Job: ${result.job_id}`;
                lastPipelineJob = result;
                
                // Mostra info para o usuário
                alert(`Pipeline iniciado com sucesso!\n\nJob ID: ${result.job_id}\n\nO arquivo será:\n1. Baixado do TorBox\n2. Convertido para MP4\n3. Enviado para GoFile\n4. Salvo no banco de dados\n\nAcompanhe os logs do servidor para o progresso.`);
            } else {
                pipelineStatus = `❌ Erro: ${result?.error || 'Falha desconhecida'}`;
                alert('Erro ao iniciar pipeline: ' + (result?.error || 'Falha desconhecida'));
            }
        } catch (err) {
            console.error('[VPS Pipeline] Erro:', err);
            pipelineStatus = `❌ Erro: ${err.message}`;
            alert('Erro ao processar: ' + err.message);
        } finally {
            pipelineProcessing = false;
        }
    }
    
    // ===== VPS: Verificar health na inicialização =====
    async function checkVpsHealth() {
        try {
            vpsHealthy = await VPSCheckHealth();
            console.log('[VPS Health]', vpsHealthy ? '✅ Online' : '❌ Offline');
        } catch (err) {
            vpsHealthy = false;
            console.log('[VPS Health] Erro:', err);
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

    $: filteredEpisodes = selectedSeason 
        ? episodes.filter(e => (e.Season || 1) === selectedSeason).sort((a, b) => a.Episode - b.Episode) 
        : episodes.slice().sort((a, b) => a.Episode - b.Episode);

</script>

<main>
    <!-- MODAL DE REPRODUÇÃO AVANÇADA -->
    {#if showPlayerModal && playerModalEpisode}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div class="player-modal-overlay" onclick={closePlayerModal}>
            <div class="player-modal" onclick={(e) => e.stopPropagation()}>
                <!-- Header do Modal -->
                <div class="modal-header">
                    <div class="modal-title-section">
                        <span class="modal-icon">🎬</span>
                        <div class="modal-title-info">
                            <h2>{selectedAnime?.Title || 'Anime'}</h2>
                            <p>{playerModalEpisode._vpsTorrentFile?.short_name || playerModalEpisode._vpsTorrentFile?.name || 'Episódio'}</p>
                        </div>
                    </div>
                    <button type="button" class="modal-close" onclick={closePlayerModal}>✕</button>
                </div>
                
                <!-- Informações de Qualidade -->
                {#if playerModalEpisode._vpsTorrentFile}
                {@const fileQuality = getFileQualityInfo(playerModalEpisode._vpsTorrentFile)}
                <div class="modal-quality-section">
                    <div class="quality-header">
                        <span class="quality-icon">📊</span>
                        <span>Qualidade do Arquivo</span>
                    </div>
                    <div class="quality-grid">
                        <div class="quality-item">
                            <span class="q-label">Resolução</span>
                            <span class="q-value highlight">{fileQuality?.resolution || '720p'}</span>
                        </div>
                        <div class="quality-item">
                            <span class="q-label">Codec</span>
                            <span class="q-value">{fileQuality?.codec || 'H.264'}</span>
                        </div>
                        <div class="quality-item">
                            <span class="q-label">Áudio</span>
                            <span class="q-value">{fileQuality?.audio || 'AAC'}</span>
                        </div>
                        <div class="quality-item">
                            <span class="q-label">Tamanho</span>
                            <span class="q-value">{fileQuality?.size || 'N/A'}</span>
                        </div>
                        {#if fileQuality?.dualAudio}
                            <div class="quality-item special">
                                <span class="q-value">🌐 Dual Audio</span>
                            </div>
                        {/if}
                        {#if fileQuality?.hdr}
                            <div class="quality-item special">
                                <span class="q-value">✨ HDR</span>
                            </div>
                        {/if}
                    </div>
                </div>
                {/if}
                
                <!-- Seletor de Modo de Upscaling -->
                <div class="modal-upscale-section">
                    <div class="upscale-header">
                        <span class="upscale-icon">🚀</span>
                        <span>Modo de Upscaling (AI)</span>
                    </div>
                    <div class="upscale-modes">
                        {#each upscaleModes as mode}
                            <button 
                                type="button"
                                class="upscale-mode-btn {selectedUpscaleMode === mode.id ? 'active' : ''}"
                                onclick={() => selectedUpscaleMode = mode.id}
                            >
                                <span class="mode-icon">{mode.icon}</span>
                                <span class="mode-name">{mode.name}</span>
                                <span class="mode-desc">{mode.desc}</span>
                                <span class="mode-gpu">GPU: {mode.gpu}</span>
                            </button>
                        {/each}
                    </div>
                </div>
                
                <!-- Ações Principais -->
                <div class="modal-actions">
                    <button type="button" class="action-btn primary" onclick={() => { console.log('[BUTTON CLICK] Assistir clicado!'); playWithSelectedMode(); }}>
                        <span class="action-icon">▶</span>
                        <span class="action-text">Assistir (MPV ou Interno)</span>
                    </button>
                    
                    <button type="button" class="action-btn secondary" onclick={() => { closePlayerModal(); playVpsTorrentFileInBrowser(playerModalEpisode); }}>
                        <span class="action-icon">🌐</span>
                        <span class="action-text">Player Interno (HTML5)</span>
                    </button>
                </div>
                
                <!-- Seção de Compartilhamento -->
                <div class="modal-share-section">
                    <div class="share-header" onclick={() => showShareOptions = !showShareOptions}>
                        <span class="share-icon">📤</span>
                        <span>Compartilhar com Amigo</span>
                        <span class="share-expand">{showShareOptions ? '▼' : '▶'}</span>
                    </div>
                    
                    {#if showShareOptions}
                        <div class="share-content">
                            {#if !shareLink}
                                <button type="button" class="generate-link-btn" onclick={generateShareLink} disabled={generatingShareLink}>
                                    {#if generatingShareLink}
                                        <span class="loading-spinner"></span>
                                        Gerando link...
                                    {:else}
                                        🔗 Gerar Link de Compartilhamento
                                    {/if}
                                </button>
                                <p class="share-hint">Seu amigo poderá assistir diretamente no GoAnime!</p>
                            {:else}
                                <div class="share-link-box">
                                    <input type="text" readonly value={shareLink} class="share-link-input" />
                                    <button type="button" class="copy-link-btn" onclick={copyShareLink}>
                                        📋 Copiar
                                    </button>
                                </div>
                                <div class="share-actions">
                                    <button type="button" class="share-action whatsapp" onclick={() => window.open(`https://wa.me/?text=${encodeURIComponent('Assista comigo no GoAnime! ' + shareLink)}`, '_blank')}>
                                        <span>📱</span> WhatsApp
                                    </button>
                                    <button type="button" class="share-action discord" onclick={() => { navigator.clipboard.writeText(shareLink); alert('Link copiado! Cole no Discord'); }}>
                                        <span>💬</span> Discord
                                    </button>
                                    <button type="button" class="share-action telegram" onclick={() => window.open(`https://t.me/share/url?url=${encodeURIComponent(shareLink)}&text=${encodeURIComponent('Assista comigo no GoAnime!')}`, '_blank')}>
                                        <span>✈️</span> Telegram
                                    </button>
                                </div>
                            {/if}
                        </div>
                    {/if}
                </div>
                
                <!-- Seção de Legendas -->
                <div class="modal-subtitle-section">
                    <div class="subtitle-header" onclick={() => showSubtitleOptions = !showSubtitleOptions}>
                        <span class="subtitle-icon">📝</span>
                        <span>Legendas</span>
                        <span class="subtitle-expand">{showSubtitleOptions ? '▼' : '▶'}</span>
                    </div>
                    
                    {#if showSubtitleOptions}
                        <div class="subtitle-content">
                            <!-- Legenda Externa (URL) -->
                            <div class="subtitle-url-section">
                                <label class="subtitle-label" for="subtitle-url-input">Legenda Externa (URL .srt/.ass/.vtt)</label>
                                <div class="subtitle-url-input-wrapper">
                                    <input 
                                        type="text" 
                                        id="subtitle-url-input"
                                        class="subtitle-url-input"
                                        bind:value={subtitleUrl}
                                        placeholder="https://exemplo.com/legenda.srt"
                                    />
                                    {#if subtitleUrl}
                                        <button type="button" class="subtitle-clear-btn" onclick={() => subtitleUrl = ''}>✕</button>
                                    {/if}
                                </div>
                            </div>
                            
                            <!-- Legendas Disponíveis -->
                            {#if availableSubtitles.length > 0}
                                <div class="subtitle-list-section">
                                    <span class="subtitle-label" role="heading" aria-level="4">Legendas Disponíveis</span>
                                    <div class="subtitle-list">
                                        {#each availableSubtitles as sub}
                                            <button 
                                                type="button"
                                                class="subtitle-item {selectedSubtitle === sub.url ? 'active' : ''}"
                                                onclick={() => { selectedSubtitle = sub.url; subtitleUrl = sub.url; }}
                                            >
                                                <span class="sub-lang">{sub.lang || '🇧🇷'}</span>
                                                <span class="sub-name">{sub.name || 'Legenda'}</span>
                                                <span class="sub-format">{sub.format || 'SRT'}</span>
                                            </button>
                                        {/each}
                                    </div>
                                </div>
                            {/if}
                            
                            <!-- Sem legenda -->
                            <button 
                                type="button"
                                class="subtitle-none-btn {!subtitleUrl ? 'active' : ''}"
                                onclick={() => { subtitleUrl = ''; selectedSubtitle = null; }}
                            >
                                🚫 Sem Legenda
                            </button>
                            
                            <p class="subtitle-hint">💡 Em breve: importar legendas automaticamente!</p>
                        </div>
                    {/if}
                </div>
                
                <!-- Opções Extras -->
                <div class="modal-extras">
                    {#if vpsHealthy}
                        <button 
                            type="button" 
                            class="extra-btn"
                            onclick={() => { closePlayerModal(); processVpsPipeline(playerModalEpisode); }}
                            disabled={pipelineProcessing}
                        >
                            <span>☁️</span>
                            <span>Enviar para GoFile</span>
                            <span class="extra-hint">Download permanente</span>
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {/if}

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
            animeCover={selectedAnime?.Image || selectedAnime?.CoverImage || null}
            skipTimes={currentSkipTimes}
            onClose={closePlayer}
            onNext={selectNextEpisode}
            onPrevious={selectPreviousEpisode}
        />
    {:else if !usuario}
        <!-- LOGIN SCREEN - MODERN WITH TABS -->
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
                
                <!-- Auth Card -->
                <div class="login-card-modern">
                    <!-- Auth Tabs -->
                    <div class="auth-tabs">
                        <button 
                            type="button"
                            class="auth-tab {authMode === 'login' ? 'active' : ''}"
                            onclick={() => { authMode = 'login'; authError = ''; }}
                        >
                            🔑 Entrar
                        </button>
                        <button 
                            type="button"
                            class="auth-tab {authMode === 'register' ? 'active' : ''}"
                            onclick={() => { authMode = 'register'; authError = ''; }}
                        >
                            ✨ Registrar
                        </button>
                    </div>
                    
                    {#if authError}
                        <div class="auth-error">
                            <span>⚠️</span> {authError}
                        </div>
                    {/if}
                    
                    <div class="card-body">
                        {#if authMode === 'login'}
                            <!-- LOGIN FORM -->
                            <div class="input-group">
                                <label for="login-user">Usuário</label>
                                <div class="input-wrapper">
                                    <span class="input-icon">👤</span>
                                    <input 
                                        id="login-user"
                                        type="text"
                                        bind:value={loginUsername} 
                                        placeholder="Digite seu usuário" 
                                        class="input-modern"
                                        disabled={authLoading}
                                    />
                                </div>
                            </div>
                            
                            <div class="input-group">
                                <label for="login-pass">Senha</label>
                                <div class="input-wrapper">
                                    <span class="input-icon">🔒</span>
                                    <input 
                                        id="login-pass"
                                        type="password"
                                        bind:value={loginPassword} 
                                        placeholder="Digite sua senha" 
                                        class="input-modern"
                                        disabled={authLoading}
                                        onkeydown={(e) => e.key === 'Enter' && handleLogin()}
                                    />
                                </div>
                            </div>
                            
                            <div class="card-footer">
                                <button type="button" class="btn-enter" onclick={handleLogin} disabled={authLoading || !loginUsername || !loginPassword}>
                                    {#if authLoading}
                                        <span class="loading-spinner"></span>
                                    {:else}
                                        <span>Entrar</span>
                                        <span class="btn-arrow">→</span>
                                    {/if}
                                </button>
                            </div>
                            
                        {:else if authMode === 'register'}
                            <!-- REGISTER FORM -->
                            <div class="input-group">
                                <label for="reg-user">Usuário *</label>
                                <div class="input-wrapper">
                                    <span class="input-icon">👤</span>
                                    <input 
                                        id="reg-user"
                                        type="text"
                                        bind:value={registerUsername} 
                                        placeholder="Escolha um nome de usuário" 
                                        class="input-modern"
                                        disabled={authLoading}
                                    />
                                </div>
                            </div>
                            
                            <div class="input-group">
                                <label for="reg-email">Email (opcional)</label>
                                <div class="input-wrapper">
                                    <span class="input-icon">📧</span>
                                    <input 
                                        id="reg-email"
                                        type="email"
                                        bind:value={registerEmail} 
                                        placeholder="seu@email.com" 
                                        class="input-modern"
                                        disabled={authLoading}
                                    />
                                </div>
                            </div>
                            
                            <div class="input-group">
                                <label for="reg-pass">Senha *</label>
                                <div class="input-wrapper">
                                    <span class="input-icon">🔒</span>
                                    <input 
                                        id="reg-pass"
                                        type="password"
                                        bind:value={registerPassword} 
                                        placeholder="Mínimo 6 caracteres" 
                                        class="input-modern"
                                        disabled={authLoading}
                                    />
                                </div>
                            </div>
                            
                            <div class="input-group">
                                <label for="reg-pass2">Confirmar Senha *</label>
                                <div class="input-wrapper">
                                    <span class="input-icon">🔒</span>
                                    <input 
                                        id="reg-pass2"
                                        type="password"
                                        bind:value={registerConfirmPassword} 
                                        placeholder="Digite novamente" 
                                        class="input-modern"
                                        disabled={authLoading}
                                        onkeydown={(e) => e.key === 'Enter' && handleRegister()}
                                    />
                                </div>
                            </div>
                            
                            <!-- Avatar Selection -->
                            <div class="avatar-group">
                                <span class="avatar-label">Escolha seu avatar</span>
                                <div class="avatar-grid">
                                    {#each [
                                        { id: 'avatar1.png', emoji: '👤' },
                                        { id: 'avatar2.png', emoji: '🦊' },
                                        { id: 'avatar3.png', emoji: '🤖' },
                                        { id: 'avatar4.png', emoji: '🐱' },
                                        { id: 'avatar5.png', emoji: '🎮' },
                                        { id: 'avatar6.png', emoji: '⚡' }
                                    ] as avatar}
                                        <button 
                                            type="button"
                                            class="avatar-option {avatarSelecionado === avatar.id ? 'selected' : ''}"
                                            onclick={() => avatarSelecionado = avatar.id}
                                            disabled={authLoading}
                                        >
                                            <span class="avatar-emoji">{avatar.emoji}</span>
                                            {#if avatarSelecionado === avatar.id}
                                                <span class="avatar-check">✓</span>
                                            {/if}
                                        </button>
                                    {/each}
                                </div>
                            </div>
                            
                            <div class="card-footer">
                                <button type="button" class="btn-enter" onclick={handleRegister} disabled={authLoading || !registerUsername || !registerPassword}>
                                    {#if authLoading}
                                        <span class="loading-spinner"></span>
                                    {:else}
                                        <span>Criar Conta</span>
                                        <span class="btn-arrow">→</span>
                                    {/if}
                                </button>
                            </div>
                        {/if}
                    </div>
                    
                    <!-- Guest Option -->
                    <div class="guest-section">
                        <div class="divider">
                            <span>ou</span>
                        </div>
                        <button type="button" class="btn-guest" onclick={handleGuestLogin} disabled={authLoading}>
                            <span>👁️ Entrar como Visitante</span>
                        </button>
                        <p class="guest-info">
                            Como visitante você pode assistir, mas não poderá salvar favoritos ou sincronizar dados.
                        </p>
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
                        <span class="logo-icon-small">{activeTab === 'manga' ? '📚' : '🎬'}</span>
                        <span class="logo-text-small">
                            <span class="go">Go</span><span class="anime">{activeTab === 'manga' ? 'Manga' : 'Anime'}</span>
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
                        <span class="user-avatar">{isGuest ? '👁️' : '👤'}</span>
                        <span class="user-name">
                            {usuario.username}
                            {#if isGuest}
                                <span class="guest-badge">Visitante</span>
                            {/if}
                        </span>
                        <span class="menu-arrow">{userMenuOpen ? '▲' : '▼'}</span>
                    </button>
                    
                    {#if userMenuOpen}
                        <div class="user-dropdown">
                            {#if !isGuest}
                                <button type="button" class="dropdown-item" onclick={() => openView('favorites')}>
                                    ⭐ Favoritos
                                </button>
                                <button type="button" class="dropdown-item" onclick={() => openView('history')}>
                                    🕐 Últimos Assistidos
                                </button>
                                <button type="button" class="dropdown-item" onclick={() => openView('settings')}>
                                    ⚙️ Configurações
                                </button>
                                <div class="dropdown-divider"></div>
                            {:else}
                                <button type="button" class="dropdown-item guest-upgrade" onclick={() => { userMenuOpen = false; handleLogout(); }}>
                                    ✨ Criar Conta
                                </button>
                                <div class="dropdown-divider"></div>
                            {/if}
                            <button type="button" class="dropdown-item logout" onclick={() => { userMenuOpen = false; handleLogout(); }}>
                                🚪 Sair
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
                        <h2 class="settings-title">
                            <span class="title-icon">⚙️</span>
                            Configurações
                        </h2>
                        
                        <!-- PREFERÊNCIAS -->
                        <div class="settings-card">
                            <div class="card-header">
                                <span class="card-icon">🎨</span>
                                <h3>Preferências</h3>
                            </div>
                            <div class="card-content">
                                <label class="setting-toggle">
                                    <input type="checkbox" bind:checked={settings.start_fullscreen} />
                                    <span class="toggle-slider"></span>
                                    <span class="toggle-label">Iniciar em tela cheia</span>
                                </label>
                                
                                <div class="setting-select">
                                    <label for="content-language-select">Conteúdo preferido:</label>
                                    <select id="content-language-select" bind:value={settings.content_language}>
                                        <option value="all">🌐 Todos (BR + EN)</option>
                                        <option value="br">🇧🇷 Apenas Português</option>
                                        <option value="en">🇺🇸 Apenas Inglês</option>
                                    </select>
                                </div>
                                
                                <label class="setting-toggle">
                                    <input type="checkbox" bind:checked={settings.use_anime4k} />
                                    <span class="toggle-slider"></span>
                                    <span class="toggle-label">Usar Anime4K (upscaling)</span>
                                </label>
                            </div>
                            <div class="card-footer">
                                <button type="button" class="btn-save" onclick={saveUserSettings}>
                                    <span class="btn-icon">💾</span>
                                    Salvar Configurações
                                </button>
                            </div>
                        </div>
                        
                        <!-- SEMEAMENTO / CONTRIBUIÇÃO -->
                        <div class="settings-card seeding-card">
                            <div class="card-header">
                                <span class="card-icon">🌱</span>
                                <h3>Semeamento Comunitário</h3>
                                <span class="badge-beta">BETA</span>
                            </div>
                            <div class="card-content">
                                <div class="seeding-description">
                                    <p>Ajude a comunidade! Quando ativado, seu computador irá:</p>
                                    <ul>
                                        <li>📥 Baixar animes populares em segundo plano</li>
                                        <li>🔄 Fazer encode para streaming otimizado</li>
                                        <li>☁️ Enviar para o GoFile automaticamente</li>
                                        <li>⚡ Tornar streams mais rápidos para todos</li>
                                    </ul>
                                </div>
                                
                                <label class="setting-toggle main-toggle">
                                    <input type="checkbox" bind:checked={settings.seeding_enabled} />
                                    <span class="toggle-slider seeding"></span>
                                    <span class="toggle-label">
                                        {settings.seeding_enabled ? '🟢 Semeamento Ativo' : '⚪ Semeamento Desativado'}
                                    </span>
                                </label>
                                
                                {#if settings.seeding_enabled}
                                    <div class="seeding-options" class:active={settings.seeding_enabled}>
                                        <div class="seeding-option">
                                            <label for="seeding-cpu-limit">
                                                <span class="option-icon">🖥️</span>
                                                Limite de CPU:
                                            </label>
                                            <div class="slider-container">
                                                <input type="range" id="seeding-cpu-limit" min="10" max="80" bind:value={settings.seeding_max_cpu} />
                                                <span class="slider-value">{settings.seeding_max_cpu}%</span>
                                            </div>
                                        </div>
                                        
                                        <div class="seeding-option">
                                            <label for="seeding-bandwidth-limit">
                                                <span class="option-icon">📶</span>
                                                Limite de Banda:
                                            </label>
                                            <div class="slider-container">
                                                <input type="range" id="seeding-bandwidth-limit" min="1" max="50" bind:value={settings.seeding_max_bandwidth} />
                                                <span class="slider-value">{settings.seeding_max_bandwidth} MB/s</span>
                                            </div>
                                        </div>
                                        
                                        <div class="seeding-option">
                                            <label for="seeding-schedule-select">
                                                <span class="option-icon">⏰</span>
                                                Quando semear:
                                            </label>
                                            <select id="seeding-schedule-select" bind:value={settings.seeding_schedule}>
                                                <option value="always">🔄 Sempre</option>
                                                <option value="idle">💤 Apenas quando PC ocioso</option>
                                                <option value="night">🌙 Apenas à noite (00h-06h)</option>
                                            </select>
                                        </div>
                                        
                                        <label class="setting-toggle">
                                            <input type="checkbox" bind:checked={settings.seeding_only_wifi} />
                                            <span class="toggle-slider"></span>
                                            <span class="toggle-label">Apenas em WiFi</span>
                                        </label>
                                    </div>
                                    
                                    <div class="seeding-stats">
                                        <div class="stat-item">
                                            <span class="stat-icon">📊</span>
                                            <div class="stat-info">
                                                <span class="stat-label">Total Contribuído</span>
                                                <span class="stat-value">{formatBytes(seedingStats?.totalBytesUploaded || settings.seeding_contributed || 0)}</span>
                                            </div>
                                        </div>
                                        <div class="stat-item">
                                            <span class="stat-icon">🎬</span>
                                            <div class="stat-info">
                                                <span class="stat-label">Episódios Processados</span>
                                                <span class="stat-value">{seedingStats?.jobsCompleted || 0}</span>
                                            </div>
                                        </div>
                                        <div class="stat-item">
                                            <span class="stat-icon">⚡</span>
                                            <div class="stat-info">
                                                <span class="stat-label">Status</span>
                                                <span class="stat-value status {seedingRunning ? 'active' : ''}">
                                                    {#if seedingRunning && seedingStats?.currentJob}
                                                        🔄 {seedingStats.currentJob}
                                                    {:else if seedingRunning}
                                                        ⏳ Aguardando job...
                                                    {:else}
                                                        ⏸️ Parado
                                                    {/if}
                                                </span>
                                            </div>
                                        </div>
                                        {#if seedingStats?.errors > 0}
                                            <div class="stat-item error">
                                                <span class="stat-icon">⚠️</span>
                                                <div class="stat-info">
                                                    <span class="stat-label">Erros</span>
                                                    <span class="stat-value">{seedingStats.errors}</span>
                                                </div>
                                            </div>
                                        {/if}
                                    </div>
                                {/if}
                            </div>
                        </div>
                        
                        <!-- BACKUP & RESTAURAÇÃO -->
                        <div class="settings-card">
                            <div class="card-header">
                                <span class="card-icon">💾</span>
                                <h3>Backup & Restauração</h3>
                            </div>
                            <div class="card-content">
                                <p class="card-description">Exporte seus dados para fazer backup ou importe para restaurar.</p>
                                <div class="backup-buttons">
                                    <button type="button" class="btn-action export" onclick={exportData}>
                                        <span class="btn-icon">📤</span>
                                        Exportar Dados
                                    </button>
                                    <button type="button" class="btn-action import" onclick={() => showImportExport = true}>
                                        <span class="btn-icon">📥</span>
                                        Importar Dados
                                    </button>
                                </div>
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
                        
                        <!-- STATUS DAS FONTES DE VÍDEO -->
                        <div class="settings-card sources-card">
                            <div class="card-header">
                                <span class="card-icon">📡</span>
                                <h3>Status das Fontes de Vídeo</h3>
                            </div>
                            <div class="card-content">
                                <p class="card-description">Mostra o estado atual das fontes de streaming e cache inteligente.</p>
                                
                                <button type="button" class="btn-action refresh" onclick={loadCacheStats}>
                                    <span class="btn-icon">🔄</span>
                                    Atualizar Status
                                </button>
                                
                                {#if cacheStats}
                                    <div class="cache-overview">
                                        <div class="cache-stat-card">
                                            <span class="cache-icon">📦</span>
                                            <span class="cache-label">Streams em Cache</span>
                                            <span class="cache-value">{cacheStats.totalStreams}</span>
                                        </div>
                                        <div class="cache-stat-card">
                                            <span class="cache-icon">💾</span>
                                            <span class="cache-label">Total em Cache</span>
                                            <span class="cache-value">{cacheStats.totalCache}</span>
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
                                        <button type="button" class="btn-action warning" onclick={resetSources}>
                                            <span class="btn-icon">🔄</span>
                                            Resetar Falhas
                                        </button>
                                        <button type="button" class="btn-action danger" onclick={clearAllCacheAction}>
                                            <span class="btn-icon">🗑️</span>
                                            Limpar Cache
                                        </button>
                                    </div>
                                {:else}
                                    <p class="no-stats">Clique em "Atualizar Status" para ver o estado das fontes.</p>
                                {/if}
                            </div>
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
                                    {#each filteredEpisodes as ep, index (`${ep.Episode || ep.Number || index}-${index}`)}
                                        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
                                        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
                                        <div 
                                            class="episode-card {selectedEpisodeURL === (ep._isGrouped ? `grouped:${ep.Episode}` : (ep._vpsTorrentFile ? `vps_torrent:${ep._vpsTorrentFile.id}` : ep.URL)) ? 'selected' : ''} {ep._vpsTorrentFile || ep._isGrouped ? 'torbox-file' : ''} {ep._isGrouped && ep._versoes?.length > 1 ? 'has-versions' : ''}"
                                            role="listitem"
                                            onclick={() => {
                                                if (ep._isGrouped) {
                                                    selectedEpisodeURL = `grouped:${ep.Episode}`;
                                                    selectedGroupedEpisode = ep;
                                                    // Seleciona primeira versão por padrão
                                                    selectedVersion = ep._versoes?.[0] || null;
                                                } else if (ep._vpsTorrentFile) {
                                                    selectedEpisodeURL = `vps_torrent:${ep._vpsTorrentFile.id}`;
                                                } else {
                                                    selectedEpisodeURL = ep.URL;
                                                }
                                            }}
                                            onkeydown={(e) => {
                                                if (e.key === 'Enter') {
                                                    if (ep._isGrouped) {
                                                        selectedEpisodeURL = `grouped:${ep.Episode}`;
                                                        selectedGroupedEpisode = ep;
                                                        selectedVersion = ep._versoes?.[0] || null;
                                                    } else if (ep._vpsTorrentFile) {
                                                        selectedEpisodeURL = `vps_torrent:${ep._vpsTorrentFile.id}`;
                                                    } else {
                                                        selectedEpisodeURL = ep.URL;
                                                    }
                                                }
                                            }}
                                            tabindex="0"
                                        >
                                            <div class="episode-number">
                                                {#if ep._isGrouped}
                                                    EP {ep.Episode}
                                                    {#if ep._versoes?.length > 1}
                                                        <span class="version-count" title="{ep._versoes.length} versões disponíveis">
                                                            +{ep._versoes.length - 1}
                                                        </span>
                                                    {/if}
                                                {:else if ep._vpsTorrentFile}
                                                    {ep.Episode > 0 ? `EP ${ep.Episode}` : `#${index + 1}`}
                                                {:else}
                                                    EP {ep.Number || ep.Episode || index + 1}
                                                {/if}
                                            </div>
                                            <div class="episode-title">
                                                {#if ep._isGrouped}
                                                    <span class="title-main">{getEpisodeDisplayTitle(ep)}</span>
                                                    {#if ep._groupedData?.titulo_episodio && ep._groupedData.titulo_episodio !== getEpisodeDisplayTitle(ep)}
                                                        <span class="ep-subtitle">{ep._groupedData.titulo_episodio}</span>
                                                    {/if}
                                                {:else if ep._vpsTorrentFile}
                                                    <span class="title-main">{cleanEpisodeTitle(ep._vpsTorrentFile.short_name || ep._vpsTorrentFile.name, ep.Episode, selectedAnime?.Title)}</span>
                                                {:else}
                                                    <span class="title-main">{ep.Title || `Episódio ${ep.Number || ep.Episode}`}</span>
                                                {/if}
                                            </div>
                                            
                                            {#if ep._isGrouped && ep._versoes?.length > 0}
                                                <!-- Badges de qualidade do episódio agrupado -->
                                                <div class="version-badges">
                                                    {#each [...new Set(ep._versoes.map(v => v.qualidade))] as quality}
                                                        <span class="q-badge small">{quality}</span>
                                                    {/each}
                                                </div>
                                            {:else if ep._vpsTorrentFile}
                                                <div class="episode-size">{ep._vpsTorrentFile.size_str}</div>
                                            {/if}
                                            
                                            {#if ep.Source}
                                                <div class="episode-source">{ep.Source}</div>
                                            {/if}
                                            
                                            {#if selectedEpisodeURL === (ep._isGrouped ? `grouped:${ep.Episode}` : (ep._vpsTorrentFile ? `vps_torrent:${ep._vpsTorrentFile.id}` : ep.URL))}
                                                <div class="episode-actions">
                                                    {#if ep._isGrouped}
                                                        <!-- ===== EPISÓDIO AGRUPADO: Seletor de versões ===== -->
                                                        {#if ep._versoes?.length > 1}
                                                            <div class="version-selector">
                                                                <span class="version-label">Escolha a versão:</span>
                                                                <div class="version-list">
                                                                    {#each ep._versoes as version, vIdx}
                                                                        <button 
                                                                            type="button"
                                                                            class="version-option {selectedVersion === version ? 'selected' : ''}"
                                                                            onclick={(e) => { e.stopPropagation(); selectedVersion = version; }}
                                                                        >
                                                                            <span class="v-quality">{version.qualidade}</span>
                                                                            <span class="v-size">{version.tamanho}</span>
                                                                            {#if version.subgrupo}
                                                                                <span class="v-group">[{version.subgrupo}]</span>
                                                                            {/if}
                                                                            {#if version.tags?.includes('REPACK')}
                                                                                <span class="v-tag repack">REPACK</span>
                                                                            {/if}
                                                                        </button>
                                                                    {/each}
                                                                </div>
                                                            </div>
                                                        {/if}
                                                        
                                                        <!-- Badges da versão selecionada -->
                                                        {#if selectedVersion}
                                                            {@const qualityInfo = getFileQualityInfo(selectedVersion._vpsTorrentFile)}
                                                            <div class="quality-badges">
                                                                <span class="q-badge resolution">{qualityInfo?.resolution || selectedVersion.qualidade}</span>
                                                                <span class="q-badge codec">{qualityInfo?.codec || 'H.264'}</span>
                                                                {#if qualityInfo?.dualAudio}
                                                                    <span class="q-badge dual">🌐 Dual</span>
                                                                {/if}
                                                            </div>
                                                            
                                                            <!-- Botão de ação -->
                                                            <div class="torbox-action-buttons">
                                                                <button type="button" class="btn-torbox-primary" onclick={(e) => { 
                                                                    e.stopPropagation(); 
                                                                    // Cria episódio temporário com o arquivo selecionado
                                                                    const tempEp = { ...ep, _vpsTorrentFile: selectedVersion._vpsTorrentFile };
                                                                    openPlayerModal(tempEp); 
                                                                }}>
                                                                    <span class="btn-icon">🎬</span>
                                                                    <span class="btn-text">Assistir</span>
                                                                </button>
                                                            </div>
                                                        {/if}
                                                        
                                                    {:else if ep._vpsTorrentFile}
                                                        <!-- Ações AVANÇADAS para VPS TorBox -->
                                                        {@const qualityInfo = getFileQualityInfo(ep._vpsTorrentFile)}
                                                        
                                                        <!-- Badges de qualidade -->
                                                        <div class="quality-badges">
                                                            <span class="q-badge resolution">{qualityInfo?.resolution || '720p'}</span>
                                                            <span class="q-badge codec">{qualityInfo?.codec || 'H.264'}</span>
                                                            {#if qualityInfo?.dualAudio}
                                                                <span class="q-badge dual">🌐 Dual</span>
                                                            {/if}
                                                            {#if qualityInfo?.hdr}
                                                                <span class="q-badge hdr">HDR</span>
                                                            {/if}
                                                        </div>
                                                        
                                                        <!-- Botão único de ação -->
                                                        <div class="torbox-action-buttons">
                                                            <button type="button" class="btn-torbox-primary" onclick={(e) => { e.stopPropagation(); openPlayerModal(ep); }}>
                                                                <span class="btn-icon">🎬</span>
                                                                <span class="btn-text">Assistir</span>
                                                            </button>
                                                        </div>
                                                    {:else}
                                                        <!-- Ações normais -->
                                                        <button type="button" class="btn-play-mpv primary" onclick={(e) => { e.stopPropagation(); playEpisode(); }} title="Recomendado - Funciona com todas as fontes">
                                                            🖥️ MPV (Recomendado)
                                                        </button>
                                                        <button type="button" class="btn-play-web" onclick={(e) => { e.stopPropagation(); playEpisodeInBrowser(); }} title="Pode não funcionar com algumas fontes">
                                                            ▶ Navegador
                                                        </button>
                                                        <Player4KButton 
                                                            videoUrl={originalStreamUrl || ''}
                                                            animeUrl={selectedAnime?.URL || ''}
                                                            episodeUrl={selectedEpisodeURL || ''}
                                                            isAnime={true}
                                                        />
                                                    {/if}
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
                        <!-- FEATURED HERO with ROTATION (only for anime tab) -->
                        {#if activeTab === 'anime' && featuredAnime && featuredAnime.banner}
                            {#key featuredAnime.id || featuredAnime.title}
                            <div 
                                class="featured-hero-ultra" 
                                style="--banner-url: url({featuredAnime.banner}); --accent-color: {featuredAnime.color || '#f5576c'}"
                            >
                                <!-- Animated Background Layers -->
                                <div class="hero-bg-layer bg-image"></div>
                                <div class="hero-bg-layer bg-blur"></div>
                                <div class="hero-bg-layer bg-gradient"></div>
                                <div class="hero-bg-layer bg-noise"></div>
                                <div class="hero-bg-layer bg-vignette"></div>
                                
                                <!-- Animated Particles -->
                                <div class="hero-particles">
                                    <div class="particle p1"></div>
                                    <div class="particle p2"></div>
                                    <div class="particle p3"></div>
                                    <div class="particle p4"></div>
                                    <div class="particle p5"></div>
                                </div>
                                
                                <!-- Glowing Lines -->
                                <div class="hero-glow-lines">
                                    <div class="glow-line gl1"></div>
                                    <div class="glow-line gl2"></div>
                                    <div class="glow-line gl3"></div>
                                </div>
                                
                                <!-- Main Content -->
                                <div class="hero-main-content">
                                    <!-- Left Side - Info -->
                                    <div class="hero-info-side">
                                        <!-- Animated Badges -->
                                        <div class="hero-badges-ultra">
                                            {#if featuredAnime.score}
                                                <div class="ultra-badge score-badge">
                                                    <span class="badge-icon">⭐</span>
                                                    <span class="badge-value">{featuredAnime.score}%</span>
                                                    <div class="badge-glow"></div>
                                                </div>
                                            {/if}
                                            {#if featuredAnime.episodes}
                                                <div class="ultra-badge eps-badge">
                                                    <span class="badge-icon">📺</span>
                                                    <span class="badge-value">{featuredAnime.episodes} eps</span>
                                                </div>
                                            {/if}
                                            {#if featuredAnime.isAiring}
                                                <div class="ultra-badge live-badge">
                                                    <span class="live-pulse"></span>
                                                    <span class="badge-text">AO VIVO</span>
                                                </div>
                                            {/if}
                                        </div>
                                        
                                        <!-- Title with Reveal Animation -->
                                        <h1 class="hero-title-ultra">
                                            <span class="title-text">{featuredAnime.title}</span>
                                            <span class="title-underline"></span>
                                        </h1>
                                        
                                        <!-- Meta Info with Icons -->
                                        <div class="hero-meta-ultra">
                                            {#each (featuredAnime.genres?.slice(0, 3) || []) as genre, i}
                                                <span class="meta-genre">{genre}</span>
                                                {#if i < 2}<span class="meta-dot">•</span>{/if}
                                            {/each}
                                            {#if featuredAnime.studio}
                                                <span class="meta-dot">•</span>
                                                <span class="meta-studio">{featuredAnime.studio}</span>
                                            {/if}
                                            {#if featuredAnime.year}
                                                <span class="meta-dot">•</span>
                                                <span class="meta-year">{featuredAnime.year}</span>
                                            {/if}
                                        </div>
                                        
                                        <!-- Description with Fade -->
                                        {#if featuredAnime.description}
                                            <p class="hero-desc-ultra">
                                                {featuredAnime.description?.replace(/<[^>]*>/g, '').slice(0, 200)}{featuredAnime.description?.length > 200 ? '...' : ''}
                                            </p>
                                        {/if}
                                        
                                        <!-- Action Buttons Ultra -->
                                        <div class="hero-actions-ultra">
                                            <button type="button" class="btn-ultra-play" onclick={() => {
                                                termoBusca = featuredAnime.title;
                                                pesquisar();
                                            }}>
                                                <span class="btn-bg"></span>
                                                <span class="btn-content">
                                                    <svg class="play-icon" viewBox="0 0 24 24" fill="currentColor">
                                                        <path d="M8 5v14l11-7z"/>
                                                    </svg>
                                                    <span>Assistir</span>
                                                </span>
                                                <span class="btn-shine"></span>
                                            </button>
                                            {#if featuredAnime.trailerUrl}
                                                <a href={featuredAnime.trailerUrl} target="_blank" class="btn-ultra-trailer">
                                                    <span class="btn-border"></span>
                                                    <span class="btn-content">
                                                        <span class="trailer-icon">🎬</span>
                                                        <span>Trailer</span>
                                                    </span>
                                                </a>
                                            {/if}
                                        </div>
                                    </div>
                                    
                                    <!-- Right Side - Poster 3D -->
                                    <div class="hero-poster-side">
                                        <div class="poster-3d-container">
                                            <div class="poster-reflection"></div>
                                            <div class="poster-card">
                                                <img src={featuredAnime.image} alt={featuredAnime.title} loading="eager" />
                                                <div class="poster-shine"></div>
                                                <div class="poster-border"></div>
                                            </div>
                                            <div class="poster-shadow"></div>
                                        </div>
                                    </div>
                                </div>
                                
                                <!-- Ultra Navigation Dots -->
                                <div class="hero-nav-ultra">
                                    <div class="nav-line"></div>
                                    {#each trendingAnimes.slice(0, 8) as anime, i}
                                        {#if anime.banner}
                                            <button 
                                                type="button"
                                                class="nav-dot-ultra {i === featuredIndex ? 'active' : ''}"
                                                onclick={() => selectFeatured(i)}
                                                title={anime.title}
                                            >
                                                <span class="dot-inner"></span>
                                                <span class="dot-ring"></span>
                                            </button>
                                        {/if}
                                    {/each}
                                    <div class="nav-line"></div>
                                </div>
                            </div>
                            {/key}
                        {:else if activeTab === 'anime' && carregando}
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
                                        <span class="hero-emoji">{activeTab === 'manga' ? '📚' : '🎬'}</span>
                                        <h1 class="hero-brand">
                                            <span class="brand-go">Go</span><span class="brand-anime">{activeTab === 'manga' ? 'Manga' : 'Anime'}</span>
                                        </h1>
                                    </div>
                                    <p class="hero-tagline">{activeTab === 'manga' ? 'Leia seus mangás favoritos online' : 'Assista seus animes favoritos em HD'}</p>
                                    <div class="hero-stats">
                                        <div class="stat">
                                            <span class="stat-number">{activeTab === 'manga' ? '147+' : '10K+'}</span>
                                            <span class="stat-label">{activeTab === 'manga' ? 'Mangás' : 'Animes'}</span>
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
                                <!-- UNIFIED SEARCH PANEL -->
                                <div class="search-panel">
                                    <!-- SOURCE TOGGLE (Fonte) -->
                                    <div class="source-toggle-row">
                                        <div class="source-toggle">
                                            <span class="toggle-icon">📡</span>
                                            <span class="toggle-label">Fonte:</span>
                                            <div class="toggle-buttons">
                                                <button 
                                                    class="toggle-btn {selectedAnimeSource === 'all' ? 'active' : ''}"
                                                    onclick={() => changeAnimeSource('all')}
                                                >
                                                    🌐 Todas
                                                </button>
                                                {#each animeSources as source}
                                                    <button 
                                                        class="toggle-btn {selectedAnimeSource === source.id ? 'active' : ''} {source.id === 'vps' ? 'vps' : ''}"
                                                        onclick={() => changeAnimeSource(source.id)}
                                                        title={source.description}
                                                    >
                                                        {#if source.id === 'vps'}🇺🇸{:else if source.language === 'pt'}🇧🇷{:else if source.language === 'multi'}🌍{:else}🇺🇸{/if} {source.name}
                                                    </button>
                                                {/each}
                                            </div>
                                        </div>
                                    </div>
                                    
                                    <!-- MAIN SEARCH BAR -->
                                    <div class="main-search-bar">
                                        <div class="search-input-container">
                                            <svg class="search-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                <circle cx="11" cy="11" r="8"></circle>
                                                <path d="m21 21-4.35-4.35"></path>
                                            </svg>
                                            <input 
                                                type="text"
                                                bind:value={termoBusca}
                                                placeholder="Buscar anime..."
                                                class="search-input-modern"
                                                oninput={() => scheduleSearch(350)}
                                                onkeydown={e => e.key === 'Enter' && pesquisar()}
                                            />
                                            {#if termoBusca}
                                                <button class="clear-search" aria-label="Limpar busca" onclick={() => { termoBusca = ''; clearGenreFilter(); }}>
                                                    <svg viewBox="0 0 24 24" fill="currentColor"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>
                                                </button>
                                            {/if}
                                        </div>
                                        <button type="button" class="search-submit" onclick={pesquisar} disabled={carregando}>
                                            {#if carregando}
                                                <span class="loading-spinner"></span>
                                            {:else}
                                                Buscar
                                            {/if}
                                        </button>
                                    </div>
                                    
                                    {#if resultadosBusca.length === 0 && !selectedGenre}
                                        <!-- POPULAR QUICK SEARCH -->
                                        <div class="quick-search-row">
                                            <span class="row-label">Popular:</span>
                                            <div class="quick-tags">
                                                <button type="button" class="quick-tag" onclick={() => { termoBusca = 'Frieren'; pesquisar(); }}>Frieren</button>
                                                <button type="button" class="quick-tag" onclick={() => { termoBusca = 'Jujutsu Kaisen'; pesquisar(); }}>Jujutsu</button>
                                                <button type="button" class="quick-tag" onclick={() => { termoBusca = 'One Piece'; pesquisar(); }}>One Piece</button>
                                                <button type="button" class="quick-tag" onclick={() => { termoBusca = 'Solo Leveling'; pesquisar(); }}>Solo Leveling</button>
                                            </div>
                                        </div>
                                        
                                        <!-- GENRE GRID -->
                                        <div class="genre-section">
                                            <span class="row-label">Gêneros:</span>
                                            <div class="genre-grid">
                                                {#each animeGenres as genre}
                                                    <button 
                                                        type="button" 
                                                        class="genre-btn"
                                                        onclick={() => searchByGenre(genre)}
                                                        title={genre.name}
                                                    >
                                                        <span class="genre-icon">{genre.icon}</span>
                                                        <span class="genre-name">{genre.name}</span>
                                                    </button>
                                                {/each}
                                            </div>
                                        </div>
                                    {:else if resultadosBusca.length > 0}
                                        <div class="results-info">
                                            {#if selectedGenre}
                                                <span class="active-filter">
                                                    <span class="filter-badge">{selectedGenre.icon} {selectedGenre.name}</span>
                                                    <span class="filter-count">{resultadosBusca.length} resultados</span>
                                                </span>
                                            {:else}
                                                <span class="search-results-text">{resultadosBusca.length} resultados para "{termoBusca}"</span>
                                            {/if}
                                            <button type="button" class="clear-filter-btn" onclick={clearGenreFilter}>
                                                <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>
                                                Limpar
                                            </button>
                                        </div>
                                    {/if}
                                </div>

                            {#if resultadosBusca.length > 0}
                                <!-- SEARCH RESULTS - Agrupado por Série quando VPS -->
                                {#if selectedAnimeSource === 'vps' && Object.keys(groupedResults).length > 0}
                                    <!-- MODO AGRUPADO POR SÉRIE -->
                                    <div class="grouped-results">
                                        {#each Object.entries(groupedResults) as [groupName, group], gIdx}
                                            <div class="series-group" class:expanded={expandedGroups[groupName]}>
                                                <!-- Header do Grupo (Série) -->
                                                <button type="button" class="group-header" onclick={() => toggleGroup(groupName)}>
                                                    <div class="group-poster">
                                                        {#if group.image}
                                                            <img src={group.image} alt={groupName} />
                                                        {:else}
                                                            <div class="no-image">🎬</div>
                                                        {/if}
                                                        {#if group.hasRedeTorrent}
                                                            <span class="br-badge">🇧🇷</span>
                                                        {/if}
                                                    </div>
                                                    <div class="group-info">
                                                        <h3 class="group-title">{groupName}</h3>
                                                        <div class="group-meta">
                                                            <span class="group-count">{group.totalItems} arquivos</span>
                                                            <span class="group-seasons">{Object.keys(group.seasons).length} temporada{Object.keys(group.seasons).length > 1 ? 's' : ''}</span>
                                                            {#if group.hasRedeTorrent}
                                                                <span class="source-tag br">🇧🇷 Rede Torrent</span>
                                                            {/if}
                                                        </div>
                                                    </div>
                                                    <span class="expand-icon">{expandedGroups[groupName] ? '▼' : '▶'}</span>
                                                </button>
                                                
                                                <!-- Conteúdo Expandido -->
                                                {#if expandedGroups[groupName]}
                                                    <div class="group-content">
                                                        {#each Object.entries(group.seasons).sort((a, b) => {
                                                            // Coloca "Todas Temporadas" (0) no topo
                                                            if (a[0] === '0') return -1;
                                                            if (b[0] === '0') return 1;
                                                            return parseInt(a[0]) - parseInt(b[0]);
                                                        }) as [seasonNum, season]}
                                                            <div class="season-section" class:all-seasons={season.isAll}>
                                                                <h4 class="season-title" class:highlight-all={season.isAll}>
                                                                    <span class="season-icon">{season.isAll ? '📦' : '📺'}</span>
                                                                    {season.label}
                                                                    {#if season.isAll}
                                                                        <span class="all-badge">COMPLETO</span>
                                                                    {/if}
                                                                </h4>
                                                                
                                                                <!-- Rede Torrent (Prioridade) -->
                                                                {#if season.sources.redeTorrent.length > 0}
                                                                    <div class="source-section rede-torrent">
                                                                        <div class="source-header">
                                                                            <span class="source-badge br">🇧🇷 Rede Torrent</span>
                                                                            <span class="source-count">{season.sources.redeTorrent.length}</span>
                                                                        </div>
                                                                        <div class="torrent-list">
                                                                            {#each season.sources.redeTorrent as anime}
                                                                                <button 
                                                                                    type="button" 
                                                                                    class="torrent-item br-highlight"
                                                                                    onclick={() => openEpisodeSelection(anime)}
                                                                                >
                                                                                    <div class="torrent-info">
                                                                                        <span class="torrent-title">{anime.Title}</span>
                                                                                        <div class="torrent-meta">
                                                                                            <span class="size">{anime.Size || 'N/A'}</span>
                                                                                            {#if anime.Seeds > 0}
                                                                                                <span class="seeds">🌱 {anime.Seeds}</span>
                                                                                            {/if}
                                                                                            {#if anime._vps_torrent?.cached}
                                                                                                <span class="cached">⚡ Cache</span>
                                                                                            {/if}
                                                                                        </div>
                                                                                    </div>
                                                                                    <span class="play-btn">▶</span>
                                                                                </button>
                                                                            {/each}
                                                                        </div>
                                                                    </div>
                                                                {/if}
                                                                
                                                                <!-- Nyaa -->
                                                                {#if season.sources.nyaa.length > 0}
                                                                    <div class="source-section nyaa">
                                                                        <div class="source-header">
                                                                            <span class="source-badge en">🇺🇸 Nyaa</span>
                                                                            <span class="source-count">{season.sources.nyaa.length}</span>
                                                                        </div>
                                                                        <div class="torrent-list">
                                                                            {#each season.sources.nyaa as anime}
                                                                                <button 
                                                                                    type="button" 
                                                                                    class="torrent-item"
                                                                                    onclick={() => openEpisodeSelection(anime)}
                                                                                >
                                                                                    <div class="torrent-info">
                                                                                        <span class="torrent-title">{anime.Title}</span>
                                                                                        <div class="torrent-meta">
                                                                                            <span class="size">{anime.Size || 'N/A'}</span>
                                                                                            {#if anime.Seeds > 0}
                                                                                                <span class="seeds">🌱 {anime.Seeds}</span>
                                                                                            {/if}
                                                                                            {#if anime._vps_torrent?.cached}
                                                                                                <span class="cached">⚡ Cache</span>
                                                                                            {/if}
                                                                                        </div>
                                                                                    </div>
                                                                                    <span class="play-btn">▶</span>
                                                                                </button>
                                                                            {/each}
                                                                        </div>
                                                                    </div>
                                                                {/if}
                                                                
                                                                <!-- Outras fontes -->
                                                                {#if season.sources.other.length > 0}
                                                                    <div class="source-section other">
                                                                        <div class="source-header">
                                                                            <span class="source-badge other">📦 Outros</span>
                                                                            <span class="source-count">{season.sources.other.length}</span>
                                                                        </div>
                                                                        <div class="torrent-list">
                                                                            {#each season.sources.other as anime}
                                                                                <button 
                                                                                    type="button" 
                                                                                    class="torrent-item"
                                                                                    onclick={() => openEpisodeSelection(anime)}
                                                                                >
                                                                                    <div class="torrent-info">
                                                                                        <span class="torrent-title">{anime.Title}</span>
                                                                                        <div class="torrent-meta">
                                                                                            <span class="size">{anime.Size || 'N/A'}</span>
                                                                                            {#if anime.Seeds > 0}
                                                                                                <span class="seeds">🌱 {anime.Seeds}</span>
                                                                                            {/if}
                                                                                        </div>
                                                                                    </div>
                                                                                    <span class="play-btn">▶</span>
                                                                                </button>
                                                                            {/each}
                                                                        </div>
                                                                    </div>
                                                                {/if}
                                                            </div>
                                                        {/each}
                                                    </div>
                                                {/if}
                                            </div>
                                        {/each}
                                    </div>
                                {:else}
                                    <!-- MODO GRID NORMAL (não VPS ou fallback) -->
                                    <div class="anime-grid large">
                                        {#each resultadosBusca as anime, idx (anime._vps_torrent?.hash || anime.URL || idx)}
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
                                                        <div class="no-image">{anime._vps_torrent ? '🧲' : '📺'}</div>
                                                    {/if}
                                                    {#if anime._vps_torrent}
                                                        <!-- VPS TorBox badges -->
                                                        <div class="torbox-badges">
                                                            {#if anime.VariantCount > 0}
                                                                <span class="torbox-badge variants">📦 +{anime.VariantCount}</span>
                                                            {/if}
                                                            {#if anime._vps_torrent.cached}
                                                                <span class="torbox-badge cached">⚡ Cache</span>
                                                            {/if}
                                                            {#if anime.Seeds > 0}
                                                                <span class="torbox-badge seeds">🌱 {anime.Seeds}</span>
                                                            {/if}
                                                        </div>
                                                    {:else if anime.Sources && anime.Sources.length > 0}
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
                                                    <div class="card-title">{anime.CleanTitle || anime.Title}</div>
                                                    {#if anime._vps_torrent}
                                                        <div class="card-source torbox">🧲 VPS TorBox • {anime.Size || ''}</div>
                                                    {:else if anime.Source}
                                                        <div class="card-source">{anime.Source}</div>
                                                    {/if}
                                                </div>
                                            </button>
                                        {/each}
                                    </div>
                                {/if}
                            {:else if selectedAnimeSource !== 'all'}
                                <!-- VPS: MOSTRA ANIMES POPULARES DIRETAMENTE -->
                                {#if selectedAnimeSource === 'vps' && trendingAnimes.length > 0}
                                    <div class="content-section popular-section-modern">
                                        <div class="section-header-modern">
                                            <div class="section-title-area">
                                                <span class="fire-icon-animated">🔥</span>
                                                <h2>Animes Populares</h2>
                                            </div>
                                            <span class="source-badge-modern vps">
                                                <span class="badge-dot"></span>
                                                VPS TorBox
                                            </span>
                                        </div>
                                        <div class="anime-grid-modern">
                                            {#each trendingAnimes.slice(0, 18) as anime, i}
                                                <button 
                                                    type="button" 
                                                    class="anime-card-modern" 
                                                    style="--delay: {i * 0.05}s"
                                                    onclick={() => { termoBusca = anime.title; pesquisar(); }}
                                                >
                                                    <div class="card-image-wrapper">
                                                        <img src={anime.image} alt={anime.title} loading="lazy" />
                                                        <div class="card-shine"></div>
                                                        <div class="card-gradient-overlay"></div>
                                                        
                                                        <!-- Score Badge -->
                                                        <div class="score-badge-modern">
                                                            <span class="score-star">★</span>
                                                            <span class="score-value">{anime.score}</span>
                                                        </div>
                                                        
                                                        <!-- Hover Overlay -->
                                                        <div class="card-hover-overlay">
                                                            <div class="play-button-modern">
                                                                <svg viewBox="0 0 24 24" fill="currentColor">
                                                                    <path d="M8 5v14l11-7z"/>
                                                                </svg>
                                                            </div>
                                                            <span class="watch-text">Assistir</span>
                                                        </div>
                                                    </div>
                                                    
                                                    <div class="card-content-modern">
                                                        <h3 class="card-title-modern">{anime.title}</h3>
                                                        <div class="card-meta-modern">
                                                            {#if anime.episodes}
                                                                <span class="meta-item">
                                                                    <span class="meta-icon">📺</span>
                                                                    {anime.episodes} eps
                                                                </span>
                                                            {/if}
                                                        </div>
                                                    </div>
                                                    
                                                    <!-- Glow Effect -->
                                                    <div class="card-glow"></div>
                                                </button>
                                            {/each}
                                        </div>
                                    </div>
                                {:else}
                                    <!-- MENSAGEM PARA OUTRAS FONTES -->
                                    <div class="source-instruction">
                                        <div class="instruction-icon">
                                            {#if selectedAnimeSource === 'enime'}🌍{:else if selectedAnimeSource === 'vps'}🌐{:else}📡{/if}
                                        </div>
                                        <h3 class="instruction-title">
                                            Fonte {animeSources.find(s => s.id === selectedAnimeSource)?.name || selectedAnimeSource} Selecionada
                                        </h3>
                                        <p class="instruction-text">
                                            Use a barra de busca acima para encontrar animes nesta fonte.
                                        </p>
                                    </div>
                                {/if}
                            {:else}
                            <!-- TRENDING SECTION (AniList HD) -->
                            {#if trendingAnimes.length > 0}
                                <div class="content-section popular-section-modern">
                                    <div class="section-header-modern">
                                        <div class="section-title-area">
                                            <span class="fire-icon-animated">🔥</span>
                                            <h2>Em Alta Agora</h2>
                                        </div>
                                        <span class="source-badge-modern anilist">
                                            <span class="badge-dot"></span>
                                            AniList HD
                                        </span>
                                    </div>
                                    <div class="anime-grid-modern">
                                        {#each trendingAnimes.slice(0, 14) as anime, i}
                                            <button 
                                                type="button" 
                                                class="anime-card-modern" 
                                                style="--delay: {i * 0.05}s"
                                                onclick={() => { termoBusca = anime.title; pesquisar(); }}
                                            >
                                                <div class="card-image-wrapper">
                                                    <img src={anime.image} alt={anime.title} loading="lazy" />
                                                    <div class="card-shine"></div>
                                                    <div class="card-gradient-overlay"></div>
                                                    
                                                    <!-- Badges -->
                                                    <div class="card-badges-modern">
                                                        {#if anime.isAiring}
                                                            <span class="status-badge airing">
                                                                <span class="live-dot"></span>
                                                                AO VIVO
                                                            </span>
                                                        {/if}
                                                    </div>
                                                    
                                                    <!-- Score Badge -->
                                                    <div class="score-badge-modern">
                                                        <span class="score-star">★</span>
                                                        <span class="score-value">{anime.score}</span>
                                                    </div>
                                                    
                                                    <!-- Hover Overlay -->
                                                    <div class="card-hover-overlay">
                                                        <div class="play-button-modern">
                                                            <svg viewBox="0 0 24 24" fill="currentColor">
                                                                <path d="M8 5v14l11-7z"/>
                                                            </svg>
                                                        </div>
                                                        <span class="watch-text">Assistir</span>
                                                    </div>
                                                </div>
                                                
                                                <div class="card-content-modern">
                                                    <h3 class="card-title-modern">{anime.title}</h3>
                                                    <div class="card-meta-modern">
                                                        <span class="meta-item">
                                                            {anime.episodes ? `${anime.episodes} eps` : 'Em exibição'}
                                                        </span>
                                                        {#if anime.studio}
                                                            <span class="meta-item studio">• {anime.studio}</span>
                                                        {/if}
                                                    </div>
                                                </div>
                                                
                                                <!-- Glow Effect -->
                                                <div class="card-glow"></div>
                                            </button>
                                        {/each}
                                    </div>
                                </div>
                            {/if}
                            
                            <!-- POPULAR SECTION (Streaming Sources) -->
                            <div class="content-section popular-section-modern">
                                <div class="section-header-modern">
                                    <div class="section-title-area">
                                        <span class="fire-icon-animated">📺</span>
                                        <h2>Disponíveis para Assistir</h2>
                                    </div>
                                    <span class="source-badge-modern sources">
                                        <span class="badge-dot"></span>
                                        AllAnime + AnimeFire
                                    </span>
                                </div>
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
                                    <div class="anime-grid-modern">
                                        {#each topAnimes as anime, i (anime.Title)}
                                            <button 
                                                type="button" 
                                                class="anime-card-modern" 
                                                style="--delay: {i * 0.03}s"
                                                onclick={() => openEpisodeSelection(anime)}
                                                onmouseenter={() => schedulePrefetch(anime)}
                                            >
                                                <div class="card-image-wrapper">
                                                    {#if anime.Image}
                                                        <img src={anime.Image} alt={anime.Title} loading="lazy" />
                                                    {:else}
                                                        <div class="no-image-modern">📺</div>
                                                    {/if}
                                                    <div class="card-shine"></div>
                                                    <div class="card-gradient-overlay"></div>
                                                    
                                                    <!-- Source Badge -->
                                                    {#if anime.Source}
                                                        <div class="source-flag-badge {anime.Source === 'AllAnime' ? 'en' : 'pt'}">
                                                            {anime.Source === 'AllAnime' ? '🇺🇸' : '🇧🇷'}
                                                        </div>
                                                    {/if}
                                                    
                                                    <!-- Hover Overlay -->
                                                    <div class="card-hover-overlay">
                                                        <div class="play-button-modern">
                                                            <svg viewBox="0 0 24 24" fill="currentColor">
                                                                <path d="M8 5v14l11-7z"/>
                                                            </svg>
                                                        </div>
                                                        <span class="watch-text">Assistir</span>
                                                    </div>
                                                </div>
                                                
                                                <div class="card-content-modern">
                                                    <h3 class="card-title-modern">{anime.Title}</h3>
                                                    {#if anime.Source}
                                                        <div class="card-meta-modern">
                                                            <span class="meta-item source-name">{anime.Source}</span>
                                                        </div>
                                                    {/if}
                                                </div>
                                                
                                                <!-- Glow Effect -->
                                                <div class="card-glow"></div>
                                            </button>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                            {/if}
                            
                            <!-- MANGA TAB -->
                            {:else if activeTab === 'manga'}
                                <!-- MANGA READER (fullscreen) -->
                                {#if selectedChapter}
                                    <div class="manga-reader-fullscreen">
                                        <header class="reader-header">
                                            <button class="btn-back" onclick={closeMangaReader}>
                                                ← Voltar
                                            </button>
                                            <div class="chapter-info">
                                                <span class="manga-name">{selectedManga?.title || 'Mangá'}</span>
                                                <span class="chapter-number">Capítulo {selectedChapter.number || selectedChapter.title}</span>
                                            </div>
                                            <div class="reader-nav">
                                                <button class="btn-nav" onclick={prevChapter} title="Capítulo anterior">◀ Anterior</button>
                                                <span class="page-count">{chapterPages.length} páginas</span>
                                                <button class="btn-nav" onclick={nextChapter} title="Próximo capítulo">Próximo ▶</button>
                                            </div>
                                        </header>
                                        
                                        <div class="reader-content">
                                            {#if loadingChapterPages}
                                                <div class="loading-pages">
                                                    <div class="spinner"></div>
                                                    <p>Carregando páginas...</p>
                                                </div>
                                            {:else if chapterPages.length === 0}
                                                <div class="no-pages">
                                                    <p>Nenhuma página encontrada neste capítulo.</p>
                                                    <button onclick={() => selectChapter(selectedChapter)}>Tentar novamente</button>
                                                </div>
                                            {:else}
                                                <div class="pages-scroll">
                                                    {#each chapterPages as page, i}
                                                        <div class="page-wrapper">
                                                            <img 
                                                                src={page.url} 
                                                                alt="Página {page.number || i + 1}"
                                                                loading={i < 5 ? "eager" : "lazy"}
                                                                decoding="async"
                                                                fetchpriority={i < 3 ? "high" : "low"}
                                                                class="manga-page"
                                                                onload={(e) => { const img = /** @type {HTMLImageElement} */ (e.target); img.classList.add('loaded'); }}
                                                                onerror={(e) => { const img = /** @type {HTMLImageElement} */ (e.target); img.classList.add('error'); }}
                                                            />
                                                            <div class="page-number">{i + 1} / {chapterPages.length}</div>
                                                        </div>
                                                    {/each}
                                                    <div class="chapter-end">
                                                        <p>Fim do capítulo</p>
                                                        <button class="btn-next-chapter" onclick={nextChapter}>
                                                            Próximo Capítulo ▶
                                                        </button>
                                                    </div>
                                                </div>
                                            {/if}
                                        </div>
                                    </div>
                                
                                <!-- MANGA DETAILS -->
                                {:else if selectedManga}
                                    <div class="manga-details-view">
                                        <button class="btn-back-manga" onclick={closeMangaDetails}>
                                            ← Voltar aos Mangás
                                        </button>
                                        
                                        <!-- Hero Section with Background -->
                                        <div class="manga-hero-wrapper">
                                            <div class="manga-hero-bg" style="background-image: url('{selectedManga.image}')"></div>
                                            <div class="manga-hero-content">
                                                <div class="manga-cover-container">
                                                    <img 
                                                        src={selectedManga.image} 
                                                        alt={selectedManga.title}
                                                        class="manga-cover-large"
                                                    />
                                                    <div class="manga-cover-shadow"></div>
                                                </div>
                                                <div class="manga-info-details">
                                                    <h1>{selectedManga.title}</h1>
                                                    <div class="manga-meta-row">
                                                        {#if selectedManga.status}
                                                            <span class="manga-status-badge {selectedManga.status === 'Em Andamento' ? 'ongoing' : 'completed'}">
                                                                {selectedManga.status === 'Em Andamento' ? '🔥' : '✅'} {selectedManga.status}
                                                            </span>
                                                        {/if}
                                                        <span class="manga-chapters-count">📚 {mangaChapters.length} capítulos</span>
                                                    </div>
                                                    {#if selectedManga.author}
                                                        <p class="manga-author">✍️ {selectedManga.author}</p>
                                                    {/if}
                                                    {#if selectedManga.genres?.length}
                                                        <div class="manga-genres-list">
                                                            {#each selectedManga.genres as genre}
                                                                <span class="genre-tag">{genre}</span>
                                                            {/each}
                                                        </div>
                                                    {/if}
                                                    {#if selectedManga.description}
                                                        <p class="manga-description">{selectedManga.description}</p>
                                                    {/if}
                                                    {#if mangaChapters.length > 0}
                                                        <button class="btn-start-reading" onclick={() => selectChapter(mangaChapters[0])}>
                                                            📖 Começar a Ler
                                                        </button>
                                                    {/if}
                                                </div>
                                            </div>
                                        </div>
                                        
                                        <div class="chapters-section">
                                            <h2>📖 Capítulos</h2>
                                            {#if loadingMangas}
                                                <div class="loading-chapters">
                                                    <div class="spinner"></div>
                                                    <p>Carregando capítulos...</p>
                                                </div>
                                            {:else if mangaChapters.length === 0}
                                                <p class="no-chapters">Nenhum capítulo encontrado.</p>
                                            {:else}
                                                <div class="chapters-list">
                                                    {#each mangaChapters as chapter}
                                                        <button 
                                                            class="chapter-item"
                                                            onclick={() => selectChapter(chapter)}
                                                        >
                                                            <span class="chapter-num">Cap. {chapter.number || '?'}</span>
                                                            <span class="chapter-title">{chapter.title}</span>
                                                            {#if chapter.date}
                                                                <span class="chapter-date">{chapter.date}</span>
                                                            {/if}
                                                        </button>
                                                    {/each}
                                                </div>
                                            {/if}
                                        </div>
                                    </div>
                                
                                <!-- MANGA BROWSE -->
                                {:else}
                                    <div class="tab-content manga-tab">
                                        <!-- Source Selector -->
                                        <div class="manga-source-selector">
                                            <span class="source-label">📡 Fonte:</span>
                                            <div class="source-buttons">
                                                <button 
                                                    class="source-btn {selectedMangaSource === 'all' ? 'active' : ''}"
                                                    onclick={() => changeMangaSource('all')}
                                                >
                                                    🌐 Todas
                                                </button>
                                                {#each mangaSources as source}
                                                    <button 
                                                        class="source-btn {selectedMangaSource === source.id ? 'active' : ''}"
                                                        onclick={() => changeMangaSource(source.id)}
                                                        title={source.description}
                                                    >
                                                        {source.name}
                                                    </button>
                                                {/each}
                                                <button 
                                                    class="source-btn manage-btn"
                                                    onclick={openMangaSourcesModal}
                                                    title="Gerenciar fontes de mangá"
                                                >
                                                    ⚙️ Gerenciar
                                                </button>
                                            </div>
                                        </div>
                                        
                                        <!-- Search Bar -->
                                        <div class="manga-search-bar">
                                            <input 
                                                type="text" 
                                                placeholder="🔍 Buscar mangás..."
                                                bind:value={mangaSearchTerm}
                                                onkeydown={(e) => e.key === 'Enter' && searchManga()}
                                            />
                                            <button onclick={searchManga}>Buscar</button>
                                        </div>
                                        
                                        <!-- Genre Filter -->
                                        <div class="manga-genres-filter">
                                            {#each mangaGenres as genre}
                                                <button 
                                                    class="genre-btn {selectedMangaGenre?.id === genre.id ? 'active' : ''}"
                                                    onclick={() => selectedMangaGenre?.id === genre.id ? clearMangaGenre() : loadMangasByGenre(genre)}
                                                >
                                                    {genre.icon} {genre.name}
                                                </button>
                                            {/each}
                                        </div>
                                        
                                        {#if loadingMangas && featuredMangas.length === 0}
                                            <!-- Skeleton Loading -->
                                            <div class="manga-section">
                                                <h2>🌟 Carregando...</h2>
                                                <div class="skeleton-grid">
                                                    {#each Array(12) as _, i}
                                                        <div class="skeleton-card">
                                                            <div class="skeleton-poster"></div>
                                                            <div class="skeleton-title"></div>
                                                            <div class="skeleton-subtitle"></div>
                                                        </div>
                                                    {/each}
                                                </div>
                                            </div>
                                        {:else if mangaSearchResults.length > 0 || selectedMangaGenre}
                                            <!-- Search/Genre Results -->
                                            <div class="manga-section">
                                                <h2>
                                                    {#if selectedMangaGenre}
                                                        {selectedMangaGenre.icon} {selectedMangaGenre.name}
                                                    {:else}
                                                        🔍 Resultados da Busca
                                                    {/if}
                                                </h2>
                                                <div class="manga-grid">
                                                    {#each mangaSearchResults as manga}
                                                        <button class="manga-card" onclick={() => selectManga(manga)}>
                                                            <div class="manga-poster">
                                                                {#if manga.image}
                                                                    <img src={manga.image} alt={manga.title} loading="lazy" />
                                                                {:else}
                                                                    <div class="no-image">📚</div>
                                                                {/if}
                                                                <div class="manga-overlay">
                                                                    <span class="read-icon">📖</span>
                                                                </div>
                                                            </div>
                                                            <div class="manga-card-info">
                                                                <div class="manga-title">{manga.title}</div>
                                                                {#if manga.latestChapter}
                                                                    <div class="manga-latest">📖 {manga.latestChapter}</div>
                                                                {/if}
                                                            </div>
                                                        </button>
                                                    {/each}
                                                </div>
                                            </div>
                                        {:else}
                                            <!-- SEÇÃO EM DESTAQUE -->
                                            {#if featuredMangas.length > 0}
                                                <div class="manga-section featured-section">
                                                    <h2>🌟 Em Destaque {#if selectedMangaSource !== 'all'}<span class="source-info">({selectedMangaSource})</span>{/if}</h2>
                                                    <div class="manga-grid">
                                                        {#each featuredMangas as manga}
                                                            <button class="manga-card" onclick={() => selectManga(manga)}>
                                                                <div class="manga-poster">
                                                                    {#if manga.image}
                                                                        <img src={manga.image} alt={manga.title} loading="lazy" />
                                                                    {:else}
                                                                        <div class="no-image">📚</div>
                                                                    {/if}
                                                                    <div class="manga-overlay">
                                                                        <span class="read-icon">📖</span>
                                                                    </div>
                                                                    {#if manga.source && selectedMangaSource === 'all'}
                                                                        <span class="source-badge">{manga.source === 'mangalivre.blog' ? '.blog' : '.to'}</span>
                                                                    {/if}
                                                                </div>
                                                                <div class="manga-card-info">
                                                                    <div class="manga-title">{manga.title}</div>
                                                                    {#if manga.latestChapter}
                                                                        <div class="manga-latest">📖 {manga.latestChapter}</div>
                                                                    {/if}
                                                                </div>
                                                            </button>
                                                        {/each}
                                                    </div>
                                                    
                                                    <!-- Botão Ver Todos -->
                                                    <div class="manga-section-footer">
                                                        <button class="btn-view-all" onclick={loadAllMangasSafe}>
                                                            📚 Ver Todos os Mangás
                                                        </button>
                                                    </div>
                                                </div>
                                            {/if}
                                            
                                            <!-- TODOS OS MANGÁS (quando carregados) -->
                                            {#if allMangas.length > 0}
                                                <div class="manga-section all-mangas-section">
                                                    <h2>📚 Todos os Mangás ({allMangas.length} títulos)</h2>
                                                    <div class="manga-grid">
                                                        {#each allMangas as manga}
                                                            <button class="manga-card" onclick={() => selectManga(manga)}>
                                                                <div class="manga-poster">
                                                                    {#if manga.image}
                                                                        <img src={manga.image} alt={manga.title} loading="lazy" />
                                                                    {:else}
                                                                        <div class="no-image">📚</div>
                                                                    {/if}
                                                                    <div class="manga-overlay">
                                                                        <span class="read-icon">📖</span>
                                                                    </div>
                                                                </div>
                                                                <div class="manga-card-info">
                                                                    <div class="manga-title">{manga.title}</div>
                                                                    {#if manga.latestChapter}
                                                                        <div class="manga-latest">📖 {manga.latestChapter}</div>
                                                                    {/if}
                                                                </div>
                                                            </button>
                                                        {/each}
                                                    </div>
                                                </div>
                                            {/if}
                                            
                                            <!-- SEÇÃO +18 (Toggle) -->
                                            <div class="manga-section adult-section">
                                                <div class="adult-section-header">
                                                    <h2>🔞 Conteúdo +18</h2>
                                                    <button 
                                                        class="btn-toggle-adult {showAdultContent ? 'active' : ''}"
                                                        onclick={toggleAdultContent}
                                                    >
                                                        {showAdultContent ? '🔓 Esconder' : '🔒 Mostrar'}
                                                    </button>
                                                </div>
                                                
                                                {#if showAdultContent}
                                                    {#if loadingMangas && adultMangas.length === 0}
                                                        <div class="loading-mangas">
                                                            <div class="spinner"></div>
                                                            <p>Carregando conteúdo adulto...</p>
                                                        </div>
                                                    {:else if adultMangas.length > 0}
                                                        <div class="adult-warning">
                                                            ⚠️ Este conteúdo é destinado apenas para maiores de 18 anos.
                                                        </div>
                                                        <div class="manga-grid">
                                                            {#each adultMangas as manga}
                                                                <button class="manga-card adult-card" onclick={() => selectManga(manga)}>
                                                                    <div class="manga-poster">
                                                                        {#if manga.image}
                                                                            <img src={manga.image} alt={manga.title} loading="lazy" />
                                                                        {:else}
                                                                            <div class="no-image">🔞</div>
                                                                        {/if}
                                                                        <div class="manga-overlay adult-overlay">
                                                                            <span class="read-icon">📖</span>
                                                                        </div>
                                                                        <span class="adult-badge">+18</span>
                                                                    </div>
                                                                    <div class="manga-card-info">
                                                                        <div class="manga-title">{manga.title}</div>
                                                                        {#if manga.latestChapter}
                                                                            <div class="manga-latest">📖 {manga.latestChapter}</div>
                                                                        {/if}
                                                                    </div>
                                                                </button>
                                                            {/each}
                                                        </div>
                                                    {:else}
                                                        <p class="no-adult-content">Nenhum conteúdo adulto encontrado.</p>
                                                    {/if}
                                                {:else}
                                                    <p class="adult-hidden-msg">
                                                        Clique no botão acima para exibir conteúdo para maiores de 18 anos.
                                                    </p>
                                                {/if}
                                            </div>
                                            
                                            <!-- Empty State -->
                                            {#if featuredMangas.length === 0}
                                                <div class="manga-empty-state">
                                                    <div class="empty-icon">📚</div>
                                                    <h3>Carregando todos os mangás...</h3>
                                                    <p>Buscando todos os mangás do MangaLivre, isso pode levar alguns segundos.</p>
                                                    <button class="btn-reload" onclick={loadMangaData}>
                                                        🔄 Carregar Mangás
                                                    </button>
                                                </div>
                                            {/if}
                                        {/if}
                                    </div>
                                {/if}
                            
                            <!-- FRIENDS TAB (Sistema Social) -->
                            {:else if activeTab === 'friends'}
                                <div class="tab-content friends-tab">
                                    {#if !hasProfile}
                                        <!-- Sem perfil - Mostrar opção de criar -->
                                        <div class="social-connect-container">
                                            <div class="social-connect-card">
                                                <div class="social-glow"></div>
                                                <div class="social-content">
                                                    <div class="social-logo-animated">
                                                        <div class="logo-ring"></div>
                                                        <div class="logo-ring ring-2"></div>
                                                        <svg viewBox="0 0 24 24" width="64" height="64" class="social-svg">
                                                            <path fill="#fff" d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
                                                        </svg>
                                                    </div>
                                                    
                                                    <h2 class="social-title">Conecte com Amigos</h2>
                                                    <p class="social-subtitle">Crie seu perfil e compartilhe o que você está assistindo!</p>
                                                    
                                                    <div class="social-features">
                                                        <div class="feature-item">
                                                            <span class="feature-icon">📺</span>
                                                            <div class="feature-text">
                                                                <strong>Status Ao Vivo</strong>
                                                                <span>Amigos veem o que você está assistindo</span>
                                                            </div>
                                                        </div>
                                                        <div class="feature-item">
                                                            <span class="feature-icon">🔗</span>
                                                            <div class="feature-text">
                                                                <strong>Código Único</strong>
                                                                <span>Compartilhe seu código para adicionar amigos</span>
                                                            </div>
                                                        </div>
                                                        <div class="feature-item">
                                                            <span class="feature-icon">👥</span>
                                                            <div class="feature-text">
                                                                <strong>Lista de Amigos</strong>
                                                                <span>Veja o que seus amigos estão assistindo</span>
                                                            </div>
                                                        </div>
                                                    </div>
                                                    
                                                    <div class="create-profile-section">
                                                        <p class="create-hint">Escolha um nome de usuário para começar:</p>
                                                        <div class="create-form">
                                                            <input 
                                                                type="text" 
                                                                bind:value={newUsername}
                                                                placeholder="Seu nome de usuário"
                                                                class="username-input"
                                                                maxlength="20"
                                                            />
                                                            <button 
                                                                type="button" 
                                                                class="btn-create-profile"
                                                                onclick={createProfile}
                                                                disabled={socialLoading || !newUsername.trim()}
                                                            >
                                                                {#if socialLoading}
                                                                    <span class="spinner-small"></span>
                                                                {:else}
                                                                    ✨ Criar Perfil
                                                                {/if}
                                                            </button>
                                                        </div>
                                                        {#if socialError}
                                                            <p class="social-error">{socialError}</p>
                                                        {/if}
                                                    </div>
                                                    
                                                    <p class="social-privacy">🔒 Sem login necessário - apenas um nome!</p>
                                                </div>
                                            </div>
                                        </div>
                                    {:else}
                                        <!-- Com perfil - Layout responsivo em grid -->
                                        <div class="friends-dashboard">
                                            <!-- Coluna Esquerda: Perfil e Adicionar -->
                                            <div class="friends-sidebar">
                                                <!-- Card do Perfil Compacto -->
                                                <div class="profile-card-modern">
                                                    <div class="profile-card-header">
                                                        <div class="profile-avatar-modern">
                                                            <span class="avatar-letter">{socialProfile?.username?.charAt(0)?.toUpperCase() || '?'}</span>
                                                            <span class="avatar-status {socialConnected ? 'online' : 'offline'}"></span>
                                                        </div>
                                                        <div class="profile-info-modern">
                                                            <h3 class="profile-username">{socialProfile?.username || 'Usuário'}</h3>
                                                            <span class="profile-status-text">{socialConnected ? '🟢 Online' : '🔴 Offline'}</span>
                                                        </div>
                                                    </div>
                                                    
                                                    <div class="profile-code-section">
                                                        <span class="code-label">Seu código de amizade</span>
                                                        <div class="code-display">
                                                            <code class="share-code-text">{socialProfile?.shareCode || '...'}</code>
                                                            <div class="code-actions">
                                                                <button type="button" class="btn-icon" onclick={copyShareCode} title="Copiar">
                                                                    <svg viewBox="0 0 24 24" width="18" height="18">
                                                                        <path fill="currentColor" d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
                                                                    </svg>
                                                                </button>
                                                                <button type="button" class="btn-icon" onclick={regenerateCode} title="Novo código">
                                                                    <svg viewBox="0 0 24 24" width="18" height="18">
                                                                        <path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0 0 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0 1 12 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
                                                                    </svg>
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </div>
                                                    
                                                    <div class="profile-quick-settings">
                                                        <div class="quick-setting">
                                                            <span class="setting-icon">📺</span>
                                                            <span class="setting-label">Mostrar atividade</span>
                                                            <button type="button" class="mini-toggle {socialProfile?.showStatus ? 'active' : ''}" aria-label="Alternar mostrar atividade" onclick={toggleSocialShowStatus}>
                                                                <span class="toggle-dot"></span>
                                                            </button>
                                                        </div>
                                                        <div class="quick-setting">
                                                            <span class="setting-icon">📋</span>
                                                            <span class="setting-label">Compartilhar lista</span>
                                                            <button type="button" class="mini-toggle {socialProfile?.shareAnimes ? 'active' : ''}" aria-label="Alternar compartilhar lista" onclick={toggleSocialShareAnimes}>
                                                                <span class="toggle-dot"></span>
                                                            </button>
                                                        </div>
                                                    </div>
                                                    
                                                    <button type="button" class="btn-delete-profile" onclick={deleteSocialProfile}>
                                                        <svg viewBox="0 0 24 24" width="14" height="14">
                                                            <path fill="currentColor" d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                                                        </svg>
                                                        Excluir Perfil
                                                    </button>
                                                </div>
                                                
                                                <!-- Adicionar Amigo -->
                                                <div class="add-friend-card">
                                                    <h4 class="card-title">
                                                        <svg viewBox="0 0 24 24" width="20" height="20">
                                                            <path fill="currentColor" d="M15 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm-9-2V7H4v3H1v2h3v3h2v-3h3v-2H6zm9 4c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
                                                        </svg>
                                                        Adicionar Amigo
                                                    </h4>
                                                    <div class="add-friend-input-group">
                                                        <input 
                                                            type="text" 
                                                            bind:value={friendCode}
                                                            placeholder="Código do amigo"
                                                            class="friend-input"
                                                            maxlength="8"
                                                        />
                                                        <button 
                                                            type="button" 
                                                            class="btn-add"
                                                            onclick={addFriend}
                                                            disabled={socialLoading || !friendCode.trim()}
                                                        >
                                                            {#if socialLoading}
                                                                <span class="spinner-mini"></span>
                                                            {:else}
                                                                <svg viewBox="0 0 24 24" width="20" height="20">
                                                                    <path fill="currentColor" d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
                                                                </svg>
                                                            {/if}
                                                        </button>
                                                    </div>
                                                    {#if socialError}
                                                        <p class="error-message">{socialError}</p>
                                                    {/if}
                                                </div>
                                                
                                                <!-- Status de Conexão -->
                                                <div class="connection-card {socialConnected ? 'connected' : 'disconnected'}">
                                                    <div class="connection-info">
                                                        <span class="connection-dot"></span>
                                                        <span class="connection-text">{socialConnected ? 'Servidor conectado' : 'Sem conexão'}</span>
                                                    </div>
                                                    <button type="button" class="btn-retry" aria-label="Tentar reconectar" onclick={checkSocialConnection}>
                                                        <svg viewBox="0 0 24 24" width="16" height="16">
                                                            <path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0 0 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0 1 12 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
                                                        </svg>
                                                    </button>
                                                </div>
                                            </div>
                                            
                                            <!-- Coluna Direita: Lista de Amigos -->
                                            <div class="friends-main">
                                                <div class="friends-list-header">
                                                    <h3 class="friends-title">
                                                        <svg viewBox="0 0 24 24" width="24" height="24">
                                                            <path fill="currentColor" d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
                                                        </svg>
                                                        Meus Amigos
                                                        <span class="friends-count">{socialFriends?.length || 0}</span>
                                                    </h3>
                                                    <button type="button" class="btn-refresh-friends" onclick={loadSocialFriendsActivity}>
                                                        <svg viewBox="0 0 24 24" width="18" height="18">
                                                            <path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0 0 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0 1 12 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
                                                        </svg>
                                                        Atualizar
                                                    </button>
                                                </div>
                                                
                                                {#if socialLoading}
                                                    <div class="friends-loading-state">
                                                        <div class="loading-spinner"></div>
                                                        <span>Carregando amigos...</span>
                                                    </div>
                                                {:else if !socialFriends || socialFriends.length === 0}
                                                    <div class="empty-friends-state">
                                                        <div class="empty-illustration">
                                                            <svg viewBox="0 0 24 24" width="80" height="80">
                                                                <path fill="currentColor" opacity="0.3" d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
                                                            </svg>
                                                        </div>
                                                        <h4>Nenhum amigo ainda</h4>
                                                        <p>Compartilhe seu código <strong>{socialProfile?.shareCode}</strong> com seus amigos!</p>
                                                        <div class="share-tip">
                                                            <span>💡 Dica: Copie o código clicando no botão ao lado dele</span>
                                                        </div>
                                                    </div>
                                                {:else}
                                                    <div class="friends-grid">
                                                        {#each socialFriends as friend}
                                                            <div class="friend-card {friend.isOnline ? 'is-online' : ''}">
                                                                <div class="friend-card-avatar">
                                                                    <span class="friend-initial">{friend.username?.charAt(0)?.toUpperCase() || '?'}</span>
                                                                    <span class="friend-status-dot {friend.isOnline ? 'online' : 'offline'}"></span>
                                                                </div>
                                                                <div class="friend-card-info">
                                                                    <span class="friend-card-name">{friend.username}</span>
                                                                    {#if friend.currentAnime}
                                                                        <div class="friend-watching-badge">
                                                                            <span class="watching-icon">▶</span>
                                                                            <span class="watching-text">{friend.currentAnime}</span>
                                                                        </div>
                                                                    {:else}
                                                                        <span class="friend-status-label">{friend.isOnline ? 'Online agora' : 'Offline'}</span>
                                                                    {/if}
                                                                </div>
                                                                <button 
                                                                    type="button" 
                                                                    class="btn-remove"
                                                                    onclick={() => removeFriendById(friend.userID)}
                                                                    title="Remover amigo"
                                                                >
                                                                    <svg viewBox="0 0 24 24" width="16" height="16">
                                                                        <path fill="currentColor" d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                                                                    </svg>
                                                                </button>
                                                            </div>
                                                        {/each}
                                                    </div>
                                                {/if}
                                            </div>
                                        </div>
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

    <!-- MANGA SOURCES MANAGEMENT MODAL -->
    {#if showMangaSourcesModal}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div class="modal-overlay" onclick={closeMangaSourcesModal} role="presentation">
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
            <div class="manga-sources-modal" onclick={(e) => e.stopPropagation()}>
                <button type="button" class="modal-close" onclick={closeMangaSourcesModal}>✕</button>
                
                <div class="sources-modal-header">
                    <div class="sources-icon">📚</div>
                    <h2>Gerenciar Fontes de Mangá</h2>
                    <p>Habilite ou desabilite fontes para personalizar sua experiência</p>
                </div>

                <!-- Language Filter -->
                <div class="sources-language-filter">
                    <span class="filter-label">🌐 Idioma:</span>
                    <div class="language-buttons">
                        <button 
                            class="lang-btn {selectedMangaSourceLanguage === 'all' ? 'active' : ''}"
                            onclick={() => selectedMangaSourceLanguage = 'all'}
                        >
                            🌐 Todos
                        </button>
                        <button 
                            class="lang-btn {selectedMangaSourceLanguage === 'pt-BR' ? 'active' : ''}"
                            onclick={() => selectedMangaSourceLanguage = 'pt-BR'}
                        >
                            🇧🇷 Português
                        </button>
                        <button 
                            class="lang-btn {selectedMangaSourceLanguage === 'en' ? 'active' : ''}"
                            onclick={() => selectedMangaSourceLanguage = 'en'}
                        >
                            🇺🇸 English
                        </button>
                        <button 
                            class="lang-btn {selectedMangaSourceLanguage === 'es' ? 'active' : ''}"
                            onclick={() => selectedMangaSourceLanguage = 'es'}
                        >
                            🇪🇸 Español
                        </button>
                        <button 
                            class="lang-btn {selectedMangaSourceLanguage === 'ja' ? 'active' : ''}"
                            onclick={() => selectedMangaSourceLanguage = 'ja'}
                        >
                            🇯🇵 日本語
                        </button>
                    </div>
                </div>

                <!-- Sources List -->
                <div class="sources-list">
                    {#if mangaSourcesLoading}
                        <div class="sources-loading">
                            <div class="spinner"></div>
                            <p>Carregando fontes...</p>
                        </div>
                    {:else if getFilteredMangaSources().length === 0}
                        <div class="sources-empty">
                            <span class="empty-icon">📭</span>
                            <p>Nenhuma fonte encontrada para este idioma</p>
                        </div>
                    {:else}
                        {#each getFilteredMangaSources() as source}
                            <div class="source-item {source.enabled ? 'enabled' : 'disabled'}">
                                <div class="source-info">
                                    <span class="source-icon">{source.icon || '📖'}</span>
                                    <div class="source-details">
                                        <h4>{source.name}</h4>
                                        <p>{source.description}</p>
                                        <div class="source-meta">
                                            <span class="source-lang">
                                                {source.language === 'pt-BR' ? '🇧🇷' : 
                                                 source.language === 'en' ? '🇺🇸' : 
                                                 source.language === 'es' ? '🇪🇸' : 
                                                 source.language === 'ja' ? '🇯🇵' : '🌐'} 
                                                {source.language}
                                            </span>
                                            {#if source.supportsLatest}
                                                <span class="source-feature">📅 Recentes</span>
                                            {/if}
                                            {#if source.supportsPopular}
                                                <span class="source-feature">🔥 Popular</span>
                                            {/if}
                                            {#if source.supportsSearch}
                                                <span class="source-feature">🔍 Busca</span>
                                            {/if}
                                        </div>
                                    </div>
                                </div>
                                <label class="toggle-switch">
                                    <input 
                                        type="checkbox" 
                                        checked={source.enabled}
                                        onchange={() => toggleMangaSourceEnabled(source.id, source.enabled)}
                                    />
                                    <span class="toggle-slider"></span>
                                </label>
                            </div>
                        {/each}
                    {/if}
                </div>

                <!-- Footer Actions -->
                <div class="sources-modal-footer">
                    <button class="btn-reset" onclick={resetMangaSourcesToDefault}>
                        🔄 Restaurar Padrão
                    </button>
                    <button class="btn-done" onclick={closeMangaSourcesModal}>
                        ✓ Concluído
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
        background: #050810;
        color: #fff;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif;
    }

    main {
        width: 100%;
        height: 100vh;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        background: 
            radial-gradient(ellipse at 10% 20%, rgba(139, 92, 246, 0.08) 0%, transparent 50%),
            radial-gradient(ellipse at 90% 80%, rgba(245, 87, 108, 0.06) 0%, transparent 50%),
            radial-gradient(ellipse at 50% 50%, rgba(99, 102, 241, 0.04) 0%, transparent 70%),
            linear-gradient(180deg, #050810 0%, #0a0e1a 50%, #050810 100%);
        position: relative;
    }
    
    /* Animated background particles effect */
    main::before {
        content: '';
        position: absolute;
        inset: 0;
        background: 
            radial-gradient(2px 2px at 20% 30%, rgba(255, 255, 255, 0.15), transparent),
            radial-gradient(2px 2px at 40% 70%, rgba(255, 255, 255, 0.1), transparent),
            radial-gradient(1px 1px at 60% 20%, rgba(255, 255, 255, 0.12), transparent),
            radial-gradient(2px 2px at 80% 60%, rgba(255, 255, 255, 0.08), transparent),
            radial-gradient(1px 1px at 10% 80%, rgba(245, 87, 108, 0.2), transparent),
            radial-gradient(1px 1px at 70% 40%, rgba(139, 92, 246, 0.2), transparent);
        background-size: 250px 250px, 300px 300px, 200px 200px, 350px 350px, 400px 400px, 280px 280px;
        pointer-events: none;
        opacity: 0.7;
        z-index: 0;
    }
    
    main > * {
        position: relative;
        z-index: 1;
    }

    /* ============================================
       PLAYER MODAL - Advanced Playback Options
       ============================================ */
    .player-modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.85);
        backdrop-filter: blur(10px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9999;
        animation: fadeIn 0.3s ease;
    }

    @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    .player-modal {
        background: linear-gradient(180deg, #1a1f35 0%, #0d1020 100%);
        border-radius: 20px;
        width: 90%;
        max-width: 600px;
        max-height: 90vh;
        overflow-y: auto;
        border: 1px solid rgba(138, 43, 226, 0.3);
        box-shadow: 0 25px 80px rgba(138, 43, 226, 0.3), 0 0 60px rgba(138, 43, 226, 0.1) inset;
        animation: modalSlideIn 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    @keyframes modalSlideIn {
        from {
            opacity: 0;
            transform: scale(0.8) translateY(50px);
        }
        to {
            opacity: 1;
            transform: scale(1) translateY(0);
        }
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        padding: 24px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        background: linear-gradient(180deg, rgba(138, 43, 226, 0.15) 0%, transparent 100%);
    }

    .modal-title-section {
        display: flex;
        gap: 16px;
        align-items: flex-start;
    }

    .modal-icon {
        font-size: 36px;
        filter: drop-shadow(0 0 10px rgba(138, 43, 226, 0.5));
    }

    .modal-title-info h2 {
        margin: 0 0 6px 0;
        font-size: 20px;
        font-weight: 700;
        color: #fff;
    }

    .modal-title-info p {
        margin: 0;
        font-size: 13px;
        color: rgba(255, 255, 255, 0.6);
        max-width: 350px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .modal-close {
        background: rgba(255, 255, 255, 0.1);
        border: none;
        color: #fff;
        width: 36px;
        height: 36px;
        border-radius: 50%;
        cursor: pointer;
        font-size: 18px;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
    }

    .modal-close:hover {
        background: rgba(255, 100, 100, 0.3);
        transform: rotate(90deg);
    }

    /* Quality Section */
    .modal-quality-section {
        padding: 20px 24px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .quality-header, .upscale-header, .share-header {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 16px;
        font-size: 14px;
        font-weight: 600;
        color: rgba(255, 255, 255, 0.9);
    }

    .quality-icon, .upscale-icon, .share-icon {
        font-size: 20px;
    }

    .quality-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
        gap: 10px;
    }

    .quality-item {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 10px;
        padding: 12px;
        text-align: center;
    }

    .quality-item .q-label {
        display: block;
        font-size: 11px;
        color: rgba(255, 255, 255, 0.5);
        margin-bottom: 4px;
        text-transform: uppercase;
    }

    .quality-item .q-value {
        display: block;
        font-size: 15px;
        font-weight: 600;
        color: #fff;
    }

    .quality-item .q-value.highlight {
        color: #00d4ff;
    }

    .quality-item.special {
        background: linear-gradient(135deg, rgba(138, 43, 226, 0.2), rgba(0, 212, 255, 0.2));
        border: 1px solid rgba(138, 43, 226, 0.3);
    }

    /* Upscale Modes Section */
    .modal-upscale-section {
        padding: 20px 24px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .upscale-modes {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 12px;
    }

    .upscale-mode-btn {
        background: rgba(255, 255, 255, 0.05);
        border: 2px solid transparent;
        border-radius: 14px;
        padding: 16px 12px;
        cursor: pointer;
        transition: all 0.3s;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        color: #fff;
    }

    .upscale-mode-btn:hover {
        background: rgba(138, 43, 226, 0.15);
        border-color: rgba(138, 43, 226, 0.4);
        transform: translateY(-2px);
    }

    .upscale-mode-btn.active {
        background: linear-gradient(135deg, rgba(138, 43, 226, 0.3), rgba(0, 150, 255, 0.2));
        border-color: #8a2be2;
        box-shadow: 0 4px 20px rgba(138, 43, 226, 0.4);
    }

    .mode-icon {
        font-size: 28px;
    }

    .mode-name {
        font-size: 14px;
        font-weight: 700;
    }

    .mode-desc {
        font-size: 11px;
        color: rgba(255, 255, 255, 0.6);
        text-align: center;
    }

    .mode-gpu {
        font-size: 10px;
        color: rgba(255, 255, 255, 0.4);
        background: rgba(0, 0, 0, 0.3);
        padding: 3px 8px;
        border-radius: 6px;
    }

    /* Modal Actions */
    .modal-actions {
        padding: 20px 24px;
        display: flex;
        gap: 12px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .action-btn {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        padding: 16px 20px;
        border: none;
        border-radius: 12px;
        cursor: pointer;
        font-size: 15px;
        font-weight: 600;
        transition: all 0.3s;
    }

    .action-btn.primary {
        background: linear-gradient(135deg, #8a2be2, #6b21a8);
        color: #fff;
        box-shadow: 0 4px 20px rgba(138, 43, 226, 0.4);
    }

    .action-btn.primary:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 30px rgba(138, 43, 226, 0.5);
    }

    .action-btn.secondary {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
    }

    .action-btn.secondary:hover {
        background: rgba(255, 255, 255, 0.15);
    }

    .action-icon {
        font-size: 18px;
    }

    /* Share Section */
    .modal-share-section {
        padding: 16px 24px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .share-header {
        cursor: pointer;
        padding: 8px;
        margin: -8px;
        border-radius: 8px;
        transition: background 0.2s;
    }

    .share-header:hover {
        background: rgba(255, 255, 255, 0.05);
    }

    .share-expand {
        margin-left: auto;
        font-size: 12px;
        color: rgba(255, 255, 255, 0.5);
    }

    .share-content {
        margin-top: 16px;
        animation: fadeIn 0.3s ease;
    }

    .generate-link-btn {
        width: 100%;
        padding: 14px;
        background: linear-gradient(135deg, rgba(0, 200, 100, 0.2), rgba(0, 150, 100, 0.1));
        border: 1px solid rgba(0, 200, 100, 0.3);
        border-radius: 10px;
        color: #00c864;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s;
    }

    .generate-link-btn:hover:not(:disabled) {
        background: linear-gradient(135deg, rgba(0, 200, 100, 0.3), rgba(0, 150, 100, 0.2));
        transform: translateY(-1px);
    }

    .generate-link-btn:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .share-hint {
        margin: 10px 0 0 0;
        font-size: 12px;
        color: rgba(255, 255, 255, 0.5);
        text-align: center;
    }

    .share-link-box {
        display: flex;
        gap: 8px;
    }

    .share-link-input {
        flex: 1;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 8px;
        padding: 12px;
        color: #fff;
        font-size: 12px;
        font-family: monospace;
    }

    .copy-link-btn {
        background: rgba(138, 43, 226, 0.3);
        border: 1px solid rgba(138, 43, 226, 0.5);
        border-radius: 8px;
        padding: 12px 16px;
        color: #fff;
        cursor: pointer;
        font-weight: 600;
        transition: all 0.2s;
    }

    .copy-link-btn:hover {
        background: rgba(138, 43, 226, 0.4);
    }

    .share-actions {
        display: flex;
        gap: 8px;
        margin-top: 12px;
    }

    .share-action {
        flex: 1;
        padding: 10px;
        border: none;
        border-radius: 8px;
        cursor: pointer;
        font-size: 12px;
        font-weight: 600;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        transition: all 0.2s;
        color: #fff;
    }

    .share-action.whatsapp {
        background: linear-gradient(135deg, #25d366, #128c7e);
    }

    .share-action.discord {
        background: linear-gradient(135deg, #5865f2, #4752c4);
    }

    .share-action.telegram {
        background: linear-gradient(135deg, #0088cc, #006699);
    }

    .share-action:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
    }

    /* Subtitle Section */
    .modal-subtitle-section {
        padding: 16px 24px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .subtitle-header {
        display: flex;
        align-items: center;
        gap: 10px;
        cursor: pointer;
        padding: 8px;
        margin: -8px;
        border-radius: 8px;
        transition: background 0.2s;
        color: #fff;
        font-weight: 500;
    }

    .subtitle-header:hover {
        background: rgba(255, 255, 255, 0.05);
    }

    .subtitle-icon {
        font-size: 18px;
    }

    .subtitle-expand {
        margin-left: auto;
        font-size: 12px;
        color: rgba(255, 255, 255, 0.5);
    }

    .subtitle-content {
        margin-top: 16px;
        animation: fadeIn 0.3s ease;
    }

    .subtitle-url-section {
        margin-bottom: 16px;
    }

    .subtitle-label {
        display: block;
        font-size: 12px;
        color: rgba(255, 255, 255, 0.6);
        margin-bottom: 8px;
        font-weight: 500;
    }

    .subtitle-url-input-wrapper {
        display: flex;
        gap: 8px;
        position: relative;
    }

    .subtitle-url-input {
        flex: 1;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 10px;
        padding: 12px 40px 12px 14px;
        color: #fff;
        font-size: 13px;
        transition: all 0.2s;
    }

    .subtitle-url-input:focus {
        outline: none;
        border-color: rgba(138, 43, 226, 0.5);
        box-shadow: 0 0 0 3px rgba(138, 43, 226, 0.1);
    }

    .subtitle-url-input::placeholder {
        color: rgba(255, 255, 255, 0.3);
    }

    .subtitle-clear-btn {
        position: absolute;
        right: 10px;
        top: 50%;
        transform: translateY(-50%);
        background: rgba(255, 255, 255, 0.1);
        border: none;
        border-radius: 50%;
        width: 24px;
        height: 24px;
        cursor: pointer;
        color: rgba(255, 255, 255, 0.6);
        font-size: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
    }

    .subtitle-clear-btn:hover {
        background: rgba(255, 100, 100, 0.3);
        color: #ff6b6b;
    }

    .subtitle-list-section {
        margin-bottom: 16px;
    }

    .subtitle-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .subtitle-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 14px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 10px;
        cursor: pointer;
        transition: all 0.2s;
        text-align: left;
        color: #fff;
    }

    .subtitle-item:hover {
        background: rgba(255, 255, 255, 0.08);
        border-color: rgba(138, 43, 226, 0.3);
    }

    .subtitle-item.active {
        background: rgba(138, 43, 226, 0.2);
        border-color: rgba(138, 43, 226, 0.5);
    }

    .sub-lang {
        font-size: 18px;
    }

    .sub-name {
        flex: 1;
        font-size: 13px;
    }

    .sub-format {
        font-size: 11px;
        padding: 3px 8px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 4px;
        color: rgba(255, 255, 255, 0.6);
    }

    .subtitle-none-btn {
        width: 100%;
        padding: 12px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 10px;
        color: rgba(255, 255, 255, 0.6);
        cursor: pointer;
        font-size: 13px;
        transition: all 0.2s;
    }

    .subtitle-none-btn:hover {
        background: rgba(255, 255, 255, 0.08);
    }

    .subtitle-none-btn.active {
        background: rgba(100, 100, 100, 0.2);
        border-color: rgba(100, 100, 100, 0.4);
        color: #fff;
    }

    .subtitle-hint {
        margin: 16px 0 0 0;
        padding: 10px;
        background: rgba(255, 200, 50, 0.1);
        border: 1px solid rgba(255, 200, 50, 0.2);
        border-radius: 8px;
        font-size: 12px;
        color: rgba(255, 200, 100, 0.8);
        text-align: center;
    }

    /* Modal Extras */
    .modal-extras {
        padding: 16px 24px;
    }

    .extra-btn {
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 14px 16px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 10px;
        color: #fff;
        cursor: pointer;
        transition: all 0.2s;
    }

    .extra-btn:hover:not(:disabled) {
        background: rgba(255, 255, 255, 0.08);
        border-color: rgba(138, 43, 226, 0.3);
    }

    .extra-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .extra-hint {
        font-size: 11px;
        color: rgba(255, 255, 255, 0.4);
    }

    .loading-spinner {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        display: inline-block;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    /* ============================================
       QUALITY BADGES - Episode Cards
       ============================================ */
    .quality-badges {
        display: flex;
        gap: 6px;
        flex-wrap: wrap;
        margin-bottom: 10px;
    }

    .q-badge {
        padding: 3px 8px;
        border-radius: 6px;
        font-size: 10px;
        font-weight: 700;
        text-transform: uppercase;
    }

    .q-badge.resolution {
        background: linear-gradient(135deg, #00d4ff, #0088ff);
        color: #000;
    }

    .q-badge.codec {
        background: rgba(138, 43, 226, 0.3);
        border: 1px solid rgba(138, 43, 226, 0.5);
        color: #e0b0ff;
    }

    .q-badge.dual {
        background: linear-gradient(135deg, rgba(0, 200, 100, 0.3), rgba(0, 150, 80, 0.2));
        border: 1px solid rgba(0, 200, 100, 0.4);
        color: #7fff7f;
    }

    .q-badge.hdr {
        background: linear-gradient(135deg, #ffd700, #ff8c00);
        color: #000;
    }

    /* TorBox Action Buttons */
    .torbox-action-buttons {
        display: flex;
        gap: 8px;
        margin-top: 8px;
    }

    .btn-torbox-primary {
        flex: 1;
        background: linear-gradient(135deg, #8a2be2, #6b21a8);
        border: none;
        border-radius: 10px;
        padding: 12px 16px;
        color: #fff;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        transition: all 0.3s;
        box-shadow: 0 4px 15px rgba(138, 43, 226, 0.3);
    }

    .btn-torbox-primary:hover {
        transform: translateY(-2px);
        box-shadow: 0 6px 20px rgba(138, 43, 226, 0.4);
    }

    /* ============================================
       GROUPED RESULTS - Por Série/Temporada
       ============================================ */
    .grouped-results {
        display: flex;
        flex-direction: column;
        gap: 16px;
        padding: 0 20px 20px;
    }

    .series-group {
        background: linear-gradient(180deg, rgba(30, 35, 55, 0.8) 0%, rgba(20, 25, 40, 0.9) 100%);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 16px;
        overflow: hidden;
        transition: all 0.3s;
    }

    .series-group:hover {
        border-color: rgba(138, 43, 226, 0.3);
    }

    .series-group.expanded {
        border-color: rgba(138, 43, 226, 0.5);
        box-shadow: 0 8px 30px rgba(138, 43, 226, 0.15);
    }

    .group-header {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 16px;
        background: transparent;
        border: none;
        width: 100%;
        cursor: pointer;
        transition: background 0.2s;
        color: #fff;
        text-align: left;
    }

    .group-header:hover {
        background: rgba(255, 255, 255, 0.05);
    }

    .group-poster {
        width: 70px;
        height: 100px;
        border-radius: 10px;
        overflow: hidden;
        flex-shrink: 0;
        position: relative;
        background: rgba(0, 0, 0, 0.3);
    }

    .group-poster img {
        width: 100%;
        height: 100%;
        object-fit: cover;
    }

    .group-poster .no-image {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        font-size: 24px;
    }

    .group-poster .br-badge {
        position: absolute;
        top: 4px;
        right: 4px;
        font-size: 14px;
        background: rgba(0, 0, 0, 0.7);
        padding: 2px 4px;
        border-radius: 4px;
    }

    .group-info {
        flex: 1;
        min-width: 0;
    }

    .group-title {
        margin: 0 0 8px 0;
        font-size: 18px;
        font-weight: 700;
        color: #fff;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .group-meta {
        display: flex;
        gap: 12px;
        flex-wrap: wrap;
        align-items: center;
    }

    .group-count, .group-seasons {
        font-size: 13px;
        color: rgba(255, 255, 255, 0.6);
    }

    .source-tag {
        font-size: 11px;
        padding: 3px 8px;
        border-radius: 6px;
        font-weight: 600;
    }

    .source-tag.br {
        background: linear-gradient(135deg, rgba(0, 156, 59, 0.3), rgba(255, 223, 0, 0.2));
        border: 1px solid rgba(0, 156, 59, 0.4);
        color: #7fff7f;
    }

    .expand-icon {
        font-size: 16px;
        color: rgba(255, 255, 255, 0.5);
        transition: transform 0.3s;
    }

    .series-group.expanded .expand-icon {
        color: #8a2be2;
    }

    .group-content {
        padding: 0 16px 16px;
        animation: slideDown 0.3s ease;
    }

    @keyframes slideDown {
        from {
            opacity: 0;
            transform: translateY(-10px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .season-section {
        margin-bottom: 20px;
    }

    .season-section:last-child {
        margin-bottom: 0;
    }

    .season-section.all-seasons {
        background: linear-gradient(135deg, rgba(56, 142, 60, 0.15), rgba(46, 125, 50, 0.1));
        border: 1px solid rgba(76, 175, 80, 0.3);
        border-radius: 12px;
        padding: 16px;
        margin-bottom: 24px;
    }

    .season-title {
        display: flex;
        align-items: center;
        gap: 8px;
        margin: 0 0 12px 0;
        font-size: 15px;
        font-weight: 600;
        color: #fff;
        padding-bottom: 8px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    }

    .season-title.highlight-all {
        font-size: 17px;
        color: #4caf50;
        border-bottom-color: rgba(76, 175, 80, 0.3);
    }

    .all-badge {
        background: linear-gradient(135deg, #4caf50, #2e7d32);
        color: white;
        padding: 2px 8px;
        border-radius: 6px;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.5px;
        margin-left: auto;
        box-shadow: 0 2px 8px rgba(76, 175, 80, 0.3);
    }

    .season-icon {
        font-size: 16px;
    }

    .source-section {
        margin-bottom: 12px;
    }

    .source-section:last-child {
        margin-bottom: 0;
    }

    .source-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;
    }

    .source-badge {
        font-size: 11px;
        padding: 4px 10px;
        border-radius: 6px;
        font-weight: 600;
    }

    .source-badge.br {
        background: linear-gradient(135deg, rgba(0, 156, 59, 0.4), rgba(255, 223, 0, 0.3));
        border: 1px solid rgba(0, 156, 59, 0.5);
        color: #90ff90;
    }

    .source-badge.en {
        background: linear-gradient(135deg, rgba(60, 90, 171, 0.4), rgba(191, 10, 48, 0.3));
        border: 1px solid rgba(60, 90, 171, 0.5);
        color: #a0c0ff;
    }

    .source-badge.other {
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        color: rgba(255, 255, 255, 0.7);
    }

    .source-count {
        font-size: 12px;
        color: rgba(255, 255, 255, 0.4);
    }

    .torrent-list {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .torrent-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 14px;
        background: rgba(0, 0, 0, 0.2);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 10px;
        cursor: pointer;
        transition: all 0.2s;
        color: #fff;
        text-align: left;
        width: 100%;
    }

    .torrent-item:hover {
        background: rgba(138, 43, 226, 0.15);
        border-color: rgba(138, 43, 226, 0.3);
        transform: translateX(4px);
    }

    .torrent-item.br-highlight {
        background: linear-gradient(90deg, rgba(0, 156, 59, 0.1), rgba(0, 0, 0, 0.2));
        border-color: rgba(0, 156, 59, 0.2);
    }

    .torrent-item.br-highlight:hover {
        background: linear-gradient(90deg, rgba(0, 156, 59, 0.2), rgba(138, 43, 226, 0.15));
        border-color: rgba(0, 156, 59, 0.4);
    }

    .torrent-info {
        flex: 1;
        min-width: 0;
    }

    .torrent-title {
        display: block;
        font-size: 13px;
        font-weight: 500;
        color: #fff;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        margin-bottom: 4px;
    }

    .torrent-meta {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
    }

    .torrent-meta .size {
        font-size: 12px;
        color: rgba(255, 255, 255, 0.5);
    }

    .torrent-meta .seeds {
        font-size: 12px;
        color: #4ade80;
    }

    .torrent-meta .cached {
        font-size: 11px;
        color: #fbbf24;
        background: rgba(251, 191, 36, 0.15);
        padding: 2px 6px;
        border-radius: 4px;
    }

    .play-btn {
        width: 32px;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: linear-gradient(135deg, #8a2be2, #6b21a8);
        border-radius: 50%;
        font-size: 12px;
        flex-shrink: 0;
        transition: transform 0.2s;
    }

    .torrent-item:hover .play-btn {
        transform: scale(1.1);
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

    /* Auth Tabs */
    .auth-tabs {
        display: flex;
        border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    }

    .auth-tab {
        flex: 1;
        padding: 18px;
        background: transparent;
        border: none;
        color: #888;
        font-size: 1rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.3s;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
    }

    .auth-tab:hover {
        color: #fff;
        background: rgba(255, 255, 255, 0.05);
    }

    .auth-tab.active {
        color: #f5576c;
        background: rgba(245, 87, 108, 0.1);
        border-bottom: 2px solid #f5576c;
    }

    /* Auth Error */
    .auth-error {
        background: rgba(255, 68, 68, 0.15);
        border: 1px solid rgba(255, 68, 68, 0.3);
        color: #ff6666;
        padding: 12px 15px;
        border-radius: 10px;
        margin: 15px 20px 0;
        font-size: 0.9rem;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    /* Guest Section */
    .guest-section {
        padding: 0 30px 30px;
    }

    .divider {
        display: flex;
        align-items: center;
        gap: 15px;
        margin: 20px 0;
        color: #666;
        font-size: 0.85rem;
    }

    .divider::before,
    .divider::after {
        content: '';
        flex: 1;
        height: 1px;
        background: rgba(255, 255, 255, 0.1);
    }

    .btn-guest {
        width: 100%;
        padding: 14px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        color: #aaa;
        font-size: 1rem;
        cursor: pointer;
        transition: all 0.3s;
    }

    .btn-guest:hover:not(:disabled) {
        background: rgba(255, 255, 255, 0.1);
        color: #fff;
        border-color: rgba(255, 255, 255, 0.2);
    }

    .btn-guest:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .guest-info {
        margin: 12px 0 0;
        text-align: center;
        color: #666;
        font-size: 0.8rem;
        line-height: 1.5;
    }

    /* Loading Spinner */
    .loading-spinner {
        width: 20px;
        height: 20px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
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

    /* ============================================
       HEADER - Premium Glass Morphism
       ============================================ */
    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 30px;
        background: linear-gradient(180deg, rgba(10, 14, 26, 0.95), rgba(5, 8, 16, 0.9));
        backdrop-filter: blur(20px);
        border-bottom: 1px solid rgba(139, 92, 246, 0.15);
        z-index: 100;
        box-shadow: 
            0 4px 30px rgba(0, 0, 0, 0.3),
            0 1px 0 rgba(255, 255, 255, 0.03) inset;
    }

    .header.minimal {
        position: absolute;
        top: 0;
        right: 0;
        background: transparent;
        backdrop-filter: none;
        border: none;
        padding: 20px 40px;
        box-shadow: none;
    }

    .header-left {
        display: flex;
        align-items: center;
    }

    :global(.header h1) {
        margin: 0;
        font-size: 1.5rem;
        background: linear-gradient(135deg, #fff, rgba(245, 87, 108, 0.9));
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        font-weight: 700;
        letter-spacing: -0.5px;
    }

    .user-section {
        color: #fff;
        display: flex;
        align-items: center;
        gap: 12px;
        background: linear-gradient(145deg, rgba(26, 31, 58, 0.8), rgba(20, 25, 45, 0.9));
        padding: 10px 20px;
        border-radius: 30px;
        cursor: pointer;
        border: 1px solid rgba(139, 92, 246, 0.2);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
    }

    .user-section:hover {
        background: linear-gradient(145deg, rgba(30, 35, 65, 0.9), rgba(25, 30, 55, 0.95));
        border-color: rgba(245, 87, 108, 0.4);
        box-shadow: 
            0 8px 25px rgba(0, 0, 0, 0.3),
            0 0 20px rgba(245, 87, 108, 0.1);
        transform: translateY(-1px);
    }

    .menu-arrow {
        font-size: 0.7rem;
        opacity: 0.7;
        transition: transform 0.2s ease;
    }
    
    .user-section:hover .menu-arrow {
        transform: translateY(2px);
    }

    .user-menu-container {
        position: relative;
    }

    .user-dropdown {
        position: absolute;
        top: calc(100% + 12px);
        right: 0;
        background: linear-gradient(180deg, rgba(26, 31, 58, 0.98), rgba(15, 20, 40, 0.99));
        backdrop-filter: blur(20px);
        border: 1px solid rgba(139, 92, 246, 0.2);
        border-radius: 16px;
        min-width: 220px;
        box-shadow: 
            0 20px 60px rgba(0, 0, 0, 0.5),
            0 0 40px rgba(139, 92, 246, 0.1);
        overflow: hidden;
        z-index: 1000;
        animation: slideDown 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    }

    @keyframes slideDown {
        from { opacity: 0; transform: translateY(-15px) scale(0.95); }
        to { opacity: 1; transform: translateY(0) scale(1); }
    }

    .dropdown-item {
        display: flex;
        align-items: center;
        gap: 10px;
        width: 100%;
        padding: 14px 20px;
        background: transparent;
        border: none;
        color: rgba(255, 255, 255, 0.9);
        font-size: 0.95rem;
        text-align: left;
        cursor: pointer;
        transition: all 0.2s ease;
        border-left: 3px solid transparent;
    }

    .dropdown-item:hover {
        background: linear-gradient(90deg, rgba(245, 87, 108, 0.15), transparent);
        border-left-color: #f5576c;
        color: #fff;
    }

    .dropdown-item.logout {
        color: #ff6666;
    }

    .dropdown-item.logout:hover {
        background: rgba(255, 68, 68, 0.2);
    }

    .dropdown-item.guest-upgrade {
        color: #ffaa00;
    }

    .dropdown-item.guest-upgrade:hover {
        background: rgba(255, 170, 0, 0.2);
    }

    .dropdown-divider {
        height: 1px;
        background: rgba(255, 255, 255, 0.1);
        margin: 5px 0;
    }

    .guest-badge {
        display: inline-block;
        padding: 2px 8px;
        background: rgba(255, 170, 0, 0.2);
        color: #ffaa00;
        font-size: 0.7rem;
        border-radius: 10px;
        margin-left: 8px;
        vertical-align: middle;
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
       HERO SECTION - PREMIUM IMMERSIVE
       ============================================ */
    .hero-section-modern {
        position: relative;
        width: 100%;
        min-height: 55vh;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        padding: 70px 20px;
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
            radial-gradient(ellipse at 20% 10%, rgba(245, 87, 108, 0.25) 0%, transparent 45%),
            radial-gradient(ellipse at 80% 20%, rgba(139, 92, 246, 0.2) 0%, transparent 40%),
            radial-gradient(ellipse at 50% 80%, rgba(79, 172, 254, 0.15) 0%, transparent 50%),
            radial-gradient(circle at 10% 90%, rgba(240, 147, 251, 0.1) 0%, transparent 35%),
            linear-gradient(180deg, rgba(5, 8, 16, 0) 0%, #050810 100%);
        animation: heroGradientPulse 8s ease-in-out infinite;
    }
    
    @keyframes heroGradientPulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.85; }
    }

    .hero-grid-pattern {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-image: 
            linear-gradient(rgba(139, 92, 246, 0.03) 1px, transparent 1px),
            linear-gradient(90deg, rgba(139, 92, 246, 0.03) 1px, transparent 1px);
        background-size: 60px 60px;
        mask-image: radial-gradient(ellipse at center, black 0%, transparent 65%);
        -webkit-mask-image: radial-gradient(ellipse at center, black 0%, transparent 65%);
        animation: gridPulse 4s ease-in-out infinite;
    }
    
    @keyframes gridPulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.6; }
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
        gap: 12px;
        margin-bottom: 25px;
    }

    .hero-emoji {
        font-size: clamp(3.5rem, 8vw, 5.5rem);
        animation: hero-float 3s ease-in-out infinite;
        filter: drop-shadow(0 10px 30px rgba(245, 87, 108, 0.3));
    }

    @keyframes hero-float {
        0%, 100% { transform: translateY(0) scale(1); }
        50% { transform: translateY(-18px) scale(1.05); }
    }

    .hero-brand {
        font-size: clamp(2.8rem, 8vw, 5rem);
        font-weight: 800;
        margin: 0;
        letter-spacing: -3px;
        text-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
    }

    .brand-go {
        color: #fff;
        text-shadow: 0 0 40px rgba(255, 255, 255, 0.2);
    }

    .brand-anime {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 35%, #8b5cf6 70%, #4facfe 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        background-size: 300% 300%;
        animation: gradient-shift 6s ease infinite;
        filter: drop-shadow(0 0 30px rgba(245, 87, 108, 0.4));
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
        max-width: 800px;
        padding: 20px;
    }
    
    .settings-title {
        display: flex;
        align-items: center;
        gap: 15px;
        font-size: 2rem;
        margin-bottom: 30px;
        animation: fadeDown 0.5s ease-out;
    }
    
    .settings-title .title-icon {
        font-size: 2.5rem;
        animation: spin 4s linear infinite;
    }
    
    /* Settings Cards */
    .settings-card {
        background: linear-gradient(135deg, rgba(26, 31, 58, 0.95) 0%, rgba(15, 20, 45, 0.98) 100%);
        border-radius: 20px;
        padding: 0;
        margin-bottom: 25px;
        border: 1px solid rgba(255, 255, 255, 0.08);
        overflow: hidden;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
        animation: fadeUp 0.5s ease-out;
        animation-fill-mode: both;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .settings-card:nth-child(2) { animation-delay: 0.1s; }
    .settings-card:nth-child(3) { animation-delay: 0.2s; }
    .settings-card:nth-child(4) { animation-delay: 0.3s; }
    .settings-card:nth-child(5) { animation-delay: 0.4s; }
    
    .settings-card:hover {
        transform: translateY(-3px);
        box-shadow: 0 15px 50px rgba(0, 0, 0, 0.4);
        border-color: rgba(245, 87, 108, 0.2);
    }
    
    .card-header {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 20px 25px;
        background: linear-gradient(135deg, rgba(245, 87, 108, 0.15) 0%, rgba(79, 172, 254, 0.1) 100%);
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }
    
    .card-header .card-icon {
        font-size: 1.5rem;
        animation: float 3s ease-in-out infinite;
    }
    
    .card-header h3 {
        margin: 0;
        font-size: 1.2rem;
        font-weight: 600;
        background: linear-gradient(135deg, #fff 0%, #ccc 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .badge-beta {
        margin-left: auto;
        padding: 4px 12px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border-radius: 20px;
        font-size: 0.65rem;
        font-weight: 700;
        letter-spacing: 1px;
        color: #fff;
        animation: pulse 2s ease-in-out infinite;
    }
    
    .card-content {
        padding: 25px;
    }
    
    .card-footer {
        padding: 20px 25px;
        background: rgba(0, 0, 0, 0.2);
        border-top: 1px solid rgba(255, 255, 255, 0.05);
    }
    
    .card-description {
        color: #888;
        font-size: 0.9rem;
        margin-bottom: 20px;
        line-height: 1.5;
    }
    
    /* Toggle Switches */
    .setting-toggle {
        display: flex;
        align-items: center;
        gap: 15px;
        cursor: pointer;
        padding: 12px 0;
        transition: all 0.2s;
    }
    
    .setting-toggle:hover {
        opacity: 0.9;
    }
    
    .setting-toggle input[type="checkbox"] {
        display: none;
    }
    
    .toggle-slider {
        position: relative;
        width: 50px;
        height: 26px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 30px;
        border: 2px solid rgba(255, 255, 255, 0.15);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .toggle-slider::after {
        content: '';
        position: absolute;
        top: 3px;
        left: 3px;
        width: 16px;
        height: 16px;
        background: #fff;
        border-radius: 50%;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
    }
    
    .setting-toggle input:checked + .toggle-slider {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border-color: transparent;
    }
    
    .setting-toggle input:checked + .toggle-slider::after {
        transform: translateX(24px);
    }
    
    .toggle-slider.seeding {
        width: 60px;
        height: 30px;
    }
    
    .toggle-slider.seeding::after {
        width: 20px;
        height: 20px;
        top: 3px;
    }
    
    .setting-toggle input:checked + .toggle-slider.seeding {
        background: linear-gradient(135deg, #4dff88 0%, #00d4aa 100%);
    }
    
    .setting-toggle input:checked + .toggle-slider.seeding::after {
        transform: translateX(30px);
    }
    
    .toggle-label {
        font-size: 1rem;
        color: #ddd;
    }
    
    .main-toggle {
        padding: 15px;
        background: rgba(255, 255, 255, 0.03);
        border-radius: 12px;
        margin: 15px 0;
    }
    
    /* Select Styling */
    .setting-select {
        margin: 15px 0;
    }
    
    .setting-select label {
        display: block;
        color: #aaa;
        font-size: 0.9rem;
        margin-bottom: 8px;
    }
    
    .setting-select select,
    .seeding-option select {
        width: 100%;
        padding: 12px 15px;
        background: rgba(10, 14, 39, 0.8);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 10px;
        color: #fff;
        font-size: 1rem;
        cursor: pointer;
        transition: all 0.2s;
        appearance: none;
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23888' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
        background-repeat: no-repeat;
        background-position: right 15px center;
    }
    
    .setting-select select:hover,
    .seeding-option select:hover {
        border-color: rgba(245, 87, 108, 0.4);
    }
    
    .setting-select select:focus,
    .seeding-option select:focus {
        outline: none;
        border-color: #f5576c;
        box-shadow: 0 0 0 3px rgba(245, 87, 108, 0.2);
    }
    
    /* Buttons */
    .btn-save {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 12px 25px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border: none;
        border-radius: 10px;
        color: #fff;
        font-size: 1rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .btn-save:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 25px rgba(245, 87, 108, 0.4);
    }
    
    .btn-action {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 12px 20px;
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 10px;
        color: #fff;
        font-size: 0.95rem;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .btn-action:hover {
        background: rgba(255, 255, 255, 0.12);
        transform: translateY(-2px);
    }
    
    .btn-action.export:hover {
        border-color: #4dff88;
        color: #4dff88;
    }
    
    .btn-action.import:hover {
        border-color: #4facfe;
        color: #4facfe;
    }
    
    .btn-action.refresh {
        background: linear-gradient(135deg, rgba(79, 172, 254, 0.2) 0%, rgba(0, 242, 254, 0.2) 100%);
        border-color: rgba(79, 172, 254, 0.5);
    }
    
    .btn-action.refresh:hover {
        background: linear-gradient(135deg, rgba(79, 172, 254, 0.3) 0%, rgba(0, 242, 254, 0.3) 100%);
    }
    
    .btn-action.warning {
        background: rgba(255, 165, 0, 0.15);
        border-color: rgba(255, 165, 0, 0.4);
        color: #ffa500;
    }
    
    .btn-action.warning:hover {
        background: rgba(255, 165, 0, 0.25);
    }
    
    .btn-action.danger {
        background: rgba(255, 77, 77, 0.15);
        border-color: rgba(255, 77, 77, 0.4);
        color: #ff6b6b;
    }
    
    .btn-action.danger:hover {
        background: rgba(255, 77, 77, 0.25);
    }
    
    .btn-icon {
        font-size: 1.1rem;
    }
    
    /* Seeding Section */
    .seeding-card {
        border-color: rgba(77, 255, 136, 0.2);
    }
    
    .seeding-card .card-header {
        background: linear-gradient(135deg, rgba(77, 255, 136, 0.15) 0%, rgba(0, 212, 170, 0.1) 100%);
    }
    
    .seeding-description {
        background: rgba(255, 255, 255, 0.03);
        padding: 20px;
        border-radius: 12px;
        border-left: 3px solid #4dff88;
        margin-bottom: 20px;
    }
    
    .seeding-description p {
        margin: 0 0 12px 0;
        color: #bbb;
        font-size: 0.95rem;
    }
    
    .seeding-description ul {
        margin: 0;
        padding-left: 5px;
        list-style: none;
    }
    
    .seeding-description li {
        padding: 6px 0;
        color: #999;
        font-size: 0.9rem;
        transition: all 0.2s;
    }
    
    .seeding-description li:hover {
        color: #fff;
        transform: translateX(5px);
    }
    
    .seeding-options {
        background: rgba(77, 255, 136, 0.03);
        padding: 20px;
        border-radius: 12px;
        margin: 15px 0;
        border: 1px solid rgba(77, 255, 136, 0.1);
        animation: fadeUp 0.3s ease-out;
    }
    
    .seeding-option {
        margin-bottom: 20px;
    }
    
    .seeding-option:last-of-type {
        margin-bottom: 0;
    }
    
    .seeding-option label {
        display: flex;
        align-items: center;
        gap: 10px;
        color: #aaa;
        font-size: 0.9rem;
        margin-bottom: 10px;
    }
    
    .seeding-option .option-icon {
        font-size: 1.2rem;
    }
    
    .slider-container {
        display: flex;
        align-items: center;
        gap: 15px;
    }
    
    .slider-container input[type="range"] {
        flex: 1;
        height: 6px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 3px;
        appearance: none;
        cursor: pointer;
    }
    
    .slider-container input[type="range"]::-webkit-slider-thumb {
        appearance: none;
        width: 20px;
        height: 20px;
        background: linear-gradient(135deg, #4dff88 0%, #00d4aa 100%);
        border-radius: 50%;
        cursor: pointer;
        box-shadow: 0 2px 10px rgba(77, 255, 136, 0.4);
        transition: all 0.2s;
    }
    
    .slider-container input[type="range"]::-webkit-slider-thumb:hover {
        transform: scale(1.2);
    }
    
    .slider-value {
        min-width: 60px;
        padding: 6px 12px;
        background: rgba(77, 255, 136, 0.1);
        border-radius: 6px;
        color: #4dff88;
        font-weight: 600;
        font-size: 0.9rem;
        text-align: center;
    }
    
    .seeding-stats {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 15px;
        margin-top: 20px;
        padding-top: 20px;
        border-top: 1px solid rgba(255, 255, 255, 0.05);
    }
    
    .stat-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 15px;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 12px;
    }
    
    .stat-item .stat-icon {
        font-size: 1.5rem;
    }
    
    .stat-info {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    
    .stat-info .stat-label {
        font-size: 0.8rem;
        color: #888;
    }
    
    .stat-info .stat-value {
        font-size: 1.1rem;
        font-weight: 600;
        color: #4dff88;
    }
    
    .stat-info .stat-value.status {
        color: #4facfe;
    }
    
    /* Backup Buttons */
    .backup-buttons {
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
    }

    /* IMPORT/EXPORT MODAL */
    .import-export-modal {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.85);
        backdrop-filter: blur(10px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 2000;
        animation: fadeIn 0.2s ease-out;
    }
    
    @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    .modal-content {
        background: linear-gradient(135deg, #1a1f3a 0%, #0f1429 100%);
        border-radius: 20px;
        padding: 30px;
        max-width: 500px;
        width: 90%;
        max-height: 80vh;
        overflow-y: auto;
        border: 1px solid rgba(255, 255, 255, 0.1);
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
        animation: scaleIn 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    @keyframes scaleIn {
        from { transform: scale(0.9); opacity: 0; }
        to { transform: scale(1); opacity: 1; }
    }

    .modal-content h4 {
        margin: 0 0 25px 0;
        font-size: 1.4rem;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .modal-content textarea {
        width: 100%;
        height: 150px;
        padding: 15px;
        background: rgba(10, 14, 39, 0.8);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        color: #fff;
        font-family: 'Fira Code', monospace;
        font-size: 0.85rem;
        resize: vertical;
        margin-bottom: 15px;
        transition: all 0.2s;
    }
    
    .modal-content textarea:focus {
        outline: none;
        border-color: #f5576c;
        box-shadow: 0 0 0 3px rgba(245, 87, 108, 0.2);
    }

    .modal-content p {
        color: #aaa;
        margin-bottom: 12px;
        font-size: 0.95rem;
    }

    .export-section,
    .import-section {
        margin-bottom: 25px;
    }

    .btn-close {
        display: block;
        width: 100%;
        padding: 14px;
        margin-top: 15px;
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 10px;
        color: #fff;
        font-size: 1rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-close:hover {
        background: rgba(255, 77, 77, 0.2);
        border-color: rgba(255, 77, 77, 0.4);
        color: #ff6b6b;
    }

    /* === SOURCES STATUS SECTION === */
    .sources-card {
        border-color: rgba(79, 172, 254, 0.2);
    }
    
    .sources-card .card-header {
        background: linear-gradient(135deg, rgba(79, 172, 254, 0.15) 0%, rgba(0, 242, 254, 0.1) 100%);
    }

    .cache-overview {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 15px;
        margin: 20px 0;
    }

    .cache-stat-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 20px;
        background: linear-gradient(135deg, rgba(79, 172, 254, 0.1) 0%, rgba(0, 242, 254, 0.05) 100%);
        border-radius: 12px;
        border: 1px solid rgba(79, 172, 254, 0.2);
        transition: all 0.3s;
    }
    
    .cache-stat-card:hover {
        transform: translateY(-3px);
        border-color: rgba(79, 172, 254, 0.4);
    }
    
    .cache-stat-card .cache-icon {
        font-size: 2rem;
    }
    
    .cache-stat-card .cache-label {
        color: #888;
        font-size: 0.85rem;
    }
    
    .cache-stat-card .cache-value {
        font-size: 1.3rem;
        font-weight: 700;
        color: #4facfe;
    }

    .sources-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 15px;
        margin: 20px 0;
    }

    .source-status-card {
        background: rgba(0, 0, 0, 0.25);
        border-radius: 14px;
        padding: 18px;
        border: 2px solid transparent;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .source-status-card:hover {
        transform: translateY(-3px);
    }

    .source-status-card.available {
        border-color: rgba(77, 255, 136, 0.25);
        background: linear-gradient(135deg, rgba(77, 255, 136, 0.05) 0%, rgba(0, 212, 170, 0.02) 100%);
    }
    
    .source-status-card.available:hover {
        border-color: rgba(77, 255, 136, 0.5);
    }

    .source-status-card.unavailable {
        border-color: rgba(255, 77, 77, 0.25);
        background: linear-gradient(135deg, rgba(255, 77, 77, 0.08) 0%, rgba(255, 77, 77, 0.02) 100%);
    }
    
    .source-status-card.unavailable:hover {
        border-color: rgba(255, 77, 77, 0.5);
    }

    .source-status-card .source-header {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 12px;
    }

    .source-status-card .source-icon {
        font-size: 1.3rem;
    }

    .source-status-card .source-name {
        font-weight: 600;
        font-size: 1rem;
    }

    .source-status-card .source-details {
        display: flex;
        flex-direction: column;
        gap: 6px;
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
        color: #777;
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

    .no-stats {
        color: #777;
        font-style: italic;
        text-align: center;
        padding: 30px;
        background: rgba(255, 255, 255, 0.02);
        border-radius: 12px;
        border: 1px dashed rgba(255, 255, 255, 0.1);
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
        /* Subtle inner glow */
        box-shadow: inset 0 0 100px rgba(139, 92, 246, 0.02);
    }

    .main-content::-webkit-scrollbar {
        width: 8px;
    }

    .main-content::-webkit-scrollbar-track {
        background: rgba(5, 8, 16, 0.9);
        border-left: 1px solid rgba(139, 92, 246, 0.1);
    }

    .main-content::-webkit-scrollbar-thumb {
        background: linear-gradient(180deg, rgba(245, 87, 108, 0.5), rgba(139, 92, 246, 0.5));
        border-radius: 4px;
        border: 2px solid rgba(5, 8, 16, 0.9);
    }

    .main-content::-webkit-scrollbar-thumb:hover {
        background: linear-gradient(180deg, rgba(245, 87, 108, 0.8), rgba(139, 92, 246, 0.8));
        box-shadow: 0 0 10px rgba(245, 87, 108, 0.3);
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

    /* === TORBOX BADGES === */
    .torbox-badges {
        position: absolute;
        top: 8px;
        left: 8px;
        right: 8px;
        display: flex;
        gap: 4px;
        flex-wrap: wrap;
    }
    
    .torbox-badge {
        font-size: 0.7rem;
        padding: 3px 8px;
        background: rgba(0, 0, 0, 0.8);
        border-radius: 12px;
        backdrop-filter: blur(4px);
        font-weight: 600;
        color: #fff;
    }
    
    .torbox-badge.cached {
        background: linear-gradient(135deg, #10b981, #059669);
        color: #fff;
    }
    
    .torbox-badge.quality {
        background: linear-gradient(135deg, #663399, #9966cc);
        color: #fff;
    }
    
    .torbox-badge.seeds {
        background: rgba(34, 197, 94, 0.3);
        color: #22c55e;
        border: 1px solid rgba(34, 197, 94, 0.5);
    }
    
    .torbox-badge.variants {
        background: rgba(99, 102, 241, 0.3);
        color: #818cf8;
        border: 1px solid rgba(99, 102, 241, 0.5);
        font-weight: 600;
    }
    
    .card-source.torbox {
        color: #9966cc;
        font-weight: 600;
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

    /* TorBox File Styles */
    .episode-card.torbox-file {
        border-left: 3px solid #4ade80;
    }

    .episode-card.torbox-file .episode-number {
        color: #4ade80;
    }
    
    /* Episódio com múltiplas versões */
    .episode-card.has-versions {
        border-left: 3px solid #f59e0b;
    }
    
    .episode-card.has-versions .episode-number {
        color: #f59e0b;
    }
    
    /* Badge de contagem de versões */
    .version-count {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        margin-left: 6px;
        padding: 2px 6px;
        background: rgba(245, 158, 11, 0.2);
        border: 1px solid rgba(245, 158, 11, 0.4);
        border-radius: 10px;
        font-size: 0.7rem;
        color: #f59e0b;
        font-weight: 600;
    }
    
    /* Subtítulo do episódio */
    .ep-subtitle {
        display: block;
        font-size: 0.8rem;
        color: rgba(255, 255, 255, 0.5);
        margin-top: 4px;
        font-style: italic;
    }
    
    /* Título principal do episódio */
    .title-main {
        display: block;
        font-weight: 500;
        color: rgba(255, 255, 255, 0.9);
        line-height: 1.3;
    }
    
    .episode-card.selected .title-main {
        color: #fff;
    }
    
    /* Badges de versões disponíveis */
    .version-badges {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
        margin: 8px 0;
    }
    
    .q-badge.small {
        font-size: 0.65rem;
        padding: 2px 6px;
    }
    
    /* Seletor de versões expandido */
    .version-selector {
        margin-top: 12px;
        padding: 12px;
        background: rgba(0, 0, 0, 0.3);
        border-radius: 8px;
        border: 1px solid rgba(255, 255, 255, 0.1);
    }
    
    .version-label {
        display: block;
        font-size: 0.8rem;
        color: rgba(255, 255, 255, 0.7);
        margin-bottom: 8px;
    }
    
    .version-list {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }
    
    .version-option {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 6px;
        cursor: pointer;
        transition: all 0.2s ease;
        text-align: left;
        color: inherit;
        font-family: inherit;
        width: 100%;
    }
    
    .version-option:hover {
        background: rgba(255, 255, 255, 0.1);
        border-color: rgba(245, 87, 108, 0.4);
    }
    
    .version-option.selected {
        background: rgba(245, 87, 108, 0.15);
        border-color: #f5576c;
        box-shadow: 0 0 10px rgba(245, 87, 108, 0.2);
    }
    
    .v-quality {
        font-weight: 600;
        color: #4ade80;
        min-width: 50px;
    }
    
    .v-size {
        color: rgba(255, 255, 255, 0.6);
        font-size: 0.85rem;
    }
    
    .v-group {
        color: rgba(255, 255, 255, 0.4);
        font-size: 0.8rem;
    }
    
    .v-tag {
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 0.7rem;
        font-weight: 600;
    }
    
    .v-tag.repack {
        background: rgba(234, 179, 8, 0.2);
        color: #eab308;
    }

    .episode-size {
        font-size: 0.8rem;
        color: rgba(255, 255, 255, 0.5);
        margin-bottom: 6px;
    }

    .torbox-btn {
        background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%) !important;
    }

    .torbox-btn:hover {
        background: linear-gradient(135deg, #16a34a 0%, #15803d 100%) !important;
    }

    /* Player4K Button */
    .btn-play-4k {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 10px 16px;
        background: linear-gradient(135deg, #6366f1, #8b5cf6) !important;
        border: none;
        border-radius: 8px;
        color: white;
        font-weight: 600;
        font-size: 0.9rem;
        cursor: pointer;
        transition: all 0.2s ease;
        box-shadow: 0 2px 10px rgba(99, 102, 241, 0.3);
    }

    .btn-play-4k:hover {
        background: linear-gradient(135deg, #5558e3, #7c4fe8) !important;
        transform: translateY(-1px);
        box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);
    }

    /* Pipeline Button (GoFile Upload) */
    .btn-pipeline {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 10px 16px;
        background: linear-gradient(135deg, #f59e0b, #d97706) !important;
        border: none;
        border-radius: 8px;
        color: white;
        font-weight: 600;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.2s ease;
        box-shadow: 0 2px 10px rgba(245, 158, 11, 0.3);
    }

    .btn-pipeline:hover:not(:disabled) {
        background: linear-gradient(135deg, #d97706, #b45309) !important;
        transform: translateY(-1px);
        box-shadow: 0 4px 15px rgba(245, 158, 11, 0.4);
    }

    .btn-pipeline:disabled {
        opacity: 0.7;
        cursor: wait;
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
       FEATURED HERO ULTRA - Revolutionary Design
       ============================================ */
    .featured-hero-ultra {
        position: relative;
        min-height: 65vh;
        max-height: 75vh;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        perspective: 1000px;
    }
    
    /* Background Layers */
    .hero-bg-layer {
        position: absolute;
        inset: 0;
        pointer-events: none;
    }
    
    .bg-image {
        background-image: var(--banner-url);
        background-size: cover;
        background-position: center 20%;
        transform: scale(1.1);
        animation: heroBgZoom 20s ease-in-out infinite alternate;
    }
    
    @keyframes heroBgZoom {
        0% { transform: scale(1.1) translateX(0); }
        100% { transform: scale(1.15) translateX(-2%); }
    }
    
    .bg-blur {
        backdrop-filter: blur(2px);
        background: rgba(5, 8, 16, 0.3);
    }
    
    .bg-gradient {
        background: 
            linear-gradient(180deg, 
                rgba(5, 8, 16, 0.1) 0%,
                rgba(5, 8, 16, 0.4) 40%,
                rgba(5, 8, 16, 0.85) 70%,
                rgba(5, 8, 16, 1) 100%
            ),
            linear-gradient(90deg,
                rgba(5, 8, 16, 0.9) 0%,
                transparent 30%,
                transparent 70%,
                rgba(5, 8, 16, 0.6) 100%
            ),
            radial-gradient(ellipse at 20% 80%, var(--accent-color) 0%, transparent 50%);
        opacity: 0.9;
    }
    
    .bg-noise {
        background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noiseFilter'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noiseFilter)'/%3E%3C/svg%3E");
        opacity: 0.03;
        mix-blend-mode: overlay;
    }
    
    .bg-vignette {
        background: radial-gradient(ellipse at center, transparent 40%, rgba(0, 0, 0, 0.6) 100%);
    }
    
    /* Animated Particles */
    .hero-particles {
        position: absolute;
        inset: 0;
        overflow: hidden;
        pointer-events: none;
    }
    
    .particle {
        position: absolute;
        width: 4px;
        height: 4px;
        background: var(--accent-color);
        border-radius: 50%;
        opacity: 0.6;
        filter: blur(1px);
        box-shadow: 0 0 10px var(--accent-color), 0 0 20px var(--accent-color);
    }
    
    .particle.p1 { left: 10%; animation: floatParticle 8s ease-in-out infinite; animation-delay: 0s; }
    .particle.p2 { left: 25%; animation: floatParticle 12s ease-in-out infinite; animation-delay: 2s; }
    .particle.p3 { left: 50%; animation: floatParticle 10s ease-in-out infinite; animation-delay: 4s; }
    .particle.p4 { left: 75%; animation: floatParticle 14s ease-in-out infinite; animation-delay: 1s; }
    .particle.p5 { left: 90%; animation: floatParticle 9s ease-in-out infinite; animation-delay: 3s; }
    
    @keyframes floatParticle {
        0%, 100% { 
            transform: translateY(100vh) scale(0); 
            opacity: 0;
        }
        10% { opacity: 0.8; transform: translateY(80vh) scale(1); }
        90% { opacity: 0.8; }
        100% { 
            transform: translateY(-20vh) scale(0.5); 
            opacity: 0;
        }
    }
    
    /* Glowing Lines */
    .hero-glow-lines {
        position: absolute;
        inset: 0;
        pointer-events: none;
        overflow: hidden;
    }
    
    .glow-line {
        position: absolute;
        height: 1px;
        background: linear-gradient(90deg, transparent, var(--accent-color), transparent);
        opacity: 0.3;
        animation: glowLineMove 4s ease-in-out infinite;
    }
    
    .glow-line.gl1 { top: 20%; width: 60%; left: -30%; animation-delay: 0s; }
    .glow-line.gl2 { top: 50%; width: 80%; left: -40%; animation-delay: 1.5s; }
    .glow-line.gl3 { top: 80%; width: 50%; left: -25%; animation-delay: 3s; }
    
    @keyframes glowLineMove {
        0% { transform: translateX(0); opacity: 0; }
        50% { opacity: 0.5; }
        100% { transform: translateX(200%); opacity: 0; }
    }
    
    /* Main Content */
    .hero-main-content {
        position: relative;
        z-index: 10;
        width: 100%;
        max-width: 1400px;
        margin: 0 auto;
        padding: 40px 60px 80px;
        display: flex;
        gap: 60px;
        align-items: center;
        animation: heroContentReveal 0.8s cubic-bezier(0.4, 0, 0.2, 1) forwards;
    }
    
    @keyframes heroContentReveal {
        0% { 
            opacity: 0; 
            transform: translateY(40px);
        }
        100% { 
            opacity: 1; 
            transform: translateY(0);
        }
    }
    
    .hero-info-side {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 20px;
    }
    
    /* Ultra Badges */
    .hero-badges-ultra {
        display: flex;
        gap: 14px;
        flex-wrap: wrap;
        animation: badgesSlideIn 0.6s ease-out 0.2s backwards;
    }
    
    @keyframes badgesSlideIn {
        from { 
            opacity: 0;
            transform: translateX(-30px);
        }
        to { 
            opacity: 1;
            transform: translateX(0);
        }
    }
    
    .ultra-badge {
        position: relative;
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 18px;
        background: rgba(255, 255, 255, 0.08);
        backdrop-filter: blur(20px);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 30px;
        font-size: 0.85rem;
        font-weight: 600;
        color: #fff;
        overflow: hidden;
        transition: all 0.3s ease;
    }
    
    .ultra-badge:hover {
        transform: translateY(-3px);
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
    }
    
    .ultra-badge .badge-icon {
        font-size: 1.1rem;
    }
    
    .ultra-badge.score-badge {
        background: linear-gradient(135deg, rgba(255, 193, 7, 0.2), rgba(255, 152, 0, 0.15));
        border-color: rgba(255, 193, 7, 0.4);
        color: #ffd54f;
    }
    
    .ultra-badge.score-badge .badge-glow {
        position: absolute;
        inset: 0;
        background: radial-gradient(circle at 30% 50%, rgba(255, 193, 7, 0.3) 0%, transparent 70%);
        animation: badgeGlowPulse 2s ease-in-out infinite;
    }
    
    @keyframes badgeGlowPulse {
        0%, 100% { opacity: 0.5; }
        50% { opacity: 1; }
    }
    
    .ultra-badge.eps-badge {
        background: linear-gradient(135deg, rgba(139, 92, 246, 0.2), rgba(99, 102, 241, 0.15));
        border-color: rgba(139, 92, 246, 0.4);
        color: #a78bfa;
    }
    
    .ultra-badge.live-badge {
        background: linear-gradient(135deg, rgba(239, 68, 68, 0.3), rgba(220, 38, 38, 0.2));
        border-color: rgba(239, 68, 68, 0.5);
        color: #fca5a5;
        animation: liveBadgePulse 1.5s ease-in-out infinite;
    }
    
    @keyframes liveBadgePulse {
        0%, 100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4); }
        50% { box-shadow: 0 0 0 8px rgba(239, 68, 68, 0); }
    }
    
    .live-pulse {
        width: 10px;
        height: 10px;
        background: #ef4444;
        border-radius: 50%;
        animation: liveDotPulse 1s ease-in-out infinite;
    }
    
    @keyframes liveDotPulse {
        0%, 100% { transform: scale(1); opacity: 1; }
        50% { transform: scale(1.3); opacity: 0.7; }
    }
    
    /* Ultra Title */
    .hero-title-ultra {
        position: relative;
        margin: 0;
        animation: titleReveal 0.8s ease-out 0.3s backwards;
    }
    
    @keyframes titleReveal {
        from { 
            opacity: 0;
            transform: translateY(30px);
            filter: blur(10px);
        }
        to { 
            opacity: 1;
            transform: translateY(0);
            filter: blur(0);
        }
    }
    
    .title-text {
        font-size: clamp(2.2rem, 5vw, 4rem);
        font-weight: 800;
        letter-spacing: -1px;
        line-height: 1.1;
        background: linear-gradient(180deg, #ffffff 0%, rgba(255, 255, 255, 0.85) 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        text-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
        display: block;
    }
    
    .title-underline {
        display: block;
        width: 0;
        height: 4px;
        margin-top: 15px;
        background: linear-gradient(90deg, var(--accent-color), transparent);
        border-radius: 2px;
        animation: underlineExpand 0.8s ease-out 0.6s forwards;
    }
    
    @keyframes underlineExpand {
        to { width: 120px; }
    }
    
    /* Meta Ultra */
    .hero-meta-ultra {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 10px;
        animation: metaFadeIn 0.6s ease-out 0.4s backwards;
    }
    
    @keyframes metaFadeIn {
        from { opacity: 0; transform: translateY(15px); }
        to { opacity: 1; transform: translateY(0); }
    }
    
    .meta-genre, .meta-studio, .meta-year {
        color: rgba(255, 255, 255, 0.7);
        font-size: 0.95rem;
        font-weight: 500;
        transition: color 0.2s;
    }
    
    .meta-genre:hover {
        color: var(--accent-color);
    }
    
    .meta-dot {
        color: rgba(255, 255, 255, 0.3);
        font-size: 0.6rem;
    }
    
    .meta-studio {
        color: rgba(139, 92, 246, 0.9);
    }
    
    .meta-year {
        color: rgba(255, 255, 255, 0.5);
    }
    
    /* Description Ultra */
    .hero-desc-ultra {
        max-width: 550px;
        color: rgba(255, 255, 255, 0.65);
        font-size: 1rem;
        line-height: 1.7;
        margin: 0;
        animation: descFadeIn 0.6s ease-out 0.5s backwards;
    }
    
    @keyframes descFadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }
    
    /* Action Buttons Ultra */
    .hero-actions-ultra {
        display: flex;
        gap: 18px;
        margin-top: 10px;
        animation: actionsFadeIn 0.6s ease-out 0.6s backwards;
    }
    
    @keyframes actionsFadeIn {
        from { opacity: 0; transform: translateY(20px); }
        to { opacity: 1; transform: translateY(0); }
    }
    
    .btn-ultra-play {
        position: relative;
        padding: 0;
        background: transparent;
        border: none;
        cursor: pointer;
        overflow: hidden;
        border-radius: 14px;
    }
    
    .btn-ultra-play .btn-bg {
        position: absolute;
        inset: 0;
        background: linear-gradient(135deg, var(--accent-color) 0%, #f093fb 50%, #8b5cf6 100%);
        background-size: 200% 200%;
        animation: gradientShift 3s ease infinite;
        border-radius: 14px;
    }
    
    @keyframes gradientShift {
        0%, 100% { background-position: 0% 50%; }
        50% { background-position: 100% 50%; }
    }
    
    .btn-ultra-play .btn-content {
        position: relative;
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px 36px;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 700;
        letter-spacing: 0.5px;
        z-index: 1;
    }
    
    .btn-ultra-play .play-icon {
        width: 22px;
        height: 22px;
        filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3));
    }
    
    .btn-ultra-play .btn-shine {
        position: absolute;
        top: 0;
        left: -100%;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.4), transparent);
        transition: left 0.6s ease;
    }
    
    .btn-ultra-play:hover .btn-shine {
        left: 100%;
    }
    
    .btn-ultra-play:hover {
        transform: translateY(-4px) scale(1.02);
        box-shadow: 
            0 15px 40px rgba(245, 87, 108, 0.4),
            0 0 60px rgba(245, 87, 108, 0.2);
    }
    
    .btn-ultra-play:active {
        transform: translateY(-2px) scale(0.98);
    }
    
    .btn-ultra-trailer {
        position: relative;
        display: flex;
        align-items: center;
        padding: 16px 32px;
        background: rgba(255, 255, 255, 0.05);
        border-radius: 14px;
        text-decoration: none;
        overflow: hidden;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .btn-ultra-trailer .btn-border {
        position: absolute;
        inset: 0;
        border-radius: 14px;
        padding: 2px;
        background: linear-gradient(135deg, rgba(255, 255, 255, 0.3), rgba(255, 255, 255, 0.1));
        -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
        mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
        -webkit-mask-composite: xor;
        mask-composite: exclude;
    }
    
    .btn-ultra-trailer .btn-content {
        position: relative;
        display: flex;
        align-items: center;
        gap: 10px;
        color: #fff;
        font-size: 1rem;
        font-weight: 600;
        z-index: 1;
    }
    
    .btn-ultra-trailer .trailer-icon {
        font-size: 1.2rem;
    }
    
    .btn-ultra-trailer:hover {
        background: rgba(255, 255, 255, 0.12);
        transform: translateY(-4px);
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
    }
    
    /* Poster 3D Side */
    .hero-poster-side {
        flex-shrink: 0;
        perspective: 1000px;
        animation: posterReveal 0.8s cubic-bezier(0.4, 0, 0.2, 1) 0.4s backwards;
    }
    
    @keyframes posterReveal {
        from { 
            opacity: 0;
            transform: translateX(60px) rotateY(-15deg);
        }
        to { 
            opacity: 1;
            transform: translateX(0) rotateY(0);
        }
    }
    
    .poster-3d-container {
        position: relative;
        width: 240px;
        transform-style: preserve-3d;
        animation: posterFloat 4s ease-in-out infinite;
    }
    
    @keyframes posterFloat {
        0%, 100% { transform: translateY(0) rotateY(-5deg); }
        50% { transform: translateY(-15px) rotateY(5deg); }
    }
    
    .poster-reflection {
        position: absolute;
        bottom: -120px;
        left: 0;
        right: 0;
        height: 120px;
        background: linear-gradient(180deg, rgba(255, 255, 255, 0.08) 0%, transparent 100%);
        transform: scaleY(-1) rotateX(30deg);
        opacity: 0.3;
        border-radius: 16px;
        filter: blur(5px);
        pointer-events: none;
    }
    
    .poster-card {
        position: relative;
        border-radius: 16px;
        overflow: hidden;
        box-shadow: 
            0 30px 60px rgba(0, 0, 0, 0.5),
            0 0 100px rgba(var(--accent-color), 0.1);
    }
    
    .poster-card img {
        width: 100%;
        display: block;
        transition: transform 0.5s ease;
    }
    
    .poster-3d-container:hover .poster-card img {
        transform: scale(1.05);
    }
    
    .poster-shine {
        position: absolute;
        inset: 0;
        background: linear-gradient(
            135deg,
            transparent 40%,
            rgba(255, 255, 255, 0.1) 50%,
            transparent 60%
        );
        background-size: 200% 200%;
        animation: posterShineSweep 3s ease-in-out infinite;
    }
    
    @keyframes posterShineSweep {
        0% { background-position: 200% 200%; }
        100% { background-position: -100% -100%; }
    }
    
    .poster-border {
        position: absolute;
        inset: 0;
        border-radius: 16px;
        border: 2px solid rgba(255, 255, 255, 0.1);
        pointer-events: none;
    }
    
    .poster-shadow {
        position: absolute;
        bottom: -30px;
        left: 10%;
        right: 10%;
        height: 30px;
        background: radial-gradient(ellipse at center, rgba(0, 0, 0, 0.6) 0%, transparent 70%);
        filter: blur(15px);
    }
    
    /* Ultra Navigation */
    .hero-nav-ultra {
        position: absolute;
        bottom: 30px;
        left: 50%;
        transform: translateX(-50%);
        display: flex;
        align-items: center;
        gap: 12px;
        z-index: 20;
        padding: 12px 24px;
        background: rgba(0, 0, 0, 0.4);
        backdrop-filter: blur(20px);
        border-radius: 40px;
        border: 1px solid rgba(255, 255, 255, 0.1);
    }
    
    .nav-line {
        width: 30px;
        height: 1px;
        background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
    }
    
    .nav-dot-ultra {
        position: relative;
        width: 14px;
        height: 14px;
        padding: 0;
        background: transparent;
        border: none;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .nav-dot-ultra .dot-inner {
        position: absolute;
        inset: 4px;
        background: rgba(255, 255, 255, 0.3);
        border-radius: 50%;
        transition: all 0.3s ease;
    }
    
    .nav-dot-ultra .dot-ring {
        position: absolute;
        inset: 0;
        border: 2px solid transparent;
        border-radius: 50%;
        transition: all 0.3s ease;
    }
    
    .nav-dot-ultra:hover .dot-inner {
        background: rgba(255, 255, 255, 0.6);
        transform: scale(1.2);
    }
    
    .nav-dot-ultra.active .dot-inner {
        background: var(--accent-color);
        box-shadow: 0 0 15px var(--accent-color);
        inset: 2px;
    }
    
    .nav-dot-ultra.active .dot-ring {
        border-color: var(--accent-color);
        animation: ringPulse 1.5s ease-in-out infinite;
    }
    
    @keyframes ringPulse {
        0%, 100% { transform: scale(1); opacity: 1; }
        50% { transform: scale(1.4); opacity: 0; }
    }
    
    /* Legacy featured-hero for fallback */
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
        /* Ultra Hero Responsive */
        .featured-hero-ultra {
            min-height: 60vh;
            max-height: none;
        }
        
        .hero-main-content {
            flex-direction: column;
            padding: 30px 20px 100px;
            gap: 30px;
            text-align: center;
        }
        
        .hero-info-side {
            align-items: center;
        }
        
        .hero-badges-ultra {
            justify-content: center;
        }
        
        .hero-title-ultra {
            text-align: center;
        }
        
        .title-text {
            font-size: clamp(1.6rem, 7vw, 2.5rem);
        }
        
        .title-underline {
            margin-left: auto;
            margin-right: auto;
        }
        
        .hero-meta-ultra {
            justify-content: center;
        }
        
        .hero-desc-ultra {
            display: none;
        }
        
        .hero-actions-ultra {
            justify-content: center;
            flex-wrap: wrap;
        }
        
        .btn-ultra-play .btn-content,
        .btn-ultra-trailer .btn-content {
            padding: 14px 28px;
            font-size: 1rem;
        }
        
        .hero-poster-side {
            order: -1;
        }
        
        .poster-3d-container {
            width: 160px;
        }
        
        .hero-nav-ultra {
            bottom: 20px;
            padding: 10px 20px;
        }
        
        .hero-particles,
        .hero-glow-lines {
            display: none;
        }
        
        /* Legacy hero responsive */
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

    /* Search bar sticky - Legacy (now using .main-search-bar) */
    .search-bar-sticky {
        display: none; /* Replaced by .search-panel */
    }

    .results-header {
        display: none; /* Replaced by .results-info */
    }

    /* SEARCH SECTION - Legacy */
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

    .title-badge.torbox {
        background: linear-gradient(135deg, #ff6b9d 0%, #c44569 100%);
    }

    .title-badge.sources {
        background: linear-gradient(135deg, #4caf50 0%, #8bc34a 100%);
    }

    /* ============================================
       TORBOX GRID (Compact Cards)
       ============================================ */
    .torbox-section {
        padding: 20px 0;
    }

    /* ============================================
       MODERN ANIME CARDS - Premium Immersive Design
       ============================================ */
    .popular-section-modern {
        padding: 35px 0;
        position: relative;
    }
    
    .popular-section-modern::before {
        content: '';
        position: absolute;
        top: 0;
        left: -50px;
        right: -50px;
        height: 1px;
        background: linear-gradient(90deg, transparent, rgba(245, 87, 108, 0.3), rgba(139, 92, 246, 0.3), transparent);
    }
    
    .section-header-modern {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 30px;
        padding: 0 5px;
    }
    
    .section-title-area {
        display: flex;
        align-items: center;
        gap: 14px;
    }
    
    .section-title-area h2 {
        margin: 0;
        font-size: 1.9rem;
        font-weight: 800;
        background: linear-gradient(135deg, #fff 0%, #e0e7ff 50%, #a78bfa 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        letter-spacing: -0.5px;
    }
    
    .fire-icon-animated {
        font-size: 2.2rem;
        animation: fireFlicker 1.5s ease-in-out infinite;
        filter: drop-shadow(0 0 10px rgba(255, 100, 50, 0.5));
    }
    
    @keyframes fireFlicker {
        0%, 100% { transform: scale(1) rotate(0deg); filter: brightness(1) drop-shadow(0 0 10px rgba(255, 100, 50, 0.5)); }
        25% { transform: scale(1.1) rotate(-3deg); filter: brightness(1.2) drop-shadow(0 0 15px rgba(255, 100, 50, 0.7)); }
        50% { transform: scale(1.05) rotate(2deg); filter: brightness(1.1) drop-shadow(0 0 12px rgba(255, 100, 50, 0.6)); }
        75% { transform: scale(1.15) rotate(-2deg); filter: brightness(1.3) drop-shadow(0 0 18px rgba(255, 100, 50, 0.8)); }
    }
    
    .source-badge-modern {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 10px 20px;
        background: linear-gradient(135deg, rgba(102, 51, 153, 0.25) 0%, rgba(153, 102, 204, 0.15) 100%);
        border: 1px solid rgba(153, 102, 204, 0.35);
        border-radius: 30px;
        font-size: 0.88rem;
        font-weight: 700;
        color: #c9a8ff;
        backdrop-filter: blur(15px);
        transition: all 0.3s ease;
        cursor: default;
        text-transform: uppercase;
        letter-spacing: 1px;
    }
    
    .source-badge-modern:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 25px rgba(153, 102, 204, 0.25);
    }
    
    .source-badge-modern.vps {
        background: linear-gradient(135deg, rgba(99, 102, 241, 0.3) 0%, rgba(139, 92, 246, 0.2) 100%);
        border-color: rgba(139, 92, 246, 0.5);
        color: #a78bfa;
        box-shadow: 0 0 20px rgba(139, 92, 246, 0.15);
    }
    
    .source-badge-modern.vps .badge-dot {
        background: linear-gradient(135deg, #6366f1, #a78bfa);
        box-shadow: 0 0 12px #8b5cf6;
    }
    
    .source-badge-modern.anilist {
        background: linear-gradient(135deg, rgba(2, 169, 255, 0.25) 0%, rgba(59, 130, 246, 0.15) 100%);
        border-color: rgba(59, 130, 246, 0.45);
        color: #60a5fa;
        box-shadow: 0 0 20px rgba(59, 130, 246, 0.15);
    }
    
    .source-badge-modern.anilist .badge-dot {
        background: linear-gradient(135deg, #0ea5e9, #3b82f6);
        box-shadow: 0 0 12px #3b82f6;
    }
    
    .source-badge-modern.sources {
        background: linear-gradient(135deg, rgba(16, 185, 129, 0.25) 0%, rgba(34, 197, 94, 0.15) 100%);
        border-color: rgba(34, 197, 94, 0.45);
        color: #4ade80;
        box-shadow: 0 0 20px rgba(34, 197, 94, 0.15);
    }
    
    .source-badge-modern.sources .badge-dot {
        background: linear-gradient(135deg, #10b981, #22c55e);
        box-shadow: 0 0 12px #22c55e;
    }
    
    .source-badge-modern .badge-dot {
        width: 10px;
        height: 10px;
        background: linear-gradient(135deg, #9966cc, #c084fc);
        border-radius: 50%;
        animation: badgePulse 2s ease-in-out infinite;
        box-shadow: 0 0 12px #9966cc;
    }
    
    @keyframes badgePulse {
        0%, 100% { transform: scale(1); opacity: 1; }
        50% { transform: scale(1.3); opacity: 0.7; }
    }
    
    .anime-grid-modern {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(175px, 1fr));
        gap: 24px;
        padding: 15px 0;
    }
    
    .anime-card-modern {
        position: relative;
        background: transparent;
        border: none;
        padding: 0;
        cursor: pointer;
        text-align: left;
        border-radius: 20px;
        overflow: visible;
        animation: cardEntrance 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) both;
        animation-delay: var(--delay);
        transform-style: preserve-3d;
        perspective: 1000px;
    }
    
    .anime-card-modern:hover {
        z-index: 10;
    }
    
    @keyframes cardEntrance {
        from {
            opacity: 0;
            transform: translateY(40px) scale(0.9) rotateX(10deg);
        }
        to {
            opacity: 1;
            transform: translateY(0) scale(1) rotateX(0);
        }
    }
    
    .anime-card-modern .card-image-wrapper {
        position: relative;
        aspect-ratio: 2/3;
        border-radius: 20px;
        overflow: hidden;
        background: linear-gradient(145deg, #1a1f3a 0%, #0d1226 100%);
        box-shadow: 
            0 4px 20px rgba(0, 0, 0, 0.3),
            0 0 0 1px rgba(255, 255, 255, 0.05);
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .anime-card-modern:hover .card-image-wrapper {
        transform: translateY(-8px) scale(1.02);
        box-shadow: 
            0 20px 50px rgba(0, 0, 0, 0.5),
            0 0 0 1px rgba(255, 255, 255, 0.1),
            0 0 60px rgba(245, 87, 108, 0.15);
    }
    
    .anime-card-modern .card-image-wrapper img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transition: all 0.6s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .anime-card-modern:hover .card-image-wrapper img {
        transform: scale(1.15);
        filter: brightness(0.6) saturate(1.2);
    }
    
    /* Shine Effect - More Dynamic */
    .card-shine {
        position: absolute;
        top: 0;
        left: -150%;
        width: 100%;
        height: 100%;
        background: linear-gradient(
            120deg,
            transparent 30%,
            rgba(255, 255, 255, 0.15) 50%,
            transparent 70%
        );
        transform: skewX(-20deg);
        transition: left 0.8s cubic-bezier(0.4, 0, 0.2, 1);
        pointer-events: none;
    }
    
    .anime-card-modern:hover .card-shine {
        left: 150%;
    }
    
    /* Gradient Overlay - More Cinematic */
    .card-gradient-overlay {
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        height: 70%;
        background: linear-gradient(
            to top, 
            rgba(0, 0, 0, 0.95) 0%, 
            rgba(0, 0, 0, 0.6) 40%,
            transparent 100%
        );
        opacity: 0;
        transition: opacity 0.4s ease;
        pointer-events: none;
    }
    
    .anime-card-modern:hover .card-gradient-overlay {
        opacity: 1;
    }
    
    /* Score Badge - Premium Look */
    .score-badge-modern {
        position: absolute;
        top: 12px;
        right: 12px;
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 7px 12px;
        background: linear-gradient(135deg, rgba(0, 0, 0, 0.8) 0%, rgba(30, 30, 40, 0.9) 100%);
        backdrop-filter: blur(15px);
        border-radius: 25px;
        border: 1px solid rgba(255, 215, 0, 0.25);
        z-index: 5;
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
    }
    
    .anime-card-modern:hover .score-badge-modern {
        background: linear-gradient(135deg, rgba(255, 215, 0, 0.25) 0%, rgba(255, 165, 0, 0.15) 100%);
        border-color: rgba(255, 215, 0, 0.7);
        transform: scale(1.08) translateY(-2px);
        box-shadow: 0 8px 25px rgba(255, 215, 0, 0.25);
    }
    
    .score-badge-modern .score-star {
        color: #ffd700;
        font-size: 0.95rem;
        filter: drop-shadow(0 0 3px rgba(255, 215, 0, 0.5));
    }
    
    .score-badge-modern .score-value {
        color: #fff;
        font-weight: 800;
        font-size: 0.9rem;
        letter-spacing: 0.5px;
    }
    
    /* Hover Overlay - Cinematic */
    .card-hover-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 12px;
        opacity: 0;
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        z-index: 4;
        background: radial-gradient(circle at center, rgba(0,0,0,0.2) 0%, transparent 70%);
    }
    
    .anime-card-modern:hover .card-hover-overlay {
        opacity: 1;
    }
    
    .play-button-modern {
        width: 65px;
        height: 65px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 50%, #4facfe 100%);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 
            0 10px 40px rgba(245, 87, 108, 0.5),
            0 0 0 4px rgba(255, 255, 255, 0.1),
            inset 0 0 20px rgba(255, 255, 255, 0.2);
        transform: scale(0.7) rotate(-10deg);
        transition: all 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
    }
    
    .anime-card-modern:hover .play-button-modern {
        transform: scale(1) rotate(0deg);
    }
    
    .play-button-modern svg {
        width: 30px;
        height: 30px;
        color: #fff;
        margin-left: 4px;
        filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3));
    }
    
    .watch-text {
        color: #fff;
        font-size: 0.85rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 3px;
        opacity: 0;
        transform: translateY(15px);
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1) 0.1s;
        text-shadow: 0 2px 10px rgba(0, 0, 0, 0.5);
    }
    
    .anime-card-modern:hover .watch-text {
        opacity: 1;
        transform: translateY(0);
    }
    
    /* Card Content - Premium Style */
    .card-content-modern {
        padding: 14px 6px 8px;
    }
    
    .card-title-modern {
        margin: 0 0 8px 0;
        font-size: 0.95rem;
        font-weight: 700;
        color: #fff;
        line-height: 1.35;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
        transition: all 0.3s ease;
        letter-spacing: 0.2px;
    }
    
    .anime-card-modern:hover .card-title-modern {
        color: #f5576c;
        text-shadow: 0 0 20px rgba(245, 87, 108, 0.3);
    }
    
    .card-meta-modern {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }
    
    .card-meta-modern .meta-item {
        display: flex;
        align-items: center;
        gap: 5px;
        font-size: 0.78rem;
        color: #9ca3af;
        padding: 3px 8px;
        background: rgba(255, 255, 255, 0.05);
        border-radius: 6px;
        transition: all 0.3s ease;
    }
    
    .anime-card-modern:hover .card-meta-modern .meta-item {
        background: rgba(245, 87, 108, 0.1);
        color: #f0abfc;
    }
    
    .card-meta-modern .meta-icon {
        font-size: 0.8rem;
    }
    
    .card-meta-modern .meta-item.studio {
        color: #666;
    }
    
    .card-meta-modern .meta-item.source-name {
        color: #888;
        font-size: 0.7rem;
    }
    
    /* Card Badges Modern */
    .card-badges-modern {
        position: absolute;
        top: 10px;
        left: 10px;
        display: flex;
        flex-direction: column;
        gap: 6px;
        z-index: 2;
    }
    
    .status-badge {
        display: flex;
        align-items: center;
        gap: 5px;
        padding: 4px 10px;
        border-radius: 15px;
        font-size: 0.65rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    
    .status-badge.airing {
        background: linear-gradient(135deg, rgba(239, 68, 68, 0.9) 0%, rgba(220, 38, 38, 0.9) 100%);
        color: #fff;
        box-shadow: 0 2px 10px rgba(239, 68, 68, 0.4);
    }
    
    .status-badge .live-dot {
        width: 6px;
        height: 6px;
        background: #fff;
        border-radius: 50%;
        animation: livePulse 1.5s ease-in-out infinite;
    }
    
    @keyframes livePulse {
        0%, 100% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.5; transform: scale(0.8); }
    }
    
    /* Source Flag Badge */
    .source-flag-badge {
        position: absolute;
        top: 10px;
        right: 10px;
        width: 32px;
        height: 32px;
        background: rgba(0, 0, 0, 0.7);
        backdrop-filter: blur(10px);
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1.1rem;
        z-index: 2;
        border: 1px solid rgba(255, 255, 255, 0.1);
        transition: all 0.3s ease;
    }
    
    .source-flag-badge.en {
        border-color: rgba(74, 144, 217, 0.4);
    }
    
    .source-flag-badge.pt {
        border-color: rgba(34, 197, 94, 0.4);
    }
    
    .anime-card-modern:hover .source-flag-badge {
        transform: scale(1.1);
    }
    
    /* No Image Placeholder */
    .no-image-modern {
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 4rem;
        background: linear-gradient(135deg, #1a1f3a 0%, #0a0e27 100%);
        color: #444;
    }
    
    /* Glow Effect - Enhanced for immersive feel */
    .card-glow {
        position: absolute;
        inset: -4px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 30%, #8b5cf6 60%, #4facfe 100%);
        border-radius: 20px;
        z-index: -1;
        opacity: 0;
        filter: blur(20px);
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        animation: glowPulse 3s ease-in-out infinite;
        animation-play-state: paused;
    }
    
    @keyframes glowPulse {
        0%, 100% { opacity: 0.4; filter: blur(20px); }
        50% { opacity: 0.7; filter: blur(25px); }
    }
    
    .anime-card-modern:hover .card-glow {
        opacity: 0.6;
        animation-play-state: running;
        inset: -8px;
    }
    
    /* VPS Card Special Glow */
    .anime-card-modern.vps-source .card-glow {
        background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #a855f7 100%);
    }
    
    /* Card Border Gradient Effect */
    .anime-card-modern::before {
        content: '';
        position: absolute;
        inset: 0;
        border-radius: 16px;
        padding: 1.5px;
        background: linear-gradient(135deg, rgba(245, 87, 108, 0.3), rgba(139, 92, 246, 0.3));
        -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
        mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
        -webkit-mask-composite: xor;
        mask-composite: exclude;
        opacity: 0;
        transition: opacity 0.3s ease;
        pointer-events: none;
    }
    
    .anime-card-modern:hover::before {
        opacity: 1;
    }

    /* ============================================
       TORBOX VPS GRID - Premium Immersive Design
       ============================================ */
    .torbox-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(175px, 1fr));
        gap: 24px;
        padding: 15px 0;
    }

    .torbox-card {
        position: relative;
        background: linear-gradient(145deg, rgba(20, 22, 34, 0.95), rgba(30, 32, 48, 0.9));
        border-radius: 16px;
        overflow: visible;
        cursor: pointer;
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        border: 1px solid rgba(139, 92, 246, 0.2);
        text-align: left;
        backdrop-filter: blur(10px);
    }
    
    .torbox-card::before {
        content: '';
        position: absolute;
        inset: -2px;
        background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #a855f7 100%);
        border-radius: 18px;
        z-index: -1;
        opacity: 0;
        filter: blur(15px);
        transition: all 0.4s ease;
    }

    .torbox-card:hover {
        transform: translateY(-10px) scale(1.02);
        box-shadow: 
            0 25px 50px rgba(139, 92, 246, 0.25),
            0 0 80px rgba(139, 92, 246, 0.15),
            inset 0 1px 0 rgba(255, 255, 255, 0.1);
        border-color: rgba(139, 92, 246, 0.6);
    }
    
    .torbox-card:hover::before {
        opacity: 0.5;
    }

    .torbox-card:active {
        transform: translateY(-5px) scale(1.01);
    }

    .torbox-poster {
        position: relative;
        aspect-ratio: 3/4;
        overflow: hidden;
        border-radius: 16px 16px 0 0;
    }

    .torbox-poster img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transition: all 0.5s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .torbox-card:hover .torbox-poster img {
        transform: scale(1.1);
        filter: brightness(1.1);
    }

    .torbox-overlay {
        position: absolute;
        inset: 0;
        background: linear-gradient(
            0deg,
            rgba(0, 0, 0, 0.9) 0%,
            rgba(139, 92, 246, 0.2) 50%,
            rgba(0, 0, 0, 0.4) 100%
        );
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.4s ease;
    }

    .torbox-card:hover .torbox-overlay {
        opacity: 1;
    }

    .torbox-overlay .play-btn {
        font-size: 3.5rem;
        color: #fff;
        text-shadow: 0 4px 20px rgba(139, 92, 246, 0.8);
        transform: scale(0.8);
        transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        filter: drop-shadow(0 0 20px rgba(139, 92, 246, 0.6));
    }
    
    .torbox-card:hover .torbox-overlay .play-btn {
        transform: scale(1);
    }

    .torbox-score {
        position: absolute;
        top: 10px;
        right: 10px;
        background: linear-gradient(135deg, rgba(0, 0, 0, 0.9), rgba(139, 92, 246, 0.3));
        padding: 5px 10px;
        border-radius: 8px;
        font-size: 0.75rem;
        font-weight: 700;
        color: #ffd700;
        backdrop-filter: blur(10px);
        border: 1px solid rgba(255, 215, 0, 0.3);
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
    }
    
    .torbox-score::before {
        content: '⭐';
        margin-right: 4px;
    }

    .torbox-info {
        padding: 12px 14px;
        background: linear-gradient(180deg, rgba(20, 22, 34, 0.95), rgba(15, 17, 28, 0.98));
    }

    .torbox-title {
        font-size: 0.9rem;
        font-weight: 700;
        color: #fff;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        margin-bottom: 4px;
        letter-spacing: 0.3px;
    }

    .torbox-meta {
        font-size: 0.75rem;
        color: rgba(139, 92, 246, 0.9);
        font-weight: 500;
        display: flex;
        align-items: center;
        gap: 6px;
    }
    
    .torbox-meta::before {
        content: '📦';
        font-size: 0.7rem;
    }

    /* ============================================
       ANIME ROW (Horizontal Scroll with Arrows)
       Premium Immersive Design
       ============================================ */
    .anime-row {
        display: flex;
        gap: 28px;
        overflow-x: auto;
        padding: 20px 10px 30px;
        scroll-snap-type: x mandatory;
        scroll-behavior: smooth;
        /* Fade edges with gradient */
        mask-image: linear-gradient(90deg, transparent 0%, black 2%, black 98%, transparent 100%);
        -webkit-mask-image: linear-gradient(90deg, transparent 0%, black 2%, black 98%, transparent 100%);
    }

    .anime-row::-webkit-scrollbar {
        height: 6px;
    }

    .anime-row::-webkit-scrollbar-track {
        background: rgba(255, 255, 255, 0.03);
        border-radius: 10px;
    }

    .anime-row::-webkit-scrollbar-thumb {
        background: linear-gradient(90deg, rgba(245, 87, 108, 0.6), rgba(139, 92, 246, 0.6));
        border-radius: 10px;
        box-shadow: 0 0 10px rgba(245, 87, 108, 0.3);
    }

    .anime-row::-webkit-scrollbar-thumb:hover {
        background: linear-gradient(90deg, rgba(245, 87, 108, 1), rgba(139, 92, 246, 1));
        box-shadow: 0 0 15px rgba(245, 87, 108, 0.5);
    }

    /* ============================================
       ANIME CARD HD (AniList Style - Enhanced)
       ============================================ */
    .anime-card-hd {
        flex-shrink: 0;
        width: 210px;
        cursor: pointer;
        scroll-snap-align: start;
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        position: relative;
    }
    
    .anime-card-hd::before {
        content: '';
        position: absolute;
        inset: -4px;
        background: linear-gradient(135deg, #f5576c 0%, #8b5cf6 50%, #4facfe 100%);
        border-radius: 16px;
        z-index: -1;
        opacity: 0;
        filter: blur(20px);
        transition: opacity 0.4s ease;
    }

    .anime-card-hd:hover {
        transform: translateY(-12px) scale(1.03);
    }
    
    .anime-card-hd:hover::before {
        opacity: 0.5;
    }

    .card-poster-hd {
        position: relative;
        aspect-ratio: 3/4;
        border-radius: 14px;
        overflow: hidden;
        background: var(--card-color);
        box-shadow: 0 8px 30px rgba(0, 0, 0, 0.4);
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .anime-card-hd:hover .card-poster-hd {
        box-shadow: 
            0 20px 50px rgba(245, 87, 108, 0.25),
            0 0 60px rgba(245, 87, 108, 0.1);
    }

    .card-poster-hd img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transition: all 0.5s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .anime-card-hd:hover .card-poster-hd img {
        transform: scale(1.12);
        filter: brightness(1.05);
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
        padding: 12px 16px;
        background: transparent;
        position: sticky;
        top: 0;
        z-index: 50;
        animation: fadeDown 0.5s ease-out;
    }
    
    @keyframes fadeDown {
        from { opacity: 0; transform: translateY(-20px); }
        to { opacity: 1; transform: translateY(0); }
    }
    
    .nav-tabs {
        display: flex;
        gap: 4px;
        padding: 5px;
        background: rgba(15, 20, 45, 0.85);
        backdrop-filter: blur(20px);
        border-radius: 30px;
        border: 1px solid rgba(255, 255, 255, 0.08);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    }
    
    .nav-tab {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 12px 24px;
        background: transparent;
        border: none;
        border-radius: 25px;
        color: rgba(255, 255, 255, 0.55);
        font-size: 0.9rem;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        position: relative;
        overflow: hidden;
    }
    
    .nav-tab::before {
        content: '';
        position: absolute;
        inset: 0;
        background: linear-gradient(135deg, rgba(255,255,255,0.1) 0%, transparent 100%);
        opacity: 0;
        transition: opacity 0.3s;
    }
    
    .nav-tab:hover {
        color: #fff;
        transform: translateY(-1px);
    }
    
    .nav-tab:hover::before {
        opacity: 1;
    }
    
    .nav-tab.active {
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        color: #fff;
        box-shadow: 0 4px 20px rgba(245, 87, 108, 0.5), 
                    inset 0 1px 0 rgba(255,255,255,0.2);
        transform: scale(1.02);
    }
    
    .nav-tab.active::before {
        opacity: 0;
    }
    
    .tab-icon {
        font-size: 1.1rem;
        filter: drop-shadow(0 2px 4px rgba(0,0,0,0.2));
    }
    
    .tab-text {
        font-weight: 600;
        letter-spacing: 0.3px;
    }
    
    .tab-badge {
        font-size: 0.65rem;
        padding: 3px 8px;
        border-radius: 10px;
        font-weight: 700;
        animation: pulse 2s ease-in-out infinite;
    }
    
    .tab-badge.notify {
        background: #fff;
        color: #f5576c;
        box-shadow: 0 2px 8px rgba(245, 87, 108, 0.4);
    }
    
    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.08); }
    }

    /* ============================================
       SEARCH PANEL - Modern & Clean
       ============================================ */
    .search-panel {
        max-width: 900px;
        margin: 0 auto 24px;
        padding: 0 16px;
        animation: fadeUp 0.6s ease-out 0.1s both;
    }
    
    @keyframes fadeUp {
        from { opacity: 0; transform: translateY(20px); }
        to { opacity: 1; transform: translateY(0); }
    }
    
    /* Source Toggle Row */
    .source-toggle-row {
        display: flex;
        justify-content: center;
        margin-bottom: 20px;
        animation: fadeUp 0.5s ease-out 0.15s both;
    }
    
    .source-toggle {
        display: inline-flex;
        align-items: center;
        gap: 12px;
        padding: 8px 10px 8px 18px;
        background: linear-gradient(135deg, rgba(20, 25, 55, 0.8) 0%, rgba(30, 35, 65, 0.8) 100%);
        border-radius: 30px;
        border: 1px solid rgba(255, 255, 255, 0.08);
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
        backdrop-filter: blur(15px);
    }
    
    .toggle-icon {
        font-size: 1rem;
        animation: float 3s ease-in-out infinite;
    }
    
    @keyframes float {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-3px); }
    }
    
    .toggle-label {
        font-size: 0.85rem;
        font-weight: 600;
        color: rgba(255, 255, 255, 0.7);
        letter-spacing: 0.5px;
    }
    
    .toggle-buttons {
        display: flex;
        gap: 6px;
    }
    
    .toggle-btn {
        padding: 8px 16px;
        background: rgba(255, 255, 255, 0.04);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 20px;
        color: rgba(255, 255, 255, 0.5);
        font-size: 0.82rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        position: relative;
        overflow: hidden;
    }
    
    .toggle-btn::before {
        content: '';
        position: absolute;
        inset: 0;
        background: linear-gradient(135deg, transparent 0%, rgba(255,255,255,0.1) 100%);
        opacity: 0;
        transition: opacity 0.3s;
    }
    
    .toggle-btn:hover {
        background: rgba(255, 255, 255, 0.08);
        color: #fff;
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
    }
    
    .toggle-btn:hover::before {
        opacity: 1;
    }
    
    .toggle-btn.active {
        background: linear-gradient(135deg, rgba(245, 87, 108, 0.25) 0%, rgba(240, 147, 251, 0.25) 100%);
        border-color: rgba(245, 87, 108, 0.5);
        color: #f5576c;
        font-weight: 600;
        box-shadow: 0 4px 15px rgba(245, 87, 108, 0.2),
                    inset 0 0 20px rgba(245, 87, 108, 0.1);
    }
    
    .toggle-btn.vps {
        background: linear-gradient(135deg, rgba(138, 43, 226, 0.15) 0%, rgba(75, 0, 130, 0.15) 100%);
        border-color: rgba(138, 43, 226, 0.3);
    }
    
    .toggle-btn.vps.active {
        background: linear-gradient(135deg, rgba(138, 43, 226, 0.35) 0%, rgba(186, 85, 211, 0.35) 100%);
        border-color: rgba(186, 85, 211, 0.6);
        color: #da70d6;
        box-shadow: 0 4px 20px rgba(138, 43, 226, 0.3),
                    inset 0 0 20px rgba(138, 43, 226, 0.15);
    }
    
    /* Main Search Bar */
    .main-search-bar {
        display: flex;
        align-items: center;
        gap: 8px;
        background: linear-gradient(135deg, rgba(18, 22, 50, 0.95) 0%, rgba(25, 30, 60, 0.95) 100%);
        border: 2px solid rgba(255, 255, 255, 0.08);
        border-radius: 60px;
        padding: 6px;
        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.25),
                    inset 0 1px 0 rgba(255, 255, 255, 0.05);
        animation: fadeUp 0.5s ease-out 0.2s both;
    }
    
    .main-search-bar:focus-within {
        border-color: rgba(245, 87, 108, 0.6);
        box-shadow: 0 8px 40px rgba(0, 0, 0, 0.3),
                    0 0 0 4px rgba(245, 87, 108, 0.15),
                    inset 0 1px 0 rgba(255, 255, 255, 0.08);
        transform: translateY(-2px);
    }
    
    .search-input-container {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 0 20px;
    }
    
    .search-icon-svg {
        width: 22px;
        height: 22px;
        color: rgba(255, 255, 255, 0.35);
        flex-shrink: 0;
        transition: all 0.3s;
    }
    
    .main-search-bar:focus-within .search-icon-svg {
        color: #f5576c;
        transform: scale(1.1);
    }
    
    .search-input-modern {
        flex: 1;
        background: transparent;
        border: none;
        outline: none;
        color: #fff;
        font-size: 1.05rem;
        padding: 14px 0;
        font-weight: 400;
    }
    
    .search-input-modern::placeholder {
        color: rgba(255, 255, 255, 0.3);
        font-weight: 400;
    }
    
    .clear-search {
        width: 28px;
        height: 28px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(255, 255, 255, 0.08);
        border: none;
        border-radius: 50%;
        color: rgba(255, 255, 255, 0.5);
        cursor: pointer;
        transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .clear-search:hover {
        background: rgba(245, 87, 108, 0.3);
        color: #fff;
        transform: rotate(90deg) scale(1.1);
    }
    
    .clear-search svg {
        width: 14px;
        height: 14px;
    }
    
    .search-submit {
        padding: 14px 32px;
        background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
        border: none;
        border-radius: 30px;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 700;
        letter-spacing: 0.5px;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        min-width: 110px;
        display: flex;
        align-items: center;
        justify-content: center;
        position: relative;
        overflow: hidden;
        box-shadow: 0 4px 15px rgba(245, 87, 108, 0.4);
    }
    
    .search-submit::before {
        content: '';
        position: absolute;
        top: 0;
        left: -100%;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(255,255,255,0.3), transparent);
        transition: left 0.5s;
    }
    
    .search-submit:hover:not(:disabled) {
        transform: translateY(-3px) scale(1.02);
        box-shadow: 0 8px 30px rgba(245, 87, 108, 0.5);
    }
    
    .search-submit:hover::before {
        left: 100%;
    }
    
    .search-submit:active:not(:disabled) {
        transform: translateY(-1px) scale(0.98);
    }
    
    .search-submit:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }
    
    .loading-spinner {
        width: 20px;
        height: 20px;
        border: 2.5px solid rgba(255, 255, 255, 0.2);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.7s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    /* Quick Search Row */
    .quick-search-row {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 14px;
        margin-top: 20px;
        flex-wrap: wrap;
        animation: fadeUp 0.5s ease-out 0.25s both;
    }
    
    .row-label {
        font-size: 0.82rem;
        color: rgba(255, 255, 255, 0.4);
        font-weight: 500;
        font-style: italic;
    }
    
    .quick-tags {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
        justify-content: center;
    }
    
    .quick-tag {
        padding: 8px 18px;
        background: rgba(255, 255, 255, 0.04);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        color: rgba(255, 255, 255, 0.7);
        font-size: 0.88rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        position: relative;
        overflow: hidden;
    }
    
    .quick-tag::before {
        content: '';
        position: absolute;
        inset: 0;
        background: linear-gradient(135deg, rgba(245, 87, 108, 0.2) 0%, rgba(240, 147, 251, 0.2) 100%);
        opacity: 0;
        transition: opacity 0.3s;
    }
    
    .quick-tag:hover {
        border-color: rgba(245, 87, 108, 0.4);
        color: #fff;
        transform: translateY(-3px) scale(1.05);
        box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
    }
    
    .quick-tag:hover::before {
        opacity: 1;
    }
    
    /* Genre Section */
    .genre-section {
        margin-top: 24px;
        text-align: center;
        animation: fadeUp 0.5s ease-out 0.3s both;
    }
    
    .genre-section .row-label {
        display: block;
        margin-bottom: 16px;
        font-size: 0.85rem;
    }
    
    .genre-grid {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        justify-content: center;
        max-width: 850px;
        margin: 0 auto;
    }
    
    .genre-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 18px;
        background: linear-gradient(135deg, rgba(30, 35, 60, 0.6) 0%, rgba(40, 45, 70, 0.6) 100%);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 25px;
        color: rgba(255, 255, 255, 0.75);
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
        position: relative;
        overflow: hidden;
        backdrop-filter: blur(10px);
    }
    
    .genre-btn::before {
        content: '';
        position: absolute;
        inset: 0;
        background: linear-gradient(135deg, rgba(245, 87, 108, 0.25) 0%, rgba(240, 147, 251, 0.25) 100%);
        opacity: 0;
        transition: opacity 0.35s;
    }
    
    .genre-btn::after {
        content: '';
        position: absolute;
        top: 50%;
        left: 50%;
        width: 0;
        height: 0;
        background: rgba(255, 255, 255, 0.2);
        border-radius: 50%;
        transform: translate(-50%, -50%);
        transition: width 0.5s, height 0.5s;
    }
    
    .genre-btn:hover {
        border-color: rgba(245, 87, 108, 0.4);
        color: #fff;
        transform: translateY(-4px) scale(1.03);
        box-shadow: 0 8px 25px rgba(0, 0, 0, 0.25),
                    0 0 20px rgba(245, 87, 108, 0.15);
    }
    
    .genre-btn:hover::before {
        opacity: 1;
    }
    
    .genre-btn:active::after {
        width: 200px;
        height: 200px;
    }
    
    .genre-icon {
        font-size: 1.15rem;
        filter: drop-shadow(0 2px 4px rgba(0,0,0,0.3));
        transition: transform 0.3s;
    }
    
    .genre-btn:hover .genre-icon {
        transform: scale(1.2) rotate(-5deg);
    }
    
    .genre-name {
        font-weight: 600;
        letter-spacing: 0.3px;
        position: relative;
        z-index: 1;
    }
    
    /* Results Info */
    .results-info {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 16px;
        margin-top: 20px;
        padding: 14px 24px;
        background: linear-gradient(135deg, rgba(20, 25, 55, 0.7) 0%, rgba(30, 35, 65, 0.7) 100%);
        border-radius: 16px;
        border: 1px solid rgba(255, 255, 255, 0.06);
        animation: fadeUp 0.4s ease-out;
        backdrop-filter: blur(10px);
    }
    
    .active-filter {
        display: flex;
        align-items: center;
        gap: 12px;
    }
    
    .filter-badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 8px 16px;
        background: linear-gradient(135deg, rgba(245, 87, 108, 0.25) 0%, rgba(240, 147, 251, 0.25) 100%);
        border-radius: 20px;
        color: #f5576c;
        font-weight: 700;
        font-size: 0.92rem;
        border: 1px solid rgba(245, 87, 108, 0.3);
        box-shadow: 0 4px 12px rgba(245, 87, 108, 0.2);
    }
    
    .filter-count {
        color: rgba(255, 255, 255, 0.6);
        font-size: 0.92rem;
    }
    
    .search-results-text {
        color: rgba(255, 255, 255, 0.7);
        font-size: 0.92rem;
    }
    
    .clear-filter-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 20px;
        background: rgba(255, 80, 80, 0.12);
        border: 1px solid rgba(255, 80, 80, 0.25);
        border-radius: 20px;
        color: #ff6b6b;
        font-size: 0.88rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .clear-filter-btn:hover {
        background: rgba(255, 80, 80, 0.25);
        transform: translateY(-2px);
        box-shadow: 0 4px 15px rgba(255, 80, 80, 0.2);
    }
    
    .clear-filter-btn svg {
        width: 14px;
        height: 14px;
        transition: transform 0.3s;
    }
    
    .clear-filter-btn:hover svg {
        transform: rotate(90deg);
    }
    
    /* Responsive */
    @media (max-width: 640px) {
        .nav-tabs {
            gap: 3px;
            padding: 4px;
        }
        
        .nav-tab {
            padding: 10px 14px;
            font-size: 0.85rem;
        }
        
        .tab-text {
            display: none;
        }
        
        .tab-icon {
            font-size: 1.2rem;
        }
        
        .source-toggle {
            flex-wrap: wrap;
            justify-content: center;
            padding: 10px 14px;
            gap: 8px;
        }
        
        .toggle-buttons {
            flex-wrap: wrap;
            justify-content: center;
        }
        
        .toggle-btn {
            padding: 6px 12px;
            font-size: 0.78rem;
        }
        
        .main-search-bar {
            flex-direction: column;
            border-radius: 24px;
            padding: 10px;
        }
        
        .search-input-container {
            width: 100%;
            padding: 6px 14px;
        }
        
        .search-submit {
            width: 100%;
            border-radius: 18px;
            padding: 12px;
        }
        
        .genre-btn {
            padding: 8px 14px;
            font-size: 0.8rem;
        }
        
        .genre-icon {
            font-size: 1rem;
        }
        
        .quick-tag {
            padding: 6px 14px;
            font-size: 0.82rem;
        }
    }

    /* ============================================
       TAB CONTENT STYLES
       ============================================ */
    .tab-content {
        padding: 24px 16px;
        max-width: 1200px;
        margin: 0 auto;
    }

    /* ============================================
       MANGA TAB STYLES
       ============================================ */
    
    /* === ANIME SOURCE SELECTOR (Legacy - redirect to new) === */
    .anime-source-selector {
        display: none; /* Replaced by .source-toggle */
    }
    
    /* === MANGA SOURCE SELECTOR === */
    .manga-source-selector {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;
        padding: 12px 16px;
        background: rgba(30, 33, 48, 0.6);
        border-radius: 10px;
        border: 1px solid rgba(102, 51, 153, 0.2);
        backdrop-filter: blur(8px);
    }
    
    .source-label {
        font-weight: 600;
        color: #fff;
        font-size: 0.85rem;
        white-space: nowrap;
    }
    
    .source-buttons {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;
    }
    
    .source-btn {
        padding: 6px 14px;
        background: rgba(102, 51, 153, 0.15);
        border: 1px solid rgba(102, 51, 153, 0.3);
        border-radius: 16px;
        color: #aaa;
        font-size: 0.8rem;
        cursor: pointer;
        transition: all 0.2s ease;
    }
    
    .source-btn:hover {
        background: rgba(102, 51, 153, 0.35);
        color: #fff;
    }
    
    .source-btn.active {
        background: linear-gradient(135deg, #663399, #9966cc);
        color: white;
        border-color: transparent;
        font-weight: 600;
        box-shadow: 0 3px 10px rgba(102, 51, 153, 0.35);
    }
    
    /* === SOURCE INSTRUCTION (quando fonte específica selecionada) === */
    .source-instruction {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 30px;
        text-align: center;
        background: linear-gradient(145deg, rgba(30, 33, 48, 0.8), rgba(20, 22, 34, 0.9));
        border-radius: 20px;
        border: 1px solid rgba(102, 51, 153, 0.3);
        margin: 20px 0;
    }
    
    .instruction-icon {
        font-size: 4rem;
        margin-bottom: 16px;
        filter: drop-shadow(0 4px 12px rgba(102, 51, 153, 0.4));
    }
    
    .instruction-title {
        font-size: 1.5rem;
        font-weight: 700;
        color: #fff;
        margin: 0 0 12px 0;
        background: linear-gradient(135deg, #663399, #cc66ff);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .instruction-text {
        font-size: 1rem;
        color: #999;
        max-width: 500px;
        line-height: 1.6;
        margin: 0 0 24px 0;
    }

    /* Responsive source selector */
    @media (max-width: 600px) {
        .manga-source-selector {
            flex-direction: column;
            align-items: stretch;
            gap: 10px;
            padding: 10px;
        }
        
        .source-label {
            text-align: center;
        }
        
        .source-buttons {
            justify-content: center;
        }
        
        .source-btn {
            flex: 0 1 auto;
            padding: 8px 12px;
            font-size: 0.75rem;
        }
    }
    
    .manga-tab {
        padding: 16px;
        max-width: 1400px;
        margin: 0 auto;
    }
    
    .manga-search-bar {
        display: flex;
        gap: 10px;
        margin-bottom: 16px;
    }
    
    .manga-search-bar input {
        flex: 1;
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 12px;
        padding: 14px 20px;
        color: #fff;
        font-size: 1rem;
        transition: all 0.3s;
    }
    
    .manga-search-bar input:focus {
        outline: none;
        border-color: #663399;
        background: rgba(255, 255, 255, 0.12);
    }
    
    .manga-search-bar button {
        background: linear-gradient(135deg, #663399, #8855bb);
        border: none;
        border-radius: 12px;
        padding: 14px 28px;
        color: #fff;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.25s;
    }
    
    .manga-search-bar button:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 20px rgba(102, 51, 153, 0.4);
    }
    
    .manga-genres-filter {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        margin-bottom: 30px;
    }
    
    .genre-btn {
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 20px;
        padding: 8px 16px;
        color: #ccc;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.25s;
    }
    
    .genre-btn:hover {
        background: rgba(102, 51, 153, 0.3);
        border-color: #663399;
        color: #fff;
    }
    
    .genre-btn.active {
        background: linear-gradient(135deg, #663399, #8855bb);
        border-color: transparent;
        color: #fff;
    }
    
    .manga-section {
        margin-bottom: 32px;
    }
    
    .manga-section h2 {
        font-size: 1.3rem;
        color: #fff;
        margin-bottom: 16px;
        display: flex;
        align-items: center;
        gap: 10px;
        font-weight: 600;
    }
    
    /* === SKELETON LOADING === */
    .skeleton-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        gap: 16px;
    }
    
    .skeleton-card {
        animation: skeleton-pulse 1.5s ease-in-out infinite;
    }
    
    .skeleton-poster {
        aspect-ratio: 2/3;
        background: linear-gradient(90deg, #1a1d2e 25%, #2a2d3e 50%, #1a1d2e 75%);
        background-size: 200% 100%;
        animation: skeleton-shimmer 1.5s ease-in-out infinite;
        border-radius: 10px;
    }
    
    .skeleton-title {
        height: 14px;
        background: linear-gradient(90deg, #1a1d2e 25%, #2a2d3e 50%, #1a1d2e 75%);
        background-size: 200% 100%;
        animation: skeleton-shimmer 1.5s ease-in-out infinite;
        border-radius: 4px;
        margin-top: 10px;
        width: 80%;
    }
    
    .skeleton-subtitle {
        height: 10px;
        background: linear-gradient(90deg, #1a1d2e 25%, #2a2d3e 50%, #1a1d2e 75%);
        background-size: 200% 100%;
        animation: skeleton-shimmer 1.5s ease-in-out infinite;
        border-radius: 4px;
        margin-top: 6px;
        width: 50%;
    }
    
    @keyframes skeleton-shimmer {
        0% { background-position: 200% 0; }
        100% { background-position: -200% 0; }
    }
    
    @keyframes skeleton-pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.7; }
    }
    
    /* === LOADING SPINNER APRIMORADO === */
    .loading-mangas {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 20px;
        gap: 16px;
    }
    
    .loading-mangas .spinner {
        width: 48px;
        height: 48px;
        border: 3px solid rgba(102, 51, 153, 0.2);
        border-top-color: #9966cc;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    .loading-mangas p {
        color: #888;
        font-size: 0.9rem;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    .manga-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        gap: 16px;
        will-change: transform;
        contain: layout;
    }
    
    /* Lazy loading skeleton para cards */
    .manga-card {
        background: transparent;
        border: none;
        cursor: pointer;
        text-align: left;
        transition: transform 0.2s ease;
        will-change: transform;
        contain: layout style;
    }
    
    .manga-card:hover {
        transform: translateY(-6px);
    }
    
    .manga-card:active {
        transform: translateY(-2px);
    }
    
    .manga-poster {
        position: relative;
        aspect-ratio: 2/3;
        border-radius: 10px;
        overflow: hidden;
        background: linear-gradient(135deg, #1a1d2e 0%, #242838 100%);
        contain: strict;
    }
    
    .manga-poster img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transition: transform 0.3s ease;
    }
    
    .manga-card:hover .manga-poster img {
        transform: scale(1.05);
    }
    
    .manga-poster .no-image {
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 2.5rem;
        background: linear-gradient(135deg, #1a1d2e, #2a2d3e);
        color: rgba(255, 255, 255, 0.3);
    }
    
    .manga-overlay {
        position: absolute;
        inset: 0;
        background: linear-gradient(180deg, transparent 50%, rgba(102, 51, 153, 0.9) 100%);
        display: flex;
        align-items: flex-end;
        justify-content: center;
        padding-bottom: 15px;
        opacity: 0;
        transition: opacity 0.2s ease;
    }
    
    .manga-card:hover .manga-overlay {
        opacity: 1;
    }
    
    .read-icon {
        font-size: 2rem;
        color: white;
        text-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
    }
    
    .manga-card-info {
        padding: 10px 2px 4px;
    }
    
    .manga-card-info .manga-title {
        font-size: 0.875rem;
        font-weight: 600;
        color: #fff;
        overflow: hidden;
        text-overflow: ellipsis;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        line-clamp: 2;
        -webkit-box-orient: vertical;
        line-height: 1.3;
    }
    
    .manga-card-info .manga-latest {
        font-size: 0.8rem;
        color: #888;
        margin-top: 4px;
    }
    
    /* === SOURCE BADGE === */
    .source-badge {
        position: absolute;
        top: 8px;
        left: 8px;
        background: linear-gradient(135deg, #663399, #9966cc);
        color: white;
        padding: 3px 6px;
        border-radius: 4px;
        font-size: 0.6rem;
        font-weight: 700;
        z-index: 5;
        text-transform: uppercase;
        backdrop-filter: blur(4px);
    }
    
    .source-info {
        font-size: 0.75rem;
        font-weight: 400;
        opacity: 0.6;
        margin-left: 8px;
    }
    
    /* === FEATURED SECTION === */
    .featured-section {
        background: linear-gradient(135deg, rgba(102, 51, 153, 0.08), rgba(30, 33, 48, 0.4));
        padding: 20px;
        border-radius: 14px;
        border: 1px solid rgba(102, 51, 153, 0.25);
        backdrop-filter: blur(8px);
    }
    
    .featured-section h2 {
        background: linear-gradient(90deg, #ffd700, #ff8c00);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .manga-section-footer {
        display: flex;
        justify-content: center;
        margin-top: 20px;
        padding-top: 20px;
        border-top: 1px solid rgba(255, 255, 255, 0.08);
    }
    
    .btn-view-all {
        background: linear-gradient(135deg, #663399, #9966cc);
        color: white;
        border: none;
        padding: 12px 32px;
        border-radius: 25px;
        font-size: 1rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s;
        display: flex;
        align-items: center;
        gap: 8px;
    }
    
    .btn-view-all:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 20px rgba(102, 51, 153, 0.4);
    }
    
    /* === ALL MANGAS SECTION === */
    .all-mangas-section {
        margin-top: 32px;
        padding: 24px;
        background: rgba(30, 33, 48, 0.5);
        border-radius: 16px;
    }
    
    /* === ADULT SECTION (+18) === */
    .adult-section {
        margin-top: 40px;
        padding: 24px;
        background: rgba(139, 0, 0, 0.1);
        border-radius: 16px;
        border: 1px solid rgba(139, 0, 0, 0.3);
    }
    
    .adult-section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
    }
    
    .adult-section-header h2 {
        margin-bottom: 0;
        color: #ff6666;
    }
    
    .btn-toggle-adult {
        background: rgba(139, 0, 0, 0.3);
        color: #ff9999;
        border: 1px solid rgba(139, 0, 0, 0.5);
        padding: 8px 20px;
        border-radius: 20px;
        font-size: 0.9rem;
        cursor: pointer;
        transition: all 0.3s;
    }
    
    .btn-toggle-adult:hover {
        background: rgba(139, 0, 0, 0.5);
    }
    
    .btn-toggle-adult.active {
        background: rgba(139, 0, 0, 0.6);
        color: white;
    }
    
    .adult-warning {
        background: rgba(255, 100, 100, 0.2);
        color: #ff9999;
        padding: 12px 16px;
        border-radius: 8px;
        text-align: center;
        margin-bottom: 20px;
        font-size: 0.9rem;
    }
    
    .adult-card .manga-poster {
        border: 2px solid rgba(139, 0, 0, 0.5);
    }
    
    .adult-overlay {
        background: rgba(139, 0, 0, 0.7) !important;
    }
    
    .adult-badge {
        position: absolute;
        top: 8px;
        right: 8px;
        background: linear-gradient(135deg, #8b0000, #ff4444);
        color: white;
        padding: 4px 8px;
        border-radius: 4px;
        font-size: 0.7rem;
        font-weight: 700;
        z-index: 5;
    }
    
    .no-adult-content,
    .adult-hidden-msg {
        text-align: center;
        color: #888;
        font-size: 0.95rem;
        padding: 20px;
    }
    
    .loading-mangas {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 80px 20px;
        color: #888;
        gap: 20px;
    }
    
    .manga-empty-state {
        text-align: center;
        padding: 80px 20px;
    }
    
    .manga-empty-state .empty-icon {
        font-size: 4rem;
        margin-bottom: 20px;
    }
    
    .manga-empty-state h3 {
        color: #fff;
        margin-bottom: 10px;
    }
    
    .manga-empty-state p {
        color: #666;
        margin-bottom: 25px;
    }
    
    .btn-reload {
        background: linear-gradient(135deg, #663399, #8855bb);
        border: none;
        border-radius: 12px;
        padding: 12px 28px;
        color: #fff;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.25s;
    }
    
    .btn-reload:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 20px rgba(102, 51, 153, 0.4);
    }
    
    /* Manga Details View - Enhanced */
    .manga-details-view {
        padding: 0;
        max-width: 100%;
        margin: 0 auto;
        min-height: 100vh;
    }
    
    .btn-back-manga {
        position: absolute;
        top: 20px;
        left: 20px;
        z-index: 10;
        background: rgba(0, 0, 0, 0.6);
        backdrop-filter: blur(10px);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 12px;
        padding: 12px 24px;
        color: #fff;
        cursor: pointer;
        transition: all 0.3s ease;
        font-weight: 500;
    }
    
    .btn-back-manga:hover {
        background: rgba(102, 51, 153, 0.6);
        transform: translateX(-5px);
    }
    
    /* Manga Hero Section with Background */
    .manga-hero-wrapper {
        position: relative;
        min-height: 450px;
        overflow: hidden;
        border-radius: 0 0 30px 30px;
        margin-bottom: 30px;
    }
    
    .manga-hero-bg {
        position: absolute;
        inset: 0;
        background-size: cover;
        background-position: center top;
        filter: blur(25px) brightness(0.4);
        transform: scale(1.2);
        z-index: 0;
    }
    
    .manga-hero-bg::after {
        content: '';
        position: absolute;
        inset: 0;
        background: linear-gradient(to bottom, 
            rgba(15, 15, 25, 0.3) 0%,
            rgba(15, 15, 25, 0.8) 60%,
            rgba(15, 15, 25, 1) 100%);
    }
    
    .manga-hero-content {
        position: relative;
        z-index: 1;
        display: flex;
        gap: 40px;
        padding: 80px 40px 40px;
        max-width: 1200px;
        margin: 0 auto;
    }
    
    .manga-cover-container {
        position: relative;
        flex-shrink: 0;
    }
    
    .manga-cover-large {
        width: 260px;
        height: auto;
        border-radius: 16px;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6), 0 0 0 1px rgba(255, 255, 255, 0.1);
        object-fit: cover;
        aspect-ratio: 2/3;
        transition: transform 0.3s ease;
    }
    
    .manga-cover-large:hover {
        transform: scale(1.02);
    }
    
    .manga-cover-shadow {
        position: absolute;
        bottom: -20px;
        left: 50%;
        transform: translateX(-50%);
        width: 80%;
        height: 30px;
        background: radial-gradient(ellipse at center, rgba(102, 51, 153, 0.4) 0%, transparent 70%);
        filter: blur(15px);
    }
    
    .manga-info-details {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 16px;
        padding-top: 20px;
    }
    
    .manga-info-details h1 {
        font-size: 2.4rem;
        color: #fff;
        margin: 0;
        line-height: 1.2;
        text-shadow: 0 2px 10px rgba(0, 0, 0, 0.5);
    }
    
    .manga-meta-row {
        display: flex;
        align-items: center;
        gap: 15px;
        flex-wrap: wrap;
    }
    
    .manga-status-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 8px 16px;
        border-radius: 25px;
        font-size: 0.9rem;
        font-weight: 600;
    }
    
    .manga-status-badge.ongoing {
        background: linear-gradient(135deg, #ff6b35, #f7931e);
        color: #fff;
    }
    
    .manga-status-badge.completed {
        background: linear-gradient(135deg, #10b981, #059669);
        color: #fff;
    }
    
    .manga-chapters-count {
        color: #aaa;
        font-size: 0.95rem;
        background: rgba(255, 255, 255, 0.1);
        padding: 8px 16px;
        border-radius: 25px;
    }
    
    .manga-author {
        color: #888;
        font-size: 1rem;
        margin: 0;
    }
    
    .manga-genres-list {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
    }
    
    .manga-genres-list .genre-tag {
        background: rgba(102, 51, 153, 0.3);
        border: 1px solid rgba(102, 51, 153, 0.5);
        color: #d8b4fe;
        padding: 6px 14px;
        border-radius: 20px;
        font-size: 0.85rem;
        transition: all 0.2s;
    }
    
    .manga-genres-list .genre-tag:hover {
        background: rgba(102, 51, 153, 0.5);
    }
    
    .manga-description {
        color: #bbb;
        font-size: 1rem;
        line-height: 1.7;
        max-width: 650px;
        display: -webkit-box;
        -webkit-line-clamp: 4;
        line-clamp: 4;
        -webkit-box-orient: vertical;
        overflow: hidden;
    }
    
    .btn-start-reading {
        display: inline-flex;
        align-items: center;
        gap: 10px;
        background: linear-gradient(135deg, #663399, #8b5cf6);
        border: none;
        border-radius: 14px;
        padding: 16px 32px;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
        width: fit-content;
        margin-top: 10px;
        box-shadow: 0 8px 25px rgba(102, 51, 153, 0.4);
    }
    
    .btn-start-reading:hover {
        transform: translateY(-3px);
        box-shadow: 0 12px 35px rgba(102, 51, 153, 0.5);
    }
    
    .chapters-section {
        padding: 0 40px 40px;
        max-width: 1200px;
        margin: 0 auto;
    }
    
    .chapters-section h2 {
        color: #fff;
        font-size: 1.5rem;
        margin-bottom: 20px;
        display: flex;
        align-items: center;
        gap: 10px;
    }
    
    .loading-chapters {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 40px;
        color: #888;
        gap: 15px;
    }
    
    .no-chapters {
        color: #666;
        text-align: center;
        padding: 40px;
    }
    
    .chapters-list {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 12px;
        max-height: 600px;
        overflow-y: auto;
        padding: 5px;
    }
    
    .chapter-item {
        display: flex;
        align-items: center;
        gap: 15px;
        padding: 16px 20px;
        background: rgba(255, 255, 255, 0.03);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.25s;
        text-align: left;
        color: #fff;
    }
    
    .chapter-item:hover {
        background: rgba(102, 51, 153, 0.2);
        border-color: rgba(102, 51, 153, 0.5);
        transform: translateY(-2px);
    }
    
    .chapter-num {
        font-weight: 700;
        color: #8b5cf6;
        min-width: 70px;
        font-size: 0.95rem;
    }
    
    .chapter-title {
        flex: 1;
        color: #ddd;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 0.9rem;
    }
    
    .chapter-date {
        color: #666;
        font-size: 0.8rem;
    }
    
    /* Manga Reader Fullscreen */
    .manga-reader-fullscreen {
        position: fixed;
        inset: 0;
        background: #0a0a0f;
        z-index: 1000;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }
    
    .reader-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 12px 20px;
        background: rgba(20, 20, 30, 0.95);
        border-bottom: 1px solid #333;
        gap: 20px;
    }
    
    .reader-header .btn-back {
        background: transparent;
        border: 1px solid #555;
        color: #fff;
        padding: 8px 16px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 0.9rem;
        transition: all 0.2s;
    }
    
    .reader-header .btn-back:hover {
        background: #333;
        border-color: #777;
    }
    
    .reader-header .chapter-info {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
    }
    
    .reader-header .manga-name {
        font-size: 1rem;
        font-weight: 600;
        color: #fff;
    }
    
    .reader-header .chapter-number {
        font-size: 0.85rem;
        color: #888;
    }
    
    .reader-nav {
        display: flex;
        align-items: center;
        gap: 15px;
    }
    
    .reader-nav .btn-nav {
        background: #663399;
        border: none;
        color: #fff;
        padding: 8px 16px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 0.85rem;
        transition: all 0.2s;
    }
    
    .reader-nav .btn-nav:hover {
        background: #7744aa;
    }
    
    .reader-nav .page-count {
        color: #888;
        font-size: 0.85rem;
    }
    
    .reader-content {
        flex: 1;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        align-items: center;
    }
    
    .loading-pages, .no-pages {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        color: #888;
        gap: 20px;
    }
    
    .no-pages button {
        background: #663399;
        border: none;
        color: #fff;
        padding: 10px 20px;
        border-radius: 8px;
        cursor: pointer;
    }
    
    .pages-scroll {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 10px 0;
        gap: 4px;
        width: 100%;
        max-width: 100%;
        background: #000;
    }
    
    .manga-page {
        width: 100%;
        max-width: 1200px;
        height: auto;
        display: block;
        object-fit: contain;
        background: #111;
    }
    
    /* Telas grandes - imagens ocupam quase toda a largura */
    @media (min-width: 1400px) {
        .manga-page {
            max-width: 1100px;
        }
        
        .manga-grid,
        .skeleton-grid {
            grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
            gap: 20px;
        }
    }
    
    @media (min-width: 1200px) and (max-width: 1399px) {
        .manga-page {
            max-width: 1000px;
        }
        
        .manga-grid,
        .skeleton-grid {
            grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
        }
    }
    
    @media (min-width: 992px) and (max-width: 1199px) {
        .manga-page {
            max-width: 900px;
        }
        
        .manga-grid,
        .skeleton-grid {
            grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        }
    }
    
    @media (min-width: 768px) and (max-width: 991px) {
        .manga-grid,
        .skeleton-grid {
            grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
            gap: 14px;
        }
        
        .manga-page {
            max-width: 750px;
        }
        
        .manga-card-info .manga-title {
            font-size: 0.8rem;
        }
    }
    
    @media (max-width: 767px) {
        .manga-grid,
        .skeleton-grid {
            grid-template-columns: repeat(3, 1fr);
            gap: 12px;
        }
        
        .manga-section h2 {
            font-size: 1.1rem;
        }
        
        .manga-card-info .manga-title {
            font-size: 0.75rem;
        }
        
        .manga-card-info .manga-latest {
            font-size: 0.7rem;
        }
        
        .source-badge {
            font-size: 0.55rem;
            padding: 2px 5px;
        }
    }
    
    @media (max-width: 480px) {
        .manga-grid,
        .skeleton-grid {
            grid-template-columns: repeat(2, 1fr);
            gap: 10px;
        }
        
        .manga-tab {
            padding: 12px;
        }
        
        .manga-section {
            margin-bottom: 24px;
        }
        
        .featured-section {
            padding: 16px;
        }
    }

    /* Page wrapper e loading states */
    .page-wrapper {
        position: relative;
        width: 100%;
        display: flex;
        flex-direction: column;
        align-items: center;
        min-height: 200px;
        background: #111;
    }
    
    .page-wrapper .manga-page {
        opacity: 0;
        transition: opacity 0.3s ease;
    }
    
    .page-wrapper :global(.manga-page.loaded) {
        opacity: 1;
    }
    
    .page-wrapper :global(.manga-page.error) {
        opacity: 0.3;
        min-height: 400px;
        background: #222 url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="%23666"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>') center no-repeat;
        background-size: 60px;
    }
    
    .page-number {
        position: absolute;
        bottom: 10px;
        right: 10px;
        background: rgba(0, 0, 0, 0.7);
        color: #fff;
        padding: 4px 10px;
        border-radius: 4px;
        font-size: 0.75rem;
        opacity: 0;
        transition: opacity 0.2s;
    }
    
    .page-wrapper:hover .page-number {
        opacity: 1;
    }
    
    .chapter-end {
        padding: 40px 20px;
        text-align: center;
        background: linear-gradient(to bottom, #000, #111);
    }
    
    .chapter-end p {
        color: #888;
        margin-bottom: 20px;
        font-size: 1.1rem;
    }
    
    .btn-next-chapter {
        background: #663399;
        border: none;
        color: #fff;
        padding: 14px 32px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 1.1rem;
        transition: all 0.2s;
    }
    
    .btn-next-chapter:hover {
        background: #7744aa;
        transform: scale(1.05);
    }

    /* Manga Responsive Styles */
    @media (max-width: 1024px) {
        .manga-hero-content {
            padding: 70px 30px 30px;
            gap: 30px;
        }
        
        .manga-cover-large {
            width: 200px;
        }
        
        .manga-info-details h1 {
            font-size: 2rem;
        }
        
        .chapters-section {
            padding: 0 30px 30px;
        }
        
        .chapters-list {
            grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
        }
    }

    @media (max-width: 768px) {
        .manga-hero-wrapper {
            min-height: auto;
            border-radius: 0 0 20px 20px;
        }
        
        .manga-hero-content {
            flex-direction: column;
            align-items: center;
            text-align: center;
            padding: 80px 20px 30px;
            gap: 25px;
        }
        
        .manga-cover-container {
            display: flex;
            flex-direction: column;
            align-items: center;
        }
        
        .manga-cover-large {
            width: 180px;
        }
        
        .manga-info-details {
            align-items: center;
            padding-top: 0;
        }
        
        .manga-info-details h1 {
            font-size: 1.6rem;
        }
        
        .manga-meta-row {
            justify-content: center;
        }
        
        .manga-genres-list {
            justify-content: center;
        }
        
        .manga-description {
            text-align: center;
            -webkit-line-clamp: 3;
            line-clamp: 3;
        }
        
        .btn-start-reading {
            width: 100%;
            justify-content: center;
        }
        
        .chapters-section {
            padding: 0 15px 30px;
        }
        
        .chapters-section h2 {
            font-size: 1.3rem;
        }
        
        .chapters-list {
            grid-template-columns: 1fr;
            max-height: none;
        }
        
        .chapter-item {
            padding: 14px 16px;
        }
        
        .reader-header {
            flex-wrap: wrap;
            gap: 10px;
            padding: 10px 15px;
        }
        
        .chapter-info {
            order: -1;
            width: 100%;
            text-align: center;
            padding-bottom: 8px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }
        
        .reader-nav {
            width: 100%;
            justify-content: center;
        }
        
        .btn-nav {
            padding: 8px 12px;
            font-size: 0.85rem;
        }
        
        .btn-back-manga {
            padding: 10px 18px;
            font-size: 0.9rem;
        }
    }
    
    @media (max-width: 480px) {
        .manga-cover-large {
            width: 150px;
        }
        
        .manga-info-details h1 {
            font-size: 1.4rem;
        }
        
        .manga-status-badge,
        .manga-chapters-count {
            font-size: 0.8rem;
            padding: 6px 12px;
        }
        
        .manga-genres-list .genre-tag {
            font-size: 0.75rem;
            padding: 4px 10px;
        }
        
        .manga-description {
            font-size: 0.9rem;
        }
        
        .btn-start-reading {
            padding: 14px 24px;
            font-size: 1rem;
        }
        
        .chapter-item {
            padding: 12px 14px;
            gap: 10px;
        }
        
        .chapter-num {
            min-width: 60px;
            font-size: 0.85rem;
        }
        
        .chapter-title {
            font-size: 0.85rem;
        }
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

    /* ============================================
       MANGA SOURCES MANAGEMENT MODAL
       ============================================ */
    .manga-sources-modal {
        position: relative;
        background: linear-gradient(135deg, #1a1a2e 0%, #0d1929 100%);
        border: 1px solid rgba(139, 92, 246, 0.3);
        border-radius: 20px;
        width: 95%;
        max-width: 700px;
        max-height: 85vh;
        padding: 30px;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6);
        display: flex;
        flex-direction: column;
    }

    .sources-modal-header {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
        gap: 8px;
        margin-bottom: 20px;
    }

    .sources-modal-header .sources-icon {
        font-size: 48px;
    }

    .sources-modal-header h2 {
        color: #fff;
        font-size: 1.5rem;
        margin: 0;
        background: linear-gradient(135deg, #8b5cf6 0%, #a78bfa 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .sources-modal-header p {
        color: #8b8b8b;
        font-size: 0.9rem;
        margin: 0;
    }

    .sources-language-filter {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 20px;
        padding: 12px 16px;
        background: rgba(139, 92, 246, 0.1);
        border: 1px solid rgba(139, 92, 246, 0.2);
        border-radius: 12px;
    }

    .sources-language-filter .filter-label {
        color: #a78bfa;
        font-size: 0.9rem;
        font-weight: 500;
    }

    .language-buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }

    .lang-btn {
        padding: 6px 12px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 8px;
        color: #888;
        font-size: 0.8rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .lang-btn:hover {
        background: rgba(139, 92, 246, 0.2);
        border-color: rgba(139, 92, 246, 0.4);
        color: #fff;
    }

    .lang-btn.active {
        background: linear-gradient(135deg, #8b5cf6 0%, #a78bfa 100%);
        border-color: transparent;
        color: #fff;
        font-weight: 500;
    }

    .sources-list {
        flex: 1;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 12px;
        padding-right: 5px;
        max-height: 400px;
    }

    .sources-list::-webkit-scrollbar {
        width: 6px;
    }

    .sources-list::-webkit-scrollbar-track {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 3px;
    }

    .sources-list::-webkit-scrollbar-thumb {
        background: rgba(139, 92, 246, 0.5);
        border-radius: 3px;
    }

    .sources-loading, .sources-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 40px 20px;
        color: #666;
    }

    .sources-loading .spinner {
        width: 40px;
        height: 40px;
        border: 3px solid rgba(139, 92, 246, 0.2);
        border-top-color: #8b5cf6;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-bottom: 15px;
    }

    .sources-empty .empty-icon {
        font-size: 48px;
        margin-bottom: 10px;
    }

    .source-item {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 16px;
        background: rgba(255, 255, 255, 0.03);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 12px;
        transition: all 0.3s;
    }

    .source-item:hover {
        background: rgba(139, 92, 246, 0.1);
        border-color: rgba(139, 92, 246, 0.3);
    }

    .source-item.enabled {
        border-color: rgba(34, 197, 94, 0.3);
    }

    .source-item.disabled {
        opacity: 0.6;
    }

    .source-info {
        display: flex;
        align-items: flex-start;
        gap: 14px;
        flex: 1;
    }

    .source-info .source-icon {
        font-size: 32px;
        line-height: 1;
    }

    .source-details h4 {
        margin: 0 0 4px 0;
        color: #fff;
        font-size: 1rem;
        font-weight: 600;
    }

    .source-details p {
        margin: 0 0 8px 0;
        color: #888;
        font-size: 0.85rem;
        line-height: 1.4;
    }

    .source-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }

    .source-meta .source-lang,
    .source-meta .source-feature {
        padding: 3px 8px;
        background: rgba(139, 92, 246, 0.15);
        border-radius: 6px;
        font-size: 0.75rem;
        color: #a78bfa;
    }

    .source-meta .source-feature {
        background: rgba(59, 130, 246, 0.15);
        color: #93c5fd;
    }

    /* Toggle Switch */
    .toggle-switch {
        position: relative;
        width: 50px;
        height: 26px;
        flex-shrink: 0;
    }

    .toggle-switch input {
        opacity: 0;
        width: 0;
        height: 0;
    }

    .toggle-slider {
        position: absolute;
        cursor: pointer;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 26px;
        transition: 0.3s;
    }

    .toggle-slider:before {
        position: absolute;
        content: "";
        height: 20px;
        width: 20px;
        left: 3px;
        bottom: 3px;
        background: #fff;
        border-radius: 50%;
        transition: 0.3s;
    }

    .toggle-switch input:checked + .toggle-slider {
        background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
    }

    .toggle-switch input:checked + .toggle-slider:before {
        transform: translateX(24px);
    }

    .sources-modal-footer {
        display: flex;
        gap: 12px;
        margin-top: 20px;
        padding-top: 20px;
        border-top: 1px solid rgba(255, 255, 255, 0.1);
    }

    .sources-modal-footer .btn-reset {
        flex: 1;
        padding: 12px 20px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 10px;
        color: #888;
        font-size: 0.9rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .sources-modal-footer .btn-reset:hover {
        background: rgba(251, 191, 36, 0.15);
        border-color: rgba(251, 191, 36, 0.4);
        color: #fbbf24;
    }

    .sources-modal-footer .btn-done {
        flex: 2;
        padding: 12px 24px;
        background: linear-gradient(135deg, #8b5cf6 0%, #a78bfa 100%);
        border: none;
        border-radius: 10px;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s;
    }

    .sources-modal-footer .btn-done:hover {
        transform: translateY(-2px);
        box-shadow: 0 6px 25px rgba(139, 92, 246, 0.4);
    }

    /* Manga Source Selector - Manage Button */
    .manga-source-selector .source-btn.manage-btn {
        background: linear-gradient(135deg, rgba(139, 92, 246, 0.2) 0%, rgba(167, 139, 250, 0.2) 100%);
        border: 1px solid rgba(139, 92, 246, 0.4);
        color: #a78bfa;
    }

    .manga-source-selector .source-btn.manage-btn:hover {
        background: linear-gradient(135deg, rgba(139, 92, 246, 0.4) 0%, rgba(167, 139, 250, 0.4) 100%);
        color: #fff;
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
       SOCIAL SYSTEM (Friends without Discord)
       ============================================ */
    
    .social-connect-container {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 60vh;
        padding: 20px;
    }
    
    .social-connect-card {
        position: relative;
        background: rgba(20, 25, 50, 0.9);
        border: 1px solid rgba(147, 51, 234, 0.3);
        border-radius: 24px;
        padding: 50px 40px;
        max-width: 480px;
        text-align: center;
        overflow: hidden;
        backdrop-filter: blur(10px);
    }
    
    .social-glow {
        position: absolute;
        top: -50%;
        left: -50%;
        width: 200%;
        height: 200%;
        background: radial-gradient(circle at center, rgba(147, 51, 234, 0.15) 0%, transparent 50%);
        animation: rotate-glow 10s linear infinite;
        pointer-events: none;
    }
    
    .social-content {
        position: relative;
        z-index: 1;
    }
    
    .social-logo-animated {
        position: relative;
        display: inline-flex;
        justify-content: center;
        align-items: center;
        width: 120px;
        height: 120px;
        margin-bottom: 25px;
    }
    
    .social-svg {
        background: linear-gradient(135deg, #9333ea 0%, #a855f7 100%);
        border-radius: 50%;
        padding: 15px;
        box-shadow: 0 8px 30px rgba(147, 51, 234, 0.4);
    }
    
    .social-title {
        color: #fff;
        font-size: 1.8rem;
        font-weight: 700;
        margin: 0 0 10px;
    }
    
    .social-subtitle {
        color: #8b8fa3;
        font-size: 1rem;
        margin: 0 0 30px;
    }
    
    .social-features {
        display: flex;
        flex-direction: column;
        gap: 16px;
        margin-bottom: 35px;
    }
    
    .social-privacy {
        color: #555;
        font-size: 0.8rem;
        margin: 20px 0 0;
    }
    
    .create-profile-section {
        margin-top: 25px;
    }
    
    .create-hint {
        color: #8b8fa3;
        font-size: 0.9rem;
        margin-bottom: 15px;
    }
    
    .create-form {
        display: flex;
        gap: 12px;
        justify-content: center;
    }
    
    .username-input {
        flex: 1;
        max-width: 200px;
        padding: 14px 18px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(147, 51, 234, 0.3);
        border-radius: 12px;
        color: #fff;
        font-size: 1rem;
        transition: all 0.3s ease;
    }
    
    .username-input:focus {
        outline: none;
        border-color: #9333ea;
        box-shadow: 0 0 20px rgba(147, 51, 234, 0.3);
    }
    
    .username-input::placeholder {
        color: #555;
    }
    
    .btn-create-profile {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 14px 24px;
        background: linear-gradient(135deg, #9333ea 0%, #a855f7 100%);
        border: none;
        border-radius: 12px;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .btn-create-profile:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 6px 25px rgba(147, 51, 234, 0.4);
    }
    
    .btn-create-profile:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    .social-error {
        color: #ef4444;
        font-size: 0.85rem;
        margin-top: 12px;
    }
    
    /* Social Profile Section */
    .social-profile-section {
        margin-bottom: 40px;
    }
    
    /* ============================================
       FRIENDS DASHBOARD - Modern Responsive Layout
       ============================================ */
    .friends-dashboard {
        display: grid;
        grid-template-columns: 320px 1fr;
        gap: 24px;
        max-width: 1200px;
    }
    
    @media (max-width: 900px) {
        .friends-dashboard {
            grid-template-columns: 1fr;
        }
    }
    
    /* Sidebar - Profile & Add Friend */
    .friends-sidebar {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }
    
    /* Profile Card Modern */
    .profile-card-modern {
        background: linear-gradient(145deg, rgba(30, 35, 60, 0.95), rgba(20, 25, 45, 0.95));
        border: 1px solid rgba(147, 51, 234, 0.25);
        border-radius: 20px;
        padding: 20px;
        backdrop-filter: blur(10px);
    }
    
    .profile-card-header {
        display: flex;
        align-items: center;
        gap: 14px;
        margin-bottom: 18px;
    }
    
    .profile-avatar-modern {
        position: relative;
        width: 56px;
        height: 56px;
        border-radius: 16px;
        background: linear-gradient(135deg, #9333ea 0%, #c084fc 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }
    
    .avatar-letter {
        color: #fff;
        font-size: 1.5rem;
        font-weight: 700;
        text-transform: uppercase;
    }
    
    .avatar-status {
        position: absolute;
        bottom: -2px;
        right: -2px;
        width: 14px;
        height: 14px;
        border-radius: 50%;
        border: 3px solid rgba(20, 25, 45, 1);
    }
    
    .avatar-status.online {
        background: #22c55e;
        box-shadow: 0 0 8px rgba(34, 197, 94, 0.5);
    }
    
    .avatar-status.offline {
        background: #6b7280;
    }
    
    .profile-info-modern {
        flex: 1;
        min-width: 0;
    }
    
    .profile-username {
        color: #fff;
        font-size: 1.1rem;
        font-weight: 600;
        margin: 0 0 4px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    
    .profile-status-text {
        font-size: 0.8rem;
        color: #a1a1aa;
    }
    
    /* Code Section */
    .profile-code-section {
        background: rgba(0, 0, 0, 0.25);
        border-radius: 12px;
        padding: 14px;
        margin-bottom: 16px;
    }
    
    .code-label {
        display: block;
        font-size: 0.7rem;
        color: #71717a;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin-bottom: 8px;
    }
    
    .code-display {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
    }
    
    .share-code-text {
        font-family: 'JetBrains Mono', 'Fira Code', monospace;
        font-size: 1.3rem;
        font-weight: 600;
        color: #c084fc;
        letter-spacing: 3px;
        background: none;
    }
    
    .code-actions {
        display: flex;
        gap: 4px;
    }
    
    .btn-icon {
        width: 34px;
        height: 34px;
        border-radius: 10px;
        background: rgba(147, 51, 234, 0.15);
        border: 1px solid rgba(147, 51, 234, 0.3);
        color: #c084fc;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.2s ease;
    }
    
    .btn-icon:hover {
        background: rgba(147, 51, 234, 0.3);
        transform: scale(1.05);
    }
    
    .btn-icon:active {
        transform: scale(0.95);
    }
    
    /* Quick Settings */
    .profile-quick-settings {
        display: flex;
        flex-direction: column;
        gap: 10px;
        margin-bottom: 16px;
    }
    
    .quick-setting {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 10px 12px;
        background: rgba(0, 0, 0, 0.15);
        border-radius: 10px;
    }
    
    .setting-icon {
        font-size: 1rem;
    }
    
    .setting-label {
        flex: 1;
        font-size: 0.85rem;
        color: #d4d4d8;
    }
    
    .mini-toggle {
        width: 40px;
        height: 22px;
        border-radius: 11px;
        background: rgba(113, 113, 122, 0.4);
        border: none;
        cursor: pointer;
        position: relative;
        transition: all 0.3s ease;
    }
    
    .mini-toggle.active {
        background: linear-gradient(90deg, #9333ea, #c084fc);
    }
    
    .mini-toggle .toggle-dot {
        position: absolute;
        top: 3px;
        left: 3px;
        width: 16px;
        height: 16px;
        border-radius: 50%;
        background: #fff;
        transition: all 0.3s ease;
        box-shadow: 0 2px 4px rgba(0,0,0,0.2);
    }
    
    .mini-toggle.active .toggle-dot {
        left: 21px;
    }
    
    /* Delete Profile Button */
    .btn-delete-profile {
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 10px;
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.2);
        border-radius: 10px;
        color: #f87171;
        font-size: 0.8rem;
        cursor: pointer;
        transition: all 0.2s ease;
    }
    
    .btn-delete-profile:hover {
        background: rgba(239, 68, 68, 0.2);
        border-color: rgba(239, 68, 68, 0.4);
    }
    
    /* Add Friend Card */
    .add-friend-card {
        background: rgba(30, 35, 60, 0.7);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 16px;
        padding: 18px;
    }
    
    .card-title {
        display: flex;
        align-items: center;
        gap: 10px;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 600;
        margin: 0 0 14px;
    }
    
    .card-title svg {
        color: #c084fc;
    }
    
    .add-friend-input-group {
        display: flex;
        gap: 10px;
    }
    
    .friend-input {
        flex: 1;
        padding: 12px 14px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        color: #fff;
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.95rem;
        text-transform: uppercase;
        letter-spacing: 2px;
        transition: all 0.2s ease;
    }
    
    .friend-input:focus {
        outline: none;
        border-color: #9333ea;
        box-shadow: 0 0 0 3px rgba(147, 51, 234, 0.15);
    }
    
    .friend-input::placeholder {
        text-transform: none;
        letter-spacing: normal;
        color: #52525b;
    }
    
    .btn-add {
        width: 48px;
        height: 48px;
        border-radius: 12px;
        background: linear-gradient(135deg, #9333ea 0%, #a855f7 100%);
        border: none;
        color: #fff;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.2s ease;
        flex-shrink: 0;
    }
    
    .btn-add:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 6px 20px rgba(147, 51, 234, 0.4);
    }
    
    .btn-add:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    .spinner-mini {
        width: 18px;
        height: 18px;
        border: 2px solid rgba(255,255,255,0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    .error-message {
        margin: 10px 0 0;
        padding: 10px 12px;
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.2);
        border-radius: 8px;
        color: #f87171;
        font-size: 0.8rem;
    }
    
    /* Connection Card */
    .connection-card {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 12px 14px;
        border-radius: 12px;
        transition: all 0.3s ease;
    }
    
    .connection-card.connected {
        background: rgba(34, 197, 94, 0.1);
        border: 1px solid rgba(34, 197, 94, 0.2);
    }
    
    .connection-card.disconnected {
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.2);
    }
    
    .connection-info {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    
    .connection-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
    }
    
    .connection-card.connected .connection-dot {
        background: #22c55e;
        box-shadow: 0 0 8px rgba(34, 197, 94, 0.6);
        animation: pulse-green 2s ease-in-out infinite;
    }
    
    .connection-card.disconnected .connection-dot {
        background: #ef4444;
    }
    
    .connection-text {
        font-size: 0.8rem;
        font-weight: 500;
    }
    
    .connection-card.connected .connection-text {
        color: #4ade80;
    }
    
    .connection-card.disconnected .connection-text {
        color: #f87171;
    }
    
    .btn-retry {
        width: 32px;
        height: 32px;
        border-radius: 8px;
        background: rgba(255, 255, 255, 0.1);
        border: none;
        color: currentColor;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.2s ease;
    }
    
    .btn-retry:hover {
        background: rgba(255, 255, 255, 0.2);
        transform: rotate(180deg);
    }
    
    @keyframes pulse-green {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; }
    }
    
    /* ============================================
       FRIENDS MAIN - List Area
       ============================================ */
    .friends-main {
        background: rgba(30, 35, 60, 0.5);
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 20px;
        padding: 20px;
        min-height: 400px;
    }
    
    .friends-list-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 20px;
        padding-bottom: 16px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    }
    
    .friends-title {
        display: flex;
        align-items: center;
        gap: 12px;
        color: #fff;
        font-size: 1.1rem;
        font-weight: 600;
        margin: 0;
    }
    
    .friends-title svg {
        color: #c084fc;
    }
    
    .friends-count {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        min-width: 26px;
        height: 26px;
        padding: 0 8px;
        background: rgba(147, 51, 234, 0.2);
        border-radius: 13px;
        font-size: 0.8rem;
        font-weight: 600;
        color: #c084fc;
    }
    
    .btn-refresh-friends {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        background: rgba(147, 51, 234, 0.15);
        border: 1px solid rgba(147, 51, 234, 0.3);
        border-radius: 10px;
        color: #c084fc;
        font-size: 0.85rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;
    }
    
    .btn-refresh-friends:hover {
        background: rgba(147, 51, 234, 0.25);
        transform: translateY(-1px);
    }
    
    /* Loading State */
    .friends-loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 16px;
        padding: 60px 20px;
        color: #a1a1aa;
    }
    
    .loading-spinner {
        width: 40px;
        height: 40px;
        border: 3px solid rgba(147, 51, 234, 0.2);
        border-top-color: #9333ea;
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    /* Empty State */
    .empty-friends-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        text-align: center;
        padding: 50px 20px;
    }
    
    .empty-illustration {
        margin-bottom: 20px;
        color: #52525b;
    }
    
    .empty-friends-state h4 {
        color: #fff;
        font-size: 1.1rem;
        margin: 0 0 8px;
    }
    
    .empty-friends-state p {
        color: #71717a;
        margin: 0 0 16px;
        font-size: 0.9rem;
    }
    
    .empty-friends-state strong {
        color: #c084fc;
        font-family: monospace;
        letter-spacing: 1px;
    }
    
    .share-tip {
        padding: 12px 16px;
        background: rgba(147, 51, 234, 0.1);
        border-radius: 10px;
        color: #a1a1aa;
        font-size: 0.8rem;
    }
    
    /* Friends Grid */
    .friends-grid {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    
    .friend-card {
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 14px 16px;
        background: rgba(0, 0, 0, 0.2);
        border: 1px solid rgba(255, 255, 255, 0.05);
        border-radius: 14px;
        transition: all 0.2s ease;
    }
    
    .friend-card:hover {
        background: rgba(0, 0, 0, 0.3);
        border-color: rgba(147, 51, 234, 0.2);
    }
    
    .friend-card.is-online {
        border-color: rgba(34, 197, 94, 0.2);
    }
    
    .friend-card-avatar {
        position: relative;
        width: 46px;
        height: 46px;
        border-radius: 12px;
        background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }
    
    .friend-initial {
        color: #fff;
        font-size: 1.2rem;
        font-weight: 600;
    }
    
    .friend-status-dot {
        position: absolute;
        bottom: -2px;
        right: -2px;
        width: 12px;
        height: 12px;
        border-radius: 50%;
        border: 2px solid rgba(30, 35, 60, 1);
    }
    
    .friend-status-dot.online {
        background: #22c55e;
        box-shadow: 0 0 6px rgba(34, 197, 94, 0.5);
    }
    
    .friend-status-dot.offline {
        background: #52525b;
    }
    
    .friend-card-info {
        flex: 1;
        min-width: 0;
    }
    
    .friend-card-name {
        display: block;
        color: #fff;
        font-size: 0.95rem;
        font-weight: 600;
        margin-bottom: 4px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    
    .friend-watching-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        background: rgba(34, 197, 94, 0.15);
        border-radius: 20px;
        max-width: 100%;
    }
    
    .watching-icon {
        color: #22c55e;
        font-size: 0.7rem;
    }
    
    .watching-text {
        color: #4ade80;
        font-size: 0.75rem;
        font-weight: 500;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    
    .friend-status-label {
        color: #71717a;
        font-size: 0.8rem;
    }
    
    .btn-remove {
        width: 34px;
        height: 34px;
        border-radius: 10px;
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid transparent;
        color: #71717a;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.2s ease;
        flex-shrink: 0;
        opacity: 0;
    }
    
    .friend-card:hover .btn-remove {
        opacity: 1;
    }
    
    .btn-remove:hover {
        background: rgba(239, 68, 68, 0.2);
        border-color: rgba(239, 68, 68, 0.3);
        color: #f87171;
    }
    
    /* Mobile responsiveness for friends */
    @media (max-width: 600px) {
        .friends-dashboard {
            gap: 16px;
        }
        
        .profile-card-modern,
        .add-friend-card,
        .friends-main {
            padding: 16px;
        }
        
        .profile-card-header {
            flex-direction: column;
            text-align: center;
        }
        
        .code-display {
            flex-direction: column;
            gap: 12px;
        }
        
        .code-actions {
            width: 100%;
            justify-content: center;
        }
        
        .friends-list-header {
            flex-direction: column;
            gap: 12px;
            align-items: stretch;
        }
        
        .btn-refresh-friends {
            justify-content: center;
        }
        
        .btn-remove {
            opacity: 1;
        }
    }
    
    /* Old styles cleanup - keeping for backward compat */
    .social-profile-card {
        position: relative;
        background: rgba(20, 25, 50, 0.9);
        border: 1px solid rgba(147, 51, 234, 0.3);
        border-radius: 20px;
        overflow: hidden;
        margin-bottom: 20px;
    }
    
    .social-profile-card .profile-background {
        height: 80px;
        background: linear-gradient(135deg, #9333ea 0%, #a855f7 50%, #9333ea 100%);
        background-size: 200% 200%;
        animation: gradient-shift 5s ease infinite;
    }
    
    .social-profile-card .profile-avatar-placeholder {
        width: 80px;
        height: 80px;
        border-radius: 50%;
        background: linear-gradient(135deg, #9333ea 0%, #a855f7 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        border: 4px solid #1a1a2e;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
    }
    
    .avatar-initial {
        color: #fff;
        font-size: 2rem;
        font-weight: 700;
    }
    
    /* Share Code Section */
    .share-code-section {
        display: flex;
        flex-direction: column;
        gap: 6px;
        margin-top: 8px;
    }
    
    .share-code-label {
        color: #8b8b8b;
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    
    .share-code-box {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(147, 51, 234, 0.3);
        border-radius: 10px;
    }
    
    .share-code {
        color: #a855f7;
        font-family: monospace;
        font-size: 1.1rem;
        font-weight: 600;
        letter-spacing: 2px;
    }
    
    .btn-copy-code, .btn-regenerate-code {
        background: none;
        border: none;
        padding: 4px 8px;
        cursor: pointer;
        font-size: 1rem;
        opacity: 0.7;
        transition: all 0.2s ease;
    }
    
    .btn-copy-code:hover, .btn-regenerate-code:hover {
        opacity: 1;
        transform: scale(1.1);
    }
    
    /* Add Friend Section */
    .add-friend-section {
        background: rgba(20, 25, 50, 0.6);
        border: 1px solid rgba(147, 51, 234, 0.15);
        border-radius: 16px;
        padding: 20px;
        margin-bottom: 30px;
    }
    
    .add-friend-section h3 {
        color: #fff;
        font-size: 1rem;
        margin: 0 0 15px 0;
    }
    
    .add-friend-form {
        display: flex;
        gap: 12px;
    }
    
    .friend-code-input {
        flex: 1;
        padding: 12px 16px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(147, 51, 234, 0.3);
        border-radius: 10px;
        color: #fff;
        font-size: 1rem;
        font-family: monospace;
        text-transform: uppercase;
        letter-spacing: 2px;
        transition: all 0.3s ease;
    }
    
    .friend-code-input:focus {
        outline: none;
        border-color: #9333ea;
        box-shadow: 0 0 15px rgba(147, 51, 234, 0.3);
    }
    
    .friend-code-input::placeholder {
        color: #555;
        text-transform: none;
        letter-spacing: normal;
    }
    
    .btn-add-friend {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 12px 20px;
        background: linear-gradient(135deg, #9333ea 0%, #a855f7 100%);
        border: none;
        border-radius: 10px;
        color: #fff;
        font-size: 0.9rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .btn-add-friend:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 4px 20px rgba(147, 51, 234, 0.4);
    }
    
    .btn-add-friend:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    /* Friends List Styles */
    .friend-activity-card .friend-avatar-placeholder {
        width: 45px;
        height: 45px;
        border-radius: 50%;
        background: linear-gradient(135deg, #9333ea 0%, #a855f7 100%);
        display: flex;
        align-items: center;
        justify-content: center;
    }
    
    .friend-activity-card .avatar-initial {
        font-size: 1.2rem;
    }
    
    .offline-dot {
        position: absolute;
        bottom: 2px;
        right: 2px;
        width: 12px;
        height: 12px;
        background: #6b7280;
        border: 2px solid #1a1a2e;
        border-radius: 50%;
    }
    
    .friend-status-offline {
        color: #6b7280;
        font-size: 0.8rem;
    }
    
    .btn-remove-friend {
        padding: 8px 12px;
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.2);
        border-radius: 8px;
        color: #ef4444;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.2s ease;
    }
    
    .btn-remove-friend:hover {
        background: rgba(239, 68, 68, 0.2);
        border-color: rgba(239, 68, 68, 0.4);
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
       SHARE MODAL - Premium Glass Design
       ============================================ */
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.85);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        backdrop-filter: blur(15px);
        animation: fadeIn 0.2s ease;
    }
    
    .share-modal {
        background: linear-gradient(180deg, rgba(26, 31, 58, 0.98) 0%, rgba(13, 16, 37, 0.99) 100%);
        border: 1px solid rgba(139, 92, 246, 0.2);
        border-radius: 24px;
        padding: 35px;
        max-width: 520px;
        width: 90%;
        position: relative;
        animation: modalSlideIn 0.35s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 
            0 30px 80px rgba(0, 0, 0, 0.5),
            0 0 100px rgba(139, 92, 246, 0.1),
            inset 0 1px 0 rgba(255, 255, 255, 0.05);
    }
    
    @keyframes modalSlideIn {
        from {
            opacity: 0;
            transform: translateY(-30px) scale(0.9);
        }
        to {
            opacity: 1;
            transform: translateY(0) scale(1);
        }
    }
    
    .modal-close {
        position: absolute;
        top: 18px;
        right: 18px;
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.1);
        color: rgba(255, 255, 255, 0.7);
        width: 36px;
        height: 36px;
        border-radius: 50%;
        cursor: pointer;
        font-size: 1.1rem;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        display: flex;
        align-items: center;
        justify-content: center;
    }
    
    .modal-close:hover {
        background: rgba(245, 87, 108, 0.3);
        border-color: rgba(245, 87, 108, 0.5);
        color: #fff;
        transform: rotate(90deg) scale(1.1);
    }
    
    .share-modal-header {
        text-align: center;
        margin-bottom: 28px;
    }
    
    .share-modal-header h2 {
        color: #fff;
        margin: 0 0 10px;
        font-size: 1.6rem;
        font-weight: 700;
        background: linear-gradient(135deg, #fff, rgba(245, 87, 108, 0.9));
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }
    
    .share-modal-header p {
        color: rgba(255, 255, 255, 0.5);
        margin: 0;
        font-size: 0.95rem;
    }
    
    .share-anime-preview {
        display: flex;
        gap: 16px;
        padding: 18px;
        background: linear-gradient(135deg, rgba(255, 255, 255, 0.03), rgba(255, 255, 255, 0.06));
        border-radius: 16px;
        border: 1px solid rgba(255, 255, 255, 0.06);
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
