package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ============================================
// CONFIGURAÇÕES
// ============================================

const (
	// API Base URL - VPS
	APIBaseURL  = "http://[2804:54:c100:2::11]:8080/api/auth"
	HTTPTimeout = 10 * time.Second
)

// ============================================
// ESTRUTURAS
// ============================================

// AuthManager gerencia autenticação
type AuthManager struct {
	mu         sync.RWMutex
	session    *UserSession
	configPath string
	httpClient *http.Client
	isGuest    bool
	listeners  []func(bool) // Callbacks para mudança de estado
}

// UserSession sessão do usuário
type UserSession struct {
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	Email          string    `json:"email,omitempty"`
	Avatar         string    `json:"avatar"`
	Token          string    `json:"token"`
	IsVIP          bool      `json:"is_vip"`
	IsPremium      bool      `json:"is_premium"`
	FriendToken    string    `json:"friend_token"`
	CreatedAt      time.Time `json:"created_at"`
	LastLogin      time.Time `json:"last_login"`
	SeedingEnabled bool      `json:"seeding_enabled"`
	SeedingBytes   int64     `json:"seeding_bytes"`
}

// LoginRequest requisição de login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest requisição de registro
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Avatar   string `json:"avatar,omitempty"`
}

// AuthResponse resposta da API
type AuthResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Error   string       `json:"error,omitempty"`
	User    *UserSession `json:"user,omitempty"`
	Token   string       `json:"token,omitempty"`
}

// GuestSession sessão de convidado (local apenas)
type GuestSession struct {
	GuestID   string    `json:"guest_id"`
	CreatedAt time.Time `json:"created_at"`
	Favorites []string  `json:"favorites"` // Só guarda URLs localmente
}

// ============================================
// INICIALIZAÇÃO
// ============================================

var defaultManager *AuthManager
var initOnce sync.Once

// GetManager retorna o gerenciador singleton
func GetManager() *AuthManager {
	initOnce.Do(func() {
		defaultManager = NewAuthManager()
	})
	return defaultManager
}

// NewAuthManager cria novo gerenciador de autenticação
func NewAuthManager() *AuthManager {
	configDir, _ := os.UserConfigDir()
	configPath := filepath.Join(configDir, "GoAnime", "auth.json")

	// Criar diretório se não existir
	os.MkdirAll(filepath.Dir(configPath), 0755)

	manager := &AuthManager{
		configPath: configPath,
		httpClient: &http.Client{Timeout: HTTPTimeout},
		listeners:  []func(bool){},
	}

	// Carregar sessão salva
	manager.loadSession()

	return manager
}

// ============================================
// MÉTODOS PÚBLICOS
// ============================================

// Register registra novo usuário
func (m *AuthManager) Register(username, email, password, avatar string) (*UserSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if username == "" || password == "" {
		return nil, errors.New("username e password são obrigatórios")
	}

	if len(password) < 6 {
		return nil, errors.New("password deve ter pelo menos 6 caracteres")
	}

	req := RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
		Avatar:   avatar,
	}

	body, _ := json.Marshal(req)
	resp, err := m.httpClient.Post(APIBaseURL+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erro de conexão: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var authResp AuthResponse
	if err := json.Unmarshal(data, &authResp); err != nil {
		return nil, fmt.Errorf("resposta inválida: %w", err)
	}

	if !authResp.Success {
		errMsg := authResp.Error
		if errMsg == "" {
			errMsg = authResp.Message
		}
		return nil, errors.New(errMsg)
	}

	m.session = authResp.User
	if m.session != nil {
		m.session.Token = authResp.Token
		m.session.LastLogin = time.Now()
	}
	m.isGuest = false
	m.saveSession()
	m.notifyListeners(true)

	return m.session, nil
}

// Login faz login com credenciais
func (m *AuthManager) Login(username, password string) (*UserSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if username == "" || password == "" {
		return nil, errors.New("username e password são obrigatórios")
	}

	req := LoginRequest{
		Username: username,
		Password: password,
	}

	body, _ := json.Marshal(req)
	resp, err := m.httpClient.Post(APIBaseURL+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erro de conexão: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var authResp AuthResponse
	if err := json.Unmarshal(data, &authResp); err != nil {
		return nil, fmt.Errorf("resposta inválida: %w", err)
	}

	if !authResp.Success {
		errMsg := authResp.Error
		if errMsg == "" {
			errMsg = authResp.Message
		}
		return nil, errors.New(errMsg)
	}

	m.session = authResp.User
	if m.session != nil {
		m.session.Token = authResp.Token
		m.session.LastLogin = time.Now()
	}
	m.isGuest = false
	m.saveSession()
	m.notifyListeners(true)

	return m.session, nil
}

// LoginAsGuest entra como convidado
func (m *AuthManager) LoginAsGuest() *UserSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	guestID := generateGuestID()
	m.session = &UserSession{
		UserID:    guestID,
		Username:  "Visitante",
		Avatar:    "guest.png",
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}
	m.isGuest = true
	m.notifyListeners(true)

	return m.session
}

// Logout faz logout
func (m *AuthManager) Logout() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.session = nil
	m.isGuest = false
	m.deleteSession()
	m.notifyListeners(false)
}

// GetSession retorna sessão atual
func (m *AuthManager) GetSession() *UserSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.session
}

// IsLoggedIn verifica se está logado
func (m *AuthManager) IsLoggedIn() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.session != nil && !m.isGuest
}

// IsGuest verifica se é convidado
func (m *AuthManager) IsGuest() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isGuest
}

// HasSession verifica se tem alguma sessão (logado ou guest)
func (m *AuthManager) HasSession() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.session != nil
}

// UpdateSeedingPreference atualiza preferência de seeding
func (m *AuthManager) UpdateSeedingPreference(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session == nil || m.isGuest {
		return errors.New("usuário não autenticado")
	}

	// Atualiza no servidor
	req, _ := http.NewRequest("POST", APIBaseURL+"/seeding", bytes.NewReader([]byte(fmt.Sprintf(`{"enabled":%v}`, enabled))))
	req.Header.Set("Authorization", "Bearer "+m.session.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	m.session.SeedingEnabled = enabled
	m.saveSession()
	return nil
}

// AddListener adiciona callback para mudança de estado
func (m *AuthManager) AddListener(fn func(bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, fn)
}

// ============================================
// MÉTODOS PRIVADOS
// ============================================

func (m *AuthManager) notifyListeners(loggedIn bool) {
	for _, fn := range m.listeners {
		go fn(loggedIn)
	}
}

func (m *AuthManager) loadSession() {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return
	}

	var session UserSession
	if err := json.Unmarshal(data, &session); err != nil {
		return
	}

	// Valida token com o servidor
	if session.Token != "" {
		req, _ := http.NewRequest("GET", APIBaseURL+"/validate", nil)
		req.Header.Set("Authorization", "Bearer "+session.Token)

		resp, err := m.httpClient.Do(req)
		if err == nil && resp.StatusCode == 200 {
			m.session = &session
			m.isGuest = false
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Token inválido, limpa sessão
	m.deleteSession()
}

func (m *AuthManager) saveSession() {
	if m.session == nil || m.isGuest {
		return
	}

	data, _ := json.MarshalIndent(m.session, "", "  ")
	os.WriteFile(m.configPath, data, 0600)
}

func (m *AuthManager) deleteSession() {
	os.Remove(m.configPath)
}

func generateGuestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "guest_" + hex.EncodeToString(b)
}
