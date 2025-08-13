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

// SimplePool is a minimal implementation that mimics the working monolithic version
type SimplePool struct {
	threadCount int
	mu          sync.RWMutex
	isRunning   bool
}

// NewSimplePool creates a new simple worker pool
func NewSimplePool(threadCount int) *SimplePool {
	return &SimplePool{
		threadCount: threadCount,
		isRunning:   false,
	}
}

// Start starts the worker pool
func (sp *SimplePool) Start() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.isRunning = true
	return nil
}

// Shutdown shuts down the worker pool
func (sp *SimplePool) Shutdown() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.isRunning = false
	return nil
}

// GetStatsCollector returns a dummy stats collector
func (sp *SimplePool) GetStatsCollector() *StatsCollector {
	return NewStatsCollector()
}

// GenerateWalletWithContext generates a wallet using the simple approach
func (sp *SimplePool) GenerateWalletWithContext(ctx context.Context, criteria wallet.GenerationCriteria) (*wallet.GenerationResult, error) {
	resultCh := make(chan *wallet.GenerationResult, 1)
	errorCh := make(chan error, 1)

	// Start workers similar to monolithic version
	var wg sync.WaitGroup
	for i := 0; i < sp.threadCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			// Worker loop
			attempts := int64(0)
			startTime := time.Now()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				attempts++
				
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
				addressStr := strings.ToLower(address.Hex())

				// Check if address matches criteria
				if isValidBlocoAddress(addressStr, criteria.Prefix, criteria.Suffix, criteria.IsChecksum) {
					// Found a match!
					privateKeyBytes := crypto.FromECDSA(privateKey)
					privateKeyHex := fmt.Sprintf("%x", privateKeyBytes)
					
					result := &wallet.GenerationResult{
						Wallet: &wallet.Wallet{
							Address:    address.Hex(),
							PrivateKey: privateKeyHex,
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
		return result, nil
	case err := <-errorCh:
		return nil, err
	case <-ctx.Done():
		return nil, errors.NewCancellationError("generate_wallet", "generation cancelled")
	}
}

// isValidBlocoAddress checks if an address matches the criteria (simplified version)
func isValidBlocoAddress(address, prefix, suffix string, isChecksum bool) bool {
	// Remove "0x" prefix
	if strings.HasPrefix(address, "0x") {
		address = address[2:]
	}
	
	// Convert to lowercase for comparison
	lowerAddress := strings.ToLower(address)
	lowerPrefix := strings.ToLower(prefix)
	lowerSuffix := strings.ToLower(suffix)
	
	// Check prefix
	if prefix != "" && !strings.HasPrefix(lowerAddress, lowerPrefix) {
		return false
	}
	
	// Check suffix
	if suffix != "" && !strings.HasSuffix(lowerAddress, lowerSuffix) {
		return false
	}
	
	// If checksum validation is required, implement EIP-55 check
	if isChecksum && (prefix != "" || suffix != "") {
		return isEIP55Checksum("0x" + address, prefix, suffix)
	}
	
	return true
}

// isEIP55Checksum validates EIP-55 checksum for specific pattern
func isEIP55Checksum(address, prefix, suffix string) bool {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	
	// Remove 0x prefix for hashing
	addrBytes := []byte(strings.ToLower(address[2:]))
	
	// Create Keccak256 hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(addrBytes)
	hash := hasher.Sum(nil)
	
	// Check checksum for prefix pattern
	if prefix != "" {
		for i, char := range prefix {
			if i >= len(address)-2 {
				break
			}
			addrChar := address[2+i]
			
			if char >= 'a' && char <= 'f' {
				// Lowercase letter in pattern
				hashByte := hash[i/2]
				var hashBit uint8
				if i%2 == 0 {
					hashBit = hashByte >> 4
				} else {
					hashBit = hashByte & 0x0f
				}
				
				if hashBit >= 8 {
					// Should be uppercase
					if addrChar != byte(char-32) {
						return false
					}
				} else {
					// Should be lowercase
					if addrChar != byte(char) {
						return false
					}
				}
			} else if char >= 'A' && char <= 'F' {
				// Uppercase letter in pattern
				hashByte := hash[i/2]
				var hashBit uint8
				if i%2 == 0 {
					hashBit = hashByte >> 4
				} else {
					hashBit = hashByte & 0x0f
				}
				
				if hashBit >= 8 {
					// Should be uppercase
					if addrChar != byte(char) {
						return false
					}
				} else {
					// Should be lowercase
					if addrChar != byte(char+32) {
						return false
					}
				}
			}
		}
	}
	
	// Similar check for suffix...
	// For now, simplified implementation
	return true
}