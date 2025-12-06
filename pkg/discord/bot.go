package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DiscordBot gerencia a integração com Discord
type DiscordBot struct {
	Token           string
	GuildID         string // ID do servidor Discord
	ChannelID       string // ID do canal de recomendações
	WebhookURL      string // Webhook para enviar mensagens
	connected       bool
	currentUser     *DiscordUser // Usuário conectado
	recommendations []Recommendation
	mutex           sync.RWMutex
}

// Recommendation representa uma recomendação de anime
type Recommendation struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	UserAvatar string    `json:"user_avatar"`
	AnimeTitle string    `json:"anime_title"`
	AnimeImage string    `json:"anime_image"`
	AnimeScore float64   `json:"anime_score"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	Likes      int       `json:"likes"`
	LikedByMe  bool      `json:"liked_by_me"`
}

// DiscordUser representa um usuário do Discord
type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	GlobalName    string `json:"global_name"`
	AvatarURL     string `json:"avatar_url"` // URL completa do avatar
}

// WebhookMessage estrutura da mensagem do webhook
type WebhookMessage struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// Embed para mensagens ricas do Discord
type Embed struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	URL         string     `json:"url,omitempty"`
	Color       int        `json:"color,omitempty"`
	Thumbnail   *Thumbnail `json:"thumbnail,omitempty"`
	Image       *Image     `json:"image,omitempty"`
	Author      *Author    `json:"author,omitempty"`
	Fields      []Field    `json:"fields,omitempty"`
	Footer      *Footer    `json:"footer,omitempty"`
	Timestamp   string     `json:"timestamp,omitempty"`
}

type Thumbnail struct {
	URL string `json:"url"`
}

type Image struct {
	URL string `json:"url"`
}

type Author struct {
	Name    string `json:"name"`
	IconURL string `json:"icon_url,omitempty"`
}

type Footer struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Webhook padrão do servidor GoAnime Community
const DefaultWebhookURL = "https://discord.com/api/webhooks/1446599728237252620/YEoU9oLqGUgs0-A0_L3eJMfDf1BBrpNj22xba5jFevHTYGER9LZoZfKIZpXvL1Q8KhLm"

var (
	botInstance *DiscordBot
	botOnce     sync.Once
)

// GetBot retorna a instância singleton do bot
func GetBot() *DiscordBot {
	botOnce.Do(func() {
		botInstance = &DiscordBot{
			WebhookURL:      DefaultWebhookURL,
			connected:       true, // Já conectado por padrão
			recommendations: make([]Recommendation, 0),
		}
	})
	return botInstance
}

// Configure configura o bot com as credenciais
func (b *DiscordBot) Configure(webhookURL string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.WebhookURL = webhookURL
	b.connected = webhookURL != ""
}

// IsConnected verifica se o bot está configurado
func (b *DiscordBot) IsConnected() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.connected
}

// SendRecommendation envia uma recomendação para o Discord via webhook
func (b *DiscordBot) SendRecommendation(rec Recommendation) error {
	if !b.IsConnected() {
		return fmt.Errorf("bot não configurado")
	}

	// Cor rosa/magenta do GoAnime
	embedColor := 0xF5576C

	// Monta a mensagem com embed rico
	msg := WebhookMessage{
		Username:  "GoAnime Bot",
		AvatarURL: "https://raw.githubusercontent.com/alvarorichard/GoAnime/main/assets/logo.png",
		Content:   fmt.Sprintf("🎬 **%s** recomendou um anime!", rec.Username),
		Embeds: []Embed{
			{
				Title:       fmt.Sprintf("📺 %s", rec.AnimeTitle),
				Description: rec.Message,
				Color:       embedColor,
				Thumbnail: &Thumbnail{
					URL: rec.AnimeImage,
				},
				Author: &Author{
					Name:    rec.Username,
					IconURL: rec.UserAvatar,
				},
				Fields: []Field{
					{
						Name:   "⭐ Score",
						Value:  fmt.Sprintf("%.1f/10", rec.AnimeScore),
						Inline: true,
					},
					{
						Name:   "🎯 Assistir",
						Value:  "Abra o GoAnime e busque!",
						Inline: true,
					},
				},
				Footer: &Footer{
					Text:    "GoAnime • Recomendação de Amigo",
					IconURL: "https://raw.githubusercontent.com/alvarorichard/GoAnime/main/assets/logo.png",
				},
				Timestamp: rec.Timestamp.Format(time.RFC3339),
			},
		},
	}

	// Envia via webhook
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	resp, err := http.Post(b.WebhookURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao enviar webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("webhook retornou status %d", resp.StatusCode)
	}

	// Salva localmente também
	b.addRecommendation(rec)

	return nil
}

// SendWebhookMessage envia uma mensagem genérica via webhook
func (b *DiscordBot) SendWebhookMessage(msg WebhookMessage) error {
	if !b.IsConnected() {
		return fmt.Errorf("bot não configurado")
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	resp, err := http.Post(b.WebhookURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao enviar webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("webhook retornou status %d", resp.StatusCode)
	}

	return nil
}

// addRecommendation adiciona uma recomendação à lista local
func (b *DiscordBot) addRecommendation(rec Recommendation) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Adiciona no início (mais recente primeiro)
	b.recommendations = append([]Recommendation{rec}, b.recommendations...)

	// Mantém apenas as últimas 50 recomendações
	if len(b.recommendations) > 50 {
		b.recommendations = b.recommendations[:50]
	}
}

// GetRecommendations retorna as recomendações
func (b *DiscordBot) GetRecommendations() []Recommendation {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Retorna uma cópia
	result := make([]Recommendation, len(b.recommendations))
	copy(result, b.recommendations)
	return result
}

// AddMockRecommendations adiciona recomendações de exemplo (para demonstração)
func (b *DiscordBot) AddMockRecommendations() {
	mockRecs := []Recommendation{
		{
			ID:         "1",
			UserID:     "123456789",
			Username:   "AkiraFan42",
			UserAvatar: "https://cdn.discordapp.com/embed/avatars/1.png",
			AnimeTitle: "Frieren: Beyond Journey's End",
			AnimeImage: "https://cdn.myanimelist.net/images/anime/1015/138006l.jpg",
			AnimeScore: 9.2,
			Message:    "Melhor anime do ano! A história é incrível e os personagens são muito bem desenvolvidos 🔥",
			Timestamp:  time.Now().Add(-1 * time.Hour),
			Likes:      15,
		},
		{
			ID:         "2",
			UserID:     "987654321",
			Username:   "OtakuMaster",
			UserAvatar: "https://cdn.discordapp.com/embed/avatars/2.png",
			AnimeTitle: "Solo Leveling",
			AnimeImage: "https://cdn.myanimelist.net/images/anime/1381/141108l.jpg",
			AnimeScore: 8.7,
			Message:    "A animação tá PERFEITA! O estúdio caprichou demais, vocês precisam assistir!",
			Timestamp:  time.Now().Add(-3 * time.Hour),
			Likes:      23,
		},
		{
			ID:         "3",
			UserID:     "456789123",
			Username:   "SakuraChan",
			UserAvatar: "https://cdn.discordapp.com/embed/avatars/3.png",
			AnimeTitle: "Bocchi the Rock!",
			AnimeImage: "https://cdn.myanimelist.net/images/anime/1448/127956l.jpg",
			AnimeScore: 8.9,
			Message:    "Muito fofo e engraçado! A Bocchi é muito relatable 💕 Recomendo demais!",
			Timestamp:  time.Now().Add(-24 * time.Hour),
			Likes:      31,
		},
		{
			ID:         "4",
			UserID:     "789123456",
			Username:   "NarutoRunner",
			UserAvatar: "https://cdn.discordapp.com/embed/avatars/4.png",
			AnimeTitle: "Jujutsu Kaisen",
			AnimeImage: "https://cdn.myanimelist.net/images/anime/1171/109222l.jpg",
			AnimeScore: 8.6,
			Message:    "As lutas são insanas! Gojo é o melhor personagem 😎",
			Timestamp:  time.Now().Add(-48 * time.Hour),
			Likes:      42,
		},
		{
			ID:         "5",
			UserID:     "321654987",
			Username:   "AnimeLover99",
			UserAvatar: "https://cdn.discordapp.com/embed/avatars/5.png",
			AnimeTitle: "Spy x Family",
			AnimeImage: "https://cdn.myanimelist.net/images/anime/1441/122795l.jpg",
			AnimeScore: 8.8,
			Message:    "Família mais fofa do anime! Anya é muito engraçada 🥜",
			Timestamp:  time.Now().Add(-72 * time.Hour),
			Likes:      56,
		},
	}

	for _, rec := range mockRecs {
		b.addRecommendation(rec)
	}
}

// LikeRecommendation adiciona um like a uma recomendação
func (b *DiscordBot) LikeRecommendation(recID string) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for i := range b.recommendations {
		if b.recommendations[i].ID == recID {
			if !b.recommendations[i].LikedByMe {
				b.recommendations[i].Likes++
				b.recommendations[i].LikedByMe = true
				return true
			}
			return false
		}
	}
	return false
}

// GetCurrentUser retorna o usuário Discord conectado
func (b *DiscordBot) GetCurrentUser() *DiscordUser {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.currentUser
}

// SetCurrentUser define o usuário conectado via OAuth2
func (b *DiscordBot) SetCurrentUser(user *DiscordUser) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.currentUser = user
	b.connected = user != nil
}

// Disconnect desconecta o usuário atual
func (b *DiscordBot) Disconnect() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.currentUser = nil
	// Mantém connected = true para o webhook continuar funcionando
	b.connected = true
}

// FetchUserFromToken busca as informações do usuário usando o access token
func (b *DiscordBot) FetchUserFromToken(accessToken string) (*DiscordUser, error) {
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord retornou status %d: %s", resp.StatusCode, string(body))
	}

	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	// Constrói a URL do avatar
	if user.Avatar != "" {
		user.AvatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=128", user.ID, user.Avatar)
	} else {
		// Avatar padrão do Discord
		defaultIdx := 0
		if user.Discriminator != "0" && user.Discriminator != "" {
			// Converte discriminator para int para calcular índice
			if d, err := strconv.Atoi(user.Discriminator); err == nil {
				defaultIdx = d % 5
			}
		} else {
			// Novo sistema de username (sem discriminator)
			// Usa hash do ID
			if id, err := strconv.ParseInt(user.ID, 10, 64); err == nil {
				defaultIdx = int((id >> 22) % 6)
			}
		}
		user.AvatarURL = fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", defaultIdx)
	}

	return &user, nil
}

// OAuth2CallbackServer inicia um servidor HTTP temporário para capturar o callback do OAuth2
func (b *DiscordBot) OAuth2CallbackServer(clientID, clientSecret, redirectURI string) (string, error) {
	// Canal para receber o código de autorização
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Extrai a porta do redirectURI
	port := ":9876" // porta padrão
	if strings.Contains(redirectURI, "localhost:") {
		parts := strings.Split(redirectURI, "localhost:")
		if len(parts) > 1 {
			portPath := strings.Split(parts[1], "/")
			port = ":" + portPath[0]
		}
	}

	// Servidor HTTP temporário
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	fmt.Printf("[Discord OAuth] Servidor de callback iniciado na porta %s\n", port)

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errStr := r.URL.Query().Get("error")
			errDesc := r.URL.Query().Get("error_description")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Erro</title><style>
				body{font-family:Arial,sans-serif;background:#1a1a2e;color:#fff;display:flex;justify-content:center;align-items:center;height:100vh;margin:0}
				.container{text-align:center;padding:40px;background:#16213e;border-radius:12px;box-shadow:0 4px 20px rgba(0,0,0,0.3)}
				h1{color:#F5576C}
			</style></head><body><div class="container"><h1>❌ Erro na Conexão</h1><p>%s: %s</p><p>Você pode fechar esta janela.</p></div></body></html>`, errStr, errDesc)
			errChan <- fmt.Errorf("%s: %s", errStr, errDesc)
			return
		}

		// Página de sucesso
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Conectado!</title><style>
			body{font-family:Arial,sans-serif;background:#1a1a2e;color:#fff;display:flex;justify-content:center;align-items:center;height:100vh;margin:0}
			.container{text-align:center;padding:40px;background:#16213e;border-radius:12px;box-shadow:0 4px 20px rgba(0,0,0,0.3)}
			h1{color:#4ade80}
			.logo{font-size:48px;margin-bottom:20px}
		</style></head><body><div class="container"><div class="logo">🎉</div><h1>Discord Conectado!</h1><p>Sua conta foi vinculada ao GoAnime com sucesso.</p><p>Você pode fechar esta janela e voltar ao aplicativo.</p></div></body></html>`)

		codeChan <- code
	})

	// Inicia o servidor em goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Aguarda o código ou erro (timeout de 5 minutos)
	select {
	case code := <-codeChan:
		// Para o servidor
		server.Close()
		return code, nil
	case err := <-errChan:
		server.Close()
		return "", err
	case <-time.After(5 * time.Minute):
		server.Close()
		return "", fmt.Errorf("timeout aguardando callback OAuth2")
	}
}

// ExchangeCodeForToken troca o código de autorização por um access token
func (b *DiscordBot) ExchangeCodeForToken(clientID, clientSecret, code, redirectURI string) (string, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao trocar código: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("discord retornou status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("erro ao decodificar token: %w", err)
	}

	return tokenResp.AccessToken, nil
}
