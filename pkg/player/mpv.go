package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PlayAnime abre o vídeo usando o MPV local
func PlayAnime(url string, title string) error {
	if url == "" {
		return fmt.Errorf("URL vazia")
	}

	// 1. Descobre onde o programa está rodando
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	// 2. Monta o caminho: ./bin/mpv.exe
	mpvPath := filepath.Join(dir, "bin", "mpv.exe")

	// 3. Verifica se o mpv.exe existe mesmo
	if _, err := os.Stat(mpvPath); os.IsNotExist(err) {
		return fmt.Errorf("mpv.exe não encontrado em: %s", mpvPath)
	}

	// 4. Configurações de Alta Qualidade (GPU)
	args := []string{
		url,
		"--force-window=immediate", // Abre a janela na hora
		"--title=" + title,         // Coloca o nome no topo
		"--hwdec=auto",             // Usa a Placa de Vídeo (Aceleração)
		"--vo=gpu",                 // Renderização via GPU
		"--geometry=50%:50%",       // Centraliza a janela
	}

	fmt.Printf("Iniciando MPV: %s com args %v\n", mpvPath, args)

	// 5. Executa
	cmd := exec.Command(mpvPath, args...)
	err = cmd.Start() // Start não trava o app principal
	
	if err != nil {
		return fmt.Errorf("erro ao iniciar MPV: %v", err)
	}

	return nil
}