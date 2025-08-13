package worker

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"

	"bloco-eth/pkg/errors"
	"bloco-eth/pkg/wallet"
)

// Pool is a minimal implementation that mimics the working monolithic version
type Pool struct {
	threadCount     int
	mu              sync.RWMutex
	isRunning       bool
	logger          *wallet.WalletLogger
	statsCollector  *StatsCollector
	statsChan       chan WorkerStats
	statsCtx        context.Context
	statsCancel     context.CancelFunc
}

// NewPool creates a new worker pool
func NewPool(threadCount int) *Pool {
	// Validate threadCount
	if threadCount < 0 {
		threadCount = 1 // Default to 1 thread for negative values
	}
	if threadCount == 0 {
		threadCount = 1 // Default to 1 thread for zero
	}
	
	logger, err := wallet.NewWalletLogger()
	if err != nil {
		// Log error but continue without file logging
		fmt.Printf("Warning: Failed to create wallet logger: %v\n", err)
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
			fmt.Printf("Warning: Failed to close wallet logger: %v\n", err)
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
				
				// Generate private key
				privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
				if err != nil {
					continue
				}

				// Get public key and address
				publicKey := privateKey.Public()
				publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
				if !ok {
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
		// Log the wallet if logger is available
		if p.logger != nil {
			if err := p.logger.LogWallet(result); err != nil {
				fmt.Printf("Warning: Failed to log wallet: %v\n", err)
			}
		}
		return result, nil
	case err := <-errorCh:
		return nil, err
	case <-ctx.Done():
		return nil, errors.NewCancellationError("generate_wallet", "generation cancelled")
	}
}

// isValidBlocoAddress checks if an address matches the criteria
func isValidBlocoAddress(address, prefix, suffix string, isChecksum bool) bool {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	
	// If checksum validation is required, use EIP-55 check
	if isChecksum && (prefix != "" || suffix != "") {
		return isEIP55Checksum(address, prefix, suffix)
	}
	
	// For non-checksum validation, use case-insensitive comparison
	// Remove "0x" prefix
	addrWithoutPrefix := strings.ToLower(address[2:])
	lowerPrefix := strings.ToLower(prefix)
	lowerSuffix := strings.ToLower(suffix)
	
	// Check prefix
	if prefix != "" && !strings.HasPrefix(addrWithoutPrefix, lowerPrefix) {
		return false
	}
	
	// Check suffix
	if suffix != "" && !strings.HasSuffix(addrWithoutPrefix, lowerSuffix) {
		return false
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
	
	// Check if the pattern matches the checksum requirements
	if prefix != "" {
		prefixPart := checksumAddr[2:2+len(prefix)]
		if !strings.EqualFold(prefixPart, prefix) {
			return false
		}
		// Check if the actual case matches the desired pattern
		if prefixPart != prefix {
			return false
		}
	}
	
	if suffix != "" {
		suffixStart := len(checksumAddr) - len(suffix)
		if suffixStart < 2 {
			return false
		}
		suffixPart := checksumAddr[suffixStart:]
		if !strings.EqualFold(suffixPart, suffix) {
			return false
		}
		// Check if the actual case matches the desired pattern
		if suffixPart != suffix {
			return false
		}
	}
	
	return true
}