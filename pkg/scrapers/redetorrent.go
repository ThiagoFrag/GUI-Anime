// pkg/scrapers/redetorrent.go
// Provider BR - RedeTorrent com gocolly
package scrapers

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// RedeTorrentProvider implementa scraping do RedeTorrent.com
type RedeTorrentProvider struct {
	baseURL string
	timeout time.Duration
}

// NewRedeTorrentProvider cria um novo provider do RedeTorrent
func NewRedeTorrentProvider() *RedeTorrentProvider {
	return &RedeTorrentProvider{
		baseURL: "https://redetorrent.com",
		timeout: 20 * time.Second,
	}
}

// Name retorna o nome do provider
func (r *RedeTorrentProvider) Name() string {
	return "RedeTorrent"
}

// IsAvailable verifica se o RedeTorrent está acessível
func (r *RedeTorrentProvider) IsAvailable(ctx context.Context) bool {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)
	c.SetRequestTimeout(10 * time.Second)

	available := false
	c.OnResponse(func(resp *colly.Response) {
		available = resp.StatusCode == 200
	})

	_ = c.Visit(r.baseURL)
	return available
}

// Search busca animes no RedeTorrent (estratégia otimizada)
func (r *RedeTorrentProvider) Search(ctx context.Context, query string) ([]AnimeResult, error) {
	var results []AnimeResult
	var mu sync.Mutex

	// Busca normalizada (slug) para paginação
	querySlug := strings.ToLower(query)
	querySlug = strings.ReplaceAll(querySlug, " ", "-")
	querySlug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(querySlug, "")

	// Prepara query para match
	queryLower := strings.ToLower(query)
	querySlugMatch := strings.ReplaceAll(queryLower, " ", "-")

	// Mapa para links únicos (evita duplicatas entre estratégias)
	contentLinks := make(map[string]bool)
	var linksMu sync.Mutex

	// ═══════════════════════════════════════════════════════════════════
	// ESTRATÉGIA 1: Paginação dinâmica /{slug}/1/, /{slug}/2/...
	// Melhor para animes com muitos resultados (ex: Naruto = 52 links)
	// ═══════════════════════════════════════════════════════════════════
	log.Printf("[RedeTorrent] Iniciando paginação dinâmica: /%s/N/", querySlug)

	for page := 1; page <= 10; page++ {
		pageURL := fmt.Sprintf("%s/%s/%d/", r.baseURL, querySlug, page)
		pageLinks := r.collectLinksFromPage(pageURL, querySlugMatch)

		if len(pageLinks) == 0 {
			log.Printf("[RedeTorrent] Página %d vazia, parando paginação", page)
			break
		}

		newCount := 0
		linksMu.Lock()
		for _, link := range pageLinks {
			if !contentLinks[link] {
				contentLinks[link] = true
				newCount++
			}
		}
		linksMu.Unlock()

		log.Printf("[RedeTorrent] /%s/%d/: +%d novos links (total: %d)", querySlug, page, newCount, len(contentLinks))
	}

	// ═══════════════════════════════════════════════════════════════════
	// ESTRATÉGIA 2: Busca principal /index.php?s=
	// Bom para animes sem paginação por slug (ex: Death Note)
	// ═══════════════════════════════════════════════════════════════════
	searchURL := fmt.Sprintf("%s/index.php?s=%s", r.baseURL, url.QueryEscape(query))
	log.Printf("[RedeTorrent] Busca principal: %s", searchURL)

	searchLinks := r.collectLinksFromPage(searchURL, querySlugMatch)
	newFromSearch := 0
	linksMu.Lock()
	for _, link := range searchLinks {
		if !contentLinks[link] {
			contentLinks[link] = true
			newFromSearch++
		}
	}
	linksMu.Unlock()
	log.Printf("[RedeTorrent] Busca: +%d novos links", newFromSearch)

	// Converte mapa para slice
	var linksList []string
	for link := range contentLinks {
		linksList = append(linksList, link)
	}

	log.Printf("[RedeTorrent] Total de links únicos: %d", len(linksList))

	// Limita para não sobrecarregar
	maxLinks := 30
	if len(linksList) > maxLinks {
		linksList = linksList[:maxLinks]
	}

	log.Printf("[RedeTorrent] Processando %d páginas de conteúdo...", len(linksList))

	// Visita cada página de conteúdo para extrair magnets
	sem := make(chan struct{}, 6) // Máximo 6 paralelos
	var wg sync.WaitGroup

	for _, link := range linksList {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(pageURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			pageResults := r.extractMagnetsFromPage(ctx, pageURL)
			if len(pageResults) > 0 {
				mu.Lock()
				results = append(results, pageResults...)
				mu.Unlock()
			}
		}(link)
	}

	wg.Wait()

	// Remove duplicatas
	results = deduplicateResults(results)
	log.Printf("[RedeTorrent] Total: %d resultados para '%s'", len(results), query)

	return results, nil
}

// collectLinksFromPage coleta links de conteúdo de uma página
func (r *RedeTorrentProvider) collectLinksFromPage(pageURL, querySlug string) []string {
	var links []string
	var mu sync.Mutex

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(10 * time.Second)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		hrefLower := strings.ToLower(href)

		// Links de conteúdo terminam com -download/
		if strings.Contains(href, "redetorrent.com/") &&
			strings.HasSuffix(href, "-download/") &&
			!strings.Contains(href, "index.php") &&
			!r.isPaginationURL(href) {

			// Verifica se o link contém o slug do anime
			if strings.Contains(hrefLower, querySlug) {
				mu.Lock()
				// Evita duplicatas
				found := false
				for _, l := range links {
					if l == href {
						found = true
						break
					}
				}
				if !found {
					links = append(links, href)
				}
				mu.Unlock()
			}
		}
	})

	_ = c.Visit(pageURL)
	c.Wait()

	return links
}

// isPaginationURL verifica se é URL de paginação (não de conteúdo)
func (r *RedeTorrentProvider) isPaginationURL(href string) bool {
	// URLs de paginação: /naruto/1/, /naruto/2/, etc.
	// Mas não: /naruto-1-temporada-download/
	parts := strings.Split(strings.TrimSuffix(href, "/"), "/")
	if len(parts) < 2 {
		return false
	}
	lastPart := parts[len(parts)-1]
	// Se o último segmento é só um número, é paginação
	for _, c := range lastPart {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(lastPart) <= 2 // Páginas são 1, 2, 3... não 10+
}

// extractMagnetsFromPage extrai magnets de uma página de conteúdo
func (r *RedeTorrentProvider) extractMagnetsFromPage(ctx context.Context, pageURL string) []AnimeResult {
	var results []AnimeResult
	var mu sync.Mutex

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(15 * time.Second)

	var pageTitle string
	var pageSize string
	var pageQuality string

	// Captura título da página
	c.OnHTML("h1.entry-title, h1.post-title, h1", func(e *colly.HTMLElement) {
		if pageTitle == "" {
			pageTitle = strings.TrimSpace(e.Text)
			// Limpa o título
			pageTitle = strings.ReplaceAll(pageTitle, " – Rede Torrent", "")
			pageTitle = strings.ReplaceAll(pageTitle, " - Rede Torrent", "")
			pageTitle = strings.ReplaceAll(pageTitle, "Download Torrent", "")
			pageTitle = strings.ReplaceAll(pageTitle, " via Torrent", "")
			pageTitle = strings.TrimSpace(pageTitle)
		}
	})

	// Captura metadados (tamanho, qualidade)
	c.OnHTML(".entry-content, .post-content, article", func(e *colly.HTMLElement) {
		text := e.Text

		// Extrai tamanho
		sizeRegex := regexp.MustCompile(`Tamanho:\s*([0-9.,]+\s*[GMTK]B)`)
		if match := sizeRegex.FindStringSubmatch(text); len(match) > 1 {
			pageSize = match[1]
		}

		// Extrai qualidade
		qualityRegex := regexp.MustCompile(`Qualidade:\s*([^\n]+)`)
		if match := qualityRegex.FindStringSubmatch(text); len(match) > 1 {
			pageQuality = strings.TrimSpace(match[1])
		}
	})

	// Procura magnets diretamente em links
	c.OnHTML("a[href^='magnet:']", func(e *colly.HTMLElement) {
		magnet := e.Attr("href")
		if magnet != "" {
			// Tenta pegar contexto do link (DUBLADO, LEGENDADO, etc)
			linkText := strings.ToUpper(e.Text)
			result := r.parseMagnetResultWithMeta(magnet, pageTitle, pageSize, pageQuality, linkText)
			if result != nil {
				mu.Lock()
				results = append(results, *result)
				mu.Unlock()
			}
		}
	})

	// Também procura magnets no HTML raw (alguns estão escondidos)
	c.OnResponse(func(resp *colly.Response) {
		html := string(resp.Body)
		magnets := r.extractMagnetsFromText(html)
		for _, magnet := range magnets {
			result := r.parseMagnetResultWithMeta(magnet, pageTitle, pageSize, pageQuality, "")
			if result != nil {
				mu.Lock()
				// Evita duplicatas
				found := false
				for _, existing := range results {
					if existing.Hash == result.Hash {
						found = true
						break
					}
				}
				if !found {
					results = append(results, *result)
				}
				mu.Unlock()
			}
		}
	})

	c.OnError(func(resp *colly.Response, err error) {
		log.Printf("[RedeTorrent] Erro em %s: %v", pageURL, err)
	})

	_ = c.Visit(pageURL)
	c.Wait()

	if len(results) > 0 {
		log.Printf("[RedeTorrent] %s -> %d magnets", pageURL, len(results))
	}

	return results
}

// extractMagnetsFromText extrai magnets de texto
func (r *RedeTorrentProvider) extractMagnetsFromText(text string) []string {
	var magnets []string
	magnetRegex := regexp.MustCompile(`magnet:\?xt=urn:btih:[a-zA-Z0-9]+[^"'\s<>]*`)
	matches := magnetRegex.FindAllString(text, -1)

	for _, m := range matches {
		// Limpa caracteres extras
		m = strings.Split(m, `"`)[0]
		m = strings.Split(m, `'`)[0]
		m = strings.Split(m, `<`)[0]
		magnets = append(magnets, m)
	}

	return magnets
}

// parseMagnetResultWithMeta cria um AnimeResult com metadados extras
func (r *RedeTorrentProvider) parseMagnetResultWithMeta(magnet, pageTitle, pageSize, pageQuality, linkContext string) *AnimeResult {
	hash := extractHashFromMagnet(magnet)
	if hash == "" {
		return nil
	}

	// Extrai nome do magnet se disponível
	title := pageTitle
	if strings.Contains(magnet, "&dn=") {
		parts := strings.Split(magnet, "&dn=")
		if len(parts) > 1 {
			dn := strings.Split(parts[1], "&")[0]
			decoded, err := url.QueryUnescape(dn)
			if err == nil && decoded != "" {
				// Se o dn for mais descritivo, usa ele
				if len(decoded) > 10 {
					title = decoded
				}
			}
		}
	}

	if title == "" {
		title = "RedeTorrent - " + hash[:8]
	}

	// Adiciona contexto do link se relevante
	if linkContext != "" {
		if strings.Contains(linkContext, "DUBLADO") && !strings.Contains(strings.ToUpper(title), "DUBLADO") {
			title = title + " [Dublado]"
		} else if strings.Contains(linkContext, "LEGENDADO") && !strings.Contains(strings.ToUpper(title), "LEGENDADO") {
			title = title + " [Legendado]"
		} else if strings.Contains(linkContext, "DUAL") && !strings.Contains(strings.ToUpper(title), "DUAL") {
			title = title + " [Dual Áudio]"
		}
	}

	// Detecta qualidade
	quality := detectQuality(title)
	if quality == "" && pageQuality != "" {
		quality = detectQuality(pageQuality)
	}
	if quality == "" {
		quality = "720p" // Padrão BR
	}

	// Usa tamanho da página se disponível
	size := pageSize

	// Calcula BRScore baseado no conteúdo
	brScore := 100 // RedeTorrent é BR por padrão
	titleUpper := strings.ToUpper(title)
	if strings.Contains(titleUpper, "DUAL") || strings.Contains(titleUpper, "DUBLADO") {
		brScore = 150 // Bonus para dual audio/dublado
	}

	return &AnimeResult{
		Title:    title,
		Magnet:   magnet,
		Hash:     hash,
		Size:     size,
		Seeders:  0, // Não disponível
		Leechers: 0,
		Source:   "RedeTorrent",
		Quality:  quality,
		BRScore:  brScore,
	}
}

// extractHashFromMagnet extrai o hash de um magnet link
func extractHashFromMagnet(magnet string) string {
	// Formato: magnet:?xt=urn:btih:HASH&...
	if !strings.Contains(magnet, "btih:") {
		return ""
	}

	parts := strings.Split(magnet, "btih:")
	if len(parts) < 2 {
		return ""
	}

	hash := parts[1]
	// Remove parâmetros após o hash
	if idx := strings.Index(hash, "&"); idx != -1 {
		hash = hash[:idx]
	}

	return strings.ToUpper(hash)
}
