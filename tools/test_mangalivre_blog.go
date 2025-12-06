package main

import (
	"GoAnimeGUI/internal/manga"
	"fmt"
)

func main() {
	fmt.Println("=== Testando MangaLivre.blog ===")

	client := manga.NewMangaLivreBlogClient()

	// Testa listagem
	fmt.Println("\n1. Buscando mangás da página 1...")
	mangas, totalPages, err := client.GetAllMangas(1)
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		return
	}
	fmt.Printf("✅ Encontrados %d mangás, %d páginas totais\n", len(mangas), totalPages)

	// Mostra primeiros 5 mangás
	fmt.Println("\nPrimeiros mangás:")
	for i, m := range mangas[:minInt(5, len(mangas))] {
		fmt.Printf("  %d. %s - %s\n", i+1, m.Title, m.URL)
	}

	if len(mangas) == 0 {
		fmt.Println("❌ Nenhum mangá encontrado!")
		return
	}

	// Testa detalhes
	testManga := mangas[0]
	fmt.Printf("\n2. Obtendo detalhes de: %s\n", testManga.Title)
	details, err := client.GetMangaDetails(testManga.URL)
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
	} else {
		fmt.Printf("✅ Título: %s\n", details.Title)
		fmt.Printf("   Gêneros: %v\n", details.Genres)
		fmt.Printf("   Status: %s\n", details.Status)
	}

	// Testa capítulos
	fmt.Printf("\n3. Obtendo capítulos de: %s\n", testManga.Title)
	chapters, err := client.GetChapters(testManga.URL)
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
	} else {
		fmt.Printf("✅ Encontrados %d capítulos\n", len(chapters))
		if len(chapters) > 0 {
			fmt.Printf("   Primeiro: Cap %s\n", chapters[0].Number)
			fmt.Printf("   Último: Cap %s\n", chapters[len(chapters)-1].Number)

			// Testa páginas do primeiro capítulo
			fmt.Printf("\n4. Obtendo páginas do capítulo 1...\n")
			pages, err := client.GetChapterPages(chapters[0].URL)
			if err != nil {
				fmt.Printf("❌ Erro: %v\n", err)
			} else {
				fmt.Printf("✅ Encontradas %d páginas\n", len(pages))
				if len(pages) > 0 {
					fmt.Printf("   Primeira página: %s\n", pages[0].URL)
				}
			}
		}
	}

	// Testa busca
	fmt.Println("\n5. Testando busca por 'One Piece'...")
	searchResults, err := client.SearchManga("One Piece")
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
	} else {
		fmt.Printf("✅ Encontrados %d resultados\n", len(searchResults))
		for i, m := range searchResults[:minInt(3, len(searchResults))] {
			fmt.Printf("  %d. %s\n", i+1, m.Title)
		}
	}

	fmt.Println("\n=== Teste concluído! ===")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
