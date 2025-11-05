package cli

import (
	"fmt"

	"github.com/MithileshwaranS/queuectl/internal/config"
	"github.com/MithileshwaranS/queuectl/internal/storage"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	store   storage.Storage
	rootCmd *cobra.Command
)

// Execute runs the CLI
func Execute(c *config.Config) error {
	cfg = c

	// Initialize storage
	var err error
	store, err = storage.NewSQLiteStorage(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := store.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create root command
	rootCmd = &cobra.Command{
		Use:   "queuectl",
		Short: "A CLI-based background job queue system",
		Long: `QueueCTL is a production-grade job queue system that manages 
background jobs with worker processes, retries with exponential backoff,
and a Dead Letter Queue (DLQ) for permanently failed jobs.`,
		Version: "1.0.0",
	}

	// Add all subcommands
	rootCmd.AddCommand(enqueueCmd())
	rootCmd.AddCommand(workerCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(dlqCmd())
	rootCmd.AddCommand(configCmd())

	return rootCmd.Execute()
}

// getStorage returns the storage instance
func getStorage() storage.Storage {
	return store
}

// getConfig returns the config instance
func getConfig() *config.Config {
	return cfg
}