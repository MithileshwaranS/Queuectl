package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/MithileshwaranS/queuectl/internal/job"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements Storage interface using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

// Initialize creates the necessary tables
func (s *SQLiteStorage) Initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		state TEXT NOT NULL,
		attempts INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		next_retry_at DATETIME,
		worker_id TEXT,
		error TEXT,
		output TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
	CREATE INDEX IF NOT EXISTS idx_jobs_next_retry ON jobs(next_retry_at);
	CREATE INDEX IF NOT EXISTS idx_jobs_worker ON jobs(worker_id);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// SaveJob inserts or updates a job
func (s *SQLiteStorage) SaveJob(j *job.Job) error {
	query := `
	INSERT INTO jobs (id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at, worker_id, error, output)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		command = excluded.command,
		state = excluded.state,
		attempts = excluded.attempts,
		max_retries = excluded.max_retries,
		updated_at = excluded.updated_at,
		next_retry_at = excluded.next_retry_at,
		worker_id = excluded.worker_id,
		error = excluded.error,
		output = excluded.output
	`

	var nextRetryAt interface{}
	if j.NextRetryAt != nil {
		nextRetryAt = j.NextRetryAt.Format(time.RFC3339)
	}

	_, err := s.db.Exec(query,
		j.ID,
		j.Command,
		j.State,
		j.Attempts,
		j.MaxRetries,
		j.CreatedAt.Format(time.RFC3339),
		j.UpdatedAt.Format(time.RFC3339),
		nextRetryAt,
		j.WorkerID,
		j.Error,
		j.Output,
	)

	if err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (s *SQLiteStorage) GetJob(id string) (*job.Job, error) {
	query := `
	SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at, worker_id, error, output
	FROM jobs WHERE id = ?
	`

	return s.scanJob(s.db.QueryRow(query, id))
}

// GetNextPendingJob gets the next available job and locks it
func (s *SQLiteStorage) GetNextPendingJob(workerID string) (*job.Job, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Find next pending job or failed job ready for retry
	query := `
	SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at, worker_id, error, output
	FROM jobs 
	WHERE (state = ? OR (state = ? AND next_retry_at <= ?))
	ORDER BY created_at ASC
	LIMIT 1
	`

	now := time.Now().Format(time.RFC3339)
	j, err := s.scanJob(tx.QueryRow(query, job.StatePending, job.StateFailed, now))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to query next job: %w", err)
	}

	// Lock the job by updating its state
	updateQuery := `
	UPDATE jobs 
	SET state = ?, worker_id = ?, updated_at = ?
	WHERE id = ? AND (state = ? OR state = ?)
	`

	result, err := tx.Exec(updateQuery,
		job.StateProcessing,
		workerID,
		time.Now().Format(time.RFC3339),
		j.ID,
		job.StatePending,
		job.StateFailed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		// Job was already taken by another worker
		return nil, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	j.State = job.StateProcessing
	j.WorkerID = workerID
	return j, nil
}

// ListJobs returns jobs filtered by state
func (s *SQLiteStorage) ListJobs(state job.State) ([]*job.Job, error) {
	var query string
	var rows *sql.Rows
	var err error

	if state == "" {
		query = `SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at, worker_id, error, output FROM jobs ORDER BY created_at DESC`
		rows, err = s.db.Query(query)
	} else {
		query = `SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at, worker_id, error, output FROM jobs WHERE state = ? ORDER BY created_at DESC`
		rows, err = s.db.Query(query, state)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*job.Job
	for rows.Next() {
		j, err := s.scanJobFromRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// GetJobStats returns job counts by state
func (s *SQLiteStorage) GetJobStats() (map[job.State]int, error) {
	query := `SELECT state, COUNT(*) FROM jobs GROUP BY state`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[job.State]int)
	for rows.Next() {
		var state job.State
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return nil, err
		}
		stats[state] = count
	}

	return stats, rows.Err()
}

// DeleteJob removes a job
func (s *SQLiteStorage) DeleteJob(id string) error {
	query := `DELETE FROM jobs WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// GetRetryableJobs returns failed jobs ready to retry
func (s *SQLiteStorage) GetRetryableJobs() ([]*job.Job, error) {
	query := `
	SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at, worker_id, error, output
	FROM jobs 
	WHERE state = ? AND next_retry_at <= ?
	ORDER BY next_retry_at ASC
	`

	now := time.Now().Format(time.RFC3339)
	rows, err := s.db.Query(query, job.StateFailed, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get retryable jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*job.Job
	for rows.Next() {
		j, err := s.scanJobFromRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// GetDLQJobs returns all dead jobs
func (s *SQLiteStorage) GetDLQJobs() ([]*job.Job, error) {
	return s.ListJobs(job.StateDead)
}

// Helper function to scan a single job from QueryRow
func (s *SQLiteStorage) scanJob(row *sql.Row) (*job.Job, error) {
	j := &job.Job{}
	var createdAt, updatedAt string
	var nextRetryAt sql.NullString
	var workerID, errMsg, output sql.NullString

	err := row.Scan(
		&j.ID,
		&j.Command,
		&j.State,
		&j.Attempts,
		&j.MaxRetries,
		&createdAt,
		&updatedAt,
		&nextRetryAt,
		&workerID,
		&errMsg,
		&output,
	)

	if err != nil {
		return nil, err
	}

	j.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	j.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if nextRetryAt.Valid {
		t, _ := time.Parse(time.RFC3339, nextRetryAt.String)
		j.NextRetryAt = &t
	}
	if workerID.Valid {
		j.WorkerID = workerID.String
	}
	if errMsg.Valid {
		j.Error = errMsg.String
	}
	if output.Valid {
		j.Output = output.String
	}

	return j, nil
}

// Helper function to scan jobs from Rows
func (s *SQLiteStorage) scanJobFromRows(rows *sql.Rows) (*job.Job, error) {
	j := &job.Job{}
	var createdAt, updatedAt string
	var nextRetryAt sql.NullString
	var workerID, errMsg, output sql.NullString

	err := rows.Scan(
		&j.ID,
		&j.Command,
		&j.State,
		&j.Attempts,
		&j.MaxRetries,
		&createdAt,
		&updatedAt,
		&nextRetryAt,
		&workerID,
		&errMsg,
		&output,
	)

	if err != nil {
		return nil, err
	}

	j.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	j.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if nextRetryAt.Valid {
		t, _ := time.Parse(time.RFC3339, nextRetryAt.String)
		j.NextRetryAt = &t
	}
	if workerID.Valid {
		j.WorkerID = workerID.String
	}
	if errMsg.Valid {
		j.Error = errMsg.String
	}
	if output.Valid {
		j.Output = output.String
	}

	return j, nil
}