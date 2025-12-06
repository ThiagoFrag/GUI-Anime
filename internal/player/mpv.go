// Package player fornece integração com reprodutores de vídeo externos
package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MPV representa a configuração do player MPV
type MPV struct {
	Path      string
	YtdlpPath string
}

// New cria uma nova instância do player MPV
func New() *MPV {
	return &MPV{}
}

// FindMPV procura o MPV em vários locais comuns
func (m *MPV) FindMPV(savedPath string) string {
	// 1) Caminho salvo pelo usuário
	if savedPath != "" {
		if _, err := os.Stat(savedPath); err == nil {
			m.Path = savedPath
			return savedPath
		}
	}

	// 2) mpv no PATH
	if path, err := exec.LookPath("mpv"); err == nil {
		m.Path = path
		return path
	}

	// 3) Caminhos possíveis no Windows
	possiblePaths := []string{}

	// Diretório atual
	if dir, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(dir, "bin", "mpv.exe"))
		possiblePaths = append(possiblePaths, filepath.Join(dir, "bin", "mpv", "mpv.exe"))
	}

	possiblePaths = append(possiblePaths,
		"bin/mpv.exe",
		"bin/mpv/mpv.exe",
		"C:\\Program Files\\mpv\\mpv.exe",
		"C:\\Program Files (x86)\\mpv\\mpv.exe",
	)

	// Adiciona caminho do usuário
	if username := os.Getenv("USERNAME"); username != "" {
		possiblePaths = append(possiblePaths,
			fmt.Sprintf("C:\\Users\\%s\\AppData\\Local\\mpv\\mpv.exe", username),
			fmt.Sprintf("C:\\Users\\%s\\scoop\\apps\\mpv\\current\\mpv.exe", username),
		)
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			m.Path = path
			return path
		}
	}

	return ""
}

// FindYtdlp procura o yt-dlp em vários locais
func (m *MPV) FindYtdlp() string {
	// No PATH
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		m.YtdlpPath = path
		return path
	}

	// Caminhos comuns
	possiblePaths := []string{}

	if dir, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths,
			filepath.Join(dir, "bin", "yt-dlp.exe"),
			filepath.Join(dir, "yt-dlp.exe"),
		)
	}

	if username := os.Getenv("USERNAME"); username != "" {
		possiblePaths = append(possiblePaths,
			fmt.Sprintf("C:\\Users\\%s\\AppData\\Local\\yt-dlp\\yt-dlp.exe", username),
			fmt.Sprintf("C:\\Users\\%s\\scoop\\apps\\yt-dlp\\current\\yt-dlp.exe", username),
		)
	}

	possiblePaths = append(possiblePaths,
		"C:\\Program Files\\yt-dlp\\yt-dlp.exe",
	)

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			m.YtdlpPath = path
			return path
		}
	}

	return ""
}

// IsInstalled verifica se o MPV está instalado
func (m *MPV) IsInstalled() bool {
	return m.Path != "" || m.FindMPV("") != ""
}

// PlayOptions representa opções de reprodução
type PlayOptions struct {
	Title      string
	Fullscreen bool
	Volume     int // 0-100
	UseYtdlp   bool
}

// DefaultOptions retorna opções padrão de reprodução
func DefaultOptions() PlayOptions {
	return PlayOptions{
		Fullscreen: false,
		Volume:     100,
		UseYtdlp:   false,
	}
}

// Play reproduz uma URL de vídeo no MPV
func (m *MPV) Play(url string, opts PlayOptions) error {
	if url == "" {
		return fmt.Errorf("URL inválida")
	}

	mpvPath := m.Path
	if mpvPath == "" {
		mpvPath = m.FindMPV("")
		if mpvPath == "" {
			return fmt.Errorf("MPV não encontrado. Instale o MPV ou coloque na pasta bin/")
		}
	}

	// Argumentos base do MPV
	args := []string{
		"--force-window=immediate",
		"--hwdec=auto",
		"--vo=gpu",
	}

	// Título da janela
	if opts.Title != "" {
		args = append(args, fmt.Sprintf("--title=%s", opts.Title))
	}

	// Fullscreen
	if opts.Fullscreen {
		args = append(args, "--fullscreen")
	}

	// Volume
	if opts.Volume > 0 && opts.Volume <= 100 {
		args = append(args, fmt.Sprintf("--volume=%d", opts.Volume))
	}

	// Só usa yt-dlp se necessário e disponível
	isDirectStream := isDirectStreamURL(url)

	if !isDirectStream && opts.UseYtdlp {
		ytdlpPath := m.YtdlpPath
		if ytdlpPath == "" {
			ytdlpPath = m.FindYtdlp()
		}

		if ytdlpPath != "" {
			fmt.Printf("[MPV] URL é página web, usando yt-dlp: %s\n", ytdlpPath)
			args = append(args, "--ytdl-path="+ytdlpPath)
			args = append(args, "--ytdl-format=best")
		}
	}

	args = append(args, url)

	fmt.Printf("[MPV] Executando: %s %v\n", mpvPath, args)
	cmd := exec.Command(mpvPath, args...)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar MPV: %v", err)
	}

	fmt.Printf("[MPV] Reproduzindo: %s\n", url)
	return nil
}

// PlayWithWait reproduz e aguarda o término
func (m *MPV) PlayWithWait(url string, opts PlayOptions) error {
	if url == "" {
		return fmt.Errorf("URL inválida")
	}

	mpvPath := m.Path
	if mpvPath == "" {
		mpvPath = m.FindMPV("")
		if mpvPath == "" {
			return fmt.Errorf("MPV não encontrado")
		}
	}

	args := []string{
		"--force-window=immediate",
		"--hwdec=auto",
		url,
	}

	cmd := exec.Command(mpvPath, args...)
	return cmd.Run()
}

// isDirectStreamURL verifica se é uma URL de stream direto
func isDirectStreamURL(url string) bool {
	lowerURL := strings.ToLower(url)
	return strings.HasSuffix(lowerURL, ".mp4") ||
		strings.HasSuffix(lowerURL, ".m3u8") ||
		strings.HasSuffix(lowerURL, ".webm") ||
		strings.Contains(lowerURL, ".mp4?") ||
		strings.Contains(lowerURL, ".m3u8?") ||
		strings.Contains(lowerURL, ".webm?")
}

// GetPath retorna o caminho do MPV
func (m *MPV) GetPath() string {
	return m.Path
}

// SetPath define o caminho do MPV
func (m *MPV) SetPath(path string) {
	m.Path = path
}
