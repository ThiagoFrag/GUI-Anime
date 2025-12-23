//go:build libmpv
// +build libmpv

// Package embeddedplayer implementa um player de vídeo integrado usando libmpv
// Este player é renderizado diretamente dentro da janela do GoAnime
// Oferece upscaling neural (FSRCNNX/Anime4K), ICC profiles e integração perfeita
//
// Para compilar com suporte a libmpv:
//
//	go build -tags libmpv
//
// Requer libmpv.dll no Windows, libmpv.so no Linux, libmpv.dylib no macOS
package embeddedplayer

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/gen2brain/go-mpv"
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
	QualityLow    QualityMode = "low"    // Para GPUs fracas/notebooks
	QualityMedium QualityMode = "medium" // FSR upscaling
	QualityHigh   QualityMode = "high"   // FSRCNNX neural network
	QualityAnime  QualityMode = "anime"  // Anime4K optimizado
)

// SkipSegment representa um segmento para pular (abertura/encerramento)
type SkipSegment struct {
	Type      string  `json:"type"`      // "opening", "ending", "recap"
	StartTime float64 `json:"startTime"` // segundos
	EndTime   float64 `json:"endTime"`   // segundos
}

// TrackInfo representa informações de uma trilha (áudio/legenda)
type TrackInfo struct {
	ID       int    `json:"id"`
	Type     string `json:"type"` // "audio", "sub"
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
	Position       float64     `json:"position"` // segundos
	Duration       float64     `json:"duration"` // segundos
	Progress       float64     `json:"progress"` // 0-100
	Volume         int         `json:"volume"`   // 0-150
	Muted          bool        `json:"muted"`
	Speed          float64     `json:"speed"` // 1.0 = normal
	Fullscreen     bool        `json:"fullscreen"`
	QualityMode    QualityMode `json:"qualityMode"`
	AudioTracks    []TrackInfo `json:"audioTracks"`
	SubtitleTracks []TrackInfo `json:"subtitleTracks"`
	CurrentAudio   int         `json:"currentAudio"`
	CurrentSub     int         `json:"currentSub"`
	BufferPercent  float64     `json:"bufferPercent"`
	DroppedFrames  int64       `json:"droppedFrames"`
	FPS            float64     `json:"fps"`
	Width          int         `json:"width"`
	Height         int         `json:"height"`
	Title          string      `json:"title"`
}

// EmbeddedPlayer é o player de vídeo integrado
type EmbeddedPlayer struct {
	mu          sync.RWMutex
	mpv         *mpv.Mpv
	state       PlayerState
	qualityMode QualityMode
	shaderPath  string

	// Callbacks para eventos
	OnStateChange  func(state PlayerState)
	OnTimeUpdate   func(position, duration float64)
	OnBuffering    func(percent float64)
	OnError        func(err error)
	OnTrackChange  func(trackType string, id int)
	OnSeek         func(position float64)
	OnVolumeChange func(volume int, muted bool)
	OnSkipSegment  func(segment SkipSegment) // Quando entrar em segmento para pular

	// Skip segments (abertura/encerramento)
	skipSegments    []SkipSegment
	autoSkipEnabled bool

	// Controle interno
	stopChan  chan struct{}
	eventLoop bool
}

var (
	globalPlayer *EmbeddedPlayer
	playerOnce   sync.Once
)

// GetPlayer retorna a instância global do player
func GetPlayer() *EmbeddedPlayer {
	playerOnce.Do(func() {
		globalPlayer = &EmbeddedPlayer{
			state:       StateIdle,
			qualityMode: QualityMedium,
			stopChan:    make(chan struct{}),
		}
	})
	return globalPlayer
}

// Initialize inicializa o player
func (p *EmbeddedPlayer) Initialize(shaderPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv != nil {
		return nil // Já inicializado
	}

	m := mpv.New()
	if m == nil {
		return fmt.Errorf("falha ao criar instância MPV")
	}

	if err := m.Initialize(); err != nil {
		return fmt.Errorf("falha ao inicializar MPV: %w", err)
	}

	p.mpv = m
	p.shaderPath = shaderPath

	// Configurar player
	p.setupBaseConfig()
	p.setupEventObservers()

	// Aplicar modo de qualidade inicial
	p.applyQualityMode(p.qualityMode)

	// Iniciar loop de eventos
	p.eventLoop = true
	go p.eventHandler()

	fmt.Println("[EmbeddedPlayer] ✓ Player inicializado")
	return nil
}

// setupBaseConfig configura opções base do MPV
func (p *EmbeddedPlayer) setupBaseConfig() {
	// === CONTROLES E OSC ===
	p.mpv.SetPropertyString("input-default-bindings", "yes")
	p.mpv.SetPropertyString("input-vo-keyboard", "yes")
	p.mpv.SetPropertyString("osc", "yes")
	p.mpv.SetPropertyString("load-scripts", "yes")

	// Estilo do OSC
	p.mpv.SetPropertyString("script-opts",
		"osc-layout=bottombar,osc-seekbarstyle=bar,osc-deadzonesize=0.5,"+
			"osc-minmousemove=0,osc-hidetimeout=2000,osc-fadeduration=250,"+
			"osc-showwindowed=yes,osc-showfullscreen=yes,osc-boxalpha=80")

	// === DECODIFICAÇÃO ===
	p.mpv.SetPropertyString("hwdec", "auto-safe")

	// === SINCRONIZAÇÃO DE VÍDEO ===
	p.mpv.SetPropertyString("video-sync", "display-resample")
	p.mpv.SetPropertyString("interpolation", "yes")
	p.mpv.SetPropertyString("tscale", "oversample")
	p.mpv.SetPropertyString("framedrop", "no")

	// === ÁUDIO ===
	p.mpv.SetPropertyString("audio-pitch-correction", "yes")
	p.mpv.SetPropertyString("audio-normalize-downmix", "yes")
	p.mpv.SetPropertyString("volume-max", "150")

	// === JANELA ===
	p.mpv.SetPropertyString("keep-open", "yes")
	p.mpv.SetPropertyString("force-window", "immediate")
	p.mpv.SetPropertyString("background", "#000000")

	// === OSD ESTILIZADO ===
	p.mpv.SetPropertyString("osd-font", "Segoe UI")
	p.mpv.SetPropertyString("osd-font-size", "36")
	p.mpv.SetPropertyString("osd-bold", "yes")
	p.mpv.SetPropertyString("osd-color", "#FFFFFFFF")
	p.mpv.SetPropertyString("osd-border-color", "#FF6B9DFF") // Rosa/roxo
	p.mpv.SetPropertyString("osd-border-size", "2.5")
	p.mpv.SetPropertyString("osd-shadow-color", "#80000000")
	p.mpv.SetPropertyString("osd-shadow-offset", "2")
	p.mpv.SetPropertyString("osd-back-color", "#60000000")
	p.mpv.SetPropertyString("osd-playing-msg", "▶ ${media-title}")

	// === LEGENDAS ESTILIZADAS ===
	p.mpv.SetPropertyString("sub-auto", "fuzzy")
	p.mpv.SetPropertyString("sub-font", "Segoe UI Semibold")
	p.mpv.SetPropertyString("sub-font-size", "46")
	p.mpv.SetPropertyString("sub-color", "#FFFFFFFF")
	p.mpv.SetPropertyString("sub-border-color", "#FF000000")
	p.mpv.SetPropertyString("sub-border-size", "2.5")
	p.mpv.SetPropertyString("sub-shadow-offset", "1")
	p.mpv.SetPropertyString("sub-margin-y", "40")

	// === CACHE PARA STREAMING ===
	p.mpv.SetPropertyString("cache", "yes")
	p.mpv.SetPropertyString("demuxer-max-bytes", "200MiB")
	p.mpv.SetPropertyString("demuxer-max-back-bytes", "100MiB")
	p.mpv.SetPropertyString("demuxer-readahead-secs", "120") // 2 min buffer

	// === CONFIGURAÇÃO POR OS ===
	switch runtime.GOOS {
	case "windows":
		p.mpv.SetPropertyString("vo", "gpu")
		p.mpv.SetPropertyString("gpu-context", "d3d11")
	case "linux":
		p.mpv.SetPropertyString("vo", "gpu")
	case "darwin":
		p.mpv.SetPropertyString("vo", "gpu")
		p.mpv.SetPropertyString("gpu-context", "macvk")
	}
}

// setupEventObservers configura observadores de eventos
func (p *EmbeddedPlayer) setupEventObservers() {
	// Observar propriedades importantes
	p.mpv.ObserveProperty(0, "time-pos", mpv.FormatDouble)
	p.mpv.ObserveProperty(0, "duration", mpv.FormatDouble)
	p.mpv.ObserveProperty(0, "pause", mpv.FormatFlag)
	p.mpv.ObserveProperty(0, "volume", mpv.FormatInt64)
	p.mpv.ObserveProperty(0, "mute", mpv.FormatFlag)
	p.mpv.ObserveProperty(0, "eof-reached", mpv.FormatFlag)
}

// eventHandler processa eventos do MPV
func (p *EmbeddedPlayer) eventHandler() {
	for p.eventLoop {
		event := p.mpv.WaitEvent(0.1) // 100ms timeout
		if event == nil {
			continue
		}

		switch event.EventID {
		case mpv.EventGetPropertyReply:
			p.handlePropertyChange(event)
		case mpv.EventStart:
			p.setState(StatePlaying)
		case mpv.EventEnd:
			p.setState(StateStopped)
		}
	}
}

// handlePropertyChange processa mudanças de propriedade
func (p *EmbeddedPlayer) handlePropertyChange(event *mpv.Event) {
	// Nota: Implementação simplificada - em produção, decodificar propriedades
	// do evento e chamar callbacks apropriados
}

// SetSkipSegments define os segmentos para pular (abertura/encerramento)
func (p *EmbeddedPlayer) SetSkipSegments(segments []SkipSegment) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.skipSegments = segments
	fmt.Printf("[EmbeddedPlayer] %d segmentos de skip definidos\n", len(segments))
}

// SetAutoSkip ativa/desativa pular automático
func (p *EmbeddedPlayer) SetAutoSkip(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.autoSkipEnabled = enabled
}

// checkSkipSegments verifica se está em um segmento para pular
func (p *EmbeddedPlayer) checkSkipSegments() {
	p.mu.RLock()
	segments := p.skipSegments
	autoSkip := p.autoSkipEnabled
	p.mu.RUnlock()

	if len(segments) == 0 {
		return
	}

	pos := p.GetPosition()

	for _, seg := range segments {
		// Se estiver dentro do segmento
		if pos >= seg.StartTime && pos < seg.EndTime-1 {
			if p.OnSkipSegment != nil {
				p.OnSkipSegment(seg)
			}

			// Auto-skip se habilitado
			if autoSkip {
				p.Seek(seg.EndTime)
				fmt.Printf("[EmbeddedPlayer] Auto-skip: %s (%.1fs -> %.1fs)\n",
					seg.Type, seg.StartTime, seg.EndTime)
			}
			break
		}
	}
}

// === CONTROLE DE REPRODUÇÃO ===

// LoadFile carrega um arquivo de vídeo
func (p *EmbeddedPlayer) LoadFile(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return fmt.Errorf("player não inicializado")
	}

	p.setState(StateLoading)

	err := p.mpv.Command([]string{"loadfile", path, "replace"})
	if err != nil {
		p.setState(StateError)
		return fmt.Errorf("erro ao carregar arquivo: %w", err)
	}

	return nil
}

// LoadURL carrega uma URL de streaming
func (p *EmbeddedPlayer) LoadURL(url string) error {
	return p.LoadFile(url) // MPV trata URLs igual a arquivos
}

// Play inicia reprodução
func (p *EmbeddedPlayer) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.SetPropertyString("pause", "no")
	p.setState(StatePlaying)
}

// Pause pausa reprodução
func (p *EmbeddedPlayer) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.SetPropertyString("pause", "yes")
	p.setState(StatePaused)
}

// TogglePause alterna play/pause
func (p *EmbeddedPlayer) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.Command([]string{"cycle", "pause"})
}

// Stop para reprodução
func (p *EmbeddedPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.Command([]string{"stop"})
	p.setState(StateStopped)
}

// Seek vai para posição em segundos
func (p *EmbeddedPlayer) Seek(seconds float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.Command([]string{"seek", fmt.Sprintf("%.2f", seconds), "absolute"})
}

// SeekRelative avança/retrocede relativamente
func (p *EmbeddedPlayer) SeekRelative(seconds float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.Command([]string{"seek", fmt.Sprintf("%.2f", seconds), "relative"})
}

// === VOLUME ===

// SetVolume define o volume (0-150)
func (p *EmbeddedPlayer) SetVolume(volume int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	if volume < 0 {
		volume = 0
	}
	if volume > 150 {
		volume = 150
	}

	p.mpv.SetPropertyString("volume", fmt.Sprintf("%d", volume))
}

// GetVolume retorna o volume atual
func (p *EmbeddedPlayer) GetVolume() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.mpv == nil {
		return 0
	}

	vol, _ := p.mpv.GetProperty("volume", mpv.FormatInt64)
	if v, ok := vol.(int64); ok {
		return int(v)
	}
	return 100
}

// ToggleMute alterna mudo
func (p *EmbeddedPlayer) ToggleMute() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.Command([]string{"cycle", "mute"})
}

// === QUALIDADE ===

// SetQualityMode define o modo de qualidade
func (p *EmbeddedPlayer) SetQualityMode(mode QualityMode) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.applyQualityMode(mode)
	p.qualityMode = mode
}

// applyQualityMode aplica as configurações do modo
func (p *EmbeddedPlayer) applyQualityMode(mode QualityMode) {
	if p.mpv == nil {
		return
	}

	// Limpar shaders anteriores
	p.mpv.SetPropertyString("glsl-shaders", "")

	switch mode {
	case QualityLow:
		p.applyLowQuality()
	case QualityMedium:
		p.applyMediumQuality()
	case QualityHigh:
		p.applyHighQuality()
	case QualityAnime:
		p.applyAnimeQuality()
	}

	fmt.Printf("[EmbeddedPlayer] Modo de qualidade: %s\n", mode)
}

func (p *EmbeddedPlayer) applyLowQuality() {
	p.mpv.SetPropertyString("profile", "fast")
	p.mpv.SetPropertyString("hwdec", "auto-safe")
	p.mpv.SetPropertyString("scale", "bilinear")
	p.mpv.SetPropertyString("cscale", "bilinear")
	p.mpv.SetPropertyString("deband", "no")
	p.mpv.SetPropertyString("interpolation", "no")
}

func (p *EmbeddedPlayer) applyMediumQuality() {
	p.mpv.SetPropertyString("profile", "gpu-hq")
	p.mpv.SetPropertyString("hwdec", "auto-safe")
	p.mpv.SetPropertyString("scale", "spline36")
	p.mpv.SetPropertyString("cscale", "spline36")
	p.mpv.SetPropertyString("dscale", "mitchell")
	p.mpv.SetPropertyString("deband", "yes")
	p.mpv.SetPropertyString("deband-iterations", "2")
	p.mpv.SetPropertyString("deband-threshold", "35")

	// FSR shader
	fsrPath := filepath.Join(p.shaderPath, "FSR.glsl")
	p.mpv.Command([]string{"change-list", "glsl-shaders", "append", fsrPath})
}

func (p *EmbeddedPlayer) applyHighQuality() {
	p.mpv.SetPropertyString("vo", "gpu-next")
	p.mpv.SetPropertyString("profile", "gpu-hq")
	p.mpv.SetPropertyString("hwdec", "auto-copy")
	p.mpv.SetPropertyString("scale", "ewa_lanczossharp")
	p.mpv.SetPropertyString("cscale", "ewa_lanczossharp")
	p.mpv.SetPropertyString("dscale", "mitchell")
	p.mpv.SetPropertyString("deband", "yes")
	p.mpv.SetPropertyString("deband-iterations", "4")
	p.mpv.SetPropertyString("deband-threshold", "48")
	p.mpv.SetPropertyString("deband-range", "24")
	p.mpv.SetPropertyString("temporal-dither", "yes")
	p.mpv.SetPropertyString("tone-mapping", "bt.2446a")

	// FSRCNNX shader (rede neural)
	fsrcnnxPath := filepath.Join(p.shaderPath, "FSRCNNX_x2_16-0-4-1.glsl")
	p.mpv.Command([]string{"change-list", "glsl-shaders", "append", fsrcnnxPath})
}

func (p *EmbeddedPlayer) applyAnimeQuality() {
	p.mpv.SetPropertyString("vo", "gpu-next")
	p.mpv.SetPropertyString("profile", "gpu-hq")
	p.mpv.SetPropertyString("hwdec", "auto-copy")
	p.mpv.SetPropertyString("scale", "ewa_lanczossharp")
	p.mpv.SetPropertyString("deband", "yes")
	p.mpv.SetPropertyString("deband-iterations", "4")

	// Anime4K shaders chain
	shaders := []string{
		"Anime4K_Clamp_Highlights.glsl",
		"Anime4K_Restore_CNN_VL.glsl",
		"Anime4K_Upscale_CNN_x2_VL.glsl",
		"Anime4K_AutoDownscalePre_x2.glsl",
		"Anime4K_AutoDownscalePre_x4.glsl",
		"Anime4K_Upscale_CNN_x2_M.glsl",
	}

	for _, shader := range shaders {
		shaderPath := filepath.Join(p.shaderPath, shader)
		p.mpv.Command([]string{"change-list", "glsl-shaders", "append", shaderPath})
	}
}

// === ICC PROFILES ===

// SetICCProfile define um perfil ICC para cores precisas
func (p *EmbeddedPlayer) SetICCProfile(profilePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return fmt.Errorf("player não inicializado")
	}

	p.mpv.SetPropertyString("icc-profile", profilePath)
	p.mpv.SetPropertyString("icc-profile-auto", "no")

	fmt.Printf("[EmbeddedPlayer] ICC Profile: %s\n", profilePath)
	return nil
}

// SetICCProfileAuto usa perfil ICC do monitor automaticamente
func (p *EmbeddedPlayer) SetICCProfileAuto(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	if enabled {
		p.mpv.SetPropertyString("icc-profile-auto", "yes")
	} else {
		p.mpv.SetPropertyString("icc-profile-auto", "no")
	}
}

// === INFORMAÇÕES ===

// GetPosition retorna a posição atual em segundos
func (p *EmbeddedPlayer) GetPosition() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.mpv == nil {
		return 0
	}

	pos, _ := p.mpv.GetProperty("time-pos", mpv.FormatDouble)
	if v, ok := pos.(float64); ok {
		return v
	}
	return 0
}

// GetDuration retorna a duração total em segundos
func (p *EmbeddedPlayer) GetDuration() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.mpv == nil {
		return 0
	}

	dur, _ := p.mpv.GetProperty("duration", mpv.FormatDouble)
	if v, ok := dur.(float64); ok {
		return v
	}
	return 0
}

// GetInfo retorna informações completas do player
func (p *EmbeddedPlayer) GetInfo() PlayerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	info := PlayerInfo{
		State:       p.state,
		Position:    p.GetPosition(),
		Duration:    p.GetDuration(),
		Volume:      p.GetVolume(),
		QualityMode: p.qualityMode,
	}

	if info.Duration > 0 {
		info.Progress = (info.Position / info.Duration) * 100
	}

	return info
}

// GetState retorna o estado atual
func (p *EmbeddedPlayer) GetState() PlayerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// setState define o estado e notifica
func (p *EmbeddedPlayer) setState(state PlayerState) {
	p.state = state
	if p.OnStateChange != nil {
		go p.OnStateChange(state)
	}
}

// === LEGENDAS ===

// LoadSubtitle carrega uma legenda externa
func (p *EmbeddedPlayer) LoadSubtitle(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return fmt.Errorf("player não inicializado")
	}

	return p.mpv.Command([]string{"sub-add", path, "select"})
}

// SetSubtitleTrack define a trilha de legenda
func (p *EmbeddedPlayer) SetSubtitleTrack(id int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.SetPropertyString("sid", fmt.Sprintf("%d", id))
}

// SetAudioTrack define a trilha de áudio
func (p *EmbeddedPlayer) SetAudioTrack(id int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.SetPropertyString("aid", fmt.Sprintf("%d", id))
}

// === TELA CHEIA ===

// SetFullscreen define modo tela cheia
func (p *EmbeddedPlayer) SetFullscreen(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	if enabled {
		p.mpv.SetPropertyString("fullscreen", "yes")
	} else {
		p.mpv.SetPropertyString("fullscreen", "no")
	}
}

// ToggleFullscreen alterna tela cheia
func (p *EmbeddedPlayer) ToggleFullscreen() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	p.mpv.Command([]string{"cycle", "fullscreen"})
}

// === VELOCIDADE ===

// SetSpeed define a velocidade de reprodução
func (p *EmbeddedPlayer) SetSpeed(speed float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mpv == nil {
		return
	}

	if speed < 0.25 {
		speed = 0.25
	}
	if speed > 4.0 {
		speed = 4.0
	}

	p.mpv.SetPropertyString("speed", fmt.Sprintf("%.2f", speed))
}

// === LIMPEZA ===

// Destroy libera recursos do player
func (p *EmbeddedPlayer) Destroy() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.eventLoop = false
	close(p.stopChan)

	if p.mpv != nil {
		p.mpv.TerminateDestroy()
		p.mpv = nil
	}

	fmt.Println("[EmbeddedPlayer] Player destruído")
}

// Run inicia o loop principal do player (bloqueante)
func (p *EmbeddedPlayer) Run() {
	for {
		select {
		case <-p.stopChan:
			return
		default:
			time.Sleep(16 * time.Millisecond) // ~60fps

			// Verificar skip segments periodicamente
			if p.state == StatePlaying {
				p.checkSkipSegments()
			}
		}
	}
}
