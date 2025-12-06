package enime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Enime API - Fonte de vídeo rápida e confiável
// API pública e gratuita
const (
	BaseURL = "https://api.enime.moe"
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	cache      = make(map[string]interface{})
	cacheMutex sync.RWMutex
)

// Anime representa um anime da Enime API
type Anime struct {
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Title         Title     `json:"title"`
	CoverImage    string    `json:"coverImage"`
	BannerImage   string    `json:"bannerImage"`
	Status        string    `json:"status"`
	Format        string    `json:"format"`
	Episodes      []Episode `json:"episodes,omitempty"`
	TotalEpisodes int       `json:"currentEpisode"`
	AnilistID     int       `json:"anilistId"`
	MalID         int       `json:"malId"`
	Description   string    `json:"description"`
	Genre         []string  `json:"genre"`
}

// Title representa os diferentes títulos de um anime
type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

// Episode representa um episódio
type Episode struct {
	ID          string   `json:"id"`
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Image       string   `json:"image,omitempty"`
	AiredAt     string   `json:"airedAt,omitempty"`
	Sources     []Source `json:"sources,omitempty"`
}

// Source representa uma fonte de streaming
type Source struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Target  string `json:"target"`
	Quality string `json:"quality,omitempty"`
}

// StreamInfo representa informações do stream
type StreamInfo struct {
	URL     string            `json:"url"`
	Referer string            `json:"referer,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// SearchResponse representa a resposta de busca
type SearchResponse struct {
	Data []Anime `json:"data"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

// EpisodeResponse representa a resposta de episódio
type EpisodeResponse struct {
	ID      string   `json:"id"`
	Number  int      `json:"number"`
	Sources []Source `json:"sources"`
}

// StreamResponse representa a resposta de stream
type StreamResponse struct {
	URL     string `json:"url"`
	Referer string `json:"referer"`
}

// Search busca animes pelo título
func Search(query string) ([]Anime, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query não pode ser vazia")
	}

	// Verifica cache
	cacheKey := "search:" + strings.ToLower(query)
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].([]Anime); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/search/%s", BaseURL, url.PathEscape(query))

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro na busca: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = result.Data
	cacheMutex.Unlock()

	return result.Data, nil
}

// GetAnimeByID busca um anime pelo ID da Enime
func GetAnimeByID(id string) (*Anime, error) {
	// Verifica cache
	cacheKey := "anime:" + id
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].(*Anime); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/anime/%s", BaseURL, id)

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar anime: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var anime Anime
	if err := json.NewDecoder(resp.Body).Decode(&anime); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = &anime
	cacheMutex.Unlock()

	return &anime, nil
}

// GetAnimeByAnilistID busca um anime pelo ID do AniList
func GetAnimeByAnilistID(anilistID int) (*Anime, error) {
	// Verifica cache
	cacheKey := fmt.Sprintf("anilist:%d", anilistID)
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].(*Anime); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/mapping/anilist/%d", BaseURL, anilistID)

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar anime: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("anime não encontrado no Enime")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var anime Anime
	if err := json.NewDecoder(resp.Body).Decode(&anime); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = &anime
	cacheMutex.Unlock()

	return &anime, nil
}

// GetEpisode busca informações de um episódio específico
func GetEpisode(animeID string, episodeNumber int) (*Episode, error) {
	// Verifica cache
	cacheKey := fmt.Sprintf("episode:%s:%d", animeID, episodeNumber)
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].(*Episode); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/anime/%s/%d", BaseURL, animeID, episodeNumber)

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar episódio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("episódio não encontrado")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var episode Episode
	if err := json.NewDecoder(resp.Body).Decode(&episode); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = &episode
	cacheMutex.Unlock()

	return &episode, nil
}

// GetStreamURL obtém a URL do stream de um episódio
func GetStreamURL(episodeID string) (*StreamInfo, error) {
	// Verifica cache
	cacheKey := "stream:" + episodeID
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].(*StreamInfo); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/source/%s", BaseURL, episodeID)

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var streamResp StreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&streamResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	streamInfo := &StreamInfo{
		URL:     streamResp.URL,
		Referer: streamResp.Referer,
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = streamInfo
	cacheMutex.Unlock()

	return streamInfo, nil
}

// GetStreamURLWithContext obtém a URL do stream com context para cancelamento
func GetStreamURLWithContext(ctx context.Context, episodeID string) (*StreamInfo, error) {
	// Verifica cache primeiro (rápido)
	cacheKey := "stream:" + episodeID
	cacheMutex.RLock()
	if cached, ok := cache[cacheKey].(*StreamInfo); ok {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	endpoint := fmt.Sprintf("%s/source/%s", BaseURL, episodeID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err() // Retorna erro de cancelamento
		}
		return nil, fmt.Errorf("erro ao buscar stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP: %d", resp.StatusCode)
	}

	var streamResp StreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&streamResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	streamInfo := &StreamInfo{
		URL:     streamResp.URL,
		Referer: streamResp.Referer,
	}

	// Salva no cache
	cacheMutex.Lock()
	cache[cacheKey] = streamInfo
	cacheMutex.Unlock()

	return streamInfo, nil
}

// FindAndGetStream busca um anime e retorna o stream do episódio (função de conveniência)
func FindAndGetStream(animeTitle string, episodeNumber int) (string, error) {
	fmt.Printf("[Enime] Buscando: %s Ep.%d\n", animeTitle, episodeNumber)

	// Busca o anime
	animes, err := Search(animeTitle)
	if err != nil {
		return "", fmt.Errorf("erro na busca: %w", err)
	}

	if len(animes) == 0 {
		return "", fmt.Errorf("anime não encontrado")
	}

	// Usa o primeiro resultado
	anime := animes[0]
	fmt.Printf("[Enime] Encontrado: %s (ID: %s)\n", anime.Title.Romaji, anime.ID)

	// Busca o episódio
	episode, err := GetEpisode(anime.ID, episodeNumber)
	if err != nil {
		return "", fmt.Errorf("erro ao buscar episódio: %w", err)
	}

	if len(episode.Sources) == 0 {
		return "", fmt.Errorf("nenhuma fonte encontrada para o episódio")
	}

	// Obtém o stream
	streamInfo, err := GetStreamURL(episode.Sources[0].ID)
	if err != nil {
		return "", fmt.Errorf("erro ao obter stream: %w", err)
	}

	fmt.Printf("[Enime] Stream URL: %s\n", streamInfo.URL)
	return streamInfo.URL, nil
}

// FindAndGetStreamWithContext busca com suporte a cancelamento
func FindAndGetStreamWithContext(ctx context.Context, animeTitle string, episodeNumber int) (string, error) {
	fmt.Printf("[Enime] Buscando (com context): %s Ep.%d\n", animeTitle, episodeNumber)

	// Busca o anime
	animes, err := Search(animeTitle)
	if err != nil {
		return "", fmt.Errorf("erro na busca: %w", err)
	}

	if len(animes) == 0 {
		return "", fmt.Errorf("anime não encontrado")
	}

	// Verifica cancelamento
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Usa o primeiro resultado
	anime := animes[0]

	// Busca o episódio
	episode, err := GetEpisode(anime.ID, episodeNumber)
	if err != nil {
		return "", fmt.Errorf("erro ao buscar episódio: %w", err)
	}

	// Verifica cancelamento
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if len(episode.Sources) == 0 {
		return "", fmt.Errorf("nenhuma fonte encontrada para o episódio")
	}

	// Obtém o stream com context
	streamInfo, err := GetStreamURLWithContext(ctx, episode.Sources[0].ID)
	if err != nil {
		return "", err
	}

	return streamInfo.URL, nil
}

// ClearCache limpa o cache
func ClearCache() {
	cacheMutex.Lock()
	cache = make(map[string]interface{})
	cacheMutex.Unlock()
}
