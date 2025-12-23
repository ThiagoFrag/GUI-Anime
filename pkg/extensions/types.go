// Package extensions fornece um sistema de plugins inspirado no Mihon/Aniyomi
// permitindo que fontes de anime sejam adicionadas/atualizadas sem recompilar o app
package extensions

import (
	"context"
	"time"
)

// ExtensionInfo contém metadados de uma extension
type ExtensionInfo struct {
	ID            string   `json:"id"`            // com.goanime.animefox
	Name          string   `json:"name"`          // AnimeFox
	Version       string   `json:"version"`       // 1.0.0
	MinAppVersion string   `json:"minAppVersion"` // 2.0.0
	Language      string   `json:"language"`      // pt-BR, en, multi
	BaseURL       string   `json:"baseUrl"`       // https://animefox.tv
	IconURL       string   `json:"iconUrl"`       // URL do ícone
	Author        string   `json:"author"`        // GoAnime Community
	NSFW          bool     `json:"nsfw"`          // Conteúdo adulto
	HasLatest     bool     `json:"hasLatest"`     // Suporta listagem de lançamentos
	HasPopular    bool     `json:"hasPopular"`    // Suporta listagem de populares
	HasSearch     bool     `json:"hasSearch"`     // Suporta busca
	Filters       []Filter `json:"filters"`       // Filtros disponíveis (gênero, ano, etc)
}

// Filter representa um filtro de busca disponível
type Filter struct {
	Name    string         `json:"name"`    // "Gênero"
	Type    string         `json:"type"`    // "select", "checkbox", "text"
	Key     string         `json:"key"`     // "genre"
	Options []FilterOption `json:"options"` // Para select/checkbox
}

// FilterOption é uma opção dentro de um filtro
type FilterOption struct {
	Label string `json:"label"` // "Ação"
	Value string `json:"value"` // "action"
}

// AnimeEntry representa um anime na listagem/busca
type AnimeEntry struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Image       string `json:"image"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"` // "ongoing", "completed"
}

// AnimeDetails contém informações detalhadas de um anime
type AnimeDetails struct {
	Title          string   `json:"title"`
	AlternateTitle string   `json:"alternateTitle,omitempty"`
	URL            string   `json:"url"`
	Image          string   `json:"image"`
	Banner         string   `json:"banner,omitempty"`
	Description    string   `json:"description"`
	Status         string   `json:"status"` // "ongoing", "completed", "hiatus"
	Genres         []string `json:"genres"`
	Year           int      `json:"year,omitempty"`
	Studio         string   `json:"studio,omitempty"`
	Rating         float64  `json:"rating,omitempty"` // 0-10
	TotalEpisodes  int      `json:"totalEpisodes,omitempty"`
}

// Episode representa um episódio
type Episode struct {
	Number    int       `json:"number"`
	Title     string    `json:"title,omitempty"` // Título do episódio se disponível
	URL       string    `json:"url"`
	Thumbnail string    `json:"thumbnail,omitempty"`
	Date      time.Time `json:"date,omitempty"` // Data de lançamento
	Filler    bool      `json:"filler"`         // Episódio filler
}

// VideoSource representa uma fonte de vídeo extraída
type VideoSource struct {
	URL       string            `json:"url"`
	Quality   string            `json:"quality"` // "1080p", "720p", "480p", "auto"
	Format    string            `json:"format"`  // "hls", "dash", "mp4"
	Server    string            `json:"server"`  // Nome do servidor (Vidstreaming, Mp4Upload)
	Headers   map[string]string `json:"headers,omitempty"`
	Subtitles []Subtitle        `json:"subtitles,omitempty"`
}

// Subtitle representa uma legenda disponível
type Subtitle struct {
	URL      string `json:"url"`
	Language string `json:"language"` // "pt-BR", "en"
	Label    string `json:"label"`    // "Português (Brasil)"
	Format   string `json:"format"`   // "vtt", "srt", "ass"
	Default  bool   `json:"default"`  // Legenda padrão
}

// ExtensionSource é a interface principal que toda extension deve implementar
type ExtensionSource interface {
	// GetInfo retorna os metadados da extension
	GetInfo() ExtensionInfo

	// Search busca animes por query
	// filters é um mapa de filtros aplicados (chave -> valor)
	Search(ctx context.Context, query string, page int, filters map[string]string) ([]AnimeEntry, bool, error)
	// retorna: resultados, hasNextPage, error

	// GetLatest retorna os últimos lançamentos
	GetLatest(ctx context.Context, page int) ([]AnimeEntry, bool, error)

	// GetPopular retorna os animes mais populares
	GetPopular(ctx context.Context, page int) ([]AnimeEntry, bool, error)

	// GetAnimeDetails retorna detalhes completos de um anime
	GetAnimeDetails(ctx context.Context, url string) (*AnimeDetails, error)

	// GetEpisodes retorna a lista de episódios de um anime
	GetEpisodes(ctx context.Context, animeURL string) ([]Episode, error)

	// GetVideoSources extrai as fontes de vídeo de um episódio
	GetVideoSources(ctx context.Context, episodeURL string) ([]VideoSource, error)
}

// ExtensionState representa o estado de uma extension instalada
type ExtensionState int

const (
	ExtensionStateEnabled ExtensionState = iota
	ExtensionStateDisabled
	ExtensionStateOutdated
	ExtensionStateError
)

// InstalledExtension representa uma extension instalada no sistema
type InstalledExtension struct {
	Info       ExtensionInfo   `json:"info"`
	State      ExtensionState  `json:"state"`
	Source     ExtensionSource `json:"-"` // Runtime, não serializado
	ScriptPath string          `json:"scriptPath"`
	Error      string          `json:"error,omitempty"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// Repository representa um repositório remoto de extensions
type Repository struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"` // URL do index.json
	Official    bool      `json:"official"`
	LastChecked time.Time `json:"lastChecked"`
}

// RepositoryIndex é o índice de extensions de um repositório
type RepositoryIndex struct {
	Version     int               `json:"version"`
	LastUpdated time.Time         `json:"lastUpdated"`
	Extensions  []RemoteExtension `json:"extensions"`
}

// RemoteExtension representa uma extension disponível em um repositório
type RemoteExtension struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Language  string `json:"language"`
	NSFW      bool   `json:"nsfw"`
	IconURL   string `json:"iconUrl"`
	ScriptURL string `json:"scriptUrl"` // URL do script Lua/JS
	Changelog string `json:"changelog,omitempty"`
}

// SearchFilters é um helper para construir filtros de busca
type SearchFilters map[string]string

func NewSearchFilters() SearchFilters {
	return make(SearchFilters)
}

func (f SearchFilters) WithGenre(genre string) SearchFilters {
	f["genre"] = genre
	return f
}

func (f SearchFilters) WithYear(year int) SearchFilters {
	f["year"] = string(rune(year))
	return f
}

func (f SearchFilters) WithStatus(status string) SearchFilters {
	f["status"] = status
	return f
}
