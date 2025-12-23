package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager gerencia todas as extensions instaladas e repositórios
type Manager struct {
	mu           sync.RWMutex
	extensions   map[string]*InstalledExtension
	repositories []Repository
	dataDir      string
	httpClient   *http.Client
}

// NewManager cria um novo gerenciador de extensions
func NewManager(dataDir string) *Manager {
	return &Manager{
		extensions:   make(map[string]*InstalledExtension),
		repositories: []Repository{},
		dataDir:      dataDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Initialize carrega extensions instaladas e repositórios
func (m *Manager) Initialize() error {
	// Cria diretórios necessários
	dirs := []string{
		filepath.Join(m.dataDir, "extensions"),
		filepath.Join(m.dataDir, "extensions", "scripts"),
		filepath.Join(m.dataDir, "extensions", "icons"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório %s: %w", dir, err)
		}
	}

	// Carrega configuração salva
	if err := m.loadConfig(); err != nil {
		// Não é erro crítico se não existir config ainda
		fmt.Printf("[Extensions] Nenhuma config anterior encontrada: %v\n", err)
	}

	// Carrega extensions instaladas
	if err := m.loadInstalledExtensions(); err != nil {
		return fmt.Errorf("erro ao carregar extensions: %w", err)
	}

	// Adiciona repositório oficial se não existir
	if len(m.repositories) == 0 {
		m.repositories = append(m.repositories, Repository{
			Name:     "GoAnime Official",
			URL:      "https://raw.githubusercontent.com/goanime/extensions/main/index.json",
			Official: true,
		})
	}

	return nil
}

// GetExtensions retorna todas as extensions instaladas
func (m *Manager) GetExtensions() []*InstalledExtension {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*InstalledExtension, 0, len(m.extensions))
	for _, ext := range m.extensions {
		result = append(result, ext)
	}
	return result
}

// GetExtension retorna uma extension específica pelo ID
func (m *Manager) GetExtension(id string) (*InstalledExtension, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ext, ok := m.extensions[id]
	return ext, ok
}

// GetEnabledSources retorna apenas sources habilitadas
func (m *Manager) GetEnabledSources() []ExtensionSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sources []ExtensionSource
	for _, ext := range m.extensions {
		if ext.State == ExtensionStateEnabled && ext.Source != nil {
			sources = append(sources, ext.Source)
		}
	}
	return sources
}

// GetSourcesByLanguage retorna sources de um idioma específico
func (m *Manager) GetSourcesByLanguage(lang string) []ExtensionSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sources []ExtensionSource
	for _, ext := range m.extensions {
		if ext.State == ExtensionStateEnabled && ext.Source != nil {
			info := ext.Source.GetInfo()
			if info.Language == lang || info.Language == "multi" {
				sources = append(sources, ext.Source)
			}
		}
	}
	return sources
}

// InstallFromRepository instala uma extension de um repositório
func (m *Manager) InstallFromRepository(ctx context.Context, repoURL, extensionID string) error {
	// Busca índice do repositório
	index, err := m.fetchRepositoryIndex(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("erro ao buscar índice do repositório: %w", err)
	}

	// Encontra a extension
	var remoteExt *RemoteExtension
	for _, ext := range index.Extensions {
		if ext.ID == extensionID {
			remoteExt = &ext
			break
		}
	}

	if remoteExt == nil {
		return fmt.Errorf("extension %s não encontrada no repositório", extensionID)
	}

	// Baixa o script
	scriptPath := filepath.Join(m.dataDir, "extensions", "scripts", extensionID+".lua")
	if err := m.downloadFile(ctx, remoteExt.ScriptURL, scriptPath); err != nil {
		return fmt.Errorf("erro ao baixar script: %w", err)
	}

	// Baixa o ícone
	iconPath := filepath.Join(m.dataDir, "extensions", "icons", extensionID+".png")
	if remoteExt.IconURL != "" {
		_ = m.downloadFile(ctx, remoteExt.IconURL, iconPath) // Ícone é opcional
	}

	// Carrega a extension
	return m.loadExtensionFromFile(extensionID, scriptPath)
}

// InstallFromFile instala uma extension de um arquivo local
func (m *Manager) InstallFromFile(scriptPath string) error {
	// Lê o script
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	// Tenta criar a extension para validar
	ext, err := NewLuaExtension(string(content))
	if err != nil {
		return fmt.Errorf("erro ao carregar extension: %w", err)
	}

	info := ext.GetInfo()

	// Copia para o diretório de extensions
	destPath := filepath.Join(m.dataDir, "extensions", "scripts", info.ID+".lua")
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("erro ao salvar extension: %w", err)
	}

	// Registra a extension
	m.mu.Lock()
	m.extensions[info.ID] = &InstalledExtension{
		Info:       info,
		State:      ExtensionStateEnabled,
		Source:     ext,
		ScriptPath: destPath,
		UpdatedAt:  time.Now(),
	}
	m.mu.Unlock()

	// Salva configuração
	return m.saveConfig()
}

// UninstallExtension remove uma extension
func (m *Manager) UninstallExtension(id string) error {
	m.mu.Lock()
	ext, ok := m.extensions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("extension %s não encontrada", id)
	}

	// Remove do mapa
	delete(m.extensions, id)
	m.mu.Unlock()

	// Remove arquivos
	if ext.ScriptPath != "" {
		os.Remove(ext.ScriptPath)
	}
	iconPath := filepath.Join(m.dataDir, "extensions", "icons", id+".png")
	os.Remove(iconPath)

	return m.saveConfig()
}

// EnableExtension habilita uma extension
func (m *Manager) EnableExtension(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ext, ok := m.extensions[id]
	if !ok {
		return fmt.Errorf("extension %s não encontrada", id)
	}

	ext.State = ExtensionStateEnabled
	return m.saveConfig()
}

// DisableExtension desabilita uma extension
func (m *Manager) DisableExtension(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ext, ok := m.extensions[id]
	if !ok {
		return fmt.Errorf("extension %s não encontrada", id)
	}

	ext.State = ExtensionStateDisabled
	return m.saveConfig()
}

// GetRepositories retorna os repositórios configurados
func (m *Manager) GetRepositories() []Repository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.repositories
}

// AddRepository adiciona um novo repositório
func (m *Manager) AddRepository(name, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verifica se já existe
	for _, repo := range m.repositories {
		if repo.URL == url {
			return fmt.Errorf("repositório já existe")
		}
	}

	m.repositories = append(m.repositories, Repository{
		Name:     name,
		URL:      url,
		Official: false,
	})

	return m.saveConfig()
}

// RemoveRepository remove um repositório
func (m *Manager) RemoveRepository(url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, repo := range m.repositories {
		if repo.URL == url {
			if repo.Official {
				return fmt.Errorf("não é possível remover repositório oficial")
			}
			m.repositories = append(m.repositories[:i], m.repositories[i+1:]...)
			return m.saveConfig()
		}
	}

	return fmt.Errorf("repositório não encontrado")
}

// FetchRepositoryExtensions busca extensions disponíveis em um repositório
func (m *Manager) FetchRepositoryExtensions(ctx context.Context, repoURL string) ([]RemoteExtension, error) {
	index, err := m.fetchRepositoryIndex(ctx, repoURL)
	if err != nil {
		return nil, err
	}
	return index.Extensions, nil
}

// CheckUpdates verifica se há atualizações disponíveis
func (m *Manager) CheckUpdates(ctx context.Context) (map[string]string, error) {
	updates := make(map[string]string) // extensionID -> nova versão

	for _, repo := range m.repositories {
		index, err := m.fetchRepositoryIndex(ctx, repo.URL)
		if err != nil {
			continue
		}

		for _, remote := range index.Extensions {
			m.mu.RLock()
			installed, ok := m.extensions[remote.ID]
			m.mu.RUnlock()

			if ok && remote.Version > installed.Info.Version {
				updates[remote.ID] = remote.Version
			}
		}
	}

	return updates, nil
}

// --- Métodos privados ---

func (m *Manager) loadConfig() error {
	configPath := filepath.Join(m.dataDir, "extensions", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config struct {
		Repositories []Repository `json:"repositories"`
		Disabled     []string     `json:"disabled"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	m.repositories = config.Repositories
	return nil
}

func (m *Manager) saveConfig() error {
	config := struct {
		Repositories []Repository `json:"repositories"`
		Disabled     []string     `json:"disabled"`
	}{
		Repositories: m.repositories,
		Disabled:     []string{},
	}

	for id, ext := range m.extensions {
		if ext.State == ExtensionStateDisabled {
			config.Disabled = append(config.Disabled, id)
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(m.dataDir, "extensions", "config.json")
	return os.WriteFile(configPath, data, 0644)
}

func (m *Manager) loadInstalledExtensions() error {
	scriptsDir := filepath.Join(m.dataDir, "extensions", "scripts")
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Diretório não existe ainda
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".lua" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-4] // Remove .lua
		scriptPath := filepath.Join(scriptsDir, entry.Name())

		if err := m.loadExtensionFromFile(id, scriptPath); err != nil {
			fmt.Printf("[Extensions] Erro ao carregar %s: %v\n", id, err)
		}
	}

	return nil
}

func (m *Manager) loadExtensionFromFile(id, scriptPath string) error {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}

	ext, err := NewLuaExtension(string(content))
	if err != nil {
		m.mu.Lock()
		m.extensions[id] = &InstalledExtension{
			Info:       ExtensionInfo{ID: id, Name: id},
			State:      ExtensionStateError,
			ScriptPath: scriptPath,
			Error:      err.Error(),
			UpdatedAt:  time.Now(),
		}
		m.mu.Unlock()
		return err
	}

	info := ext.GetInfo()

	m.mu.Lock()
	m.extensions[info.ID] = &InstalledExtension{
		Info:       info,
		State:      ExtensionStateEnabled,
		Source:     ext,
		ScriptPath: scriptPath,
		UpdatedAt:  time.Now(),
	}
	m.mu.Unlock()

	fmt.Printf("[Extensions] Carregado: %s v%s\n", info.Name, info.Version)
	return nil
}

func (m *Manager) fetchRepositoryIndex(ctx context.Context, url string) (*RepositoryIndex, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var index RepositoryIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}

	return &index, nil
}

func (m *Manager) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
