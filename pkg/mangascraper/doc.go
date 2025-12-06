// Package mangascraper provides a portable manga scraping library.
// It supports multiple sources and can be easily integrated into other projects.
//
// Installation (from this repo):
//
//	go get github.com/ThiagoFrag/GUI-Anime/pkg/mangascraper
//
// Or copy the entire pkg/mangascraper folder to your project.
//
// Basic usage:
//
//	scraper := mangascraper.New()
//	mangas, totalPages, err := scraper.GetAllMangas("mangalivre.blog", 1)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// For more examples, see the README.md file.
package mangascraper
