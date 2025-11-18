package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"bloco-eth/internal/config"
	internalCrypto "bloco-eth/internal/crypto"
	"bloco-eth/pkg/errors"
	"bloco-eth/pkg/logging"
	"bloco-eth/pkg/wallet"
)

// Pool is a minimal implementation that mimics the working monolithic version
type Pool struct {
	threadCount    int
	adapter        internalCrypto.ChainAdapter
	mu             sync.RWMutex
	isRunning      bool
	logger         logging.SecureLogger
	statsCollector *StatsCollector
	statsChan      chan WorkerStats
	statsCtx       context.Context
	statsCancel    context.CancelFunc
}

// NewPool creates a new worker pool
func NewPool(threadCount int) *Pool {
	return NewPoolWithConfig(threadCount, nil)
}

// NewPoolWithConfig creates a new worker pool with configuration
func NewPoolWithConfig(threadCount int, cfg *config.Config) *Pool {
	// Validate threadCount
	if threadCount < 0 {
		threadCount = 1 // Default to 1 thread for negative values
	}
	if threadCount == 0 {
		threadCount = 1 // Default to 1 thread for zero
	}

	// Use default config if none provided
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Get the appropriate adapter for the configured chain
	adapter, err := internalCrypto.GetAdapter(cfg.Chain)
	if err != nil {
		// Fall back to Ethereum adapter if chain is not supported
		fmt.Printf("Warning: %v, falling back to Ethereum\n", err)
		adapter = internalCrypto.NewEthereumAdapter()
	}

	// Create SecureLogger from configuration
	logConfig, err := createLogConfigFromAppConfig(cfg)
	if err != nil {
		// Log error but continue with disabled logging
		fmt.Printf("Warning: Failed to create log configuration: %v\n", err)
		// Create a disabled logger as fallback
		logConfig = &logging.LogConfig{Enabled: false}
	}

	logger, err := logging.NewSecureLogger(logConfig)
	if err != nil {
		// Log error but continue with disabled logging
		fmt.Printf("Warning: Failed to create secure logger: %v\n", err)
		// Create a disabled logger as fallback
		logger, _ = logging.NewSecureLogger(&logging.LogConfig{Enabled: false})
	}

	statsCollector := NewStatsCollector()
	statsChan := make(chan WorkerStats, threadCount*2) // Buffered channel

	// Create context for stats collection
	statsCtx, statsCancel := context.WithCancel(context.Background())

	return &Pool{
		threadCount:    threadCount,
		adapter:        adapter,
		isRunning:      false,
		logger:         logger,
		statsCollector: statsCollector,
		statsChan:      statsChan,
		statsCtx:       statsCtx,
		statsCancel:    statsCancel,
	}
}

// Start starts the worker pool
func (p *Pool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isRunning = true

	// Start stats collection
	if p.statsCollector != nil && p.statsChan != nil {
		p.statsCollector.Start(p.statsChan, p.statsCtx)
	}

	return nil
}

// Shutdown shuts down the worker pool
func (p *Pool) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isRunning = false

	// Cancel stats context
	if p.statsCancel != nil {
		p.statsCancel()
	}

	// Close the logger if it exists
	if p.logger != nil {
		if err := p.logger.Close(); err != nil {
			fmt.Printf("Warning: Failed to close secure logger: %v\n", err)
		}
	}

	return nil
}

// GetStatsCollector returns the stats collector
func (p *Pool) GetStatsCollector() *StatsCollector {
	return p.statsCollector
}

// GenerateWalletWithContext generates a wallet using the worker pool
func (p *Pool) GenerateWalletWithContext(ctx context.Context, criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	// Validate pattern using the adapter
	if err := p.adapter.ValidatePattern(criteria.Prefix, criteria.Suffix); err != nil {
		return nil, fmt.Errorf("pattern validation failed: %w", err)
	}

	// Log operation start
	if p.logger != nil {
		params := map[string]interface{}{
			"prefix":       criteria.Prefix,
			"suffix":       criteria.Suffix,
			"checksum":     criteria.IsChecksum,
			"threads":      p.threadCount,
			"use_mnemonic": criteria.UseMnemonic,
			"chain":        p.adapter.ChainName(),
		}
		if err := p.logger.LogOperationStart("wallet_generation", params); err != nil {
			fmt.Printf("Warning: Failed to log operation start: %v\n", err)
		}
	}

	resultCh := make(chan *wallet.GenerationResult, 1)
	errorCh := make(chan error, 1)

	// Start workers similar to monolithic version
	var wg sync.WaitGroup
	for i := 0; i < p.threadCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Worker loop
			attempts := int64(0)
			startTime := time.Now()
			lastStatsUpdate := startTime

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				attempts++

				// Send stats update every 100ms or 1000 attempts
				now := time.Now()
				if now.Sub(lastStatsUpdate) >= 100*time.Millisecond || attempts%1000 == 0 {
					elapsed := now.Sub(startTime)
					var speed float64
					if elapsed.Seconds() > 0 {
						speed = float64(attempts) / elapsed.Seconds()
					}

					// Send stats to collector
					select {
					case p.statsChan <- WorkerStats{
						WorkerID:   workerID,
						Attempts:   attempts,
						Speed:      speed,
						LastUpdate: now,
						IsHealthy:  true,
						ErrorCount: 0,
					}:
					default:
						// Non-blocking send
					}
					lastStatsUpdate = now
				}

				// Generate key material using the adapter
				km, err := p.adapter.GenerateKeyMaterial()
				if err != nil {
					// Log crypto error (but don't stop the worker)
					if p.logger != nil {
						context := map[string]interface{}{
							"worker_id": workerID,
							"attempts":  attempts,
							"chain":     p.adapter.ChainName(),
						}
						if logErr := p.logger.LogError("crypto_key_generation", err, context); logErr != nil {
							_ = logErr // Make the empty branch explicit
						}
					}
					continue
				}

				// Format address using the adapter
				address, err := p.adapter.FormatAddress(km)
				if err != nil {
					if p.logger != nil {
						context := map[string]interface{}{
							"worker_id": workerID,
							"attempts":  attempts,
							"chain":     p.adapter.ChainName(),
						}
						if logErr := p.logger.LogError("address_formatting", err, context); logErr != nil {
							_ = logErr
						}
					}
					continue
				}

				// Check if address matches criteria using the adapter
				if p.adapter.MatchesPattern(address, criteria) {
					// Found a match!
					privateKeyHex, err := p.adapter.FormatPrivateKey(km)
					if err != nil {
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id": workerID,
								"attempts":  attempts,
							}
							if logErr := p.logger.LogError("private_key_formatting", err, context); logErr != nil {
								_ = logErr
							}
						}
						continue
					}

					publicKeyHex, err := p.adapter.FormatPublicKey(km)
					if err != nil {
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id": workerID,
								"attempts":  attempts,
							}
							if logErr := p.logger.LogError("public_key_formatting", err, context); logErr != nil {
								_ = logErr
							}
						}
						continue
					}

					result := &wallet.GenerationResult{
						Wallet: &wallet.Wallet{
							Address:    address,
							PublicKey:  publicKeyHex,
							PrivateKey: privateKeyHex,
							Mnemonic:   "", // TODO: Add mnemonic support to adapters
							Chain:      p.adapter.ChainName(),
							Encoding:   p.adapter.AddressEncoding(),
							CreatedAt:  time.Now(),
						},
						Attempts: attempts,
						Duration: time.Since(startTime),
						WorkerID: workerID,
					}

					select {
					case resultCh <- result:
					case <-ctx.Done():
					}
					return
				}
			}
		}(i)
	}

	// Wait for result or cancellation
	select {
	case result := <-resultCh:
		// Log the wallet generation and operation completion
		if p.logger != nil {
			// Log the specific wallet generated
			if err := p.logger.LogWalletGenerated(
				result.Wallet.Address,
				int(result.Attempts),
				result.Duration,
				result.WorkerID,
			); err != nil {
				fmt.Printf("Warning: Failed to log wallet: %v\n", err)
			}

			// Log operation completion
			stats := logging.OperationStats{
				Duration:     result.Duration,
				Success:      true,
				ItemsCount:   1,
				ErrorCount:   0,
				ThroughputPS: 1.0 / result.Duration.Seconds(),
			}
			if err := p.logger.LogOperationComplete("wallet_generation", stats); err != nil {
				fmt.Printf("Warning: Failed to log operation completion: %v\n", err)
			}
		}
		return result, nil
	case err := <-errorCh:
		// Log the error using secure logging
		if p.logger != nil {
			context := map[string]interface{}{
				"threads": p.threadCount,
			}
			if logErr := p.logger.LogError("wallet_generation", err, context); logErr != nil {
				fmt.Printf("Warning: Failed to log error: %v\n", logErr)
			}
		}
		return nil, err
	case <-ctx.Done():
		cancellationErr := errors.NewCancellationError("generate_wallet", "generation cancelled")
		// Log the cancellation as an error
		if p.logger != nil {
			context := map[string]interface{}{
				"threads": p.threadCount,
				"reason":  "context_cancelled",
			}
			if logErr := p.logger.LogError("wallet_generation", cancellationErr, context); logErr != nil {
				fmt.Printf("Warning: Failed to log cancellation: %v\n", logErr)
			}
		}
		return nil, cancellationErr
	}
}


// createLogConfigFromAppConfig converts internal config to logging package config
func createLogConfigFromAppConfig(cfg *config.Config) (*logging.LogConfig, error) {
	if !cfg.Logging.Enabled {
		// Return a valid disabled config with default values to pass validation
		return &logging.LogConfig{
			Enabled:     false,
			Level:       logging.INFO,
			Format:      logging.TEXT,
			OutputFile:  "",
			MaxFileSize: 10 * 1024 * 1024, // 10MB default
			MaxFiles:    5,
			BufferSize:  1000,
		}, nil
	}

	// Parse log level
	level, err := logging.ParseLogLevel(cfg.Logging.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", cfg.Logging.Level, err)
	}

	// Parse log format
	var format logging.LogFormat
	switch strings.ToLower(cfg.Logging.Format) {
	case "json":
		format = logging.JSON
	case "structured":
		format = logging.STRUCTURED
	case "text":
		format = logging.TEXT
	default:
		return nil, fmt.Errorf("invalid log format %q, must be one of: text, json, structured", cfg.Logging.Format)
	}

	return &logging.LogConfig{
		Enabled:     cfg.Logging.Enabled,
		Level:       level,
		Format:      format,
		OutputFile:  cfg.Logging.OutputFile,
		MaxFileSize: cfg.Logging.MaxFileSize,
		MaxFiles:    cfg.Logging.MaxFiles,
		BufferSize:  cfg.Logging.BufferSize,
	}, nil
}
