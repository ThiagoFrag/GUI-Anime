/**
 * Serviço para interação com API de usuário
 * Encapsula todas as chamadas ao backend Go relacionadas ao usuário
 */
import { 
    GetCurrentUser, CreateUser, GetSettings, SaveSettings,
    GetFavorites, AddToFavorites, RemoveFromFavorites, IsFavorite,
    GetWatchHistory, AddToWatchHistory,
    ExportUserData, ImportUserData
} from '../../../wailsjs/go/main/App';

/**
 * Obtém usuário atual
 * @returns {Promise<Object|null>}
 */
export async function getCurrentUser() {
    try {
        const user = await GetCurrentUser();
        return user?.username ? user : null;
    } catch (err) {
        console.error('[getCurrentUser] Error:', err);
        return null;
    }
}

/**
 * Cria novo usuário
 * @param {string} username - Nome do usuário
 * @param {string} avatar - Avatar selecionado
 * @returns {Promise<Object>}
 */
export async function createUser(username, avatar) {
    return await CreateUser(username, avatar);
}

/**
 * Obtém configurações do usuário
 * @returns {Promise<Object>}
 */
export async function getSettings() {
    try {
        const settings = await GetSettings();
        return settings || {
            start_fullscreen: false,
            content_language: 'all',
            default_quality: 'auto',
            use_anime4k: true
        };
    } catch (err) {
        console.error('[getSettings] Error:', err);
        return {
            start_fullscreen: false,
            content_language: 'all',
            default_quality: 'auto',
            use_anime4k: true
        };
    }
}

/**
 * Salva configurações do usuário
 * @param {Object} settings - Configurações a salvar
 */
export async function saveSettings(settings) {
    await SaveSettings(settings);
}

/**
 * Obtém lista de favoritos
 * @returns {Promise<Array>}
 */
export async function getFavorites() {
    try {
        const favs = await GetFavorites();
        return favs || [];
    } catch (err) {
        console.error('[getFavorites] Error:', err);
        return [];
    }
}

/**
 * Adiciona anime aos favoritos
 * @param {Object} anime - Anime a adicionar
 */
export async function addToFavorites(anime) {
    await AddToFavorites(anime);
}

/**
 * Remove anime dos favoritos
 * @param {string} animeUrl - URL do anime
 */
export async function removeFromFavorites(animeUrl) {
    await RemoveFromFavorites(animeUrl);
}

/**
 * Verifica se anime está nos favoritos
 * @param {string} animeUrl - URL do anime
 * @returns {Promise<boolean>}
 */
export async function isFavorite(animeUrl) {
    return await IsFavorite(animeUrl);
}

/**
 * Alterna favorito (adiciona/remove)
 * @param {Object} anime - Anime
 * @returns {Promise<boolean>} Novo estado (true = favorito)
 */
export async function toggleFavorite(anime) {
    const isFav = await IsFavorite(anime.URL);
    if (isFav) {
        await RemoveFromFavorites(anime.URL);
        return false;
    } else {
        await AddToFavorites(anime);
        return true;
    }
}

/**
 * Obtém histórico de visualização
 * @returns {Promise<Array>}
 */
export async function getWatchHistory() {
    try {
        const history = await GetWatchHistory();
        return history || [];
    } catch (err) {
        console.error('[getWatchHistory] Error:', err);
        return [];
    }
}

/**
 * Adiciona ao histórico de visualização
 * @param {Object} anime - Anime assistido
 * @param {Object} episode - Episódio assistido
 */
export async function addToHistory(anime, episode) {
    await AddToWatchHistory(anime, episode);
}

/**
 * Exporta dados do usuário
 * @returns {Promise<string>} JSON exportado
 */
export async function exportUserData() {
    return await ExportUserData();
}

/**
 * Importa dados do usuário
 * @param {string} jsonData - JSON a importar
 */
export async function importUserData(jsonData) {
    await ImportUserData(jsonData);
}
