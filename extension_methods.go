package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"GoAnimeGUI/pkg/extensions"
)

// extensionManager é o gerenciador global de extensions
var extensionManager *extensions.Manager

// initExtensions inicializa o sistema de extensions
func initExtensions() error {
	if extensionManager != nil {
		return nil // Já inicializado
	}

	// Obtém diretório de dados do app
	dataDir := "." // Por enquanto usa diretório atual

	extensionManager = extensions.NewManager(dataDir)
	if err := extensionManager.Initialize(); err != nil {
		return fmt.Errorf("erro ao inicializar extensions: %w", err)
	}

	fmt.Printf("[Extensions] Sistema inicializado\n")
	return nil
}

// ExtensionInfo para o frontend
type ExtensionInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language"`
	IconURL  string `json:"iconUrl"`
	Enabled  bool   `json:"enabled"`
	HasError bool   `json:"hasError"`
	Error    string `json:"error,omitempty"`
}

// RepositoryInfo para o frontend
type RepositoryInfo struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Official bool   `json:"official"`
}

// RemoteExtensionInfo para o frontend
type RemoteExtensionInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Language  string `json:"language"`
	IconURL   string `json:"iconUrl"`
	Changelog string `json:"changelog"`
	Installed bool   `json:"installed"`
}

// GetInstalledExtensions retorna as extensions instaladas
func (a *App) GetInstalledExtensions() []ExtensionInfo {
	if err := initExtensions(); err != nil {
		fmt.Printf("[Extensions] Erro: %v\n", err)
		return []ExtensionInfo{}
	}

	exts := extensionManager.GetExtensions()
	result := make([]ExtensionInfo, 0, len(exts))

	for _, ext := range exts {
		result = append(result, ExtensionInfo{
			ID:       ext.Info.ID,
			Name:     ext.Info.Name,
			Version:  ext.Info.Version,
			Language: ext.Info.Language,
			IconURL:  ext.Info.IconURL,
			Enabled:  ext.State == extensions.ExtensionStateEnabled,
			HasError: ext.State == extensions.ExtensionStateError,
			Error:    ext.Error,
		})
	}

	return result
}

// GetExtensionRepositories retorna os repositórios configurados
func (a *App) GetExtensionRepositories() []RepositoryInfo {
	if err := initExtensions(); err != nil {
		return []RepositoryInfo{}
	}

	repos := extensionManager.GetRepositories()
	result := make([]RepositoryInfo, 0, len(repos))

	for _, repo := range repos {
		result = append(result, RepositoryInfo{
			Name:     repo.Name,
			URL:      repo.URL,
			Official: repo.Official,
		})
	}

	return result
}

// FetchRepositoryExtensions busca extensions disponíveis em um repositório
func (a *App) FetchRepositoryExtensions(repoURL string) []RemoteExtensionInfo {
	if err := initExtensions(); err != nil {
		return []RemoteExtensionInfo{}
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	remoteExts, err := extensionManager.FetchRepositoryExtensions(ctx, repoURL)
	if err != nil {
		fmt.Printf("[Extensions] Erro ao buscar repositório: %v\n", err)
		return []RemoteExtensionInfo{}
	}

	result := make([]RemoteExtensionInfo, 0, len(remoteExts))

	for _, ext := range remoteExts {
		_, installed := extensionManager.GetExtension(ext.ID)
		result = append(result, RemoteExtensionInfo{
			ID:        ext.ID,
			Name:      ext.Name,
			Version:   ext.Version,
			Language:  ext.Language,
			IconURL:   ext.IconURL,
			Changelog: ext.Changelog,
			Installed: installed,
		})
	}

	return result
}

// InstallExtension instala uma extension de um repositório
func (a *App) InstallExtension(repoURL, extensionID string) error {
	if err := initExtensions(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()

	if err := extensionManager.InstallFromRepository(ctx, repoURL, extensionID); err != nil {
		return fmt.Errorf("erro ao instalar extension: %w", err)
	}

	fmt.Printf("[Extensions] Instalado: %s\n", extensionID)
	return nil
}

// InstallExtensionFromFile instala uma extension de um arquivo local
func (a *App) InstallExtensionFromFile(filePath string) error {
	if err := initExtensions(); err != nil {
		return err
	}

	// Verifica extensão do arquivo
	if ext := filepath.Ext(filePath); ext != ".lua" {
		return fmt.Errorf("arquivo deve ter extensão .lua")
	}

	if err := extensionManager.InstallFromFile(filePath); err != nil {
		return fmt.Errorf("erro ao instalar extension: %w", err)
	}

	fmt.Printf("[Extensions] Instalado de arquivo: %s\n", filePath)
	return nil
}

// UninstallExtension remove uma extension
func (a *App) UninstallExtension(extensionID string) error {
	if err := initExtensions(); err != nil {
		return err
	}

	if err := extensionManager.UninstallExtension(extensionID); err != nil {
		return fmt.Errorf("erro ao desinstalar extension: %w", err)
	}

	fmt.Printf("[Extensions] Desinstalado: %s\n", extensionID)
	return nil
}

// ToggleExtension habilita/desabilita uma extension
func (a *App) ToggleExtension(extensionID string, enabled bool) error {
	if err := initExtensions(); err != nil {
		return err
	}

	if enabled {
		return extensionManager.EnableExtension(extensionID)
	}
	return extensionManager.DisableExtension(extensionID)
}

// AddExtensionRepository adiciona um novo repositório
func (a *App) AddExtensionRepository(name, url string) error {
	if err := initExtensions(); err != nil {
		return err
	}

	return extensionManager.AddRepository(name, url)
}

// RemoveExtensionRepository remove um repositório
func (a *App) RemoveExtensionRepository(url string) error {
	if err := initExtensions(); err != nil {
		return err
	}

	return extensionManager.RemoveRepository(url)
}

// CheckExtensionUpdates verifica se há atualizações disponíveis
func (a *App) CheckExtensionUpdates() map[string]string {
	if err := initExtensions(); err != nil {
		return map[string]string{}
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	updates, err := extensionManager.CheckUpdates(ctx)
	if err != nil {
		fmt.Printf("[Extensions] Erro ao verificar atualizações: %v\n", err)
		return map[string]string{}
	}

	return updates
}

// SearchWithExtension busca anime usando uma extension específica
func (a *App) SearchWithExtension(extensionID, query string, page int) ([]extensions.AnimeEntry, bool, error) {
	if err := initExtensions(); err != nil {
		return nil, false, err
	}

	ext, ok := extensionManager.GetExtension(extensionID)
	if !ok {
		return nil, false, fmt.Errorf("extension não encontrada: %s", extensionID)
	}

	if ext.Source == nil {
		return nil, false, fmt.Errorf("extension sem source: %s", extensionID)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	return ext.Source.Search(ctx, query, page, nil)
}

// GetExtensionLatest busca últimos lançamentos de uma extension
func (a *App) GetExtensionLatest(extensionID string, page int) ([]extensions.AnimeEntry, bool, error) {
	if err := initExtensions(); err != nil {
		return nil, false, err
	}

	ext, ok := extensionManager.GetExtension(extensionID)
	if !ok {
		return nil, false, fmt.Errorf("extension não encontrada: %s", extensionID)
	}

	if ext.Source == nil {
		return nil, false, fmt.Errorf("extension sem source: %s", extensionID)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	return ext.Source.GetLatest(ctx, page)
}

// GetExtensionAnimeDetails obtém detalhes de um anime via extension
func (a *App) GetExtensionAnimeDetails(extensionID, animeURL string) (*extensions.AnimeDetails, error) {
	if err := initExtensions(); err != nil {
		return nil, err
	}

	ext, ok := extensionManager.GetExtension(extensionID)
	if !ok {
		return nil, fmt.Errorf("extension não encontrada: %s", extensionID)
	}

	if ext.Source == nil {
		return nil, fmt.Errorf("extension sem source: %s", extensionID)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	return ext.Source.GetAnimeDetails(ctx, animeURL)
}

// GetExtensionEpisodes obtém episódios de um anime via extension
func (a *App) GetExtensionEpisodes(extensionID, animeURL string) ([]extensions.Episode, error) {
	if err := initExtensions(); err != nil {
		return nil, err
	}

	ext, ok := extensionManager.GetExtension(extensionID)
	if !ok {
		return nil, fmt.Errorf("extension não encontrada: %s", extensionID)
	}

	if ext.Source == nil {
		return nil, fmt.Errorf("extension sem source: %s", extensionID)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	return ext.Source.GetEpisodes(ctx, animeURL)
}

// GetExtensionVideoSources obtém fontes de vídeo de um episódio via extension
func (a *App) GetExtensionVideoSources(extensionID, episodeURL string) ([]extensions.VideoSource, error) {
	if err := initExtensions(); err != nil {
		return nil, err
	}

	ext, ok := extensionManager.GetExtension(extensionID)
	if !ok {
		return nil, fmt.Errorf("extension não encontrada: %s", extensionID)
	}

	if ext.Source == nil {
		return nil, fmt.Errorf("extension sem source: %s", extensionID)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()

	return ext.Source.GetVideoSources(ctx, episodeURL)
}
