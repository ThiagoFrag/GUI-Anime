package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// ============================================
// ADMIN DASHBOARD SERVER
// ============================================

var (
	// Configuração via env ou defaults
	AdminPort    = getEnvOrDefault("ADMIN_PORT", "9090")
	AdminUser    = getEnvOrDefault("ADMIN_USER", "admin")
	AdminPass    = getEnvOrDefault("ADMIN_PASS", "goanime2024")
	DBConnString = getEnvOrDefault("DATABASE_URL", "postgres://goanime:4f1bb8b37450cb30ffcef24e4ca28586@localhost:5432/goanime_connect?sslmode=disable")
)

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// AdminServer servidor de administração
type AdminServer struct {
	db           *sql.DB
	sessions     map[string]*AdminSession
	sessionMu    sync.RWMutex
	seedingStats *SeedingStatsCache
}

// AdminSession sessão de admin
type AdminSession struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IP        string    `json:"ip"`
}

// SeedingStatsCache cache de estatísticas de seeding
type SeedingStatsCache struct {
	mu              sync.RWMutex
	TotalSeeders    int            `json:"total_seeders"`
	ActiveSeeders   int            `json:"active_seeders"`
	TotalUploaded   int64          `json:"total_uploaded_bytes"`
	JobsCompleted   int            `json:"jobs_completed"`
	JobsPending     int            `json:"jobs_pending"`
	SeedersByStatus map[string]int `json:"seeders_by_status"`
	LastUpdated     time.Time      `json:"last_updated"`
}

// UserInfo informação completa do usuário
type UserInfo struct {
	ID            int        `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email,omitempty"`
	FriendToken   string     `json:"friend_token,omitempty"`
	IsOnline      bool       `json:"is_online"`
	LastSeen      time.Time  `json:"last_seen"`
	TotalWatched  int        `json:"total_watched"`
	IsVIP         bool       `json:"is_vip"`
	IsPremium     bool       `json:"is_premium"`
	VIPExpiresAt  *time.Time `json:"vip_expires_at,omitempty"`
	IsBanned      bool       `json:"is_banned"`
	BanReason     string     `json:"ban_reason,omitempty"`
	SeedingActive bool       `json:"seeding_active"`
	SeedingBytes  int64      `json:"seeding_bytes"`
	CreatedAt     time.Time  `json:"created_at"`
}

// DashboardStats estatísticas do dashboard
type DashboardStats struct {
	TotalUsers      int       `json:"total_users"`
	OnlineUsers     int       `json:"online_users"`
	VIPUsers        int       `json:"vip_users"`
	PremiumUsers    int       `json:"premium_users"`
	BannedUsers     int       `json:"banned_users"`
	ActiveSeeders   int       `json:"active_seeders"`
	TotalSeeded     int64     `json:"total_seeded_bytes"`
	PendingEncodes  int       `json:"pending_encodes"`
	RecentRegisters int       `json:"recent_registers_24h"`
	LastUpdated     time.Time `json:"last_updated"`
}

// NewAdminServer cria servidor admin
func NewAdminServer() (*AdminServer, error) {
	log.Printf("Conectando ao banco: %s", strings.Split(DBConnString, "@")[1])

	db, err := sql.Open("postgres", DBConnString)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar DB: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao pingar DB: %w", err)
	}

	log.Println("✅ Conectado ao PostgreSQL")

	server := &AdminServer{
		db:       db,
		sessions: make(map[string]*AdminSession),
		seedingStats: &SeedingStatsCache{
			SeedersByStatus: make(map[string]int),
		},
	}

	// Inicia goroutine para limpar sessões expiradas
	go server.cleanupSessions()

	return server, nil
}

// cleanupSessions remove sessões expiradas
func (s *AdminServer) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.sessionMu.Lock()
		now := time.Now()
		for token, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, token)
			}
		}
		s.sessionMu.Unlock()
	}
}

// generateToken gera token seguro
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ============================================
// MIDDLEWARES
// ============================================

// corsMiddleware adiciona headers CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authMiddleware verifica autenticação
func (s *AdminServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		s.sessionMu.RLock()
		session, exists := s.sessions[token]
		s.sessionMu.RUnlock()

		if !exists || time.Now().After(session.ExpiresAt) {
			http.Error(w, `{"error":"Não autorizado"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "session", session)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// ============================================
// HANDLERS
// ============================================

func (s *AdminServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
		return
	}

	if req.Username != AdminUser || req.Password != AdminPass {
		http.Error(w, `{"error":"Credenciais inválidas"}`, http.StatusUnauthorized)
		return
	}

	token := generateToken()
	session := &AdminSession{
		Token:     token,
		Username:  req.Username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IP:        r.RemoteAddr,
	}

	s.sessionMu.Lock()
	s.sessions[token] = session
	s.sessionMu.Unlock()

	s.logAction(req.Username, "login", "", nil, r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
		"expires": session.ExpiresAt,
	})
}

func (s *AdminServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Admin-Token")
	s.sessionMu.Lock()
	delete(s.sessions, token)
	s.sessionMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *AdminServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	stats := DashboardStats{LastUpdated: time.Now()}

	s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_online = true AND last_seen > NOW() - INTERVAL '5 minutes'`).Scan(&stats.OnlineUsers)
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_vip = true AND (vip_expires_at IS NULL OR vip_expires_at > NOW())`).Scan(&stats.VIPUsers)
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_premium = true`).Scan(&stats.PremiumUsers)
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_banned = true`).Scan(&stats.BannedUsers)
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE seeding_active = true`).Scan(&stats.ActiveSeeders)
	s.db.QueryRow(`SELECT COALESCE(SUM(seeding_bytes), 0) FROM users`).Scan(&stats.TotalSeeded)
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&stats.RecentRegisters)
	s.db.QueryRow(`SELECT COUNT(*) FROM seeding_jobs WHERE status = 'pending'`).Scan(&stats.PendingEncodes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *AdminServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	filter := r.URL.Query().Get("filter")
	limit := 50

	query := `
		SELECT 
			id, username, email, friend_token,
			COALESCE(is_online, false), COALESCE(last_seen, created_at), 
			COALESCE(total_watched, 0),
			COALESCE(is_vip, false), vip_expires_at,
			COALESCE(is_premium, false),
			COALESCE(is_banned, false), COALESCE(ban_reason, ''),
			COALESCE(seeding_active, false), COALESCE(seeding_bytes, 0),
			created_at
		FROM users WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	if search != "" {
		query += fmt.Sprintf(` AND (username ILIKE $%d OR email ILIKE $%d OR friend_token ILIKE $%d)`, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	switch filter {
	case "online":
		query += ` AND is_online = true AND last_seen > NOW() - INTERVAL '5 minutes'`
	case "vip":
		query += ` AND (is_vip = true OR is_premium = true)`
	case "banned":
		query += ` AND is_banned = true`
	case "seeding":
		query += ` AND seeding_active = true`
	}

	query += ` ORDER BY COALESCE(last_seen, created_at) DESC NULLS LAST LIMIT $` + fmt.Sprint(argIdx)
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("Erro ao buscar usuários: %v", err)
		http.Error(w, `{"error":"Erro ao buscar usuários"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []UserInfo{}
	for rows.Next() {
		var u UserInfo
		var vipExpires sql.NullTime
		var lastSeen sql.NullTime
		var friendToken sql.NullString
		var email sql.NullString

		err := rows.Scan(
			&u.ID, &u.Username, &email, &friendToken,
			&u.IsOnline, &lastSeen, &u.TotalWatched,
			&u.IsVIP, &vipExpires, &u.IsPremium,
			&u.IsBanned, &u.BanReason,
			&u.SeedingActive, &u.SeedingBytes, &u.CreatedAt,
		)
		if err != nil {
			log.Printf("Erro scan: %v", err)
			continue
		}

		if vipExpires.Valid {
			u.VIPExpiresAt = &vipExpires.Time
		}
		if lastSeen.Valid {
			u.LastSeen = lastSeen.Time
		}
		if friendToken.Valid {
			u.FriendToken = friendToken.String
		}
		if email.Valid {
			u.Email = email.String
		}

		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users, "total": len(users)})
}

func (s *AdminServer) handleUserAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Action string `json:"action"`
		Reason string `json:"reason,omitempty"`
		Days   int    `json:"days,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
		return
	}

	session := r.Context().Value("session").(*AdminSession)
	var err error

	switch req.Action {
	case "ban":
		_, err = s.db.Exec(`UPDATE users SET is_banned = true, ban_reason = $1 WHERE id = $2`, req.Reason, req.UserID)
	case "unban":
		_, err = s.db.Exec(`UPDATE users SET is_banned = false, ban_reason = '' WHERE id = $1`, req.UserID)
	case "vip":
		days := req.Days
		if days <= 0 {
			days = 30
		}
		_, err = s.db.Exec(`UPDATE users SET is_vip = true, vip_expires_at = NOW() + $1 * INTERVAL '1 day' WHERE id = $2`, days, req.UserID)
	case "unvip":
		_, err = s.db.Exec(`UPDATE users SET is_vip = false, vip_expires_at = NULL WHERE id = $1`, req.UserID)
	case "delete":
		s.db.Exec(`DELETE FROM friendships WHERE user_id = $1 OR friend_id = $1`, req.UserID)
		s.db.Exec(`DELETE FROM watching_status WHERE user_id = $1`, req.UserID)
		_, err = s.db.Exec(`DELETE FROM users WHERE id = $1`, req.UserID)
	case "reset_seeding":
		_, err = s.db.Exec(`UPDATE users SET seeding_bytes = 0, seeding_active = false WHERE id = $1`, req.UserID)
	default:
		http.Error(w, `{"error":"Ação desconhecida"}`, http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Erro %s: %v", req.Action, err)
		http.Error(w, `{"error":"Erro ao executar"}`, http.StatusInternalServerError)
		return
	}

	s.logAction(session.Username, req.Action, req.UserID, map[string]interface{}{"reason": req.Reason, "days": req.Days}, r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *AdminServer) handleSeedingStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{}
	var activeSeeders, totalSeeders int
	var totalBytes int64

	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE seeding_active = true`).Scan(&activeSeeders)
	s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE seeding_bytes > 0`).Scan(&totalSeeders)
	s.db.QueryRow(`SELECT COALESCE(SUM(seeding_bytes), 0) FROM users`).Scan(&totalBytes)

	stats["active_seeders"] = activeSeeders
	stats["total_seeders"] = totalSeeders
	stats["total_bytes"] = totalBytes

	jobStats := make(map[string]int)
	rows, _ := s.db.Query(`SELECT status, COUNT(*) FROM seeding_jobs GROUP BY status`)
	if rows != nil {
		for rows.Next() {
			var status string
			var count int
			rows.Scan(&status, &count)
			jobStats[status] = count
		}
		rows.Close()
	}
	stats["jobs_by_status"] = jobStats

	topSeeders := []map[string]interface{}{}
	rows, _ = s.db.Query(`SELECT username, COALESCE(seeding_bytes, 0), COALESCE(seeding_active, false) FROM users WHERE seeding_bytes > 0 ORDER BY seeding_bytes DESC LIMIT 10`)
	if rows != nil {
		for rows.Next() {
			var username string
			var bytes int64
			var active bool
			rows.Scan(&username, &bytes, &active)
			topSeeders = append(topSeeders, map[string]interface{}{"username": username, "bytes": bytes, "active": active})
		}
		rows.Close()
	}
	stats["top_seeders"] = topSeeders

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *AdminServer) handleSeedingJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := s.db.Query(`SELECT id, anime_name, episode_num, file_size, status, assigned_to, created_at FROM seeding_jobs ORDER BY created_at DESC LIMIT 100`)
		if err != nil {
			http.Error(w, `{"error":"Erro"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		jobs := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var animeName string
			var epNum int
			var fileSize int64
			var jobStatus string
			var assignedTo sql.NullString
			var createdAt time.Time
			rows.Scan(&id, &animeName, &epNum, &fileSize, &jobStatus, &assignedTo, &createdAt)
			jobs = append(jobs, map[string]interface{}{"id": id, "anime_name": animeName, "episode_num": epNum, "file_size": fileSize, "status": jobStatus, "assigned_to": assignedTo.String, "created_at": createdAt})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jobs": jobs})

	case "POST":
		var job struct {
			AnimeName  string `json:"anime_name"`
			EpisodeNum int    `json:"episode_num"`
			FileURL    string `json:"file_url"`
			FileSize   int64  `json:"file_size"`
		}
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
			return
		}
		s.db.Exec(`INSERT INTO seeding_jobs (anime_name, episode_num, file_url, file_size, status) VALUES ($1, $2, $3, $4, 'pending')`, job.AnimeName, job.EpisodeNum, job.FileURL, job.FileSize)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	case "DELETE":
		jobID := r.URL.Query().Get("id")
		s.db.Exec(`DELETE FROM seeding_jobs WHERE id = $1`, jobID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func (s *AdminServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	rows, _ := s.db.Query(`SELECT admin_user, action, target_user, details, ip_address, created_at FROM admin_logs ORDER BY created_at DESC LIMIT 100`)
	logs := []map[string]interface{}{}
	if rows != nil {
		for rows.Next() {
			var adminUser, action, ip string
			var targetUser sql.NullString
			var details []byte
			var createdAt time.Time
			rows.Scan(&adminUser, &action, &targetUser, &details, &ip, &createdAt)
			var detailsMap map[string]interface{}
			json.Unmarshal(details, &detailsMap)
			logs = append(logs, map[string]interface{}{"admin": adminUser, "action": action, "target": targetUser.String, "details": detailsMap, "ip": ip, "created_at": createdAt})
		}
		rows.Close()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"logs": logs})
}

func (s *AdminServer) logAction(admin, action, target string, details map[string]interface{}, ip string) {
	detailsJSON, _ := json.Marshal(details)
	s.db.Exec(`INSERT INTO admin_logs (admin_user, action, target_user, details, ip_address) VALUES ($1, $2, $3, $4, $5)`, admin, action, target, detailsJSON, ip)
}

// ============================================
// MAIN
// ============================================

func main() {
	log.Println("🚀 Iniciando GoAnime Admin Dashboard...")

	server, err := NewAdminServer()
	if err != nil {
		log.Fatalf("Erro: %v", err)
	}
	defer server.db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/login", server.handleLogin)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/logout", server.authMiddleware(server.handleLogout))
	mux.HandleFunc("/api/dashboard", server.authMiddleware(server.handleDashboard))
	mux.HandleFunc("/api/users", server.authMiddleware(server.handleUsers))
	mux.HandleFunc("/api/users/action", server.authMiddleware(server.handleUserAction))
	mux.HandleFunc("/api/seeding/stats", server.authMiddleware(server.handleSeedingStats))
	mux.HandleFunc("/api/seeding/jobs", server.authMiddleware(server.handleSeedingJobs))
	mux.HandleFunc("/api/logs", server.authMiddleware(server.handleLogs))

	mux.Handle("/", http.FileServer(http.Dir("./dashboard")))

	handler := corsMiddleware(mux)
	addr := ":" + AdminPort

	log.Printf("🔐 http://0.0.0.0%s", addr)
	log.Printf("📝 Login: %s / %s", AdminUser, strings.Repeat("*", len(AdminPass)))
	log.Printf("🗄️ DB: %s", strings.Split(DBConnString, "@")[1])

	http.ListenAndServe(addr, handler)
}
