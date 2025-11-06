package cli

import (
	"fmt"

	"github.com/MithileshwaranS/queuectl/internal/worker"
	"github.com/spf13/cobra"
)

func workerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Manage worker processes",
		Long:  `Start, stop, and manage background worker processes that execute jobs.`,
	}

	cmd.AddCommand(workerStartCmd())
	cmd.AddCommand(workerStopCmd())

	return cmd
}

func workerStartCmd() *cobra.Command {
	var count int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start worker processes",
		Long: `Start one or more worker processes to execute jobs from the queue.

Workers will run in the foreground and can be stopped with Ctrl+C.
They will gracefully finish any currently processing jobs before exiting.

Examples:
  queuectl worker start              # Start 1 worker (default)
  queuectl worker start --count 3    # Start 3 workers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if count < 1 {
				return fmt.Errorf("worker count must be at least 1")
			}

			// Cleanup any orphaned PID files from previous runs
			if err := worker.CleanupOrphanedPIDs(); err != nil {
				fmt.Printf("Warning: Failed to cleanup old PID files: %v\n", err)
			}

			// Create worker pool
			pool := worker.NewPool(getStorage(), getConfig(), count)

			// Start workers
			if err := pool.Start(); err != nil {
				return fmt.Errorf("failed to start workers: %w", err)
			}

			// Wait for shutdown signal
			pool.Wait()

			return nil
		},
	}

	cmd.Flags().IntVarP(&count, "count", "c", 1, "Number of workers to start")

	return cmd
}

func workerStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop running workers",
		Long: `Stop all running worker processes gracefully.

This command will signal workers to stop processing new jobs and wait
for currently executing jobs to complete before shutting down.

Note: This is primarily for documentation. In practice, workers are
stopped by pressing Ctrl+C in the terminal where they're running.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("To stop workers, press Ctrl+C in the terminal where they are running.")
			fmt.Println("Workers will gracefully finish their current jobs before stopping.")
			return nil
		},
	}

	return cmd
}
