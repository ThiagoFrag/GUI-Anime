package consumet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Consumet API client
// Fornece links de streaming de múltiplas fontes (Gogoanime, Zoro, etc)
// Documentação: https://docs.consumet.org/

// Endpoints públicos da Consumet API
// Você pode rodar sua própria instância para maior velocidade
const (
	// API pública (pode ter rate limits)
	DefaultAPIURL = "https://api.consumet.org"
	// Provedores suportados
	ProviderGogoanime = "gogoanime"
	ProviderZoro      = "zoro"
	ProviderAnimePahe = "animepahe"
	Provider9Anime    = "9anime"
	ProviderEnime     = "enime"
)

var (
	httpClient = &http.Client{Timeout: 15 * time.Second}
	apiURL     = DefaultAPIURL
	cache      = make(map[string]interface{})
	cacheMutex sync.RWMutex
)

// SetAPIURL permite configurar uma instância própria da API
func SetAPIURL(url string) {
	apiURL = strings.TrimSuffix(url, "/")
}

// AnimeInfo representa informações básicas do anime
type AnimeInfo struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Image         string   `json:"image"`
	ReleaseDate   string   `json:"releaseDate"`
	SubOrDub      string   `json:"subOrDub"`
	TotalEpisodes int      `json:"totalEpisodes,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Description   string   `json:"description,omitempty"`
	Status        string   `json:"status,omitempty"`
}

// Episode representa um episódio
type Episode struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
	Image       string `json:"image,omitempty"`
	Description string `json:"description,omitempty"`
}

// StreamingSource representa uma fonte de streaming
type StreamingSource struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
	IsM3U8  bool   `json:"isM3U8"`
}

// StreamingInfo contém todas as fontes de streaming de um episódio
type StreamingInfo struct {
	Headers  map[string]string `json:"headers,omitempty"`
	Sources  []StreamingSource `json:"sources"`
	Download string            `json:"download,omitempty"`
}

// SearchResult representa o resultado de uma busca
type SearchResult struct {
	CurrentPage int         `json:"currentPage"`
	HasNextPage bool        `json:"hasNextPage"`
	Results     []AnimeInfo `json:"results"`
}

// AnimeDetails representa detalhes completos de um anime
type AnimeDetails struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	Image         string    `json:"image"`
	ReleaseDate   string    `json:"releaseDate"`
	Description   string    `json:"description"`
	Genres        []string  `json:"genres"`
	SubOrDub      string    `json:"subOrDub"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	OtherNames    []string  `json:"otherName"`
	TotalEpisodes int       `json:"totalEpisodes"`
	Episodes      []Episode `json:"episodes"`
}

// Search busca animes em um provedor específico
func Search(query string, provider string) (*SearchResult, error) {
	if provider == "" {
		provider = ProviderGogoanime
	}

	cacheKey := fmt.Sprintf("search:%s:%s", provider, strings.ToLower(query))
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].(*SearchResult); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/anime/%s/%s", apiURL, provider, url.PathEscape(query))

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro na busca: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = &result
	cacheMutex.Unlock()

	return &result, nil
}

// SearchMultiProvider busca em múltiplos provedores simultaneamente
func SearchMultiProvider(query string) ([]AnimeInfo, error) {
	providers := []string{ProviderGogoanime, ProviderZoro}
	var allResults []AnimeInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, provider := range providers {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			result, err := Search(query, p)
			if err == nil && result != nil {
				mu.Lock()
				for _, anime := range result.Results {
					anime.ID = fmt.Sprintf("%s:%s", p, anime.ID) // Prefixo do provider
					allResults = append(allResults, anime)
				}
				mu.Unlock()
			}
		}(provider)
	}

	wg.Wait()
	return allResults, nil
}

// GetAnimeInfo retorna detalhes de um anime específico
func GetAnimeInfo(animeID string, provider string) (*AnimeDetails, error) {
	if provider == "" {
		provider = ProviderGogoanime
	}

	cacheKey := fmt.Sprintf("info:%s:%s", provider, animeID)
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].(*AnimeDetails); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/anime/%s/info/%s", apiURL, provider, url.PathEscape(animeID))

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var details AnimeDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = &details
	cacheMutex.Unlock()

	return &details, nil
}

// GetEpisodeSources retorna as fontes de streaming de um episódio
func GetEpisodeSources(episodeID string, provider string) (*StreamingInfo, error) {
	if provider == "" {
		provider = ProviderGogoanime
	}

	// Não cacheia streams pois podem expirar
	endpoint := fmt.Sprintf("%s/anime/%s/watch/%s", apiURL, provider, url.PathEscape(episodeID))

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var info StreamingInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	return &info, nil
}

// GetBestStream retorna a melhor URL de streaming disponível
func GetBestStream(episodeID string, provider string) (string, bool, error) {
	info, err := GetEpisodeSources(episodeID, provider)
	if err != nil {
		return "", false, err
	}

	if len(info.Sources) == 0 {
		return "", false, fmt.Errorf("nenhuma fonte encontrada")
	}

	// Prioridade: 1080p > 720p > 480p > default
	priorities := []string{"1080p", "1080", "720p", "720", "480p", "480", "360p", "360", "default", "auto"}

	for _, quality := range priorities {
		for _, source := range info.Sources {
			if strings.Contains(strings.ToLower(source.Quality), strings.ToLower(quality)) {
				return source.URL, source.IsM3U8, nil
			}
		}
	}

	// Retorna a primeira se nenhuma prioridade encontrada
	return info.Sources[0].URL, info.Sources[0].IsM3U8, nil
}

// GetRecentEpisodes retorna episódios recentes
func GetRecentEpisodes(provider string, page int) ([]AnimeInfo, error) {
	if provider == "" {
		provider = ProviderGogoanime
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("%s/anime/%s/recent-episodes?page=%d", apiURL, provider, page)

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

// GetTopAiring retorna animes em exibição mais populares
func GetTopAiring(provider string, page int) ([]AnimeInfo, error) {
	if provider == "" {
		provider = ProviderGogoanime
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("%s/anime/%s/top-airing?page=%d", apiURL, provider, page)

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

// FindAnimeAndGetStream busca um anime e retorna o stream do primeiro episódio
// Útil como fallback quando o scraper principal falha
func FindAnimeAndGetStream(title string, episodeNumber int) (string, bool, error) {
	// Tenta com Gogoanime primeiro
	results, err := Search(title, ProviderGogoanime)
	if err != nil || len(results.Results) == 0 {
		// Fallback para Zoro
		results, err = Search(title, ProviderZoro)
		if err != nil || len(results.Results) == 0 {
			return "", false, fmt.Errorf("anime não encontrado em nenhum provedor")
		}
	}

	// Pega o primeiro resultado
	animeID := results.Results[0].ID
	provider := ProviderGogoanime
	if strings.HasPrefix(animeID, "zoro:") {
		provider = ProviderZoro
		animeID = strings.TrimPrefix(animeID, "zoro:")
	}

	// Busca detalhes para pegar episódios
	details, err := GetAnimeInfo(animeID, provider)
	if err != nil {
		return "", false, err
	}

	// Encontra o episódio desejado
	var episodeID string
	for _, ep := range details.Episodes {
		if ep.Number == episodeNumber {
			episodeID = ep.ID
			break
		}
	}

	if episodeID == "" {
		// Tenta construir o ID baseado em padrões comuns
		if provider == ProviderGogoanime {
			episodeID = fmt.Sprintf("%s-episode-%d", animeID, episodeNumber)
		} else {
			return "", false, fmt.Errorf("episódio %d não encontrado", episodeNumber)
		}
	}

	return GetBestStream(episodeID, provider)
}
