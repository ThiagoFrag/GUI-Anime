package gofilecloud

import (
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
"time"
)

const (
// BaseURL da API do Anime Downloader
BaseURL = "http://localhost:8888/api/v1"

// SourceName identificador da fonte
SourceName = "GoFileCloud"

// Language idioma do conteudo
Language = "pt-BR"
)

// Client cliente da API GoFileCloud
type Client struct {
httpClient *http.Client
baseURL    string
}

// Anime representa um anime da biblioteca
type Anime struct {
ID          int64  `json:"id"`
Name        string `json:"name"`
Slug        string `json:"slug"`
Cover       string `json:"cover"`
Category    string `json:"category"`     // dubbed-ptbr, sub-ptbr, dual-audio, original
AudioLang   string `json:"audio_lang"`
SubLang     string `json:"sub_lang"`
TotalEps    int    `json:"total_episodes"`
Description string `json:"description,omitempty"`
}

// Episode representa um episodio
type Episode struct {
ID           int64  `json:"id"`
AnimeID      int64  `json:"anime_id"`
AnimeName    string `json:"anime_name,omitempty"`
AnimeSlug    string `json:"anime_slug,omitempty"`
AnimeCover   string `json:"anime_cover,omitempty"`
Number       int    `json:"episode_number"`
Season       int    `json:"season"`
Quality      string `json:"quality"`
AudioLang    string `json:"audio_lang"`
SubLang      string `json:"sub_lang"`
FileSize     int64  `json:"file_size"`
GoFileID     string `json:"gofile_id"`
GoFileURL    string `json:"gofile_url"`
DirectURL    string `json:"direct_url,omitempty"`
ExpiresAt    string `json:"expires_at,omitempty"`
IsUploaded   bool   `json:"is_uploaded"`
}

// StatsResponse estatisticas da biblioteca
type StatsResponse struct {
Success bool  `json:"success"`
Stats   Stats `json:"stats"`
}

type Stats struct {
TotalAnimes    int     `json:"total_animes"`
TotalEpisodes  int     `json:"total_episodes"`
TotalAccounts  int     `json:"total_accounts"`
TotalSizeBytes int64   `json:"total_size_bytes"`
TotalSizeGB    float64 `json:"total_size_gb"`
Uploaded       int     `json:"uploaded"`
DubbedPTBR     int     `json:"dubbed_ptbr"`
SubPTBR        int     `json:"sub_ptbr"`
DualAudio      int     `json:"dual_audio"`
Original       int     `json:"original"`
Quality1080p   int     `json:"quality_1080p"`
Quality4K      int     `json:"quality_4k"`
}

// AnimesResponse resposta de listagem de animes
type AnimesResponse struct {
Success bool    `json:"success"`
Animes  []Anime `json:"animes"`
Total   int     `json:"total"`
Page    int     `json:"page"`
Limit   int     `json:"limit"`
}

// EpisodesResponse resposta de listagem de episodios
type EpisodesResponse struct {
Success  bool      `json:"success"`
Episodes []Episode `json:"episodes"`
Total    int       `json:"total"`
}

// SearchResponse resposta de busca
type SearchResponse struct {
Success bool      `json:"success"`
Results []Episode `json:"results"`
Total   int       `json:"total"`
}

// StreamResponse resposta de streaming
type StreamResponse struct {
Success   bool              `json:"success"`
StreamURL string            `json:"stream_url"`
Episode   Episode           `json:"episode"`
Headers   map[string]string `json:"headers,omitempty"`
ExpiresAt string            `json:"expires_at,omitempty"`
}

// CategoriesResponse categorias disponiveis
type CategoriesResponse struct {
Success    bool       `json:"success"`
Categories []Category `json:"categories"`
}

type Category struct {
ID    string `json:"id"`
Name  string `json:"name"`
Count int    `json:"count"`
}

// NewClient cria um novo cliente GoFileCloud
func NewClient() *Client {
return &Client{
httpClient: &http.Client{
Timeout: 30 * time.Second,
},
baseURL: BaseURL,
}
}

// NewClientWithURL cria cliente com URL customizada
func NewClientWithURL(baseURL string) *Client {
return &Client{
httpClient: &http.Client{
Timeout: 30 * time.Second,
},
baseURL: baseURL,
}
}

// IsAvailable verifica se a API esta disponivel
func (c *Client) IsAvailable() bool {
resp, err := c.httpClient.Get(c.baseURL + "/stats")
if err != nil {
return false
}
defer resp.Body.Close()
return resp.StatusCode == http.StatusOK
}

// GetStats obtem estatisticas da biblioteca
func (c *Client) GetStats() (*Stats, error) {
resp, err := c.httpClient.Get(c.baseURL + "/stats")
if err != nil {
return nil, fmt.Errorf("erro ao obter stats: %w", err)
}
defer resp.Body.Close()

var result StatsResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, fmt.Errorf("erro ao decodificar stats: %w", err)
}

if !result.Success {
return nil, fmt.Errorf("API retornou erro")
}

return &result.Stats, nil
}

// GetAnimes lista todos os animes
func (c *Client) GetAnimes(category, quality string, page, limit int) ([]Anime, int, error) {
reqURL := fmt.Sprintf("%s/animes?page=%d&limit=%d", c.baseURL, page, limit)
if category != "" {
reqURL += "&category=" + url.QueryEscape(category)
}
if quality != "" {
reqURL += "&quality=" + url.QueryEscape(quality)
}

resp, err := c.httpClient.Get(reqURL)
if err != nil {
return nil, 0, fmt.Errorf("erro ao listar animes: %w", err)
}
defer resp.Body.Close()

var result AnimesResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, 0, fmt.Errorf("erro ao decodificar animes: %w", err)
}

if !result.Success {
return nil, 0, fmt.Errorf("API retornou erro")
}

return result.Animes, result.Total, nil
}

// GetAnime obtem detalhes de um anime pelo slug
func (c *Client) GetAnime(slug string) (*Anime, error) {
reqURL := fmt.Sprintf("%s/anime/%s", c.baseURL, url.PathEscape(slug))

resp, err := c.httpClient.Get(reqURL)
if err != nil {
return nil, fmt.Errorf("erro ao obter anime: %w", err)
}
defer resp.Body.Close()

var result struct {
Success bool  `json:"success"`
Anime   Anime `json:"anime"`
}
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, fmt.Errorf("erro ao decodificar anime: %w", err)
}

if !result.Success {
return nil, fmt.Errorf("anime nao encontrado")
}

return &result.Anime, nil
}

// GetEpisodes lista episodios de um anime
func (c *Client) GetEpisodes(animeID int64) ([]Episode, error) {
reqURL := fmt.Sprintf("%s/episodes/%d", c.baseURL, animeID)

resp, err := c.httpClient.Get(reqURL)
if err != nil {
return nil, fmt.Errorf("erro ao listar episodios: %w", err)
}
defer resp.Body.Close()

var result EpisodesResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, fmt.Errorf("erro ao decodificar episodios: %w", err)
}

if !result.Success {
return nil, fmt.Errorf("API retornou erro")
}

return result.Episodes, nil
}

// Search busca animes/episodios
func (c *Client) Search(query string, filters ...string) ([]Episode, error) {
reqURL := fmt.Sprintf("%s/search?q=%s", c.baseURL, url.QueryEscape(query))

// Adiciona filtros opcionais
for _, filter := range filters {
reqURL += "&filter=" + url.QueryEscape(filter)
}

resp, err := c.httpClient.Get(reqURL)
if err != nil {
return nil, fmt.Errorf("erro ao buscar: %w", err)
}
defer resp.Body.Close()

var result SearchResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, fmt.Errorf("erro ao decodificar busca: %w", err)
}

if !result.Success {
return nil, fmt.Errorf("busca retornou erro")
}

return result.Results, nil
}

// GetStreamURL obtem URL de streaming de um episodio
func (c *Client) GetStreamURL(episodeID int64) (*StreamResponse, error) {
reqURL := fmt.Sprintf("%s/stream/%d", c.baseURL, episodeID)

resp, err := c.httpClient.Get(reqURL)
if err != nil {
return nil, fmt.Errorf("erro ao obter stream: %w", err)
}
defer resp.Body.Close()

body, _ := io.ReadAll(resp.Body)

var result StreamResponse
if err := json.Unmarshal(body, &result); err != nil {
return nil, fmt.Errorf("erro ao decodificar stream: %w", err)
}

if !result.Success {
return nil, fmt.Errorf("stream nao disponivel")
}

return &result, nil
}

// GetCategories lista categorias disponiveis
func (c *Client) GetCategories() ([]Category, error) {
resp, err := c.httpClient.Get(c.baseURL + "/categories")
if err != nil {
return nil, fmt.Errorf("erro ao listar categorias: %w", err)
}
defer resp.Body.Close()

var result CategoriesResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, fmt.Errorf("erro ao decodificar categorias: %w", err)
}

if !result.Success {
return nil, fmt.Errorf("API retornou erro")
}

return result.Categories, nil
}

// RefreshEpisode força renovacao de um episodio
func (c *Client) RefreshEpisode(episodeID int64) error {
reqURL := fmt.Sprintf("%s/refresh/%d", c.baseURL, episodeID)

req, err := http.NewRequest("POST", reqURL, nil)
if err != nil {
return fmt.Errorf("erro ao criar request: %w", err)
}

resp, err := c.httpClient.Do(req)
if err != nil {
return fmt.Errorf("erro ao renovar: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return fmt.Errorf("erro ao renovar: status %d", resp.StatusCode)
}

return nil
}

// GetSourceName retorna nome da fonte
func (c *Client) GetSourceName() string {
return SourceName
}

// GetLanguage retorna idioma
func (c *Client) GetLanguage() string {
return Language
}

// GetPlayerURL retorna URL do player HTML5 embutido
func (c *Client) GetPlayerURL(episodeID int64) string {
return fmt.Sprintf("%s/player/%d", c.baseURL, episodeID)
}
