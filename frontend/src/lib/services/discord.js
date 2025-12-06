/**
 * Serviço para interação com Discord
 * Encapsula todas as chamadas ao backend Go relacionadas ao Discord
 */
import { 
    GetDiscordLinkStatus, LinkDiscordWithCode, UnlinkDiscord,
    GetDiscordServerInvite, UpdateDiscordWatchingStatus, 
    GetDiscordFriendsActivity, SetDiscordShowStatus, SetDiscordShareAnimes,
    SendDiscordRecommendation
} from '../../../wailsjs/go/main/App';

/**
 * Obtém status da vinculação Discord
 * @returns {Promise<Object>}
 */
export async function getLinkStatus() {
    try {
        return await GetDiscordLinkStatus();
    } catch (err) {
        console.error('[getLinkStatus] Error:', err);
        return { isLinked: false };
    }
}

/**
 * Vincula conta Discord usando código
 * @param {string} code - Código de vinculação
 * @returns {Promise<Object>}
 */
export async function linkWithCode(code) {
    return await LinkDiscordWithCode(code.trim());
}

/**
 * Desvincula conta Discord
 */
export async function unlink() {
    await UnlinkDiscord();
}

/**
 * Obtém link do servidor Discord
 * @returns {Promise<string>}
 */
export async function getServerInvite() {
    try {
        return await GetDiscordServerInvite();
    } catch (err) {
        console.error('[getServerInvite] Error:', err);
        return '';
    }
}

/**
 * Atualiza status de "assistindo"
 * @param {string} animeTitle - Título do anime
 * @param {number} episodeNum - Número do episódio
 * @param {string} animeImage - URL da imagem do anime
 * @param {number} totalEpisodes - Total de episódios
 */
export async function updateWatchingStatus(animeTitle, episodeNum, animeImage, totalEpisodes = 0) {
    try {
        await UpdateDiscordWatchingStatus(animeTitle, episodeNum, animeImage, totalEpisodes);
    } catch (err) {
        console.error('[updateWatchingStatus] Error:', err);
    }
}

/**
 * Obtém atividade dos amigos
 * @returns {Promise<Array>}
 */
export async function getFriendsActivity() {
    try {
        const activities = await GetDiscordFriendsActivity();
        return activities || [];
    } catch (err) {
        console.error('[getFriendsActivity] Error:', err);
        return [];
    }
}

/**
 * Define se deve mostrar status
 * @param {boolean} value - Novo valor
 */
export async function setShowStatus(value) {
    await SetDiscordShowStatus(value);
}

/**
 * Define se deve compartilhar animes
 * @param {boolean} value - Novo valor
 */
export async function setShareAnimes(value) {
    await SetDiscordShareAnimes(value);
}

/**
 * Envia recomendação para amigos
 * @param {string} title - Título do anime
 * @param {string} image - URL da imagem
 * @param {number} score - Score do anime
 * @param {string} message - Mensagem personalizada
 */
export async function sendRecommendation(title, image, score, message) {
    await SendDiscordRecommendation(title, image, score, message);
}
