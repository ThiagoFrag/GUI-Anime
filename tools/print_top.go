package main

import (
	"GoAnimeGUI/pkg/jikan"
	"fmt"
)

func main() {
	animes, err := jikan.FetchTopAnimes()
	if err != nil {
		fmt.Println("Erro ao buscar top animes:", err)
		return
	}
	fmt.Printf("Total: %d\n", len(animes))
	for i, a := range animes {
		fmt.Printf("%d: %s -> %s\n", i+1, a.Title, a.Image)
		if i >= 9 { // mostra apenas os 10 primeiros
			break
		}
	}
}
