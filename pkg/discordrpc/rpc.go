// Package discordrpc implementa Discord Rich Presence para GoAnime
// Mostra no perfil do Discord o que o usuário está assistindo
package discordrpc

import (
	"fmt"
	"sync"
	"time"

	"github.com/hugolgst/rich-go/client"
)

const (
	// Discord Application ID (criar em https://discord.com/developers/applications)
	// GoAnime Application ID
	ApplicationID = "1234567890123456789" // TODO: Substituir por ID real
)

// RichPresence gerencia a presença no Discord
type RichPresence struct {
	mu           sync.Mutex
	connected    bool
	currentAnime string
	currentEp    string
	startTime    time.Time
	endTime      time.Time
	coverURL     string
	episodeDur   float64
	currentPos   float64
	isPaused     bool
	updateTicker *time.Ticker
	stopChan     chan struct{}
}

// Activity representa a atividade a ser exibida no Discord
type Activity struct {
	AnimeName     string  `json:"animeName"`
	EpisodeTitle  string  `json:"episodeTitle"`
	EpisodeNumber int     `json:"episodeNumber"`
	CoverURL      string  `json:"coverURL"`
	Duration      float64 `json:"duration"` // segundos
	Position      float64 `json:"position"` // segundos
	IsPaused      bool    `json:"isPaused"`
}

var (
	instance *RichPresence
	once     sync.Once
)

// Get retorna a instância singleton do RichPresence
func Get() *RichPresence {
	once.Do(func() {
		instance = &RichPresence{
			stopChan: make(chan struct{}),
		}
	})
	return instance
}

// Connect conecta ao Discord RPC
func (r *RichPresence) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	err := client.Login(ApplicationID)
	if err != nil {
		return fmt.Errorf("erro ao conectar ao Discord: %w", err)
	}

	r.connected = true
	fmt.Println("[Discord RPC] ✓ Conectado ao Discord")

	// Inicia ticker para atualizar presença periodicamente
	r.updateTicker = time.NewTicker(15 * time.Second)
	go r.updateLoop()

	return nil
}

// Disconnect desconecta do Discord RPC
func (r *RichPresence) Disconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return
	}

	if r.updateTicker != nil {
		r.updateTicker.Stop()
	}

	close(r.stopChan)
	client.Logout()
	r.connected = false
	fmt.Println("[Discord RPC] Desconectado do Discord")
}

// IsConnected verifica se está conectado
func (r *RichPresence) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connected
}

// SetActivity define a atividade atual no Discord
func (r *RichPresence) SetActivity(activity Activity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		// Tenta conectar automaticamente
		if err := client.Login(ApplicationID); err != nil {
			return fmt.Errorf("discord não conectado")
		}
		r.connected = true
	}

	r.currentAnime = activity.AnimeName
	r.currentEp = activity.EpisodeTitle
	r.coverURL = activity.CoverURL
	r.episodeDur = activity.Duration
	r.currentPos = activity.Position
	r.isPaused = activity.IsPaused
	r.startTime = time.Now().Add(-time.Duration(activity.Position) * time.Second)
	r.endTime = time.Now().Add(time.Duration(activity.Duration-activity.Position) * time.Second)

	return r.updatePresence()
}

// UpdatePosition atualiza apenas a posição (chamado frequentemente)
func (r *RichPresence) UpdatePosition(position float64, isPaused bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.currentPos = position
	r.isPaused = isPaused

	if isPaused {
		r.endTime = time.Time{} // Remove timestamp quando pausado
	} else {
		r.endTime = time.Now().Add(time.Duration(r.episodeDur-position) * time.Second)
	}
}

// Clear limpa a presença
func (r *RichPresence) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return
	}

	client.Logout()
	r.currentAnime = ""
	r.currentEp = ""
	r.coverURL = ""
	fmt.Println("[Discord RPC] Presença limpa")
}

// updatePresence atualiza a presença no Discord
func (r *RichPresence) updatePresence() error {
	if r.currentAnime == "" {
		return nil
	}

	// Formata detalhes
	details := r.currentAnime
	if len(details) > 128 {
		details = details[:125] + "..."
	}

	state := r.currentEp
	if r.isPaused {
		state = "⏸ " + state + " (Pausado)"
	} else {
		// Adiciona tempo restante
		remaining := r.episodeDur - r.currentPos
		mins := int(remaining) / 60
		if mins > 0 {
			state = fmt.Sprintf("▶ %s (%d min restantes)", state, mins)
		}
	}
	if len(state) > 128 {
		state = state[:125] + "..."
	}

	// Monta a atividade
	activity := client.Activity{
		Details:    details,
		State:      state,
		LargeImage: r.coverURL,
		LargeText:  r.currentAnime,
		SmallImage: "goanime_icon", // Ícone do GoAnime no Discord App
		SmallText:  "GoAnime - Anime Player",
		Buttons: []*client.Button{
			{
				Label: "🎬 Baixar GoAnime",
				Url:   "https://github.com/goanime/goanime",
			},
		},
	}

	// Timestamps (mostra tempo decorrido/restante)
	if !r.isPaused && !r.endTime.IsZero() {
		activity.Timestamps = &client.Timestamps{
			Start: &r.startTime,
			End:   &r.endTime,
		}
	}

	err := client.SetActivity(activity)
	if err != nil {
		fmt.Printf("[Discord RPC] Erro ao atualizar: %v\n", err)
		return err
	}

	return nil
}

// updateLoop atualiza a presença periodicamente
func (r *RichPresence) updateLoop() {
	for {
		select {
		case <-r.stopChan:
			return
		case <-r.updateTicker.C:
			r.mu.Lock()
			if r.connected && r.currentAnime != "" {
				r.updatePresence()
			}
			r.mu.Unlock()
		}
	}
}

// Watching define que o usuário está assistindo um anime
// Wrapper simplificado para uso fácil
func Watching(animeName string, episode string, episodeNum int, coverURL string, duration float64) error {
	return Get().SetActivity(Activity{
		AnimeName:     animeName,
		EpisodeTitle:  fmt.Sprintf("Episódio %d - %s", episodeNum, episode),
		EpisodeNumber: episodeNum,
		CoverURL:      coverURL,
		Duration:      duration,
		Position:      0,
		IsPaused:      false,
	})
}

// StopWatching para de mostrar que está assistindo
func StopWatching() {
	Get().Clear()
}
