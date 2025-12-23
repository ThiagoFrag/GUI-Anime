package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MangaSourceState gerencia o estado das fontes de mangá
type MangaSourceState struct {
	mu             sync.RWMutex
	enabledSources map[string]bool
	configPath     string
}

var mangaSourceState *MangaSourceState

// MangaSourceDetail representa detalhes de uma fonte de mangá
type MangaSourceDetail struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	URL             string `json:"url"`
	Language        string `json:"language"`
	Icon            string `json:"icon"`
	Enabled         bool   `json:"enabled"`
	SupportsLatest  bool   `json:"supportsLatest"`
	SupportsPopular bool   `json:"supportsPopular"`
	SupportsSearch  bool   `json:"supportsSearch"`
}

// MangaSourceRepository representa um repositório de fontes
type MangaSourceRepository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Official bool   `json:"official"`
}

// initMangaSourceState inicializa o gerenciador de fontes
func initMangaSourceState() {
	if mangaSourceState != nil {
		return
	}

	dataDir := "."
	configPath := filepath.Join(dataDir, "manga_sources.json")

	mangaSourceState = &MangaSourceState{
		enabledSources: make(map[string]bool),
		configPath:     configPath,
	}

	// Carrega configuração salva
	mangaSourceState.loadConfig()

	// Habilita fontes padrão se nenhuma configuração existir
	if len(mangaSourceState.enabledSources) == 0 {
		mangaSourceState.enabledSources["mangalivre.to"] = true
		mangaSourceState.enabledSources["mangalivre.blog"] = true
		mangaSourceState.saveConfig()
	}

	fmt.Println("[MangaSources] Gerenciador de fontes inicializado")
}

func (s *MangaSourceState) loadConfig() {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return
	}

	var config struct {
		EnabledSources map[string]bool `json:"enabledSources"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return
	}

	s.enabledSources = config.EnabledSources
}

func (s *MangaSourceState) saveConfig() error {
	config := struct {
		EnabledSources map[string]bool `json:"enabledSources"`
	}{
		EnabledSources: s.enabledSources,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.configPath, data, 0644)
}

// GetAllMangaSources retorna todas as fontes de mangá disponíveis
func (a *App) GetAllMangaSources() []MangaSourceDetail {
	initMangaSourceState()

	// Lista de fontes disponíveis (hardcoded por enquanto)
	sources := []MangaSourceDetail{
		{
			ID:              "mangalivre.to",
			Name:            "MangaLivre.to",
			Description:     "Fonte principal com grande acervo de mangás em português",
			URL:             "https://mangalivre.to",
			Language:        "pt-BR",
			Icon:            "📚",
			SupportsLatest:  true,
			SupportsPopular: true,
			SupportsSearch:  true,
		},
		{
			ID:              "mangalivre.blog",
			Name:            "MangaLivre.blog",
			Description:     "Fonte alternativa com mangás atualizados frequentemente",
			URL:             "https://mangalivre.blog",
			Language:        "pt-BR",
			Icon:            "📖",
			SupportsLatest:  true,
			SupportsPopular: true,
			SupportsSearch:  true,
		},
	}

	// Define enabled state
	mangaSourceState.mu.RLock()
	defer mangaSourceState.mu.RUnlock()

	for i := range sources {
		enabled, exists := mangaSourceState.enabledSources[sources[i].ID]
		sources[i].Enabled = exists && enabled
	}

	return sources
}

// GetEnabledMangaSources retorna apenas as fontes habilitadas
func (a *App) GetEnabledMangaSources() []MangaSourceDetail {
	all := a.GetAllMangaSources()
	enabled := make([]MangaSourceDetail, 0)

	for _, s := range all {
		if s.Enabled {
			enabled = append(enabled, s)
		}
	}

	return enabled
}

// ToggleMangaSource habilita/desabilita uma fonte
func (a *App) ToggleMangaSource(sourceID string, enabled bool) error {
	initMangaSourceState()

	mangaSourceState.mu.Lock()
	defer mangaSourceState.mu.Unlock()

	mangaSourceState.enabledSources[sourceID] = enabled

	if err := mangaSourceState.saveConfig(); err != nil {
		return fmt.Errorf("erro ao salvar configuração: %w", err)
	}

	action := "desabilitada"
	if enabled {
		action = "habilitada"
	}
	fmt.Printf("[MangaSources] Fonte %s %s\n", sourceID, action)

	return nil
}

// IsMangaSourceEnabled verifica se uma fonte está habilitada
func (a *App) IsMangaSourceEnabled(sourceID string) bool {
	initMangaSourceState()

	mangaSourceState.mu.RLock()
	defer mangaSourceState.mu.RUnlock()

	enabled, exists := mangaSourceState.enabledSources[sourceID]
	return exists && enabled
}

// GetMangaSourcesByLanguage retorna fontes de um idioma específico
func (a *App) GetMangaSourcesByLanguage(language string) []MangaSourceDetail {
	all := a.GetAllMangaSources()
	filtered := make([]MangaSourceDetail, 0)

	for _, s := range all {
		if language == "all" || s.Language == language {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// GetAvailableLanguages retorna os idiomas disponíveis
func (a *App) GetAvailableLanguages() []string {
	return []string{"pt-BR", "en", "es", "ja"}
}

// ResetMangaSources restaura as configurações padrão
func (a *App) ResetMangaSources() error {
	initMangaSourceState()

	mangaSourceState.mu.Lock()
	defer mangaSourceState.mu.Unlock()

	mangaSourceState.enabledSources = map[string]bool{
		"mangalivre.to":   true,
		"mangalivre.blog": true,
	}

	return mangaSourceState.saveConfig()
}
