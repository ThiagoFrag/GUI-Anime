// social_methods.go - Métodos do sistema social para o frontend
// Expõe funcionalidades de amigos, perfil e atividades
package main

import (
	"time"

	"GoAnimeGUI/pkg/social"
)

var friendSystem *social.FriendSystem

// getFriendSystem retorna ou cria o sistema de amigos singleton
func getFriendSystem() *social.FriendSystem {
	if friendSystem == nil {
		friendSystem = social.GetFriendSystem()
	}
	return friendSystem
}

// ==============================
// PERFIL SOCIAL
// ==============================

// HasSocialProfile verifica se existe um perfil social
func (a *App) HasSocialProfile() bool {
	fs := getFriendSystem()
	return fs.HasProfile()
}

// GetSocialProfile retorna o perfil social do usuário
func (a *App) GetSocialProfile() *social.UserProfile {
	fs := getFriendSystem()
	return fs.GetProfile()
}

// CreateSocialProfile cria um novo perfil social
func (a *App) CreateSocialProfile(username string) (*social.UserProfile, error) {
	fs := getFriendSystem()
	return fs.CreateProfile(username)
}

// UpdateSocialUsername atualiza o nome de usuário
func (a *App) UpdateSocialUsername(username string) error {
	fs := getFriendSystem()
	return fs.UpdateUsername(username)
}

// RegenerateSocialShareCode regenera o código de compartilhamento
func (a *App) RegenerateSocialShareCode() (string, error) {
	fs := getFriendSystem()
	return fs.RegenerateShareCode()
}

// DeleteSocialProfile deleta o perfil social
func (a *App) DeleteSocialProfile() error {
	fs := getFriendSystem()
	return fs.DeleteProfile()
}

// ==============================
// CONFIGURAÇÕES DE PRIVACIDADE
// ==============================

// SetSocialShowStatus ativa/desativa mostrar status online
func (a *App) SetSocialShowStatus(enabled bool) {
	fs := getFriendSystem()
	fs.SetShowStatus(enabled)
}

// SetSocialShareAnimes ativa/desativa compartilhar animes assistidos
func (a *App) SetSocialShareAnimes(enabled bool) {
	fs := getFriendSystem()
	fs.SetShareAnimes(enabled)
}

// ==============================
// AMIGOS
// ==============================

// AddFriendByCode adiciona um amigo pelo código de compartilhamento
func (a *App) AddFriendByCode(shareCode string) (*social.Friend, error) {
	fs := getFriendSystem()
	return fs.AddFriendByCode(shareCode)
}

// RemoveFriend remove um amigo
func (a *App) RemoveFriend(userID string) error {
	fs := getFriendSystem()
	return fs.RemoveFriend(userID)
}

// GetFriendsList retorna a lista de amigos
func (a *App) GetFriendsList() []social.Friend {
	fs := getFriendSystem()
	return fs.GetFriends()
}

// GetFriendsActivity retorna a atividade dos amigos
func (a *App) GetFriendsActivity() []social.FriendActivity {
	fs := getFriendSystem()
	activities, _ := fs.GetFriendsActivity() // ignora erro, retorna slice vazio se falhar
	if activities == nil {
		return []social.FriendActivity{}
	}
	return activities
}

// ==============================
// STATUS DE VISUALIZAÇÃO
// ==============================

// UpdateSocialWatchingStatus atualiza o status de visualização
func (a *App) UpdateSocialWatchingStatus(animeTitle string, animeImage string, episodeNum int, totalEpisodes int) error {
	fs := getFriendSystem()
	status := &social.WatchingStatus{
		AnimeTitle:    animeTitle,
		AnimeImage:    animeImage,
		EpisodeNum:    episodeNum,
		TotalEpisodes: totalEpisodes,
		StartedAt:     time.Now().Unix(),
	}
	return fs.UpdateWatchingStatus(status)
}

// ClearSocialWatchingStatus limpa o status de visualização
func (a *App) ClearSocialWatchingStatus() {
	fs := getFriendSystem()
	fs.ClearWatchingStatus()
}

// ==============================
// CONEXÃO E SINCRONIZAÇÃO
// ==============================

// GetSocialConnectionStatus retorna o status da conexão com o servidor social
func (a *App) GetSocialConnectionStatus() (bool, string) {
	fs := getFriendSystem()
	return fs.GetConnectionStatus()
}

// SyncSocialWithServer sincroniza dados com o servidor
func (a *App) SyncSocialWithServer() error {
	fs := getFriendSystem()
	return fs.SyncWithServer()
}
