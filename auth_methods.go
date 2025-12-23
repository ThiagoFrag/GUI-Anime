// auth_methods.go - Métodos de autenticação para o frontend
// Expõe funcionalidades de login, registro e sessão
package main

import (
	"GoAnimeGUI/pkg/auth"
)

var authManager *auth.AuthManager

// getAuthManager retorna ou cria o gerenciador de autenticação singleton
func getAuthManager() *auth.AuthManager {
	if authManager == nil {
		authManager = auth.GetManager()
	}
	return authManager
}

// ==============================
// REGISTRO E LOGIN
// ==============================

// AuthRegister registra um novo usuário
func (a *App) AuthRegister(username, email, password, avatar string) (*auth.UserSession, error) {
	am := getAuthManager()
	return am.Register(username, email, password, avatar)
}

// AuthLogin faz login com credenciais
func (a *App) AuthLogin(username, password string) (*auth.UserSession, error) {
	am := getAuthManager()
	return am.Login(username, password)
}

// AuthLoginAsGuest entra como convidado
func (a *App) AuthLoginAsGuest() *auth.UserSession {
	am := getAuthManager()
	return am.LoginAsGuest()
}

// AuthLogout faz logout
func (a *App) AuthLogout() {
	am := getAuthManager()
	am.Logout()
}

// ==============================
// SESSÃO
// ==============================

// AuthGetSession retorna a sessão atual do usuário
func (a *App) AuthGetSession() *auth.UserSession {
	am := getAuthManager()
	return am.GetSession()
}

// AuthIsLoggedIn verifica se o usuário está logado
func (a *App) AuthIsLoggedIn() bool {
	am := getAuthManager()
	return am.IsLoggedIn()
}

// AuthIsGuest verifica se o usuário é convidado
func (a *App) AuthIsGuest() bool {
	am := getAuthManager()
	return am.IsGuest()
}

// ==============================
// PREFERÊNCIAS DE SEEDING
// ==============================

// AuthSetSeedingEnabled define se seeding está habilitado
func (a *App) AuthSetSeedingEnabled(enabled bool) error {
	am := getAuthManager()
	return am.UpdateSeedingPreference(enabled)
}

// AuthGetSeedingEnabled retorna se seeding está habilitado
func (a *App) AuthGetSeedingEnabled() bool {
	am := getAuthManager()
	session := am.GetSession()
	if session != nil {
		return session.SeedingEnabled
	}
	return false
}
