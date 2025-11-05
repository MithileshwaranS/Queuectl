package job

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// State represents the current state of a job
type State string

const (
	StatePending    State = "pending"
	StateProcessing State = "processing"
	StateCompleted  State = "completed"
	StateFailed     State = "failed"
	StateDead       State = "dead"
)

// Job represents a background job to be executed
type Job struct {
	ID         string    `json:"id"`
	Command    string    `json:"command"`
	State      State     `json:"state"`
	Attempts   int       `json:"attempts"`
	MaxRetries int       `json:"max_retries"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
	WorkerID   string    `json:"worker_id,omitempty"`
	Error      string    `json:"error,omitempty"`
	Output     string    `json:"output,omitempty"`
}

// NewJob creates a new job with default values
func NewJob(command string, maxRetries int) *Job {
	now := time.Now().UTC()
	return &Job{
		ID:         uuid.New().String(),
		Command:    command,
		State:      StatePending,
		Attempts:   0,
		MaxRetries: maxRetries,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// FromJSON creates a job from JSON string
func FromJSON(data string) (*Job, error) {
	var job Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		return nil, fmt.Errorf("failed to parse job JSON: %w", err)
	}

	// Set defaults if not provided
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.State == "" {
		job.State = StatePending
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3 // default
	}
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}

	return &job, nil
}

// ToJSON converts job to JSON string
func (j *Job) ToJSON() (string, error) {
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal job to JSON: %w", err)
	}
	return string(data), nil
}

// Validate checks if the job is valid
func (j *Job) Validate() error {
	if j.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	if j.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	return nil
}

// CanRetry checks if the job can be retried
func (j *Job) CanRetry() bool {
	return j.Attempts < j.MaxRetries
}

// ShouldRetryNow checks if the job should be retried now
func (j *Job) ShouldRetryNow() bool {
	if j.NextRetryAt == nil {
		return true
	}
	return time.Now().After(*j.NextRetryAt)
}

// MarkAsProcessing marks the job as being processed by a worker
func (j *Job) MarkAsProcessing(workerID string) {
	j.State = StateProcessing
	j.WorkerID = workerID
	j.UpdatedAt = time.Now()
}

// MarkAsCompleted marks the job as successfully completed
func (j *Job) MarkAsCompleted(output string) {
	j.State = StateCompleted
	j.Output = output
	j.UpdatedAt = time.Now()
}

// MarkAsFailed marks the job as failed and increments attempts
func (j *Job) MarkAsFailed(errMsg string, nextRetryAt *time.Time) {
	j.State = StateFailed
	j.Error = errMsg
	j.Attempts++
	j.NextRetryAt = nextRetryAt
	j.UpdatedAt = time.Now()
	j.WorkerID = ""
}

// MarkAsDead marks the job as permanently failed
func (j *Job) MarkAsDead(errMsg string) {
	j.State = StateDead
	j.Error = errMsg
	j.UpdatedAt = time.Now()
	j.WorkerID = ""
}

// ResetForRetry resets the job to pending state for retry from DLQ
func (j *Job) ResetForRetry() {
	j.State = StatePending
	j.Attempts = 0
	j.Error = ""
	j.NextRetryAt = nil
	j.WorkerID = ""
	j.UpdatedAt = time.Now()
}