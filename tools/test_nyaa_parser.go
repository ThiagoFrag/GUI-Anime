package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	query := "naruto"
	searchURL := fmt.Sprintf("https://nyaa.si/?f=0&c=1_2&q=%s&s=seeders&o=desc", url.QueryEscape(query))

	client := &http.Client{}
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	rows := strings.Split(html, "<tr class=\"")
	fmt.Printf("Total rows: %d\n\n", len(rows)-1)

	count := 0
	for idx, row := range rows[1:] {
		if count >= 3 {
			break
		}
		
		if !strings.Contains(row, "success") && !strings.Contains(row, "default") && !strings.Contains(row, "danger") {
			continue
		}

		fmt.Printf("=== ROW %d ===\n", idx)

		// Localiza o <td colspan="2"> que contém o título
		colspanIdx := strings.Index(row, `colspan="2"`)
		if colspanIdx == -1 {
			fmt.Println("colspan não encontrado!")
			continue
		}

		// Pega o conteúdo a partir do colspan até </td>
		afterColspan := row[colspanIdx:]
		tdEndIdx := strings.Index(afterColspan, `</td>`)
		if tdEndIdx == -1 {
			fmt.Println("</td> não encontrado!")
			continue
		}
		tdContent := afterColspan[:tdEndIdx]
		fmt.Printf("TD Content (primeiros 300 chars):\n%s\n\n", tdContent[:min(300, len(tdContent))])

		// Procura links /view/ dentro do td
		viewLinks := strings.Split(tdContent, `<a href="/view/`)
		fmt.Printf("Links /view/ encontrados: %d\n", len(viewLinks)-1)

		for i := 1; i < len(viewLinks); i++ {
			link := viewLinks[i]
			snippet := link
			if len(snippet) > 150 {
				snippet = snippet[:150]
			}
			fmt.Printf("  Link %d: %s\n", i, snippet)

			// Pula se for link de comentários
			if strings.Contains(link, "#comments") {
				fmt.Println("    -> PULANDO (comentários)")
				continue
			}

			// Extrai title="..."
			titleIdx := strings.Index(link, `title="`)
			if titleIdx >= 0 {
				titleStart := titleIdx + 7
				rest := link[titleStart:]
				titleEnd := strings.Index(rest, `"`)
				if titleEnd > 0 {
					title := rest[:titleEnd]
					fmt.Printf("    -> TÍTULO EXTRAÍDO: %s\n", title)
					break
				}
			}
		}

		fmt.Println()
		count++
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
