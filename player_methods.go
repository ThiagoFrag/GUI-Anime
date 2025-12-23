// player_methods.go - Métodos do player integrado para o frontend
// Expõe controles do player libmpv e funcionalidades avançadas
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"GoAnimeGUI/pkg/discordrpc"
	"GoAnimeGUI/pkg/embeddedplayer"
)

// ==============================
// PLAYER INTEGRADO (libmpv)
// ==============================

// InitEmbeddedPlayer inicializa o player integrado
func (a *App) InitEmbeddedPlayer() error {
	player := embeddedplayer.GetPlayer()

	// Busca caminho dos shaders
	shaderPath := a.getShaderPath()

	err := player.Initialize(shaderPath)
	if err != nil {
		return fmt.Errorf("erro ao inicializar player: %w", err)
	}

	// Configura callbacks
	player.OnStateChange = func(state embeddedplayer.PlayerState) {
		// Notifica frontend via eventos Wails
		a.emitEvent("player:stateChange", string(state))
	}

	player.OnTimeUpdate = func(position, duration float64) {
		a.emitEvent("player:timeUpdate", map[string]float64{
			"position": position,
			"duration": duration,
		})
	}

	player.OnSkipSegment = func(segment embeddedplayer.SkipSegment) {
		a.emitEvent("player:skipSegment", segment)
	}

	return nil
}

// getShaderPath retorna o caminho dos shaders
func (a *App) getShaderPath() string {
	// Tenta várias localizações
	paths := []string{
		filepath.Join(a.getAppDir(), "shaders"),
		filepath.Join(a.getAppDir(), "player4k", "shaders"),
		`C:\Users\th\Documents\codigos\player4k\shaders`,
	}

	for _, p := range paths {
		if exists(p) {
			return p
		}
	}

	return paths[0] // Retorna primeiro como fallback
}

// PlayerLoad carrega um vídeo no player integrado
func (a *App) PlayerLoad(url string, title string) error {
	player := embeddedplayer.GetPlayer()

	if err := player.LoadURL(url); err != nil {
		return err
	}

	return nil
}

// PlayerPlay inicia reprodução
func (a *App) PlayerPlay() {
	embeddedplayer.GetPlayer().Play()
}

// PlayerPause pausa reprodução
func (a *App) PlayerPause() {
	embeddedplayer.GetPlayer().Pause()
}

// PlayerToggle alterna play/pause
func (a *App) PlayerToggle() {
	embeddedplayer.GetPlayer().TogglePause()
}

// PlayerStop para reprodução
func (a *App) PlayerStop() {
	embeddedplayer.GetPlayer().Stop()
}

// PlayerSeek vai para posição em segundos
func (a *App) PlayerSeek(seconds float64) {
	embeddedplayer.GetPlayer().Seek(seconds)
}

// PlayerSeekRelative avança/retrocede
func (a *App) PlayerSeekRelative(seconds float64) {
	embeddedplayer.GetPlayer().SeekRelative(seconds)
}

// PlayerSetVolume define volume (0-150)
func (a *App) PlayerSetVolume(volume int) {
	embeddedplayer.GetPlayer().SetVolume(volume)
}

// PlayerGetVolume retorna volume atual
func (a *App) PlayerGetVolume() int {
	return embeddedplayer.GetPlayer().GetVolume()
}

// PlayerToggleMute alterna mudo
func (a *App) PlayerToggleMute() {
	embeddedplayer.GetPlayer().ToggleMute()
}

// PlayerToggleFullscreen alterna tela cheia
func (a *App) PlayerToggleFullscreen() {
	embeddedplayer.GetPlayer().ToggleFullscreen()
}

// PlayerSetQuality define modo de qualidade
// mode: "low", "medium", "high", "anime"
func (a *App) PlayerSetQuality(mode string) {
	var qm embeddedplayer.QualityMode
	switch mode {
	case "low":
		qm = embeddedplayer.QualityLow
	case "medium":
		qm = embeddedplayer.QualityMedium
	case "high":
		qm = embeddedplayer.QualityHigh
	case "anime":
		qm = embeddedplayer.QualityAnime
	default:
		qm = embeddedplayer.QualityMedium
	}
	embeddedplayer.GetPlayer().SetQualityMode(qm)
}

// PlayerGetInfo retorna informações do player
func (a *App) PlayerGetInfo() embeddedplayer.PlayerInfo {
	return embeddedplayer.GetPlayer().GetInfo()
}

// PlayerLoadSubtitle carrega legenda externa
func (a *App) PlayerLoadSubtitle(path string) error {
	return embeddedplayer.GetPlayer().LoadSubtitle(path)
}

// PlayerSetSubtitleTrack define trilha de legenda
func (a *App) PlayerSetSubtitleTrack(id int) {
	embeddedplayer.GetPlayer().SetSubtitleTrack(id)
}

// PlayerSetAudioTrack define trilha de áudio
func (a *App) PlayerSetAudioTrack(id int) {
	embeddedplayer.GetPlayer().SetAudioTrack(id)
}

// PlayerSetSpeed define velocidade de reprodução
func (a *App) PlayerSetSpeed(speed float64) {
	embeddedplayer.GetPlayer().SetSpeed(speed)
}

// PlayerSetICCProfile define perfil ICC para cores precisas
func (a *App) PlayerSetICCProfile(profilePath string) {
	embeddedplayer.GetPlayer().SetICCProfile(profilePath)
}

// PlayerSetICCProfileAuto usa perfil ICC do monitor
func (a *App) PlayerSetICCProfileAuto(enabled bool) {
	embeddedplayer.GetPlayer().SetICCProfileAuto(enabled)
}

// PlayerSetAutoSkip ativa/desativa pular automático
func (a *App) PlayerSetAutoSkip(enabled bool) {
	embeddedplayer.GetPlayer().SetAutoSkip(enabled)
}

// ==============================
// ANISKIP - PULAR ABERTURA
// ==============================

// ConfigurePlayerSkipSegments configura segmentos de skip no player integrado
// Chamado após GetSkipTimes de app.go
func (a *App) ConfigurePlayerSkipSegments(skipTimes *SkipTimesResult) {
	if skipTimes == nil {
		return
	}

	var segments []embeddedplayer.SkipSegment
	if skipTimes.HasOpening {
		segments = append(segments, embeddedplayer.SkipSegment{
			Type:      "opening",
			StartTime: skipTimes.OpeningStart,
			EndTime:   skipTimes.OpeningEnd,
		})
	}
	if skipTimes.HasEnding {
		segments = append(segments, embeddedplayer.SkipSegment{
			Type:      "ending",
			StartTime: skipTimes.EndingStart,
			EndTime:   skipTimes.EndingEnd,
		})
	}
	if skipTimes.HasRecap {
		segments = append(segments, embeddedplayer.SkipSegment{
			Type:      "recap",
			StartTime: skipTimes.RecapStart,
			EndTime:   skipTimes.RecapEnd,
		})
	}

	embeddedplayer.GetPlayer().SetSkipSegments(segments)
	fmt.Printf("[AniSkip] Configurados %d segmentos de skip no player\n", len(segments))
}

// SkipOpening pula a abertura do episódio atual
func (a *App) SkipOpening() {
	info := a.PlayerGetInfo()
	if info.Duration == 0 {
		return
	}

	// Busca segmentos do player
	player := embeddedplayer.GetPlayer()
	// Pula para o fim da abertura se estiver dentro dela
	// Implementação simplificada - em produção usar os segmentos armazenados
	player.SeekRelative(90) // Pula 90 segundos (duração média de abertura)
}

// SkipEnding pula o encerramento do episódio atual
func (a *App) SkipEnding() {
	player := embeddedplayer.GetPlayer()
	info := player.GetInfo()

	// Vai para 90 segundos antes do fim (duração média de ending)
	if info.Duration > 90 {
		player.Seek(info.Duration - 5) // Vai quase pro fim
	}
}

// ==============================
// DISCORD RICH PRESENCE
// ==============================

// DiscordRPCConnect conecta ao Discord Rich Presence
func (a *App) DiscordRPCConnect() error {
	return discordrpc.Get().Connect()
}

// DiscordRPCDisconnect desconecta do Discord
func (a *App) DiscordRPCDisconnect() {
	discordrpc.Get().Disconnect()
}

// DiscordRPCIsConnected verifica se está conectado
func (a *App) DiscordRPCIsConnected() bool {
	return discordrpc.Get().IsConnected()
}

// DiscordRPCSetWatching define que está assistindo um anime
func (a *App) DiscordRPCSetWatching(animeName string, episodeTitle string, episodeNum int, coverURL string, duration float64) error {
	return discordrpc.Watching(animeName, episodeTitle, episodeNum, coverURL, duration)
}

// DiscordRPCUpdatePosition atualiza posição de reprodução
func (a *App) DiscordRPCUpdatePosition(position float64, isPaused bool) {
	discordrpc.Get().UpdatePosition(position, isPaused)
}

// DiscordRPCClear limpa a presença
func (a *App) DiscordRPCClear() {
	discordrpc.StopWatching()
}

// ==============================
// QUALITY MODES INFO
// ==============================

// QualityModeInfo informações de um modo de qualidade
type QualityModeInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	GPURequired string   `json:"gpuRequired"`
	Features    []string `json:"features"`
}

// GetQualityModes retorna todos os modos de qualidade disponíveis
func (a *App) GetQualityModes() []QualityModeInfo {
	return []QualityModeInfo{
		{
			ID:          "low",
			Name:        "Econômico",
			Description: "Para notebooks e GPUs integradas",
			Icon:        "🔋",
			GPURequired: "Intel HD / AMD APU",
			Features:    []string{"Hardware decoding", "Baixo consumo", "Compatível com qualquer PC"},
		},
		{
			ID:          "medium",
			Name:        "Equilibrado",
			Description: "FSR upscaling - boa qualidade com performance",
			Icon:        "⚖️",
			GPURequired: "GTX 1050 / RX 560",
			Features:    []string{"AMD FSR", "Debanding", "Dithering"},
		},
		{
			ID:          "high",
			Name:        "Ultra Neural",
			Description: "FSRCNNX - upscaling por rede neural",
			Icon:        "🚀",
			GPURequired: "RTX 3060 / RX 6700",
			Features:    []string{"FSRCNNX Neural Network", "HDR Tone Mapping", "Temporal Dithering"},
		},
		{
			ID:          "anime",
			Name:        "Anime4K",
			Description: "Otimizado especificamente para anime",
			Icon:        "🎌",
			GPURequired: "GTX 1060 / RX 580",
			Features:    []string{"Anime4K Shaders", "Line Art Enhancement", "Upscale 2x/4x"},
		},
	}
}

// ==============================
// HELPERS
// ==============================

// getAppDir retorna o diretório do executável
func (a *App) getAppDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}

// emitEvent emite um evento para o frontend
func (a *App) emitEvent(event string, data interface{}) {
	// TODO: Usar wails runtime para emitir eventos
	// runtime.EventsEmit(a.ctx, event, data)
	fmt.Printf("[Event] %s: %v\n", event, data)
}

// exists verifica se arquivo/diretório existe
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ==============================
// PLAYER 4K FUNCTIONS (usando player4k.exe externo)
// ==============================

// findPlayer4KPath procura o player4k.exe em vários locais
func (a *App) findPlayer4KPath() string {
	possiblePaths := []string{}

	// Diretório do executável
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		possiblePaths = append(possiblePaths,
			filepath.Join(exeDir, "player4k", "player4k.exe"),
			filepath.Join(exeDir, "bin", "player4k.exe"),
			filepath.Join(exeDir, "player4k.exe"),
		)
	}

	// Diretório atual
	if dir, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths,
			filepath.Join(dir, "player4k", "player4k.exe"),
			filepath.Join(dir, "bin", "player4k.exe"),
			filepath.Join(dir, "player4k.exe"),
		)
	}

	// Diretório do projeto player4k (desenvolvimento)
	username := os.Getenv("USERNAME")
	possiblePaths = append(possiblePaths,
		filepath.Join("C:\\Users", username, "Documents", "codigos", "player4k", "player4k.exe"),
		"..\\player4k\\player4k.exe",
	)

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("[Player4K] Encontrado em: %s\n", path)
			return path
		}
	}

	return ""
}

// IsPlayer4KAvailable verifica se o player 4K está disponível
func (a *App) IsPlayer4KAvailable() bool {
	path := a.findPlayer4KPath()
	return path != ""
}

// GetPlayer4KModes retorna os modos de qualidade disponíveis
func (a *App) GetPlayer4KModes() []QualityModeInfo {
	return a.GetQualityModes()
}

// PlayWithPlayer4K inicia reprodução com o Player 4K
func (a *App) PlayWithPlayer4K(url string, mode string, useAnimeShaders bool) error {
	player4kPath := a.findPlayer4KPath()
	if player4kPath == "" {
		return fmt.Errorf("player4k.exe não encontrado")
	}

	args := []string{
		"--mode=" + mode,
		"--fs", // fullscreen
	}

	if useAnimeShaders {
		args = append(args, "--anime")
	}

	args = append(args, url)

	fmt.Printf("[Player4K] Executando: %s %v\n", player4kPath, args)
	cmd := exec.Command(player4kPath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar player4k: %v", err)
	}

	return nil
}

// StopPlayer4K para o Player 4K
func (a *App) StopPlayer4K() error {
	// Player4K é um processo separado, para parar precisaria matar o processo
	// Por ora, deixamos o usuário fechar manualmente
	return nil
}

// PlayWithPlayer4KTitle inicia reprodução com Player 4K incluindo título
func (a *App) PlayWithPlayer4KTitle(url string, title string, mode string) error {
	player4kPath := a.findPlayer4KPath()
	if player4kPath == "" {
		return fmt.Errorf("player4k.exe não encontrado")
	}

	args := []string{
		"--mode=" + mode,
		"--fs", // fullscreen
	}

	if title != "" {
		args = append(args, "--title="+title)
	}

	args = append(args, url)

	fmt.Printf("[Player4K] Executando: %s %v\n", player4kPath, args)
	cmd := exec.Command(player4kPath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar player4k: %v", err)
	}

	return nil
}

// PlayWithPlayer4KTitleSub inicia reprodução com Player 4K incluindo título e legenda
func (a *App) PlayWithPlayer4KTitleSub(url string, title string, mode string, subtitleUrl string) error {
	player4kPath := a.findPlayer4KPath()
	if player4kPath == "" {
		return fmt.Errorf("player4k.exe não encontrado")
	}

	args := []string{
		"--mode=" + mode,
		"--fs", // fullscreen
	}

	if title != "" {
		args = append(args, "--title="+title)
	}

	if subtitleUrl != "" {
		args = append(args, "--sub="+subtitleUrl)
	}

	args = append(args, url)

	fmt.Printf("[Player4K] Executando com legenda: %s %v\n", player4kPath, args)
	cmd := exec.Command(player4kPath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar player4k: %v", err)
	}

	return nil
}
