package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MithileshwaranS/queuectl/internal/job"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show summary of all job states and active workers",
		Long:  `Display a summary of job counts by state and list active workers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get job statistics
			stats, err := getStorage().GetJobStats()
			if err != nil {
				return fmt.Errorf("failed to get job stats: %w", err)
			}

			// Calculate totals
			total := 0
			for _, count := range stats {
				total += count
			}

			// Display job statistics
			fmt.Println("=== Job Queue Status ===")
			fmt.Println()
			fmt.Printf("Total Jobs: %d\n", total)
			fmt.Println()

			// Show counts for each state
			states := []job.State{
				job.StatePending,
				job.StateProcessing,
				job.StateCompleted,
				job.StateFailed,
				job.StateDead,
			}

			fmt.Println("Job States:")
			for _, state := range states {
				count := stats[state]
				icon := getStateIcon(state)
				fmt.Printf("  %s %-12s: %d\n", icon, state, count)
			}

			// Show active workers
			fmt.Println()
			fmt.Println("Active Workers:")
			workers := getActiveWorkers()
			if len(workers) == 0 {
				fmt.Println("  No active workers")
			} else {
				for _, w := range workers {
					fmt.Printf("  • Worker %s (PID: %s)\n", w.ID, w.PID)
				}
			}

			// Show configuration
			fmt.Println()
			fmt.Println("Configuration:")
			fmt.Printf("  Max Retries: %d\n", getConfig().MaxRetries)
			fmt.Printf("  Backoff Base: %.1f\n", getConfig().BackoffBase)
			fmt.Printf("  Database: %s\n", getConfig().DBPath)

			return nil
		},
	}

	return cmd
}

// getStateIcon returns an emoji/icon for each state
func getStateIcon(state job.State) string {
	switch state {
	case job.StatePending:
		return "⏳"
	case job.StateProcessing:
		return "🔄"
	case job.StateCompleted:
		return "✓"
	case job.StateFailed:
		return "⚠"
	case job.StateDead:
		return "✗"
	default:
		return "•"
	}
}

// Worker represents an active worker process
type Worker struct {
	ID  string
	PID string
}

// getActiveWorkers reads worker PIDs from filesystem
func getActiveWorkers() []Worker {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	workerDir := filepath.Join(homeDir, ".queuectl", "workers")
	entries, err := os.ReadDir(workerDir)
	if err != nil {
		return nil
	}

	var workers []Worker
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pid") {
			pidData, err := os.ReadFile(filepath.Join(workerDir, entry.Name()))
			if err != nil {
				continue
			}

			pid := strings.TrimSpace(string(pidData))
			// Check if process is still running
			if isProcessRunning(pid) {
				workerID := strings.TrimSuffix(entry.Name(), ".pid")
				workers = append(workers, Worker{
					ID:  workerID,
					PID: pid,
				})
			}
		}
	}

	return workers
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

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	// to check if process actually exists
	err = process.Signal(os.Signal(nil))
	return err == nil
}