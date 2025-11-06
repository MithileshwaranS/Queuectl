package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/MithileshwaranS/queuectl/internal/config"
	"github.com/MithileshwaranS/queuectl/internal/job"
	"github.com/MithileshwaranS/queuectl/internal/retry"
	"github.com/MithileshwaranS/queuectl/internal/storage"
	"github.com/google/uuid"
)

// Worker represents a background worker that processes jobs
type Worker struct {
	ID      string
	storage storage.Storage
	config  *config.Config
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  *log.Logger
}

// NewWorker creates a new worker instance
func NewWorker(store storage.Storage, cfg *config.Config, logger *log.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Worker{
		ID:      uuid.New().String()[:8], // Short ID for display
		storage: store,
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}
}

// Start begins processing jobs
func (w *Worker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.logger.Printf("[Worker %s] Stopping gracefully...", w.ID)
	w.cancel()
	w.wg.Wait()
	w.logger.Printf("[Worker %s] Stopped", w.ID)
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.wg.Done()
	
	w.logger.Printf("[Worker %s] Started", w.ID)
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.processNext()
		}
	}
}

// processNext fetches and processes the next available job
func (w *Worker) processNext() {
	// Get next pending job (with locking)
	j, err := w.storage.GetNextPendingJob(w.ID)
	if err != nil {
		w.logger.Printf("[Worker %s] Error fetching job: %v", w.ID, err)
		return
	}

	if j == nil {
		// No jobs available
		return
	}

	w.logger.Printf("[Worker %s] Processing job %s: %s", w.ID, j.ID, j.Command)
	
	// Execute the job
	w.executeJob(j)
}

// executeJob executes a single job and handles its result
func (w *Worker) executeJob(j *job.Job) {
	// Mark as processing
	j.MarkAsProcessing(w.ID)
	if err := w.storage.SaveJob(j); err != nil {
		w.logger.Printf("[Worker %s] Error saving job state: %v", w.ID, err)
		return
	}

	// Execute command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", j.Command)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		w.handleFailure(j, err, output, duration)
	} else {
		w.handleSuccess(j, output, duration)
	}
}

// handleSuccess marks job as completed
func (w *Worker) handleSuccess(j *job.Job, output string, duration time.Duration) {
	w.logger.Printf("[Worker %s] Job %s completed successfully (%.2fs)", w.ID, j.ID, duration.Seconds())
	
	j.MarkAsCompleted(output)
	
	if err := w.storage.SaveJob(j); err != nil {
		w.logger.Printf("[Worker %s] Error saving completed job: %v", w.ID, err)
	}
}

// handleFailure handles job failure with retry logic
func (w *Worker) handleFailure(j *job.Job, execErr error, output string, duration time.Duration) {
	errMsg := fmt.Sprintf("%v", execErr)
	if output != "" {
		errMsg = fmt.Sprintf("%v\nOutput: %s", execErr, output)
	}

	w.logger.Printf("[Worker %s] Job %s failed (%.2fs): %v", w.ID, j.ID, duration.Seconds(), execErr)

	// Check if we can retry
	if j.CanRetry() {
		// Calculate next retry time with exponential backoff
		nextRetryAt := retry.GetNextRetryAt(j.Attempts, w.config.BackoffBase)
		j.MarkAsFailed(errMsg, nextRetryAt)
		
		delay := nextRetryAt.Sub(time.Now())
		w.logger.Printf("[Worker %s] Job %s will retry in %s (attempt %d/%d)", 
			w.ID, j.ID, delay.Round(time.Second), j.Attempts+1, j.MaxRetries)
	} else {
		// Move to Dead Letter Queue
		j.MarkAsDead(errMsg)
		w.logger.Printf("[Worker %s] Job %s moved to DLQ after %d attempts", w.ID, j.ID, j.Attempts)
	}

	if err := w.storage.SaveJob(j); err != nil {
		w.logger.Printf("[Worker %s] Error saving failed job: %v", w.ID, err)
	}
}

// GetID returns the worker ID
func (w *Worker) GetID() string {
	return w.ID
}