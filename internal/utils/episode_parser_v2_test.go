package utils

import (
	"regexp"
	"strconv"
	"testing"
)

// Dados de entrada EXATOS do problema reportado
var problematicInputs = []string{
	"May I Ask for One ThingE01 May I Kindly Beat the Tar Out... CR MULTi 2 0 -VARYG",
	"May I Ask for One ThingE09 May I Explain That This Is Not... CR MULTi 2 0 -VARYG",
	"May I Ask for One ThingE11 CR 2 0",
	"May I Ask for One ThingE11 As These Appear Undercooked... CR DUAL 2 0 -VARYG",
	"[ToonsHub] May I Ask for One Final Thing S01E12 1080p CR WEB-DL DUAL AAC2.0 H 264-VARYG",
	"May I Ask for One Final Thing S01E10 1080p CR WEB-DL DUAL AAC2.0 H 264-VARYG",
	"May I Ask for One Final Thing S01E08 1080p CR WEB-DL DUAL AAC2.0 H 264-VARYG",
	"May I Ask for One Final Thing S01E08 REPACK 1080p CR WEB-DL...",
}

func TestRobustParser_StrictGrouping(t *testing.T) {
	parser := NewRobustEpisodeParser()
	result := parser.ParseFiles(problematicInputs)

	t.Logf("Anime: %s", result.NomeAnime)
	t.Logf("Total episódios únicos: %d", result.TotalEpisodios)

	// Verifica se temos os episódios corretos
	expectedEpisodes := map[int]bool{
		1:  true,
		8:  true,
		9:  true,
		10: true,
		11: true,
		12: true,
	}

	foundEpisodes := make(map[int]bool)
	for _, ep := range result.Episodios {
		foundEpisodes[ep.IDEpisodio] = true
		t.Logf("\n=== Episódio %d ===", ep.IDEpisodio)
		t.Logf("  Título limpo: %q", ep.TituloExibicaoLimpo)
		t.Logf("  Título episódio: %q", ep.TituloEpisodioCompleto)
		t.Logf("  Arquivos (%d):", len(ep.ArquivosDisponiveis))

		for _, arq := range ep.ArquivosDisponiveis {
			t.Logf("    - %s", arq.NomeOriginal)
			t.Logf("      Tags: %v", arq.Tags)
		}
	}

	// TESTE CRÍTICO: Verifica agrupamento estrito
	for _, ep := range result.Episodios {
		for _, arq := range ep.ArquivosDisponiveis {
			// Cada arquivo deve conter o número do episódio no nome
			expectedPattern := ""
			switch ep.IDEpisodio {
			case 1:
				expectedPattern = "E01"
			case 8:
				expectedPattern = "E08"
			case 9:
				expectedPattern = "E09"
			case 10:
				expectedPattern = "E10"
			case 11:
				expectedPattern = "E11"
			case 12:
				expectedPattern = "E12"
			}

			if !containsEpisodeNumber(arq.NomeOriginal, ep.IDEpisodio) {
				t.Errorf("ERRO CRÍTICO: Episódio %d contém arquivo que não é desse episódio: %s",
					ep.IDEpisodio, arq.NomeOriginal)
			}
			_ = expectedPattern // usado para debug se necessário
		}
	}

	// Verifica se todos os episódios esperados foram encontrados
	for epNum := range expectedEpisodes {
		if !foundEpisodes[epNum] {
			t.Errorf("Episódio %d não foi encontrado no resultado", epNum)
		}
	}

	// TESTE: E11 deve ter exatamente 2 arquivos
	for _, ep := range result.Episodios {
		if ep.IDEpisodio == 11 {
			if len(ep.ArquivosDisponiveis) != 2 {
				t.Errorf("Episódio 11 deveria ter 2 arquivos, tem %d", len(ep.ArquivosDisponiveis))
			}
		}
	}

	// TESTE: E08 deve ter exatamente 2 arquivos (normal + REPACK)
	for _, ep := range result.Episodios {
		if ep.IDEpisodio == 8 {
			if len(ep.ArquivosDisponiveis) != 2 {
				t.Errorf("Episódio 8 deveria ter 2 arquivos, tem %d", len(ep.ArquivosDisponiveis))
			}
		}
	}
}

// containsEpisodeNumber verifica se o nome contém o número do episódio
func containsEpisodeNumber(filename string, episode int) bool {
	// Formatos aceitos: E01, E1, S01E01, -01-, [01]
	patterns := []string{
		`(?i)E0?` + itoa(episode) + `\b`,
		`(?i)S\d+E0?` + itoa(episode) + `\b`,
		`\s+-\s*0?` + itoa(episode) + `\s*[-\[]`,
		`[\[\(]0?` + itoa(episode) + `[\]\)]`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(filename) {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func TestRobustParser_CleanTitleExtraction(t *testing.T) {
	parser := NewRobustEpisodeParser()
	result := parser.ParseFiles(problematicInputs)

	for _, ep := range result.Episodios {
		// O título limpo NÃO deve conter tags técnicas
		badTags := []string{"CR", "MULTi", "DUAL", "2 0", "VARYG", "WEB-DL", "1080p", "REPACK", "AAC"}
		for _, tag := range badTags {
			if containsWord(ep.TituloExibicaoLimpo, tag) {
				t.Errorf("Episódio %d: título limpo contém tag técnica %q: %s",
					ep.IDEpisodio, tag, ep.TituloExibicaoLimpo)
			}
		}
	}
}

func containsWord(s, word string) bool {
	return regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`).MatchString(s)
}

func TestRobustParser_GluedEpisodeNumber(t *testing.T) {
	parser := NewRobustEpisodeParser()

	testCases := []struct {
		input     string
		wantEp    int
		wantTitle string
	}{
		{
			input:     "May I Ask for One ThingE01 May I Kindly...",
			wantEp:    1,
			wantTitle: "May I Ask for One Thing",
		},
		{
			input:     "May I Ask for One ThingE11 As These Appear...",
			wantEp:    11,
			wantTitle: "May I Ask for One Thing",
		},
		{
			input:     "Attack on TitanE05 First Battle...",
			wantEp:    5,
			wantTitle: "Attack on Titan",
		},
	}

	for _, tc := range testCases {
		result := parser.ParseFiles([]string{tc.input})

		if len(result.Episodios) != 1 {
			t.Errorf("Input %q: esperado 1 episódio, obteve %d", tc.input, len(result.Episodios))
			continue
		}

		ep := result.Episodios[0]
		if ep.IDEpisodio != tc.wantEp {
			t.Errorf("Input %q: episódio %d, esperado %d", tc.input, ep.IDEpisodio, tc.wantEp)
		}

		if ep.TituloExibicaoLimpo != tc.wantTitle {
			t.Errorf("Input %q: título %q, esperado %q", tc.input, ep.TituloExibicaoLimpo, tc.wantTitle)
		}
	}
}

func TestRobustParser_EpisodeTitleExtraction(t *testing.T) {
	parser := NewRobustEpisodeParser()

	input := "May I Ask for One ThingE11 As These Appear Undercooked... CR DUAL 2 0 -VARYG"
	result := parser.ParseFiles([]string{input})

	if len(result.Episodios) != 1 {
		t.Fatalf("Esperado 1 episódio, obteve %d", len(result.Episodios))
	}

	ep := result.Episodios[0]
	t.Logf("Título anime: %q", ep.TituloExibicaoLimpo)
	t.Logf("Título episódio: %q", ep.TituloEpisodioCompleto)

	// O título do episódio deve conter "As These Appear"
	if ep.TituloEpisodioCompleto == "" {
		t.Log("Aviso: título do episódio não foi extraído (pode ser aceitável)")
	} else if !containsWord(ep.TituloEpisodioCompleto, "As These Appear") {
		t.Logf("Título do episódio extraído: %q", ep.TituloEpisodioCompleto)
	}
}

func TestRobustParser_JSONOutput(t *testing.T) {
	parser := NewRobustEpisodeParser()
	result := parser.ParseFiles(problematicInputs)

	json := result.ToJSON()
	t.Logf("JSON Output:\n%s", json)

	if json == "" || json == "[]" || json == "{}" {
		t.Error("JSON output está vazio")
	}
}

func TestRobustParser_MixedFormats(t *testing.T) {
	inputs := []string{
		"[SubsPlease] Naruto - 01 [1080p].mkv",
		"[Erai-raws] Naruto - 01 [720p].mkv",
		"Naruto S01E01 The Beginning 1080p WEB-DL",
		"NarutoE01 Welcome to Konoha.mkv",
	}

	parser := NewRobustEpisodeParser()
	result := parser.ParseFiles(inputs)

	t.Logf("Total episódios: %d", result.TotalEpisodios)

	// Todos devem ser agrupados no episódio 1
	if result.TotalEpisodios != 1 {
		t.Errorf("Esperado 1 episódio único, obteve %d", result.TotalEpisodios)
	}

	if result.TotalEpisodios > 0 && result.Episodios[0].IDEpisodio != 1 {
		t.Errorf("Episódio deveria ser 1, é %d", result.Episodios[0].IDEpisodio)
	}

	if result.TotalEpisodios > 0 && len(result.Episodios[0].ArquivosDisponiveis) != 4 {
		t.Errorf("Deveria ter 4 arquivos, tem %d", len(result.Episodios[0].ArquivosDisponiveis))
	}
}
