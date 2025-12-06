package manga

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Manga representa um mangá com suas informações
type Manga struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Image       string   `json:"image"`
	URL         string   `json:"url"`
	LatestChap  string   `json:"latestChapter"`
	Genres      []string `json:"genres"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Rating      float64  `json:"rating"`
	Views       int      `json:"views"`
	Author      string   `json:"author"`
}

// MangaChapter representa um capítulo de mangá
type MangaChapter struct {
	Number      string  `json:"number"`
	NumberFloat float64 `json:"numberFloat"` // Para ordenação correta
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Date        string  `json:"date"`
	MangaID     string  `json:"mangaId"`
	MangaName   string  `json:"mangaName"`
}

// MangaPage representa uma página de mangá (imagem)
type MangaPage struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// MangaClient é o cliente para fazer scraping do MangaLivre
type MangaClient struct {
	baseURL    string
	httpClient *http.Client
	cache      map[string]interface{}
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
}

// NewMangaClient cria um novo cliente de mangá
func NewMangaClient() *MangaClient {
	return &MangaClient{
		baseURL: "https://mangalivre.to",
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

// makeRequest faz uma requisição HTTP com headers apropriados
func (c *MangaClient) makeRequest(urlStr string) (*http.Response, error) {
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
func (c *MangaClient) GetAllMangas(page int) ([]Manga, int, error) {
	pageURL := fmt.Sprintf("%s/manga/", c.baseURL)
	if page > 1 {
		pageURL = fmt.Sprintf("%s/manga/page/%d/", c.baseURL, page)
	}

	fmt.Printf("[MangaClient] Buscando mangás da página %d: %s\n", page, pageURL)

	resp, err := c.makeRequest(pageURL)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao acessar MangaLivre: %v", err)
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

	fmt.Printf("[MangaClient] Página %d: encontrou %d mangás, total de páginas: %d\n", page, len(mangas), totalPages)
	return mangas, totalPages, nil
}

// GetPopularMangas retorna os mangás populares (ordenados por views)
func (c *MangaClient) GetPopularMangas() ([]Manga, error) {
	pageURL := fmt.Sprintf("%s/manga/?m_orderby=views", c.baseURL)
	return c.fetchMangaList(pageURL, "populares")
}

// GetLatestUpdates retorna os mangás com atualizações recentes
func (c *MangaClient) GetLatestUpdates() ([]Manga, error) {
	pageURL := fmt.Sprintf("%s/manga/?m_orderby=latest", c.baseURL)
	return c.fetchMangaList(pageURL, "últimas atualizações")
}

// fetchMangaList busca lista de mangás de uma URL
func (c *MangaClient) fetchMangaList(pageURL, listType string) ([]Manga, error) {
	fmt.Printf("[MangaClient] Buscando %s: %s\n", listType, pageURL)

	resp, err := c.makeRequest(pageURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar MangaLivre: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	mangas := c.extractMangasFromListPage(doc)
	fmt.Printf("[MangaClient] %s: encontrou %d mangás\n", listType, len(mangas))
	return mangas, nil
}

// extractMangasFromListPage extrai mangás de uma página de listagem
func (c *MangaClient) extractMangasFromListPage(doc *goquery.Document) []Manga {
	var mangas []Manga
	seenURLs := make(map[string]bool)

	// Método 1: Estrutura do Madara theme (.page-item-detail)
	doc.Find(".page-item-detail").Each(func(i int, s *goquery.Selection) {
		manga := c.extractMangaFromCard(s)
		if manga.Title != "" && manga.URL != "" && !seenURLs[manga.URL] {
			seenURLs[manga.URL] = true
			mangas = append(mangas, manga)
		}
	})

	if len(mangas) > 0 {
		return mangas
	}

	// Método 2: Estrutura c-tabs-item
	doc.Find(".c-tabs-item__content").Each(func(i int, s *goquery.Selection) {
		manga := c.extractMangaFromCard(s)
		if manga.Title != "" && manga.URL != "" && !seenURLs[manga.URL] {
			seenURLs[manga.URL] = true
			mangas = append(mangas, manga)
		}
	})

	if len(mangas) > 0 {
		return mangas
	}

	// Método 3: Busca direta em h3 > a com link para /manga/ (estrutura mais comum)
	doc.Find("h3 a[href*='/manga/'], h5 a[href*='/manga/']").Each(func(i int, a *goquery.Selection) {
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
		parent := a.Closest(".page-item-detail, article, div")
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

	// Método 4: Fallback mais amplo - qualquer link para mangá dentro de container
	doc.Find("div, article").Each(func(i int, s *goquery.Selection) {
		// Pega links para mangás (não capítulos)
		link := s.Find("a[href*='/manga/']").First()
		href, exists := link.Attr("href")
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

		// Busca título
		title := s.Find("h3, h5, .post-title").First().Text()
		title = strings.TrimSpace(title)
		if title == "" || len(title) < 2 {
			return
		}

		// Busca imagem (opcional agora - não é obrigatório)
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

// extractMangaFromCard extrai informações de um card de mangá
func (c *MangaClient) extractMangaFromCard(s *goquery.Selection) Manga {
	var manga Manga

	// Busca link para o mangá (não capítulo)
	var mangaURL string
	s.Find("a[href*='/manga/']").Each(func(i int, a *goquery.Selection) {
		href, exists := a.Attr("href")
		if exists && !strings.Contains(href, "/capitulo") && !strings.Contains(href, "/chapter") {
			if mangaURL == "" {
				mangaURL = href
			}
		}
	})

	if mangaURL == "" {
		return manga
	}

	manga.URL = normalizeURL(mangaURL, c.baseURL)
	manga.ID = extractMangaID(mangaURL)

	// Título - procura em h3, h5 ou .post-title
	titleEl := s.Find("h3 a, h3, h5 a, h5, .post-title a, .post-title").First()
	manga.Title = strings.TrimSpace(titleEl.Text())

	// Imagem
	img := s.Find("img").First()
	manga.Image = normalizeImageURL(getImageSrc(img), c.baseURL)

	// Último capítulo
	s.Find("a[href*='/capitulo'], a[href*='/chapter']").First().Each(func(i int, ch *goquery.Selection) {
		manga.LatestChap = strings.TrimSpace(ch.Text())
	})

	// Autor
	s.Find("a[href*='/manga-author/']").First().Each(func(i int, a *goquery.Selection) {
		manga.Author = strings.TrimSpace(a.Text())
	})

	return manga
}

// ============== BUSCA ==============

// SearchManga busca mangás por termo
func (c *MangaClient) SearchManga(query string) ([]Manga, error) {
	searchURL := fmt.Sprintf("%s/?s=%s&post_type=wp-manga", c.baseURL, url.QueryEscape(query))

	fmt.Printf("[MangaClient] Buscando mangás com query '%s': %s\n", query, searchURL)

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
	fmt.Printf("[MangaClient] Busca por '%s' retornou %d resultados\n", query, len(mangas))
	return mangas, nil
}

// ============== DETALHES ==============

// GetMangaDetails obtém detalhes completos de um mangá
func (c *MangaClient) GetMangaDetails(mangaURL string) (*Manga, error) {
	fmt.Printf("[MangaClient] Obtendo detalhes de: %s\n", mangaURL)

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

	// Título
	manga.Title = strings.TrimSpace(doc.Find("h1").First().Text())

	// Imagem de capa
	coverImg := doc.Find(".summary_image img, img.wp-post-image").First()
	manga.Image = normalizeImageURL(getImageSrc(coverImg), c.baseURL)

	// Descrição
	desc := doc.Find(".summary__content, .description-summary, .manga-excerpt").First()
	manga.Description = strings.TrimSpace(desc.Text())

	// Gêneros
	doc.Find("a[href*='/genero/']").Each(func(i int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		if genre != "" && !containsString(manga.Genres, genre) {
			manga.Genres = append(manga.Genres, genre)
		}
	})

	// Status
	doc.Find(".post-status, .summary-content").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		if strings.Contains(text, "andamento") || strings.Contains(text, "ongoing") {
			manga.Status = "Em Andamento"
		} else if strings.Contains(text, "completo") || strings.Contains(text, "completed") || strings.Contains(text, "finalizado") {
			manga.Status = "Completo"
		}
	})

	// Autor
	doc.Find("a[href*='/manga-author/']").First().Each(func(i int, s *goquery.Selection) {
		manga.Author = strings.TrimSpace(s.Text())
	})

	fmt.Printf("[MangaClient] Detalhes obtidos: %s\n", manga.Title)
	return manga, nil
}

// ============== CAPÍTULOS (ORDENADOS) ==============

// GetChapters obtém a lista de capítulos de um mangá ORDENADOS DO PRIMEIRO AO ÚLTIMO
// O site só mostra os últimos capítulos na página, então geramos a lista completa
// baseado nos links "Ler primeiro capítulo" e "Ler último capítulo"
func (c *MangaClient) GetChapters(mangaURL string) ([]MangaChapter, error) {
	fmt.Printf("[MangaClient] Obtendo capítulos de: %s\n", mangaURL)

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

	// Coleta links de capítulos dos botões "Ler primeiro" e "Ler último"
	var firstChapterURL, lastChapterURL string

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(strings.TrimSpace(s.Text()))
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		href = strings.TrimSpace(href)
		if !strings.HasPrefix(href, "http") {
			href = c.baseURL + href
		}

		// IMPORTANTE: Só aceita links que pertencem a ESTE mangá (verificação exata)
		if !urlBelongsToManga(href, mangaID) {
			return
		}

		// Link para primeiro capítulo
		if strings.Contains(text, "primeiro") || strings.Contains(text, "first") {
			firstChapterURL = href
			fmt.Printf("[MangaClient] Primeiro capítulo URL: %s\n", firstChapterURL)
		}

		// Link para último capítulo
		if strings.Contains(text, "último") || strings.Contains(text, "ultimo") || strings.Contains(text, "last") {
			lastChapterURL = href
			fmt.Printf("[MangaClient] Último capítulo URL: %s\n", lastChapterURL)
		}
	})

	// Seletores específicos do Madara theme para lista de capítulos
	// Inclui qualquer link que termine com capitulo-, chapter-, ou completo
	chapterSelectors := []string{
		"li.wp-manga-chapter a",
		".version-chap a",
		".chapter-link",
	}

	for _, selector := range chapterSelectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists || href == "" {
				return
			}

			href = strings.TrimSpace(href)
			if !strings.HasPrefix(href, "http") {
				href = c.baseURL + href
			}

			// IMPORTANTE: Só aceita links que pertencem a ESTE mangá (verificação exata)
			if !urlBelongsToManga(href, mangaID) {
				return
			}

			// Ignora se já vimos esta URL
			if seenURLs[href] {
				return
			}
			seenURLs[href] = true

			// Extrai número do capítulo da URL
			chapter := c.parseChapterFromURL(href, mangaID, mangaName)
			if chapter != nil {
				chapters = append(chapters, *chapter)
			}
		})

		if len(chapters) > 0 {
			break // Se encontrou capítulos, não precisa tentar outros seletores
		}
	}

	// Se não encontrou com seletores específicos, busca links gerais de capítulos
	if len(chapters) == 0 {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists || href == "" {
				return
			}

			href = strings.TrimSpace(href)
			if !strings.HasPrefix(href, "http") {
				href = c.baseURL + href
			}

			// IMPORTANTE: Só aceita links que pertencem a ESTE mangá (verificação exata)
			if !urlBelongsToManga(href, mangaID) {
				return
			}

			// Verifica se é uma URL de capítulo
			lowerHref := strings.ToLower(href)
			isChapter := strings.Contains(lowerHref, "/capitulo-") ||
				strings.Contains(lowerHref, "/chapter-") ||
				strings.HasSuffix(lowerHref, "/completo/") ||
				strings.HasSuffix(lowerHref, "/oneshot/")

			if !isChapter {
				return
			}

			// Ignora se já vimos esta URL
			if seenURLs[href] {
				return
			}
			seenURLs[href] = true

			chapter := c.parseChapterFromURL(href, mangaID, mangaName)
			if chapter != nil {
				chapters = append(chapters, *chapter)
			}
		})
	}

	// Adiciona o primeiro capítulo se ainda não foi adicionado
	if firstChapterURL != "" {
		fmt.Printf("[MangaClient] Tentando adicionar primeiro capítulo: %s (já visto: %v)\n", firstChapterURL, seenURLs[firstChapterURL])
		if !seenURLs[firstChapterURL] {
			seenURLs[firstChapterURL] = true
			if chapter := c.parseChapterFromURL(firstChapterURL, mangaID, mangaName); chapter != nil {
				chapters = append(chapters, *chapter)
				fmt.Printf("[MangaClient] Primeiro capítulo adicionado: %s\n", chapter.Title)
			} else {
				fmt.Printf("[MangaClient] Erro: parseChapterFromURL retornou nil para primeiro capítulo\n")
			}
		}
	}

	// Adiciona o último capítulo se ainda não foi adicionado
	if lastChapterURL != "" && lastChapterURL != firstChapterURL {
		fmt.Printf("[MangaClient] Tentando adicionar último capítulo: %s (já visto: %v)\n", lastChapterURL, seenURLs[lastChapterURL])
		if !seenURLs[lastChapterURL] {
			seenURLs[lastChapterURL] = true
			if chapter := c.parseChapterFromURL(lastChapterURL, mangaID, mangaName); chapter != nil {
				chapters = append(chapters, *chapter)
				fmt.Printf("[MangaClient] Último capítulo adicionado: %s\n", chapter.Title)
			} else {
				fmt.Printf("[MangaClient] Erro: parseChapterFromURL retornou nil para último capítulo\n")
			}
		}
	}

	// ORDENA CAPÍTULOS DO PRIMEIRO AO ÚLTIMO
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].NumberFloat < chapters[j].NumberFloat
	})

	fmt.Printf("[MangaClient] Encontrados %d capítulos para %s\n", len(chapters), mangaName)

	if len(chapters) > 0 {
		fmt.Printf("[MangaClient] Primeiro: Cap %s (%.1f), Último: Cap %s (%.1f)\n",
			chapters[0].Number, chapters[0].NumberFloat,
			chapters[len(chapters)-1].Number, chapters[len(chapters)-1].NumberFloat)
	}

	return chapters, nil
}

// parseChapterFromURL extrai informações de um capítulo a partir da URL
func (c *MangaClient) parseChapterFromURL(href, mangaID, mangaName string) *MangaChapter {
	lowerHref := strings.ToLower(href)

	// Primeiro, tenta extrair número do capítulo
	// Suporta: capitulo-1, capitulo-01, capitulo-1-5, capitulo-extra, chapter-1, etc.
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
				// Para extras, adiciona 0.5 para ordenar depois do capítulo normal
				numFloat += 0.5
				chapterStr = fmt.Sprintf("%d %s", chapterNum, strings.ToUpper(matches[2]))
			}
		}
	} else if strings.Contains(lowerHref, "/completo") {
		// Mangá oneshot/completo - capítulo único
		chapterNum = 1
		numFloat = 1.0
		chapterStr = "Completo"
	} else if strings.Contains(lowerHref, "/oneshot") {
		// Oneshot
		chapterNum = 1
		numFloat = 1.0
		chapterStr = "Oneshot"
		fmt.Printf("[parseChapter] Detectado /oneshot\n")
	} else {
		// Não conseguiu identificar o formato
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
func (c *MangaClient) GetChapterPages(chapterURL string) ([]MangaPage, error) {
	fmt.Printf("[MangaClient] Buscando páginas de: %s\n", chapterURL)

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

	// Método 1: Estrutura do Madara theme - .reading-content .page-break img
	doc.Find(".reading-content .page-break img").Each(func(i int, s *goquery.Selection) {
		imgSrc := getImageSrc(s)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && isValidMangaImage(imgSrc) {
			seenURLs[imgSrc] = true
			pages = append(pages, MangaPage{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, c.baseURL),
			})
		}
	})

	if len(pages) > 0 {
		fmt.Printf("[MangaClient] Encontrou %d páginas (método page-break)\n", len(pages))
		return pages, nil
	}

	// Método 2: Imagens com classe wp-manga-chapter-img
	doc.Find("img.wp-manga-chapter-img").Each(func(i int, s *goquery.Selection) {
		imgSrc := getImageSrc(s)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && isValidMangaImage(imgSrc) {
			seenURLs[imgSrc] = true
			pages = append(pages, MangaPage{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, c.baseURL),
			})
		}
	})

	if len(pages) > 0 {
		fmt.Printf("[MangaClient] Encontrou %d páginas (método wp-manga-chapter-img)\n", len(pages))
		return pages, nil
	}

	// Método 3: Busca genérica dentro de reading-content
	doc.Find(".reading-content img, #chapter-content img, .chapter-content img").Each(func(i int, s *goquery.Selection) {
		imgSrc := getImageSrc(s)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && isValidMangaImage(imgSrc) {
			seenURLs[imgSrc] = true
			pages = append(pages, MangaPage{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, c.baseURL),
			})
		}
	})

	if len(pages) > 0 {
		fmt.Printf("[MangaClient] Encontrou %d páginas (método genérico)\n", len(pages))
		return pages, nil
	}

	// Método 4: Qualquer imagem grande no body
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		imgSrc := getImageSrc(s)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && isValidMangaImage(imgSrc) {
			seenURLs[imgSrc] = true
			pages = append(pages, MangaPage{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, c.baseURL),
			})
		}
	})

	fmt.Printf("[MangaClient] Total de páginas encontradas: %d\n", len(pages))
	return pages, nil
}

// ============== GÊNEROS ==============

// GetMangasByGenre retorna mangás de um gênero específico
func (c *MangaClient) GetMangasByGenre(genre string) ([]Manga, error) {
	genreSlug := normalizeGenreSlug(genre)
	genreURL := fmt.Sprintf("%s/genero/%s/", c.baseURL, genreSlug)

	fmt.Printf("[MangaClient] Buscando mangás do gênero '%s': %s\n", genre, genreURL)

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

	// Adiciona o gênero a todos os mangás
	for i := range mangas {
		if !containsString(mangas[i].Genres, genre) {
			mangas[i].Genres = append(mangas[i].Genres, genre)
		}
	}

	fmt.Printf("[MangaClient] Gênero '%s' retornou %d mangás\n", genre, len(mangas))
	return mangas, nil
}

// GetGenres retorna a lista de gêneros disponíveis
func (c *MangaClient) GetGenres() ([]string, error) {
	resp, err := c.makeRequest(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar MangaLivre: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %v", err)
	}

	var genres []string
	seenGenres := make(map[string]bool)

	doc.Find("a[href*='/genero/']").Each(func(i int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		// Remove números entre parênteses (ex: "Ação (67)")
		re := regexp.MustCompile(`\s*\(\d+\)\s*$`)
		genre = re.ReplaceAllString(genre, "")
		genre = strings.TrimSpace(genre)

		if genre != "" && !seenGenres[genre] && len(genre) > 1 {
			seenGenres[genre] = true
			genres = append(genres, genre)
		}
	})

	fmt.Printf("[MangaClient] Encontrou %d gêneros\n", len(genres))
	return genres, nil
}

// ============== FUNÇÕES AUXILIARES ==============

// getImageSrc extrai a URL da imagem de um elemento img
func getImageSrc(img *goquery.Selection) string {
	// Tenta data-src primeiro (lazy loading)
	if dataSrc, exists := img.Attr("data-src"); exists && dataSrc != "" && !strings.HasPrefix(dataSrc, "data:") {
		return strings.TrimSpace(dataSrc)
	}

	// Tenta data-lazy-src
	if lazySrc, exists := img.Attr("data-lazy-src"); exists && lazySrc != "" {
		return strings.TrimSpace(lazySrc)
	}

	// Tenta src
	if src, exists := img.Attr("src"); exists && src != "" && !strings.HasPrefix(src, "data:") {
		return strings.TrimSpace(src)
	}

	// Tenta data-cfsrc
	if cfSrc, exists := img.Attr("data-cfsrc"); exists && cfSrc != "" {
		return strings.TrimSpace(cfSrc)
	}

	// Tenta srcset
	if srcset, exists := img.Attr("srcset"); exists && srcset != "" {
		parts := strings.Split(srcset, ",")
		if len(parts) > 0 {
			firstPart := strings.TrimSpace(parts[0])
			urlParts := strings.Split(firstPart, " ")
			if len(urlParts) > 0 {
				return strings.TrimSpace(urlParts[0])
			}
		}
	}

	return ""
}

// normalizeURL normaliza uma URL relativa para absoluta
func normalizeURL(href, baseURL string) string {
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		return baseURL + href
	}
	return baseURL + "/" + href
}

// normalizeImageURL normaliza a URL de uma imagem
func normalizeImageURL(imgSrc, baseURL string) string {
	imgSrc = strings.TrimSpace(imgSrc)
	if imgSrc == "" {
		return ""
	}
	return normalizeURL(imgSrc, baseURL)
}

// extractMangaID extrai o ID do mangá da URL
func extractMangaID(href string) string {
	href = strings.TrimSuffix(href, "/")
	parts := strings.Split(href, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && parts[i] != "manga" && !strings.HasPrefix(parts[i], "capitulo") && !strings.HasPrefix(parts[i], "chapter") {
			return parts[i]
		}
	}
	return ""
}

// urlBelongsToManga verifica se uma URL pertence EXATAMENTE a um mangá específico
// Evita que "solo-leveling" aceite "solo-leveling-ragnarok"
func urlBelongsToManga(href, mangaID string) bool {
	// Extrai o mangaID da URL do capítulo
	urlMangaID := extractMangaID(href)
	return urlMangaID == mangaID
}

// extractChapterNumberAndFloat extrai o número do capítulo como string e float para ordenação
func extractChapterNumberAndFloat(href string) (string, float64) {
	lowerHref := strings.ToLower(href)

	// Padrão 1: capitulo-1167, capitulo-03-6, capitulo-200-final
	re1 := regexp.MustCompile(`(?:capitulo|chapter|cap)-(\d+(?:[.-]\d+)?)(?:-[a-zA-Z]+)?/?$`)
	if matches := re1.FindStringSubmatch(lowerHref); len(matches) > 1 {
		numStr := matches[1]
		// Substitui hífen por ponto para números decimais como "03-6" -> "03.6"
		numStr = strings.Replace(numStr, "-", ".", 1)

		numFloat, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			// Tenta extrair apenas o número principal
			numOnlyRe := regexp.MustCompile(`^(\d+)`)
			if numMatches := numOnlyRe.FindStringSubmatch(numStr); len(numMatches) > 1 {
				numFloat, _ = strconv.ParseFloat(numMatches[1], 64)
			}
		}
		return matches[1], numFloat
	}

	// Padrão 2: /1167/ ou /cap-1/ no meio da URL
	re2 := regexp.MustCompile(`/(\d+)/?$`)
	if matches := re2.FindStringSubmatch(lowerHref); len(matches) > 1 {
		numFloat, _ := strconv.ParseFloat(matches[1], 64)
		return matches[1], numFloat
	}

	return "", 0
}

// extractFloatFromText tenta extrair um número de um texto
func extractFloatFromText(text string) float64 {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		num, _ := strconv.ParseFloat(matches[1], 64)
		return num
	}
	return 0
}

// isValidMangaImage verifica se é uma imagem válida de mangá
func isValidMangaImage(imgSrc string) bool {
	lowerSrc := strings.ToLower(imgSrc)

	// Ignora imagens de UI
	if strings.Contains(lowerSrc, "logo") ||
		strings.Contains(lowerSrc, "icon") ||
		strings.Contains(lowerSrc, "avatar") ||
		strings.Contains(lowerSrc, "banner") ||
		strings.Contains(lowerSrc, "gravatar") ||
		strings.Contains(lowerSrc, "loading") ||
		strings.Contains(lowerSrc, "placeholder") {
		return false
	}

	// Aceita imagens de manga/uploads
	if strings.Contains(lowerSrc, "manga") ||
		strings.Contains(lowerSrc, "chapter") ||
		strings.Contains(lowerSrc, "wp-manga") ||
		strings.Contains(lowerSrc, "uploads") ||
		strings.Contains(lowerSrc, "content") {
		return true
	}

	// Aceita extensões comuns de imagem
	return strings.HasSuffix(lowerSrc, ".webp") ||
		strings.HasSuffix(lowerSrc, ".jpg") ||
		strings.HasSuffix(lowerSrc, ".jpeg") ||
		strings.HasSuffix(lowerSrc, ".png")
}

// normalizeGenreSlug normaliza o nome do gênero para URL
func normalizeGenreSlug(genre string) string {
	slug := strings.ToLower(genre)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Substitui acentos
	replacements := map[string]string{
		"ã": "a", "á": "a", "à": "a", "â": "a",
		"ç": "c",
		"é": "e", "ê": "e", "è": "e",
		"í": "i", "ì": "i", "î": "i",
		"ó": "o", "ô": "o", "õ": "o", "ò": "o",
		"ú": "u", "ù": "u", "û": "u",
	}

	for from, to := range replacements {
		slug = strings.ReplaceAll(slug, from, to)
	}

	return slug
}

// containsString verifica se uma slice contém uma string
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
