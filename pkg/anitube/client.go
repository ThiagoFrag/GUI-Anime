package anitube

import (
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
"time"
)

const (
// BaseURL é a URL base da API Anitube local
BaseURL = "http://localhost:8081"

// SourceName é o nome da fonte
SourceName = "Anitube"

// Language é o idioma dos conteúdos
Language = "pt-BR"
)

// Client é o cliente da API Anitube
type Client struct {
httpClient *http.Client
baseURL    string
}

// Anime representa um anime da API
type Anime struct {
ID          string `json:"id"`
Title       string `json:"title"`
Image       string `json:"image"`
URL         string `json:"url"`
Type        string `json:"type"`
Description string `json:"description,omitempty"`
}

// Episode representa um episódio
type Episode struct {
ID     string `json:"id"`
Number string `json:"number"`
Title  string `json:"title"`
Image  string `json:"image"`
URL    string `json:"url"`
Type   string `json:"type"`
}

// SearchResult representa resultado de busca
type SearchResult struct {
Query   string  `json:"query"`
Results []Anime `json:"results"`
Total   int     `json:"total"`
}

// StreamInfo representa informações de stream
type StreamInfo struct {
EpisodeID string   `json:"episode_id"`
Title     string   `json:"title"`
StreamURL string   `json:"stream_url"`
M3U8URL   string   `json:"m3u8_url,omitempty"`
MP4URL    string   `json:"mp4_url,omitempty"`
Sources   []Source `json:"sources,omitempty"`
}

// Source representa uma fonte de vídeo
type Source struct {
URL     string `json:"url"`
Quality string `json:"quality"`
Type    string `json:"type"`
}

// AnimeDetails representa detalhes do anime com episódios
type AnimeDetails struct {
ID          string    `json:"id"`
Title       string    `json:"title"`
Image       string    `json:"image"`
URL         string    `json:"url"`
Description string    `json:"description"`
Genres      []string  `json:"genres"`
Status      string    `json:"status"`
Year        string    `json:"year"`
Episodes    []Episode `json:"episodes"`
}

// NewClient cria um novo cliente Anitube
func NewClient() *Client {
return &Client{
httpClient: &http.Client{
Timeout: 30 * time.Second,
},
baseURL: BaseURL,
}
}

// NewClientWithURL cria um cliente com URL customizada
func NewClientWithURL(baseURL string) *Client {
return &Client{
httpClient: &http.Client{
Timeout: 30 * time.Second,
},
baseURL: baseURL,
}
}

// IsAvailable verifica se a API está disponível
func (c *Client) IsAvailable() bool {
resp, err := c.httpClient.Get(c.baseURL + "/health")
if err != nil {
return false
}
defer resp.Body.Close()
return resp.StatusCode == http.StatusOK
}

// Search busca animes por termo
func (c *Client) Search(query string) ([]Anime, error) {
endpoint := fmt.Sprintf("%s/api/search?q=%s", c.baseURL, url.QueryEscape(query))

resp, err := c.httpClient.Get(endpoint)
if err != nil {
return nil, fmt.Errorf("erro ao buscar: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var result SearchResult
if err := json.Unmarshal(body, &result); err != nil {
return nil, err
}

return result.Results, nil
}

// GetLatestEpisodes retorna os últimos episódios
func (c *Client) GetLatestEpisodes() ([]Episode, error) {
endpoint := c.baseURL + "/api/latest"

resp, err := c.httpClient.Get(endpoint)
if err != nil {
return nil, err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var episodes []Episode
if err := json.Unmarshal(body, &episodes); err != nil {
return nil, err
}

return episodes, nil
}

// GetPopularAnimes retorna animes populares
func (c *Client) GetPopularAnimes() ([]Anime, error) {
endpoint := c.baseURL + "/api/popular"

resp, err := c.httpClient.Get(endpoint)
if err != nil {
return nil, err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var animes []Anime
if err := json.Unmarshal(body, &animes); err != nil {
return nil, err
}

return animes, nil
}

// GetAnimeDetails retorna detalhes de um anime incluindo episódios
func (c *Client) GetAnimeDetails(id string) (*AnimeDetails, error) {
endpoint := fmt.Sprintf("%s/api/anime/%s", c.baseURL, id)

resp, err := c.httpClient.Get(endpoint)
if err != nil {
return nil, err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var details AnimeDetails
if err := json.Unmarshal(body, &details); err != nil {
return nil, err
}

return &details, nil
}

// GetEpisodeStream retorna URLs de stream para um episódio
func (c *Client) GetEpisodeStream(episodeID string) (*StreamInfo, error) {
endpoint := fmt.Sprintf("%s/api/episode/%s", c.baseURL, episodeID)

resp, err := c.httpClient.Get(endpoint)
if err != nil {
return nil, err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var stream StreamInfo
if err := json.Unmarshal(body, &stream); err != nil {
return nil, err
}

return &stream, nil
}

// GetAnimeList retorna lista de animes (com filtro opcional por letra)
func (c *Client) GetAnimeList(letter string) ([]Anime, error) {
endpoint := c.baseURL + "/api/animes"
if letter != "" && letter != "all" {
endpoint += "?letter=" + url.QueryEscape(letter)
}

resp, err := c.httpClient.Get(endpoint)
if err != nil {
return nil, err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var animes []Anime
if err := json.Unmarshal(body, &animes); err != nil {
return nil, err
}

return animes, nil
}
