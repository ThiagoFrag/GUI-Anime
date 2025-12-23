// Package api - animesflix.go cliente para API AnimeFlix
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"GoAnimeGUI/pkg/store"
)

// AnimesFlixClient para buscar na API AnimeFlix local
type AnimesFlixClient struct {
	baseURL string
	client  *http.Client
}

// AnimesFlixAnime representa um anime da API AnimeFlix
type AnimesFlixAnime struct {
	ID     string   `json:"id"`
	Slug   string   `json:"slug"`
	Title  string   `json:"title"`
	Cover  string   `json:"cover"`
	Rating string   `json:"rating"`
	Year   string   `json:"year"`
	Genres []string `json:"genres"`
	URL    string   `json:"url"`
}

// AnimesFlixEpisode representa um episódio da API AnimeFlix
type AnimesFlixEpisode struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// AnimesFlixSeason representa uma temporada
type AnimesFlixSeason struct {
	Number   int                 `json:"number"`
	Episodes []AnimesFlixEpisode `json:"episodes"`
}

// AnimesFlixAnimeDetails detalhes completos do anime
type AnimesFlixAnimeDetails struct {
	AnimesFlixAnime
	Synopsis string             `json:"synopsis"`
	Seasons  []AnimesFlixSeason `json:"seasons"`
}

// AnimesFlixStream representa um stream de vídeo
type AnimesFlixStream struct {
	PlayURL  string `json:"play_url"`
	Quality  string `json:"quality"`
	FormatID int    `json:"format_id"`
}

// AnimesFlixResolveResponse resposta do endpoint /resolve
type AnimesFlixResolveResponse struct {
	Success   bool               `json:"success"`
	Thumbnail string             `json:"thumbnail"`
	Streams   []AnimesFlixStream `json:"streams"`
	Error     string             `json:"error,omitempty"`
}

// AnimesFlixListResponse resposta do endpoint /api/animes
type AnimesFlixListResponse struct {
	Success     bool              `json:"success"`
	Page        int               `json:"page"`
	HasNext     bool              `json:"has_next"`
	HasPrevious bool              `json:"has_previous"`
	Data        []AnimesFlixAnime `json:"data"`
}

// AnimesFlixDetailsResponse resposta do endpoint /api/anime
type AnimesFlixDetailsResponse struct {
	Success bool                    `json:"success"`
	Anime   *AnimesFlixAnimeDetails `json:"anime"`
	Error   string                  `json:"error,omitempty"`
}

// AnimesFlixSearchResponse resposta do endpoint /api/search
type AnimesFlixSearchResponse struct {
	Success bool              `json:"success"`
	Cached  bool              `json:"cached"`
	Total   int               `json:"total"`
	Data    []AnimesFlixAnime `json:"data"`
	Error   string            `json:"error,omitempty"`
}

// NewAnimesFlixClient cria um novo cliente AnimeFlix
func NewAnimesFlixClient() *AnimesFlixClient {
	return &AnimesFlixClient{
		baseURL: "http://localhost:8082",
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// IsAvailable verifica se a API AnimeFlix está disponível
func (c *AnimesFlixClient) IsAvailable() bool {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// Search busca animes no AnimeFlix
func (c *AnimesFlixClient) Search(query string) ([]AnimesFlixAnime, error) {
	encoded := url.QueryEscape(query)
	reqURL := fmt.Sprintf("%s/api/search?q=%s", c.baseURL, encoded)

	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	var result AnimesFlixSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("API error: %s", result.Error)
	}

	return result.Data, nil
}

// GetAnimeList lista animes por página
func (c *AnimesFlixClient) GetAnimeList(page int) ([]AnimesFlixAnime, bool, error) {
	reqURL := fmt.Sprintf("%s/api/animes?page=%d", c.baseURL, page)

	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, false, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	var result AnimesFlixListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if !result.Success {
		return nil, false, fmt.Errorf("API error")
	}

	return result.Data, result.HasNext, nil
}

// GetAnimeDetails obtém detalhes de um anime
func (c *AnimesFlixClient) GetAnimeDetails(animeID string) (*AnimesFlixAnimeDetails, error) {
	encoded := url.QueryEscape(animeID)
	reqURL := fmt.Sprintf("%s/api/anime?id=%s", c.baseURL, encoded)

	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	var result AnimesFlixDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if !result.Success || result.Anime == nil {
		return nil, fmt.Errorf("API error: %s", result.Error)
	}

	return result.Anime, nil
}

// ResolveEpisode resolve o stream de um episódio
func (c *AnimesFlixClient) ResolveEpisode(episodeURL string) (*AnimesFlixResolveResponse, error) {
	encoded := url.QueryEscape(episodeURL)
	reqURL := fmt.Sprintf("%s/resolve?url=%s", c.baseURL, encoded)

	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	var result AnimesFlixResolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// GetBestStreamURL retorna a melhor URL de stream (maior qualidade)
func (c *AnimesFlixClient) GetBestStreamURL(episodeURL string) (string, error) {
	resolved, err := c.ResolveEpisode(episodeURL)
	if err != nil {
		return "", err
	}

	if !resolved.Success || len(resolved.Streams) == 0 {
		return "", fmt.Errorf("nenhum stream encontrado: %s", resolved.Error)
	}

	// Retorna a primeira stream (geralmente a de maior qualidade)
	return resolved.Streams[0].PlayURL, nil
}

// GetAnimesByCategory lista animes por categoria
func (c *AnimesFlixClient) GetAnimesByCategory(category string, page int) ([]AnimesFlixAnime, bool, error) {
	encoded := url.QueryEscape(category)
	reqURL := fmt.Sprintf("%s/api/category/%s?page=%d", c.baseURL, encoded, page)

	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, false, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool              `json:"success"`
		HasNext bool              `json:"has_next"`
		Data    []AnimesFlixAnime `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return result.Data, result.HasNext, nil
}

// ToSavedAnime converte AnimesFlixAnime para store.SavedAnime
func (a *AnimesFlixAnime) ToSavedAnime() store.SavedAnime {
	return store.SavedAnime{
		Title: a.Title,
		Image: a.Cover,
		URL:   a.URL,
		Sources: []store.AnimeSource{{
			Name:     "AnimeFlix",
			Language: "pt-BR",
			URL:      a.URL,
		}},
	}
}

// ToEpisodes converte temporadas para []store.Episode
func (d *AnimesFlixAnimeDetails) ToEpisodes() []store.Episode {
	var episodes []store.Episode

	for _, season := range d.Seasons {
		for _, ep := range season.Episodes {
			num := 0
			fmt.Sscanf(ep.Number, "%d", &num)

			episodes = append(episodes, store.Episode{
				Title:  ep.Title,
				URL:    ep.URL,
				Season: season.Number,
				Number: num,
				Source: "AnimeFlix",
			})
		}
	}

	return episodes
}

// GoFileStreamResponse resposta do endpoint /api/v2/stream/resolve
type GoFileStreamResponse struct {
	StreamURL   string            `json:"stream_url"`
	Source      string            `json:"source"`
	GoFileID    string            `json:"gofile_id,omitempty"`
	GoFileToken string            `json:"gofile_token,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ContentID   string            `json:"content_id,omitempty"`
	EpisodeKey  string            `json:"episode_key,omitempty"`
	ExpiresAt   string            `json:"expires_at,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// ResolveStreamGoFile resolve a URL de streaming via GoFile
func (c *AnimesFlixClient) ResolveStreamGoFile(episodeURL string) (*GoFileStreamResponse, error) {
	reqURL := fmt.Sprintf("%s/api/v2/stream/resolve?url=%s", c.baseURL, url.QueryEscape(episodeURL))

	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("erro na requisicao: %v", err)
	}
	defer resp.Body.Close()

	var result GoFileStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	if result.Error != "" {
		return &result, fmt.Errorf("erro da API: %s", result.Error)
	}

	return &result, nil
}

// GetStreamURLWithGoFile tenta obter URL de streaming preferindo GoFile
func (c *AnimesFlixClient) GetStreamURLWithGoFile(episodeURL string) (string, map[string]string, error) {
	gofileResp, err := c.ResolveStreamGoFile(episodeURL)
	if err == nil && gofileResp.StreamURL != "" && gofileResp.Source == "gofile" {
		headers := gofileResp.Headers
		if headers == nil {
			headers = make(map[string]string)
		}
		if gofileResp.GoFileToken != "" && headers["Cookie"] == "" {
			headers["Cookie"] = "accountToken=" + gofileResp.GoFileToken
		}
		return gofileResp.StreamURL, headers, nil
	}

	if gofileResp != nil && gofileResp.StreamURL != "" {
		return gofileResp.StreamURL, gofileResp.Headers, nil
	}

	streamURL, err := c.GetBestStreamURL(episodeURL)
	if err != nil {
		return "", nil, err
	}

	return streamURL, nil, nil
}
