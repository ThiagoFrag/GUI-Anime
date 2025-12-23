package animesflix

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// CacheCheckResponse resposta do endpoint /api/v2/cache/check
type CacheCheckResponse struct {
	Success     bool              `json:"success"`
	Cached      bool              `json:"cached"`
	EpisodeID   string            `json:"episode_id"`
	AnimeID     string            `json:"anime_id"`
	AnimeTitle  string            `json:"anime_title"`
	Season      int               `json:"season"`
	Episode     int               `json:"episode"`
	Quality     string            `json:"quality"`
	Status      string            `json:"status"`
	GoFileCode  string            `json:"gofile_code"`
	GoFileLink  string            `json:"gofile_link"`
	GoFileToken string            `json:"gofile_token"`
	Headers     map[string]string `json:"headers"`
}

// CachedEpisodeInfo informacoes de um episodio cacheado
type CachedEpisodeInfo struct {
	EpisodeID  string `json:"episode_id"`
	Season     int    `json:"season"`
	Episode    int    `json:"episode"`
	Quality    string `json:"quality"`
	GoFileCode string `json:"gofile_code"`
	Status     string `json:"status"`
}

// CachedAnimeInfo informacoes de um anime no cache
type CachedAnimeInfo struct {
	AnimeID    string              `json:"anime_id"`
	AnimeTitle string              `json:"anime_title"`
	Episodes   []CachedEpisodeInfo `json:"episodes"`
	// Campos derivados para compatibilidade
	URL   string `json:"-"` // URL calculada
	Cover string `json:"-"` // Capa (buscar depois se necessario)
}

// GetURL retorna a URL do anime no AnimeFlix
func (c *CachedAnimeInfo) GetURL() string {
	if c.URL != "" {
		return c.URL
	}
	// Gera URL padrao do AnimeFlix
	return fmt.Sprintf("https://animesflix.net/assistir/%s", c.AnimeID)
}

// CacheSearchResponse resposta do endpoint /api/v2/cache/search
type CacheSearchResponse struct {
	Success bool              `json:"success"`
	Query   string            `json:"query"`
	Count   int               `json:"count"`
	Animes  int               `json:"animes"`
	Results []CachedAnimeInfo `json:"results"`
}

// CheckEpisodeCache verifica se um episodio esta no cache do GoFile
func (c *Client) CheckEpisodeCache(animeID string, season, episode int) (*CacheCheckResponse, error) {
	reqURL := fmt.Sprintf("%s/api/v2/cache/check?anime_id=%s&season=%d&episode=%d",
		c.baseURL, animeID, season, episode)

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar cache: %w", err)
	}
	defer resp.Body.Close()

	var result CacheCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	return &result, nil
}

// SearchCache busca episodios cacheados por nome do anime
func (c *Client) SearchCache(query string) (*CacheSearchResponse, error) {
	reqURL := fmt.Sprintf("%s/api/v2/cache/search?q=%s", c.baseURL, url.QueryEscape(query))

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cache: %w", err)
	}
	defer resp.Body.Close()

	var result CacheSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	return &result, nil
}

// GetCachedStreamURL obtem URL de streaming do GoFile para um episodio cacheado
func (c *Client) GetCachedStreamURL(animeID string, season, episode int) (string, map[string]string, error) {
	reqURL := fmt.Sprintf("%s/api/v2/stream/resolve?anime_id=%s&season=%d&ep=%d",
		c.baseURL, animeID, season, episode)

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return "", nil, fmt.Errorf("erro ao resolver stream: %w", err)
	}
	defer resp.Body.Close()

	var result GoFileStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	if result.StreamURL == "" {
		return "", nil, fmt.Errorf("stream nao encontrado")
	}

	headers := result.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	if result.GoFileToken != "" && headers["Cookie"] == "" {
		headers["Cookie"] = "accountToken=" + result.GoFileToken
	}

	return result.StreamURL, headers, nil
}
