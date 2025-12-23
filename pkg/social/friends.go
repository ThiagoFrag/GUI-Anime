package social

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ============================================
// CONFIGURAÇÕES E CONSTANTES
// ============================================

const (
	// API Base URL - Usando a mesma VPS do streaming
	// O servidor social será um endpoint adicional na VPS existente
	APIBaseURL = "http://[2804:54:c100:2::11]:8080/social"

	// Timeouts para requisições HTTP
	HTTPTimeout = 5 * time.Second

	// Intervalo de heartbeat para status online
	HeartbeatInterval = 30 * time.Second
)

// ============================================
// ESTRUTURAS DE DADOS
// ============================================

// FriendSystem gerencia o sistema de amizade
type FriendSystem struct {
	apiBaseURL    string
	profile       *UserProfile
	authToken     string // Token JWT para autenticação
	friends       []Friend
	mutex         sync.RWMutex
	configPath    string
	httpClient    *http.Client
	heartbeatStop chan struct{}
	isOnline      bool // Status de conexão com servidor
}

// UserProfile perfil do usuário local
type UserProfile struct {
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Avatar       string    `json:"avatar,omitempty"`
	ShareCode    string    `json:"share_code"`
	AuthToken    string    `json:"auth_token,omitempty"` // Token para autenticação
	CreatedAt    time.Time `json:"created_at"`
	ShowStatus   bool      `json:"show_status"`
	ShareAnimes  bool      `json:"share_animes"`
	TotalWatched int       `json:"total_watched"`
	LastSync     time.Time `json:"last_sync"`
}

// Friend representa um amigo
type Friend struct {
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Avatar       string    `json:"avatar,omitempty"`
	ShareCode    string    `json:"share_code,omitempty"`
	AddedAt      time.Time `json:"added_at"`
	IsOnline     bool      `json:"is_online"`
	LastSeen     time.Time `json:"last_seen"`
	CurrentAnime string    `json:"current_anime,omitempty"`
	CurrentEp    int       `json:"current_ep,omitempty"`
	TotalWatched int       `json:"total_watched"`
}

// FriendActivity atividade de um amigo
type FriendActivity struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Avatar       string `json:"avatar,omitempty"`
	AnimeTitle   string `json:"anime_title,omitempty"`
	AnimeImage   string `json:"anime_image,omitempty"`
	EpisodeNum   int    `json:"episode_num,omitempty"`
	IsWatching   bool   `json:"is_watching"`
	IsOnline     bool   `json:"is_online"`
	LastActivity string `json:"last_activity"`
}

// WatchingStatus status atual de visualização
type WatchingStatus struct {
	AnimeTitle    string `json:"anime_title"`
	AnimeImage    string `json:"anime_image"`
	EpisodeNum    int    `json:"episode_num"`
	TotalEpisodes int    `json:"total_episodes"`
	StartedAt     int64  `json:"started_at"`
}

// AnimeRecommendation representa uma recomendação de anime da comunidade
type AnimeRecommendation struct {
	AnimeID       string   `json:"anime_id"`
	Title         string   `json:"title"`
	TitleAlt      string   `json:"title_alt,omitempty"`
	Image         string   `json:"image"`
	Description   string   `json:"description,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Rating        float64  `json:"rating"`
	WatchCount    int      `json:"watch_count"`    // Quantos usuários assistiram
	Trending      bool     `json:"trending"`       // Se está em alta
	RecommendedBy string   `json:"recommended_by"` // Motivo da recomendação
	Score         float64  `json:"score"`          // Score de relevância
}

// TrendingAnime anime em alta na comunidade
type TrendingAnime struct {
	AnimeID    string  `json:"anime_id"`
	Title      string  `json:"title"`
	Image      string  `json:"image"`
	Watchers   int     `json:"watchers"` // Quantos estão assistindo agora
	TotalViews int     `json:"total_views"`
	TrendScore float64 `json:"trend_score"`
}

// ============================================
// ESTRUTURAS DE RESPOSTA DA API
// ============================================

// APIResponse resposta genérica da API
type APIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// RegisterResponse resposta do registro
type RegisterResponse struct {
	UserID    string `json:"user_id"`
	ShareCode string `json:"share_code"`
	AuthToken string `json:"auth_token"`
}

// UserLookupResponse resposta da busca de usuário
type UserLookupResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar,omitempty"`
	IsOnline bool   `json:"is_online"`
}

// ============================================
// SINGLETON
// ============================================

var (
	friendSystem     *FriendSystem
	friendSystemOnce sync.Once
)

// GetFriendSystem retorna a instância singleton do sistema de amigos
func GetFriendSystem() *FriendSystem {
	friendSystemOnce.Do(func() {
		configDir, _ := os.UserConfigDir()
		configPath := filepath.Join(configDir, "GoAnime", "social.json")

		friendSystem = &FriendSystem{
			apiBaseURL: APIBaseURL,
			configPath: configPath,
			friends:    []Friend{},
			httpClient: &http.Client{
				Timeout: HTTPTimeout,
			},
			heartbeatStop: make(chan struct{}),
		}

		// Carrega dados salvos
		friendSystem.loadFromDisk()

		// Inicia heartbeat se tiver perfil
		if friendSystem.profile != nil {
			go friendSystem.startHeartbeat()
		}
	})
	return friendSystem
}

// ============================================
// PERSISTÊNCIA LOCAL
// ============================================

// loadFromDisk carrega os dados do disco
func (fs *FriendSystem) loadFromDisk() {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	data, err := os.ReadFile(fs.configPath)
	if err != nil {
		return
	}

	var saved struct {
		Profile *UserProfile `json:"profile"`
		Friends []Friend     `json:"friends"`
	}

	if err := json.Unmarshal(data, &saved); err == nil {
		fs.profile = saved.Profile
		fs.friends = saved.Friends

		// Restaura o token de autenticação
		if fs.profile != nil && fs.profile.AuthToken != "" {
			fs.authToken = fs.profile.AuthToken
		}
	}
}

// saveToDisk salva os dados no disco
func (fs *FriendSystem) saveToDisk() error {
	// Garante que o diretório existe
	dir := filepath.Dir(fs.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	saved := struct {
		Profile *UserProfile `json:"profile"`
		Friends []Friend     `json:"friends"`
	}{
		Profile: fs.profile,
		Friends: fs.friends,
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.configPath, data, 0600)
}

// ============================================
// UTILITÁRIOS
// ============================================

// GenerateShareCode gera um código de compartilhamento único (8 chars hex)
func GenerateShareCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return strings.ToUpper(fmt.Sprintf("%X", b))
}

// GenerateUserID gera um ID único para o usuário
func GenerateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// makeAuthenticatedRequest faz uma requisição autenticada com o token
func (fs *FriendSystem) makeAuthenticatedRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Adiciona token de autenticação se disponível
	if fs.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+fs.authToken)
	}

	// Adiciona User-Agent para identificação
	req.Header.Set("User-Agent", "GoAnime-Desktop/1.0")

	return fs.httpClient.Do(req)
}

// ============================================
// GERENCIAMENTO DE PERFIL
// ============================================

// CreateProfile cria um novo perfil de usuário
func (fs *FriendSystem) CreateProfile(username string) (*UserProfile, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Primeiro tenta registrar no servidor
	profile, err := fs.registerOnServer(username)
	if err != nil {
		// Se falhar, cria perfil local
		fmt.Printf("[Social] Servidor offline, criando perfil local: %v\n", err)

		profile = &UserProfile{
			UserID:      GenerateUserID(),
			Username:    username,
			ShareCode:   GenerateShareCode(),
			CreatedAt:   time.Now(),
			ShowStatus:  true,
			ShareAnimes: true,
		}
	}

	fs.profile = profile
	fs.authToken = profile.AuthToken
	fs.saveToDisk()

	// Inicia heartbeat
	go fs.startHeartbeat()

	return fs.profile, nil
}

// registerOnServer registra o usuário no servidor PostgreSQL
func (fs *FriendSystem) registerOnServer(username string) (*UserProfile, error) {
	url := fs.apiBaseURL + "/register"

	payload := map[string]interface{}{
		"username":    username,
		"app_version": "1.0.0",
	}

	resp, err := fs.makeAuthenticatedRequest("POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("erro de conexão: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("servidor retornou %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			UserID    string `json:"user_id"`
			Username  string `json:"username"`
			ShareCode string `json:"share_code"`
			AuthToken string `json:"auth_token"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("%s", result.Error)
	}

	return &UserProfile{
		UserID:      result.Data.UserID,
		Username:    result.Data.Username,
		ShareCode:   result.Data.ShareCode,
		AuthToken:   result.Data.AuthToken,
		CreatedAt:   time.Now(),
		ShowStatus:  true,
		ShareAnimes: true,
		LastSync:    time.Now(),
	}, nil
}

// GetProfile retorna o perfil atual
func (fs *FriendSystem) GetProfile() *UserProfile {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.profile
}

// HasProfile verifica se existe um perfil
func (fs *FriendSystem) HasProfile() bool {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.profile != nil
}

// UpdateUsername atualiza o nome de usuário
func (fs *FriendSystem) UpdateUsername(username string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.profile == nil {
		return fmt.Errorf("nenhum perfil criado")
	}

	// Tenta atualizar no servidor
	go fs.syncUsernameToServer(username)

	fs.profile.Username = username
	fs.saveToDisk()
	return nil
}

func (fs *FriendSystem) syncUsernameToServer(username string) {
	if fs.profile == nil || fs.authToken == "" {
		return
	}

	url := fs.apiBaseURL + "/profile/update"
	payload := map[string]interface{}{
		"user_id":  fs.profile.UserID,
		"username": username,
	}

	fs.makeAuthenticatedRequest("PUT", url, payload)
}

// RegenerateShareCode gera um novo código de compartilhamento
func (fs *FriendSystem) RegenerateShareCode() (string, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.profile == nil {
		return "", fmt.Errorf("nenhum perfil criado")
	}

	newCode := GenerateShareCode()

	// Tenta atualizar no servidor
	if fs.authToken != "" {
		url := fs.apiBaseURL + "/profile/regenerate-code"
		payload := map[string]interface{}{
			"user_id": fs.profile.UserID,
		}

		resp, err := fs.makeAuthenticatedRequest("POST", url, payload)
		if err == nil {
			defer resp.Body.Close()

			var result struct {
				Success bool `json:"success"`
				Data    struct {
					ShareCode string `json:"share_code"`
				} `json:"data"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Success {
				newCode = result.Data.ShareCode
			}
		}
	}

	fs.profile.ShareCode = newCode
	fs.saveToDisk()
	return fs.profile.ShareCode, nil
}

// ============================================
// GERENCIAMENTO DE AMIGOS
// ============================================

// AddFriendByCode adiciona um amigo pelo código de compartilhamento
func (fs *FriendSystem) AddFriendByCode(shareCode string) (*Friend, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.profile == nil {
		return nil, fmt.Errorf("crie um perfil primeiro")
	}

	// Normaliza o código (uppercase, remove espaços)
	shareCode = strings.ToUpper(strings.TrimSpace(shareCode))

	// Verifica se não está adicionando a si mesmo
	if fs.profile.ShareCode == shareCode {
		return nil, fmt.Errorf("você não pode adicionar a si mesmo")
	}

	// Verifica se já é amigo
	for _, f := range fs.friends {
		if f.ShareCode == shareCode || f.UserID == shareCode {
			return nil, fmt.Errorf("este usuário já é seu amigo")
		}
	}

	// Busca o usuário no servidor
	friend, err := fs.lookupUserByCode(shareCode)
	if err != nil {
		return nil, fmt.Errorf("usuário não encontrado: %w", err)
	}

	// Notifica o servidor sobre a nova amizade
	go fs.notifyFriendshipCreated(friend.UserID)

	fs.friends = append(fs.friends, *friend)
	fs.saveToDisk()

	return friend, nil
}

// lookupUserByCode busca um usuário pelo código no servidor
func (fs *FriendSystem) lookupUserByCode(shareCode string) (*Friend, error) {
	url := fmt.Sprintf("%s/user/lookup?code=%s", fs.apiBaseURL, shareCode)

	resp, err := fs.makeAuthenticatedRequest("GET", url, nil)
	if err != nil {
		// Se servidor offline, cria amigo local com código
		fmt.Printf("[Social] Servidor offline, adicionando amigo local\n")
		return &Friend{
			UserID:    shareCode,
			Username:  fmt.Sprintf("Amigo-%s", shareCode[:4]),
			ShareCode: shareCode,
			AddedAt:   time.Now(),
			IsOnline:  false,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("código inválido ou usuário não existe")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro do servidor: %s", string(body))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			Avatar   string `json:"avatar"`
			IsOnline bool   `json:"is_online"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("resposta inválida: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("usuário não encontrado")
	}

	return &Friend{
		UserID:    result.Data.UserID,
		Username:  result.Data.Username,
		Avatar:    result.Data.Avatar,
		ShareCode: shareCode,
		AddedAt:   time.Now(),
		IsOnline:  result.Data.IsOnline,
	}, nil
}

// notifyFriendshipCreated notifica o servidor sobre nova amizade
func (fs *FriendSystem) notifyFriendshipCreated(friendUserID string) {
	if fs.profile == nil || fs.authToken == "" {
		return
	}

	url := fs.apiBaseURL + "/friends/add"
	payload := map[string]interface{}{
		"user_id":   fs.profile.UserID,
		"friend_id": friendUserID,
	}

	fs.makeAuthenticatedRequest("POST", url, payload)
}

// RemoveFriend remove um amigo
func (fs *FriendSystem) RemoveFriend(userID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	found := false
	for i, f := range fs.friends {
		if f.UserID == userID {
			fs.friends = append(fs.friends[:i], fs.friends[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("amigo não encontrado")
	}

	// Notifica servidor
	go fs.notifyFriendshipRemoved(userID)

	fs.saveToDisk()
	return nil
}

func (fs *FriendSystem) notifyFriendshipRemoved(friendUserID string) {
	if fs.profile == nil || fs.authToken == "" {
		return
	}

	url := fs.apiBaseURL + "/friends/remove"
	payload := map[string]interface{}{
		"user_id":   fs.profile.UserID,
		"friend_id": friendUserID,
	}

	fs.makeAuthenticatedRequest("DELETE", url, payload)
}

// GetFriends retorna a lista de amigos
func (fs *FriendSystem) GetFriends() []Friend {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	// Faz cópia para evitar race conditions
	friends := make([]Friend, len(fs.friends))
	copy(friends, fs.friends)
	return friends
}

// GetFriendsActivity retorna a atividade dos amigos
func (fs *FriendSystem) GetFriendsActivity() ([]FriendActivity, error) {
	fs.mutex.RLock()
	profile := fs.profile
	friends := make([]Friend, len(fs.friends))
	copy(friends, fs.friends)
	fs.mutex.RUnlock()

	if profile == nil {
		return nil, fmt.Errorf("nenhum perfil criado")
	}

	// Tenta buscar atividade do servidor
	activities, err := fs.fetchFriendsActivityFromServer()
	if err != nil {
		fmt.Printf("[Social] Usando dados locais: %v\n", err)
		// Retorna dados locais se servidor falhar
		return fs.buildLocalActivity(friends), nil
	}

	// Atualiza dados locais com informações do servidor
	fs.updateLocalFriendsFromActivity(activities)

	return activities, nil
}

// fetchFriendsActivityFromServer busca atividade do servidor
func (fs *FriendSystem) fetchFriendsActivityFromServer() ([]FriendActivity, error) {
	if fs.profile == nil || fs.authToken == "" {
		return nil, fmt.Errorf("não autenticado")
	}

	// Monta lista de IDs dos amigos
	friendIDs := make([]string, len(fs.friends))
	for i, f := range fs.friends {
		friendIDs[i] = f.UserID
	}

	url := fs.apiBaseURL + "/friends/activity"
	payload := map[string]interface{}{
		"user_id":    fs.profile.UserID,
		"friend_ids": friendIDs,
	}

	resp, err := fs.makeAuthenticatedRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("servidor retornou %d", resp.StatusCode)
	}

	var result struct {
		Success bool             `json:"success"`
		Data    []FriendActivity `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// buildLocalActivity constrói atividade a partir de dados locais
func (fs *FriendSystem) buildLocalActivity(friends []Friend) []FriendActivity {
	activities := make([]FriendActivity, len(friends))
	for i, f := range friends {
		activities[i] = FriendActivity{
			UserID:       f.UserID,
			Username:     f.Username,
			Avatar:       f.Avatar,
			AnimeTitle:   f.CurrentAnime,
			EpisodeNum:   f.CurrentEp,
			IsWatching:   f.CurrentAnime != "",
			IsOnline:     f.IsOnline,
			LastActivity: formatTimeAgo(f.LastSeen),
		}
	}
	return activities
}

// updateLocalFriendsFromActivity atualiza dados locais com info do servidor
func (fs *FriendSystem) updateLocalFriendsFromActivity(activities []FriendActivity) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	activityMap := make(map[string]FriendActivity)
	for _, a := range activities {
		activityMap[a.UserID] = a
	}

	for i := range fs.friends {
		if activity, ok := activityMap[fs.friends[i].UserID]; ok {
			fs.friends[i].IsOnline = activity.IsOnline
			fs.friends[i].CurrentAnime = activity.AnimeTitle
			fs.friends[i].CurrentEp = activity.EpisodeNum
			if activity.IsOnline {
				fs.friends[i].LastSeen = time.Now()
			}
		}
	}

	fs.saveToDisk()
}

// ============================================
// STATUS DE VISUALIZAÇÃO
// ============================================

// UpdateWatchingStatus atualiza o status de visualização
func (fs *FriendSystem) UpdateWatchingStatus(status *WatchingStatus) error {
	fs.mutex.RLock()
	profile := fs.profile
	showStatus := profile != nil && profile.ShowStatus
	fs.mutex.RUnlock()

	if !showStatus {
		return nil
	}

	// Envia para a API em background
	go func() {
		url := fs.apiBaseURL + "/status/update"

		payload := map[string]interface{}{
			"user_id":        profile.UserID,
			"anime_title":    status.AnimeTitle,
			"anime_image":    status.AnimeImage,
			"episode_num":    status.EpisodeNum,
			"total_episodes": status.TotalEpisodes,
			"timestamp":      time.Now().Unix(),
		}

		fs.makeAuthenticatedRequest("POST", url, payload)
	}()

	return nil
}

// ClearWatchingStatus limpa o status de visualização
func (fs *FriendSystem) ClearWatchingStatus() {
	fs.mutex.RLock()
	profile := fs.profile
	fs.mutex.RUnlock()

	if profile == nil {
		return
	}

	go func() {
		url := fs.apiBaseURL + "/status/clear"
		payload := map[string]interface{}{
			"user_id": profile.UserID,
		}
		fs.makeAuthenticatedRequest("POST", url, payload)
	}()
}

// ============================================
// CONFIGURAÇÕES
// ============================================

// SetShowStatus define se mostra o status
func (fs *FriendSystem) SetShowStatus(show bool) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.profile != nil {
		fs.profile.ShowStatus = show
		fs.saveToDisk()

		// Sincroniza com servidor
		go fs.syncSettingsToServer()
	}
}

// SetShareAnimes define se compartilha animes
func (fs *FriendSystem) SetShareAnimes(share bool) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.profile != nil {
		fs.profile.ShareAnimes = share
		fs.saveToDisk()

		// Sincroniza com servidor
		go fs.syncSettingsToServer()
	}
}

func (fs *FriendSystem) syncSettingsToServer() {
	if fs.profile == nil || fs.authToken == "" {
		return
	}

	url := fs.apiBaseURL + "/profile/settings"
	payload := map[string]interface{}{
		"user_id":      fs.profile.UserID,
		"show_status":  fs.profile.ShowStatus,
		"share_animes": fs.profile.ShareAnimes,
	}

	fs.makeAuthenticatedRequest("PUT", url, payload)
}

// IncrementWatched incrementa o contador de episódios assistidos
func (fs *FriendSystem) IncrementWatched() {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.profile != nil {
		fs.profile.TotalWatched++
		fs.saveToDisk()
	}
}

// ============================================
// HEARTBEAT E ONLINE STATUS
// ============================================

// startHeartbeat inicia o heartbeat para manter status online
func (fs *FriendSystem) startHeartbeat() {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	// Envia heartbeat inicial
	fs.sendHeartbeat()

	for {
		select {
		case <-ticker.C:
			fs.sendHeartbeat()
		case <-fs.heartbeatStop:
			return
		}
	}
}

// sendHeartbeat envia sinal de que o usuário está online
func (fs *FriendSystem) sendHeartbeat() {
	fs.mutex.RLock()
	profile := fs.profile
	fs.mutex.RUnlock()

	if profile == nil || fs.authToken == "" {
		return
	}

	url := fs.apiBaseURL + "/heartbeat"
	payload := map[string]interface{}{
		"user_id":   profile.UserID,
		"timestamp": time.Now().Unix(),
	}

	fs.makeAuthenticatedRequest("POST", url, payload)
}

// StopHeartbeat para o heartbeat
func (fs *FriendSystem) StopHeartbeat() {
	select {
	case fs.heartbeatStop <- struct{}{}:
	default:
	}
}

// ============================================
// DELETAR PERFIL
// ============================================

// DeleteProfile apaga o perfil e desconecta
func (fs *FriendSystem) DeleteProfile() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Notifica servidor sobre deleção
	if fs.profile != nil && fs.authToken != "" {
		url := fs.apiBaseURL + "/profile/delete"
		payload := map[string]interface{}{
			"user_id": fs.profile.UserID,
		}
		fs.makeAuthenticatedRequest("DELETE", url, payload)
	}

	// Para heartbeat
	fs.StopHeartbeat()

	fs.profile = nil
	fs.authToken = ""
	fs.friends = []Friend{}

	// Remove arquivo de configuração
	os.Remove(fs.configPath)

	return nil
}

// ============================================
// SINCRONIZAÇÃO
// ============================================

// SyncWithServer sincroniza dados com o servidor
func (fs *FriendSystem) SyncWithServer() error {
	fs.mutex.RLock()
	profile := fs.profile
	fs.mutex.RUnlock()

	if profile == nil {
		return fmt.Errorf("nenhum perfil criado")
	}

	// Busca lista de amigos do servidor
	url := fs.apiBaseURL + "/friends/list"
	payload := map[string]interface{}{
		"user_id": profile.UserID,
	}

	resp, err := fs.makeAuthenticatedRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool     `json:"success"`
		Data    []Friend `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Success {
		fs.mutex.Lock()
		fs.friends = result.Data
		fs.profile.LastSync = time.Now()
		fs.saveToDisk()
		fs.mutex.Unlock()
	}

	return nil
}

// GetConnectionStatus retorna status da conexão com o servidor
func (fs *FriendSystem) GetConnectionStatus() (bool, string) {
	// Primeiro testa o endpoint social específico
	socialURL := fs.apiBaseURL + "/health"
	resp, err := fs.httpClient.Get(socialURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fs.mutex.Lock()
			fs.isOnline = true
			fs.mutex.Unlock()
			return true, "Conectado ao servidor social"
		}
	}

	// Se o social não está disponível, verifica se a VPS principal está online
	vpsURL := "http://[2804:54:c100:2::11]:8080/health"
	resp2, err2 := fs.httpClient.Get(vpsURL)
	if err2 == nil {
		defer resp2.Body.Close()
		if resp2.StatusCode == http.StatusOK {
			fs.mutex.Lock()
			fs.isOnline = true // VPS está online, modo híbrido
			fs.mutex.Unlock()
			return true, "VPS online (social local)"
		}
	}

	fs.mutex.Lock()
	fs.isOnline = false
	fs.mutex.Unlock()

	// Totalmente offline
	return false, "Modo offline"
}

// IsServerOnline verifica se está conectado ao servidor
func (fs *FriendSystem) IsServerOnline() bool {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.isOnline
}

// ============================================
// UTILITÁRIOS
// ============================================

// formatTimeAgo formata tempo relativo
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "nunca"
	}

	diff := time.Since(t)

	if diff < time.Minute {
		return "agora"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d min atrás", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh atrás", hours)
	}

	days := int(diff.Hours() / 24)
	return fmt.Sprintf("%dd atrás", days)
}

// ============================================
// SISTEMA DE RECOMENDAÇÕES
// ============================================

// GetRecommendations busca recomendações personalizadas baseadas no histórico do usuário
func (fs *FriendSystem) GetRecommendations(limit int) ([]AnimeRecommendation, error) {
	fs.mutex.RLock()
	if fs.profile == nil {
		fs.mutex.RUnlock()
		return nil, fmt.Errorf("perfil não encontrado")
	}
	fs.mutex.RUnlock()

	url := fmt.Sprintf("%s/recommendations?limit=%d", fs.apiBaseURL, limit)

	resp, err := fs.makeAuthenticatedRequest("GET", url, nil)
	if err != nil {
		return fs.getLocalRecommendations(limit), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fs.getLocalRecommendations(limit), nil
	}

	var result struct {
		Success bool                  `json:"success"`
		Data    []AnimeRecommendation `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fs.getLocalRecommendations(limit), nil
	}

	if result.Success && len(result.Data) > 0 {
		return result.Data, nil
	}

	return fs.getLocalRecommendations(limit), nil
}

// GetTrendingAnimes busca animes em alta na comunidade
func (fs *FriendSystem) GetTrendingAnimes(limit int) ([]TrendingAnime, error) {
	url := fmt.Sprintf("%s/trending?limit=%d", fs.apiBaseURL, limit)

	resp, err := fs.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar trending: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("servidor retornou %d", resp.StatusCode)
	}

	var result struct {
		Success bool            `json:"success"`
		Data    []TrendingAnime `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetFriendsWatching busca o que amigos estão assistindo para recomendações
func (fs *FriendSystem) GetFriendsWatching() ([]AnimeRecommendation, error) {
	fs.mutex.RLock()
	if fs.profile == nil {
		fs.mutex.RUnlock()
		return nil, fmt.Errorf("perfil não encontrado")
	}
	fs.mutex.RUnlock()

	url := fs.apiBaseURL + "/friends/watching"

	resp, err := fs.makeAuthenticatedRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool                  `json:"success"`
		Data    []AnimeRecommendation `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ReportWatching reporta para o servidor o que está assistindo (para estatísticas)
func (fs *FriendSystem) ReportWatching(animeID, title, image string, episode int) error {
	fs.mutex.RLock()
	if fs.profile == nil {
		fs.mutex.RUnlock()
		return nil // Silent fail se não tem perfil
	}
	fs.mutex.RUnlock()

	url := fs.apiBaseURL + "/stats/watching"

	payload := map[string]interface{}{
		"anime_id": animeID,
		"title":    title,
		"image":    image,
		"episode":  episode,
	}

	resp, err := fs.makeAuthenticatedRequest("POST", url, payload)
	if err != nil {
		return nil // Silent fail
	}
	defer resp.Body.Close()

	return nil
}

// getLocalRecommendations retorna recomendações locais padrão quando offline
func (fs *FriendSystem) getLocalRecommendations(limit int) []AnimeRecommendation {
	// Recomendações populares padrão quando offline
	defaults := []AnimeRecommendation{
		{
			AnimeID:       "solo-leveling",
			Title:         "Solo Leveling",
			Image:         "https://cdn.myanimelist.net/images/anime/1376/121828.jpg",
			Rating:        8.8,
			WatchCount:    15000,
			Trending:      true,
			RecommendedBy: "Popular na comunidade",
		},
		{
			AnimeID:       "demon-slayer",
			Title:         "Demon Slayer",
			Image:         "https://cdn.myanimelist.net/images/anime/1286/99889.jpg",
			Rating:        8.5,
			WatchCount:    25000,
			Trending:      true,
			RecommendedBy: "Mais assistido",
		},
		{
			AnimeID:       "jujutsu-kaisen",
			Title:         "Jujutsu Kaisen",
			Image:         "https://cdn.myanimelist.net/images/anime/1171/109222.jpg",
			Rating:        8.7,
			WatchCount:    20000,
			Trending:      true,
			RecommendedBy: "Trending",
		},
		{
			AnimeID:       "one-piece",
			Title:         "One Piece",
			Image:         "https://cdn.myanimelist.net/images/anime/1244/138851.jpg",
			Rating:        8.9,
			WatchCount:    50000,
			Trending:      false,
			RecommendedBy: "Clássico atemporal",
		},
		{
			AnimeID:       "attack-on-titan",
			Title:         "Attack on Titan",
			Image:         "https://cdn.myanimelist.net/images/anime/1000/110531.jpg",
			Rating:        9.0,
			WatchCount:    45000,
			Trending:      false,
			RecommendedBy: "Obra-prima",
		},
	}

	if limit > len(defaults) {
		limit = len(defaults)
	}

	return defaults[:limit]
}
