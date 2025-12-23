package manga

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// MangaLivreBlogClient é o cliente para fazer scraping do mangalivre.blog
type MangaLivreBlogClient struct {
	baseURL    string
	httpClient *http.Client
	cache      map[string]interface{}
	cacheTTL   time.Duration
}

// NewMangaLivreBlogClient cria um novo cliente para mangalivre.blog
func NewMangaLivreBlogClient() *MangaLivreBlogClient {
	return &MangaLivreBlogClient{
		baseURL: "https://mangalivre.blog",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
		cache:    make(map[string]interface{}),
		cacheTTL: 10 * time.Minute,
	}
}

// GetSourceName retorna o nome da fonte
func (c *MangaLivreBlogClient) GetSourceName() string {
	return "MangaLivre.blog"
}

// makeRequest faz uma requisição HTTP com headers apropriados
func (c *MangaLivreBlogClient) makeRequest(urlStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Referer", c.baseURL)
	req.Header.Set("Cache-Control", "no-cache")

	return c.httpClient.Do(req)
}

// ============== FUNÇÕES DE LISTAGEM ==============

// GetAllMangas retorna todos os mangás do site com paginação
func (c *MangaLivreBlogClient) GetAllMangas(page int) ([]Manga, int, error) {
	pageURL := fmt.Sprintf("%s/manga/", c.baseURL)
	if page > 1 {
		pageURL = fmt.Sprintf("%s/manga/page/%d/", c.baseURL, page)
	}

	fmt.Printf("[MangaLivreBlog] Buscando mangás da página %d: %s\n", page, pageURL)

	resp, err := c.makeRequest(pageURL)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao acessar MangaLivre.blog: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	var mangas []Manga
	totalPages := 1

	// Extrai total de páginas procurando por links de paginação
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, "/manga/page/") {
			re := regexp.MustCompile(`/page/(\d+)`)
			if matches := re.FindStringSubmatch(href); len(matches) > 1 {
				if p, err := strconv.Atoi(matches[1]); err == nil && p > totalPages {
					totalPages = p
				}
			}
		}
	})

	// Busca os cards de mangá na listagem principal
	mangas = c.extractMangasFromListPage(doc)

	fmt.Printf("[MangaLivreBlog] Página %d: encontrou %d mangás, total de páginas: %d\n", page, len(mangas), totalPages)
	return mangas, totalPages, nil
}

// GetPopularMangas retorna os mangás populares
func (c *MangaLivreBlogClient) GetPopularMangas() ([]Manga, error) {
	pageURL := fmt.Sprintf("%s/manga/?m_orderby=views", c.baseURL)
	return c.fetchMangaList(pageURL, "populares")
}

// GetLatestUpdates retorna os mangás com atualizações recentes
func (c *MangaLivreBlogClient) GetLatestUpdates() ([]Manga, error) {
	pageURL := fmt.Sprintf("%s/manga/?m_orderby=latest", c.baseURL)
	return c.fetchMangaList(pageURL, "últimas atualizações")
}

// fetchMangaList busca lista de mangás de uma URL
func (c *MangaLivreBlogClient) fetchMangaList(pageURL, listType string) ([]Manga, error) {
	fmt.Printf("[MangaLivreBlog] Buscando %s: %s\n", listType, pageURL)

	resp, err := c.makeRequest(pageURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar MangaLivre.blog: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	mangas := c.extractMangasFromListPage(doc)
	fmt.Printf("[MangaLivreBlog] %s: encontrou %d mangás\n", listType, len(mangas))
	return mangas, nil
}

// extractMangasFromListPage extrai mangás de uma página de listagem
func (c *MangaLivreBlogClient) extractMangasFromListPage(doc *goquery.Document) []Manga {
	var mangas []Manga
	seenURLs := make(map[string]bool)

	// MangaLivre.blog usa SlimeRead theme - busca títulos em h3 ou h2 com links
	doc.Find("h3 a[href*='/manga/'], h2 a[href*='/manga/']").Each(func(i int, a *goquery.Selection) {
		href, exists := a.Attr("href")
		if !exists {
			return
		}

		href = strings.TrimSpace(href)
		// Ignora links de capítulos
		if strings.Contains(href, "/capitulo") || strings.Contains(href, "/chapter") {
			return
		}

		href = normalizeURL(href, c.baseURL)
		if seenURLs[href] {
			return
		}

		title := strings.TrimSpace(a.Text())
		if title == "" || len(title) < 2 {
			return
		}

		// Tenta achar imagem no parent container
		parent := a.Closest("article, div, .manga-item, .post")
		imgSrc := ""
		if parent.Length() > 0 {
			img := parent.Find("img").First()
			imgSrc = getImageSrc(img)
		}

		seenURLs[href] = true
		mangas = append(mangas, Manga{
			ID:    extractMangaID(href),
			Title: title,
			Image: normalizeImageURL(imgSrc, c.baseURL),
			URL:   href,
		})
	})

	if len(mangas) > 0 {
		return mangas
	}

	// Fallback: busca em artigos
	doc.Find("article, .manga-item").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a[href*='/manga/']").First()
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		href = strings.TrimSpace(href)
		if strings.Contains(href, "/capitulo") || strings.Contains(href, "/chapter") {
			return
		}

		href = normalizeURL(href, c.baseURL)
		if seenURLs[href] {
			return
		}

		// Busca título
		title := s.Find("h2, h3, h4, .title").First().Text()
		title = strings.TrimSpace(title)
		if title == "" || len(title) < 2 {
			return
		}

		// Busca imagem
		img := s.Find("img").First()
		imgSrc := getImageSrc(img)

		seenURLs[href] = true
		mangas = append(mangas, Manga{
			ID:    extractMangaID(href),
			Title: title,
			Image: normalizeImageURL(imgSrc, c.baseURL),
			URL:   href,
		})
	})

	return mangas
}

// ============== BUSCA ==============

// SearchManga busca mangás por termo
func (c *MangaLivreBlogClient) SearchManga(query string) ([]Manga, error) {
	searchURL := fmt.Sprintf("%s/?s=%s&post_type=wp-manga", c.baseURL, url.QueryEscape(query))

	fmt.Printf("[MangaLivreBlog] Buscando mangás com query '%s': %s\n", query, searchURL)

	resp, err := c.makeRequest(searchURL)
	if err != nil {
		return nil, fmt.Errorf("erro na busca: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	mangas := c.extractMangasFromListPage(doc)
	fmt.Printf("[MangaLivreBlog] Busca por '%s' retornou %d resultados\n", query, len(mangas))
	return mangas, nil
}

// ============== DETALHES ==============

// GetMangaDetails obtém detalhes completos de um mangá
func (c *MangaLivreBlogClient) GetMangaDetails(mangaURL string) (*Manga, error) {
	fmt.Printf("[MangaLivreBlog] Obtendo detalhes de: %s\n", mangaURL)

	resp, err := c.makeRequest(mangaURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar mangá: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	manga := &Manga{
		URL: mangaURL,
		ID:  extractMangaID(mangaURL),
	}

	// Título - No SlimeRead theme, o título do mangá geralmente está em um h1 que não seja "MANGALIVRE"
	// Ou no próprio slug da URL. Vamos usar o slug formatado.
	manga.Title = formatMangaTitle(extractMangaID(mangaURL))

	// Tenta pegar título melhor do breadcrumb ou meta
	doc.Find("h1, .manga-title, .entry-title").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Ignora se for o título do site
		if text != "" && !strings.EqualFold(text, "MANGALIVRE") && !strings.EqualFold(text, "mangalivre") && len(text) > 3 {
			// Pega o primeiro título válido encontrado
			if manga.Title == formatMangaTitle(extractMangaID(mangaURL)) {
				manga.Title = text
			}
		}
	})

	// Imagem de capa
	coverImg := doc.Find(".summary_image img, img.wp-post-image, .manga-cover img, .manga-thumb img").First()
	manga.Image = normalizeImageURL(getImageSrc(coverImg), c.baseURL)

	// Descrição - Sinopse
	doc.Find(".summary__content, .description-summary, .manga-excerpt, .manga-description, h3:contains('Sinopse')").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "Sinopse") {
			// Pega o próximo elemento
			desc := s.Next().Text()
			if desc != "" {
				manga.Description = strings.TrimSpace(desc)
			}
		}
	})

	// Fallback: pega texto após "Sinopse"
	if manga.Description == "" {
		html, _ := doc.Html()
		if idx := strings.Index(html, "Sinopse"); idx != -1 {
			// Tenta pegar uma descrição genérica
			desc := doc.Find("p").First().Text()
			manga.Description = strings.TrimSpace(desc)
		}
	}

	// Status - Busca "Em Lançamento" ou "Completo"
	fullText := doc.Text()
	lowerText := strings.ToLower(fullText)
	if strings.Contains(lowerText, "em lançamento") || strings.Contains(lowerText, "ongoing") {
		manga.Status = "Em Lançamento"
	} else if strings.Contains(lowerText, "completo") || strings.Contains(lowerText, "completed") || strings.Contains(lowerText, "finalizado") {
		manga.Status = "Completo"
	}

	// Gêneros - busca links com /genero/ ou /genre/
	doc.Find("a[href*='/genero/'], a[href*='/genre/']").Each(func(i int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		if genre != "" && !containsString(manga.Genres, genre) {
			manga.Genres = append(manga.Genres, genre)
		}
	})

	// Autor
	doc.Find("a[href*='/manga-author/'], a[href*='/author/']").First().Each(func(i int, s *goquery.Selection) {
		manga.Author = strings.TrimSpace(s.Text())
	})

	fmt.Printf("[MangaLivreBlog] Detalhes obtidos: %s\n", manga.Title)
	return manga, nil
}

// formatMangaTitle formata o slug do mangá para um título legível
func formatMangaTitle(slug string) string {
	// Substitui hífens por espaços
	title := strings.ReplaceAll(slug, "-", " ")
	// Capitaliza cada palavra
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// ============== CAPÍTULOS ==============

// GetChapters obtém a lista de capítulos de um mangá
func (c *MangaLivreBlogClient) GetChapters(mangaURL string) ([]MangaChapter, error) {
	fmt.Printf("[MangaLivreBlog] Obtendo capítulos de: %s\n", mangaURL)

	resp, err := c.makeRequest(mangaURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar mangá: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	mangaID := extractMangaID(mangaURL)
	mangaName := strings.TrimSpace(doc.Find("h1").First().Text())

	var chapters []MangaChapter
	seenURLs := make(map[string]bool)

	// MangaLivre.blog usa URLs como: /capitulo/nome-do-manga-capitulo-123/
	doc.Find("a[href*='/capitulo/']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		href = strings.TrimSpace(href)
		if !strings.HasPrefix(href, "http") {
			href = c.baseURL + href
		}

		// Ignora se já vimos esta URL
		if seenURLs[href] {
			return
		}

		// Verifica se a URL contém o mangaID (parte do slug)
		if !strings.Contains(strings.ToLower(href), strings.ToLower(mangaID)) {
			return
		}

		seenURLs[href] = true

		chapter := c.parseChapterFromURL(href, mangaID, mangaName)
		if chapter != nil {
			chapters = append(chapters, *chapter)
		}
	})

	// ORDENA CAPÍTULOS DO PRIMEIRO AO ÚLTIMO
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].NumberFloat < chapters[j].NumberFloat
	})

	fmt.Printf("[MangaLivreBlog] Encontrados %d capítulos para %s\n", len(chapters), mangaName)

	if len(chapters) > 0 {
		fmt.Printf("[MangaLivreBlog] Primeiro: Cap %s (%.1f), Último: Cap %s (%.1f)\n",
			chapters[0].Number, chapters[0].NumberFloat,
			chapters[len(chapters)-1].Number, chapters[len(chapters)-1].NumberFloat)
	}

	return chapters, nil
}

// parseChapterFromURL extrai informações de um capítulo a partir da URL
func (c *MangaLivreBlogClient) parseChapterFromURL(href, mangaID, mangaName string) *MangaChapter {
	lowerHref := strings.ToLower(href)

	// URL format: /capitulo/nome-do-manga-capitulo-123/
	// Padrão: capitulo-1, capitulo-01, capitulo-1-5, capitulo-extra
	re := regexp.MustCompile(`(?:capitulo|chapter)-(\d+)(?:[.-](\d+|extra|final))?`)
	matches := re.FindStringSubmatch(lowerHref)

	var chapterNum int
	var numFloat float64
	var chapterStr string

	if len(matches) >= 2 {
		chapterNum, _ = strconv.Atoi(matches[1])
		numFloat = float64(chapterNum)
		chapterStr = fmt.Sprintf("%d", chapterNum)

		// Suporta decimais como capitulo-3-5 -> 3.5
		if len(matches) > 2 && matches[2] != "" {
			if decimal, err := strconv.Atoi(matches[2]); err == nil {
				numFloat += float64(decimal) / 10.0
				chapterStr = fmt.Sprintf("%d.%s", chapterNum, matches[2])
			} else if matches[2] == "extra" || matches[2] == "final" {
				numFloat += 0.5
				chapterStr = fmt.Sprintf("%d %s", chapterNum, strings.ToUpper(matches[2]))
			}
		}
	} else if strings.Contains(lowerHref, "/completo") {
		chapterNum = 1
		numFloat = 1.0
		chapterStr = "Completo"
	} else if strings.Contains(lowerHref, "/oneshot") {
		chapterNum = 1
		numFloat = 1.0
		chapterStr = "Oneshot"
	} else {
		return nil
	}

	if chapterNum == 0 && chapterStr == "" {
		return nil
	}

	title := fmt.Sprintf("Capítulo %s", chapterStr)
	if chapterStr == "Completo" || chapterStr == "Oneshot" {
		title = chapterStr
	}

	return &MangaChapter{
		Number:      chapterStr,
		NumberFloat: numFloat,
		Title:       title,
		URL:         href,
		Date:        "",
		MangaID:     mangaID,
		MangaName:   mangaName,
	}
}

// ============== PÁGINAS DO CAPÍTULO ==============

// GetChapterPages obtém as páginas (imagens) de um capítulo
func (c *MangaLivreBlogClient) GetChapterPages(chapterURL string) ([]MangaPage, error) {
	fmt.Printf("[MangaLivreBlog] Buscando páginas de: %s\n", chapterURL)

	resp, err := c.makeRequest(chapterURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar capítulo: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	var pages []MangaPage
	seenURLs := make(map[string]bool)

	// MangaLivre.blog - As imagens do mangá tem alt="Página X"
	// Primeiro busca imagens com alt contendo "Página"
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		alt, _ := s.Attr("alt")
		if !strings.Contains(strings.ToLower(alt), "página") && !strings.Contains(strings.ToLower(alt), "pagina") {
			return
		}

		imgSrc := getImageSrc(s)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && strings.Contains(imgSrc, "wp-content/uploads") {
			seenURLs[imgSrc] = true
			pages = append(pages, MangaPage{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, c.baseURL),
			})
		}
	})

	// Se encontrou páginas pelo método alt, retorna
	if len(pages) > 0 {
		fmt.Printf("[MangaLivreBlog] Encontrou %d páginas (método alt='Página')\n", len(pages))
		return pages, nil
	}

	// Fallback: busca todas as imagens em wp-content/uploads
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		imgSrc := getImageSrc(s)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && c.isValidMangaImage(imgSrc) {
			seenURLs[imgSrc] = true
			pages = append(pages, MangaPage{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, c.baseURL),
			})
		}
	})

	fmt.Printf("[MangaLivreBlog] Total de páginas encontradas: %d\n", len(pages))
	return pages, nil
}

// isValidMangaImage verifica se é uma imagem válida de mangá
func (c *MangaLivreBlogClient) isValidMangaImage(imgSrc string) bool {
	lowerSrc := strings.ToLower(imgSrc)

	// Ignora imagens de UI
	if strings.Contains(lowerSrc, "logo") ||
		strings.Contains(lowerSrc, "icon") ||
		strings.Contains(lowerSrc, "avatar") ||
		strings.Contains(lowerSrc, "banner") ||
		strings.Contains(lowerSrc, "gravatar") ||
		strings.Contains(lowerSrc, "loading") ||
		strings.Contains(lowerSrc, "placeholder") ||
		strings.Contains(lowerSrc, "ads") ||
		strings.Contains(lowerSrc, "advertisement") {
		return false
	}

	// Aceita imagens de uploads (formato do WordPress)
	if strings.Contains(lowerSrc, "wp-content/uploads") {
		return true
	}

	// Aceita extensões comuns de imagem
	return strings.HasSuffix(lowerSrc, ".webp") ||
		strings.HasSuffix(lowerSrc, ".jpg") ||
		strings.HasSuffix(lowerSrc, ".jpeg") ||
		strings.HasSuffix(lowerSrc, ".png")
}

// ============== GÊNEROS ==============

// GetMangasByGenre retorna mangás de um gênero específico
func (c *MangaLivreBlogClient) GetMangasByGenre(genre string) ([]Manga, error) {
	genreSlug := normalizeGenreSlug(genre)
	genreURL := fmt.Sprintf("%s/genero/%s/", c.baseURL, genreSlug)

	fmt.Printf("[MangaLivreBlog] Buscando mangás do gênero '%s': %s\n", genre, genreURL)

	resp, err := c.makeRequest(genreURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar gênero: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	mangas := c.extractMangasFromListPage(doc)

	for i := range mangas {
		if !containsString(mangas[i].Genres, genre) {
			mangas[i].Genres = append(mangas[i].Genres, genre)
		}
	}

	fmt.Printf("[MangaLivreBlog] Gênero '%s' retornou %d mangás\n", genre, len(mangas))
	return mangas, nil
}

// GetGenres retorna a lista de gêneros disponíveis
func (c *MangaLivreBlogClient) GetGenres() ([]string, error) {
	resp, err := c.makeRequest(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar MangaLivre.blog: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	var genres []string
	seenGenres := make(map[string]bool)

	doc.Find("a[href*='/genero/'], a[href*='/genre/']").Each(func(i int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		re := regexp.MustCompile(`\s*\(\d+\)\s*$`)
		genre = re.ReplaceAllString(genre, "")
		genre = strings.TrimSpace(genre)

		if genre != "" && !seenGenres[genre] && len(genre) > 1 {
			seenGenres[genre] = true
			genres = append(genres, genre)
		}
	})

	fmt.Printf("[MangaLivreBlog] Encontrou %d gêneros\n", len(genres))
	return genres, nil
}
