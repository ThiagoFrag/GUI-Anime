package discord

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LinkingSystem gerencia a vinculação de contas via código
type LinkingSystem struct {
	apiBaseURL    string
	linkedAccount *LinkedAccount
	mutex         sync.RWMutex
	configPath    string
}

// LinkedAccount representa uma conta Discord vinculada
type LinkedAccount struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Avatar      string    `json:"avatar"`
	LinkedAt    time.Time `json:"linked_at"`
	LinkCode    string    `json:"link_code"`    // Código usado para vincular
	ServerID    string    `json:"server_id"`    // Servidor Discord do GoAnime
	ShowStatus  bool      `json:"show_status"`  // Mostrar status "Assistindo..."
	ShareAnimes bool      `json:"share_animes"` // Compartilhar animes assistidos
}

// WatchingStatus representa o status atual do usuário
type WatchingStatus struct {
	UserID        string `json:"user_id"`
	AnimeTitle    string `json:"anime_title"`
	EpisodeNum    int    `json:"episode_num"`
	AnimeImage    string `json:"anime_image"`
	StartedAt     int64  `json:"started_at"`
	TotalEpisodes int    `json:"total_episodes,omitempty"`
}

// FriendActivity representa a atividade de um amigo
type FriendActivity struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	Avatar     string `json:"avatar"`
	AnimeTitle string `json:"anime_title"`
	EpisodeNum int    `json:"episode_num"`
	AnimeImage string `json:"anime_image"`
	UpdatedAt  int64  `json:"updated_at"`
	IsOnline   bool   `json:"is_online"`
}

// API do servidor GoAnime Community (você precisará hospedar isso)
// Por enquanto, vamos usar um sistema local que simula isso
const (
	// URL base da API (você pode hospedar no Vercel, Railway, etc)
	DefaultAPIURL = "https://goanime-api.vercel.app" // Placeholder

	// Discord Server do GoAnime
	GoAnimeServerInvite = "https://discord.gg/goanime"
	GoAnimeServerID     = "1234567890" // ID real do servidor
)

var (
	linkingInstance *LinkingSystem
	linkingOnce     sync.Once
)

// GetLinkingSystem retorna a instância singleton do sistema de vinculação
func GetLinkingSystem() *LinkingSystem {
	linkingOnce.Do(func() {
		// Obtém o diretório de dados do usuário
		configDir, _ := os.UserConfigDir()
		configPath := filepath.Join(configDir, "GoAnime", "discord_link.json")

		linkingInstance = &LinkingSystem{
			apiBaseURL: DefaultAPIURL,
			configPath: configPath,
		}

		// Carrega conta vinculada se existir
		linkingInstance.loadLinkedAccount()
	})
	return linkingInstance
}

// loadLinkedAccount carrega a conta vinculada do disco
func (ls *LinkingSystem) loadLinkedAccount() {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	data, err := os.ReadFile(ls.configPath)
	if err != nil {
		return // Arquivo não existe, conta não vinculada
	}

	var account LinkedAccount
	if err := json.Unmarshal(data, &account); err == nil {
		ls.linkedAccount = &account
		fmt.Printf("[Discord Link] Conta carregada: %s\n", account.Username)
	}
}

// saveLinkedAccount salva a conta vinculada no disco
func (ls *LinkingSystem) saveLinkedAccount() error {
	ls.mutex.RLock()
	account := ls.linkedAccount
	ls.mutex.RUnlock()

	if account == nil {
		// Remove o arquivo se não há conta
		os.Remove(ls.configPath)
		return nil
	}

	// Garante que o diretório existe
	dir := filepath.Dir(ls.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ls.configPath, data, 0600)
}

// GenerateLinkCode gera um código único para vinculação
func GenerateLinkCode() string {
	// Gera 4 bytes aleatórios
	bytes := make([]byte, 4)
	rand.Read(bytes)

	// Converte para hex e formata como ANIME-XXXX
	code := strings.ToUpper(hex.EncodeToString(bytes))
	return fmt.Sprintf("ANIME-%s", code[:8])
}

// IsLinked verifica se há uma conta vinculada
func (ls *LinkingSystem) IsLinked() bool {
	ls.mutex.RLock()
	defer ls.mutex.RUnlock()
	return ls.linkedAccount != nil
}

// GetLinkedAccount retorna a conta vinculada
func (ls *LinkingSystem) GetLinkedAccount() *LinkedAccount {
	ls.mutex.RLock()
	defer ls.mutex.RUnlock()

	if ls.linkedAccount == nil {
		return nil
	}

	// Retorna uma cópia
	account := *ls.linkedAccount
	return &account
}

// LinkWithCode vincula uma conta usando o código gerado pelo bot
func (ls *LinkingSystem) LinkWithCode(code string) (*LinkedAccount, error) {
	// Normaliza o código
	code = strings.ToUpper(strings.TrimSpace(code))

	// Valida o formato
	if !strings.HasPrefix(code, "ANIME-") || len(code) != 14 {
		return nil, fmt.Errorf("código inválido. Use o formato ANIME-XXXXXXXX")
	}

	// Tenta verificar o código na API
	account, err := ls.verifyCodeWithAPI(code)
	if err != nil {
		// Se a API não estiver disponível, usa modo offline/simulado
		fmt.Printf("[Discord Link] API indisponível, usando modo local: %v\n", err)
		account = ls.createLocalAccount(code)
	}

	// Salva a conta
	ls.mutex.Lock()
	ls.linkedAccount = account
	ls.mutex.Unlock()

	if err := ls.saveLinkedAccount(); err != nil {
		return nil, fmt.Errorf("erro ao salvar conta: %w", err)
	}

	fmt.Printf("[Discord Link] Conta vinculada: %s\n", account.Username)
	return account, nil
}

// verifyCodeWithAPI verifica o código com a API do servidor
func (ls *LinkingSystem) verifyCodeWithAPI(code string) (*LinkedAccount, error) {
	url := fmt.Sprintf("%s/api/link/verify?code=%s", ls.apiBaseURL, code)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("código não encontrado ou expirado")
	}

	var account LinkedAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, err
	}

	return &account, nil
}

// createLocalAccount cria uma conta local quando API não está disponível
func (ls *LinkingSystem) createLocalAccount(code string) *LinkedAccount {
	// Extrai um "ID" do código
	codeID := strings.TrimPrefix(code, "ANIME-")

	return &LinkedAccount{
		UserID:      fmt.Sprintf("local_%s", codeID),
		Username:    fmt.Sprintf("Usuário GoAnime"),
		Avatar:      "",
		LinkedAt:    time.Now(),
		LinkCode:    code,
		ServerID:    GoAnimeServerID,
		ShowStatus:  true,
		ShareAnimes: true,
	}
}

// Unlink remove a vinculação da conta
func (ls *LinkingSystem) Unlink() error {
	ls.mutex.Lock()
	ls.linkedAccount = nil
	ls.mutex.Unlock()

	// Remove o arquivo de configuração
	os.Remove(ls.configPath)

	fmt.Println("[Discord Link] Conta desvinculada")
	return nil
}

// UpdateWatchingStatus atualiza o status "Assistindo..." no Discord
func (ls *LinkingSystem) UpdateWatchingStatus(status WatchingStatus) error {
	account := ls.GetLinkedAccount()
	if account == nil {
		return fmt.Errorf("conta não vinculada")
	}

	if !account.ShowStatus {
		return nil // Usuário desabilitou compartilhamento de status
	}

	status.UserID = account.UserID
	status.StartedAt = time.Now().Unix()

	// Envia para a API (se disponível)
	go ls.sendStatusToAPI(status)

	// Também envia via webhook do Discord
	go ls.sendStatusViaWebhook(status, account)

	return nil
}

// sendStatusToAPI envia o status para a API
func (ls *LinkingSystem) sendStatusToAPI(status WatchingStatus) {
	data, _ := json.Marshal(status)

	url := fmt.Sprintf("%s/api/status/update", ls.apiBaseURL)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Printf("[Discord Link] Erro ao atualizar status na API: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

// sendStatusViaWebhook envia o status via webhook do Discord
func (ls *LinkingSystem) sendStatusViaWebhook(status WatchingStatus, account *LinkedAccount) {
	bot := GetBot()
	if !bot.IsConnected() {
		return
	}

	// Cor verde para "assistindo"
	embedColor := 0x00D166

	episodeText := fmt.Sprintf("Episódio %d", status.EpisodeNum)
	if status.TotalEpisodes > 0 {
		episodeText = fmt.Sprintf("Episódio %d/%d", status.EpisodeNum, status.TotalEpisodes)
	}

	msg := WebhookMessage{
		Username:  "GoAnime Activity",
		AvatarURL: "https://raw.githubusercontent.com/alvarorichard/GoAnime/main/assets/logo.png",
		Embeds: []Embed{
			{
				Title:       fmt.Sprintf("📺 %s", status.AnimeTitle),
				Description: fmt.Sprintf("**%s** está assistindo agora!", account.Username),
				Color:       embedColor,
				Thumbnail: &Thumbnail{
					URL: status.AnimeImage,
				},
				Fields: []Field{
					{
						Name:   "🎬 Episódio",
						Value:  episodeText,
						Inline: true,
					},
				},
				Footer: &Footer{
					Text: "GoAnime • Atividade ao Vivo",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	bot.SendWebhookMessage(msg)
}

// GetFriendsActivity busca a atividade dos amigos
func (ls *LinkingSystem) GetFriendsActivity() ([]FriendActivity, error) {
	account := ls.GetLinkedAccount()
	if account == nil {
		return nil, fmt.Errorf("conta não vinculada")
	}

	// Tenta buscar da API
	activities, err := ls.fetchFriendsFromAPI(account.UserID)
	if err != nil {
		// Retorna lista vazia se API não disponível
		fmt.Printf("[Discord Link] API indisponível para amigos: %v\n", err)
		return []FriendActivity{}, nil
	}

	return activities, nil
}

// fetchFriendsFromAPI busca atividades dos amigos da API
func (ls *LinkingSystem) fetchFriendsFromAPI(userID string) ([]FriendActivity, error) {
	url := fmt.Sprintf("%s/api/friends/activity?user_id=%s", ls.apiBaseURL, userID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro ao buscar amigos: status %d", resp.StatusCode)
	}

	var activities []FriendActivity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, err
	}

	return activities, nil
}

// GetServerInviteLink retorna o link de convite do servidor
func GetServerInviteLink() string {
	return GoAnimeServerInvite
}

// SetShowStatus habilita/desabilita compartilhamento de status
func (ls *LinkingSystem) SetShowStatus(enabled bool) error {
	ls.mutex.Lock()
	if ls.linkedAccount != nil {
		ls.linkedAccount.ShowStatus = enabled
	}
	ls.mutex.Unlock()

	return ls.saveLinkedAccount()
}

// SetShareAnimes habilita/desabilita compartilhamento de animes
func (ls *LinkingSystem) SetShareAnimes(enabled bool) error {
	ls.mutex.Lock()
	if ls.linkedAccount != nil {
		ls.linkedAccount.ShareAnimes = enabled
	}
	ls.mutex.Unlock()

	return ls.saveLinkedAccount()
}
