/**
 * Utilitários gerais do aplicativo
 */

/**
 * Formata tempo relativo (ex: "há 5 minutos")
 * @param {number} timestamp - Timestamp em milissegundos
 * @returns {string} Tempo formatado
 */
export function formatTimeAgo(timestamp) {
    const diff = Date.now() - timestamp;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (days > 0) return `${days}d atrás`;
    if (hours > 0) return `${hours}h atrás`;
    if (minutes > 0) return `${minutes}min atrás`;
    return 'agora';
}

/**
 * Cria uma promise com timeout
 * @param {Promise} promise - Promise original
 * @param {number} ms - Timeout em milissegundos
 * @returns {Promise} Promise com timeout
 */
export function withTimeout(promise, ms) {
    return Promise.race([
        promise,
        new Promise((_, reject) => 
            setTimeout(() => reject(new Error('Timeout')), ms)
        )
    ]);
}

/**
 * Debounce de função
 * @param {Function} fn - Função a ser executada
 * @param {number} delay - Delay em milissegundos
 * @returns {Function} Função com debounce
 */
export function debounce(fn, delay = 300) {
    let timeoutId;
    return (...args) => {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => fn(...args), delay);
    };
}

/**
 * Throttle de função
 * @param {Function} fn - Função a ser executada
 * @param {number} limit - Limite em milissegundos
 * @returns {Function} Função com throttle
 */
export function throttle(fn, limit) {
    let inThrottle;
    return (...args) => {
        if (!inThrottle) {
            fn(...args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

/**
 * Scroll suave para o topo de um elemento
 * @param {HTMLElement} element - Elemento para scroll
 * @param {boolean} smooth - Se deve usar animação suave
 */
export function scrollToTop(element, smooth = true) {
    if (element) {
        element.scrollTo({ 
            top: 0, 
            behavior: smooth ? 'smooth' : 'instant' 
        });
    }
}

/**
 * Copia texto para a área de transferência
 * @param {string} text - Texto a copiar
 * @returns {Promise<boolean>} Se a cópia foi bem sucedida
 */
export async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch {
        return false;
    }
}

/**
 * Gera um ID único
 * @returns {string} ID único
 */
export function generateId() {
    return Math.random().toString(36).substring(2, 9);
}

/**
 * Verifica se uma URL é de HLS (m3u8)
 * @param {string} url - URL a verificar
 * @returns {boolean}
 */
export function isHLSUrl(url) {
    return url?.includes('.m3u8') || url?.includes('m3u8');
}

/**
 * Verifica se uma URL é do SharePoint/OneDrive
 * @param {string} url - URL a verificar
 * @returns {boolean}
 */
export function isSharePointUrl(url) {
    return url?.includes('sharepoint.com') || 
           url?.includes('microsoft.com') ||
           url?.includes('onedrive') ||
           url?.includes('download.aspx');
}

/**
 * Trunca texto com ellipsis
 * @param {string} text - Texto original
 * @param {number} maxLength - Comprimento máximo
 * @returns {string} Texto truncado
 */
export function truncateText(text, maxLength = 100) {
    if (!text || text.length <= maxLength) return text;
    return text.substring(0, maxLength - 3) + '...';
}

/**
 * Formata número com separadores de milhar
 * @param {number} num - Número a formatar
 * @returns {string} Número formatado
 */
export function formatNumber(num) {
    return new Intl.NumberFormat('pt-BR').format(num);
}

/**
 * Calcula score de cor baseado em valor (0-10)
 * @param {number} score - Score de 0 a 10
 * @returns {string} Classe CSS de cor
 */
export function getScoreColorClass(score) {
    if (score >= 8) return 'score-high';
    if (score >= 6) return 'score-medium';
    return 'score-low';
}
