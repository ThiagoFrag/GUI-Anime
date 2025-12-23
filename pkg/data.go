package store

import (
	"encoding/json"
	"os"
	// ATENÇÃO: NÃO ADICIONE NENHUM OUTRO IMPORT AQUI
)

// SavedAnime é a estrutura pública (com letra Maiúscula)
type SavedAnime struct {
	Title string `json:"title"`
	Image string `json:"image"`
	URL   string `json:"url"`
}

type UserData struct {
	Username  string       `json:"username"`
	Avatar    string       `json:"avatar"`
	History   []SavedAnime `json:"history"`
	Favorites []SavedAnime `json:"favorites"`
}

const dbFile = "goanime_user.json"

func LoadUser() *UserData {
	data, err := os.ReadFile(dbFile)
	if err != nil {
		return nil
	}
	var user UserData
	if err := json.Unmarshal(data, &user); err != nil {
		return nil
	}
	return &user
}

func SaveUser(user *UserData) error {
	data, _ := json.MarshalIndent(user, "", "  ")
	return os.WriteFile(dbFile, data, 0600)
}
