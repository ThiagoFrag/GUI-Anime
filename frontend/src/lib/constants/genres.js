/**
 * Lista de gêneros de anime com termos de busca otimizados
 * Cada gênero contém animes populares para melhorar resultados de busca
 */
export const ANIME_GENRES = [
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

/**
 * Lista de avatares disponíveis para o perfil do usuário
 */
export const AVATARS = [
    { id: 'avatar1.png', emoji: '👤', label: 'Usuário' },
    { id: 'avatar2.png', emoji: '🦊', label: 'Raposa' },
    { id: 'avatar3.png', emoji: '🤖', label: 'Robô' },
    { id: 'avatar4.png', emoji: '🐱', label: 'Gato' },
    { id: 'avatar5.png', emoji: '🎮', label: 'Gamer' },
    { id: 'avatar6.png', emoji: '⚡', label: 'Energia' }
];

/**
 * Configurações padrão do aplicativo
 */
export const DEFAULT_SETTINGS = {
    start_fullscreen: false,
    content_language: 'all',
    default_quality: 'auto',
    use_anime4k: true
};
