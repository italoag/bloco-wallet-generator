package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Worker WorkerConfig `yaml:"worker"`
	TUI    TUIConfig    `yaml:"tui"`
	Crypto CryptoConfig `yaml:"crypto"`
	CLI    CLIConfig    `yaml:"cli"`
}

// WorkerConfig contains worker-related configuration
type WorkerConfig struct {
	ThreadCount       int           `yaml:"thread_count"`
	MinBatchSize      int           `yaml:"min_batch_size"`
	MaxBatchSize      int           `yaml:"max_batch_size"`
	UpdateInterval    time.Duration `yaml:"update_interval"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
}

// TUIConfig contains TUI-related configuration
type TUIConfig struct {
	Enabled          bool          `yaml:"enabled"`
	RefreshRate      time.Duration `yaml:"refresh_rate"`
	ProgressBarWidth int           `yaml:"progress_bar_width"`
	MaxTableRows     int           `yaml:"max_table_rows"`
	ColorSupport     string        `yaml:"color_support"`   // auto, enabled, disabled
	UnicodeSupport   string        `yaml:"unicode_support"` // auto, enabled, disabled
}

// CryptoConfig contains cryptographic configuration
type CryptoConfig struct {
	PoolSize         int  `yaml:"pool_size"`
	SecureRandom     bool `yaml:"secure_random"`
	OptimizedHashing bool `yaml:"optimized_hashing"`
	MemoryClearing   bool `yaml:"memory_clearing"`
}

// CLIConfig contains CLI-related configuration
type CLIConfig struct {
	ProgressUpdateInterval time.Duration `yaml:"progress_update_interval"`
	VerboseOutput          bool          `yaml:"verbose_output"`
	QuietMode              bool          `yaml:"quiet_mode"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Worker: WorkerConfig{
			ThreadCount:       runtime.NumCPU(),
			MinBatchSize:      100,
			MaxBatchSize:      10000,
			UpdateInterval:    100 * time.Millisecond,
			HealthCheckPeriod: time.Second,
			ShutdownTimeout:   5 * time.Second,
		},
		TUI: TUIConfig{
			Enabled:          true,
			RefreshRate:      500 * time.Millisecond,
			ProgressBarWidth: 40,
			MaxTableRows:     8,
			ColorSupport:     "auto",
			UnicodeSupport:   "auto",
		},
		Crypto: CryptoConfig{
			PoolSize:         runtime.NumCPU() * 2,
			SecureRandom:     true,
			OptimizedHashing: true,
			MemoryClearing:   true,
		},
		CLI: CLIConfig{
			ProgressUpdateInterval: 500 * time.Millisecond,
			VerboseOutput:          false,
			QuietMode:              false,
		},
	}
}

// LoadFromEnvironment loads configuration from environment variables
func (c *Config) LoadFromEnvironment() {
	// Worker configuration
	if threads := os.Getenv("BLOCO_THREADS"); threads != "" {
		if val, err := strconv.Atoi(threads); err == nil && val > 0 {
			c.Worker.ThreadCount = val
		}
	}

	if batchSize := os.Getenv("BLOCO_BATCH_SIZE"); batchSize != "" {
		if val, err := strconv.Atoi(batchSize); err == nil && val > 0 {
			c.Worker.MaxBatchSize = val
		}
	}

	// TUI configuration
	if tuiEnabled := os.Getenv("BLOCO_TUI"); tuiEnabled != "" {
		c.TUI.Enabled = parseBoolEnv(tuiEnabled, c.TUI.Enabled)
	}

	if colorSupport := os.Getenv("BLOCO_COLOR"); colorSupport != "" {
		c.TUI.ColorSupport = colorSupport
	}

	// Check NO_COLOR standard
	if os.Getenv("NO_COLOR") != "" {
		c.TUI.ColorSupport = "disabled"
	}

	// CLI configuration
	if verbose := os.Getenv("BLOCO_VERBOSE"); verbose != "" {
		c.CLI.VerboseOutput = parseBoolEnv(verbose, c.CLI.VerboseOutput)
	}

	if quiet := os.Getenv("BLOCO_QUIET"); quiet != "" {
		c.CLI.QuietMode = parseBoolEnv(quiet, c.CLI.QuietMode)
	}
}

// Validate validates the configuration and returns any errors
func (c *Config) Validate() error {
	// Validate worker configuration
	if c.Worker.ThreadCount <= 0 {
		return fmt.Errorf("worker thread count must be positive, got %d", c.Worker.ThreadCount)
	}

	if c.Worker.ThreadCount > 128 {
		return fmt.Errorf("worker thread count too high (max 128), got %d", c.Worker.ThreadCount)
	}

	if c.Worker.MinBatchSize <= 0 {
		return fmt.Errorf("worker min batch size must be positive, got %d", c.Worker.MinBatchSize)
	}

	if c.Worker.MaxBatchSize < c.Worker.MinBatchSize {
		return fmt.Errorf("worker max batch size (%d) must be >= min batch size (%d)",
			c.Worker.MaxBatchSize, c.Worker.MinBatchSize)
	}

	// Validate TUI configuration
	if c.TUI.ProgressBarWidth <= 0 {
		return fmt.Errorf("TUI progress bar width must be positive, got %d", c.TUI.ProgressBarWidth)
	}

	if c.TUI.MaxTableRows <= 0 {
		return fmt.Errorf("TUI max table rows must be positive, got %d", c.TUI.MaxTableRows)
	}

	// Validate color support setting
	validColorSettings := []string{"auto", "enabled", "disabled"}
	if !contains(validColorSettings, c.TUI.ColorSupport) {
		return fmt.Errorf("invalid color support setting: %s (valid: %v)",
			c.TUI.ColorSupport, validColorSettings)
	}

	// Validate unicode support setting
	validUnicodeSettings := []string{"auto", "enabled", "disabled"}
	if !contains(validUnicodeSettings, c.TUI.UnicodeSupport) {
		return fmt.Errorf("invalid unicode support setting: %s (valid: %v)",
			c.TUI.UnicodeSupport, validUnicodeSettings)
	}

	// Validate crypto configuration
	if c.Crypto.PoolSize <= 0 {
		return fmt.Errorf("crypto pool size must be positive, got %d", c.Crypto.PoolSize)
	}

	// Validate CLI configuration - quiet and verbose are mutually exclusive
	if c.CLI.QuietMode && c.CLI.VerboseOutput {
		return fmt.Errorf("quiet mode and verbose output are mutually exclusive")
	}

	return nil
}

// ApplyOverrides applies command-line overrides to the configuration
func (c *Config) ApplyOverrides(overrides ConfigOverrides) {
	if overrides.ThreadCount != nil {
		c.Worker.ThreadCount = *overrides.ThreadCount
	}

	if overrides.TUIEnabled != nil {
		c.TUI.Enabled = *overrides.TUIEnabled
	}

	if overrides.VerboseOutput != nil {
		c.CLI.VerboseOutput = *overrides.VerboseOutput
	}

	if overrides.QuietMode != nil {
		c.CLI.QuietMode = *overrides.QuietMode
	}
}

// ConfigOverrides represents command-line configuration overrides
type ConfigOverrides struct {
	ThreadCount   *int
	TUIEnabled    *bool
	VerboseOutput *bool
	QuietMode     *bool
}

// parseBoolEnv parses a boolean environment variable with fallback
func parseBoolEnv(value string, fallback bool) bool {
	switch value {
	case "true", "1", "yes", "on", "enabled", "force":
		return true
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		return fallback
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetEffectiveThreadCount returns the effective thread count considering system limits
func (c *Config) GetEffectiveThreadCount() int {
	maxRecommended := runtime.NumCPU() * 2
	if c.Worker.ThreadCount > maxRecommended {
		return maxRecommended
	}
	return c.Worker.ThreadCount
}

// IsTUIEnabled returns whether TUI should be enabled based on configuration and environment
func (c *Config) IsTUIEnabled() bool {
	if !c.TUI.Enabled {
		return false
	}

	// Check if we're in a CI environment
	ciEnvVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "JENKINS_URL",
		"TRAVIS", "CIRCLECI", "APPVEYOR", "GITLAB_CI", "BUILDKITE",
		"DRONE", "GITHUB_ACTIONS", "TF_BUILD", "TEAMCITY_VERSION",
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return false
		}
	}

	return true
}
