/**
 * GoAnime Frontend Library
 * 
 * Estrutura organizada:
 * - components/: Componentes Svelte reutilizáveis
 * - stores/: Estados globais (Svelte 5 runes)
 * - services/: Chamadas ao backend Go
 * - utils/: Funções utilitárias
 * - constants/: Constantes e configurações
 */

// Components
export * from './components/index.js';

// Stores
export * from './stores/index.svelte.js';

// Services
export * from './services/index.js';

// Utils
export * from './utils/helpers.js';

// Constants
export * from './constants/genres.js';
