package store

import (
	"encoding/json"
	"os"
	// ATENÇÃO: NÃO ADICIONE NENHUM OUTRO IMPORT AQUI
)

// AnimeSource representa uma fonte de anime
type AnimeSource struct {
	Name     string `json:"Name"`     // AllAnime, AnimeFire
	Language string `json:"Language"` // en, pt-BR
	URL      string `json:"URL"`
}

// SavedAnime é a estrutura pública (com letra Maiúscula)
type SavedAnime struct {
	Title   string        `json:"Title"`
	Image   string        `json:"Image"`
	URL     string        `json:"URL"`
	Source  string        `json:"Source,omitempty"`  // Fonte principal (AllAnime, AnimeFire)
	Sources []AnimeSource `json:"Sources,omitempty"` // Múltiplas fontes disponíveis
}

// Episode representa um episódio de uma série
type Episode struct {
	Title  string `json:"Title"`
	URL    string `json:"URL"`
	Season int    `json:"Season"`
	Number int    `json:"Number"`
	Source string `json:"Source"` // AllAnime, AnimeFire, etc
}

// EpisodeStream representa uma opção de stream para um episódio (qualidade -> URL)
type EpisodeStream struct {
	Quality  string            `json:"Quality"`
	URL      string            `json:"URL"`
	Metadata map[string]string `json:"Metadata,omitempty"`
}

// UserSettings representa as configurações do utilizador
type UserSettings struct {
	StartFullscreen bool   `json:"start_fullscreen"` // Iniciar em tela cheia
	ContentLanguage string `json:"content_language"` // "all", "br", "en"
	DefaultQuality  string `json:"default_quality"`  // "auto", "1080p", "720p", etc
	UseAnime4K      bool   `json:"use_anime4k"`      // Usar upscaling Anime4K

	// Seeding / Contribuição
	SeedingEnabled      bool   `json:"seeding_enabled"`       // Ativar semeamento
	SeedingMaxCPU       int    `json:"seeding_max_cpu"`       // Limite de CPU (%)
	SeedingMaxBandwidth int    `json:"seeding_max_bandwidth"` // Limite de banda (MB/s)
	SeedingOnlyWifi     bool   `json:"seeding_only_wifi"`     // Apenas em WiFi
	SeedingSchedule     string `json:"seeding_schedule"`      // "always", "night", "idle"
	SeedingContributed  int64  `json:"seeding_contributed"`   // Total contribuído (bytes)
}

// WatchedEpisode guarda informação de um episódio assistido
type WatchedEpisode struct {
	AnimeTitle   string `json:"anime_title"`
	AnimeImage   string `json:"anime_image"`
	AnimeURL     string `json:"anime_url"`
	EpisodeTitle string `json:"episode_title"`
	EpisodeURL   string `json:"episode_url"`
	EpisodeNum   int    `json:"episode_num"`
	WatchedAt    string `json:"watched_at"` // ISO 8601 timestamp
	Progress     int    `json:"progress"`   // Percentagem assistida (0-100)
}

type UserData struct {
	Username       string           `json:"username"`
	Avatar         string           `json:"avatar"`
	History        []SavedAnime     `json:"history"`
	Favorites      []SavedAnime     `json:"favorites"`
	WatchHistory   []WatchedEpisode `json:"watch_history"`
	Settings       UserSettings     `json:"settings"`
	MPVPath        string           `json:"mpv_path,omitempty"`
	DefaultQuality string           `json:"default_quality,omitempty"`
}

const dbFile = "goanime_user.json"

func LoadUser() *UserData {
	data, err := os.ReadFile(dbFile)
	if err != nil {
		return nil
	}
	var user UserData
	if err := json.Unmarshal(data, &user); err != nil {
		return nil
	}

	// Inicializa settings padrão se não existir
	if user.Settings.ContentLanguage == "" {
		user.Settings.ContentLanguage = "all"
	}

	return &user
}

func SaveUser(user *UserData) error {
	data, _ := json.MarshalIndent(user, "", "  ")
	return os.WriteFile(dbFile, data, 0600)
}

// GetDefaultSettings retorna as configurações padrão
func GetDefaultSettings() UserSettings {
	return UserSettings{
		StartFullscreen:     false,
		ContentLanguage:     "all",
		DefaultQuality:      "auto",
		UseAnime4K:          true,
		SeedingEnabled:      false,
		SeedingMaxCPU:       30,
		SeedingMaxBandwidth: 5,
		SeedingOnlyWifi:     true,
		SeedingSchedule:     "idle",
		SeedingContributed:  0,
	}
}
