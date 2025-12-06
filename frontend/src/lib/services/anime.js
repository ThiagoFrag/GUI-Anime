/**
 * Serviço para interação com API de Animes
 * Encapsula todas as chamadas ao backend Go relacionadas a animes
 */
import { 
    GetCurrentUser, CreateUser, BuscarAnimes, BuscarAnimesMulti, 
    GetTopAnimes, GetAnimeURL, GetEpisodes, GetEpisodesForSource, 
    PlayAnime, GetStreamURLForEpisode, AssistirEpisodio, GetProxyURLForVideo,
    GetTrendingAnimes, GetPopularAnimes, SearchAniList, GetAnimeHDImage,
    ClearEpisodesCache, ClearAllCache, GetCacheStats, ResetSourceFailures,
    GetSkipTimes
} from '../../../wailsjs/go/main/App';
import { withTimeout } from '../utils/helpers.js';

/**
 * Carrega dados iniciais (trending e top animes)
 * @returns {Promise<{trending: Array, top: Array}>}
 */
export async function loadInitialData() {
    const [trendingRes, topRes] = await Promise.allSettled([
        withTimeout(GetTrendingAnimes(15), 8000),
        withTimeout(GetTopAnimes(), 8000)
    ]);
    
    return {
        trending: trendingRes.status === 'fulfilled' ? trendingRes.value || [] : [],
        top: topRes.status === 'fulfilled' ? topRes.value || [] : []
    };
}

/**
 * Busca animes por termo
 * @param {string} termo - Termo de busca
 * @returns {Promise<Array>}
 */
export async function searchAnimes(termo) {
    if (!termo) return [];
    const res = await BuscarAnimes(termo);
    return Array.isArray(res) ? res : [];
}

/**
 * Busca animes por múltiplos termos (gênero)
 * @param {string[]} searchTerms - Array de termos de busca
 * @returns {Promise<Array>}
 */
export async function searchAnimesByGenre(searchTerms) {
    const results = await BuscarAnimesMulti(searchTerms);
    return Array.isArray(results) ? results : [];
}

/**
 * Obtém URL do anime
 * @param {string} title - Título do anime
 * @returns {Promise<string>}
 */
export async function getAnimeUrl(title) {
    return await GetAnimeURL(title);
}

/**
 * Obtém lista de episódios
 * @param {string} animeUrl - URL do anime
 * @returns {Promise<Array>}
 */
export async function getEpisodes(animeUrl) {
    const eps = await GetEpisodes(animeUrl);
    return Array.isArray(eps) ? eps : [];
}

/**
 * Obtém lista de episódios de uma fonte específica
 * @param {string} url - URL da fonte
 * @param {string} sourceName - Nome da fonte
 * @returns {Promise<Array>}
 */
export async function getEpisodesFromSource(url, sourceName) {
    const eps = await GetEpisodesForSource(url, sourceName);
    return Array.isArray(eps) ? eps : [];
}

/**
 * Obtém URL do stream para um episódio
 * @param {string} animeUrl - URL do anime
 * @param {string} episodeUrl - URL do episódio
 * @returns {Promise<string>}
 */
export async function getStreamUrl(animeUrl, episodeUrl) {
    return await GetStreamURLForEpisode(animeUrl, episodeUrl);
}

/**
 * Obtém URL proxy para o vídeo
 * @param {string} streamUrl - URL original do stream
 * @returns {Promise<string>}
 */
export async function getProxyUrl(streamUrl) {
    return await GetProxyURLForVideo(streamUrl);
}

/**
 * Reproduz episódio no MPV
 * @param {string} animeUrl - URL do anime
 * @param {string} episodeUrl - URL do episódio
 * @param {string} title - Título do episódio
 */
export async function playInMpv(animeUrl, episodeUrl, title) {
    await AssistirEpisodio(animeUrl, episodeUrl, title);
}

/**
 * Busca skip times (abertura/encerramento) para o episódio
 * @param {number} malId - MAL ID do anime
 * @param {number} episodeNumber - Número do episódio
 * @returns {Promise<Object|null>}
 */
export async function getSkipTimes(malId, episodeNumber) {
    if (!malId || malId <= 0 || !episodeNumber || episodeNumber <= 0) {
        return null;
    }
    return await GetSkipTimes(malId, episodeNumber);
}

/**
 * Busca MAL ID pelo título no AniList
 * @param {string} title - Título do anime
 * @returns {Promise<number>}
 */
export async function getMalIdByTitle(title) {
    if (!title) return 0;
    try {
        const results = await SearchAniList(title, 1);
        if (results?.length > 0 && results[0].malId > 0) {
            return results[0].malId;
        }
    } catch (err) {
        console.error('[getMalIdByTitle] Error:', err);
    }
    return 0;
}

/**
 * Obtém estatísticas do cache
 * @returns {Promise<Object>}
 */
export async function getCacheStats() {
    return await GetCacheStats();
}

/**
 * Limpa cache de episódios
 */
export async function clearEpisodesCache() {
    await ClearEpisodesCache();
}

/**
 * Limpa todo o cache
 */
export async function clearAllCache() {
    await ClearAllCache();
}

/**
 * Reseta falhas de fontes
 */
export async function resetSourceFailures() {
    await ResetSourceFailures();
}
