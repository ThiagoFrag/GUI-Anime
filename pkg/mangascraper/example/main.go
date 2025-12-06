//go:build ignore
// +build ignore

// This file is an example of how to use the mangascraper package.
// Run it with: go run pkg/mangascraper/example/main.go
package main

import (
	"fmt"
	"log"

	"GoAnimeGUI/pkg/mangascraper"
)

func main() {
	fmt.Println("=== MangaScraper Example ===")
	fmt.Println()

	// Create a new scraper with default settings
	scraper := mangascraper.New()

	// List available sources
	sources := scraper.GetSources()
	fmt.Println("Available sources:", sources)
	fmt.Println()

	// Get detailed source info
	for _, info := range scraper.GetSourceInfo() {
		fmt.Printf("  - %s (%s): %s\n", info.Name, info.DisplayName, info.BaseURL)
	}
	fmt.Println()

	// Example 1: Get mangas from a specific source
	fmt.Println("=== Example 1: Get mangas from mangalivre.blog ===")
	mangas, totalPages, err := scraper.GetAllMangas("mangalivre.blog", 1)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Found %d mangas, %d total pages\n", len(mangas), totalPages)
		for i, m := range mangas[:min(5, len(mangas))] {
			fmt.Printf("  %d. %s\n", i+1, m.Title)
		}
	}
	fmt.Println()

	// Example 2: Get chapters from a manga
	if len(mangas) > 0 {
		fmt.Println("=== Example 2: Get chapters ===")
		chapters, err := scraper.GetChapters(mangas[0].URL)
		if err != nil {
			log.Printf("Error: %v", err)
		} else {
			fmt.Printf("Found %d chapters for '%s'\n", len(chapters), mangas[0].Title)
			for i, ch := range chapters[:min(5, len(chapters))] {
				fmt.Printf("  %d. %s: %s\n", i+1, ch.Number, ch.Title)
			}
		}
		fmt.Println()

		// Example 3: Get pages from a chapter
		if len(chapters) > 0 {
			fmt.Println("=== Example 3: Get chapter pages ===")
			pages, err := scraper.GetChapterPages(chapters[0].URL)
			if err != nil {
				log.Printf("Error: %v", err)
			} else {
				fmt.Printf("Chapter has %d pages\n", len(pages))
				for _, p := range pages[:min(3, len(pages))] {
					fmt.Printf("  Page %d: %s\n", p.Number, truncate(p.URL, 60))
				}
			}
		}
	}
	fmt.Println()

	// Example 4: Search across all sources
	fmt.Println("=== Example 4: Search all sources ===")
	results, err := scraper.SearchAllSources("one piece")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		for _, result := range results {
			if result.Error != nil {
				fmt.Printf("  %s: error - %v\n", result.Source, result.Error)
			} else {
				fmt.Printf("  %s: found %d results\n", result.Source, len(result.Mangas))
			}
		}
	}
	fmt.Println()

	// Example 5: Custom configuration
	fmt.Println("=== Example 5: Custom configuration ===")
	config := mangascraper.DefaultConfig()
	config.EnableCache = false // Disable caching for this example
	config.MaxRetries = 5

	customScraper := mangascraper.NewWithConfig(config)
	fmt.Printf("Created scraper with %d sources (cache disabled)\n", len(customScraper.GetSources()))

	fmt.Println()
	fmt.Println("=== Done! ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
