package animesflix

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// BaseURL e a URL base da API AnimeFlix local
	BaseURL = "http://localhost:8082"

	// SourceName e o nome da fonte
	SourceName = "AnimeFlix"

	// Language e o idioma dos conteudos
	Language = "pt-BR"
)

// Client e o cliente da API AnimeFlix
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// Anime representa um anime da API AnimeFlix
type Anime struct {
	ID         string `json:"id"`
	Nome       string `json:"title"` // API retorna como "title"
	Capa       string `json:"cover"` // API retorna como "cover"
	URL        string `json:"url"`
	Slug       string `json:"slug"`
	Categoria  string `json:"categoria"`
	Descricao  string `json:"descricao,omitempty"`
	Temporadas int    `json:"temporadas,omitempty"`
}

// Episode representa um episodio
type Episode struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// Season representa uma temporada
type Season struct {
	Number   int       `json:"number"`
	Episodes []Episode `json:"episodes"`
}

// Temporada representa uma temporada (alias para compatibilidade)
type Temporada struct {
	Numero    int       `json:"numero"`
	Episodios []Episode `json:"episodios"`
}

// SearchResponse representa resposta de busca da API
type SearchResponse struct {
	Success bool    `json:"success"`
	Cached  bool    `json:"cached"`
	Data    []Anime `json:"data"`
	Error   string  `json:"error,omitempty"`
}

// SearchResult representa resultado de busca
type SearchResult struct {
	Query   string  `json:"query"`
	Results []Anime `json:"results"`
	Total   int     `json:"total"`
}

// StreamInfo representa informacoes de stream
type StreamInfo struct {
	EpisodeID string   `json:"episode_id"`
	Title     string   `json:"title"`
	StreamURL string   `json:"stream_url"`
	M3U8URL   string   `json:"m3u8_url,omitempty"`
	MP4URL    string   `json:"mp4_url,omitempty"`
	Sources   []Source `json:"sources,omitempty"`
}

// Source representa uma fonte de video
type Source struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
	Type    string `json:"type"`
}

// AnimeDetails representa detalhes do anime com episodios
type AnimeDetails struct {
	ID       string   `json:"id"`
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Cover    string   `json:"cover"`
	URL      string   `json:"url"`
	Synopsis string   `json:"synopsis"`
	Genres   []string `json:"genres"`
	Seasons  []Season `json:"seasons"`
}

// AnimeDetailsResponse representa a resposta da API
type AnimeDetailsResponse struct {
	Success bool         `json:"success"`
	Anime   AnimeDetails `json:"anime"`
}

// NewClient cria um novo cliente AnimeFlix
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: BaseURL,
	}
}

// IsAvailable verifica se a API esta disponivel
func (c *Client) IsAvailable() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Search busca animes
func (c *Client) Search(query string) ([]Anime, error) {
	searchURL := fmt.Sprintf("%s/api/search?q=%s", c.baseURL, url.QueryEscape(query))

	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d ao buscar", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Tenta como formato {success, cached, data}
	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err == nil && searchResp.Success {
		return searchResp.Data, nil
	}

	// Tenta primeiro como array direto
	var animes []Anime
	if err := json.Unmarshal(body, &animes); err == nil {
		return animes, nil
	}

	// Tenta como objeto com campo results
	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar: %w", err)
	}

	return result.Results, nil
}

// GetAnimeDetails obtem detalhes de um anime
func (c *Client) GetAnimeDetails(animeURL string) (*AnimeDetails, error) {
	detailsURL := fmt.Sprintf("%s/api/anime?url=%s", c.baseURL, url.QueryEscape(animeURL))

	resp, err := c.httpClient.Get(detailsURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter detalhes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d ao obter detalhes", resp.StatusCode)
	}

	var response AnimeDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar detalhes: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API retornou erro")
	}

	return &response.Anime, nil
}

// GetEpisodes obtem todos os episodios de um anime
func (c *Client) GetEpisodes(animeURL string) ([]Episode, error) {
	details, err := c.GetAnimeDetails(animeURL)
	if err != nil {
		return nil, err
	}

	// Coleta todos os episodios de todas as temporadas
	var allEpisodes []Episode
	for _, season := range details.Seasons {
		allEpisodes = append(allEpisodes, season.Episodes...)
	}

	return allEpisodes, nil
}

// GetStream obtem URL de stream de um episodio (metodo antigo)
func (c *Client) GetStream(episodeURL string) (*StreamInfo, error) {
	streamURL := fmt.Sprintf("%s/api/stream?url=%s", c.baseURL, url.QueryEscape(episodeURL))

	resp, err := c.httpClient.Get(streamURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d ao obter stream", resp.StatusCode)
	}

	var stream StreamInfo
	if err := json.NewDecoder(resp.Body).Decode(&stream); err != nil {
		return nil, fmt.Errorf("erro ao decodificar stream: %w", err)
	}

	return &stream, nil
}

// GetStreamURL obtem a URL de stream real de um episodio usando /resolve
func (c *Client) GetStreamURL(episodeURL string) (string, error) {
	resolveURL := fmt.Sprintf("%s/resolve?url=%s", c.baseURL, url.QueryEscape(episodeURL))

	resp, err := c.httpClient.Get(resolveURL)
	if err != nil {
		return "", fmt.Errorf("erro ao resolver stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d ao resolver stream", resp.StatusCode)
	}

	var response struct {
		Success bool `json:"success"`
		Streams []struct {
			PlayURL  string `json:"play_url"`
			Quality  string `json:"quality"`
			FormatID int    `json:"format_id"`
		} `json:"streams"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if !response.Success || len(response.Streams) == 0 {
		if response.Error != "" {
			return "", fmt.Errorf("API erro: %s", response.Error)
		}
		return "", fmt.Errorf("nenhum stream encontrado")
	}

	// Prefere 720p (format_id 22), senao pega o primeiro disponivel
	for _, stream := range response.Streams {
		if stream.FormatID == 22 || stream.Quality == "720p" {
			return stream.PlayURL, nil
		}
	}

	// Fallback para o primeiro stream
	return response.Streams[0].PlayURL, nil
}

// GetSourceName retorna o nome da fonte
func (c *Client) GetSourceName() string {
	return SourceName
}

// GetLanguage retorna o idioma
func (c *Client) GetLanguage() string {
	return Language
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

// GetStreamURLWithGoFile tenta obter URL de streaming preferindo GoFile cache
func (c *Client) GetStreamURLWithGoFile(episodeURL string) (string, map[string]string, error) {
	// Primeiro tenta resolver via GoFile (cache)
	reqURL := fmt.Sprintf("%s/api/v2/stream/resolve?url=%s", c.baseURL, url.QueryEscape(episodeURL))

	resp, err := c.httpClient.Get(reqURL)
	if err == nil {
		defer resp.Body.Close()

		var gofileResp GoFileStreamResponse
		if json.NewDecoder(resp.Body).Decode(&gofileResp) == nil {
			if gofileResp.StreamURL != "" && gofileResp.Source == "gofile" {
				headers := gofileResp.Headers
				if headers == nil {
					headers = make(map[string]string)
				}
				// Garantir que o token esta no header
				if gofileResp.GoFileToken != "" && headers["Cookie"] == "" {
					headers["Cookie"] = "accountToken=" + gofileResp.GoFileToken
				}
				fmt.Printf("[GetStreamURLWithGoFile] GoFile hit! URL: %s\n", gofileResp.StreamURL)
				return gofileResp.StreamURL, headers, nil
			}

			// Stream direto do Google Video
			if gofileResp.StreamURL != "" {
				fmt.Printf("[GetStreamURLWithGoFile] Stream direto: %s\n", gofileResp.StreamURL)
				return gofileResp.StreamURL, gofileResp.Headers, nil
			}
		}
	}

	// Fallback para metodo antigo
	fmt.Printf("[GetStreamURLWithGoFile] Fallback para GetStreamURL\n")
	streamURL, err := c.GetStreamURL(episodeURL)
	if err != nil {
		return "", nil, err
	}

	return streamURL, nil, nil
}
