package mangascraper

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// MangaLivreToSource implements the Source interface for mangalivre.to
type MangaLivreToSource struct {
	*baseSource
}

// NewMangaLivreToSource creates a new MangaLivre.to source
func NewMangaLivreToSource(config *Config) *MangaLivreToSource {
	return &MangaLivreToSource{
		baseSource: newBaseSource("mangalivre.to", "MangaLivre.to", "https://mangalivre.to", config),
	}
}

// GetAllMangas returns all mangas with pagination
func (s *MangaLivreToSource) GetAllMangas(page int) ([]Manga, int, error) {
	pageURL := fmt.Sprintf("%s/manga/", s.baseURL)
	if page > 1 {
		pageURL = fmt.Sprintf("%s/manga/page/%d/", s.baseURL, page)
	}

	doc, err := s.fetchDocument(pageURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to access MangaLivre.to: %v", err)
	}

	mangas := s.extractMangasFromListPage(doc)
	totalPages := s.extractTotalPages(doc)

	for i := range mangas {
		mangas[i].Source = s.name
	}

	return mangas, totalPages, nil
}

// GetPopularMangas returns popular mangas
func (s *MangaLivreToSource) GetPopularMangas() ([]Manga, error) {
	pageURL := fmt.Sprintf("%s/manga/?m_orderby=views", s.baseURL)
	doc, err := s.fetchDocument(pageURL)
	if err != nil {
		return nil, err
	}

	mangas := s.extractMangasFromListPage(doc)
	for i := range mangas {
		mangas[i].Source = s.name
	}
	return mangas, nil
}

// GetLatestUpdates returns recently updated mangas
func (s *MangaLivreToSource) GetLatestUpdates() ([]Manga, error) {
	pageURL := fmt.Sprintf("%s/manga/?m_orderby=latest", s.baseURL)
	doc, err := s.fetchDocument(pageURL)
	if err != nil {
		return nil, err
	}

	mangas := s.extractMangasFromListPage(doc)
	for i := range mangas {
		mangas[i].Source = s.name
	}
	return mangas, nil
}

// SearchManga searches for mangas by query
func (s *MangaLivreToSource) SearchManga(query string) ([]Manga, error) {
	searchURL := fmt.Sprintf("%s/?s=%s&post_type=wp-manga", s.baseURL, url.QueryEscape(query))
	doc, err := s.fetchDocument(searchURL)
	if err != nil {
		return nil, err
	}

	mangas := s.extractMangasFromListPage(doc)
	for i := range mangas {
		mangas[i].Source = s.name
	}
	return mangas, nil
}

// GetMangaDetails returns detailed information about a manga
func (s *MangaLivreToSource) GetMangaDetails(mangaURL string) (*Manga, error) {
	doc, err := s.fetchDocument(mangaURL)
	if err != nil {
		return nil, err
	}

	manga := &Manga{
		URL:    mangaURL,
		ID:     extractMangaID(mangaURL),
		Source: s.name,
	}

	// Title
	manga.Title = strings.TrimSpace(doc.Find("h1").First().Text())

	// Cover image
	coverImg := doc.Find(".summary_image img, img.wp-post-image").First()
	manga.Image = normalizeImageURL(getImageSrc(coverImg), s.baseURL)

	// Description
	desc := doc.Find(".summary__content, .description-summary, .manga-excerpt").First()
	manga.Description = strings.TrimSpace(desc.Text())

	// Genres
	doc.Find("a[href*='/genero/']").Each(func(i int, sel *goquery.Selection) {
		genre := strings.TrimSpace(sel.Text())
		if genre != "" && !containsString(manga.Genres, genre) {
			manga.Genres = append(manga.Genres, genre)
		}
	})

	// Status
	doc.Find(".post-status, .summary-content").Each(func(i int, sel *goquery.Selection) {
		text := strings.ToLower(sel.Text())
		if strings.Contains(text, "andamento") || strings.Contains(text, "ongoing") {
			manga.Status = "Em Andamento"
		} else if strings.Contains(text, "completo") || strings.Contains(text, "completed") {
			manga.Status = "Completo"
		}
	})

	// Author
	doc.Find("a[href*='/manga-author/']").First().Each(func(i int, sel *goquery.Selection) {
		manga.Author = strings.TrimSpace(sel.Text())
	})

	return manga, nil
}

// GetChapters returns all chapters of a manga
func (s *MangaLivreToSource) GetChapters(mangaURL string) ([]Chapter, error) {
	doc, err := s.fetchDocument(mangaURL)
	if err != nil {
		return nil, err
	}

	mangaID := extractMangaID(mangaURL)
	mangaName := strings.TrimSpace(doc.Find("h1").First().Text())

	var chapters []Chapter
	seenURLs := make(map[string]bool)

	// Look for chapter links
	doc.Find("a[href*='/capitulo'], a[href*='/chapter']").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		href = normalizeURL(strings.TrimSpace(href), s.baseURL)
		if seenURLs[href] {
			return
		}
		seenURLs[href] = true

		chapter := s.parseChapterFromElement(sel, href, mangaID, mangaName)
		if chapter != nil {
			chapters = append(chapters, *chapter)
		}
	})

	// Sort chapters
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].NumberFloat < chapters[j].NumberFloat
	})

	return chapters, nil
}

// GetChapterPages returns all pages/images of a chapter
func (s *MangaLivreToSource) GetChapterPages(chapterURL string) ([]Page, error) {
	doc, err := s.fetchDocument(chapterURL)
	if err != nil {
		return nil, err
	}

	var pages []Page
	seenURLs := make(map[string]bool)

	// Look for reader images
	doc.Find(".reading-content img, .chapter-content img, img.wp-manga-chapter-img").Each(func(i int, sel *goquery.Selection) {
		imgSrc := getImageSrc(sel)
		if imgSrc == "" || seenURLs[imgSrc] {
			return
		}

		if s.isValidMangaImage(imgSrc) {
			seenURLs[imgSrc] = true
			pages = append(pages, Page{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, s.baseURL),
			})
		}
	})

	return pages, nil
}

// GetMangasByGenre returns mangas filtered by genre
func (s *MangaLivreToSource) GetMangasByGenre(genre string) ([]Manga, error) {
	genreSlug := s.normalizeGenreSlug(genre)
	genreURL := fmt.Sprintf("%s/genero/%s/", s.baseURL, genreSlug)

	doc, err := s.fetchDocument(genreURL)
	if err != nil {
		return nil, err
	}

	mangas := s.extractMangasFromListPage(doc)
	for i := range mangas {
		mangas[i].Source = s.name
		if !containsString(mangas[i].Genres, genre) {
			mangas[i].Genres = append(mangas[i].Genres, genre)
		}
	}

	return mangas, nil
}

// GetGenres returns available genres
func (s *MangaLivreToSource) GetGenres() ([]string, error) {
	doc, err := s.fetchDocument(s.baseURL)
	if err != nil {
		return nil, err
	}

	var genres []string
	seenGenres := make(map[string]bool)

	doc.Find("a[href*='/genero/']").Each(func(i int, sel *goquery.Selection) {
		genre := strings.TrimSpace(sel.Text())
		// Remove count suffix like "(123)"
		re := regexp.MustCompile(`\s*\(\d+\)\s*$`)
		genre = re.ReplaceAllString(genre, "")
		genre = strings.TrimSpace(genre)

		if genre != "" && !seenGenres[genre] && len(genre) > 1 {
			seenGenres[genre] = true
			genres = append(genres, genre)
		}
	})

	return genres, nil
}

// Helper methods

func (s *MangaLivreToSource) extractTotalPages(doc *goquery.Document) int {
	totalPages := 1
	doc.Find("a").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if exists && strings.Contains(href, "/manga/page/") {
			re := regexp.MustCompile(`/page/(\d+)`)
			if matches := re.FindStringSubmatch(href); len(matches) > 1 {
				if p, err := strconv.Atoi(matches[1]); err == nil && p > totalPages {
					totalPages = p
				}
			}
		}
	})
	return totalPages
}

func (s *MangaLivreToSource) extractMangasFromListPage(doc *goquery.Document) []Manga {
	var mangas []Manga
	seenURLs := make(map[string]bool)

	// Method 1: Madara theme structure (.page-item-detail)
	doc.Find(".page-item-detail").Each(func(i int, sel *goquery.Selection) {
		manga := s.extractMangaFromCard(sel)
		if manga.Title != "" && manga.URL != "" && !seenURLs[manga.URL] {
			seenURLs[manga.URL] = true
			mangas = append(mangas, manga)
		}
	})

	if len(mangas) > 0 {
		return mangas
	}

	// Method 2: c-tabs-item structure
	doc.Find(".c-tabs-item__content").Each(func(i int, sel *goquery.Selection) {
		manga := s.extractMangaFromCard(sel)
		if manga.Title != "" && manga.URL != "" && !seenURLs[manga.URL] {
			seenURLs[manga.URL] = true
			mangas = append(mangas, manga)
		}
	})

	if len(mangas) > 0 {
		return mangas
	}

	// Method 3: Direct search in h3 > a with /manga/ link
	doc.Find("h3 a[href*='/manga/'], h5 a[href*='/manga/']").Each(func(i int, a *goquery.Selection) {
		href, exists := a.Attr("href")
		if !exists {
			return
		}

		href = strings.TrimSpace(href)
		if strings.Contains(href, "/capitulo") || strings.Contains(href, "/chapter") {
			return
		}

		href = normalizeURL(href, s.baseURL)
		if seenURLs[href] {
			return
		}

		title := strings.TrimSpace(a.Text())
		if title == "" || len(title) < 2 {
			return
		}

		parent := a.Closest(".page-item-detail, article, div")
		imgSrc := ""
		if parent.Length() > 0 {
			img := parent.Find("img").First()
			imgSrc = getImageSrc(img)
		}

		seenURLs[href] = true
		mangas = append(mangas, Manga{
			ID:     extractMangaID(href),
			Title:  title,
			Image:  normalizeImageURL(imgSrc, s.baseURL),
			URL:    href,
			Source: s.name,
		})
	})

	return mangas
}

func (s *MangaLivreToSource) extractMangaFromCard(sel *goquery.Selection) Manga {
	var manga Manga
	manga.Source = s.name

	// Find manga link (not chapter)
	var mangaURL string
	sel.Find("a[href*='/manga/']").Each(func(i int, a *goquery.Selection) {
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

	manga.URL = normalizeURL(mangaURL, s.baseURL)
	manga.ID = extractMangaID(mangaURL)

	// Title
	titleEl := sel.Find("h3 a, h3, h5 a, h5, .post-title a, .post-title").First()
	manga.Title = strings.TrimSpace(titleEl.Text())

	// Image
	img := sel.Find("img").First()
	manga.Image = normalizeImageURL(getImageSrc(img), s.baseURL)

	// Latest chapter
	sel.Find("a[href*='/capitulo'], a[href*='/chapter']").First().Each(func(i int, ch *goquery.Selection) {
		manga.LatestChap = strings.TrimSpace(ch.Text())
	})

	// Author
	sel.Find("a[href*='/manga-author/']").First().Each(func(i int, a *goquery.Selection) {
		manga.Author = strings.TrimSpace(a.Text())
	})

	return manga
}

func (s *MangaLivreToSource) parseChapterFromElement(sel *goquery.Selection, href, mangaID, mangaName string) *Chapter {
	text := strings.TrimSpace(sel.Text())
	numFloat := parseChapterNumber(text)

	if numFloat == 0 {
		// Try extracting from URL
		re := regexp.MustCompile(`(?:capitulo|chapter)-(\d+)`)
		if matches := re.FindStringSubmatch(strings.ToLower(href)); len(matches) > 1 {
			if num, err := strconv.ParseFloat(matches[1], 64); err == nil {
				numFloat = num
			}
		}
	}

	if numFloat == 0 {
		return nil
	}

	return &Chapter{
		Number:      fmt.Sprintf("%.0f", numFloat),
		NumberFloat: numFloat,
		Title:       text,
		URL:         href,
		MangaID:     mangaID,
		MangaName:   mangaName,
	}
}

func (s *MangaLivreToSource) isValidMangaImage(imgSrc string) bool {
	lowerSrc := strings.ToLower(imgSrc)

	// Skip UI images
	if strings.Contains(lowerSrc, "logo") ||
		strings.Contains(lowerSrc, "icon") ||
		strings.Contains(lowerSrc, "avatar") ||
		strings.Contains(lowerSrc, "banner") ||
		strings.Contains(lowerSrc, "gravatar") {
		return false
	}

	// Accept WordPress uploads
	if strings.Contains(lowerSrc, "wp-content/uploads") {
		return true
	}

	return strings.HasSuffix(lowerSrc, ".webp") ||
		strings.HasSuffix(lowerSrc, ".jpg") ||
		strings.HasSuffix(lowerSrc, ".jpeg") ||
		strings.HasSuffix(lowerSrc, ".png")
}

func (s *MangaLivreToSource) normalizeGenreSlug(genre string) string {
	slug := strings.ToLower(genre)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "ã", "a")
	slug = strings.ReplaceAll(slug, "á", "a")
	slug = strings.ReplaceAll(slug, "é", "e")
	slug = strings.ReplaceAll(slug, "í", "i")
	slug = strings.ReplaceAll(slug, "ó", "o")
	slug = strings.ReplaceAll(slug, "ú", "u")
	slug = strings.ReplaceAll(slug, "ç", "c")
	return slug
}
