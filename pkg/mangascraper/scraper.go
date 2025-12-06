package mangascraper

import (
	"fmt"
	"strings"
	"sync"
)

// Scraper is the main entry point for the manga scraper library
type Scraper struct {
	sources map[string]Source
	order   []string
	cache   *Cache
	config  *Config
	mu      sync.RWMutex
}

// New creates a new Scraper with default sources and configuration
func New() *Scraper {
	return NewWithConfig(nil)
}

// NewWithConfig creates a new Scraper with custom configuration
func NewWithConfig(config *Config) *Scraper {
	if config == nil {
		config = DefaultConfig()
	}

	var cache *Cache
	if config.EnableCache {
		cache = NewCache(config.CacheDir)
	} else {
		cache = NewDisabledCache()
	}

	s := &Scraper{
		sources: make(map[string]Source),
		order:   []string{},
		cache:   cache,
		config:  config,
	}

	// Register default sources
	s.RegisterSource(NewMangaLivreToSource(config))
	s.RegisterSource(NewMangaLivreBlogSource(config))

	return s
}

// RegisterSource adds a new source to the scraper
func (s *Scraper) RegisterSource(source Source) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := source.Name()
	s.sources[name] = source
	s.order = append(s.order, name)
}

// GetSources returns the list of available source names
func (s *Scraper) GetSources() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string{}, s.order...)
}

// GetSourceInfo returns information about all available sources
func (s *Scraper) GetSourceInfo() []SourceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]SourceInfo, 0, len(s.order))
	for _, name := range s.order {
		source := s.sources[name]
		infos = append(infos, SourceInfo{
			Name:        source.Name(),
			DisplayName: source.DisplayName(),
			BaseURL:     source.BaseURL(),
			Language:    "pt-BR",
			NSFW:        false,
		})
	}
	return infos
}

// GetSource returns a specific source by name
func (s *Scraper) GetSource(name string) (Source, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	source, ok := s.sources[name]
	return source, ok
}

// DetectSourceFromURL determines the source based on URL
func (s *Scraper) DetectSourceFromURL(url string) string {
	lowerURL := strings.ToLower(url)

	if strings.Contains(lowerURL, "mangalivre.blog") {
		return "mangalivre.blog"
	}
	if strings.Contains(lowerURL, "mangalivre.to") {
		return "mangalivre.to"
	}

	// Default to first source
	if len(s.order) > 0 {
		return s.order[0]
	}
	return ""
}

// GetAllMangas returns mangas from a specific source with pagination
func (s *Scraper) GetAllMangas(sourceName string, page int) ([]Manga, int, error) {
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, 0, fmt.Errorf("source not found: %s", sourceName)
	}

	// Check cache
	cacheKey := fmt.Sprintf("mangas:%s:page:%d", sourceName, page)
	if mangas, ok := s.cache.GetMangas(cacheKey); ok {
		return mangas, 0, nil // totalPages not cached, but that's OK
	}

	mangas, totalPages, err := source.GetAllMangas(page)
	if err != nil {
		return nil, 0, err
	}

	// Store in cache
	s.cache.SetMangas(cacheKey, mangas, TTLMangaList)

	return mangas, totalPages, nil
}

// GetAllMangasFromAllSources returns mangas from all sources
func (s *Scraper) GetAllMangasFromAllSources(page int) ([]Manga, int, error) {
	s.mu.RLock()
	sources := s.order
	s.mu.RUnlock()

	var allMangas []Manga
	maxPages := 0
	var lastErr error
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, sourceName := range sources {
		source, ok := s.GetSource(sourceName)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(sName string, src Source) {
			defer wg.Done()

			mangas, totalPages, err := src.GetAllMangas(page)
			if err != nil {
				mu.Lock()
				lastErr = err
				mu.Unlock()
				return
			}

			mu.Lock()
			allMangas = append(allMangas, mangas...)
			if totalPages > maxPages {
				maxPages = totalPages
			}
			mu.Unlock()
		}(sourceName, source)
	}

	wg.Wait()

	if len(allMangas) == 0 && lastErr != nil {
		return nil, 0, lastErr
	}

	return allMangas, maxPages, nil
}

// SearchManga searches for mangas in a specific source
func (s *Scraper) SearchManga(sourceName, query string) ([]Manga, error) {
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}

	// Check cache
	cacheKey := fmt.Sprintf("search:%s:%s", sourceName, query)
	if mangas, ok := s.cache.GetMangas(cacheKey); ok {
		return mangas, nil
	}

	mangas, err := source.SearchManga(query)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.SetMangas(cacheKey, mangas, TTLMangaSearch)

	return mangas, nil
}

// SearchAllSources searches for mangas across all sources
func (s *Scraper) SearchAllSources(query string) ([]SearchResult, error) {
	s.mu.RLock()
	sources := s.order
	s.mu.RUnlock()

	results := make([]SearchResult, 0, len(sources))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, sourceName := range sources {
		source, ok := s.GetSource(sourceName)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(sName string, src Source) {
			defer wg.Done()

			mangas, err := src.SearchManga(query)

			mu.Lock()
			results = append(results, SearchResult{
				Mangas: mangas,
				Source: sName,
				Error:  err,
			})
			mu.Unlock()
		}(sourceName, source)
	}

	wg.Wait()

	return results, nil
}

// GetMangaDetails returns detailed information about a manga
// The source is automatically detected from the URL
func (s *Scraper) GetMangaDetails(mangaURL string) (*Manga, error) {
	sourceName := s.DetectSourceFromURL(mangaURL)
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found for URL: %s", mangaURL)
	}

	// Check cache
	cacheKey := fmt.Sprintf("details:%s", mangaURL)
	if mangas, ok := s.cache.GetMangas(cacheKey); ok && len(mangas) > 0 {
		return &mangas[0], nil
	}

	manga, err := source.GetMangaDetails(mangaURL)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.SetMangas(cacheKey, []Manga{*manga}, TTLMangaDetails)

	return manga, nil
}

// GetChapters returns all chapters of a manga
// The source is automatically detected from the URL
func (s *Scraper) GetChapters(mangaURL string) ([]Chapter, error) {
	sourceName := s.DetectSourceFromURL(mangaURL)
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found for URL: %s", mangaURL)
	}

	// Check cache
	cacheKey := fmt.Sprintf("chapters:%s", mangaURL)
	if chapters, ok := s.cache.GetChapters(cacheKey); ok {
		return chapters, nil
	}

	chapters, err := source.GetChapters(mangaURL)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.SetChapters(cacheKey, chapters, TTLMangaChapters)

	return chapters, nil
}

// GetChapterPages returns all pages/images of a chapter
// The source is automatically detected from the URL
func (s *Scraper) GetChapterPages(chapterURL string) ([]Page, error) {
	sourceName := s.DetectSourceFromURL(chapterURL)
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found for URL: %s", chapterURL)
	}

	// Check cache
	cacheKey := fmt.Sprintf("pages:%s", chapterURL)
	if pages, ok := s.cache.GetPages(cacheKey); ok {
		return pages, nil
	}

	pages, err := source.GetChapterPages(chapterURL)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.SetPages(cacheKey, pages, TTLMangaPages)

	return pages, nil
}

// GetPopularMangas returns popular mangas from a specific source
func (s *Scraper) GetPopularMangas(sourceName string) ([]Manga, error) {
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}

	return source.GetPopularMangas()
}

// GetLatestUpdates returns recently updated mangas from a specific source
func (s *Scraper) GetLatestUpdates(sourceName string) ([]Manga, error) {
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}

	return source.GetLatestUpdates()
}

// GetMangasByGenre returns mangas filtered by genre from a specific source
func (s *Scraper) GetMangasByGenre(sourceName, genre string) ([]Manga, error) {
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}

	return source.GetMangasByGenre(genre)
}

// GetGenres returns available genres from a specific source
func (s *Scraper) GetGenres(sourceName string) ([]string, error) {
	source, ok := s.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}

	return source.GetGenres()
}

// GetAllGenres returns genres from all sources (deduplicated)
func (s *Scraper) GetAllGenres() ([]string, error) {
	s.mu.RLock()
	sources := s.order
	s.mu.RUnlock()

	seenGenres := make(map[string]bool)
	var allGenres []string

	for _, sourceName := range sources {
		source, ok := s.GetSource(sourceName)
		if !ok {
			continue
		}

		genres, err := source.GetGenres()
		if err != nil {
			continue
		}

		for _, genre := range genres {
			if !seenGenres[genre] {
				seenGenres[genre] = true
				allGenres = append(allGenres, genre)
			}
		}
	}

	return allGenres, nil
}

// ClearCache clears all cached data
func (s *Scraper) ClearCache() {
	s.cache.Clear()
}

// GetCache returns the cache instance for advanced usage
func (s *Scraper) GetCache() *Cache {
	return s.cache
}
