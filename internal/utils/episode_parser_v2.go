// Package utils fornece funções utilitárias comuns
package utils

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ============================================================================
// TIPOS DE DADOS - Estrutura de saída conforme especificação
// ============================================================================

// EpisodeFile representa um arquivo individual dentro de um episódio
type EpisodeFile struct {
	NomeOriginal string   `json:"nome_original"`
	Tags         []string `json:"tags"`
	Quality      string   `json:"qualidade,omitempty"`
	SubGroup     string   `json:"subgrupo,omitempty"`
}

// Episode representa um episódio agrupado com todos seus arquivos
type Episode struct {
	IDEpisodio             int           `json:"id_episodio"`
	Temporada              int           `json:"temporada"`
	TituloExibicaoLimpo    string        `json:"titulo_exibicao_limpo"`
	TituloEpisodioCompleto string        `json:"titulo_episodio_completo,omitempty"`
	ArquivosDisponiveis    []EpisodeFile `json:"arquivos_disponiveis"`
}

// EpisodeParseResult representa o resultado completo do parsing
type EpisodeParseResult struct {
	NomeAnime      string    `json:"nome_anime"`
	Episodios      []Episode `json:"episodios"`
	TotalEpisodios int       `json:"total_episodios"`
}

// ============================================================================
// PARSER - Implementação robusta
// ============================================================================

// RobustEpisodeParser é o parser principal com regras estritas
type RobustEpisodeParser struct {
	// Tags técnicas a serem removidas
	technicalTags    []string
	technicalTagsSet map[string]bool

	// Padrões compilados
	patterns map[string]*regexp.Regexp
}

// NewRobustEpisodeParser cria um novo parser com configuração robusta
func NewRobustEpisodeParser() *RobustEpisodeParser {
	tags := []string{
		// Qualidade
		"2160p", "1080p", "720p", "480p", "360p", "4K", "UHD", "FHD", "HD",
		// Fontes
		"WEB-DL", "WEBDL", "WEB-RIP", "WEBRIP", "WEBRip", "BluRay", "BDRip", "BRRip",
		"HDTV", "DVDRip", "CR", "Crunchyroll", "Funimation", "Netflix",
		// Codec
		"x264", "x265", "H.264", "H 264", "H264", "H.265", "H 265", "H265",
		"HEVC", "AVC", "10bit", "10-bit", "8bit",
		// Audio
		"AAC", "AAC2.0", "AAC2 0", "AAC5.1", "FLAC", "AC3", "DTS", "EAC3",
		"DUAL", "DUAL-AUDIO", "MULTi", "MULTI", "MULTI-AUDIO", "DualAudio",
		"2.0", "2 0", "5.1", "5 1",
		// Release Groups
		"VARYG", "-VARYG", "SubsPlease", "Erai-raws", "HorribleSubs",
		"ASW", "Judas", "EMBER", "YuiSubs", "Anime Time", "ToonsHub",
		"SSA", "Tsundere", "DeadFish", "Golumpa", "SCY", "Beatrice-Raws",
		// Outros
		"BATCH", "COMPLETE", "Complete", "REPACK", "PROPER", "v2", "v3",
		"RAW", "ENG", "ENGSUB", "HARDSUB", "SOFTSUB", "ASS", "SRT",
	}

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[strings.ToLower(tag)] = true
	}

	p := &RobustEpisodeParser{
		technicalTags:    tags,
		technicalTagsSet: tagSet,
		patterns:         make(map[string]*regexp.Regexp),
	}
	p.compilePatterns()
	return p
}

func (p *RobustEpisodeParser) compilePatterns() {
	// Padrão principal S01E01 - mais preciso e prioritário
	p.patterns["sxex"] = regexp.MustCompile(`(?i)\bS(\d{1,2})E(\d{1,4})\b`)

	// Padrão colado ao título: TitleE01, ThingE11
	// Captura: grupo1=título, grupo2=número do episódio
	p.patterns["title_glued_e"] = regexp.MustCompile(`([A-Za-z][A-Za-z\s']*?)E(\d{1,4})\b`)

	// Padrão - XX - (traço número traço)
	p.patterns["dash_episode"] = regexp.MustCompile(`\s+-\s*(\d{1,4})\s*[-\[\(]`)

	// Padrão Episode/Episódio/Ep XX
	p.patterns["ep_word"] = regexp.MustCompile(`(?i)\b(?:Episode|Episódio|Ep\.?)\s*(\d{1,4})\b`)

	// Padrão [XX] ou (XX) - número em colchetes/parênteses
	p.patterns["bracket_num"] = regexp.MustCompile(`[\[\(](\d{2,4})[\]\)]`)

	// Qualidade
	p.patterns["quality"] = regexp.MustCompile(`(?i)\b(2160|1080|720|480|360)p?\b`)

	// Subgrupo [Nome] no início
	p.patterns["subgroup"] = regexp.MustCompile(`^\[([^\]]+)\]`)

	// Release group no final -NOME
	p.patterns["release_group"] = regexp.MustCompile(`-([A-Za-z][A-Za-z0-9]{2,})(?:\s*$|\s+)`)

	// Extensão de arquivo
	p.patterns["extension"] = regexp.MustCompile(`(?i)\.(mkv|mp4|avi|webm|m3u8|ts)$`)

	// Texto após o número do episódio (possível título do episódio)
	// Captura texto entre Exx e as tags técnicas
	p.patterns["episode_title"] = regexp.MustCompile(`(?i)E\d+\s+([A-Z][A-Za-z\s]+?)(?:\s+(?:CR|MULTi|DUAL|1080|720|WEB|$))`)
}

// ParsedFile representa um arquivo parseado internamente
type parsedFile struct {
	originalName string
	episode      int
	season       int
	animeTitle   string
	episodeTitle string
	quality      string
	subGroup     string
	releaseGroup string
	tags         []string
}

// ParseFiles processa uma lista de nomes de arquivos e retorna resultado estruturado
func (p *RobustEpisodeParser) ParseFiles(filenames []string) EpisodeParseResult {
	// Fase 1: Parse individual de cada arquivo
	parsed := make([]parsedFile, 0, len(filenames))
	for _, filename := range filenames {
		if pf := p.parseOne(filename); pf != nil && pf.episode > 0 {
			parsed = append(parsed, *pf)
		}
	}

	// Fase 2: Agrupamento ESTRITO por episódio
	episodeMap := make(map[int]*Episode)

	for _, pf := range parsed {
		ep, exists := episodeMap[pf.episode]
		if !exists {
			ep = &Episode{
				IDEpisodio:             pf.episode,
				Temporada:              pf.season,
				TituloExibicaoLimpo:    pf.animeTitle,
				TituloEpisodioCompleto: pf.episodeTitle,
				ArquivosDisponiveis:    make([]EpisodeFile, 0),
			}
			episodeMap[pf.episode] = ep
		}

		// Adiciona arquivo ao episódio CORRETO
		ef := EpisodeFile{
			NomeOriginal: pf.originalName,
			Tags:         pf.tags,
			Quality:      pf.quality,
			SubGroup:     pf.subGroup,
		}
		ep.ArquivosDisponiveis = append(ep.ArquivosDisponiveis, ef)

		// Atualiza título do episódio se disponível
		if pf.episodeTitle != "" && ep.TituloEpisodioCompleto == "" {
			ep.TituloEpisodioCompleto = pf.episodeTitle
		}

		// Escolhe o melhor título do anime
		if len(pf.animeTitle) > len(ep.TituloExibicaoLimpo) {
			ep.TituloExibicaoLimpo = pf.animeTitle
		}
	}

	// Fase 3: Converter mapa para slice ordenado
	episodes := make([]Episode, 0, len(episodeMap))
	for _, ep := range episodeMap {
		episodes = append(episodes, *ep)
	}

	// Ordena por número do episódio
	sort.Slice(episodes, func(i, j int) bool {
		if episodes[i].Temporada != episodes[j].Temporada {
			return episodes[i].Temporada < episodes[j].Temporada
		}
		return episodes[i].IDEpisodio < episodes[j].IDEpisodio
	})

	// Fase 4: Determinar nome do anime (título mais comum/longo)
	animeName := p.determineBestAnimeName(episodes)

	return EpisodeParseResult{
		NomeAnime:      animeName,
		Episodios:      episodes,
		TotalEpisodios: len(episodes),
	}
}

// parseOne processa um único nome de arquivo
func (p *RobustEpisodeParser) parseOne(filename string) *parsedFile {
	pf := &parsedFile{
		originalName: filename,
		season:       1,
		episode:      0,
		tags:         make([]string, 0),
	}

	working := filename

	// Remove extensão
	working = p.patterns["extension"].ReplaceAllString(working, "")

	// Extrai subgrupo [Nome]
	if matches := p.patterns["subgroup"].FindStringSubmatch(working); len(matches) >= 2 {
		pf.subGroup = matches[1]
		working = strings.TrimPrefix(working, "["+matches[1]+"]")
		working = strings.TrimSpace(working)
	}

	// Extrai qualidade
	if matches := p.patterns["quality"].FindStringSubmatch(working); len(matches) >= 2 {
		pf.quality = matches[1] + "p"
		pf.tags = append(pf.tags, pf.quality)
	}

	// Extrai release group
	if matches := p.patterns["release_group"].FindStringSubmatch(working); len(matches) >= 2 {
		group := matches[1]
		if !p.isTechnicalTag(group) || group == "VARYG" {
			pf.releaseGroup = group
			pf.tags = append(pf.tags, group)
		}
	}

	// ========================================
	// EXTRAÇÃO DO EPISÓDIO - Ordem de prioridade
	// ========================================

	episodeExtracted := false

	// 1. Tenta S01E01 (formato mais confiável)
	if matches := p.patterns["sxex"].FindStringSubmatch(working); len(matches) >= 3 {
		if s, err := strconv.Atoi(matches[1]); err == nil {
			pf.season = s
		}
		if e, err := strconv.Atoi(matches[2]); err == nil {
			pf.episode = e
			episodeExtracted = true
		}
	}

	// 2. Tenta padrão colado: TitleE01 (CRÍTICO para o problema reportado)
	if !episodeExtracted {
		if matches := p.patterns["title_glued_e"].FindStringSubmatch(working); len(matches) >= 3 {
			if e, err := strconv.Atoi(matches[2]); err == nil && e > 0 && e < 2000 {
				pf.episode = e
				episodeExtracted = true
			}
		}
	}

	// 3. Tenta Episode/Ep XX
	if !episodeExtracted {
		if matches := p.patterns["ep_word"].FindStringSubmatch(working); len(matches) >= 2 {
			if e, err := strconv.Atoi(matches[1]); err == nil {
				pf.episode = e
				episodeExtracted = true
			}
		}
	}

	// 4. Tenta - XX -
	if !episodeExtracted {
		if matches := p.patterns["dash_episode"].FindStringSubmatch(working); len(matches) >= 2 {
			if e, err := strconv.Atoi(matches[1]); err == nil && e > 0 && e < 2000 {
				pf.episode = e
				episodeExtracted = true
			}
		}
	}

	// 5. Tenta [XX]
	if !episodeExtracted {
		if matches := p.patterns["bracket_num"].FindStringSubmatch(working); len(matches) >= 2 {
			if e, err := strconv.Atoi(matches[1]); err == nil && e > 0 && e < 2000 {
				pf.episode = e
				episodeExtracted = true
			}
		}
	}

	// Se não extraiu episódio, retorna nil
	if pf.episode == 0 {
		return nil
	}

	// ========================================
	// EXTRAÇÃO DO TÍTULO
	// ========================================

	pf.animeTitle = p.extractAnimeTitle(working, pf.episode)
	pf.episodeTitle = p.extractEpisodeTitle(working)

	// Extrai tags adicionais
	pf.tags = append(pf.tags, p.extractTags(working)...)

	return pf
}

// extractAnimeTitle extrai o título do anime limpo
func (p *RobustEpisodeParser) extractAnimeTitle(input string, episode int) string {
	title := input

	// Remove subgrupo do início
	title = p.patterns["subgroup"].ReplaceAllString(title, "")

	// Para padrão colado (TitleE01), extrai apenas a parte antes do E
	if matches := p.patterns["title_glued_e"].FindStringSubmatch(input); len(matches) >= 2 {
		title = strings.TrimSpace(matches[1])
	} else {
		// Para S01E01, extrai tudo antes do padrão
		if idx := p.patterns["sxex"].FindStringIndex(title); idx != nil {
			title = title[:idx[0]]
		}
	}

	// Remove tags técnicas
	title = p.cleanTechnicalTags(title)

	// Limpeza final
	title = p.finalCleanup(title)

	return title
}

// extractEpisodeTitle extrai o título específico do episódio (se disponível)
func (p *RobustEpisodeParser) extractEpisodeTitle(input string) string {
	// Procura texto após Exx que parece ser título do episódio
	// Exemplo: "ThingE11 As These Appear Undercooked..." -> "As These Appear Undercooked"

	// Padrão 1: Depois de Exx com texto que começa com maiúscula
	re := regexp.MustCompile(`(?i)E\d+\s+([A-Z][A-Za-z\s,'!?]+?)(?:\s*\.{3}|\s+(?:CR|MULTi|DUAL|1080|720|WEB|\[|$))`)
	if matches := re.FindStringSubmatch(input); len(matches) >= 2 {
		epTitle := strings.TrimSpace(matches[1])
		// Remove reticências
		epTitle = strings.TrimSuffix(epTitle, "...")
		epTitle = strings.TrimSpace(epTitle)
		if len(epTitle) > 5 && !p.isTechnicalTag(epTitle) {
			return epTitle
		}
	}

	// Padrão 2: Título entre S01E01 e tags
	re2 := regexp.MustCompile(`(?i)S\d+E\d+\s+([A-Z][A-Za-z\s,'!?]+?)(?:\s+(?:1080|720|WEB|CR|\[|$))`)
	if matches := re2.FindStringSubmatch(input); len(matches) >= 2 {
		epTitle := strings.TrimSpace(matches[1])
		if len(epTitle) > 5 && !p.isTechnicalTag(epTitle) {
			return epTitle
		}
	}

	return ""
}

// extractTags extrai tags técnicas encontradas
func (p *RobustEpisodeParser) extractTags(input string) []string {
	tags := make([]string, 0)
	inputLower := strings.ToLower(input)

	checkTags := []string{"WEB-DL", "CR", "DUAL", "MULTi", "REPACK", "BATCH"}
	for _, tag := range checkTags {
		if strings.Contains(inputLower, strings.ToLower(tag)) {
			tags = append(tags, tag)
		}
	}

	return tags
}

// cleanTechnicalTags remove todas as tags técnicas do título
func (p *RobustEpisodeParser) cleanTechnicalTags(title string) string {
	for _, tag := range p.technicalTags {
		// Case insensitive, word boundary
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(tag) + `\b`)
		title = re.ReplaceAllString(title, " ")
	}

	// Remove padrões numéricos órfãos (2 0, 5 1, etc)
	title = regexp.MustCompile(`\b\d\s+\d\b`).ReplaceAllString(title, " ")

	return title
}

// finalCleanup faz a limpeza final do título
func (p *RobustEpisodeParser) finalCleanup(title string) string {
	// Remove caracteres especiais
	title = regexp.MustCompile(`[\[\]\(\)\{\}]`).ReplaceAllString(title, " ")
	title = regexp.MustCompile(`\s*-+\s*$`).ReplaceAllString(title, "")
	title = regexp.MustCompile(`\s*-+\s*`).ReplaceAllString(title, " ")
	title = regexp.MustCompile(`\s*_+\s*`).ReplaceAllString(title, " ")
	title = regexp.MustCompile(`\.+`).ReplaceAllString(title, " ")

	// Remove múltiplos espaços
	title = regexp.MustCompile(`\s{2,}`).ReplaceAllString(title, " ")

	// Trim
	title = strings.TrimSpace(title)

	// Remove caracteres especiais do final
	title = strings.TrimRight(title, " -_.")

	return title
}

// isTechnicalTag verifica se é uma tag técnica conhecida
func (p *RobustEpisodeParser) isTechnicalTag(s string) bool {
	return p.technicalTagsSet[strings.ToLower(s)]
}

// determineBestAnimeName escolhe o melhor nome de anime baseado nos episódios
func (p *RobustEpisodeParser) determineBestAnimeName(episodes []Episode) string {
	if len(episodes) == 0 {
		return ""
	}

	// Conta frequência de cada título
	titleCount := make(map[string]int)
	for _, ep := range episodes {
		normalized := strings.ToLower(strings.TrimSpace(ep.TituloExibicaoLimpo))
		if normalized != "" {
			titleCount[normalized]++
		}
	}

	// Encontra o mais frequente
	bestTitle := ""
	bestCount := 0
	for title, count := range titleCount {
		if count > bestCount || (count == bestCount && len(title) > len(bestTitle)) {
			bestCount = count
			bestTitle = title
		}
	}

	// Retorna com capitalização original
	for _, ep := range episodes {
		if strings.ToLower(ep.TituloExibicaoLimpo) == bestTitle {
			return ep.TituloExibicaoLimpo
		}
	}

	return episodes[0].TituloExibicaoLimpo
}

// ToJSON converte o resultado para JSON formatado
func (r *EpisodeParseResult) ToJSON() string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
}

// ToEpisodesJSON retorna apenas os episódios como JSON
func (r *EpisodeParseResult) ToEpisodesJSON() string {
	data, err := json.MarshalIndent(r.Episodios, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
}

// ============================================================================
// FUNÇÕES DE CONVENIÊNCIA
// ============================================================================

// ParseEpisodeFiles é a função principal de conveniência
func ParseEpisodeFiles(filenames []string) EpisodeParseResult {
	parser := NewRobustEpisodeParser()
	return parser.ParseFiles(filenames)
}

// ParseEpisodeFilesJSON retorna o resultado diretamente em JSON
func ParseEpisodeFilesJSON(filenames []string) string {
	result := ParseEpisodeFiles(filenames)
	return result.ToJSON()
}
