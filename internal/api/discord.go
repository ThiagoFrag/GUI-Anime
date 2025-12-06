// Package api - discord.go gerencia integração com Discord
package api

import (
	"fmt"
	"time"

	"goanime-gui/pkg/discord"
)

// DiscordService gerencia operações do Discord
type DiscordService struct {
	bot           *discord.Bot
	linkingSystem *discord.LinkingSystem
}

// NewDiscordService cria um novo serviço Discord
func NewDiscordService() *DiscordService {
	return &DiscordService{
		bot:           discord.GetBot(),
		linkingSystem: discord.GetLinkingSystem(),
	}
}

// DiscordRecommendation representa uma recomendação
type DiscordRecommendation struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	UserAvatar string  `json:"userAvatar"`
	AnimeTitle string  `json:"animeTitle"`
	AnimeImage string  `json:"animeImage"`
	AnimeScore float64 `json:"animeScore"`
	Message    string  `json:"message"`
	Timestamp  int64   `json:"timestamp"`
	Likes      int     `json:"likes"`
	LikedByMe  bool    `json:"likedByMe"`
}

// DiscordStatus representa o status da conexão
type DiscordStatus struct {
	Connected   bool   `json:"connected"`
	WebhookURL  string `json:"webhookUrl"`
	Username    string `json:"username"`
	ServerName  string `json:"serverName"`
	ChannelName string `json:"channelName"`
}

// DiscordLinkInfo representa informações da conta vinculada
type DiscordLinkInfo struct {
	IsLinked    bool   `json:"isLinked"`
	UserID      string `json:"userId"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	LinkedAt    string `json:"linkedAt"`
	ShowStatus  bool   `json:"showStatus"`
	ShareAnimes bool   `json:"shareAnimes"`
}

// DiscordUserInfo representa informações do usuário
type DiscordUserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
	Connected bool   `json:"connected"`
}

// DiscordFriendActivity representa atividade de um amigo
type DiscordFriendActivity struct {
	UserID     string `json:"userId"`
	Username   string `json:"username"`
	Avatar     string `json:"avatar"`
	AnimeTitle string `json:"animeTitle"`
	EpisodeNum int    `json:"episodeNum"`
	AnimeImage string `json:"animeImage"`
	IsOnline   bool   `json:"isOnline"`
}

// GetStatus retorna o status do Discord
func (s *DiscordService) GetStatus() DiscordStatus {
	return DiscordStatus{
		Connected:   s.bot.IsConnected(),
		ServerName:  "GoAnime Community",
		ChannelName: "#recomendações",
	}
}

// Connect conecta via webhook
func (s *DiscordService) Connect(webhookURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL é obrigatório")
	}

	s.bot.Configure(webhookURL)

	if len(s.bot.GetRecommendations()) == 0 {
		s.bot.AddMockRecommendations()
	}

	return nil
}

// Disconnect desconecta do Discord
func (s *DiscordService) Disconnect() {
	s.bot.Configure("")
}

// GetRecommendations retorna as recomendações
func (s *DiscordService) GetRecommendations() []DiscordRecommendation {
	recs := s.bot.GetRecommendations()

	result := make([]DiscordRecommendation, len(recs))
	for i, rec := range recs {
		result[i] = DiscordRecommendation{
			ID:         rec.ID,
			Username:   rec.Username,
			UserAvatar: rec.UserAvatar,
			AnimeTitle: rec.AnimeTitle,
			AnimeImage: rec.AnimeImage,
			AnimeScore: rec.AnimeScore,
			Message:    rec.Message,
			Timestamp:  rec.Timestamp.UnixMilli(),
			Likes:      rec.Likes,
			LikedByMe:  rec.LikedByMe,
		}
	}

	return result
}

// SendRecommendation envia uma recomendação
func (s *DiscordService) SendRecommendation(username, animeTitle, animeImage string, animeScore float64, message string) error {
	if !s.bot.IsConnected() {
		return fmt.Errorf("Discord não está conectado")
	}

	rec := discord.Recommendation{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		UserID:     "local",
		Username:   username,
		UserAvatar: "https://cdn.discordapp.com/embed/avatars/0.png",
		AnimeTitle: animeTitle,
		AnimeImage: animeImage,
		AnimeScore: animeScore,
		Message:    message,
		Timestamp:  time.Now(),
	}

	return s.bot.SendRecommendation(rec)
}

// LikeRecommendation adiciona um like
func (s *DiscordService) LikeRecommendation(recID string) bool {
	return s.bot.LikeRecommendation(recID)
}

// SimulateConnect simula conexão para demonstração
func (s *DiscordService) SimulateConnect() error {
	s.bot.Configure("demo")
	s.bot.AddMockRecommendations()
	return nil
}

// === LINKING SYSTEM ===

// GetLinkStatus retorna o status da vinculação
func (s *DiscordService) GetLinkStatus() DiscordLinkInfo {
	account := s.linkingSystem.GetLinkedAccount()

	if account == nil {
		return DiscordLinkInfo{IsLinked: false}
	}

	return DiscordLinkInfo{
		IsLinked:    true,
		UserID:      account.UserID,
		Username:    account.Username,
		Avatar:      account.Avatar,
		LinkedAt:    account.LinkedAt.Format("02/01/2006"),
		ShowStatus:  account.ShowStatus,
		ShareAnimes: account.ShareAnimes,
	}
}

// LinkWithCode vincula usando um código
func (s *DiscordService) LinkWithCode(code string) (DiscordLinkInfo, error) {
	account, err := s.linkingSystem.LinkWithCode(code)
	if err != nil {
		return DiscordLinkInfo{IsLinked: false}, err
	}

	return DiscordLinkInfo{
		IsLinked:    true,
		UserID:      account.UserID,
		Username:    account.Username,
		Avatar:      account.Avatar,
		LinkedAt:    account.LinkedAt.Format("02/01/2006"),
		ShowStatus:  account.ShowStatus,
		ShareAnimes: account.ShareAnimes,
	}, nil
}

// Unlink desvincula a conta
func (s *DiscordService) Unlink() error {
	return s.linkingSystem.Unlink()
}

// GetServerInvite retorna o link do servidor
func (s *DiscordService) GetServerInvite() string {
	return discord.GetServerInviteLink()
}

// GenerateLinkCode gera um código de vinculação
func (s *DiscordService) GenerateLinkCode() string {
	return discord.GenerateLinkCode()
}

// UpdateWatchingStatus atualiza o status de "assistindo"
func (s *DiscordService) UpdateWatchingStatus(animeTitle string, episodeNum int, animeImage string, totalEpisodes int) error {
	status := discord.WatchingStatus{
		AnimeTitle:    animeTitle,
		EpisodeNum:    episodeNum,
		AnimeImage:    animeImage,
		TotalEpisodes: totalEpisodes,
	}

	return s.linkingSystem.UpdateWatchingStatus(status)
}

// GetFriendsActivity busca atividade dos amigos
func (s *DiscordService) GetFriendsActivity() []DiscordFriendActivity {
	activities, err := s.linkingSystem.GetFriendsActivity()
	if err != nil {
		return []DiscordFriendActivity{}
	}

	result := make([]DiscordFriendActivity, len(activities))
	for i, act := range activities {
		result[i] = DiscordFriendActivity{
			UserID:     act.UserID,
			Username:   act.Username,
			Avatar:     act.Avatar,
			AnimeTitle: act.AnimeTitle,
			EpisodeNum: act.EpisodeNum,
			AnimeImage: act.AnimeImage,
			IsOnline:   act.IsOnline,
		}
	}

	return result
}

// SetShowStatus habilita/desabilita compartilhamento de status
func (s *DiscordService) SetShowStatus(enabled bool) error {
	return s.linkingSystem.SetShowStatus(enabled)
}

// SetShareAnimes habilita/desabilita compartilhamento de animes
func (s *DiscordService) SetShareAnimes(enabled bool) error {
	return s.linkingSystem.SetShareAnimes(enabled)
}

// GetUser retorna o usuário Discord conectado
func (s *DiscordService) GetUser() *DiscordUserInfo {
	user := s.bot.GetCurrentUser()

	if user == nil {
		return &DiscordUserInfo{Connected: false}
	}

	displayName := user.GlobalName
	if displayName == "" {
		displayName = user.Username
	}

	return &DiscordUserInfo{
		ID:        user.ID,
		Username:  displayName,
		AvatarURL: user.AvatarURL,
		Connected: true,
	}
}

// DisconnectUser desconecta o usuário
func (s *DiscordService) DisconnectUser() {
	s.bot.Disconnect()
}
