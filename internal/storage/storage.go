package storage

import (
	"github.com/MithileshwaranS/queuectl/internal/job"
)

// Storage defines the interface for job persistence
type Storage interface {
	// Initialize sets up the storage (create tables, etc.)
	Initialize() error

	// Close closes the storage connection
	Close() error

	// SaveJob creates or updates a job
	SaveJob(j *job.Job) error

	// GetJob retrieves a job by ID
	GetJob(id string) (*job.Job, error)

	// GetNextPendingJob gets the next available pending job and locks it
	// Returns nil if no jobs available
	GetNextPendingJob(workerID string) (*job.Job, error)

	// ListJobs returns all jobs matching the given state
	// If state is empty, returns all jobs
	ListJobs(state job.State) ([]*job.Job, error)

	// GetJobStats returns counts of jobs by state
	GetJobStats() (map[job.State]int, error)

	// DeleteJob removes a job by ID
	DeleteJob(id string) error

	// GetRetryableJobs returns failed jobs that are ready to retry
	GetRetryableJobs() ([]*job.Job, error)

	// GetDLQJobs returns all jobs in the dead letter queue
	GetDLQJobs() ([]*job.Job, error)
}
