/**
 * Index de componentes
 * Re-exporta todos os componentes para facilitar importação
 */

// Layout
export { default as Header } from './layout/Header.svelte';
export { default as NavTabs } from './layout/NavTabs.svelte';

// UI
export { default as SearchBar } from './ui/SearchBar.svelte';
export { default as SplashScreen } from './ui/SplashScreen.svelte';
export { default as LoadingSpinner } from './ui/LoadingSpinner.svelte';

// Anime
export { default as AnimeCard } from './anime/AnimeCard.svelte';
export { default as AnimeGrid } from './anime/AnimeGrid.svelte';
export { default as GenreFilter } from './anime/GenreFilter.svelte';
export { default as FeaturedHero } from './anime/FeaturedHero.svelte';

// Manga
export { default as MangaCard } from './manga/MangaCard.svelte';
export { default as MangaGrid } from './manga/MangaGrid.svelte';
export { default as MangaReader } from './manga/MangaReader.svelte';
export { default as MangaDetails } from './manga/MangaDetails.svelte';

// Views
export { default as LoginScreen } from './views/LoginScreen.svelte';

// Modals
export { default as DiscordLinkModal } from './modals/DiscordLinkModal.svelte';
