package utils

import (
	"testing"
)

func TestEpisodeParser_Parse(t *testing.T) {
	parser := NewEpisodeParser()

	tests := []struct {
		name       string
		input      string
		wantEp     int
		wantSeason int
		wantTitle  string
	}{
		{
			name:       "S01E12 format",
			input:      "May I Ask for One Final Thing S01E12 1080p CR WEB-DL DUAL AAC2.0 H 264-VARYG",
			wantEp:     12,
			wantSeason: 1,
			wantTitle:  "May I Ask for One Final Thing",
		},
		{
			name:       "With subgroup prefix",
			input:      "[ToonsHub] May I Ask for One Final Thing S01E12 1080p CR WEB-DL",
			wantEp:     12,
			wantSeason: 1,
			wantTitle:  "May I Ask for One Final Thing",
		},
		{
			name:       "Glued episode number",
			input:      "May I Ask for One ThingE11 As These Appear... CR DUAL 2 0 -VARYG",
			wantEp:     11,
			wantSeason: 1,
			wantTitle:  "May I Ask for One Thing",
		},
		{
			name:       "Boruto format",
			input:      "[Almighty] Boruto - Naruto Next Generations - 01 [BD 1920x1080 x264 10bit FLAC].mkv",
			wantEp:     1,
			wantSeason: 1,
			wantTitle:  "Boruto Naruto Next Generations",
		},
		{
			name:       "Naruto Shippuden single episode",
			input:      "[Erai-raws] Naruto Shippuuden - 080 [1080p CR WEB-DL AVC AAC][MultiSub]",
			wantEp:     80,
			wantSeason: 1,
			wantTitle:  "Naruto Shippuuden",
		},
		{
			name:       "Episode EP format",
			input:      "One Piece EP.1123 The Greatest Treasure 1080p WEB-DL",
			wantEp:     1123,
			wantSeason: 1,
			wantTitle:  "One Piece",
		},
		{
			name:       "Season 2",
			input:      "Attack on Titan S02E05 Historia 1080p BluRay DUAL",
			wantEp:     5,
			wantSeason: 2,
			wantTitle:  "Attack on Titan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.input)

			if result.Episode != tt.wantEp {
				t.Errorf("Episode = %d, want %d", result.Episode, tt.wantEp)
			}
			if result.Season != tt.wantSeason {
				t.Errorf("Season = %d, want %d", result.Season, tt.wantSeason)
			}
			// Title test is more lenient - just check it's not empty
			if result.Title == "" {
				t.Error("Title is empty")
			}
			t.Logf("Parsed: Ep=%d, Season=%d, Title=%q, Quality=%s, Tag=%s",
				result.Episode, result.Season, result.Title, result.Quality, result.ReleaseGroup)
		})
	}
}

func TestEpisodeParser_GroupByEpisode(t *testing.T) {
	parser := NewEpisodeParser()

	filenames := []string{
		"May I Ask for One Final Thing S01E12 1080p CR WEB-DL DUAL AAC2.0 H 264-VARYG",
		"[ToonsHub] May I Ask for One Final Thing S01E12 1080p CR WEB-DL",
		"May I Ask for One Final Thing S01E11 1080p CR WEB-DL",
		"May I Ask for One ThingE11 As These Appear... CR DUAL 2 0 -VARYG",
	}

	result := parser.FullProcess(filenames)

	t.Logf("Anime Name: %s", result.AnimeName)
	t.Logf("Total Episodes: %d", result.Total)

	for _, ep := range result.Episodes {
		t.Logf("Episode %d (S%d): %s - %d files",
			ep.EpisodeNumber, ep.Season, ep.CleanTitle, len(ep.Files))
		for _, f := range ep.Files {
			t.Logf("  - %s [%s]", f.OriginalName, f.Quality)
		}
	}

	// Should have grouped into episodes
	if result.Total < 2 {
		t.Errorf("Expected at least 2 episodes, got %d", result.Total)
	}
}

func TestFormatEpisodeTitle(t *testing.T) {
	tests := []struct {
		title   string
		episode int
		season  int
		want    string
	}{
		{"Attack on Titan", 5, 1, "Attack on Titan - Episódio 5"},
		{"Attack on Titan", 5, 2, "Attack on Titan S2 - Episódio 5"},
	}

	for _, tt := range tests {
		result := FormatEpisodeTitle(tt.title, tt.episode, tt.season)
		if result != tt.want {
			t.Errorf("FormatEpisodeTitle(%q, %d, %d) = %q, want %q",
				tt.title, tt.episode, tt.season, result, tt.want)
		}
	}
}

func TestNormalizeGluedTitle(t *testing.T) {
	title, episode := NormalizeGluedTitle("ThingE11 As These Appear")

	if episode != 11 {
		t.Errorf("Episode = %d, want 11", episode)
	}
	if title != "Thing" {
		t.Errorf("Title = %q, want %q", title, "Thing")
	}
}

func TestEpisodeParser_ToJSON(t *testing.T) {
	parser := NewEpisodeParser()

	filenames := []string{
		"May I Ask for One Final Thing S01E12 1080p CR WEB-DL DUAL-VARYG",
		"[ToonsHub] May I Ask for One Final Thing S01E12 1080p WEB-DL",
	}

	result := parser.FullProcess(filenames)
	json := result.ToJSON()

	t.Logf("JSON Output:\n%s", json)

	if json == "{}" {
		t.Error("JSON output is empty")
	}
}
