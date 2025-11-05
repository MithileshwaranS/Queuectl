package cli

import (
	"fmt"
	"strconv"

	"github.com/MithileshwaranS/queuectl/internal/config"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View or modify queuectl configuration settings.`,
	}

	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configListCmd())

	return cmd
}

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Long: `Get the value of a specific configuration key.

Available keys:
  - max-retries: Maximum number of retry attempts
  - backoff-base: Base for exponential backoff calculation
  - db-path: Path to the SQLite database
  - worker-count: Default number of workers`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			cfg := getConfig()

			var value interface{}
			switch key {
			case "max-retries":
				value = cfg.MaxRetries
			case "backoff-base":
				value = cfg.BackoffBase
			case "db-path":
				value = cfg.DBPath
			case "worker-count":
				value = cfg.WorkerCount
			default:
				return fmt.Errorf("unknown config key: %s", key)
			}

			fmt.Printf("%s = %v\n", key, value)
			return nil
		},
	}
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long: `Set a configuration value and persist it to disk.

Available keys:
  - max-retries: Maximum number of retry attempts (integer)
  - backoff-base: Base for exponential backoff calculation (float)
  - db-path: Path to the SQLite database (string)
  - worker-count: Default number of workers (integer)

Examples:
  queuectl config set max-retries 5
  queuectl config set backoff-base 2.5
  queuectl config set worker-count 3`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			valueStr := args[1]

			var value interface{}
			var err error

			switch key {
			case "max-retries":
				value, err = strconv.Atoi(valueStr)
				if err != nil {
					return fmt.Errorf("max-retries must be an integer")
				}
			case "backoff-base":
				value, err = strconv.ParseFloat(valueStr, 64)
				if err != nil {
					return fmt.Errorf("backoff-base must be a number")
				}
			case "db-path":
				value = valueStr
			case "worker-count":
				value, err = strconv.Atoi(valueStr)
				if err != nil {
					return fmt.Errorf("worker-count must be an integer")
				}
			default:
				return fmt.Errorf("unknown config key: %s", key)
			}

			if err := config.Set(key, value); err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}

			fmt.Printf("✓ Configuration updated: %s = %v\n", key, value)
			fmt.Printf("Config saved to: %s\n", config.GetConfigPath())

			return nil
		},
	}
}

func configListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Long:  `Display all current configuration values.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig()

			fmt.Println("=== Configuration ===")
			fmt.Println()
			fmt.Printf("max-retries   = %d\n", cfg.MaxRetries)
			fmt.Printf("backoff-base  = %.1f\n", cfg.BackoffBase)
			fmt.Printf("db-path       = %s\n", cfg.DBPath)
			fmt.Printf("worker-count  = %d\n", cfg.WorkerCount)
			fmt.Println()
			fmt.Printf("Config file: %s\n", config.GetConfigPath())

			return nil
		},
	}
}