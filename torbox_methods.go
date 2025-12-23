package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"GoAnimeGUI/internal/utils"
	"GoAnimeGUI/pkg/jikan"
	"GoAnimeGUI/pkg/torbox"
)

// TorBoxConfig configuração do TorBox
type TorBoxConfig struct {
	APIKey string `json:"apiKey"`
}

// TorBoxFileInfo arquivo de vídeo disponível para streaming
type TorBoxFileInfo struct {
	ID         int    `json:"id"`
	TorrentID  int    `json:"torrentId"`
	Name       string `json:"name"`
	ShortName  string `json:"shortName"`
	Size       int64  `json:"size"`
	SizeStr    string `json:"sizeStr"`
	Episode    int    `json:"episode"`
	Season     int    `json:"season"`
	IsPlayable bool   `json:"isPlayable"`
}

// TorBoxTorrentInfo informações completas do torrent para exibição
type TorBoxTorrentInfo struct {
	ID           int              `json:"id"`
	Hash         string           `json:"hash"`
	Name         string           `json:"name"`
	Progress     float64          `json:"progress"`
	Status       string           `json:"status"`
	Cached       bool             `json:"cached"`
	Files        []TorBoxFileInfo `json:"files"`
	TotalSize    int64            `json:"totalSize"`
	TotalSizeStr string           `json:"totalSizeStr"`
}

// torboxClient cliente TorBox compartilhado
var torboxClient *torbox.Client

// InitTorBox inicializa o cliente TorBox com a API key
func (a *App) InitTorBox(apiKey string) bool {
	if apiKey == "" {
		fmt.Println("[TorBox] API key vazia")
		return false
	}
	torboxClient = torbox.New(apiKey)
	fmt.Println("[TorBox] Cliente inicializado")
	return true
}

// TorBoxGetUser retorna informações do usuário TorBox
func (a *App) TorBoxGetUser() *torbox.User {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	user, err := torboxClient.GetUser(a.ctx)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao obter usuário: %v\n", err)
		return nil
	}

	return user
}

// TorBoxSearchTorrents busca torrents de anime no Nyaa.si
func (a *App) TorBoxSearchTorrents(query string) []torbox.AnimeTorrent {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	results, err := torboxClient.SearchAnimeTorrents(a.ctx, query)
	if err != nil {
		fmt.Printf("[TorBox] Erro na busca: %v\n", err)
		return nil
	}

	return results
}

// TorBoxGetInstantStream busca e retorna link de streaming direto
func (a *App) TorBoxGetInstantStream(query string) *torbox.InstantStreamResult {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	result, err := torboxClient.GetInstantStream(a.ctx, query)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao obter stream: %v\n", err)
		return nil
	}

	return result
}

// TorBoxCheckCached verifica se hashes estão em cache
func (a *App) TorBoxCheckCached(hashes []string) map[string]bool {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	result, err := torboxClient.CheckCached(a.ctx, hashes)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao verificar cache: %v\n", err)
		return nil
	}

	return result
}

// TorBoxGetTorrents lista todos os torrents do usuário
func (a *App) TorBoxGetTorrents() []torbox.Torrent {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	torrents, err := torboxClient.GetTorrents(a.ctx)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao listar torrents: %v\n", err)
		return nil
	}

	return torrents
}

// TorBoxAddMagnet adiciona um torrent via magnet link
func (a *App) TorBoxAddMagnet(magnet string) *torbox.Torrent {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	torrent, err := torboxClient.AddMagnet(a.ctx, magnet, false)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao adicionar magnet: %v\n", err)
		return nil
	}

	return torrent
}

// TorBoxDeleteTorrent remove um torrent
func (a *App) TorBoxDeleteTorrent(torrentID int) bool {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return false
	}

	err := torboxClient.DeleteTorrent(a.ctx, torrentID)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao deletar torrent: %v\n", err)
		return false
	}

	return true
}

// TorBoxGetDownloadLink obtém link direto de download/streaming
func (a *App) TorBoxGetDownloadLink(torrentID, fileID int) string {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return ""
	}

	link, err := torboxClient.GetDownloadLink(a.ctx, torrentID, fileID)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao obter link: %v\n", err)
		return ""
	}

	return link
}

// TorBoxClearCache limpa o cache local do TorBox
func (a *App) TorBoxClearCache() {
	torbox.ClearCache()
	fmt.Println("[TorBox] Cache limpo")
}

// TorBoxStreamAnimeEpisode busca stream de um episódio específico de anime
// Usa o título do anime + número do episódio para buscar no Nyaa.si
func (a *App) TorBoxStreamAnimeEpisode(animeTitle string, episode int, quality string) *torbox.InstantStreamResult {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	// Formata query de busca
	query := fmt.Sprintf("%s %02d", animeTitle, episode)
	if quality != "" {
		query += " " + quality
	} else {
		query += " 1080p" // Qualidade padrão
	}

	fmt.Printf("[TorBox] Buscando stream: %s\n", query)

	result, err := torboxClient.GetInstantStream(a.ctx, query)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao obter stream: %v\n", err)
		return nil
	}

	return result
}

// TorBoxSearchAnimes busca animes via TorBox e retorna no formato para o frontend
func (a *App) TorBoxSearchAnimes(termo string) []TorBoxAnimeResult {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado - configure a API key")
		return nil
	}

	fmt.Printf("[TorBox] Buscando animes: %s\n", termo)

	results, err := torboxClient.SearchAnimeTorrents(a.ctx, termo)
	if err != nil {
		fmt.Printf("[TorBox] Erro na busca: %v\n", err)
		return nil
	}

	// Log para debug
	for i, r := range results {
		magnetPreview := r.Magnet
		if len(magnetPreview) > 50 {
			magnetPreview = magnetPreview[:50] + "..."
		}
		fmt.Printf("[TorBox] Resultado %d: Title='%s', Hash='%s', Magnet='%s'\n", i, r.Title, r.Hash, magnetPreview)
	}

	// Converte para o formato do frontend
	animes := make([]TorBoxAnimeResult, 0, len(results))
	seen := make(map[string]bool)

	for _, r := range results {
		// Usa o título original do torrent (já limpo pelo parser)
		title := r.Title

		// Se o título contém HTML, está errado - ignora
		if strings.Contains(title, "<") || strings.Contains(title, ">") {
			fmt.Printf("[TorBox] AVISO: Título com HTML ignorado: %s\n", title)
			continue
		}

		// Extrai nome limpo para display
		displayName := extractAnimeName(title)
		if displayName == "" {
			displayName = title
		}

		// Evita duplicatas pelo hash
		if r.Hash == "" || seen[r.Hash] {
			continue
		}
		seen[r.Hash] = true

		animes = append(animes, TorBoxAnimeResult{
			Title:    displayName,
			FullName: title,
			Quality:  r.Quality,
			Size:     r.Size,
			Seeds:    r.Seeds,
			Cached:   r.Cached,
			Magnet:   r.Magnet,
			Hash:     r.Hash,
			Source:   "TorBox",
		})
	}

	fmt.Printf("[TorBox] Encontrados %d animes\n", len(animes))
	return animes
}

// TorBoxAnimeResult resultado de busca do TorBox para o frontend
type TorBoxAnimeResult struct {
	Title    string `json:"title"`
	FullName string `json:"fullName"`
	Quality  string `json:"quality"`
	Size     string `json:"size"`
	Seeds    int    `json:"seeds"`
	Cached   bool   `json:"cached"`
	Magnet   string `json:"magnet"`
	Hash     string `json:"hash"`
	Source   string `json:"source"`
}

// extractAnimeName extrai o nome limpo do anime do título do torrent
func extractAnimeName(torrentTitle string) string {
	title := torrentTitle

	// Decodifica HTML entities
	title = strings.ReplaceAll(title, "&amp;", "&")
	title = strings.ReplaceAll(title, "&lt;", "<")
	title = strings.ReplaceAll(title, "&gt;", ">")
	title = strings.ReplaceAll(title, "&quot;", "\"")
	title = strings.ReplaceAll(title, "&#39;", "'")
	title = strings.ReplaceAll(title, "&nbsp;", " ")

	// Remove URL encoded characters PRIMEIRO
	title = strings.ReplaceAll(title, "%20", " ")
	title = strings.ReplaceAll(title, "%5B", "[")
	title = strings.ReplaceAll(title, "%5D", "]")
	title = strings.ReplaceAll(title, "%28", "(")
	title = strings.ReplaceAll(title, "%29", ")")
	title = strings.ReplaceAll(title, "%2B", "+")
	title = strings.ReplaceAll(title, "%3A", ":")

	// Remove tags HTML se houver
	if strings.Contains(title, "<") {
		// Remove tudo entre < e >
		for strings.Contains(title, "<") && strings.Contains(title, ">") {
			start := strings.Index(title, "<")
			end := strings.Index(title, ">")
			if start >= 0 && end > start {
				title = title[:start] + title[end+1:]
			} else {
				break
			}
		}
	}

	// Remove tags entre colchetes no início [SubGroup] [1080p] etc
	for len(title) > 0 && title[0] == '[' {
		end := 0
		for i, c := range title {
			if c == ']' {
				end = i
				break
			}
		}
		if end > 0 {
			title = strings.TrimSpace(title[end+1:])
		} else {
			break
		}
	}

	// Remove tags entre parênteses no final (1080p) (Complete) etc
	for strings.Contains(title, "(") {
		start := strings.LastIndex(title, "(")
		end := strings.LastIndex(title, ")")
		if start >= 0 && end > start {
			title = strings.TrimSpace(title[:start])
		} else {
			break
		}
	}

	// Remove qualidade e outras tags do final
	parts := strings.Split(title, " - ")
	if len(parts) > 0 {
		title = parts[0]
	}

	// Remove números de episódio no final (S01, E01, 01-12, etc)
	title = strings.TrimSpace(title)

	return title
}

// TorBoxStreamMagnet inicia streaming de um magnet/hash específico
// Retorna a URL do stream para reproduzir no player
func (a *App) TorBoxStreamMagnet(magnet string, hash string) string {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return ""
	}

	fmt.Printf("[TorBox] Iniciando stream - Hash: %s\n", hash)

	magnetPreview := magnet
	if len(magnet) > 100 {
		magnetPreview = magnet[:100] + "..."
	}
	fmt.Printf("[TorBox] Magnet recebido (len=%d): %s\n", len(magnet), magnetPreview)

	// Valida magnet
	if magnet == "" || !strings.HasPrefix(magnet, "magnet:") {
		fmt.Printf("[TorBox] ERRO: Magnet inválido! Valor: '%s'\n", magnet)
		return ""
	}

	// Primeiro verifica se está em cache para stream instantâneo
	if hash != "" {
		cached, err := torboxClient.CheckCached(a.ctx, []string{hash})
		if err == nil && cached[hash] {
			fmt.Println("[TorBox] Hash está em cache!")

			// Busca nos torrents existentes
			torrents, err := torboxClient.GetTorrents(a.ctx)
			if err == nil {
				for _, t := range torrents {
					if t.Hash == hash && len(t.Files) > 0 {
						// Pega o maior arquivo (provavelmente o vídeo)
						bestFile := t.Files[0]
						for _, f := range t.Files {
							if f.Size > bestFile.Size {
								bestFile = f
							}
						}
						link, err := torboxClient.GetDownloadLink(a.ctx, t.ID, bestFile.ID)
						if err == nil && link != "" {
							fmt.Printf("[TorBox] Stream URL (cache): %s\n", link)
							return link
						}
					}
				}
			}
		}
	}

	// Se não está em cache, adiciona o magnet e aguarda
	fmt.Println("[TorBox] Adicionando magnet ao TorBox...")
	torrent, err := torboxClient.AddMagnet(a.ctx, magnet, true)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao adicionar magnet: %v\n", err)
		return ""
	}

	if torrent != nil && torrent.ID > 0 {
		fmt.Printf("[TorBox] Torrent adicionado, ID: %d\n", torrent.ID)

		// Busca o torrent com arquivos
		torrents, err := torboxClient.GetTorrents(a.ctx)
		if err == nil {
			for _, t := range torrents {
				if t.ID == torrent.ID && len(t.Files) > 0 {
					// Pega o maior arquivo (provavelmente o vídeo)
					bestFile := t.Files[0]
					for _, f := range t.Files {
						if f.Size > bestFile.Size {
							bestFile = f
						}
					}
					link, err := torboxClient.GetDownloadLink(a.ctx, t.ID, bestFile.ID)
					if err == nil && link != "" {
						fmt.Printf("[TorBox] Stream URL: %s\n", link)
						return link
					}
				}
			}
		}
	}

	fmt.Println("[TorBox] Não foi possível obter stream URL")
	return ""
}

// TorBoxGetTorrentFiles obtém os arquivos de vídeo de um torrent para seleção
// Retorna lista de arquivos ordenados por episódio
func (a *App) TorBoxGetTorrentFiles(magnet string, hash string) *TorBoxTorrentInfo {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return nil
	}

	fmt.Printf("[TorBox] Obtendo arquivos - Hash: %s\n", hash)

	// Valida magnet
	if magnet == "" || !strings.HasPrefix(magnet, "magnet:") {
		fmt.Printf("[TorBox] ERRO: Magnet inválido!\n")
		return nil
	}

	var torrentID int
	var torrentInfo *torbox.Torrent

	// Primeiro verifica se já existe nos torrents
	torrents, err := torboxClient.GetTorrents(a.ctx)
	if err == nil {
		for _, t := range torrents {
			if t.Hash == hash {
				torrentID = t.ID
				torrentInfo = &t
				fmt.Printf("[TorBox] Torrent encontrado: ID=%d, Files=%d\n", t.ID, len(t.Files))
				break
			}
		}
	}

	// Se não existe, adiciona o magnet
	if torrentID == 0 {
		fmt.Println("[TorBox] Adicionando magnet ao TorBox...")
		torrent, err := torboxClient.AddMagnet(a.ctx, magnet, true)
		if err != nil {
			fmt.Printf("[TorBox] Erro ao adicionar magnet: %v\n", err)
			return nil
		}

		if torrent != nil && torrent.ID > 0 {
			torrentID = torrent.ID

			// Busca novamente para pegar os arquivos
			torrents, err = torboxClient.GetTorrents(a.ctx)
			if err == nil {
				for _, t := range torrents {
					if t.ID == torrentID {
						torrentInfo = &t
						break
					}
				}
			}
		}
	}

	if torrentInfo == nil {
		fmt.Println("[TorBox] Não foi possível obter informações do torrent")
		return nil
	}

	// Filtra apenas arquivos de vídeo
	videoExts := []string{".mkv", ".mp4", ".avi", ".webm", ".mov", ".m4v"}
	var files []TorBoxFileInfo

	for _, f := range torrentInfo.Files {
		nameLower := strings.ToLower(f.Name)
		isVideo := false
		for _, ext := range videoExts {
			if strings.HasSuffix(nameLower, ext) {
				isVideo = true
				break
			}
		}

		if isVideo {
			episode, season := extractEpisodeNumber(f.Name)
			files = append(files, TorBoxFileInfo{
				ID:         f.ID,
				TorrentID:  torrentID,
				Name:       f.Name,
				ShortName:  f.ShortName,
				Size:       f.Size,
				SizeStr:    formatFileSize(f.Size),
				Episode:    episode,
				Season:     season,
				IsPlayable: true,
			})
		}
	}

	// Ordena por episódio
	sort.Slice(files, func(i, j int) bool {
		if files[i].Season != files[j].Season {
			return files[i].Season < files[j].Season
		}
		return files[i].Episode < files[j].Episode
	})

	fmt.Printf("[TorBox] Encontrados %d arquivos de vídeo\n", len(files))

	return &TorBoxTorrentInfo{
		ID:           torrentID,
		Hash:         torrentInfo.Hash,
		Name:         torrentInfo.Name,
		Progress:     torrentInfo.Progress,
		Status:       torrentInfo.Status,
		Cached:       torrentInfo.Cached,
		Files:        files,
		TotalSize:    torrentInfo.Size,
		TotalSizeStr: formatFileSize(torrentInfo.Size),
	}
}

// TorBoxGetFileStreamURL obtém a URL de streaming de um arquivo específico
func (a *App) TorBoxGetFileStreamURL(torrentID int, fileID int) string {
	if torboxClient == nil {
		fmt.Println("[TorBox] Cliente não inicializado")
		return ""
	}

	fmt.Printf("[TorBox] Obtendo stream URL: Torrent=%d, File=%d\n", torrentID, fileID)

	link, err := torboxClient.GetDownloadLink(a.ctx, torrentID, fileID)
	if err != nil {
		fmt.Printf("[TorBox] Erro ao obter link: %v\n", err)
		return ""
	}

	fmt.Printf("[TorBox] Stream URL: %s\n", link)
	return link
}

// extractEpisodeNumber extrai o número do episódio e temporada do nome do arquivo
func extractEpisodeNumber(filename string) (episode int, season int) {
	episode = 0
	season = 1

	// Padrões comuns de episódio
	patterns := []string{
		`[Ss](\d+)[Ee](\d+)`,                // S01E01
		`[Ss]eason\s*(\d+).*[Ee]p?\s*(\d+)`, // Season 1 Episode 1
		`-\s*(\d+)\s*-`,                     // - 01 -
		`\s+(\d{2,3})\s*[\[\(]`,             // 01 [ ou 01 (
		`[Ee][Pp]?\.?\s*(\d+)`,              // E01, Ep01, Ep.01
		`\s+(\d{2,3})\s*[vV]?\d*\s*[\.\[]`,  // 01. ou 01v2.
		`\[(\d{2,3})\]`,                     // [01]
		`\s+-\s*(\d+)\s+`,                   // - 01
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(filename)
		if len(matches) >= 2 {
			if len(matches) >= 3 {
				// Tem season e episode
				if s, err := strconv.Atoi(matches[1]); err == nil {
					season = s
				}
				if e, err := strconv.Atoi(matches[2]); err == nil {
					episode = e
				}
			} else {
				// Só episode
				if e, err := strconv.Atoi(matches[1]); err == nil {
					episode = e
				}
			}
			if episode > 0 {
				break
			}
		}
	}

	return
}

// formatFileSize formata o tamanho do arquivo para exibição
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// GetAnimePoster busca a capa de um anime pelo título usando Jikan/AniList/Kitsu
func (a *App) GetAnimePoster(title string) string {
	if title == "" {
		return ""
	}

	fmt.Printf("[Poster] Buscando capa para: %s\n", title)

	// Usa a função multi-source que busca em Jikan, AniList e Kitsu
	poster := jikan.FetchPosterMultiSource(title)

	if poster != "" {
		fmt.Printf("[Poster] Encontrada: %s\n", poster)
	} else {
		fmt.Println("[Poster] Nenhuma capa encontrada")
	}

	return poster
}

// PosterResult resultado da busca batch de posters
type PosterResult struct {
	Title  string `json:"title"`
	Poster string `json:"poster"`
}

// GetAnimePostersMulti busca capas de múltiplos animes em paralelo (3 workers)
// Retorna um mapa de título -> URL da capa
func (a *App) GetAnimePostersMulti(titles []string) map[string]string {
	if len(titles) == 0 {
		return nil
	}

	fmt.Printf("[Poster Multi] Buscando %d capas (3 workers)...\n", len(titles))
	start := time.Now()

	// Usa 3 workers (otimizado para 6 itens, evita sobrecarga de rede)
	results := jikan.FetchPostersMultiThread(titles, 3)

	// Converte para map
	posterMap := make(map[string]string)
	found := 0
	for _, r := range results {
		if r.Poster != "" {
			posterMap[r.Title] = r.Poster
			found++
		}
	}

	fmt.Printf("[Poster Multi] OK: %d/%d capas em %v\n", found, len(titles), time.Since(start))

	return posterMap
}

// IsTorBoxConfigured verifica se o TorBox está configurado
func (a *App) IsTorBoxConfigured() bool {
	return torboxClient != nil
}

// TorBoxGetFilesFromMagnet obtém arquivos de um magnet link
func (a *App) TorBoxGetFilesFromMagnet(magnet string) *TorBoxTorrentInfo {
	if torboxClient == nil {
		return nil
	}
	// Extrai hash do magnet
	hash := extractHashFromMagnet(magnet)
	if hash == "" {
		return nil
	}
	return a.TorBoxGetTorrentFiles(magnet, hash)
}

// TorBoxGetStreamLinkLocal obtém link de streaming local do TorBox
func (a *App) TorBoxGetStreamLinkLocal(torrentID int, fileID int) string {
	return a.TorBoxGetFileStreamURL(torrentID, fileID)
}

// extractHashFromMagnet extrai o hash de um magnet link
func extractHashFromMagnet(magnet string) string {
	re := regexp.MustCompile(`btih:([a-fA-F0-9]{40})`)
	matches := re.FindStringSubmatch(magnet)
	if len(matches) > 1 {
		return strings.ToLower(matches[1])
	}
	return ""
}

// EpisodeFileInfo representa um arquivo individual dentro de um episódio
type EpisodeFileInfo struct {
	NomeOriginal string   `json:"nome_original"`
	Tags         []string `json:"tags"`
	Quality      string   `json:"qualidade,omitempty"`
	SubGroup     string   `json:"subgrupo,omitempty"`
}

// EpisodeInfo representa um episódio com seus arquivos
type EpisodeInfo struct {
	IDEpisodio             int               `json:"id_episodio"`
	Temporada              int               `json:"temporada"`
	TituloExibicaoLimpo    string            `json:"titulo_exibicao_limpo"`
	TituloEpisodioCompleto string            `json:"titulo_episodio_completo,omitempty"`
	ArquivosDisponiveis    []EpisodeFileInfo `json:"arquivos_disponiveis"`
}

// EpisodeParseResultInfo resultado do parsing para o frontend
type EpisodeParseResultInfo struct {
	NomeAnime      string        `json:"nome_anime"`
	Episodios      []EpisodeInfo `json:"episodios"`
	TotalEpisodios int           `json:"total_episodios"`
}

// ParseEpisodeFilenamesV2 analisa e agrupa nomes de arquivos de episódios
// Usa o parser robusto v2 com agrupamento estrito por número de episódio
func (a *App) ParseEpisodeFilenamesV2(filenames []string) EpisodeParseResultInfo {
	result := utils.ParseEpisodeFiles(filenames)

	// Converte para tipos do frontend
	episodes := make([]EpisodeInfo, 0, len(result.Episodios))
	for _, ep := range result.Episodios {
		files := make([]EpisodeFileInfo, 0, len(ep.ArquivosDisponiveis))
		for _, f := range ep.ArquivosDisponiveis {
			files = append(files, EpisodeFileInfo{
				NomeOriginal: f.NomeOriginal,
				Tags:         f.Tags,
				Quality:      f.Quality,
				SubGroup:     f.SubGroup,
			})
		}
		episodes = append(episodes, EpisodeInfo{
			IDEpisodio:             ep.IDEpisodio,
			Temporada:              ep.Temporada,
			TituloExibicaoLimpo:    ep.TituloExibicaoLimpo,
			TituloEpisodioCompleto: ep.TituloEpisodioCompleto,
			ArquivosDisponiveis:    files,
		})
	}

	return EpisodeParseResultInfo{
		NomeAnime:      result.NomeAnime,
		Episodios:      episodes,
		TotalEpisodios: result.TotalEpisodios,
	}
}

// ParseEpisodeFilenamesJSON retorna o resultado em JSON string
func (a *App) ParseEpisodeFilenamesJSON(filenames []string) string {
	return utils.ParseEpisodeFilesJSON(filenames)
}

// ============================================================================
// MÉTODOS LEGADOS (mantidos para compatibilidade)
// ============================================================================

// ParsedEpisodeInfo representa um episódio parseado para o frontend (LEGADO)
type ParsedEpisodeInfo struct {
	OriginalName string `json:"original"`
	Title        string `json:"titulo"`
	Season       int    `json:"temporada"`
	Episode      int    `json:"episodio"`
	Quality      string `json:"qualidade"`
	ReleaseGroup string `json:"tag"`
	AudioType    string `json:"audio_tipo"`
}

// GroupedEpisodeInfo representa um episódio agrupado para o frontend (LEGADO)
type GroupedEpisodeInfo struct {
	EpisodeNumber int                 `json:"episodio_numero"`
	Season        int                 `json:"temporada"`
	CleanTitle    string              `json:"titulo_limpo"`
	Files         []ParsedEpisodeInfo `json:"arquivos"`
}

// EpisodeGroupResultInfo resultado do agrupamento para o frontend (LEGADO)
type EpisodeGroupResultInfo struct {
	AnimeName string               `json:"anime_nome"`
	Episodes  []GroupedEpisodeInfo `json:"episodios"`
	Total     int                  `json:"total_episodios"`
}

// ParseEpisodeFilenames analisa e agrupa nomes de arquivos (LEGADO - use ParseEpisodeFilenamesV2)
func (a *App) ParseEpisodeFilenames(filenames []string) EpisodeGroupResultInfo {
	parser := utils.NewEpisodeParser()
	result := parser.FullProcess(filenames)

	// Converte para tipos do frontend
	episodes := make([]GroupedEpisodeInfo, 0, len(result.Episodes))
	for _, ep := range result.Episodes {
		files := make([]ParsedEpisodeInfo, 0, len(ep.Files))
		for _, f := range ep.Files {
			files = append(files, ParsedEpisodeInfo{
				OriginalName: f.OriginalName,
				Title:        f.Title,
				Season:       f.Season,
				Episode:      f.Episode,
				Quality:      f.Quality,
				ReleaseGroup: f.ReleaseGroup,
				AudioType:    f.AudioType,
			})
		}
		episodes = append(episodes, GroupedEpisodeInfo{
			EpisodeNumber: ep.EpisodeNumber,
			Season:        ep.Season,
			CleanTitle:    ep.CleanTitle,
			Files:         files,
		})
	}

	return EpisodeGroupResultInfo{
		AnimeName: result.AnimeName,
		Episodes:  episodes,
		Total:     result.Total,
	}
}

// ParseSingleEpisodeFilename analisa um único nome de arquivo (LEGADO)
func (a *App) ParseSingleEpisodeFilename(filename string) ParsedEpisodeInfo {
	parser := utils.NewEpisodeParser()
	p := parser.Parse(filename)

	return ParsedEpisodeInfo{
		OriginalName: p.OriginalName,
		Title:        p.Title,
		Season:       p.Season,
		Episode:      p.Episode,
		Quality:      p.Quality,
		ReleaseGroup: p.ReleaseGroup,
		AudioType:    p.AudioType,
	}
}

// FormatEpisodeTitle formata título do episódio para exibição
// Exemplo: "May I Ask for One Final Thing - Episódio 12"
func (a *App) FormatEpisodeTitle(filename string) string {
	return utils.ParseSingleToFormattedTitle(filename)
}
