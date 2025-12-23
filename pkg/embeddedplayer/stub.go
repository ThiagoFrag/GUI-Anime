//go:build !libmpv
// +build !libmpv

// Package embeddedplayer - versão stub sem libmpv
// Para compilar com libmpv real, use: go build -tags libmpv
package embeddedplayer

import (
	"fmt"
	"sync"
)

// PlayerState representa o estado atual do player
type PlayerState string

const (
	StateIdle      PlayerState = "idle"
	StateLoading   PlayerState = "loading"
	StatePlaying   PlayerState = "playing"
	StatePaused    PlayerState = "paused"
	StateStopped   PlayerState = "stopped"
	StateBuffering PlayerState = "buffering"
	StateError     PlayerState = "error"
)

// QualityMode representa os modos de qualidade disponíveis
type QualityMode string

const (
	QualityLow    QualityMode = "low"
	QualityMedium QualityMode = "medium"
	QualityHigh   QualityMode = "high"
	QualityAnime  QualityMode = "anime"
)

// SkipSegment representa um segmento para pular
type SkipSegment struct {
	Type      string  `json:"type"`
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
}

// TrackInfo representa informações de uma trilha
type TrackInfo struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Language string `json:"language"`
	Default  bool   `json:"default"`
	Forced   bool   `json:"forced"`
	External bool   `json:"external"`
	Codec    string `json:"codec"`
}

// PlayerInfo contém informações do estado atual
type PlayerInfo struct {
	State          PlayerState `json:"state"`
	Position       float64     `json:"position"`
	Duration       float64     `json:"duration"`
	Volume         int         `json:"volume"`
	Muted          bool        `json:"muted"`
	QualityMode    QualityMode `json:"qualityMode"`
	AudioTracks    []TrackInfo `json:"audioTracks"`
	SubtitleTracks []TrackInfo `json:"subtitleTracks"`
	CurrentAudio   int         `json:"currentAudio"`
	CurrentSub     int         `json:"currentSub"`
	VideoWidth     int         `json:"videoWidth"`
	VideoHeight    int         `json:"videoHeight"`
	IsFullscreen   bool        `json:"isFullscreen"`
	BufferPercent  float64     `json:"bufferPercent"`
}

// EmbeddedPlayer - versão stub
type EmbeddedPlayer struct {
	mu           sync.RWMutex
	state        PlayerState
	qualityMode  QualityMode
	volume       int
	muted        bool
	skipSegments []SkipSegment
	autoSkip     bool

	OnStateChange func(state PlayerState)
	OnTimeUpdate  func(position, duration float64)
	OnSkipSegment func(segment SkipSegment)
	OnError       func(err error)
	OnTrackChange func(trackType string, trackID int)
}

var (
	playerInstance *EmbeddedPlayer
	playerOnce     sync.Once
)

// GetPlayer retorna a instância singleton do player (stub)
func GetPlayer() *EmbeddedPlayer {
	playerOnce.Do(func() {
		playerInstance = &EmbeddedPlayer{
			state:       StateIdle,
			qualityMode: QualityMedium,
			volume:      100,
			autoSkip:    true,
		}
	})
	return playerInstance
}

// IsInitialized verifica se o player foi inicializado
func (p *EmbeddedPlayer) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state != StateIdle
}

func (p *EmbeddedPlayer) Initialize(shaderPath string) error {
	fmt.Println("[EmbeddedPlayer STUB] libmpv não disponível - use player4k externo")
	p.mu.Lock()
	p.state = StateStopped
	p.mu.Unlock()
	return nil
}

func (p *EmbeddedPlayer) LoadURL(url string) error {
	fmt.Printf("[EmbeddedPlayer STUB] LoadURL: %s (não implementado)\n", url)
	return nil
}

func (p *EmbeddedPlayer) Play() {
	fmt.Println("[EmbeddedPlayer STUB] Play")
}

func (p *EmbeddedPlayer) Pause() {
	fmt.Println("[EmbeddedPlayer STUB] Pause")
}

func (p *EmbeddedPlayer) TogglePause() {
	fmt.Println("[EmbeddedPlayer STUB] TogglePause")
}

func (p *EmbeddedPlayer) Stop() {
	fmt.Println("[EmbeddedPlayer STUB] Stop")
}

func (p *EmbeddedPlayer) Seek(position float64) {
	fmt.Printf("[EmbeddedPlayer STUB] Seek to %.1f\n", position)
}

func (p *EmbeddedPlayer) SeekRelative(offset float64) {
	fmt.Printf("[EmbeddedPlayer STUB] SeekRelative %.1f\n", offset)
}

func (p *EmbeddedPlayer) SetVolume(volume int) {
	p.mu.Lock()
	p.volume = volume
	p.mu.Unlock()
}

func (p *EmbeddedPlayer) GetVolume() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.volume
}

func (p *EmbeddedPlayer) SetMute(muted bool) {
	p.mu.Lock()
	p.muted = muted
	p.mu.Unlock()
}

func (p *EmbeddedPlayer) ToggleMute() {
	p.mu.Lock()
	p.muted = !p.muted
	p.mu.Unlock()
}

func (p *EmbeddedPlayer) SetFullscreen(enabled bool) {
	fmt.Printf("[EmbeddedPlayer STUB] Fullscreen: %v\n", enabled)
}

func (p *EmbeddedPlayer) ToggleFullscreen() {
	fmt.Println("[EmbeddedPlayer STUB] ToggleFullscreen")
}

func (p *EmbeddedPlayer) SetQualityMode(mode QualityMode) {
	p.mu.Lock()
	p.qualityMode = mode
	p.mu.Unlock()
	fmt.Printf("[EmbeddedPlayer STUB] Quality: %s\n", mode)
}

func (p *EmbeddedPlayer) GetQualityMode() QualityMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.qualityMode
}

func (p *EmbeddedPlayer) SetAudioTrack(trackID int) {
	fmt.Printf("[EmbeddedPlayer STUB] Audio track: %d\n", trackID)
}

func (p *EmbeddedPlayer) SetSubtitleTrack(trackID int) {
	fmt.Printf("[EmbeddedPlayer STUB] Subtitle track: %d\n", trackID)
}

func (p *EmbeddedPlayer) AddExternalSubtitle(path string) error {
	fmt.Printf("[EmbeddedPlayer STUB] Add subtitle: %s\n", path)
	return nil
}

func (p *EmbeddedPlayer) LoadSubtitle(path string) error {
	fmt.Printf("[EmbeddedPlayer STUB] Load subtitle: %s\n", path)
	return nil
}

func (p *EmbeddedPlayer) SetSpeed(speed float64) {
	fmt.Printf("[EmbeddedPlayer STUB] Speed: %.2fx\n", speed)
}

func (p *EmbeddedPlayer) SetSubtitleDelay(delay float64) {
	fmt.Printf("[EmbeddedPlayer STUB] Subtitle delay: %.1f\n", delay)
}

func (p *EmbeddedPlayer) SetAudioDelay(delay float64) {
	fmt.Printf("[EmbeddedPlayer STUB] Audio delay: %.1f\n", delay)
}

func (p *EmbeddedPlayer) SetICCProfile(profilePath string) {
	fmt.Printf("[EmbeddedPlayer STUB] ICC Profile: %s\n", profilePath)
}

func (p *EmbeddedPlayer) SetICCProfileAuto(enabled bool) {
	fmt.Printf("[EmbeddedPlayer STUB] ICC Auto: %v\n", enabled)
}

func (p *EmbeddedPlayer) SetSkipSegments(segments []SkipSegment) {
	p.mu.Lock()
	p.skipSegments = segments
	p.mu.Unlock()
	fmt.Printf("[EmbeddedPlayer STUB] %d skip segments\n", len(segments))
}

func (p *EmbeddedPlayer) SetAutoSkip(enabled bool) {
	p.mu.Lock()
	p.autoSkip = enabled
	p.mu.Unlock()
}

func (p *EmbeddedPlayer) SkipCurrentSegment() {
	fmt.Println("[EmbeddedPlayer STUB] SkipCurrentSegment")
}

func (p *EmbeddedPlayer) GetPosition() float64 {
	return 0
}

func (p *EmbeddedPlayer) GetDuration() float64 {
	return 0
}

func (p *EmbeddedPlayer) GetInfo() PlayerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PlayerInfo{
		State:       p.state,
		Volume:      p.volume,
		Muted:       p.muted,
		QualityMode: p.qualityMode,
	}
}

func (p *EmbeddedPlayer) Destroy() {
	fmt.Println("[EmbeddedPlayer STUB] Destroy")
}

func (p *EmbeddedPlayer) Screenshot(path string) error {
	return fmt.Errorf("screenshot não disponível no modo stub")
}
