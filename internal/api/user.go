// Package api - user.go gerencia operações relacionadas ao usuário
package api

import (
	"encoding/json"
	"fmt"
	"time"

	"GoAnimeGUI/pkg/store"
)

// UserService gerencia operações do usuário
type UserService struct {
	user *store.UserData
}

// NewUserService cria um novo serviço de usuário
func NewUserService(user *store.UserData) *UserService {
	return &UserService{user: user}
}

// SetUser define o usuário atual
func (s *UserService) SetUser(user *store.UserData) {
	s.user = user
}

// GetUser retorna o usuário atual
func (s *UserService) GetUser() *store.UserData {
	return s.user
}

// CreateUser cria um novo usuário
func (s *UserService) CreateUser(username, avatar string) *store.UserData {
	newUser := &store.UserData{
		Username:     username,
		Avatar:       avatar,
		History:      []store.SavedAnime{},
		Favorites:    []store.SavedAnime{},
		WatchHistory: []store.WatchedEpisode{},
		Settings:     store.GetDefaultSettings(),
	}
	s.user = newUser
	_ = store.SaveUser(s.user)
	return s.user
}

// === FAVORITOS ===

// GetFavorites retorna a lista de favoritos
func (s *UserService) GetFavorites() []store.SavedAnime {
	if s.user == nil {
		return []store.SavedAnime{}
	}
	return s.user.Favorites
}

// AddToFavorites adiciona um anime aos favoritos
func (s *UserService) AddToFavorites(anime store.SavedAnime) bool {
	if s.user == nil {
		return false
	}

	// Verifica se já existe
	for _, fav := range s.user.Favorites {
		if fav.URL == anime.URL || fav.Title == anime.Title {
			return false
		}
	}

	s.user.Favorites = append(s.user.Favorites, anime)
	_ = store.SaveUser(s.user)
	return true
}

// RemoveFromFavorites remove um anime dos favoritos
func (s *UserService) RemoveFromFavorites(animeURL string) bool {
	if s.user == nil {
		return false
	}

	for i, fav := range s.user.Favorites {
		if fav.URL == animeURL {
			s.user.Favorites = append(s.user.Favorites[:i], s.user.Favorites[i+1:]...)
			_ = store.SaveUser(s.user)
			return true
		}
	}
	return false
}

// IsFavorite verifica se um anime está nos favoritos
func (s *UserService) IsFavorite(animeURL string) bool {
	if s.user == nil {
		return false
	}
	for _, fav := range s.user.Favorites {
		if fav.URL == animeURL {
			return true
		}
	}
	return false
}

// === HISTÓRICO ===

// GetWatchHistory retorna o histórico de episódios assistidos
func (s *UserService) GetWatchHistory() []store.WatchedEpisode {
	if s.user == nil {
		return []store.WatchedEpisode{}
	}
	return s.user.WatchHistory
}

// AddToWatchHistory adiciona um episódio ao histórico
func (s *UserService) AddToWatchHistory(episode store.WatchedEpisode) {
	if s.user == nil {
		return
	}

	if episode.WatchedAt == "" {
		episode.WatchedAt = time.Now().Format(time.RFC3339)
	}

	// Remove duplicata
	for i, e := range s.user.WatchHistory {
		if e.EpisodeURL == episode.EpisodeURL {
			s.user.WatchHistory = append(s.user.WatchHistory[:i], s.user.WatchHistory[i+1:]...)
			break
		}
	}

	// Adiciona no início
	s.user.WatchHistory = append([]store.WatchedEpisode{episode}, s.user.WatchHistory...)

	// Limita a 50 entradas
	if len(s.user.WatchHistory) > 50 {
		s.user.WatchHistory = s.user.WatchHistory[:50]
	}

	_ = store.SaveUser(s.user)
}

// === CONFIGURAÇÕES ===

// GetSettings retorna as configurações do usuário
func (s *UserService) GetSettings() store.UserSettings {
	if s.user == nil {
		return store.GetDefaultSettings()
	}
	return s.user.Settings
}

// SaveSettings salva as configurações do usuário
func (s *UserService) SaveSettings(settings store.UserSettings) bool {
	if s.user == nil {
		return false
	}
	s.user.Settings = settings
	_ = store.SaveUser(s.user)
	return true
}

// === EXPORT / IMPORT ===

// ExportUserData exporta dados do usuário como JSON
func (s *UserService) ExportUserData() (string, error) {
	if s.user == nil {
		return "", fmt.Errorf("usuário não encontrado")
	}

	data, err := json.MarshalIndent(s.user, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportUserData importa dados do usuário de JSON
func (s *UserService) ImportUserData(jsonData string) error {
	var userData store.UserData
	if err := json.Unmarshal([]byte(jsonData), &userData); err != nil {
		return fmt.Errorf("erro ao processar JSON: %w", err)
	}

	if userData.Username == "" {
		return fmt.Errorf("nome de usuário inválido")
	}

	if userData.Settings.ContentLanguage == "" {
		userData.Settings.ContentLanguage = "all"
	}

	s.user = &userData
	_ = store.SaveUser(s.user)
	return nil
}

// GetMPVPath retorna o caminho do MPV salvo
func (s *UserService) GetMPVPath() string {
	if s.user == nil {
		return ""
	}
	return s.user.MPVPath
}

// SetMPVPath salva o caminho do MPV
func (s *UserService) SetMPVPath(path string) {
	if s.user == nil {
		return
	}
	s.user.MPVPath = path
	_ = store.SaveUser(s.user)
}
