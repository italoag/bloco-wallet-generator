package main

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// NewWorker creates a new worker with the given ID and channels
func NewWorker(id int, workChan chan WorkItem, resultChan chan WorkResult, statsChan chan WorkerStats, shutdownChan chan struct{}, wg *sync.WaitGroup) *Worker {
	return &Worker{
		id:           id,
		workChan:     workChan,
		resultChan:   resultChan,
		statsChan:    statsChan,
		shutdownChan: shutdownChan,
		wg:           wg,
		localStats: WorkerStats{
			WorkerID:   id,
			Attempts:   0,
			Speed:      0,
			LastUpdate: time.Now(),
		},
	}
}

// Start begins the worker's processing loop
func (w *Worker) Start() {
	// Create worker-local object pools for better performance
	cryptoPool := NewCryptoPool()
	hasherPool := NewHasherPool()
	bufferPool := NewBufferPool()

	// Track statistics
	startTime := time.Now()
	lastStatsUpdate := startTime
	var attempts int64 = 0

	// Process work items until shutdown
	go func() {
		// This will be called when the goroutine exits
		// It signals to the worker pool that this worker has finished
		defer func() {
			// Signal that this worker has finished
			if w.wg != nil {
				w.wg.Done()
			}
		}()

		for {
			select {
			case workItem, ok := <-w.workChan:
				if !ok {
					// Work channel closed, exit
					return
				}

				// Process the work item
				result := w.processWorkItem(workItem, cryptoPool, hasherPool, bufferPool)

				// Update local statistics
				attempts += result.Attempts
				now := time.Now()
				elapsed := now.Sub(startTime)

				// Update speed calculation every 100ms for more frequent updates
				if now.Sub(lastStatsUpdate) >= 100*time.Millisecond {
					w.localStats.Attempts = attempts
					w.localStats.Speed = float64(attempts) / elapsed.Seconds()
					w.localStats.LastUpdate = now

					// Send stats update (non-blocking)
					select {
					case w.statsChan <- w.localStats:
						// Stats sent successfully
					default:
						// Channel full, skip this update
					}

					lastStatsUpdate = now
				}

				// Send result (always try to send, even if not found for stats purposes)
				if result.Found {
					// This is a critical result, so we'll block until we can send it
					select {
					case w.resultChan <- result:
						// Result sent successfully
					case <-w.shutdownChan:
						// Shutdown signal received, exit
						return
					}
				} else {
					// For non-matching results, use non-blocking send
					select {
					case w.resultChan <- result:
						// Result sent successfully
					default:
						// Channel is full, skip this update
					}
				}

			case <-w.shutdownChan:
				// Final stats update before shutdown
				w.localStats.Attempts = attempts
				elapsed := time.Since(startTime)
				if elapsed.Seconds() > 0 {
					w.localStats.Speed = float64(attempts) / elapsed.Seconds()
				}
				w.localStats.LastUpdate = time.Now()

				// Try to send final stats (non-blocking)
				select {
				case w.statsChan <- w.localStats:
					// Stats sent successfully
				default:
					// Channel full, skip this update
				}

				return
			}
		}
	}()
}

// processWorkItem processes a single work item using optimized cryptographic operations
func (w *Worker) processWorkItem(item WorkItem, cryptoPool *CryptoPool, hasherPool *HasherPool, bufferPool *BufferPool) WorkResult {
	pre := item.Prefix
	suf := item.Suffix
	batchSize := item.BatchSize

	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	// For non-checksum validation, convert to lowercase
	if !item.IsChecksum {
		pre = strings.ToLower(pre)
		suf = strings.ToLower(suf)
	}

	var attempts int64 = 0

	// Process a batch of wallet generation attempts
	for i := 0; i < batchSize; i++ {
		// Get private key buffer from pool
		privateKey := cryptoPool.GetPrivateKeyBuffer()

		// Generate random private key
		_, err := rand.Read(privateKey)
		if err != nil {
			cryptoPool.PutPrivateKeyBuffer(privateKey)
			return WorkResult{
				Attempts: attempts,
				WorkerID: w.id,
				Found:    false,
				Error:    err,
			}
		}

		attempts++

		// Optimized address generation using worker-local pools
		address := w.privateToAddressOptimized(privateKey, cryptoPool, hasherPool, bufferPool)

		// Check if address matches the pattern
		if w.isValidBlocoAddressOptimized(address, pre, suf, item.IsChecksum, hasherPool, bufferPool) {
			// Found a match, create wallet
			wallet := &Wallet{
				Address: address,
				PrivKey: hex.EncodeToString(privateKey),
			}

			// Clear sensitive data before returning
			cryptoPool.PutPrivateKeyBuffer(privateKey)

			return WorkResult{
				Wallet:   wallet,
				Attempts: attempts,
				WorkerID: w.id,
				Found:    true,
			}
		}

		// Return private key buffer to pool
		cryptoPool.PutPrivateKeyBuffer(privateKey)
	}

	// No match found in this batch
	return WorkResult{
		Attempts: attempts,
		WorkerID: w.id,
		Found:    false,
	}
}

// privateToAddressOptimized is an optimized version of privateToAddress using worker-local pools
func (w *Worker) privateToAddressOptimized(privateKey []byte, cryptoPool *CryptoPool, hasherPool *HasherPool, bufferPool *BufferPool) string {
	// Get objects from pools
	privateKeyInt := cryptoPool.GetBigInt()
	privateKeyECDSA := cryptoPool.GetECDSAKey()
	hasher := hasherPool.GetKeccak()
	publicKeyBytes := cryptoPool.GetPublicKeyBuffer()
	hexBuffer := bufferPool.GetHexBuffer()

	defer func() {
		// Return objects to pools
		cryptoPool.PutBigInt(privateKeyInt)
		cryptoPool.PutECDSAKey(privateKeyECDSA)
		hasherPool.PutKeccak(hasher)
		cryptoPool.PutPublicKeyBuffer(publicKeyBytes)
		bufferPool.PutHexBuffer(hexBuffer)
	}()

	// Convert private key bytes to ECDSA private key
	privateKeyInt.SetBytes(privateKey)
	privateKeyECDSA.D = privateKeyInt

	// Calculate public key coordinates
	// Using crypto.S256().ScalarBaseMult directly is deprecated, but we're using it for compatibility
	// In a future update, this should be replaced with crypto/ecdh package
	privateKeyECDSA.PublicKey.X, privateKeyECDSA.PublicKey.Y = crypto.S256().ScalarBaseMult(privateKey)

	// Get uncompressed public key bytes (without 0x04 prefix)
	// This is a simplified version - in a real implementation we'd use crypto.FromECDSAPub
	// and extract the bytes after the prefix
	publicKeyBytes = publicKeyBytes[:0] // Reset but keep capacity

	// Append X and Y coordinates
	publicKeyBytes = append(publicKeyBytes, privateKeyECDSA.PublicKey.X.Bytes()...)
	publicKeyBytes = append(publicKeyBytes, privateKeyECDSA.PublicKey.Y.Bytes()...)

	// Calculate Keccak256 hash using pooled hasher
	hasher.Reset()
	hasher.Write(publicKeyBytes)
	hash := hasher.Sum(nil)

	// Take the last 20 bytes as the address
	address := hash[len(hash)-20:]

	// Convert to hex string
	hex.Encode(hexBuffer, address)
	return string(hexBuffer[:40])
}

// isValidBlocoAddressOptimized is an optimized version of isValidBlocoAddress
func (w *Worker) isValidBlocoAddressOptimized(address, prefix, suffix string, isChecksum bool, hasherPool *HasherPool, bufferPool *BufferPool) bool {
	// Protect against index out of range
	if len(prefix) > len(address) {
		return false
	}
	if len(suffix) > len(address) {
		return false
	}

	// Extract prefix and suffix from address
	addressPrefix := address[:len(prefix)]
	addressSuffix := ""
	if len(suffix) > 0 {
		addressSuffix = address[len(address)-len(suffix):]
	}

	if !isChecksum {
		// For non-checksum validation, just compare case-insensitively
		prefixMatch := len(prefix) == 0 || strings.EqualFold(prefix, addressPrefix)
		suffixMatch := len(suffix) == 0 || strings.EqualFold(suffix, addressSuffix)
		return prefixMatch && suffixMatch
	}

	// For checksum validation, first check if lowercase versions match
	prefixMatch := len(prefix) == 0 || strings.EqualFold(prefix, addressPrefix)
	suffixMatch := len(suffix) == 0 || strings.EqualFold(suffix, addressSuffix)

	if !prefixMatch || !suffixMatch {
		return false
	}

	// Perform checksum validation
	return w.isValidChecksumOptimized(address, prefix, suffix, hasherPool, bufferPool)
}

// isValidChecksumOptimized is an optimized version of isValidChecksum
func (w *Worker) isValidChecksumOptimized(address, prefix, suffix string, hasherPool *HasherPool, bufferPool *BufferPool) bool {
	// Get hasher and string builder from pools
	hasher := hasherPool.GetKeccak()
	sb := bufferPool.GetStringBuilder()

	defer func() {
		hasherPool.PutKeccak(hasher)
		bufferPool.PutStringBuilder(sb)
	}()

	// Calculate Keccak256 hash of the address
	hasher.Write([]byte(address))
	hashBytes := hasher.Sum(nil)

	// Convert hash to hex string using string builder for efficiency
	for _, b := range hashBytes {
		sb.WriteByte("0123456789abcdef"[b>>4])
		sb.WriteByte("0123456789abcdef"[b&0x0f])
	}
	hash := sb.String()

	// Check prefix checksum
	for i := 0; i < len(prefix); i++ {
		hashChar, _ := strconv.ParseInt(string(hash[i]), 16, 64)
		expectedChar := string(address[i])
		if hashChar >= 8 {
			expectedChar = strings.ToUpper(expectedChar)
		}
		if string(prefix[i]) != expectedChar {
			return false
		}
	}

	// Check suffix checksum
	for i := 0; i < len(suffix); i++ {
		j := i + 40 - len(suffix)
		hashChar, _ := strconv.ParseInt(string(hash[j]), 16, 64)
		expectedChar := string(address[j])
		if hashChar >= 8 {
			expectedChar = strings.ToUpper(expectedChar)
		}
		if string(suffix[i]) != expectedChar {
			return false
		}
	}

	return true
}
