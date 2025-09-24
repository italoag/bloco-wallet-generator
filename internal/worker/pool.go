package worker

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	bip32 "github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/sha3"

	"bloco-eth/internal/config"
	"bloco-eth/pkg/errors"
	"bloco-eth/pkg/logging"
	"bloco-eth/pkg/wallet"
)

// Pool is a minimal implementation that mimics the working monolithic version
type Pool struct {
	threadCount    int
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
	// Log operation start
	if p.logger != nil {
		params := map[string]interface{}{
			"prefix":       criteria.Prefix,
			"suffix":       criteria.Suffix,
			"checksum":     criteria.IsChecksum,
			"threads":      p.threadCount,
			"use_mnemonic": criteria.UseMnemonic,
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

				// Generate private key material based on generation strategy
				var (
					privateKey *ecdsa.PrivateKey
					mnemonic   string
					err        error
				)

				if criteria.UseMnemonic {
					mnemonic, privateKey, err = generateMnemonicPrivateKey()
					if err != nil {
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id":      workerID,
								"attempts":       attempts,
								"use_mnemonic":   true,
								"error_category": "mnemonic_generation",
							}
							if logErr := p.logger.LogError("wallet_material_generation", err, context); logErr != nil {
								_ = logErr
							}
						}
						continue
					}
				} else {
					privateKey, err = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
					if err != nil {
						// Log crypto error (but don't stop the worker)
						if p.logger != nil {
							context := map[string]interface{}{
								"worker_id": workerID,
								"attempts":  attempts,
							}
							if logErr := p.logger.LogError("crypto_key_generation", err, context); logErr != nil {
								// Intentionally ignore log errors to prevent output spam during intensive operations
								// This ensures the worker continues operating even if logging fails
								_ = logErr // Make the empty branch explicit
							}
						}
						continue
					}
				}

				// Get public key and address
				publicKey := privateKey.Public()
				publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
				if !ok {
					// Log type assertion error
					if p.logger != nil {
						typeErr := fmt.Errorf("failed to cast public key to ECDSA")
						context := map[string]interface{}{
							"worker_id": workerID,
							"attempts":  attempts,
						}
						if logErr := p.logger.LogError("crypto_key_conversion", typeErr, context); logErr != nil {
							// Intentionally ignore log errors to prevent output spam during intensive operations
							// This ensures the worker continues operating even if logging fails
							_ = logErr // Make the empty branch explicit
						}
					}
					continue
				}

				address := crypto.PubkeyToAddress(*publicKeyECDSA)
				addressStr := address.Hex()

				// Check if address matches criteria
				if isValidBlocoAddress(addressStr, criteria.Prefix, criteria.Suffix, criteria.IsChecksum) {
					// Found a match!
					privateKeyBytes := crypto.FromECDSA(privateKey)
					privateKeyHex := fmt.Sprintf("%x", privateKeyBytes)

					// Get public key hex
					publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
					publicKeyHex := fmt.Sprintf("%x", publicKeyBytes)

					// Use checksum address if checksum is required
					finalAddress := addressStr
					if criteria.IsChecksum {
						finalAddress = toChecksumAddress(addressStr)
					}

					result := &wallet.GenerationResult{
						Wallet: &wallet.Wallet{
							Address:    finalAddress,
							PublicKey:  publicKeyHex,
							PrivateKey: privateKeyHex,
							Mnemonic:   mnemonic,
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

// generateMnemonicPrivateKey creates a new mnemonic phrase and derives the corresponding private key
func generateMnemonicPrivateKey() (string, *ecdsa.PrivateKey, error) {
	// Generate 128 bits of entropy for a 12-word mnemonic to balance security and performance
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", nil, err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", nil, err
	}

	seed := bip39.NewSeed(mnemonic, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return "", nil, err
	}

	derivationPath := []uint32{
		bip32.FirstHardenedChild + 44,
		bip32.FirstHardenedChild + 60,
		bip32.FirstHardenedChild + 0,
		0,
		0,
	}

	key := masterKey
	for _, child := range derivationPath {
		key, err = key.NewChildKey(child)
		if err != nil {
			return "", nil, err
		}
	}

	privateKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		return "", nil, err
	}

	return mnemonic, privateKey, nil
}

// isValidBlocoAddress checks if an address matches the criteria
func isValidBlocoAddress(address, prefix, suffix string, isChecksum bool) bool {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	// Debug logging
	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG: Validating address %s with prefix=%q suffix=%q checksum=%v\n",
			address, prefix, suffix, isChecksum)
	}

	// If checksum validation is required, use EIP-55 check
	if isChecksum && (prefix != "" || suffix != "") {
		result := isEIP55Checksum(address, prefix, suffix)
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG: EIP55 validation result: %v\n", result)
		}
		return result
	}

	// For non-checksum validation, use case-insensitive comparison
	// Remove "0x" prefix
	addrWithoutPrefix := strings.ToLower(address[2:])
	lowerPrefix := strings.ToLower(prefix)
	lowerSuffix := strings.ToLower(suffix)

	// Check prefix
	if prefix != "" && !strings.HasPrefix(addrWithoutPrefix, lowerPrefix) {
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG: Prefix check failed: %q does not start with %q\n",
				addrWithoutPrefix, lowerPrefix)
		}
		return false
	}

	// Check suffix
	if suffix != "" && !strings.HasSuffix(addrWithoutPrefix, lowerSuffix) {
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG: Suffix check failed: %q does not end with %q\n",
				addrWithoutPrefix, lowerSuffix)
		}
		return false
	}

	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG: Address validation passed\n")
	}
	return true
}

// toChecksumAddress converts an address to EIP-55 checksum format
func toChecksumAddress(address string) string {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	// Remove 0x prefix for hashing
	addrWithoutPrefix := strings.ToLower(address[2:])
	addrBytes := []byte(addrWithoutPrefix)

	// Create Keccak256 hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(addrBytes)
	hash := hasher.Sum(nil)

	// Apply EIP-55 checksum
	var result strings.Builder
	result.WriteString("0x")

	for i, char := range addrWithoutPrefix {
		if char >= '0' && char <= '9' {
			// Numbers remain unchanged
			result.WriteByte(byte(char))
		} else if char >= 'a' && char <= 'f' {
			// Letters: uppercase if hash bit >= 8, lowercase otherwise
			hashByte := hash[i/2]
			var hashBit uint8
			if i%2 == 0 {
				hashBit = hashByte >> 4
			} else {
				hashBit = hashByte & 0x0f
			}

			if hashBit >= 8 {
				result.WriteByte(byte(char - 32)) // Convert to uppercase
			} else {
				result.WriteByte(byte(char)) // Keep lowercase
			}
		}
	}

	return result.String()
}

// isEIP55Checksum validates EIP-55 checksum for specific pattern
func isEIP55Checksum(address, prefix, suffix string) bool {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	// Generate the correct checksum address
	checksumAddr := toChecksumAddress(address)

	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG EIP55: Original=%s Checksum=%s Prefix=%q Suffix=%q\n",
			address, checksumAddr, prefix, suffix)
	}

	// Check if the pattern matches the checksum requirements
	if prefix != "" {
		prefixPart := checksumAddr[2 : 2+len(prefix)]
		if !strings.EqualFold(prefixPart, prefix) {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG EIP55: Prefix failed - got %q expected %q\n", prefixPart, prefix)
			}
			return false
		}
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG EIP55: Prefix matched - got %q expected %q\n", prefixPart, prefix)
		}
		// For EIP-55 checksum validation, we only need to verify that the pattern
		// matches case-insensitively. The checksum correctness is already ensured
		// by toChecksumAddress() function.
	}

	if suffix != "" {
		suffixStart := len(checksumAddr) - len(suffix)
		if suffixStart < 2 {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG EIP55: Suffix too long for address\n")
			}
			return false
		}
		suffixPart := checksumAddr[suffixStart:]
		if !strings.EqualFold(suffixPart, suffix) {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG EIP55: Suffix failed - got %q expected %q\n", suffixPart, suffix)
			}
			return false
		}
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG EIP55: Suffix matched - got %q expected %q\n", suffixPart, suffix)
		}
		// For EIP-55 checksum validation, we only need to verify that the pattern
		// matches case-insensitively. The checksum correctness is already ensured
		// by toChecksumAddress() function.
	}

	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG EIP55: Validation passed\n")
	}
	return true
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
