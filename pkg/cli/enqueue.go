package cli

import (
	"fmt"

	"github.com/MithileshwaranS/queuectl/internal/job"
	"github.com/spf13/cobra"
)

func enqueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enqueue [job-json]",
		Short: "Add a new job to the queue",
		Long: `Enqueue a new job by providing a JSON string with job details.

Example:
  queuectl enqueue '{"command":"echo Hello World"}'
  queuectl enqueue '{"command":"sleep 5", "max_retries":5}'
  queuectl enqueue '{"id":"custom-id","command":"ls -la"}'

Job JSON fields:
  - command (required): Shell command to execute
  - id (optional): Custom job ID (auto-generated if not provided)
  - max_retries (optional): Maximum retry attempts (default: 3)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse job from JSON
			j, err := job.FromJSON(args[0])
			if err != nil {
				return fmt.Errorf("invalid job JSON: %w", err)
			}

			// Validate job
			if err := j.Validate(); err != nil {
				return fmt.Errorf("invalid job: %w", err)
			}

			// Use config default for max_retries if not specified
			if j.MaxRetries == 0 {
				j.MaxRetries = getConfig().MaxRetries
			}

			// Save to storage
			if err := getStorage().SaveJob(j); err != nil {
				return fmt.Errorf("failed to enqueue job: %w", err)
			}

			// Print success with job details
			fmt.Printf("✓ Job enqueued successfully\n")
			fmt.Printf("  ID: %s\n", j.ID)
			fmt.Printf("  Command: %s\n", j.Command)
			fmt.Printf("  State: %s\n", j.State)
			fmt.Printf("  Max Retries: %d\n", j.MaxRetries)

			return nil
		},
	}

	return cmd
}