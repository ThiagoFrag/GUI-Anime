/**
 * Serviço para interação com API de Mangás
 * Encapsula todas as chamadas ao backend Go relacionadas a mangás
 */
import { 
    GetPopularMangas, GetLatestMangas, SearchMangas, 
    GetMangaDetails, GetMangaChapters, GetChapterPages,
    GetMangasByGenre
} from '../../../wailsjs/go/main/App';
import { withTimeout } from '../utils/helpers.js';

/**
 * Carrega mangás populares
 * @returns {Promise<Array>}
 */
export async function loadPopularMangas() {
    try {
        const mangas = await withTimeout(GetPopularMangas(), 10000);
        return Array.isArray(mangas) ? mangas : [];
    } catch (error) {
        console.error('[manga.js] Erro ao carregar populares:', error);
        return [];
    }
}

/**
 * Carrega mangás com atualizações recentes
 * @returns {Promise<Array>}
 */
export async function loadLatestMangas() {
    try {
        const mangas = await withTimeout(GetLatestMangas(), 10000);
        return Array.isArray(mangas) ? mangas : [];
    } catch (error) {
        console.error('[manga.js] Erro ao carregar últimos:', error);
        return [];
    }
}

/**
 * Busca mangás por termo
 * @param {string} query - Termo de busca
 * @returns {Promise<Array>}
 */
export async function searchMangas(query) {
    if (!query) return [];
    try {
        const mangas = await SearchMangas(query);
        return Array.isArray(mangas) ? mangas : [];
    } catch (error) {
        console.error('[manga.js] Erro na busca:', error);
        return [];
    }
}

/**
 * Obtém detalhes completos de um mangá
 * @param {string} mangaUrl - URL do mangá
 * @returns {Promise<Object|null>}
 */
export async function getMangaDetails(mangaUrl) {
    try {
        return await GetMangaDetails(mangaUrl);
    } catch (error) {
        console.error('[manga.js] Erro ao obter detalhes:', error);
        return null;
    }
}

/**
 * Obtém lista de capítulos de um mangá
 * @param {string} mangaUrl - URL do mangá
 * @returns {Promise<Array>}
 */
export async function getMangaChapters(mangaUrl) {
    try {
        const chapters = await GetMangaChapters(mangaUrl);
        return Array.isArray(chapters) ? chapters : [];
    } catch (error) {
        console.error('[manga.js] Erro ao obter capítulos:', error);
        return [];
    }
}

/**
 * Obtém as páginas (imagens) de um capítulo
 * @param {string} chapterUrl - URL do capítulo
 * @returns {Promise<Array>}
 */
export async function getChapterPages(chapterUrl) {
    try {
        const pages = await GetChapterPages(chapterUrl);
        return Array.isArray(pages) ? pages : [];
    } catch (error) {
        console.error('[manga.js] Erro ao obter páginas:', error);
        return [];
    }
}

/**
 * Obtém mangás de um gênero específico
 * @param {string} genre - Nome do gênero
 * @returns {Promise<Array>}
 */
export async function getMangasByGenre(genre) {
    try {
        const mangas = await GetMangasByGenre(genre);
        return Array.isArray(mangas) ? mangas : [];
    } catch (error) {
        console.error('[manga.js] Erro ao obter por gênero:', error);
        return [];
    }
}

/**
 * Carrega dados iniciais de mangás (populares e últimos)
 * @returns {Promise<{popular: Array, latest: Array}>}
 */
export async function loadInitialMangaData() {
    const [popularRes, latestRes] = await Promise.allSettled([
        withTimeout(GetPopularMangas(), 10000),
        withTimeout(GetLatestMangas(), 10000)
    ]);
    
    return {
        popular: popularRes.status === 'fulfilled' ? popularRes.value || [] : [],
        latest: latestRes.status === 'fulfilled' ? latestRes.value || [] : []
    };
}

// Lista de gêneros de mangá disponíveis
export const MANGA_GENRES = [
    '+18', 'Ação', 'Adulto', 'Artes Marciais', 'Aventura', 'Comédia',
    'Demônios', 'Doujinshi', 'Drama', 'Ecchi', 'Escolar', 'Esportes',
    'Fantasia', 'Harem', 'Historico', 'Isekai', 'Light Novels', 'Mangá',
    'Manhuas', 'Manhwa', 'Psicológico', 'Reencarnação', 'Romance', 'Seinen',
    'Shoujo', 'Shounen', 'Slice of Life', 'Sobrenatural', 'Suspense',
    'Tragédia', 'Vampiros', 'Webtoon', 'Yuri'
];
