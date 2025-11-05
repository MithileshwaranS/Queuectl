package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	MaxRetries  int     `mapstructure:"max_retries"`
	BackoffBase float64 `mapstructure:"backoff_base"`
	DBPath      string  `mapstructure:"db_path"`
	WorkerCount int     `mapstructure:"worker_count"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
)

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:  3,
		BackoffBase: 2.0,
		DBPath:      getDefaultDBPath(),
		WorkerCount: 1,
	}
}

// getDefaultDBPath returns the default database path
func getDefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./queuectl.db"
	}
	configDir := filepath.Join(homeDir, ".queuectl")
	os.MkdirAll(configDir, 0755)
	return filepath.Join(configDir, "queuectl.db")
}

// Load loads configuration from file or creates default
func Load() (*Config, error) {
	var loadErr error
	once.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		
		// Add config paths
		homeDir, _ := os.UserHomeDir()
		viper.AddConfigPath(filepath.Join(homeDir, ".queuectl"))
		viper.AddConfigPath(".")

		// Set defaults
		defaultCfg := DefaultConfig()
		viper.SetDefault("max_retries", defaultCfg.MaxRetries)
		viper.SetDefault("backoff_base", defaultCfg.BackoffBase)
		viper.SetDefault("db_path", defaultCfg.DBPath)
		viper.SetDefault("worker_count", defaultCfg.WorkerCount)

		// Try to read config file
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				loadErr = fmt.Errorf("error reading config file: %w", err)
				return
			}
			// Config file not found, use defaults
		}

		instance = &Config{}
		if err := viper.Unmarshal(instance); err != nil {
			loadErr = fmt.Errorf("error unmarshaling config: %w", err)
			return
		}
	})

	return instance, loadErr
}

// Get returns the singleton config instance
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	
	if instance == nil {
		cfg, _ := Load()
		return cfg
	}
	return instance
}

// Set updates a configuration value
func Set(key string, value interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	viper.Set(key, value)
	
	// Update instance
	if instance == nil {
		instance = &Config{}
	}
	
	switch key {
	case "max_retries", "max-retries":
		if v, ok := value.(int); ok {
			instance.MaxRetries = v
		}
	case "backoff_base", "backoff-base":
		if v, ok := value.(float64); ok {
			instance.BackoffBase = v
		}
	case "db_path", "db-path":
		if v, ok := value.(string); ok {
			instance.DBPath = v
		}
	case "worker_count", "worker-count":
		if v, ok := value.(int); ok {
			instance.WorkerCount = v
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save()
}

// Save persists the current configuration to file
func Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".queuectl")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	return viper.WriteConfigAs(configFile)
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./config.yaml"
	}
	return filepath.Join(homeDir, ".queuectl", "config.yaml")
}