package cli

import (
	"fmt"
	"strings"

	"github.com/MithileshwaranS/queuectl/internal/job"
	"github.com/spf13/cobra"
)

func dlqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dlq",
		Short: "Manage Dead Letter Queue",
		Long: `View and retry jobs in the Dead Letter Queue (DLQ).

The DLQ contains jobs that have permanently failed after exhausting
all retry attempts. These jobs require manual intervention.`,
	}

	cmd.AddCommand(dlqListCmd())
	cmd.AddCommand(dlqRetryCmd())
	cmd.AddCommand(dlqDeleteCmd())
	cmd.AddCommand(dlqClearCmd())

	return cmd
}

func dlqListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs in the Dead Letter Queue",
		Long:  `Display all jobs that have permanently failed and moved to the DLQ.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jobs, err := getStorage().GetDLQJobs()
			if err != nil {
				return fmt.Errorf("failed to list DLQ jobs: %w", err)
			}

			if len(jobs) == 0 {
				fmt.Println("✓ Dead Letter Queue is empty")
				return nil
			}

			fmt.Printf("=== Dead Letter Queue (%d jobs) ===\n\n", len(jobs))

			for i, j := range jobs {
				if i > 0 {
					fmt.Println(strings.Repeat("-", 60))
				}

				fmt.Printf("Job ID: %s\n", j.ID)
				fmt.Printf("Command: %s\n", j.Command)
				fmt.Printf("Attempts: %d/%d\n", j.Attempts, j.MaxRetries)
				fmt.Printf("Created: %s\n", j.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("Failed: %s\n", j.UpdatedAt.Format("2006-01-02 15:04:05"))

				if j.Error != "" {
					// Truncate long errors
					errMsg := j.Error
					if len(errMsg) > 300 {
						errMsg = errMsg[:300] + "..."
					}
					fmt.Printf("Error: %s\n", errMsg)
				}

				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func dlqRetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry [job-id]",
		Short: "Retry a job from the Dead Letter Queue",
		Long: `Move a job from the DLQ back to pending state for retry.

This resets the job's attempt counter and clears the error.
The job will be picked up by the next available worker.

Example:
  queuectl dlq retry abc123-def456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]

			// Get the job
			j, err := getStorage().GetJob(jobID)
			if err != nil {
				return fmt.Errorf("failed to get job: %w", err)
			}

			// Verify it's in DLQ
			if j.State != job.StateDead {
				return fmt.Errorf("job %s is not in the Dead Letter Queue (current state: %s)", jobID, j.State)
			}

			// Reset for retry
			j.ResetForRetry()

			// Save updated job
			if err := getStorage().SaveJob(j); err != nil {
				return fmt.Errorf("failed to retry job: %w", err)
			}

			fmt.Printf("✓ Job %s moved from DLQ to pending queue\n", jobID)
			fmt.Println("  The job will be picked up by the next available worker")

			return nil
		},
	}

	return cmd
}

func dlqDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [job-id]",
		Short: "Delete a job from the Dead Letter Queue",
		Long: `Permanently delete a job from the DLQ.

Warning: This action cannot be undone.

Example:
  queuectl dlq delete abc123-def456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]

			// Get the job first to verify it exists and is in DLQ
			j, err := getStorage().GetJob(jobID)
			if err != nil {
				return fmt.Errorf("failed to get job: %w", err)
			}

			if j.State != job.StateDead {
				return fmt.Errorf("job %s is not in the Dead Letter Queue (current state: %s)", jobID, j.State)
			}

			// Delete the job
			if err := getStorage().DeleteJob(jobID); err != nil {
				return fmt.Errorf("failed to delete job: %w", err)
			}

			fmt.Printf("✓ Job %s permanently deleted from DLQ\n", jobID)

			return nil
		},
	}

	return cmd
}

func dlqClearCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all jobs from the Dead Letter Queue",
		Long: `Delete all jobs from the DLQ.

Warning: This action cannot be undone. Use --force to confirm.

Example:
  queuectl dlq clear --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				return fmt.Errorf("this action requires --force flag to confirm")
			}

			// Get all DLQ jobs
			jobs, err := getStorage().GetDLQJobs()
			if err != nil {
				return fmt.Errorf("failed to list DLQ jobs: %w", err)
			}

			if len(jobs) == 0 {
				fmt.Println("✓ Dead Letter Queue is already empty")
				return nil
			}

			// Delete all jobs
			deletedCount := 0
			for _, j := range jobs {
				if err := getStorage().DeleteJob(j.ID); err != nil {
					fmt.Printf("Warning: Failed to delete job %s: %v\n", j.ID, err)
					continue
				}
				deletedCount++
			}

			fmt.Printf("✓ Cleared %d job(s) from Dead Letter Queue\n", deletedCount)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Confirm deletion of all DLQ jobs")

	return cmd
}
