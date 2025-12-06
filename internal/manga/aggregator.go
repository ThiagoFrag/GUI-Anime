package manga

import (
	"fmt"
	"strings"
	"sync"
)

// MangaSource representa uma fonte de mangá
type MangaSource interface {
	GetSourceName() string
	GetAllMangas(page int) ([]Manga, int, error)
	GetPopularMangas() ([]Manga, error)
	GetLatestUpdates() ([]Manga, error)
	SearchManga(query string) ([]Manga, error)
	GetMangaDetails(mangaURL string) (*Manga, error)
	GetChapters(mangaURL string) ([]MangaChapter, error)
	GetChapterPages(chapterURL string) ([]MangaPage, error)
	GetMangasByGenre(genre string) ([]Manga, error)
	GetGenres() ([]string, error)
}

// MangaAggregator combina múltiplas fontes de mangá
type MangaAggregator struct {
	sources     map[string]MangaSource
	sourceOrder []string
	mu          sync.RWMutex
}

// NewMangaAggregator cria um novo agregador com todas as fontes disponíveis
func NewMangaAggregator() *MangaAggregator {
	agg := &MangaAggregator{
		sources:     make(map[string]MangaSource),
		sourceOrder: []string{},
	}

	// Adiciona mangalivre.to como fonte primária
	mangaLivreTo := NewMangaClient()
	agg.sources["mangalivre.to"] = &mangaClientAdapter{mangaLivreTo}
	agg.sourceOrder = append(agg.sourceOrder, "mangalivre.to")

	// Adiciona mangalivre.blog como fonte secundária
	mangaLivreBlog := NewMangaLivreBlogClient()
	agg.sources["mangalivre.blog"] = mangaLivreBlog
	agg.sourceOrder = append(agg.sourceOrder, "mangalivre.blog")

	return agg
}

// GetSources retorna lista de fontes disponíveis
func (a *MangaAggregator) GetSources() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sourceOrder
}

// GetSource retorna uma fonte específica pelo nome
func (a *MangaAggregator) GetSource(name string) (MangaSource, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	source, ok := a.sources[name]
	return source, ok
}

// GetAllMangasFromSource busca mangás de uma fonte específica
func (a *MangaAggregator) GetAllMangasFromSource(sourceName string, page int) ([]Manga, int, error) {
	source, ok := a.GetSource(sourceName)
	if !ok {
		return nil, 0, fmt.Errorf("fonte não encontrada: %s", sourceName)
	}

	mangas, totalPages, err := source.GetAllMangas(page)
	if err != nil {
		return nil, 0, err
	}

	// Adiciona informação da fonte em cada mangá
	for i := range mangas {
		mangas[i].ID = sourceName + ":" + mangas[i].ID
	}

	return mangas, totalPages, nil
}

// GetAllMangasFromAllSources busca mangás de todas as fontes
func (a *MangaAggregator) GetAllMangasFromAllSources(page int) ([]Manga, int, error) {
	a.mu.RLock()
	sources := a.sourceOrder
	a.mu.RUnlock()

	var allMangas []Manga
	maxPages := 0
	var lastErr error

	for _, sourceName := range sources {
		source, ok := a.GetSource(sourceName)
		if !ok {
			continue
		}

		mangas, totalPages, err := source.GetAllMangas(page)
		if err != nil {
			lastErr = err
			fmt.Printf("[MangaAggregator] Erro na fonte %s: %v\n", sourceName, err)
			continue
		}

		// Adiciona prefixo da fonte
		for i := range mangas {
			mangas[i].ID = sourceName + ":" + mangas[i].ID
		}

		allMangas = append(allMangas, mangas...)

		if totalPages > maxPages {
			maxPages = totalPages
		}
	}

	if len(allMangas) == 0 && lastErr != nil {
		return nil, 0, lastErr
	}

	return allMangas, maxPages, nil
}

// SearchAllSources busca em todas as fontes
func (a *MangaAggregator) SearchAllSources(query string) ([]Manga, error) {
	a.mu.RLock()
	sources := a.sourceOrder
	a.mu.RUnlock()

	var allMangas []Manga
	var lastErr error
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, sourceName := range sources {
		source, ok := a.GetSource(sourceName)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(sName string, s MangaSource) {
			defer wg.Done()

			mangas, err := s.SearchManga(query)
			if err != nil {
				fmt.Printf("[MangaAggregator] Erro na busca em %s: %v\n", sName, err)
				mu.Lock()
				lastErr = err
				mu.Unlock()
				return
			}

			// Adiciona prefixo da fonte
			for i := range mangas {
				mangas[i].ID = sName + ":" + mangas[i].ID
			}

			mu.Lock()
			allMangas = append(allMangas, mangas...)
			mu.Unlock()
		}(sourceName, source)
	}

	wg.Wait()

	if len(allMangas) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return allMangas, nil
}

// getSourceFromURL determina a fonte a partir da URL
func getSourceFromURL(mangaURL string) string {
	if strings.Contains(mangaURL, "mangalivre.blog") {
		return "mangalivre.blog"
	}
	if strings.Contains(mangaURL, "mangalivre.to") {
		return "mangalivre.to"
	}
	// Default para mangalivre.to
	return "mangalivre.to"
}

// GetMangaDetails obtém detalhes de um mangá (detecta fonte pela URL)
func (a *MangaAggregator) GetMangaDetails(mangaURL string) (*Manga, error) {
	sourceName := getSourceFromURL(mangaURL)
	source, ok := a.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("fonte não encontrada para URL: %s", mangaURL)
	}

	return source.GetMangaDetails(mangaURL)
}

// GetChapters obtém capítulos de um mangá (detecta fonte pela URL)
func (a *MangaAggregator) GetChapters(mangaURL string) ([]MangaChapter, error) {
	sourceName := getSourceFromURL(mangaURL)
	source, ok := a.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("fonte não encontrada para URL: %s", mangaURL)
	}

	return source.GetChapters(mangaURL)
}

// GetChapterPages obtém páginas de um capítulo (detecta fonte pela URL)
func (a *MangaAggregator) GetChapterPages(chapterURL string) ([]MangaPage, error) {
	sourceName := getSourceFromURL(chapterURL)
	source, ok := a.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("fonte não encontrada para URL: %s", chapterURL)
	}

	return source.GetChapterPages(chapterURL)
}

// GetGenresFromAllSources retorna gêneros de todas as fontes
func (a *MangaAggregator) GetGenresFromAllSources() ([]string, error) {
	a.mu.RLock()
	sources := a.sourceOrder
	a.mu.RUnlock()

	seenGenres := make(map[string]bool)
	var allGenres []string

	for _, sourceName := range sources {
		source, ok := a.GetSource(sourceName)
		if !ok {
			continue
		}

		genres, err := source.GetGenres()
		if err != nil {
			fmt.Printf("[MangaAggregator] Erro ao obter gêneros de %s: %v\n", sourceName, err)
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

// mangaClientAdapter adapta o MangaClient original para a interface MangaSource
type mangaClientAdapter struct {
	client *MangaClient
}

func (a *mangaClientAdapter) GetSourceName() string {
	return "MangaLivre.to"
}

func (a *mangaClientAdapter) GetAllMangas(page int) ([]Manga, int, error) {
	return a.client.GetAllMangas(page)
}

func (a *mangaClientAdapter) GetPopularMangas() ([]Manga, error) {
	return a.client.GetPopularMangas()
}

func (a *mangaClientAdapter) GetLatestUpdates() ([]Manga, error) {
	return a.client.GetLatestUpdates()
}

func (a *mangaClientAdapter) SearchManga(query string) ([]Manga, error) {
	return a.client.SearchManga(query)
}

func (a *mangaClientAdapter) GetMangaDetails(mangaURL string) (*Manga, error) {
	return a.client.GetMangaDetails(mangaURL)
}

func (a *mangaClientAdapter) GetChapters(mangaURL string) ([]MangaChapter, error) {
	return a.client.GetChapters(mangaURL)
}

func (a *mangaClientAdapter) GetChapterPages(chapterURL string) ([]MangaPage, error) {
	return a.client.GetChapterPages(chapterURL)
}

func (a *mangaClientAdapter) GetMangasByGenre(genre string) ([]Manga, error) {
	return a.client.GetMangasByGenre(genre)
}

func (a *mangaClientAdapter) GetGenres() ([]string, error) {
	return a.client.GetGenres()
}
