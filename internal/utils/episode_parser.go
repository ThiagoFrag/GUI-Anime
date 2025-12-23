// Package utils fornece funções utilitárias comuns
package utils

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ParsedEpisode representa um episódio parseado de um nome de arquivo
type ParsedEpisode struct {
	OriginalName string `json:"original"`
	Title        string `json:"titulo"`
	Season       int    `json:"temporada"`
	Episode      int    `json:"episodio"`
	Quality      string `json:"qualidade"`
	ReleaseGroup string `json:"tag"`
	AudioType    string `json:"audio_tipo"`
	SubGroup     string `json:"subgrupo"`
}

// GroupedEpisode representa um episódio agrupado com múltiplos arquivos
type GroupedEpisode struct {
	EpisodeNumber int             `json:"episodio_numero"`
	Season        int             `json:"temporada"`
	CleanTitle    string          `json:"titulo_limpo"`
	Files         []ParsedEpisode `json:"arquivos"`
}

// EpisodeGroupResult representa o resultado do agrupamento
type EpisodeGroupResult struct {
	AnimeName string           `json:"anime_nome"`
	Episodes  []GroupedEpisode `json:"episodios"`
	Total     int              `json:"total_episodios"`
}

// EpisodeParser contém configuração e métodos para parsing de episódios
type EpisodeParser struct {
	// Tags técnicas a serem removidas do título
	TechnicalTags []string
	// Padrões regex para extração
	patterns map[string]*regexp.Regexp
}

// NewEpisodeParser cria uma nova instância do parser com configuração padrão
func NewEpisodeParser() *EpisodeParser {
	ep := &EpisodeParser{
		TechnicalTags: []string{
			// Qualidade
			"2160p", "1080p", "720p", "480p", "360p", "4K", "UHD", "FHD", "HD",
			// Fontes
			"WEB-DL", "WEBDL", "WEB-RIP", "WEBRIP", "WEBRip", "BluRay", "BDRip", "BRRip",
			"HDTV", "DVDRip", "DVDRip", "HDTV", "CR", "Crunchyroll", "Funimation",
			// Codec
			"x264", "x265", "H.264", "H 264", "H264", "H.265", "H 265", "H265",
			"HEVC", "AVC", "10bit", "10-bit", "8bit",
			// Audio
			"AAC", "AAC2.0", "AAC2 0", "AAC5.1", "FLAC", "AC3", "DTS", "EAC3",
			"DUAL", "DUAL-AUDIO", "MULTI", "MULTI-AUDIO", "DualAudio",
			// Release Groups comuns
			"VARYG", "-VARYG", "SubsPlease", "Erai-raws", "HorribleSubs",
			"ASW", "Judas", "EMBER", "YuiSubs", "Anime Time", "ToonsHub",
			"SSA", "Tsundere", "DeadFish", "Golumpa", "SCY", "Beatrice-Raws",
			// Outros
			"BATCH", "COMPLETE", "Complete", "REPACK", "PROPER", "v2", "v3",
			"RAW", "ENG", "ENGSUB", "HARDSUB", "SOFTSUB", "ASS", "SRT",
		},
		patterns: make(map[string]*regexp.Regexp),
	}

	// Inicializa padrões regex
	ep.initPatterns()

	return ep
}

func (ep *EpisodeParser) initPatterns() {
	// Padrão para S01E01, S1E1, etc
	ep.patterns["sxex"] = regexp.MustCompile(`(?i)[Ss](\d{1,2})[Ee](\d{1,4})`)

	// Padrão para S01 E01 (com espaço)
	ep.patterns["sxex_space"] = regexp.MustCompile(`(?i)[Ss](\d{1,2})\s+[Ee](\d{1,4})`)

	// Padrão para Season 1 Episode 1
	ep.patterns["season_episode"] = regexp.MustCompile(`(?i)Season\s*(\d+).*?Episode\s*(\d+)`)

	// Padrão para Temporada 1 Episódio 1
	ep.patterns["temporada_episodio"] = regexp.MustCompile(`(?i)Temporada\s*(\d+).*?Epis[óo]dio\s*(\d+)`)

	// Padrão colado: TitleE01, NameE12 (título grudado no E + número)
	ep.patterns["title_exx"] = regexp.MustCompile(`([A-Za-z\s]+)[Ee](\d{1,4})\b`)

	// Padrão para - 01 - (traço número traço) - apenas números simples, não ranges
	ep.patterns["dash_number"] = regexp.MustCompile(`\s+-\s*(\d{1,4})\s*(?:-|$|\[)`)

	// Padrão para Episode 01 ou Ep 01 ou Ep.01
	ep.patterns["ep_number"] = regexp.MustCompile(`(?i)(?:Episode|Episódio|Ep\.?)\s*(\d{1,4})`)

	// Padrão para [01] ou (01)
	ep.patterns["bracket_number"] = regexp.MustCompile(`[\[\(](\d{2,4})[\]\)]`)

	// Padrão para número no final: " 01 " ou " 01."
	ep.patterns["space_number"] = regexp.MustCompile(`\s+(\d{2,4})(?:\s*[\.\[\(\-]|$)`)

	// Padrão para qualidade
	ep.patterns["quality"] = regexp.MustCompile(`(?i)(2160|1080|720|480|360)[pP]?`)

	// Padrão para subgrupo [Nome]
	ep.patterns["subgroup"] = regexp.MustCompile(`^\[([^\]]+)\]`)

	// Padrão para release group no final (-VARYG, -Judas, etc)
	ep.patterns["release_group"] = regexp.MustCompile(`-([A-Za-z][A-Za-z0-9]+)(?:\s|$|\.)`)

	// Remove extensão de arquivo
	ep.patterns["extension"] = regexp.MustCompile(`\.(mkv|mp4|avi|webm|m3u8)$`)

	// Remove ano entre parênteses
	ep.patterns["year"] = regexp.MustCompile(`\s*\(?(19|20)\d{2}\)?`)

	// Remove múltiplos espaços
	ep.patterns["multi_space"] = regexp.MustCompile(`\s{2,}`)

	// Remove caracteres especiais do final
	ep.patterns["trailing_special"] = regexp.MustCompile(`[\s\-_\.]+$`)

	// Remove WEB, DL, CR e outras tags comuns do título
	ep.patterns["web_tags"] = regexp.MustCompile(`(?i)\b(WEB|DL|CR|BD|TV|OVA|ONA|AAC|AVC|FLAC|MultiSub|x|x264|x265)\b`)

	// Remove range de episódios (080 ~ 500, 001-500, etc)
	ep.patterns["episode_range"] = regexp.MustCompile(`\d+\s*[-~]\s*\d+`)
}

// Parse analisa um nome de arquivo e extrai informações do episódio
func (ep *EpisodeParser) Parse(filename string) ParsedEpisode {
	result := ParsedEpisode{
		OriginalName: filename,
		Season:       1,
		Episode:      0,
	}

	// Remove extensão
	working := ep.patterns["extension"].ReplaceAllString(filename, "")

	// Extrai subgrupo [Nome] no início
	if matches := ep.patterns["subgroup"].FindStringSubmatch(working); len(matches) >= 2 {
		result.SubGroup = matches[1]
		working = strings.TrimPrefix(working, "["+matches[1]+"]")
		working = strings.TrimSpace(working)
	}

	// Extrai qualidade
	if matches := ep.patterns["quality"].FindStringSubmatch(working); len(matches) >= 2 {
		result.Quality = matches[1] + "p"
	}

	// Extrai release group do final
	if matches := ep.patterns["release_group"].FindStringSubmatch(working); len(matches) >= 2 {
		// Verifica se não é uma tag técnica comum
		group := matches[1]
		if !ep.isTechnicalTag(group) {
			result.ReleaseGroup = group
		}
	}

	// Detecta tipo de áudio
	workingLower := strings.ToLower(working)
	if strings.Contains(workingLower, "dual") {
		result.AudioType = "DUAL"
	} else if strings.Contains(workingLower, "multi") {
		result.AudioType = "MULTI"
	}

	// Extrai temporada e episódio - tenta múltiplos padrões em ordem de preferência
	seasonFound, episodeFound := false, false

	// Tenta S01E01
	if matches := ep.patterns["sxex"].FindStringSubmatch(working); len(matches) >= 3 {
		if s, err := strconv.Atoi(matches[1]); err == nil {
			result.Season = s
			seasonFound = true
		}
		if e, err := strconv.Atoi(matches[2]); err == nil {
			result.Episode = e
			episodeFound = true
		}
	}

	// Tenta S01 E01 (com espaço)
	if !episodeFound {
		if matches := ep.patterns["sxex_space"].FindStringSubmatch(working); len(matches) >= 3 {
			if s, err := strconv.Atoi(matches[1]); err == nil {
				result.Season = s
				seasonFound = true
			}
			if e, err := strconv.Atoi(matches[2]); err == nil {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Tenta Season 1 Episode 1
	if !episodeFound {
		if matches := ep.patterns["season_episode"].FindStringSubmatch(working); len(matches) >= 3 {
			if s, err := strconv.Atoi(matches[1]); err == nil {
				result.Season = s
				seasonFound = true
			}
			if e, err := strconv.Atoi(matches[2]); err == nil {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Tenta Temporada/Episódio
	if !episodeFound {
		if matches := ep.patterns["temporada_episodio"].FindStringSubmatch(working); len(matches) >= 3 {
			if s, err := strconv.Atoi(matches[1]); err == nil {
				result.Season = s
				seasonFound = true
			}
			if e, err := strconv.Atoi(matches[2]); err == nil {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Tenta padrão colado: TitleE01, NameE12
	if !episodeFound {
		if matches := ep.patterns["title_exx"].FindStringSubmatch(working); len(matches) >= 3 {
			if e, err := strconv.Atoi(matches[2]); err == nil {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Tenta Episode/Ep/Episódio
	if !episodeFound {
		if matches := ep.patterns["ep_number"].FindStringSubmatch(working); len(matches) >= 2 {
			if e, err := strconv.Atoi(matches[1]); err == nil {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Tenta - 01 -
	if !episodeFound {
		if matches := ep.patterns["dash_number"].FindStringSubmatch(working); len(matches) >= 2 {
			if e, err := strconv.Atoi(matches[1]); err == nil && e > 0 && e < 2000 {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Tenta [01] ou (01)
	if !episodeFound {
		if matches := ep.patterns["bracket_number"].FindStringSubmatch(working); len(matches) >= 2 {
			if e, err := strconv.Atoi(matches[1]); err == nil && e > 0 && e < 2000 {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Tenta número espaçado
	if !episodeFound {
		if matches := ep.patterns["space_number"].FindStringSubmatch(working); len(matches) >= 2 {
			if e, err := strconv.Atoi(matches[1]); err == nil && e > 0 && e < 2000 {
				result.Episode = e
				episodeFound = true
			}
		}
	}

	// Extrai título limpo
	result.Title = ep.cleanTitle(working, result.Season, result.Episode, seasonFound)

	return result
}

// cleanTitle limpa o título removendo tags técnicas e informações de episódio
func (ep *EpisodeParser) cleanTitle(working string, season, episode int, hasSeasonInfo bool) string {
	title := working

	// Remove subgrupo do início [Nome]
	title = ep.patterns["subgroup"].ReplaceAllString(title, "")

	// Remove padrões de temporada/episódio
	title = ep.patterns["sxex"].ReplaceAllString(title, " ")
	title = ep.patterns["sxex_space"].ReplaceAllString(title, " ")
	title = ep.patterns["season_episode"].ReplaceAllString(title, " ")
	title = ep.patterns["temporada_episodio"].ReplaceAllString(title, " ")
	title = ep.patterns["ep_number"].ReplaceAllString(title, " ")
	title = ep.patterns["dash_number"].ReplaceAllString(title, " ")
	title = ep.patterns["bracket_number"].ReplaceAllString(title, " ")

	// Trata padrão colado: "ThingE11" -> "Thing"
	if matches := ep.patterns["title_exx"].FindStringSubmatch(working); len(matches) >= 2 {
		// Se achou o padrão colado, extrai só o título
		potentialTitle := strings.TrimSpace(matches[1])
		if len(potentialTitle) > 3 {
			title = potentialTitle
		}
	}

	// Remove qualidade
	title = ep.patterns["quality"].ReplaceAllString(title, " ")

	// Remove release group
	title = ep.patterns["release_group"].ReplaceAllString(title, " ")

	// Remove ano
	title = ep.patterns["year"].ReplaceAllString(title, " ")

	// Remove tags técnicas
	for _, tag := range ep.TechnicalTags {
		// Case insensitive replace
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(tag) + `\b`)
		title = re.ReplaceAllString(title, " ")
	}

	// Limpa caracteres especiais isolados
	title = regexp.MustCompile(`[\[\]\(\)\{\}]`).ReplaceAllString(title, " ")
	title = regexp.MustCompile(`\s*-+\s*`).ReplaceAllString(title, " ")
	title = regexp.MustCompile(`\s*_+\s*`).ReplaceAllString(title, " ")
	title = regexp.MustCompile(`\.+`).ReplaceAllString(title, " ")

	// Remove WEB, DL, CR e outras tags técnicas restantes
	title = ep.patterns["web_tags"].ReplaceAllString(title, " ")

	// Remove range de episódios (080 ~ 500, 001-500)
	title = ep.patterns["episode_range"].ReplaceAllString(title, " ")

	// Limpa espaços múltiplos
	title = ep.patterns["multi_space"].ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	// Remove caracteres especiais do final
	title = ep.patterns["trailing_special"].ReplaceAllString(title, "")

	return title
}

// isTechnicalTag verifica se uma string é uma tag técnica conhecida
func (ep *EpisodeParser) isTechnicalTag(s string) bool {
	sLower := strings.ToLower(s)
	for _, tag := range ep.TechnicalTags {
		if strings.ToLower(tag) == sLower {
			return true
		}
	}
	return false
}

// ParseMultiple analisa múltiplos nomes de arquivo
func (ep *EpisodeParser) ParseMultiple(filenames []string) []ParsedEpisode {
	results := make([]ParsedEpisode, 0, len(filenames))
	for _, filename := range filenames {
		results = append(results, ep.Parse(filename))
	}
	return results
}

// GroupByEpisode agrupa episódios parseados pelo número do episódio
func (ep *EpisodeParser) GroupByEpisode(parsed []ParsedEpisode) []GroupedEpisode {
	// Mapa para agrupar: chave = "S{season}E{episode}"
	groups := make(map[string]*GroupedEpisode)

	for _, p := range parsed {
		key := "S" + strconv.Itoa(p.Season) + "E" + strconv.Itoa(p.Episode)

		if existing, ok := groups[key]; ok {
			// Adiciona ao grupo existente
			existing.Files = append(existing.Files, p)
		} else {
			// Cria novo grupo
			groups[key] = &GroupedEpisode{
				EpisodeNumber: p.Episode,
				Season:        p.Season,
				CleanTitle:    p.Title,
				Files:         []ParsedEpisode{p},
			}
		}
	}

	// Converte mapa para slice e ordena
	result := make([]GroupedEpisode, 0, len(groups))
	for _, g := range groups {
		// Escolhe o melhor título (mais longo, mais limpo)
		g.CleanTitle = ep.chooseBestTitle(g.Files)
		result = append(result, *g)
	}

	// Ordena por temporada e depois por episódio
	sort.Slice(result, func(i, j int) bool {
		if result[i].Season != result[j].Season {
			return result[i].Season < result[j].Season
		}
		return result[i].EpisodeNumber < result[j].EpisodeNumber
	})

	return result
}

// chooseBestTitle escolhe o melhor título entre os arquivos do grupo
func (ep *EpisodeParser) chooseBestTitle(files []ParsedEpisode) string {
	if len(files) == 0 {
		return ""
	}

	// Conta frequência dos títulos normalizados
	titleCounts := make(map[string]int)
	titleOriginal := make(map[string]string) // normalized -> original

	for _, f := range files {
		normalized := strings.ToLower(strings.TrimSpace(f.Title))
		if normalized == "" {
			continue
		}
		titleCounts[normalized]++
		// Mantém a versão original (com capitalização correta)
		if existing, ok := titleOriginal[normalized]; ok {
			// Prefere o mais longo (mais completo)
			if len(f.Title) > len(existing) {
				titleOriginal[normalized] = f.Title
			}
		} else {
			titleOriginal[normalized] = f.Title
		}
	}

	// Encontra o título mais frequente
	bestCount := 0
	bestNormalized := ""
	for normalized, count := range titleCounts {
		if count > bestCount || (count == bestCount && len(normalized) > len(bestNormalized)) {
			bestCount = count
			bestNormalized = normalized
		}
	}

	if original, ok := titleOriginal[bestNormalized]; ok {
		return original
	}

	return files[0].Title
}

// Deduplicate remove episódios duplicados (mesmo ep, mesma qualidade)
func (ep *EpisodeParser) Deduplicate(parsed []ParsedEpisode) []ParsedEpisode {
	seen := make(map[string]bool)
	result := make([]ParsedEpisode, 0, len(parsed))

	for _, p := range parsed {
		// Chave única: episódio + qualidade + audio
		key := strconv.Itoa(p.Season) + "x" + strconv.Itoa(p.Episode) + "-" + p.Quality + "-" + p.AudioType
		if !seen[key] {
			seen[key] = true
			result = append(result, p)
		}
	}

	return result
}

// FullProcess realiza o pipeline completo: parse -> deduplicate -> group
func (ep *EpisodeParser) FullProcess(filenames []string) EpisodeGroupResult {
	// Parse all
	parsed := ep.ParseMultiple(filenames)

	// Deduplicate
	deduped := ep.Deduplicate(parsed)

	// Group by episode
	grouped := ep.GroupByEpisode(deduped)

	// Determina nome do anime baseado no título mais comum
	animeName := ""
	if len(grouped) > 0 {
		animeName = grouped[0].CleanTitle
	}

	return EpisodeGroupResult{
		AnimeName: animeName,
		Episodes:  grouped,
		Total:     len(grouped),
	}
}

// ToJSON converte o resultado para JSON formatado
func (result *EpisodeGroupResult) ToJSON() string {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// FormatEpisodeTitle formata o título do episódio para exibição
// Exemplo: "May I Ask for One Final Thing - Episódio 12"
func FormatEpisodeTitle(title string, episode int, season int) string {
	if season > 1 {
		return title + " S" + strconv.Itoa(season) + " - Episódio " + strconv.Itoa(episode)
	}
	return title + " - Episódio " + strconv.Itoa(episode)
}

// NormalizeGluedTitle corrige títulos com número colado
// Exemplo: "ThingE11 As These Appear" -> "Thing - Episódio 11"
func NormalizeGluedTitle(input string) (title string, episode int) {
	re := regexp.MustCompile(`([A-Za-z\s]+)[Ee](\d{1,4})`)
	if matches := re.FindStringSubmatch(input); len(matches) >= 3 {
		title = strings.TrimSpace(matches[1])
		if e, err := strconv.Atoi(matches[2]); err == nil {
			episode = e
		}
		return
	}
	return input, 0
}

// ParseSingleToFormattedTitle função de conveniência para um único arquivo
func ParseSingleToFormattedTitle(filename string) string {
	parser := NewEpisodeParser()
	p := parser.Parse(filename)
	return FormatEpisodeTitle(p.Title, p.Episode, p.Season)
}
