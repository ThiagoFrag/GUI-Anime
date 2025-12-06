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
	
	    static createFrom(source: any = {}) {
	        return new UserSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_fullscreen = source["start_fullscreen"];
	        this.content_language = source["content_language"];
	        this.default_quality = source["default_quality"];
	        this.use_anime4k = source["use_anime4k"];
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

