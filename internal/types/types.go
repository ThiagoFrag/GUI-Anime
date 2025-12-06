// Package types contém tipos comuns usados em toda a aplicação
package types

import (
	"sync"
	"time"
)

// CacheEntry representa uma entrada no cache com TTL
type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	CreatedAt  time.Time
	Source     string // Qual fonte gerou este cache
	ValidUntil time.Time
}

// IsExpired verifica se a entrada expirou
func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// StreamCacheEntry representa um stream cacheado com metadados
type StreamCacheEntry struct {
	URL           string
	Source        string
	CachedAt      time.Time
	ExpiresAt     time.Time
	LastValidated time.Time
	ValidationErr string
}

// IsExpired verifica se o cache expirou
func (s *StreamCacheEntry) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// NeedsRevalidation verifica se precisa revalidar
func (s *StreamCacheEntry) NeedsRevalidation() bool {
	return time.Since(s.LastValidated) > 5*time.Minute
}

// SourceFailure rastreia falhas de uma fonte
type SourceFailure struct {
	Source      string
	FailedAt    time.Time
	CooldownEnd time.Time
	FailCount   int
	LastError   string
	mu          sync.RWMutex
}

// IsInCooldown verifica se a fonte está em cooldown
func (s *SourceFailure) IsInCooldown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().Before(s.CooldownEnd)
}

// SourceStatus representa o status de uma fonte para estatísticas
type SourceStatus struct {
	Source       string    `json:"source"`
	Failures     int       `json:"failures"`
	LastFailure  time.Time `json:"lastFailure"`
	CooldownEnd  time.Time `json:"cooldownEnd"`
	IsInCooldown bool      `json:"isInCooldown"`
}

// AniListAnime representa um anime com dados do AniList para o frontend
type AniListAnime struct {
	ID           int      `json:"id"`
	MalID        int      `json:"malId"`
	Title        string   `json:"title"`
	TitleEnglish string   `json:"titleEnglish"`
	TitleNative  string   `json:"titleNative"`
	Description  string   `json:"description"`
	Image        string   `json:"image"`  // Cover HD
	Banner       string   `json:"banner"` // Banner para hero section
	Color        string   `json:"color"`  // Cor predominante
	Genres       []string `json:"genres"`
	Episodes     int      `json:"episodes"`
	Duration     int      `json:"duration"`
	Status       string   `json:"status"`
	Season       string   `json:"season"`
	Year         int      `json:"year"`
	Score        int      `json:"score"` // AverageScore (0-100)
	Popularity   int      `json:"popularity"`
	Studio       string   `json:"studio"`
	TrailerURL   string   `json:"trailerUrl"`
	IsAiring     bool     `json:"isAiring"`
	NextEpisode  int      `json:"nextEpisode"`
}

// ConsometAnime representa um anime do Consumet
type ConsumetAnime struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Image         string   `json:"image"`
	TotalEpisodes int      `json:"totalEpisodes"`
	SubOrDub      string   `json:"subOrDub"`
	Genres        []string `json:"genres"`
	Description   string   `json:"description"`
	Provider      string   `json:"provider"`
}

// ConsumetEpisode representa um episódio do Consumet
type ConsumetEpisode struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Provider string `json:"provider"`
}

// SmartStreamResult é o resultado da busca inteligente de stream
type SmartStreamResult struct {
	URL      string  `json:"url"`
	Source   string  `json:"source"`
	Duration float64 `json:"duration"` // em milissegundos
	Success  bool    `json:"success"`
	Error    string  `json:"error,omitempty"`
}

// SkipTimesResult contém os timestamps para pular abertura/encerramento
type SkipTimesResult struct {
	HasOpening    bool    `json:"hasOpening"`
	OpeningStart  float64 `json:"openingStart"`
	OpeningEnd    float64 `json:"openingEnd"`
	HasEnding     bool    `json:"hasEnding"`
	EndingStart   float64 `json:"endingStart"`
	EndingEnd     float64 `json:"endingEnd"`
	HasRecap      bool    `json:"hasRecap"`
	RecapStart    float64 `json:"recapStart"`
	RecapEnd      float64 `json:"recapEnd"`
	EpisodeLength float64 `json:"episodeLength"`
}

// EnimeAnime representa um anime da Enime API
type EnimeAnime struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	TitleNative string   `json:"titleNative"`
	Image       string   `json:"image"`
	Banner      string   `json:"banner"`
	AnilistID   int      `json:"anilistId"`
	MalID       int      `json:"malId"`
	Episodes    int      `json:"episodes"`
	Status      string   `json:"status"`
	Genre       []string `json:"genre"`
}
