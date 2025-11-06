package worker

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"github.com/MithileshwaranS/queuectl/internal/config"
	"github.com/MithileshwaranS/queuectl/internal/storage"
)

// Pool manages multiple workers
type Pool struct {
	workers []*Worker
	storage storage.Storage
	config  *config.Config
	logger  *log.Logger
	mu      sync.Mutex
}

// NewPool creates a new worker pool
func NewPool(store storage.Storage, cfg *config.Config, count int) *Pool {
	logger := log.New(os.Stdout, "", log.LstdFlags)
    
	pool := &Pool{
		workers: make([]*Worker, 0, count),
		storage: store,
		config:  cfg,
		logger:  logger,
	}

	// Create workers
	for i := 0; i < count; i++ {
		worker := NewWorker(store, cfg, logger)
		pool.workers = append(pool.workers, worker)
	}

	return pool
}

// Start starts all workers in the pool
func (p *Pool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Printf("Starting %d worker(s)...", len(p.workers))

	// Start all workers
	for _, w := range p.workers {
		w.Start()
		p.logger.Printf("Worker %s started", w.ID)
        
		// Save worker PID for tracking
		if err := p.saveWorkerPID(w.ID); err != nil {
			p.logger.Printf("Warning: Failed to save worker PID: %v", err)
		}
	}

	p.logger.Println("All workers started successfully")
	p.logger.Println("Press Ctrl+C to stop workers gracefully")

	return nil
}

// Stop stops all workers gracefully
func (p *Pool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Println("Stopping all workers...")

	// Stop all workers
	var wg sync.WaitGroup
	for _, w := range p.workers {
		wg.Add(1)
		go func(worker *Worker) {
			defer wg.Done()
			worker.Stop()
			p.removeWorkerPID(worker.ID)
		}(w)
	}

	wg.Wait()
	p.logger.Println("All workers stopped")
}

// Wait blocks until workers are stopped (by signal)
func (p *Pool) Wait() {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	p.logger.Printf("Received signal: %v", sig)

	// Stop all workers gracefully
	p.Stop()
}

// GetWorkerCount returns the number of workers
func (p *Pool) GetWorkerCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.workers)
}

// saveWorkerPID saves the worker's process ID to a file
func (p *Pool) saveWorkerPID(workerID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	workerDir := filepath.Join(homeDir, ".queuectl", "workers")
	if err := os.MkdirAll(workerDir, 0755); err != nil {
		return err
	}

	pidFile := filepath.Join(workerDir, fmt.Sprintf("%s.pid", workerID))
	pid := strconv.Itoa(os.Getpid())

	return os.WriteFile(pidFile, []byte(pid), 0644)
}

// removeWorkerPID removes the worker's PID file
func (p *Pool) removeWorkerPID(workerID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	pidFile := filepath.Join(homeDir, ".queuectl", "workers", fmt.Sprintf("%s.pid", workerID))
	return os.Remove(pidFile)
}

// CleanupOrphanedPIDs removes PID files for workers that are no longer running
func CleanupOrphanedPIDs() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	workerDir := filepath.Join(homeDir, ".queuectl", "workers")
	entries, err := os.ReadDir(workerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, nothing to clean
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		pidFile := filepath.Join(workerDir, entry.Name())
		pidData, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}

		pid := string(pidData)
		if !isProcessRunning(pid) {
			os.Remove(pidFile)
		}
	}

	return nil
}

// isProcessRunning checks if a process with given PID is running
func isProcessRunning(pid string) bool {
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pidInt)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}