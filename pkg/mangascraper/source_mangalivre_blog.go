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

// MangaLivreBlogSource implements the Source interface for mangalivre.blog
type MangaLivreBlogSource struct {
	*baseSource
}

// NewMangaLivreBlogSource creates a new MangaLivre.blog source
func NewMangaLivreBlogSource(config *Config) *MangaLivreBlogSource {
	return &MangaLivreBlogSource{
		baseSource: newBaseSource("mangalivre.blog", "MangaLivre.blog", "https://mangalivre.blog", config),
	}
}

// GetAllMangas returns all mangas with pagination
func (s *MangaLivreBlogSource) GetAllMangas(page int) ([]Manga, int, error) {
	pageURL := fmt.Sprintf("%s/manga/", s.baseURL)
	if page > 1 {
		pageURL = fmt.Sprintf("%s/manga/page/%d/", s.baseURL, page)
	}

	doc, err := s.fetchDocument(pageURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to access MangaLivre.blog: %v", err)
	}

	mangas := s.extractMangasFromListPage(doc)
	totalPages := s.extractTotalPages(doc)

	for i := range mangas {
		mangas[i].Source = s.name
	}

	return mangas, totalPages, nil
}

// GetPopularMangas returns popular mangas
func (s *MangaLivreBlogSource) GetPopularMangas() ([]Manga, error) {
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
func (s *MangaLivreBlogSource) GetLatestUpdates() ([]Manga, error) {
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
func (s *MangaLivreBlogSource) SearchManga(query string) ([]Manga, error) {
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
func (s *MangaLivreBlogSource) GetMangaDetails(mangaURL string) (*Manga, error) {
	doc, err := s.fetchDocument(mangaURL)
	if err != nil {
		return nil, err
	}

	manga := &Manga{
		URL:    mangaURL,
		ID:     extractMangaID(mangaURL),
		Source: s.name,
	}

	// Title - In SlimeRead theme, the manga title is in h1 but not "MANGALIVRE"
	manga.Title = s.formatMangaTitle(extractMangaID(mangaURL))

	// Try to get a better title
	doc.Find("h1, .manga-title, .entry-title").Each(func(i int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text != "" && !strings.EqualFold(text, "MANGALIVRE") && !strings.EqualFold(text, "mangalivre") && len(text) > 3 {
			if manga.Title == s.formatMangaTitle(extractMangaID(mangaURL)) {
				manga.Title = text
			}
		}
	})

	// Cover image
	coverImg := doc.Find(".summary_image img, img.wp-post-image, .manga-cover img, .manga-thumb img").First()
	manga.Image = normalizeImageURL(getImageSrc(coverImg), s.baseURL)

	// Description
	doc.Find(".summary__content, .description-summary, .manga-excerpt, .manga-description").Each(func(i int, sel *goquery.Selection) {
		if manga.Description == "" {
			manga.Description = strings.TrimSpace(sel.Text())
		}
	})

	// Status
	fullText := doc.Text()
	lowerText := strings.ToLower(fullText)
	if strings.Contains(lowerText, "em lançamento") || strings.Contains(lowerText, "ongoing") {
		manga.Status = "Em Lançamento"
	} else if strings.Contains(lowerText, "completo") || strings.Contains(lowerText, "completed") || strings.Contains(lowerText, "finalizado") {
		manga.Status = "Completo"
	}

	// Genres
	doc.Find("a[href*='/genero/'], a[href*='/genre/']").Each(func(i int, sel *goquery.Selection) {
		genre := strings.TrimSpace(sel.Text())
		if genre != "" && !containsString(manga.Genres, genre) {
			manga.Genres = append(manga.Genres, genre)
		}
	})

	// Author
	doc.Find("a[href*='/manga-author/'], a[href*='/author/']").First().Each(func(i int, sel *goquery.Selection) {
		manga.Author = strings.TrimSpace(sel.Text())
	})

	return manga, nil
}

// GetChapters returns all chapters of a manga
func (s *MangaLivreBlogSource) GetChapters(mangaURL string) ([]Chapter, error) {
	doc, err := s.fetchDocument(mangaURL)
	if err != nil {
		return nil, err
	}

	mangaID := extractMangaID(mangaURL)
	mangaName := strings.TrimSpace(doc.Find("h1").First().Text())

	var chapters []Chapter
	seenURLs := make(map[string]bool)

	// MangaLivre.blog uses URLs like: /capitulo/manga-name-capitulo-123/
	doc.Find("a[href*='/capitulo/']").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		href = strings.TrimSpace(href)
		if !strings.HasPrefix(href, "http") {
			href = s.baseURL + href
		}

		if seenURLs[href] {
			return
		}

		// Check if URL contains the mangaID
		if !strings.Contains(strings.ToLower(href), strings.ToLower(mangaID)) {
			return
		}

		seenURLs[href] = true

		chapter := s.parseChapterFromURL(href, mangaID, mangaName)
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
func (s *MangaLivreBlogSource) GetChapterPages(chapterURL string) ([]Page, error) {
	doc, err := s.fetchDocument(chapterURL)
	if err != nil {
		return nil, err
	}

	var pages []Page
	seenURLs := make(map[string]bool)

	// MangaLivre.blog - Images have alt="Página X"
	doc.Find("img").Each(func(i int, sel *goquery.Selection) {
		alt, _ := sel.Attr("alt")
		if !strings.Contains(strings.ToLower(alt), "página") && !strings.Contains(strings.ToLower(alt), "pagina") {
			return
		}

		imgSrc := getImageSrc(sel)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && strings.Contains(imgSrc, "wp-content/uploads") {
			seenURLs[imgSrc] = true
			pages = append(pages, Page{
				Number: len(pages) + 1,
				URL:    normalizeImageURL(imgSrc, s.baseURL),
			})
		}
	})

	if len(pages) > 0 {
		return pages, nil
	}

	// Fallback: search all images in wp-content/uploads
	doc.Find("img").Each(func(i int, sel *goquery.Selection) {
		imgSrc := getImageSrc(sel)
		imgSrc = strings.TrimSpace(imgSrc)

		if imgSrc != "" && !seenURLs[imgSrc] && s.isValidMangaImage(imgSrc) {
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
func (s *MangaLivreBlogSource) GetMangasByGenre(genre string) ([]Manga, error) {
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
func (s *MangaLivreBlogSource) GetGenres() ([]string, error) {
	doc, err := s.fetchDocument(s.baseURL)
	if err != nil {
		return nil, err
	}

	var genres []string
	seenGenres := make(map[string]bool)

	doc.Find("a[href*='/genero/'], a[href*='/genre/']").Each(func(i int, sel *goquery.Selection) {
		genre := strings.TrimSpace(sel.Text())
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

func (s *MangaLivreBlogSource) extractTotalPages(doc *goquery.Document) int {
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

func (s *MangaLivreBlogSource) extractMangasFromListPage(doc *goquery.Document) []Manga {
	var mangas []Manga
	seenURLs := make(map[string]bool)

	// SlimeRead theme - search titles in h3 or h2 with links
	doc.Find("h3 a[href*='/manga/'], h2 a[href*='/manga/']").Each(func(i int, a *goquery.Selection) {
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

		parent := a.Closest("article, div, .manga-item, .post")
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

	if len(mangas) > 0 {
		return mangas
	}

	// Fallback: search in articles
	doc.Find("article, .manga-item").Each(func(i int, sel *goquery.Selection) {
		link := sel.Find("a[href*='/manga/']").First()
		href, exists := link.Attr("href")
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

		title := sel.Find("h2, h3, h4, .title").First().Text()
		title = strings.TrimSpace(title)
		if title == "" || len(title) < 2 {
			return
		}

		img := sel.Find("img").First()
		imgSrc := getImageSrc(img)

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

func (s *MangaLivreBlogSource) parseChapterFromURL(href, mangaID, mangaName string) *Chapter {
	lowerHref := strings.ToLower(href)

	// URL format: /capitulo/manga-name-capitulo-123/
	re := regexp.MustCompile(`(?:capitulo|chapter)-(\d+)(?:[.-](\d+|extra|final))?`)
	matches := re.FindStringSubmatch(lowerHref)

	var chapterNum int
	var numFloat float64
	var chapterStr string

	if len(matches) >= 2 {
		chapterNum, _ = strconv.Atoi(matches[1])
		numFloat = float64(chapterNum)
		chapterStr = fmt.Sprintf("%d", chapterNum)

		// Support decimals like capitulo-3-5 -> 3.5
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

	return &Chapter{
		Number:      chapterStr,
		NumberFloat: numFloat,
		Title:       title,
		URL:         href,
		MangaID:     mangaID,
		MangaName:   mangaName,
	}
}

func (s *MangaLivreBlogSource) formatMangaTitle(slug string) string {
	title := strings.ReplaceAll(slug, "-", " ")
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

func (s *MangaLivreBlogSource) isValidMangaImage(imgSrc string) bool {
	lowerSrc := strings.ToLower(imgSrc)

	// Skip UI images
	if strings.Contains(lowerSrc, "logo") ||
		strings.Contains(lowerSrc, "icon") ||
		strings.Contains(lowerSrc, "avatar") ||
		strings.Contains(lowerSrc, "banner") ||
		strings.Contains(lowerSrc, "gravatar") ||
		strings.Contains(lowerSrc, "loading") ||
		strings.Contains(lowerSrc, "placeholder") ||
		strings.Contains(lowerSrc, "ads") {
		return false
	}

	if strings.Contains(lowerSrc, "wp-content/uploads") {
		return true
	}

	return strings.HasSuffix(lowerSrc, ".webp") ||
		strings.HasSuffix(lowerSrc, ".jpg") ||
		strings.HasSuffix(lowerSrc, ".jpeg") ||
		strings.HasSuffix(lowerSrc, ".png")
}

func (s *MangaLivreBlogSource) normalizeGenreSlug(genre string) string {
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
