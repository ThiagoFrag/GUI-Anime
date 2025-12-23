// TorBox Service - Streaming de anime via torrent
import { InitTorBox, TorBoxGetUser, TorBoxSearchTorrents, TorBoxGetInstantStream, 
         TorBoxCheckCached, TorBoxGetTorrents, TorBoxAddMagnet, TorBoxDeleteTorrent,
         TorBoxGetDownloadLink, TorBoxClearCache, TorBoxStreamAnimeEpisode } from '../../wailsjs/go/main/App';

// API key do TorBox (deve ser configurado pelo usuário)
let apiKeyConfigured = false;

/**
 * Inicializa o cliente TorBox com a API key
 * @param {string} apiKey - API key do TorBox
 * @returns {Promise<boolean>} - true se inicializado com sucesso
 */
export async function initTorBox(apiKey) {
    try {
        const result = await InitTorBox(apiKey);
        apiKeyConfigured = result;
        return result;
    } catch (error) {
        console.error('[TorBox] Erro ao inicializar:', error);
        return false;
    }
}

/**
 * Verifica se o TorBox está configurado
 * @returns {boolean}
 */
export function isTorBoxConfigured() {
    return apiKeyConfigured;
}

/**
 * Obtém informações do usuário TorBox
 * @returns {Promise<Object|null>}
 */
export async function getTorBoxUser() {
    try {
        return await TorBoxGetUser();
    } catch (error) {
        console.error('[TorBox] Erro ao obter usuário:', error);
        return null;
    }
}

/**
 * Busca torrents de anime no Nyaa.si
 * @param {string} query - Termo de busca
 * @returns {Promise<Array>}
 */
export async function searchTorrents(query) {
    try {
        return await TorBoxSearchTorrents(query) || [];
    } catch (error) {
        console.error('[TorBox] Erro na busca:', error);
        return [];
    }
}

/**
 * Busca e retorna link de streaming direto
 * Função principal para assistir anime
 * @param {string} query - Termo de busca (ex: "Frieren 01 1080p")
 * @returns {Promise<Object|null>}
 */
export async function getInstantStream(query) {
    try {
        return await TorBoxGetInstantStream(query);
    } catch (error) {
        console.error('[TorBox] Erro ao obter stream:', error);
        return null;
    }
}

/**
 * Busca stream de um episódio específico
 * @param {string} animeTitle - Título do anime
 * @param {number} episode - Número do episódio
 * @param {string} quality - Qualidade (1080p, 720p, etc)
 * @returns {Promise<Object|null>}
 */
export async function streamAnimeEpisode(animeTitle, episode, quality = '1080p') {
    try {
        return await TorBoxStreamAnimeEpisode(animeTitle, episode, quality);
    } catch (error) {
        console.error('[TorBox] Erro ao obter stream do episódio:', error);
        return null;
    }
}

/**
 * Verifica se hashes estão em cache no TorBox
 * @param {Array<string>} hashes - Lista de hashes
 * @returns {Promise<Object>}
 */
export async function checkCached(hashes) {
    try {
        return await TorBoxCheckCached(hashes) || {};
    } catch (error) {
        console.error('[TorBox] Erro ao verificar cache:', error);
        return {};
    }
}

/**
 * Lista todos os torrents do usuário
 * @returns {Promise<Array>}
 */
export async function getTorrents() {
    try {
        return await TorBoxGetTorrents() || [];
    } catch (error) {
        console.error('[TorBox] Erro ao listar torrents:', error);
        return [];
    }
}

/**
 * Adiciona um torrent via magnet link
 * @param {string} magnet - Link magnet
 * @returns {Promise<Object|null>}
 */
export async function addMagnet(magnet) {
    try {
        return await TorBoxAddMagnet(magnet);
    } catch (error) {
        console.error('[TorBox] Erro ao adicionar magnet:', error);
        return null;
    }
}

/**
 * Remove um torrent
 * @param {number} torrentID - ID do torrent
 * @returns {Promise<boolean>}
 */
export async function deleteTorrent(torrentID) {
    try {
        return await TorBoxDeleteTorrent(torrentID);
    } catch (error) {
        console.error('[TorBox] Erro ao deletar torrent:', error);
        return false;
    }
}

/**
 * Obtém link direto de download/streaming
 * @param {number} torrentID - ID do torrent
 * @param {number} fileID - ID do arquivo
 * @returns {Promise<string>}
 */
export async function getDownloadLink(torrentID, fileID) {
    try {
        return await TorBoxGetDownloadLink(torrentID, fileID) || '';
    } catch (error) {
        console.error('[TorBox] Erro ao obter link:', error);
        return '';
    }
}

/**
 * Limpa o cache local do TorBox
 */
export async function clearCache() {
    try {
        await TorBoxClearCache();
    } catch (error) {
        console.error('[TorBox] Erro ao limpar cache:', error);
    }
}

// Exporta tudo como default também
export default {
    initTorBox,
    isTorBoxConfigured,
    getTorBoxUser,
    searchTorrents,
    getInstantStream,
    streamAnimeEpisode,
    checkCached,
    getTorrents,
    addMagnet,
    deleteTorrent,
    getDownloadLink,
    clearCache
};

