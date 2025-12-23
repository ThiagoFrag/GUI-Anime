export namespace auth {
	
	export class UserSession {
	    user_id: string;
	    username: string;
	    email?: string;
	    avatar: string;
	    token: string;
	    is_vip: boolean;
	    is_premium: boolean;
	    friend_token: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    last_login: any;
	    seeding_enabled: boolean;
	    seeding_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new UserSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.email = source["email"];
	        this.avatar = source["avatar"];
	        this.token = source["token"];
	        this.is_vip = source["is_vip"];
	        this.is_premium = source["is_premium"];
	        this.friend_token = source["friend_token"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.last_login = this.convertValues(source["last_login"], null);
	        this.seeding_enabled = source["seeding_enabled"];
	        this.seeding_bytes = source["seeding_bytes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace embeddedplayer {
	
	export class TrackInfo {
	    id: number;
	    type: string;
	    title: string;
	    language: string;
	    default: boolean;
	    forced: boolean;
	    external: boolean;
	    codec: string;
	
	    static createFrom(source: any = {}) {
	        return new TrackInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.language = source["language"];
	        this.default = source["default"];
	        this.forced = source["forced"];
	        this.external = source["external"];
	        this.codec = source["codec"];
	    }
	}
	export class PlayerInfo {
	    state: string;
	    position: number;
	    duration: number;
	    volume: number;
	    muted: boolean;
	    qualityMode: string;
	    audioTracks: TrackInfo[];
	    subtitleTracks: TrackInfo[];
	    currentAudio: number;
	    currentSub: number;
	    videoWidth: number;
	    videoHeight: number;
	    isFullscreen: boolean;
	    bufferPercent: number;
	
	    static createFrom(source: any = {}) {
	        return new PlayerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.position = source["position"];
	        this.duration = source["duration"];
	        this.volume = source["volume"];
	        this.muted = source["muted"];
	        this.qualityMode = source["qualityMode"];
	        this.audioTracks = this.convertValues(source["audioTracks"], TrackInfo);
	        this.subtitleTracks = this.convertValues(source["subtitleTracks"], TrackInfo);
	        this.currentAudio = source["currentAudio"];
	        this.currentSub = source["currentSub"];
	        this.videoWidth = source["videoWidth"];
	        this.videoHeight = source["videoHeight"];
	        this.isFullscreen = source["isFullscreen"];
	        this.bufferPercent = source["bufferPercent"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace extensions {
	
	export class AnimeDetails {
	    title: string;
	    alternateTitle?: string;
	    url: string;
	    image: string;
	    banner?: string;
	    description: string;
	    status: string;
	    genres: string[];
	    year?: number;
	    studio?: string;
	    rating?: number;
	    totalEpisodes?: number;
	
	    static createFrom(source: any = {}) {
	        return new AnimeDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.alternateTitle = source["alternateTitle"];
	        this.url = source["url"];
	        this.image = source["image"];
	        this.banner = source["banner"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.genres = source["genres"];
	        this.year = source["year"];
	        this.studio = source["studio"];
	        this.rating = source["rating"];
	        this.totalEpisodes = source["totalEpisodes"];
	    }
	}
	export class AnimeEntry {
	    title: string;
	    url: string;
	    image: string;
	    description?: string;
	    status?: string;
	
	    static createFrom(source: any = {}) {
	        return new AnimeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.image = source["image"];
	        this.description = source["description"];
	        this.status = source["status"];
	    }
	}
	export class Episode {
	    number: number;
	    title?: string;
	    url: string;
	    thumbnail?: string;
	    // Go type: time
	    date?: any;
	    filler: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Episode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.thumbnail = source["thumbnail"];
	        this.date = this.convertValues(source["date"], null);
	        this.filler = source["filler"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Subtitle {
	    url: string;
	    language: string;
	    label: string;
	    format: string;
	    default: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Subtitle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.language = source["language"];
	        this.label = source["label"];
	        this.format = source["format"];
	        this.default = source["default"];
	    }
	}
	export class VideoSource {
	    url: string;
	    quality: string;
	    format: string;
	    server: string;
	    headers?: Record<string, string>;
	    subtitles?: Subtitle[];
	
	    static createFrom(source: any = {}) {
	        return new VideoSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.quality = source["quality"];
	        this.format = source["format"];
	        this.server = source["server"];
	        this.headers = source["headers"];
	        this.subtitles = this.convertValues(source["subtitles"], Subtitle);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class AniListAnime {
	    id: number;
	    malId: number;
	    title: string;
	    titleEnglish: string;
	    titleNative: string;
	    description: string;
	    image: string;
	    banner: string;
	    color: string;
	    genres: string[];
	    episodes: number;
	    duration: number;
	    status: string;
	    season: string;
	    year: number;
	    score: number;
	    popularity: number;
	    studio: string;
	    trailerUrl: string;
	    isAiring: boolean;
	    nextEpisode: number;
	
	    static createFrom(source: any = {}) {
	        return new AniListAnime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.malId = source["malId"];
	        this.title = source["title"];
	        this.titleEnglish = source["titleEnglish"];
	        this.titleNative = source["titleNative"];
	        this.description = source["description"];
	        this.image = source["image"];
	        this.banner = source["banner"];
	        this.color = source["color"];
	        this.genres = source["genres"];
	        this.episodes = source["episodes"];
	        this.duration = source["duration"];
	        this.status = source["status"];
	        this.season = source["season"];
	        this.year = source["year"];
	        this.score = source["score"];
	        this.popularity = source["popularity"];
	        this.studio = source["studio"];
	        this.trailerUrl = source["trailerUrl"];
	        this.isAiring = source["isAiring"];
	        this.nextEpisode = source["nextEpisode"];
	    }
	}
	export class AnimeSourceInfo {
	    id: string;
	    name: string;
	    description: string;
	    language: string;
	    priority: number;
	
	    static createFrom(source: any = {}) {
	        return new AnimeSourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.language = source["language"];
	        this.priority = source["priority"];
	    }
	}
	export class SourceStatus {
	    name: string;
	    isAvailable: boolean;
	    failCount: number;
	    lastError?: string;
	    retryAfter?: string;
	    cachedUrls: number;
	
	    static createFrom(source: any = {}) {
	        return new SourceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isAvailable = source["isAvailable"];
	        this.failCount = source["failCount"];
	        this.lastError = source["lastError"];
	        this.retryAfter = source["retryAfter"];
	        this.cachedUrls = source["cachedUrls"];
	    }
	}
	export class CacheStats {
	    sources: SourceStatus[];
	    totalStreams: number;
	    totalCache: number;
	
	    static createFrom(source: any = {}) {
	        return new CacheStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = this.convertValues(source["sources"], SourceStatus);
	        this.totalStreams = source["totalStreams"];
	        this.totalCache = source["totalCache"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConsumetAnime {
	    id: string;
	    title: string;
	    image: string;
	    totalEpisodes: number;
	    subOrDub: string;
	    genres: string[];
	    description: string;
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new ConsumetAnime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.image = source["image"];
	        this.totalEpisodes = source["totalEpisodes"];
	        this.subOrDub = source["subOrDub"];
	        this.genres = source["genres"];
	        this.description = source["description"];
	        this.provider = source["provider"];
	    }
	}
	export class ConsumetEpisode {
	    id: string;
	    number: number;
	    title: string;
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new ConsumetEpisode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.provider = source["provider"];
	    }
	}
	export class DiscordFriendActivity {
	    userId: string;
	    username: string;
	    avatar: string;
	    animeTitle: string;
	    episodeNum: number;
	    animeImage: string;
	    isOnline: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscordFriendActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userId = source["userId"];
	        this.username = source["username"];
	        this.avatar = source["avatar"];
	        this.animeTitle = source["animeTitle"];
	        this.episodeNum = source["episodeNum"];
	        this.animeImage = source["animeImage"];
	        this.isOnline = source["isOnline"];
	    }
	}
	export class DiscordLinkInfo {
	    isLinked: boolean;
	    userId: string;
	    username: string;
	    avatar: string;
	    linkedAt: string;
	    showStatus: boolean;
	    shareAnimes: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscordLinkInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isLinked = source["isLinked"];
	        this.userId = source["userId"];
	        this.username = source["username"];
	        this.avatar = source["avatar"];
	        this.linkedAt = source["linkedAt"];
	        this.showStatus = source["showStatus"];
	        this.shareAnimes = source["shareAnimes"];
	    }
	}
	export class DiscordRecommendation {
	    id: string;
	    username: string;
	    userAvatar: string;
	    animeTitle: string;
	    animeImage: string;
	    animeScore: number;
	    message: string;
	    timestamp: number;
	    likes: number;
	    likedByMe: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscordRecommendation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.userAvatar = source["userAvatar"];
	        this.animeTitle = source["animeTitle"];
	        this.animeImage = source["animeImage"];
	        this.animeScore = source["animeScore"];
	        this.message = source["message"];
	        this.timestamp = source["timestamp"];
	        this.likes = source["likes"];
	        this.likedByMe = source["likedByMe"];
	    }
	}
	export class DiscordStatus {
	    connected: boolean;
	    webhookUrl: string;
	    username: string;
	    serverName: string;
	    channelName: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscordStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.webhookUrl = source["webhookUrl"];
	        this.username = source["username"];
	        this.serverName = source["serverName"];
	        this.channelName = source["channelName"];
	    }
	}
	export class DiscordUserInfo {
	    id: string;
	    username: string;
	    avatarUrl: string;
	    connected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscordUserInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.avatarUrl = source["avatarUrl"];
	        this.connected = source["connected"];
	    }
	}
	export class EnimeAnime {
	    id: string;
	    title: string;
	    titleNative: string;
	    image: string;
	    banner: string;
	    anilistId: number;
	    malId: number;
	    episodes: number;
	    status: string;
	    genre: string[];
	
	    static createFrom(source: any = {}) {
	        return new EnimeAnime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.titleNative = source["titleNative"];
	        this.image = source["image"];
	        this.banner = source["banner"];
	        this.anilistId = source["anilistId"];
	        this.malId = source["malId"];
	        this.episodes = source["episodes"];
	        this.status = source["status"];
	        this.genre = source["genre"];
	    }
	}
	export class EpisodeFileInfo {
	    nome_original: string;
	    tags: string[];
	    qualidade?: string;
	    subgrupo?: string;
	
	    static createFrom(source: any = {}) {
	        return new EpisodeFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nome_original = source["nome_original"];
	        this.tags = source["tags"];
	        this.qualidade = source["qualidade"];
	        this.subgrupo = source["subgrupo"];
	    }
	}
	export class ParsedEpisodeInfo {
	    original: string;
	    titulo: string;
	    temporada: number;
	    episodio: number;
	    qualidade: string;
	    tag: string;
	    audio_tipo: string;
	
	    static createFrom(source: any = {}) {
	        return new ParsedEpisodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original = source["original"];
	        this.titulo = source["titulo"];
	        this.temporada = source["temporada"];
	        this.episodio = source["episodio"];
	        this.qualidade = source["qualidade"];
	        this.tag = source["tag"];
	        this.audio_tipo = source["audio_tipo"];
	    }
	}
	export class GroupedEpisodeInfo {
	    episodio_numero: number;
	    temporada: number;
	    titulo_limpo: string;
	    arquivos: ParsedEpisodeInfo[];
	
	    static createFrom(source: any = {}) {
	        return new GroupedEpisodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.episodio_numero = source["episodio_numero"];
	        this.temporada = source["temporada"];
	        this.titulo_limpo = source["titulo_limpo"];
	        this.arquivos = this.convertValues(source["arquivos"], ParsedEpisodeInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EpisodeGroupResultInfo {
	    anime_nome: string;
	    episodios: GroupedEpisodeInfo[];
	    total_episodios: number;
	
	    static createFrom(source: any = {}) {
	        return new EpisodeGroupResultInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anime_nome = source["anime_nome"];
	        this.episodios = this.convertValues(source["episodios"], GroupedEpisodeInfo);
	        this.total_episodios = source["total_episodios"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EpisodeInfo {
	    id_episodio: number;
	    temporada: number;
	    titulo_exibicao_limpo: string;
	    titulo_episodio_completo?: string;
	    arquivos_disponiveis: EpisodeFileInfo[];
	
	    static createFrom(source: any = {}) {
	        return new EpisodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id_episodio = source["id_episodio"];
	        this.temporada = source["temporada"];
	        this.titulo_exibicao_limpo = source["titulo_exibicao_limpo"];
	        this.titulo_episodio_completo = source["titulo_episodio_completo"];
	        this.arquivos_disponiveis = this.convertValues(source["arquivos_disponiveis"], EpisodeFileInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EpisodeParseResultInfo {
	    nome_anime: string;
	    episodios: EpisodeInfo[];
	    total_episodios: number;
	
	    static createFrom(source: any = {}) {
	        return new EpisodeParseResultInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nome_anime = source["nome_anime"];
	        this.episodios = this.convertValues(source["episodios"], EpisodeInfo);
	        this.total_episodios = source["total_episodios"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExtensionInfo {
	    id: string;
	    name: string;
	    version: string;
	    language: string;
	    iconUrl: string;
	    enabled: boolean;
	    hasError: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtensionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.language = source["language"];
	        this.iconUrl = source["iconUrl"];
	        this.enabled = source["enabled"];
	        this.hasError = source["hasError"];
	        this.error = source["error"];
	    }
	}
	
	export class MangaChapterInfo {
	    number: string;
	    title: string;
	    url: string;
	    date: string;
	    mangaId: string;
	    mangaName: string;
	
	    static createFrom(source: any = {}) {
	        return new MangaChapterInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.date = source["date"];
	        this.mangaId = source["mangaId"];
	        this.mangaName = source["mangaName"];
	    }
	}
	export class MangaInfo {
	    id: string;
	    title: string;
	    image: string;
	    url: string;
	    latestChapter: string;
	    genres: string[];
	    description: string;
	    status: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new MangaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.image = source["image"];
	        this.url = source["url"];
	        this.latestChapter = source["latestChapter"];
	        this.genres = source["genres"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.source = source["source"];
	    }
	}
	export class MangaListResult {
	    mangas: MangaInfo[];
	    totalPages: number;
	    page: number;
	
	    static createFrom(source: any = {}) {
	        return new MangaListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mangas = this.convertValues(source["mangas"], MangaInfo);
	        this.totalPages = source["totalPages"];
	        this.page = source["page"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MangaPageInfo {
	    number: number;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new MangaPageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.url = source["url"];
	    }
	}
	export class MangaSourceDetail {
	    id: string;
	    name: string;
	    description: string;
	    url: string;
	    language: string;
	    icon: string;
	    enabled: boolean;
	    supportsLatest: boolean;
	    supportsPopular: boolean;
	    supportsSearch: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MangaSourceDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.url = source["url"];
	        this.language = source["language"];
	        this.icon = source["icon"];
	        this.enabled = source["enabled"];
	        this.supportsLatest = source["supportsLatest"];
	        this.supportsPopular = source["supportsPopular"];
	        this.supportsSearch = source["supportsSearch"];
	    }
	}
	export class MangaSourceInfo {
	    id: string;
	    name: string;
	    description: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new MangaSourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.url = source["url"];
	    }
	}
	
	export class PipelineResult {
	    success: boolean;
	    torrent_id: number;
	    hash: string;
	    file_name: string;
	    file_size: number;
	    gofile_code: string;
	    gofile_link: string;
	    encoded_link?: string;
	    download_time: number;
	    upload_time: number;
	    encode_time?: number;
	    torrent_deleted: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PipelineResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.torrent_id = source["torrent_id"];
	        this.hash = source["hash"];
	        this.file_name = source["file_name"];
	        this.file_size = source["file_size"];
	        this.gofile_code = source["gofile_code"];
	        this.gofile_link = source["gofile_link"];
	        this.encoded_link = source["encoded_link"];
	        this.download_time = source["download_time"];
	        this.upload_time = source["upload_time"];
	        this.encode_time = source["encode_time"];
	        this.torrent_deleted = source["torrent_deleted"];
	        this.error = source["error"];
	    }
	}
	export class QualityModeInfo {
	    id: string;
	    name: string;
	    description: string;
	    icon: string;
	    gpuRequired: string;
	    features: string[];
	
	    static createFrom(source: any = {}) {
	        return new QualityModeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	        this.gpuRequired = source["gpuRequired"];
	        this.features = source["features"];
	    }
	}
	export class RemoteAnimeInfo {
	    id: number;
	    mal_id?: number;
	    title: string;
	    title_en?: string;
	    cover_image?: string;
	    episodes: number;
	    status: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteAnimeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.mal_id = source["mal_id"];
	        this.title = source["title"];
	        this.title_en = source["title_en"];
	        this.cover_image = source["cover_image"];
	        this.episodes = source["episodes"];
	        this.status = source["status"];
	        this.source = source["source"];
	    }
	}
	export class RemoteEpisodeInfo {
	    id: number;
	    anime_id: number;
	    number: number;
	    title?: string;
	    gofile_id?: string;
	    torbox_id?: string;
	    magnet_link?: string;
	    quality?: string;
	    has_ptbr: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RemoteEpisodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.anime_id = source["anime_id"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.gofile_id = source["gofile_id"];
	        this.torbox_id = source["torbox_id"];
	        this.magnet_link = source["magnet_link"];
	        this.quality = source["quality"];
	        this.has_ptbr = source["has_ptbr"];
	    }
	}
	export class RemoteExtensionInfo {
	    id: string;
	    name: string;
	    version: string;
	    language: string;
	    iconUrl: string;
	    changelog: string;
	    installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RemoteExtensionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.language = source["language"];
	        this.iconUrl = source["iconUrl"];
	        this.changelog = source["changelog"];
	        this.installed = source["installed"];
	    }
	}
	export class RemoteStreamLink {
	    direct_url: string;
	    filename: string;
	    size: number;
	    content_type: string;
	    expires_at?: number;
	
	    static createFrom(source: any = {}) {
	        return new RemoteStreamLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.direct_url = source["direct_url"];
	        this.filename = source["filename"];
	        this.size = source["size"];
	        this.content_type = source["content_type"];
	        this.expires_at = source["expires_at"];
	    }
	}
	export class RemoteTorrentFile {
	    id: number;
	    torrent_id?: number;
	    name: string;
	    short_name: string;
	    size: number;
	    size_str: string;
	    episode: number;
	    season: number;
	    is_video: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RemoteTorrentFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.torrent_id = source["torrent_id"];
	        this.name = source["name"];
	        this.short_name = source["short_name"];
	        this.size = source["size"];
	        this.size_str = source["size_str"];
	        this.episode = source["episode"];
	        this.season = source["season"];
	        this.is_video = source["is_video"];
	    }
	}
	export class RemoteTorrentInfo {
	    hash: string;
	    name: string;
	    size: number;
	    size_str: string;
	    status: string;
	    progress: number;
	    files: RemoteTorrentFile[];
	
	    static createFrom(source: any = {}) {
	        return new RemoteTorrentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.size_str = source["size_str"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.files = this.convertValues(source["files"], RemoteTorrentFile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RemoteTorrentResult {
	    title: string;
	    name?: string;
	    raw_title?: string;
	    magnet?: string;
	    hash: string;
	    size: string;
	    seeds: number;
	    leeches: number;
	    source: string;
	    page_url?: string;
	    is_brazilian: boolean;
	    clean_title?: string;
	    variants?: RemoteTorrentResult[];
	
	    static createFrom(source: any = {}) {
	        return new RemoteTorrentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.name = source["name"];
	        this.raw_title = source["raw_title"];
	        this.magnet = source["magnet"];
	        this.hash = source["hash"];
	        this.size = source["size"];
	        this.seeds = source["seeds"];
	        this.leeches = source["leeches"];
	        this.source = source["source"];
	        this.page_url = source["page_url"];
	        this.is_brazilian = source["is_brazilian"];
	        this.clean_title = source["clean_title"];
	        this.variants = this.convertValues(source["variants"], RemoteTorrentResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RepositoryInfo {
	    name: string;
	    url: string;
	    official: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RepositoryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.official = source["official"];
	    }
	}
	export class SeedingStats {
	    jobsCompleted: number;
	    errors: number;
	    totalBytesUploaded: number;
	    currentJob?: string;
	    currentJobId?: string;
	    lastJobTime: number;
	    averageJobTime: number;
	    isRunning: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SeedingStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobsCompleted = source["jobsCompleted"];
	        this.errors = source["errors"];
	        this.totalBytesUploaded = source["totalBytesUploaded"];
	        this.currentJob = source["currentJob"];
	        this.currentJobId = source["currentJobId"];
	        this.lastJobTime = source["lastJobTime"];
	        this.averageJobTime = source["averageJobTime"];
	        this.isRunning = source["isRunning"];
	    }
	}
	export class SkipTimesResult {
	    hasOpening: boolean;
	    openingStart: number;
	    openingEnd: number;
	    hasEnding: boolean;
	    endingStart: number;
	    endingEnd: number;
	    hasRecap: boolean;
	    recapStart: number;
	    recapEnd: number;
	    episodeLength: number;
	
	    static createFrom(source: any = {}) {
	        return new SkipTimesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasOpening = source["hasOpening"];
	        this.openingStart = source["openingStart"];
	        this.openingEnd = source["openingEnd"];
	        this.hasEnding = source["hasEnding"];
	        this.endingStart = source["endingStart"];
	        this.endingEnd = source["endingEnd"];
	        this.hasRecap = source["hasRecap"];
	        this.recapStart = source["recapStart"];
	        this.recapEnd = source["recapEnd"];
	        this.episodeLength = source["episodeLength"];
	    }
	}
	export class SmartStreamResult {
	    url: string;
	    source: string;
	    duration: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SmartStreamResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.source = source["source"];
	        this.duration = source["duration"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	
	export class SubtitleSearchResult {
	    title: string;
	    language: string;
	    format: string;
	    source: string;
	    download_url: string;
	    match_score: number;
	
	    static createFrom(source: any = {}) {
	        return new SubtitleSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.language = source["language"];
	        this.format = source["format"];
	        this.source = source["source"];
	        this.download_url = source["download_url"];
	        this.match_score = source["match_score"];
	    }
	}
	export class SubtitleSearchResponse {
	    query: string;
	    episode: number;
	    language: string;
	    total_found: number;
	    results: SubtitleSearchResult[];
	
	    static createFrom(source: any = {}) {
	        return new SubtitleSearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.episode = source["episode"];
	        this.language = source["language"];
	        this.total_found = source["total_found"];
	        this.results = this.convertValues(source["results"], SubtitleSearchResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TorBoxAnimeResult {
	    title: string;
	    fullName: string;
	    quality: string;
	    size: string;
	    seeds: number;
	    cached: boolean;
	    magnet: string;
	    hash: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new TorBoxAnimeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.fullName = source["fullName"];
	        this.quality = source["quality"];
	        this.size = source["size"];
	        this.seeds = source["seeds"];
	        this.cached = source["cached"];
	        this.magnet = source["magnet"];
	        this.hash = source["hash"];
	        this.source = source["source"];
	    }
	}
	export class TorBoxFileInfo {
	    id: number;
	    torrentId: number;
	    name: string;
	    shortName: string;
	    size: number;
	    sizeStr: string;
	    episode: number;
	    season: number;
	    isPlayable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TorBoxFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.torrentId = source["torrentId"];
	        this.name = source["name"];
	        this.shortName = source["shortName"];
	        this.size = source["size"];
	        this.sizeStr = source["sizeStr"];
	        this.episode = source["episode"];
	        this.season = source["season"];
	        this.isPlayable = source["isPlayable"];
	    }
	}
	export class TorBoxTorrentInfo {
	    id: number;
	    hash: string;
	    name: string;
	    progress: number;
	    status: string;
	    cached: boolean;
	    files: TorBoxFileInfo[];
	    totalSize: number;
	    totalSizeStr: string;
	
	    static createFrom(source: any = {}) {
	        return new TorBoxTorrentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.hash = source["hash"];
	        this.name = source["name"];
	        this.progress = source["progress"];
	        this.status = source["status"];
	        this.cached = source["cached"];
	        this.files = this.convertValues(source["files"], TorBoxFileInfo);
	        this.totalSize = source["totalSize"];
	        this.totalSizeStr = source["totalSizeStr"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TorrentResult {
	    title: string;
	    magnet: string;
	    hash: string;
	    size: string;
	    quality: string;
	    seeders: number;
	    source: string;
	    isBr: boolean;
	    dualAudio: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TorrentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.magnet = source["magnet"];
	        this.hash = source["hash"];
	        this.size = source["size"];
	        this.quality = source["quality"];
	        this.seeders = source["seeders"];
	        this.source = source["source"];
	        this.isBr = source["isBr"];
	        this.dualAudio = source["dualAudio"];
	    }
	}
	export class TorrentSource {
	    id: string;
	    name: string;
	    description: string;
	    isBr: boolean;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TorrentSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.isBr = source["isBr"];
	        this.available = source["available"];
	    }
	}
	export class VPSPipelineResponse {
	    status: string;
	    job_id: string;
	    stream_url: string;
	    file_name: string;
	    encode: boolean;
	    upload_gofile: boolean;
	    message: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VPSPipelineResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.job_id = source["job_id"];
	        this.stream_url = source["stream_url"];
	        this.file_name = source["file_name"];
	        this.encode = source["encode"];
	        this.upload_gofile = source["upload_gofile"];
	        this.message = source["message"];
	        this.error = source["error"];
	    }
	}
	export class VPSSearchResult {
	    status: string;
	    stream_url: string;
	    torrent_id: number;
	    file_id: number;
	    file_name: string;
	    file_size: number;
	    quality: string;
	    cached: boolean;
	    title: string;
	    hash: string;
	    message: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VPSSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.stream_url = source["stream_url"];
	        this.torrent_id = source["torrent_id"];
	        this.file_id = source["file_id"];
	        this.file_name = source["file_name"];
	        this.file_size = source["file_size"];
	        this.quality = source["quality"];
	        this.cached = source["cached"];
	        this.title = source["title"];
	        this.hash = source["hash"];
	        this.message = source["message"];
	        this.error = source["error"];
	    }
	}

}

export namespace social {
	
	export class Friend {
	    user_id: string;
	    username: string;
	    avatar?: string;
	    share_code?: string;
	    // Go type: time
	    added_at: any;
	    is_online: boolean;
	    // Go type: time
	    last_seen: any;
	    current_anime?: string;
	    current_ep?: number;
	    total_watched: number;
	
	    static createFrom(source: any = {}) {
	        return new Friend(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.avatar = source["avatar"];
	        this.share_code = source["share_code"];
	        this.added_at = this.convertValues(source["added_at"], null);
	        this.is_online = source["is_online"];
	        this.last_seen = this.convertValues(source["last_seen"], null);
	        this.current_anime = source["current_anime"];
	        this.current_ep = source["current_ep"];
	        this.total_watched = source["total_watched"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FriendActivity {
	    user_id: string;
	    username: string;
	    avatar?: string;
	    anime_title?: string;
	    anime_image?: string;
	    episode_num?: number;
	    is_watching: boolean;
	    is_online: boolean;
	    last_activity: string;
	
	    static createFrom(source: any = {}) {
	        return new FriendActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.avatar = source["avatar"];
	        this.anime_title = source["anime_title"];
	        this.anime_image = source["anime_image"];
	        this.episode_num = source["episode_num"];
	        this.is_watching = source["is_watching"];
	        this.is_online = source["is_online"];
	        this.last_activity = source["last_activity"];
	    }
	}
	export class UserProfile {
	    user_id: string;
	    username: string;
	    avatar?: string;
	    share_code: string;
	    auth_token?: string;
	    // Go type: time
	    created_at: any;
	    show_status: boolean;
	    share_animes: boolean;
	    total_watched: number;
	    // Go type: time
	    last_sync: any;
	
	    static createFrom(source: any = {}) {
	        return new UserProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.avatar = source["avatar"];
	        this.share_code = source["share_code"];
	        this.auth_token = source["auth_token"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.show_status = source["show_status"];
	        this.share_animes = source["share_animes"];
	        this.total_watched = source["total_watched"];
	        this.last_sync = this.convertValues(source["last_sync"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace store {
	
	export class AnimeSource {
	    Name: string;
	    Language: string;
	    URL: string;
	
	    static createFrom(source: any = {}) {
	        return new AnimeSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Language = source["Language"];
	        this.URL = source["URL"];
	    }
	}
	export class Episode {
	    Title: string;
	    URL: string;
	    Season: number;
	    Number: number;
	    Source: string;
	
	    static createFrom(source: any = {}) {
	        return new Episode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.URL = source["URL"];
	        this.Season = source["Season"];
	        this.Number = source["Number"];
	        this.Source = source["Source"];
	    }
	}
	export class SavedAnime {
	    Title: string;
	    Image: string;
	    URL: string;
	    Source?: string;
	    Sources?: AnimeSource[];
	
	    static createFrom(source: any = {}) {
	        return new SavedAnime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Image = source["Image"];
	        this.URL = source["URL"];
	        this.Source = source["Source"];
	        this.Sources = this.convertValues(source["Sources"], AnimeSource);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UserSettings {
	    start_fullscreen: boolean;
	    content_language: string;
	    default_quality: string;
	    use_anime4k: boolean;
	    seeding_enabled: boolean;
	    seeding_max_cpu: number;
	    seeding_max_bandwidth: number;
	    seeding_only_wifi: boolean;
	    seeding_schedule: string;
	    seeding_contributed: number;
	
	    static createFrom(source: any = {}) {
	        return new UserSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_fullscreen = source["start_fullscreen"];
	        this.content_language = source["content_language"];
	        this.default_quality = source["default_quality"];
	        this.use_anime4k = source["use_anime4k"];
	        this.seeding_enabled = source["seeding_enabled"];
	        this.seeding_max_cpu = source["seeding_max_cpu"];
	        this.seeding_max_bandwidth = source["seeding_max_bandwidth"];
	        this.seeding_only_wifi = source["seeding_only_wifi"];
	        this.seeding_schedule = source["seeding_schedule"];
	        this.seeding_contributed = source["seeding_contributed"];
	    }
	}
	export class WatchedEpisode {
	    anime_title: string;
	    anime_image: string;
	    anime_url: string;
	    episode_title: string;
	    episode_url: string;
	    episode_num: number;
	    watched_at: string;
	    progress: number;
	
	    static createFrom(source: any = {}) {
	        return new WatchedEpisode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anime_title = source["anime_title"];
	        this.anime_image = source["anime_image"];
	        this.anime_url = source["anime_url"];
	        this.episode_title = source["episode_title"];
	        this.episode_url = source["episode_url"];
	        this.episode_num = source["episode_num"];
	        this.watched_at = source["watched_at"];
	        this.progress = source["progress"];
	    }
	}
	export class UserData {
	    username: string;
	    avatar: string;
	    history: SavedAnime[];
	    favorites: SavedAnime[];
	    watch_history: WatchedEpisode[];
	    settings: UserSettings;
	    mpv_path?: string;
	    default_quality?: string;
	
	    static createFrom(source: any = {}) {
	        return new UserData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.avatar = source["avatar"];
	        this.history = this.convertValues(source["history"], SavedAnime);
	        this.favorites = this.convertValues(source["favorites"], SavedAnime);
	        this.watch_history = this.convertValues(source["watch_history"], WatchedEpisode);
	        this.settings = this.convertValues(source["settings"], UserSettings);
	        this.mpv_path = source["mpv_path"];
	        this.default_quality = source["default_quality"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace torbox {
	
	export class AnimeTorrent {
	    title: string;
	    magnet: string;
	    hash: string;
	    size: string;
	    seeds: number;
	    leeches: number;
	    // Go type: time
	    date: any;
	    source: string;
	    quality: string;
	    cached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AnimeTorrent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.magnet = source["magnet"];
	        this.hash = source["hash"];
	        this.size = source["size"];
	        this.seeds = source["seeds"];
	        this.leeches = source["leeches"];
	        this.date = this.convertValues(source["date"], null);
	        this.source = source["source"];
	        this.quality = source["quality"];
	        this.cached = source["cached"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FileStream {
	    file_id: number;
	    file_name: string;
	    file_size: number;
	    stream_url: string;
	
	    static createFrom(source: any = {}) {
	        return new FileStream(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_id = source["file_id"];
	        this.file_name = source["file_name"];
	        this.file_size = source["file_size"];
	        this.stream_url = source["stream_url"];
	    }
	}
	export class InstantStreamResult {
	    success: boolean;
	    stream_url: string;
	    torrent_id: number;
	    file_id: number;
	    file_name: string;
	    file_size: number;
	    quality: string;
	    cached: boolean;
	    title: string;
	    hash: string;
	    all_files?: FileStream[];
	
	    static createFrom(source: any = {}) {
	        return new InstantStreamResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.stream_url = source["stream_url"];
	        this.torrent_id = source["torrent_id"];
	        this.file_id = source["file_id"];
	        this.file_name = source["file_name"];
	        this.file_size = source["file_size"];
	        this.quality = source["quality"];
	        this.cached = source["cached"];
	        this.title = source["title"];
	        this.hash = source["hash"];
	        this.all_files = this.convertValues(source["all_files"], FileStream);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TorrentFile {
	    id: number;
	    name: string;
	    size: number;
	    mimetype: string;
	    short_name: string;
	    IsPlayable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TorrentFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.mimetype = source["mimetype"];
	        this.short_name = source["short_name"];
	        this.IsPlayable = source["IsPlayable"];
	    }
	}
	export class Torrent {
	    id: number;
	    hash: string;
	    name: string;
	    size: number;
	    progress: number;
	    download_state: string;
	    seeds: number;
	    peers: number;
	    ratio: number;
	    download_speed: number;
	    upload_speed: number;
	    expires_at: string;
	    files: TorrentFile[];
	    created_at: string;
	    total_downloaded: number;
	    total_uploaded: number;
	    cached: boolean;
	    download_finished: boolean;
	    active: boolean;
	    availability: number;
	
	    static createFrom(source: any = {}) {
	        return new Torrent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.hash = source["hash"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.progress = source["progress"];
	        this.download_state = source["download_state"];
	        this.seeds = source["seeds"];
	        this.peers = source["peers"];
	        this.ratio = source["ratio"];
	        this.download_speed = source["download_speed"];
	        this.upload_speed = source["upload_speed"];
	        this.expires_at = source["expires_at"];
	        this.files = this.convertValues(source["files"], TorrentFile);
	        this.created_at = source["created_at"];
	        this.total_downloaded = source["total_downloaded"];
	        this.total_uploaded = source["total_uploaded"];
	        this.cached = source["cached"];
	        this.download_finished = source["download_finished"];
	        this.active = source["active"];
	        this.availability = source["availability"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class User {
	    id: number;
	    email: string;
	    plan: number;
	    PlanName: string;
	    total_downloaded: number;
	    created_at: string;
	    is_subscribed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.email = source["email"];
	        this.plan = source["plan"];
	        this.PlanName = source["PlanName"];
	        this.total_downloaded = source["total_downloaded"];
	        this.created_at = source["created_at"];
	        this.is_subscribed = source["is_subscribed"];
	    }
	}

}

