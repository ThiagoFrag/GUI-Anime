// seeding_handlers.go - Handlers para o sistema de seeding comunitário
// Adicione este código ao gateway server

package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ============================================
// TIPOS PARA SEEDING
// ============================================

// SeedingJob representa um job de encoding
type SeedingJob struct {
	ID          int64     `json:"id"`
	EpisodeID   int64     `json:"episode_id,omitempty"`
	AnimeName   string    `json:"anime_name"`
	EpisodeNum  int       `json:"episode_num"`
	FileURL     string    `json:"file_url"`
	FileSize    int64     `json:"file_size"`
	Status      string    `json:"status"` // pending, assigned, processing, completed, error
	AssignedTo  string    `json:"assigned_to,omitempty"`
	AssignedAt  time.Time `json:"assigned_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	GoFileURL   string    `json:"gofile_url,omitempty"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============================================
// HANDLERS DE SEEDING (adicionar ao handleTunnel switch)
// ============================================

// handleClaimEncodeJob atribui um job de encoding a um seeder
func handleClaimEncodeJob(db *sql.DB, payload map[string]interface{}) map[string]interface{} {
	clientID, _ := payload["client_id"].(string)
	_, _ = payload["has_ffmpeg"].(bool) // Para uso futuro
	maxSizeMB, _ := payload["max_size_mb"].(float64)

	if clientID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "client_id obrigatório",
		}
	}

	// Limite de tamanho
	if maxSizeMB <= 0 {
		maxSizeMB = 2000 // 2GB padrão
	}
	maxSizeBytes := int64(maxSizeMB * 1024 * 1024)

	// Busca próximo job pendente que cabe no limite
	var job SeedingJob
	err := db.QueryRow(`
		SELECT id, anime_name, episode_num, file_url, file_size, status, created_at
		FROM seeding_jobs
		WHERE status = 'pending' AND file_size <= $1
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, maxSizeBytes).Scan(&job.ID, &job.AnimeName, &job.EpisodeNum, &job.FileURL, &job.FileSize, &job.Status, &job.CreatedAt)

	if err == sql.ErrNoRows {
		return map[string]interface{}{
			"success": false,
			"message": "Nenhum job disponível",
		}
	}
	if err != nil {
		log.Printf("[Seeding] Erro ao buscar job: %v", err)
		return map[string]interface{}{
			"success": false,
			"message": "Erro interno",
		}
	}

	// Atribui job ao seeder
	_, err = db.Exec(`
		UPDATE seeding_jobs 
		SET status = 'assigned', assigned_to = $1, assigned_at = NOW()
		WHERE id = $2
	`, clientID, job.ID)

	if err != nil {
		log.Printf("[Seeding] Erro ao atribuir job: %v", err)
		return map[string]interface{}{
			"success": false,
			"message": "Erro ao atribuir job",
		}
	}

	// Atualiza status do usuário como seeding ativo
	db.Exec(`UPDATE social_users SET seeding_active = true WHERE user_id = $1`, clientID)

	log.Printf("[Seeding] Job %d atribuído a %s: %s Ep %d", job.ID, clientID, job.AnimeName, job.EpisodeNum)

	return map[string]interface{}{
		"success": true,
		"job": map[string]interface{}{
			"id":           fmt.Sprint(job.ID),
			"anime_name":   job.AnimeName,
			"episode":      job.EpisodeNum,
			"stream_url":   job.FileURL,
			"file_size":    job.FileSize,
			"torrent_hash": "", // Se aplicável
		},
	}
}

// handleCompleteEncodeJob marca job como completo
func handleCompleteEncodeJob(db *sql.DB, payload map[string]interface{}) map[string]interface{} {
	jobID, _ := payload["job_id"].(string)
	clientID, _ := payload["client_id"].(string)
	gofileURL, _ := payload["gofile_url"].(string)

	if jobID == "" || clientID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "job_id e client_id obrigatórios",
		}
	}

	// Atualiza job
	result, err := db.Exec(`
		UPDATE seeding_jobs 
		SET status = 'completed', gofile_url = $1, completed_at = NOW()
		WHERE id = $2 AND assigned_to = $3
	`, gofileURL, jobID, clientID)

	if err != nil {
		log.Printf("[Seeding] Erro ao completar job: %v", err)
		return map[string]interface{}{
			"success": false,
			"message": "Erro interno",
		}
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return map[string]interface{}{
			"success": false,
			"message": "Job não encontrado ou não pertence ao cliente",
		}
	}

	// Busca tamanho do arquivo para atualizar stats do usuário
	var fileSize int64
	db.QueryRow(`SELECT file_size FROM seeding_jobs WHERE id = $1`, jobID).Scan(&fileSize)

	// Atualiza bytes semeados do usuário
	db.Exec(`UPDATE social_users SET seeding_bytes = seeding_bytes + $1 WHERE user_id = $2`, fileSize, clientID)

	log.Printf("[Seeding] ✓ Job %s completo por %s: %s", jobID, clientID, gofileURL)

	return map[string]interface{}{
		"success": true,
		"message": "Job completado com sucesso",
	}
}

// handleFailEncodeJob marca job como falho
func handleFailEncodeJob(db *sql.DB, payload map[string]interface{}) map[string]interface{} {
	jobID, _ := payload["job_id"].(string)
	clientID, _ := payload["client_id"].(string)
	errorMsg, _ := payload["error"].(string)

	if jobID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "job_id obrigatório",
		}
	}

	// Volta job para pending para outro seeder tentar
	_, err := db.Exec(`
		UPDATE seeding_jobs 
		SET status = 'pending', assigned_to = NULL, assigned_at = NULL, error_msg = $1
		WHERE id = $2
	`, errorMsg, jobID)

	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "Erro interno",
		}
	}

	log.Printf("[Seeding] Job %s falhou (%s): %s", jobID, clientID, errorMsg)

	return map[string]interface{}{
		"success": true,
		"message": "Job devolvido à fila",
	}
}

// handleGetSeedingStats retorna estatísticas de seeding
func handleGetSeedingStats(db *sql.DB, payload map[string]interface{}) map[string]interface{} {
	clientID, _ := payload["client_id"].(string)

	stats := map[string]interface{}{}

	// Stats globais
	var totalSeeders, activeSeeders int
	var totalBytes int64
	db.QueryRow(`SELECT COUNT(*) FROM social_users WHERE seeding_bytes > 0`).Scan(&totalSeeders)
	db.QueryRow(`SELECT COUNT(*) FROM social_users WHERE seeding_active = true`).Scan(&activeSeeders)
	db.QueryRow(`SELECT COALESCE(SUM(seeding_bytes), 0) FROM social_users`).Scan(&totalBytes)

	stats["total_seeders"] = totalSeeders
	stats["active_seeders"] = activeSeeders
	stats["total_bytes"] = totalBytes

	// Stats por status de jobs
	rows, _ := db.Query(`SELECT status, COUNT(*) FROM seeding_jobs GROUP BY status`)
	jobStats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		rows.Scan(&status, &count)
		jobStats[status] = count
	}
	rows.Close()
	stats["jobs_by_status"] = jobStats

	// Stats do cliente específico
	if clientID != "" {
		var userBytes int64
		var userJobs int
		db.QueryRow(`SELECT COALESCE(seeding_bytes, 0) FROM social_users WHERE user_id = $1`, clientID).Scan(&userBytes)
		db.QueryRow(`SELECT COUNT(*) FROM seeding_jobs WHERE assigned_to = $1 AND status = 'completed'`, clientID).Scan(&userJobs)

		stats["user_bytes"] = userBytes
		stats["user_jobs"] = userJobs
	}

	return map[string]interface{}{
		"success": true,
		"stats":   stats,
	}
}

// ============================================
// ADICIONAR AO SWITCH DO handleTunnel:
// ============================================
/*
No handleTunnel, adicione estes cases ao switch de ações:

case "claim_encode_job":
	response = handleClaimEncodeJob(db, payload)

case "complete_encode_job":
	response = handleCompleteEncodeJob(db, payload)

case "fail_encode_job":
	response = handleFailEncodeJob(db, payload)

case "get_seeding_stats":
	response = handleGetSeedingStats(db, payload)
*/

// ============================================
// CRIAR TABELA seeding_jobs (executar no PostgreSQL)
// ============================================
/*
CREATE TABLE IF NOT EXISTS seeding_jobs (
    id BIGSERIAL PRIMARY KEY,
    episode_id BIGINT,
    anime_name VARCHAR(500) NOT NULL,
    episode_num INTEGER NOT NULL,
    file_url TEXT NOT NULL,
    file_size BIGINT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    assigned_to VARCHAR(64),
    assigned_at TIMESTAMP,
    completed_at TIMESTAMP,
    gofile_url TEXT,
    error_msg TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_seeding_jobs_status ON seeding_jobs(status);
CREATE INDEX IF NOT EXISTS idx_seeding_jobs_assigned ON seeding_jobs(assigned_to);

-- Adicionar colunas ao social_users se não existirem
ALTER TABLE social_users ADD COLUMN IF NOT EXISTS seeding_active BOOLEAN DEFAULT FALSE;
ALTER TABLE social_users ADD COLUMN IF NOT EXISTS seeding_bytes BIGINT DEFAULT 0;
*/
