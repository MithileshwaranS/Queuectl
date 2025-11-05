package cli

import (
	"fmt"
	"strings"

	"github.com/MithileshwaranS/queuectl/internal/job"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var stateFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs by state",
		Long: `List all jobs or filter by specific state.

States: pending, processing, completed, failed, dead

Examples:
  queuectl list                    # List all jobs
  queuectl list --state pending    # List only pending jobs
  queuectl list --state failed     # List failed jobs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var state job.State
			if stateFilter != "" {
				state = job.State(stateFilter)
				// Validate state
				validStates := []job.State{
					job.StatePending,
					job.StateProcessing,
					job.StateCompleted,
					job.StateFailed,
					job.StateDead,
				}
				valid := false
				for _, s := range validStates {
					if state == s {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("invalid state: %s (valid: pending, processing, completed, failed, dead)", stateFilter)
				}
			}

			// Get jobs from storage
			jobs, err := getStorage().ListJobs(state)
			if err != nil {
				return fmt.Errorf("failed to list jobs: %w", err)
			}

			// Display results
			if len(jobs) == 0 {
				if stateFilter != "" {
					fmt.Printf("No jobs found with state: %s\n", stateFilter)
				} else {
					fmt.Println("No jobs found")
				}
				return nil
			}

			// Print header
			if stateFilter != "" {
				fmt.Printf("=== Jobs (state: %s) ===\n\n", stateFilter)
			} else {
				fmt.Println("=== All Jobs ===\n")
			}

			// Print jobs
			for i, j := range jobs {
				if i > 0 {
					fmt.Println(strings.Repeat("-", 60))
				}

				icon := getStateIcon(j.State)
				fmt.Printf("Job ID: %s\n", j.ID)
				fmt.Printf("Command: %s\n", j.Command)
				fmt.Printf("State: %s %s\n", icon, j.State)
				fmt.Printf("Attempts: %d/%d\n", j.Attempts, j.MaxRetries)
				fmt.Printf("Created: %s\n", j.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("Updated: %s\n", j.UpdatedAt.Format("2006-01-02 15:04:05"))

				if j.NextRetryAt != nil {
					fmt.Printf("Next Retry: %s\n", j.NextRetryAt.Format("2006-01-02 15:04:05"))
				}

				if j.WorkerID != "" {
					fmt.Printf("Worker: %s\n", j.WorkerID)
				}

				if j.Error != "" {
					fmt.Printf("Error: %s\n", j.Error)
				}

				if j.Output != "" {
					// Truncate long output
					output := j.Output
					if len(output) > 200 {
						output = output[:200] + "..."
					}
					fmt.Printf("Output: %s\n", output)
				}

				fmt.Println()
			}

			fmt.Printf("Total: %d job(s)\n", len(jobs))

			return nil
		},
	}

	cmd.Flags().StringVarP(&stateFilter, "state", "s", "", "Filter by state (pending, processing, completed, failed, dead)")

	return cmd
}