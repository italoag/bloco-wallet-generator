package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash"
	"math"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
	
	tea "github.com/charmbracelet/bubbletea"
	"bloco-eth/tui"
)

// Global context for signal handling
var (
	globalCtx    context.Context
	globalCancel context.CancelFunc
)

// Wallet represents an Ethereum wallet with address and private key
type Wallet struct {
	Address string `json:"address"`
	PrivKey string `json:"privKey"`
}

// WalletResult represents the result of wallet generation with attempt count
type WalletResult struct {
	Wallet   *Wallet `json:"wallet,omitempty"`
	Attempts int     `json:"attempts"`
	Error    string  `json:"error,omitempty"`
}

// Statistics holds generation statistics and progress information
type Statistics struct {
	Difficulty      float64
	Probability50   int64
	CurrentAttempts int64
	Speed           float64
	Probability     float64
	EstimatedTime   time.Duration
	StartTime       time.Time
	LastUpdate      time.Time
	Pattern         string
	IsChecksum      bool
}

// BenchmarkResult holds benchmark statistics
type BenchmarkResult struct {
	TotalAttempts         int64
	TotalDuration         time.Duration
	AverageSpeed          float64
	MinSpeed              float64
	MaxSpeed              float64
	SpeedSamples          []float64
	DurationSamples       []time.Duration
	SingleThreadSpeed     float64 // Speed estimate for single-thread execution
	ThreadCount           int     // Number of threads used
	ScalabilityEfficiency float64 // Actual speedup / ideal speedup ratio

	// Enhanced scalability metrics
	ThreadBalanceScore    float64 // How evenly work is distributed (0-1)
	ThreadUtilization     float64 // Average thread utilization (0-1)
	SpeedupVsSingleThread float64 // Actual speedup compared to single-thread
	AmdahlsLawLimit       float64 // Theoretical maximum speedup based on Amdahl's Law
}

// WorkItem represents a work unit for parallel processing
type WorkItem struct {
	Prefix     string
	Suffix     string
	IsChecksum bool
	BatchSize  int
}

// WorkResult represents the result from a worker
type WorkResult struct {
	Wallet   *Wallet
	Attempts int64
	WorkerID int
	Found    bool
	Error    error
}

// WorkerStats holds statistics for individual workers
type WorkerStats struct {
	WorkerID   int
	Attempts   int64
	Speed      float64
	LastUpdate time.Time
}

// Worker represents an individual worker thread
type Worker struct {
	id           int
	workChan     chan WorkItem
	resultChan   chan WorkResult
	statsChan    chan WorkerStats
	shutdownChan chan struct{}
	localStats   WorkerStats
	wg           *sync.WaitGroup // Reference to the worker pool's WaitGroup
}

// WorkerPool manages multiple workers for parallel processing
type WorkerPool struct {
	numWorkers   int
	workers      []*Worker
	workChan     chan WorkItem
	resultChan   chan WorkResult
	statsChan    chan WorkerStats
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// CryptoPool provides object pooling for cryptographic operations
type CryptoPool struct {
	privateKeyPool sync.Pool
	publicKeyPool  sync.Pool
	bigIntPool     sync.Pool
	ecdsaKeyPool   sync.Pool
}

// HasherPool provides object pooling for Keccak256 hash instances
type HasherPool struct {
	keccakPool sync.Pool
}

// BufferPool provides object pooling for byte and string buffers
type BufferPool struct {
	byteBufferPool    sync.Pool
	stringBuilderPool sync.Pool
	hexBufferPool     sync.Pool
}

const (
	// Step defines how many attempts before showing progress
	Step = 500
	// ProgressUpdateInterval defines how often to update progress display
	ProgressUpdateInterval = time.Millisecond * 500
)

// Global object pools for performance optimization
var (
	globalCryptoPool *CryptoPool
	globalHasherPool *HasherPool
	globalBufferPool *BufferPool
)

// detectCPUCount returns the number of available CPU cores
func detectCPUCount() int {
	return runtime.NumCPU()
}

// validateThreads validates the thread count and sets appropriate values
// It handles auto-detection, validation of values, and provides user feedback
func validateThreads() {
	// Check for negative values
	if threads < 0 {
		fmt.Println("❌ Error: Number of threads cannot be negative")
		fmt.Println("💡 Use --threads=0 for automatic detection")
		os.Exit(1)
	}

	// Auto-detect when threads=0
	if threads == 0 {
		cpuCount := detectCPUCount()
		threads = cpuCount
		fmt.Printf("🔧 Auto-detected %d CPU cores, using %d threads\n", cpuCount, threads)
	} else {
		fmt.Printf("🔧 Using %d threads as specified\n", threads)
	}

	// Validate thread count doesn't exceed reasonable limits
	cpuCount := detectCPUCount()
	maxRecommendedThreads := cpuCount * 2 // Allow up to 2x CPU cores as reasonable maximum

	if threads > maxRecommendedThreads {
		fmt.Printf("⚠️  Warning: Using %d threads on %d CPU cores may not be optimal\n", threads, cpuCount)
		fmt.Printf("💡 Recommended maximum is %d threads (2x CPU cores)\n", maxRecommendedThreads)
	}

	// Check for unreasonably high values that might cause system instability
	if threads > 128 {
		fmt.Println("❌ Error: Thread count is unreasonably high")
		fmt.Printf("💡 Maximum allowed is 128 threads, you specified %d\n", threads)
		os.Exit(1)
	}
}

// NewCryptoPool creates a new CryptoPool with initialized pools
func NewCryptoPool() *CryptoPool {
	return &CryptoPool{
		privateKeyPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 32)
			},
		},
		publicKeyPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 64)
			},
		},
		bigIntPool: sync.Pool{
			New: func() interface{} {
				return new(big.Int)
			},
		},
		ecdsaKeyPool: sync.Pool{
			New: func() interface{} {
				return &ecdsa.PrivateKey{
					PublicKey: ecdsa.PublicKey{
						Curve: crypto.S256(),
					},
				}
			},
		},
	}
}

// GetPrivateKeyBuffer gets a private key buffer from the pool
func (cp *CryptoPool) GetPrivateKeyBuffer() []byte {
	return cp.privateKeyPool.Get().([]byte)
}

// PutPrivateKeyBuffer returns a private key buffer to the pool
func (cp *CryptoPool) PutPrivateKeyBuffer(buf []byte) {
	// Clear the buffer for security
	for i := range buf {
		buf[i] = 0
	}
	cp.privateKeyPool.Put(buf)
}

// GetPublicKeyBuffer gets a public key buffer from the pool
func (cp *CryptoPool) GetPublicKeyBuffer() []byte {
	return cp.publicKeyPool.Get().([]byte)
}

// PutPublicKeyBuffer returns a public key buffer to the pool
func (cp *CryptoPool) PutPublicKeyBuffer(buf []byte) {
	// Clear the buffer for security
	for i := range buf {
		buf[i] = 0
	}
	cp.publicKeyPool.Put(buf)
}

// GetBigInt gets a big.Int from the pool
func (cp *CryptoPool) GetBigInt() *big.Int {
	bigInt := cp.bigIntPool.Get().(*big.Int)
	bigInt.SetInt64(0) // Reset to zero
	return bigInt
}

// PutBigInt returns a big.Int to the pool
func (cp *CryptoPool) PutBigInt(bigInt *big.Int) {
	bigInt.SetInt64(0) // Clear for security
	cp.bigIntPool.Put(bigInt)
}

// GetECDSAKey gets an ECDSA private key from the pool
func (cp *CryptoPool) GetECDSAKey() *ecdsa.PrivateKey {
	key := cp.ecdsaKeyPool.Get().(*ecdsa.PrivateKey)
	// Reset the key
	key.D = nil
	key.PublicKey.X = nil
	key.PublicKey.Y = nil
	return key
}

// PutECDSAKey returns an ECDSA private key to the pool
func (cp *CryptoPool) PutECDSAKey(key *ecdsa.PrivateKey) {
	// Clear sensitive data
	if key.D != nil {
		key.D.SetInt64(0)
	}
	if key.PublicKey.X != nil {
		key.PublicKey.X.SetInt64(0)
	}
	if key.PublicKey.Y != nil {
		key.PublicKey.Y.SetInt64(0)
	}
	cp.ecdsaKeyPool.Put(key)
}

// NewHasherPool creates a new HasherPool with initialized Keccak256 pool
func NewHasherPool() *HasherPool {
	return &HasherPool{
		keccakPool: sync.Pool{
			New: func() interface{} {
				return sha3.NewLegacyKeccak256()
			},
		},
	}
}

// GetKeccak gets a Keccak256 hasher from the pool
func (hp *HasherPool) GetKeccak() hash.Hash {
	hasher := hp.keccakPool.Get().(hash.Hash)
	hasher.Reset() // Reset the hasher state
	return hasher
}

// PutKeccak returns a Keccak256 hasher to the pool
func (hp *HasherPool) PutKeccak(hasher hash.Hash) {
	hasher.Reset() // Clear any remaining state
	hp.keccakPool.Put(hasher)
}

// NewBufferPool creates a new BufferPool with initialized buffer pools
func NewBufferPool() *BufferPool {
	return &BufferPool{
		byteBufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 64) // Pre-allocate capacity for typical use
			},
		},
		stringBuilderPool: sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
		hexBufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 64) // For hex encoding/decoding
			},
		},
	}
}

// GetByteBuffer gets a byte buffer from the pool
func (bp *BufferPool) GetByteBuffer() []byte {
	buf := bp.byteBufferPool.Get().([]byte)
	return buf[:0] // Reset length but keep capacity
}

// PutByteBuffer returns a byte buffer to the pool
func (bp *BufferPool) PutByteBuffer(buf []byte) {
	// Clear the buffer for security
	for i := range buf {
		buf[i] = 0
	}
	bp.byteBufferPool.Put(buf[:0])
}

// GetStringBuilder gets a string builder from the pool
func (bp *BufferPool) GetStringBuilder() *strings.Builder {
	sb := bp.stringBuilderPool.Get().(*strings.Builder)
	sb.Reset() // Clear any existing content
	return sb
}

// PutStringBuilder returns a string builder to the pool
func (bp *BufferPool) PutStringBuilder(sb *strings.Builder) {
	sb.Reset() // Clear content
	bp.stringBuilderPool.Put(sb)
}

// GetHexBuffer gets a hex buffer from the pool
func (bp *BufferPool) GetHexBuffer() []byte {
	return bp.hexBufferPool.Get().([]byte)
}

// PutHexBuffer returns a hex buffer to the pool
func (bp *BufferPool) PutHexBuffer(buf []byte) {
	// Clear the buffer for security
	for i := range buf {
		buf[i] = 0
	}
	bp.hexBufferPool.Put(buf)
}

// setupSignalHandling configures global signal handling for graceful shutdown
func setupSignalHandling() {
	globalCtx, globalCancel = context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n🛑 Received interrupt signal, shutting down gracefully...\n")
		globalCancel()
	}()
}

// initializePools initializes the global object pools
func initializePools() {
	globalCryptoPool = NewCryptoPool()
	globalHasherPool = NewHasherPool()
	globalBufferPool = NewBufferPool()
}

// computeDifficulty calculates the difficulty of finding a bloco address
func computeDifficulty(prefix, suffix string, isChecksum bool) float64 {
	pattern := prefix + suffix
	baseDifficulty := math.Pow(16, float64(len(pattern)))

	if !isChecksum {
		return baseDifficulty
	}

	// Count letters (a-f, A-F) in the pattern for checksum calculation
	letterCount := 0
	for _, char := range pattern {
		if (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F') {
			letterCount++
		}
	}

	return baseDifficulty * math.Pow(2, float64(letterCount))
}

// computeProbability calculates the probability of finding an address after N attempts
func computeProbability(difficulty float64, attempts int64) float64 {
	if difficulty <= 0 {
		return 0
	}
	return 1 - math.Pow(1-1/difficulty, float64(attempts))
}

// computeProbability50 calculates how many attempts are needed for 50% probability
func computeProbability50(difficulty float64) int64 {
	if difficulty <= 0 {
		return 0
	}
	result := math.Log(0.5) / math.Log(1-1/difficulty)
	if math.IsInf(result, 0) || result < 0 {
		return -1 // Nearly impossible
	}
	return int64(math.Floor(result))
}

// isValidHex checks if a string contains only valid hex characters
func isValidHex(hex string) bool {
	if len(hex) == 0 {
		return true
	}
	for _, char := range hex {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

// formatNumber formats a number with space separators for thousands
func formatNumber(num int64) string {
	str := strconv.FormatInt(num, 10)
	result := ""
	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += " "
		}
		result += string(char)
	}
	return result
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "Nearly impossible"
	}

	seconds := d.Seconds()

	// If more than 200 years, return "Thousands of years"
	if seconds > 200*365.25*24*3600 {
		return "Thousands of years"
	}

	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%.1fm", seconds/60)
	} else if seconds < 86400 {
		return fmt.Sprintf("%.1fh", seconds/3600)
	} else if seconds < 31536000 {
		return fmt.Sprintf("%.1fd", seconds/86400)
	} else {
		return fmt.Sprintf("%.1fy", seconds/31536000)
	}
}

// newStatistics creates a new Statistics instance
func newStatistics(prefix, suffix string, isChecksum bool) *Statistics {
	pattern := prefix + suffix
	difficulty := computeDifficulty(prefix, suffix, isChecksum)
	probability50 := computeProbability50(difficulty)

	return &Statistics{
		Difficulty:      difficulty,
		Probability50:   probability50,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         pattern,
		IsChecksum:      isChecksum,
	}
}

// update updates the statistics with new attempt count
func (s *Statistics) update(attempts int64) {
	s.CurrentAttempts = attempts
	s.Probability = computeProbability(s.Difficulty, attempts) * 100

	now := time.Now()
	elapsed := now.Sub(s.StartTime)

	if elapsed.Seconds() > 0 {
		s.Speed = float64(attempts) / elapsed.Seconds()

		if s.Probability50 > 0 && s.Speed > 0 {
			remainingAttempts := s.Probability50 - attempts
			if remainingAttempts > 0 {
				s.EstimatedTime = time.Duration(float64(remainingAttempts)/s.Speed) * time.Second
			} else {
				s.EstimatedTime = 0
			}
		}
	}

	s.LastUpdate = now
}

// updateFromAggregated updates statistics from aggregated multi-thread data
func (s *Statistics) updateFromAggregated(aggregated AggregatedStats) {
	s.CurrentAttempts = aggregated.TotalAttempts
	s.Probability = aggregated.Probability
	s.Speed = aggregated.TotalSpeed
	s.EstimatedTime = aggregated.EstimatedTime
	s.LastUpdate = aggregated.LastUpdate
}

// displayProgress shows a progress bar and statistics
func (s *Statistics) displayProgress() {
	// Clear line and move cursor to beginning
	fmt.Print("\r\033[K")

	// Calculate progress bar
	barWidth := 40
	progressPercent := s.Probability
	if progressPercent > 100 {
		progressPercent = 100
	}

	filledWidth := int((progressPercent / 100) * float64(barWidth))

	// Create progress bar
	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	// Format output
	fmt.Printf("%s %.2f%% | %s attempts | %.0f addr/s | Difficulty: %s",
		bar,
		s.Probability,
		formatNumber(s.CurrentAttempts),
		s.Speed,
		formatNumber(int64(s.Difficulty)),
	)

	if s.EstimatedTime > 0 {
		fmt.Printf(" | ETA: %s", formatDuration(s.EstimatedTime))
	}
}

// displayProgressParallel shows progress for parallel wallet generation
func displayProgressParallel(stats *Statistics, statsManager *StatsManager, shutdownChan chan struct{}, walletResultsChan chan WalletResultForTUI) {
	// Check if TUI should be used
	tuiManager := tui.NewTUIManager()
	if tuiManager.ShouldUseTUI() {
		
		return
	}

	// Fallback to text-based progress display
	progressManager := NewProgressManager(stats, statsManager)
	progressManager.Start()
	<-shutdownChan
	progressManager.Stop()
}

// SilentStatsUpdater updates statistics without displaying progress
type SilentStatsUpdater struct {
	stats        *Statistics
	statsManager *StatsManager
	ticker       *time.Ticker
	stopChan     chan struct{}
	running      bool
}

// NewSilentStatsUpdater creates a new silent stats updater
func NewSilentStatsUpdater(stats *Statistics, statsManager *StatsManager) *SilentStatsUpdater {
	return &SilentStatsUpdater{
		stats:        stats,
		statsManager: statsManager,
		stopChan:     make(chan struct{}),
	}
}

// Start begins the silent update process
func (ssu *SilentStatsUpdater) Start() {
	if ssu.running {
		return
	}
	
	ssu.running = true
	ssu.ticker = time.NewTicker(100 * time.Millisecond)
	
	go func() {
		for {
			select {
			case <-ssu.ticker.C:
				// Update stats with current data from workers
				metrics := ssu.statsManager.GetMetrics()
				ssu.stats.Speed = metrics.TotalThroughput
				ssu.stats.LastUpdate = time.Now()
			case <-ssu.stopChan:
				return
			}
		}
	}()
}

// Stop stops the silent update process
func (ssu *SilentStatsUpdater) Stop() {
	if !ssu.running {
		return
	}
	
	ssu.running = false
	
	if ssu.ticker != nil {
		ssu.ticker.Stop()
	}
	
	close(ssu.stopChan)
	ssu.stopChan = make(chan struct{}) // Recreate for potential reuse
}

// StatsManagerAdapter adapts the main StatsManager to the TUI interface
type StatsManagerAdapter struct {
	*StatsManager
}

func (sma *StatsManagerAdapter) GetMetrics() tui.ThreadMetrics {
	metrics := sma.StatsManager.GetMetrics()
	return tui.ThreadMetrics{
		EfficiencyRatio: metrics.EfficiencyRatio,
		TotalSpeed:      metrics.TotalThroughput,
		ThreadCount:     metrics.WorkerCount,
	}
}

// WalletResultForTUI represents a wallet result that can be sent to TUI
type WalletResultForTUI struct {
	Index      int
	Address    string
	PrivateKey string
	Attempts   int
	Duration   time.Duration
	Error      string
}

// StatsUpdateForTUI represents a statistics update message for TUI
type StatsUpdateForTUI struct {
	Attempts         int64
	Speed            float64
	Probability      float64
	ETA              time.Duration
	CompletedWallets int     // Number of wallets successfully generated
	TotalWallets     int     // Total wallets requested
	ProgressPercent  float64 // Progress as percentage (0-100)
	Difficulty       float64 // The calculated difficulty
}

// displayProgressTUI shows progress using TUI interface with wallet results


func privateToAddress(privateKey []byte) string {
	// Get objects from pools
	privateKeyInt := globalCryptoPool.GetBigInt()
	privateKeyECDSA := globalCryptoPool.GetECDSAKey()
	hasher := globalHasherPool.GetKeccak()
	publicKeyBytes := globalCryptoPool.GetPublicKeyBuffer()
	hexBuffer := globalBufferPool.GetHexBuffer()

	defer func() {
		// Return objects to pools
		globalCryptoPool.PutBigInt(privateKeyInt)
		globalCryptoPool.PutECDSAKey(privateKeyECDSA)
		globalHasherPool.PutKeccak(hasher)
		globalCryptoPool.PutPublicKeyBuffer(publicKeyBytes)
		globalBufferPool.PutHexBuffer(hexBuffer)
	}()

	// Convert private key bytes to ECDSA private key
	privateKeyInt.SetBytes(privateKey)
	privateKeyECDSA.D = privateKeyInt
	privateKeyECDSA.PublicKey.Curve = crypto.S256()

	// Calculate public key coordinates
	privateKeyECDSA.PublicKey.X, privateKeyECDSA.PublicKey.Y = crypto.S256().ScalarBaseMult(privateKey)

	// Get uncompressed public key bytes (without 0x04 prefix)
	// Using pre-allocated buffer instead of creating a new one
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

	// Use pre-allocated buffer for hex encoding
	hex.Encode(hexBuffer, address)
	return string(hexBuffer[:40])
}

// getRandomWallet generates a random wallet with private key and address
func getRandomWallet() (*Wallet, error) {
	// Get private key buffer from pool
	privateKey := globalCryptoPool.GetPrivateKeyBuffer()
	defer globalCryptoPool.PutPrivateKeyBuffer(privateKey)

	// Generate 32 random bytes for private key
	_, err := rand.Read(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %v", err)
	}

	// Generate address from private key
	address := privateToAddress(privateKey)
	if address == "" {
		return nil, fmt.Errorf("failed to generate address from private key")
	}

	return &Wallet{
		Address: address,
		PrivKey: hex.EncodeToString(privateKey),
	}, nil
}

// isValidChecksum validates the checksum of an Ethereum address
func isValidChecksum(address, prefix, suffix string) bool {
	// Get hasher, string builder, and byte buffer from pools
	hasher := globalHasherPool.GetKeccak()
	sb := globalBufferPool.GetStringBuilder()
	addressBytes := globalBufferPool.GetByteBuffer()

	defer func() {
		globalHasherPool.PutKeccak(hasher)
		globalBufferPool.PutStringBuilder(sb)
		globalBufferPool.PutByteBuffer(addressBytes)
	}()

	// Convert address to bytes without allocating a new slice
	addressBytes = append(addressBytes, address...)

	// Calculate Keccak256 hash of the address
	hasher.Reset()
	hasher.Write(addressBytes)

	// Use a fixed-size array for hash bytes to avoid heap allocation
	var hashBytes [32]byte
	hasher.Sum(hashBytes[:0])

	// Pre-allocate capacity in string builder
	sb.Grow(64) // 32 bytes * 2 hex chars per byte

	// Convert hash to hex string using string builder and lookup table for efficiency
	const hexChars = "0123456789abcdef"
	for _, b := range hashBytes {
		sb.WriteByte(hexChars[b>>4])
		sb.WriteByte(hexChars[b&0x0f])
	}

	// Get the hash string without allocating a new string
	hash := sb.String()

	// Fast path: if prefix and suffix are empty, return true
	if len(prefix) == 0 && len(suffix) == 0 {
		return true
	}

	// Check prefix checksum - avoid string allocations in the loop
	for i := 0; i < len(prefix); i++ {
		// Convert hex char to int directly without ParseInt
		var hashChar int64
		hexChar := hash[i]
		if hexChar >= '0' && hexChar <= '9' {
			hashChar = int64(hexChar - '0')
		} else if hexChar >= 'a' && hexChar <= 'f' {
			hashChar = int64(hexChar - 'a' + 10)
		}

		addrChar := address[i]
		var expectedChar byte
		if hashChar >= 8 {
			// Convert to uppercase if needed
			if addrChar >= 'a' && addrChar <= 'f' {
				expectedChar = addrChar - 32 // 'a' to 'A' ASCII difference
			} else {
				expectedChar = addrChar
			}
		} else {
			// Convert to lowercase if needed
			if addrChar >= 'A' && addrChar <= 'F' {
				expectedChar = addrChar + 32 // 'A' to 'a' ASCII difference
			} else {
				expectedChar = addrChar
			}
		}

		if prefix[i] != expectedChar {
			return false
		}
	}

	// Check suffix checksum - avoid string allocations in the loop
	for i := 0; i < len(suffix); i++ {
		j := i + 40 - len(suffix)

		// Convert hex char to int directly without ParseInt
		var hashChar int64
		hexChar := hash[j]
		if hexChar >= '0' && hexChar <= '9' {
			hashChar = int64(hexChar - '0')
		} else if hexChar >= 'a' && hexChar <= 'f' {
			hashChar = int64(hexChar - 'a' + 10)
		}

		addrChar := address[j]
		var expectedChar byte
		if hashChar >= 8 {
			// Convert to uppercase if needed
			if addrChar >= 'a' && addrChar <= 'f' {
				expectedChar = addrChar - 32 // 'a' to 'A' ASCII difference
			} else {
				expectedChar = addrChar
			}
		} else {
			// Convert to lowercase if needed
			if addrChar >= 'A' && addrChar <= 'F' {
				expectedChar = addrChar + 32 // 'A' to 'a' ASCII difference
			} else {
				expectedChar = addrChar
			}
		}

		if suffix[i] != expectedChar {
			return false
		}
	}

	return true
}

// isValidBlocoAddress checks if a wallet address matches the given constraints
func isValidBlocoAddress(address, prefix, suffix string, isChecksum bool) bool {
	// Protect against index out of range
	if len(prefix) > len(address) {
		return false
	}
	if len(suffix) > len(address) {
		return false
	}

	// Fast path: if both prefix and suffix are empty, return true immediately
	if len(prefix) == 0 && len(suffix) == 0 {
		return true
	}

	// Extract prefix and suffix from address - no allocations here
	var addressPrefix, addressSuffix string
	if len(prefix) > 0 {
		addressPrefix = address[:len(prefix)]
	}
	if len(suffix) > 0 {
		addressSuffix = address[len(address)-len(suffix):]
	}

	if !isChecksum {
		// For non-checksum validation, just compare case-insensitively
		// Use direct comparison for empty strings to avoid function call overhead
		prefixMatch := len(prefix) == 0 || strings.EqualFold(prefix, addressPrefix)
		suffixMatch := len(suffix) == 0 || strings.EqualFold(suffix, addressSuffix)
		return prefixMatch && suffixMatch
	}

	// For checksum validation, first check if lowercase versions match
	// This avoids expensive checksum validation when there's no match
	prefixMatch := len(prefix) == 0 || strings.EqualFold(prefix, addressPrefix)
	suffixMatch := len(suffix) == 0 || strings.EqualFold(suffix, addressSuffix)

	if !prefixMatch || !suffixMatch {
		return false
	}

	return isValidChecksum(address, prefix, suffix)
}

// toChecksumAddress converts an address to checksum format
func toChecksumAddress(address string) string {
	// Get hasher, string builders, and byte buffer from pools
	hasher := globalHasherPool.GetKeccak()
	hashSB := globalBufferPool.GetStringBuilder()
	resultSB := globalBufferPool.GetStringBuilder()
	addressBytes := globalBufferPool.GetByteBuffer()

	defer func() {
		globalHasherPool.PutKeccak(hasher)
		globalBufferPool.PutStringBuilder(hashSB)
		globalBufferPool.PutStringBuilder(resultSB)
		globalBufferPool.PutByteBuffer(addressBytes)
	}()

	// Convert address to bytes without allocating a new slice
	addressBytes = append(addressBytes, address...)

	// Calculate Keccak256 hash of the address
	hasher.Reset()
	hasher.Write(addressBytes)

	// Use a fixed-size array for hash bytes to avoid heap allocation
	var hashBytes [32]byte
	hasher.Sum(hashBytes[:0])

	// Pre-allocate capacity in string builders
	hashSB.Grow(64)   // 32 bytes * 2 hex chars per byte
	resultSB.Grow(40) // Address length

	// Convert hash to hex string using string builder and lookup table for efficiency
	const hexChars = "0123456789abcdef"
	for _, b := range hashBytes {
		hashSB.WriteByte(hexChars[b>>4])
		hashSB.WriteByte(hexChars[b&0x0f])
	}
	hash := hashSB.String()

	// Build result using string builder for efficiency - avoid allocations in the loop
	for i, char := range address {
		// Convert hex char to int directly without ParseInt
		var hashChar int64
		hexChar := hash[i]
		if hexChar >= '0' && hexChar <= '9' {
			hashChar = int64(hexChar - '0')
		} else if hexChar >= 'a' && hexChar <= 'f' {
			hashChar = int64(hexChar - 'a' + 10)
		}

		if hashChar >= 8 {
			// Convert to uppercase if needed
			if char >= 'a' && char <= 'f' {
				resultSB.WriteByte(byte(char) - 32) // 'a' to 'A' ASCII difference
			} else {
				resultSB.WriteByte(byte(char))
			}
		} else {
			// Convert to lowercase if needed
			if char >= 'A' && char <= 'F' {
				resultSB.WriteByte(byte(char) + 32) // 'A' to 'a' ASCII difference
			} else {
				resultSB.WriteByte(byte(char))
			}
		}
	}
	return resultSB.String()
}

// generateBlocoWallet generates a wallet that matches the given constraints with statistics
func generateBlocoWallet(prefix, suffix string, isChecksum bool, showProgress bool) *WalletResult {
	// Use background context for backward compatibility
	ctx := context.Background()
	return generateBlocoWalletWithContext(ctx, prefix, suffix, isChecksum, showProgress)
}

// generateBlocoWalletWithContext generates a wallet that matches the given constraints with context cancellation support
func generateBlocoWalletWithContext(ctx context.Context, prefix, suffix string, isChecksum bool, showProgress bool) *WalletResult {
	pre := prefix
	suf := suffix

	if !isChecksum {
		pre = strings.ToLower(prefix)
		suf = strings.ToLower(suffix)
	}

	// For testing or simple cases, use single-threaded generation
	if !showProgress || threads <= 1 {
		return generateBlocoWalletSingleThreadWithContext(ctx, pre, suf, isChecksum, showProgress)
	}

	// Initialize statistics
	var stats *Statistics
	if showProgress {
		stats = newStatistics(prefix, suffix, isChecksum)
		
		// Only show CLI output if TUI is not being used
		tuiManager := tui.NewTUIManager()
		if !tuiManager.ShouldUseTUI() {
			fmt.Printf("\n🎯 Generating bloco wallet with pattern: %s%s%s\n", prefix, strings.Repeat("*", 40-len(prefix)-len(suffix)), suffix)
			fmt.Printf("📊 Difficulty: %s | 50%% probability: %s attempts\n\n",
				formatNumber(int64(stats.Difficulty)),
				formatNumber(stats.Probability50))
			fmt.Printf("🧵 Using %d worker threads\n\n", threads)
		}
	}

	// Create a worker pool with the specified number of threads
	workerPool := NewWorkerPool(threads)

	// Create a stats manager for thread-safe statistics collection
	statsManager := NewStatsManager()

	// Start the worker pool
	workerPool.Start()

	// Start a goroutine to display progress
	if showProgress {
		// For single wallet generation, we don't need wallet results channel
		go displayProgressParallel(stats, statsManager, workerPool.shutdownChan, nil)
	}

	// Generate wallet using the worker pool with context
	wallet, attempts := workerPool.GenerateWalletWithContext(ctx, pre, suf, isChecksum, statsManager)

	// Final progress update
	if showProgress {
		stats.update(attempts)
		
		// Only show CLI output if TUI is not being used
		tuiManager := tui.NewTUIManager()
		if !tuiManager.ShouldUseTUI() {
			stats.displayProgress()
			fmt.Printf("\n✅ Success! Found matching address in %s attempts\n", formatNumber(attempts))

			// Display thread utilization statistics
			metrics := statsManager.GetMetrics()
			fmt.Printf("🧵 Thread utilization: %.2f%% efficiency\n", metrics.EfficiencyRatio*100)
			fmt.Printf("⚡ Peak performance: %.0f addr/s\n\n", statsManager.GetPeakSpeed())
		}
	}

	if wallet != nil {
		checksumAddress := "0x" + toChecksumAddress(wallet.Address)
		return &WalletResult{
			Wallet: &Wallet{
				Address: checksumAddress,
				PrivKey: wallet.PrivKey,
			},
			Attempts: int(attempts),
		}
	}

	// This should not happen unless there's an error
	return &WalletResult{
		Error:    "Failed to generate wallet",
		Attempts: int(attempts),
	}
}

// generateBlocoWalletSingleThread generates a wallet using a simple single-threaded approach
func generateBlocoWalletSingleThread(prefix, suffix string, isChecksum bool, showProgress bool) *WalletResult {
	// Use background context for backward compatibility
	ctx := context.Background()
	return generateBlocoWalletSingleThreadWithContext(ctx, prefix, suffix, isChecksum, showProgress)
}

// generateBlocoWalletSingleThreadWithContext generates a wallet using a simple single-threaded approach with context cancellation
func generateBlocoWalletSingleThreadWithContext(ctx context.Context, prefix, suffix string, isChecksum bool, showProgress bool) *WalletResult {
	var attempts int = 0
	var stats *Statistics

	if showProgress {
		stats = newStatistics(prefix, suffix, isChecksum)
		
		// Only show CLI output if TUI is not being used
		tuiManager := tui.NewTUIManager()
		if !tuiManager.ShouldUseTUI() {
			fmt.Printf("\n🎯 Generating bloco wallet with pattern: %s%s%s\n", prefix, strings.Repeat("*", 40-len(prefix)-len(suffix)), suffix)
			fmt.Printf("📊 Difficulty: %s | 50%% probability: %s attempts\n\n",
				formatNumber(int64(stats.Difficulty)),
				formatNumber(stats.Probability50))
		}
	}

	lastProgressUpdate := time.Now()

	for {
		// Check for context cancellation every 100 attempts to avoid excessive overhead
		if attempts%100 == 0 {
			select {
			case <-ctx.Done():
				if showProgress {
					// Only show CLI output if TUI is not being used
					tuiManager := tui.NewTUIManager()
					if !tuiManager.ShouldUseTUI() {
						fmt.Printf("\n🛑 Generation cancelled after %s attempts\n", formatNumber(int64(attempts)))
						fmt.Printf("Reason: %v\n", ctx.Err())
					}
				}
				return &WalletResult{
					Error:    fmt.Sprintf("Generation cancelled: %v", ctx.Err()),
					Attempts: attempts,
				}
			default:
			}
		}

		attempts++

		// Generate a random wallet
		wallet, err := getRandomWallet()
		if err != nil {
			return &WalletResult{
				Error:    fmt.Sprintf("Failed to generate wallet: %v", err),
				Attempts: attempts,
			}
		}

		// Check if the address matches our pattern
		if isValidBlocoAddress(wallet.Address, prefix, suffix, isChecksum) {
			// Found a match!
			checksumAddress := "0x" + toChecksumAddress(wallet.Address)

			if showProgress {
				stats.update(int64(attempts))
				
				// Only show CLI output if TUI is not being used
				tuiManager := tui.NewTUIManager()
				if !tuiManager.ShouldUseTUI() {
					stats.displayProgress()
					fmt.Printf("\n✅ Success! Found matching address in %s attempts\n", formatNumber(int64(attempts)))
				}
			}

			return &WalletResult{
				Wallet: &Wallet{
					Address: checksumAddress,
					PrivKey: wallet.PrivKey,
				},
				Attempts: attempts,
			}
		}

		// Update progress periodically
		if showProgress && time.Since(lastProgressUpdate) >= ProgressUpdateInterval {
			stats.update(int64(attempts))
			stats.displayProgress()
			lastProgressUpdate = time.Now()
		}
	}
}

// runQuickSingleThreadBenchmark runs a quick benchmark with a single thread to establish baseline performance
func runQuickSingleThreadBenchmark(pattern string, isChecksum bool) float64 {
	// Use background context for backward compatibility
	ctx := context.Background()
	return runQuickSingleThreadBenchmarkWithContext(ctx, pattern, isChecksum)
}

// runQuickSingleThreadBenchmarkWithContext runs a quick benchmark with context cancellation support
func runQuickSingleThreadBenchmarkWithContext(ctx context.Context, pattern string, isChecksum bool) float64 {
	fmt.Printf("🔍 Running quick single-thread benchmark for baseline...\n")

	// Save original thread count and temporarily set to 1
	originalThreads := threads
	threads = 1

	// Create a worker pool with just one thread
	workerPool := NewWorkerPool(1)

	// Create a stats manager for thread-safe statistics collection
	statsManager := NewStatsManager()

	// Start the worker pool
	workerPool.Start()

	// Start collecting stats
	workerPool.CollectStats(statsManager)

	// Number of attempts for quick benchmark - enough to get a stable measurement
	quickBenchmarkAttempts := int64(10000)

	// Submit work
	workerPool.Submit(WorkItem{
		Prefix:     pattern,
		Suffix:     "",
		IsChecksum: isChecksum,
		BatchSize:  int(quickBenchmarkAttempts),
	})

	// Wait for completion
	startTime := time.Now()

	// Wait for the benchmark to complete or timeout
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Track speed samples to get a more stable measurement
	var speedSamples []float64
	lastAttempts := int64(0)
	lastTime := startTime

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, return a reasonable estimate
			attempts := statsManager.GetTotalAttempts()
			if attempts > 0 {
				duration := time.Since(startTime)
				speed := float64(attempts) / duration.Seconds()
				workerPool.Shutdown()
				threads = originalThreads
				return speed
			}
			workerPool.Shutdown()
			threads = originalThreads
			return 1000 // Default fallback speed
		case <-ticker.C:
			now := time.Now()
			attempts := statsManager.GetTotalAttempts()

			// Calculate speed for this sample
			if attempts > lastAttempts {
				sampleDuration := now.Sub(lastTime)
				sampleAttempts := attempts - lastAttempts
				sampleSpeed := float64(sampleAttempts) / sampleDuration.Seconds()

				// Only add valid samples
				if sampleSpeed > 0 {
					speedSamples = append(speedSamples, sampleSpeed)
				}

				lastAttempts = attempts
				lastTime = now
			}

			if attempts >= quickBenchmarkAttempts {
				// Benchmark completed
				duration := time.Since(startTime)
				overallSpeed := float64(attempts) / duration.Seconds()

				// Calculate median speed from samples for more stability
				var medianSpeed float64
				if len(speedSamples) > 0 {
					// Sort samples and take median
					sort.Float64s(speedSamples)
					medianSpeed = speedSamples[len(speedSamples)/2]

					// Use median if it's reasonable, otherwise use overall speed
					if medianSpeed > 0 && medianSpeed < overallSpeed*2 {
						overallSpeed = medianSpeed
					}
				}

				// Shutdown the worker pool
				workerPool.Shutdown()

				// Restore original thread count
				threads = originalThreads

				fmt.Printf("✓ Single-thread baseline: %.0f addr/s\n", overallSpeed)
				return overallSpeed
			}
		case <-timeout:
			// Timeout after 15 seconds
			attempts := statsManager.GetTotalAttempts()
			duration := time.Since(startTime)
			speed := float64(attempts) / duration.Seconds()

			// Shutdown the worker pool
			workerPool.Shutdown()

			// Restore original thread count
			threads = originalThreads

			fmt.Printf("⚠️ Single-thread benchmark timed out, using estimate: %.0f addr/s\n", speed)
			return speed
		case <-ctx.Done():
			// Context cancelled
			attempts := statsManager.GetTotalAttempts()
			if attempts > 0 {
				duration := time.Since(startTime)
				speed := float64(attempts) / duration.Seconds()
				workerPool.Shutdown()
				threads = originalThreads
				return speed
			}
			workerPool.Shutdown()
			threads = originalThreads
			return 1000 // Default fallback speed
		}
	}
}

// runBenchmark runs a performance benchmark with multi-thread support and scalability analysis
func runBenchmark(maxAttempts int64, pattern string, isChecksum bool) *BenchmarkResult {
	// Check if TUI should be used
	tuiManager := tui.NewTUIManager()
	if tuiManager.ShouldUseTUI() {
		return runBenchmarkTUI(maxAttempts, pattern, isChecksum)
	}
	
	// Use global context for signal handling
	return runBenchmarkWithContext(globalCtx, maxAttempts, pattern, isChecksum)
}

// runBenchmarkTUI runs a performance benchmark using TUI interface
func runBenchmarkTUI(maxAttempts int64, pattern string, isChecksum bool) *BenchmarkResult {
	ctx := globalCtx
	
	if maxAttempts == 0 {
		maxAttempts = 10000
	}

	fmt.Printf("🚀 Starting multi-threaded benchmark with pattern '%s' (checksum: %t)\n", pattern, isChecksum)
	fmt.Printf("📈 Target: %s attempts | Step size: %d\n", formatNumber(maxAttempts), Step)

	// First run a quick single-thread benchmark to establish baseline
	singleThreadSpeed := runQuickSingleThreadBenchmarkWithContext(ctx, pattern, isChecksum)

	result := &BenchmarkResult{
		SpeedSamples:      make([]float64, 0),
		DurationSamples:   make([]time.Duration, 0),
		SingleThreadSpeed: singleThreadSpeed,
		ThreadCount:       threads,
		MinSpeed:          math.MaxFloat64,
		MaxSpeed:          0,
	}

	// Create TUI benchmark model
	tuiManager := tui.NewTUIManager()
	benchmarkModel := tuiManager.CreateBenchmarkModel()

	// Create TUI program
	program := tea.NewProgram(benchmarkModel, tea.WithAltScreen())

	// Create a worker pool with the specified number of threads
	workerPool := NewWorkerPool(threads)

	// Start the worker pool
	workerPool.Start()

	var attempts int64 = 0
	startTime := time.Now()
	done := make(chan struct{})

	// Start benchmark goroutine
	go func() {
		defer close(done)
		
		// Give TUI time to initialize
		time.Sleep(200 * time.Millisecond)
		
		for attempts < maxAttempts {
			select {
			case <-ctx.Done():
				return
			default:
				// Run benchmark step
				stepStart := time.Now()
				
				// Create work items for the benchmark step
				workItems := make([]WorkItem, Step)
				for i := range workItems {
					workItems[i] = WorkItem{
						Prefix:     pattern,
						Suffix:     "",
						IsChecksum: isChecksum,
						BatchSize:  1,
					}
				}
				
				// Submit batch work
				workerPool.SubmitBatch(workItems)
				
				attempts += Step
				stepDuration := time.Since(stepStart)
				
				// Calculate speed for this step
				stepSpeed := float64(Step) / stepDuration.Seconds()
				result.SpeedSamples = append(result.SpeedSamples, stepSpeed)
				result.DurationSamples = append(result.DurationSamples, stepDuration)
				
				// Update min/max speeds
				if len(result.SpeedSamples) == 1 {
					result.MinSpeed = stepSpeed
					result.MaxSpeed = stepSpeed
				} else {
					if stepSpeed < result.MinSpeed {
						result.MinSpeed = stepSpeed
					}
					if stepSpeed > result.MaxSpeed {
						result.MaxSpeed = stepSpeed
					}
				}
				
				// Calculate average speed so far
				totalSpeed := 0.0
				for _, s := range result.SpeedSamples {
					totalSpeed += s
				}
				avgSpeed := totalSpeed / float64(len(result.SpeedSamples))
				
				// Calculate current scalability efficiency
				currentEfficiency := 0.0
				if singleThreadSpeed > 0 {
					actualSpeedup := avgSpeed / singleThreadSpeed
					idealSpeedup := float64(threads)
					currentEfficiency = actualSpeedup / idealSpeedup
				}
				
				// Update TUI with progress
				program.Send(tui.BenchmarkUpdateMsg{
					Results: &tui.BenchmarkResult{
						TotalAttempts:         attempts,
						TotalDuration:         time.Since(startTime),
						AverageSpeed:          avgSpeed,
						MinSpeed:              result.MinSpeed,
						MaxSpeed:              result.MaxSpeed,
						SpeedSamples:          result.SpeedSamples,
						DurationSamples:       result.DurationSamples,
						SingleThreadSpeed:     result.SingleThreadSpeed,
						ThreadCount:           result.ThreadCount,
						ScalabilityEfficiency: currentEfficiency,
					},
					Running: true,
					Progress: tui.ProgressMsg{
						Attempts:      attempts,
						Speed:         stepSpeed,
						Pattern:       pattern,
						Difficulty:    computeDifficulty(pattern, "", isChecksum),
						EstimatedTime: time.Duration(float64(maxAttempts-attempts)/avgSpeed) * time.Second,
					},
				})
				
				time.Sleep(10 * time.Millisecond) // Small delay for UI updates
			}
		}
	}()

	// Start a goroutine to handle TUI completion
	go func() {
		select {
		case <-done:
			// Benchmark completed, calculate final results
			result.TotalAttempts = attempts
			result.TotalDuration = time.Since(startTime)
			
			if len(result.SpeedSamples) > 0 {
				total := 0.0
				min := result.SpeedSamples[0]
				max := result.SpeedSamples[0]
				
				for _, speed := range result.SpeedSamples {
					total += speed
					if speed < min {
						min = speed
					}
					if speed > max {
						max = speed
					}
				}
				
				result.AverageSpeed = total / float64(len(result.SpeedSamples))
				result.MinSpeed = min
				result.MaxSpeed = max
			}
			
			// Calculate scalability efficiency
			if singleThreadSpeed > 0 {
				actualSpeedup := result.AverageSpeed / singleThreadSpeed
				idealSpeedup := float64(threads)
				result.ScalabilityEfficiency = actualSpeedup / idealSpeedup
			}
			
			// Send completion message to TUI
			program.Send(tui.BenchmarkCompleteMsg{
				Results: &tui.BenchmarkResult{
					TotalAttempts:         result.TotalAttempts,
					TotalDuration:         result.TotalDuration,
					AverageSpeed:          result.AverageSpeed,
					MinSpeed:              result.MinSpeed,
					MaxSpeed:              result.MaxSpeed,
					SpeedSamples:          result.SpeedSamples,
					DurationSamples:       result.DurationSamples,
					SingleThreadSpeed:     result.SingleThreadSpeed,
					ThreadCount:           result.ThreadCount,
					ScalabilityEfficiency: result.ScalabilityEfficiency,
				},
			})
			
		case <-ctx.Done():
			// Context cancelled
		}
	}()

	// Run the TUI program
	if _, err := program.Run(); err != nil {
		// If TUI fails, stop workers and return text-based fallback
		workerPool.Shutdown()
		fmt.Printf("TUI failed: %v, falling back to text mode\n", err)
		return runBenchmarkWithContext(ctx, maxAttempts, pattern, isChecksum)
	}

	// Stop the worker pool
	workerPool.Shutdown()

	return result
}

// runBenchmarkWithContext runs a performance benchmark with context cancellation support
func runBenchmarkWithContext(ctx context.Context, maxAttempts int64, pattern string, isChecksum bool) *BenchmarkResult {
	if maxAttempts == 0 {
		maxAttempts = 10000
	}

	fmt.Printf("🚀 Starting multi-threaded benchmark with pattern '%s' (checksum: %t)\n", pattern, isChecksum)
	fmt.Printf("📈 Target: %s attempts | Step size: %d\n", formatNumber(maxAttempts), Step)

	// First run a quick single-thread benchmark to establish baseline
	singleThreadSpeed := runQuickSingleThreadBenchmarkWithContext(ctx, pattern, isChecksum)

	fmt.Printf("🧵 Using %d worker threads for multi-threaded benchmark\n", threads)
	fmt.Printf("🔍 Single-thread baseline: %.0f addr/s\n\n", singleThreadSpeed)

	result := &BenchmarkResult{
		SpeedSamples:      make([]float64, 0),
		DurationSamples:   make([]time.Duration, 0),
		SingleThreadSpeed: singleThreadSpeed,
		ThreadCount:       threads,
	}

	// Create a worker pool with the specified number of threads
	workerPool := NewWorkerPool(threads)

	// Create a stats manager for thread-safe statistics collection
	statsManager := NewStatsManager()

	// Start the worker pool
	workerPool.Start()

	// Start collecting stats
	workerPool.CollectStats(statsManager)

	var attempts int64 = 0
	startTime := time.Now()
	lastStepTime := startTime
	stepAttempts := int64(0)

	// Create a channel to signal benchmark completion
	done := make(chan struct{})

	// We're using a custom progress display for benchmarks, so we don't need additional statistics tracking

	// Start the progress display loop with custom update logic
	go func() {
		ticker := time.NewTicker(ProgressUpdateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("\n🛑 Benchmark cancelled: %v\n", ctx.Err())
				close(done)
				return
			case <-ticker.C:
				now := time.Now()
				currentAttempts := statsManager.GetTotalAttempts()
				newAttempts := currentAttempts - attempts
				attempts = currentAttempts
				stepAttempts += newAttempts

				// Check if we completed a step or reached max attempts
				if stepAttempts >= Step || attempts >= maxAttempts {
					stepDuration := now.Sub(lastStepTime)
					stepSpeed := float64(stepAttempts) / stepDuration.Seconds()

					result.DurationSamples = append(result.DurationSamples, stepDuration)
					result.SpeedSamples = append(result.SpeedSamples, stepSpeed)

					// Display progress
					progress := float64(attempts) / float64(maxAttempts) * 100
					if progress > 100 {
						progress = 100
					}

					// Enhanced progress display with thread metrics
					metrics := statsManager.GetMetrics()
					threadEfficiency := metrics.ThreadEfficiency * 100
					speedupVsSingleThread := metrics.SpeedupVsSingleThread

					fmt.Printf("📊 %s/%s (%.1f%%) | %.0f addr/s | Avg: %.0f addr/s | Speedup: %.2fx | Efficiency: %.1f%%\n",
						formatNumber(attempts),
						formatNumber(maxAttempts),
						progress,
						stepSpeed,
						float64(attempts)/time.Since(startTime).Seconds(),
						speedupVsSingleThread,
						threadEfficiency)

					lastStepTime = now
					stepAttempts = 0

					// Check if we've reached the target attempts
					if attempts >= maxAttempts {
						close(done)
						return
					}
				}

			case <-done:
				return
			}
		}
	}()

	// Run benchmark for specified number of attempts
	batchSize := 1000 // Use larger batch size for benchmark

	// Submit initial work items
	for i := 0; i < workerPool.numWorkers; i++ {
		workerPool.Submit(WorkItem{
			Prefix:     pattern,
			Suffix:     "",
			IsChecksum: isChecksum,
			BatchSize:  batchSize,
		})
	}

	// Ensure we stop when we reach the target attempts
	go func() {
		// Use a timeout to prevent the benchmark from running forever
		timeout := time.After(2 * time.Minute) // 2 minute timeout

		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("\n🛑 Benchmark cancelled: %v\n", ctx.Err())
				close(done)
				return
			case <-ticker.C:
				currentAttempts := statsManager.GetTotalAttempts()

				if currentAttempts >= maxAttempts {
					close(done)
					return
				}

				// Submit more work if needed
				remainingAttempts := maxAttempts - currentAttempts
				if remainingAttempts > 0 && remainingAttempts < int64(workerPool.numWorkers*batchSize) {
					// Adjust batch size for final iterations to avoid overshooting
					smallerBatchSize := int(remainingAttempts / int64(workerPool.numWorkers))
					if smallerBatchSize > 0 {
						for i := 0; i < workerPool.numWorkers; i++ {
							workerPool.Submit(WorkItem{
								Prefix:     pattern,
								Suffix:     "",
								IsChecksum: isChecksum,
								BatchSize:  smallerBatchSize,
							})
						}
					}
				} else if remainingAttempts > 0 {
					// Submit regular work items
					for i := 0; i < workerPool.numWorkers; i++ {
						workerPool.Submit(WorkItem{
							Prefix:     pattern,
							Suffix:     "",
							IsChecksum: isChecksum,
							BatchSize:  batchSize,
						})
					}
				}

			case <-timeout:
				fmt.Println("\n⚠️ Benchmark timed out after 2 minutes")
				fmt.Println("💡 Consider using a simpler pattern or fewer attempts")
				close(done)
				return
			}
		}
	}()

	// Wait for benchmark to complete or timeout
	select {
	case <-done:
		// Benchmark completed
	case <-time.After(5 * time.Minute):
		// Timeout after 5 minutes
		close(done)
	}

	// Shutdown the worker pool
	workerPool.Shutdown()

	// Get final stats
	attempts = statsManager.GetTotalAttempts()
	totalDuration := time.Since(startTime)
	averageSpeed := float64(attempts) / totalDuration.Seconds()

	// Calculate min/max speeds
	if len(result.SpeedSamples) > 0 {
		result.MinSpeed = result.SpeedSamples[0]
		result.MaxSpeed = result.SpeedSamples[0]
		for _, speed := range result.SpeedSamples {
			if speed < result.MinSpeed {
				result.MinSpeed = speed
			}
			if speed > result.MaxSpeed {
				result.MaxSpeed = speed
			}
		}
	}

	result.TotalAttempts = attempts
	result.TotalDuration = totalDuration
	result.AverageSpeed = averageSpeed

	// Calculate scalability metrics
	speedup := averageSpeed / result.SingleThreadSpeed
	idealSpeedup := float64(threads)
	result.ScalabilityEfficiency = speedup / idealSpeedup

	// Calculate Amdahl's Law limit
	// Amdahl's Law: S(n) = 1 / ((1 - p) + p/n)
	// where p is the proportion of parallelizable code and n is the number of threads
	// We can estimate p based on observed speedup
	var p float64 = 0.95 // Assume 95% of code is parallelizable by default
	if threads > 1 && speedup > 1 {
		// Solve for p: p = (speedup * n - n) / (speedup * (n - 1))
		p = (speedup*float64(threads) - float64(threads)) / (speedup * (float64(threads) - 1))
		// Clamp p to reasonable values (0.5 to 0.99)
		if p < 0.5 {
			p = 0.5
		} else if p > 0.99 {
			p = 0.99
		}
	}

	// Calculate theoretical maximum speedup based on Amdahl's Law
	amdahlsLawLimit := 1 / ((1 - p) + p/float64(threads))
	result.AmdahlsLawLimit = amdahlsLawLimit

	// Calculate additional metrics
	result.ThreadBalanceScore = statsManager.GetThreadBalanceScore()
	result.ThreadUtilization = statsManager.GetThreadEfficiency()
	result.SpeedupVsSingleThread = speedup

	// Display final results
	fmt.Printf("\n🏁 Benchmark completed!\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("📈 Total attempts: %s\n", formatNumber(result.TotalAttempts))
	fmt.Printf("⏱️  Total duration: %v\n", result.TotalDuration.Round(time.Millisecond))
	fmt.Printf("⚡ Average speed: %.0f addr/s\n", result.AverageSpeed)

	if len(result.SpeedSamples) > 0 {
		fmt.Printf("📊 Speed range: %.0f - %.0f addr/s\n", result.MinSpeed, result.MaxSpeed)

		// Calculate standard deviation
		var sum float64
		for _, speed := range result.SpeedSamples {
			sum += speed
		}
		mean := sum / float64(len(result.SpeedSamples))

		var variance float64
		for _, speed := range result.SpeedSamples {
			variance += math.Pow(speed-mean, 2)
		}
		stdDev := math.Sqrt(variance / float64(len(result.SpeedSamples)))

		fmt.Printf("📏 Speed std dev: ±%.0f addr/s\n", stdDev)
	}

	// Enhanced multi-thread metrics
	metrics := statsManager.GetMetrics()

	// Thread utilization statistics
	fmt.Printf("\n🧵 Multi-Thread Performance Analysis\n")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")
	fmt.Printf("• Single-thread speed: %.0f addr/s\n", result.SingleThreadSpeed)
	fmt.Printf("• Multi-thread speed: %.0f addr/s\n", averageSpeed)
	fmt.Printf("• Actual speedup: %.2fx\n", speedup)
	fmt.Printf("• Ideal speedup: %.0fx\n", idealSpeedup)
	fmt.Printf("• Scalability efficiency: %.1f%%\n", result.ScalabilityEfficiency*100)
	fmt.Printf("• Thread utilization: %.1f%%\n", metrics.ThreadEfficiency*100)
	fmt.Printf("• Thread balance: %.1f%%\n", metrics.ThreadBalanceScore*100)
	fmt.Printf("• Peak performance: %.0f addr/s\n", statsManager.GetPeakSpeed())

	// Amdahl's Law analysis
	fmt.Printf("\n📐 Scalability Analysis\n")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")
	fmt.Printf("• Parallelizable code estimate: %.1f%%\n", p*100)
	fmt.Printf("• Theoretical max speedup (Amdahl's Law): %.2fx\n", amdahlsLawLimit)
	fmt.Printf("• Performance of theoretical max: %.1f%%\n", (speedup/amdahlsLawLimit)*100)

	// Scalability projection
	fmt.Printf("\n📈 Scalability Projection\n")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")
	fmt.Printf("%-10s | %-15s | %-15s\n", "Threads", "Projected Speed", "Efficiency")
	fmt.Printf("%-10s-+-%-15s-+-%-15s\n",
		"----------", "---------------", "---------------")

	// Project performance for different thread counts
	currentThreads := threads
	for t := 1; t <= currentThreads*2; t *= 2 {
		if t == currentThreads {
			// For current thread count, use actual measured speed
			projectedSpeed := averageSpeed
			efficiency := speedup / float64(t) * 100
			fmt.Printf("%-10d | %-15.0f | %-14.1f%% (actual)\n",
				t, projectedSpeed, efficiency)
		} else {
			// For other thread counts, use Amdahl's Law to project
			projectedSpeedup := 1 / ((1 - p) + p/float64(t))
			projectedSpeed := result.SingleThreadSpeed * projectedSpeedup
			efficiency := projectedSpeedup / float64(t) * 100
			fmt.Printf("%-10d | %-15.0f | %-14.1f%%\n",
				t, projectedSpeed, efficiency)
		}
	}

	// Display detailed thread metrics
	DisplayThreadMetrics(statsManager)

	// Hardware info
	fmt.Printf("\n💻 Platform: Go %s with %d threads\n",
		strings.TrimPrefix(fmt.Sprintf("%s", "go1.21+"), "go"),
		threads)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")

	return result
}
func generateMultipleWallets(prefix, suffix string, count int, isChecksum, showProgress bool) {
	// Use background context for backward compatibility
	ctx := context.Background()
	generateMultipleWalletsWithContext(ctx, prefix, suffix, count, isChecksum, showProgress)
}

func generateMultipleWalletsWithContext(ctx context.Context, prefix, suffix string, count int, isChecksum, showProgress bool) []*WalletResult {
	// Check if TUI should be used and remember this decision
	tuiManager := tui.NewTUIManager()
	useTUI := tuiManager.ShouldUseTUI()

	if useTUI && showProgress {
		// TUI mode
		stats := newStatistics(prefix, suffix, isChecksum)
		statsManager := NewStatsManager()
		walletResultsChan := make(chan WalletResultForTUI, count)
		statsUpdateChan := make(chan StatsUpdateForTUI, 100)

		// Start wallet generation in the background
		generationDone := make(chan []*WalletResult, 1)
		go func() {
			results := generateWalletsInBackground(ctx, prefix, suffix, count, isChecksum, walletResultsChan, statsUpdateChan)
			generationDone <- results
		}()

		// Create and run the TUI
		tuiStats := &tui.Statistics{
			Difficulty:      stats.Difficulty,
			Probability50:   stats.Probability50,
			StartTime:       stats.StartTime,
			Pattern:         stats.Pattern,
			IsChecksum:      isChecksum,
		}
		statsAdapter := &StatsManagerAdapter{statsManager}
		progressModel := tuiManager.CreateProgressModel(tuiStats, statsAdapter)
		program := tea.NewProgram(progressModel, tea.WithAltScreen())

		// Start a goroutine to feed updates to the TUI
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					program.Send(tui.QuitMsg{})
					return
				case statsUpdate := <-statsUpdateChan:
					program.Send(tui.ProgressMsg{
						Attempts:         statsUpdate.Attempts,
						Speed:            statsUpdate.Speed,
						Probability:      statsUpdate.Probability,
						EstimatedTime:    statsUpdate.ETA,
						CompletedWallets: statsUpdate.CompletedWallets,
						TotalWallets:     statsUpdate.TotalWallets,
						ProgressPercent:  statsUpdate.ProgressPercent,
						IsComplete:       (statsUpdate.CompletedWallets >= statsUpdate.TotalWallets),
						Difficulty:       statsUpdate.Difficulty,
					})
				case walletResult := <-walletResultsChan:
					program.Send(tui.WalletResultMsg{Result: tui.WalletResult{
						Index:      walletResult.Index,
						Address:    walletResult.Address,
						PrivateKey: walletResult.PrivateKey,
						Attempts:   walletResult.Attempts,
						Time:       walletResult.Duration,
						Error:      walletResult.Error,
					}})
				}
			}
		}()

		// Run the TUI. This blocks until the TUI exits.
		if model, err := program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			globalCancel() // Cancel everything if TUI fails
		} else if m, ok := model.(tui.ProgressModel); ok && m.Quitting() {
			globalCancel() // Cancel if the user quit
		}

		// Wait for the generation to finish or for the context to be cancelled
		select {
		case results := <-generationDone:
			return results // Generation finished successfully
		case <-ctx.Done():
			return nil // Return nil to indicate cancellation
		}
	}

	// CLI mode: generate wallets directly without TUI
	return generateWalletsDirectly(ctx, prefix, suffix, count, isChecksum, showProgress)
}

// generateWalletsDirectly generates wallets directly for CLI mode
func generateWalletsDirectly(ctx context.Context, prefix, suffix string, count int, isChecksum, showProgress bool) []*WalletResult {
	fmt.Printf("Generating %d bloco wallets with prefix '%s' and suffix '%s'\n", count, prefix, suffix)
	fmt.Printf("Checksum validation: %t\n", isChecksum)
	fmt.Println(strings.Repeat("-", 80))

	startTime := time.Now()
	totalAttempts := 0
	results := make([]*WalletResult, 0, count)

	for i := 0; i < count; i++ {
		// Check for cancellation before each wallet generation
		select {
		case <-ctx.Done():
			fmt.Printf("\n🛑 Operation cancelled after generating %d/%d wallets\n", i, count)
			fmt.Printf("Reason: %v\n", ctx.Err())
			return results
		default:
		}

		fmt.Printf("\nGenerating wallet %d/%d...\n", i+1, count)

		walletStart := time.Now()
		result := generateBlocoWalletWithContext(ctx, prefix, suffix, isChecksum, showProgress)
		walletDuration := time.Since(walletStart)

		if result.Error != "" {
			fmt.Printf("Error generating wallet %d: %s\n", i+1, result.Error)

			// Check if error is due to cancellation
			if ctx.Err() != nil {
				fmt.Printf("🛑 Operation cancelled during wallet %d generation\n", i+1)
				return results
			}
			continue
		}

		totalAttempts += result.Attempts
		results = append(results, result)

		fmt.Printf("✓ Wallet %d generated successfully!\n", i+1)
		fmt.Printf("  Address:     %s\n", result.Wallet.Address)
		fmt.Printf("  Private Key: 0x%s\n", result.Wallet.PrivKey)
		fmt.Printf("  Attempts:    %d\n", result.Attempts)
		fmt.Printf("  Time:        %v\n", walletDuration)
	}

	totalDuration := time.Since(startTime)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Generation completed!\n")
	fmt.Printf("Total wallets: %d\n", len(results))
	fmt.Printf("Total attempts: %d\n", totalAttempts)
	fmt.Printf("Total time: %v\n", totalDuration)
	if len(results) > 0 {
		fmt.Printf("Average attempts per wallet: %.2f\n", float64(totalAttempts)/float64(len(results)))
	}

	return results
}

// generateWalletsInBackground generates wallets in background for TUI mode
func generateWalletsInBackground(ctx context.Context, prefix, suffix string, count int, isChecksum bool, walletResultsChan chan<- WalletResultForTUI, statsUpdateChan chan<- StatsUpdateForTUI) []*WalletResult {
	difficulty := computeDifficulty(prefix, suffix, isChecksum)
	results := make([]*WalletResult, 0, count)
	
	// Track statistics for updates with mutex for thread safety
	startTime := time.Now()
	var totalAttempts int64
	var completedWallets int
	var statsMutex sync.Mutex
	
	// Create stats for difficulty calculation
	stats := newStatistics(prefix, suffix, isChecksum)
	
	// Create completion signal channel to coordinate with TUI
	completionSignal := make(chan struct{})
	defer close(completionSignal)
	
	// Function to send stats update
	sendStatsUpdate := func() {
		if statsUpdateChan != nil {
			statsMutex.Lock()
			currentAttempts := totalAttempts
			currentCompleted := completedWallets
			statsMutex.Unlock()
			
			elapsed := time.Since(startTime)
			speed := float64(currentAttempts) / elapsed.Seconds()
			
			// Calculate progress as percentage of wallets completed
			progressPercent := (float64(currentCompleted) / float64(count)) * 100.0
			
			// Calculate probability differently: how far we are in the expected attempts
			probability := progressPercent // Progress is the real probability of completion
			
			// Calculate ETA based on remaining wallets and current speed
			remaining := count - currentCompleted
			var eta time.Duration
			if remaining > 0 && speed > 0 {
				// Estimate remaining attempts based on average so far
				var avgAttemptsPerWallet float64
				if currentCompleted > 0 {
					avgAttemptsPerWallet = float64(currentAttempts) / float64(currentCompleted)
				} else {
					avgAttemptsPerWallet = float64(stats.Probability50)
				}
				
				estimatedRemainingAttempts := float64(remaining) * avgAttemptsPerWallet
				eta = time.Duration(estimatedRemainingAttempts/speed) * time.Second
			}
			
			select {
			case statsUpdateChan <- StatsUpdateForTUI{
						Attempts:         totalAttempts,
						Speed:            speed,
						Probability:      probability,
						ETA:              eta,
						CompletedWallets: len(results),
						TotalWallets:     count,
						ProgressPercent:  float64(len(results)) / float64(count) * 100,
						Difficulty:       difficulty,
					}:
			case <-ctx.Done():
				return
			}
		}
	}
	
	// Send periodic stats updates
	statsUpdateTicker := time.NewTicker(500 * time.Millisecond)
	defer statsUpdateTicker.Stop()
	
	// Stats update goroutine
	statsUpdateDone := make(chan struct{})
	go func() {
		defer close(statsUpdateDone)
		for {
			select {
			case <-ctx.Done():
				return
			case <-completionSignal:
				// Send final stats update before stopping
				sendStatsUpdate()
				return
			case <-statsUpdateTicker.C:
				sendStatsUpdate()
			}
		}
	}()

	for i := 0; i < count; i++ {
		// Check for cancellation before each wallet generation
		select {
		case <-ctx.Done():
			// Send cancellation to TUI if needed
			if walletResultsChan != nil {
				select {
				case walletResultsChan <- WalletResultForTUI{
					Index: i + 1,
					Error: ctx.Err().Error(),
				}:
				default:
				}
			}
			// Wait for stats updates to finish before closing channels
			<-statsUpdateDone
			if walletResultsChan != nil {
				close(walletResultsChan)
			}
			if statsUpdateChan != nil {
				close(statsUpdateChan)
			}
			return results
		default:
		}

		walletStart := time.Now()
		result := generateBlocoWalletWithContext(ctx, prefix, suffix, isChecksum, false) // No progress for individual wallets
		walletDuration := time.Since(walletStart)

		if result.Error != "" {
			// Send error result to TUI
			if walletResultsChan != nil {
				select {
				case walletResultsChan <- WalletResultForTUI{
					Index:    i + 1,
					Error:    result.Error,
					Duration: walletDuration,
				}:
				case <-ctx.Done():
				}
			}

			// Check if error is due to cancellation
			if ctx.Err() != nil {
				// Wait for stats updates to finish before closing channels
				<-statsUpdateDone
				if walletResultsChan != nil {
					close(walletResultsChan)
				}
				if statsUpdateChan != nil {
					close(statsUpdateChan)
				}
				return results
			}
			continue
		}

		results = append(results, result)
		
		// Update statistics tracking
		statsMutex.Lock()
		totalAttempts += int64(result.Attempts)
		completedWallets++
		statsMutex.Unlock()

		// Send successful wallet result to TUI
		if walletResultsChan != nil {
			select {
			case walletResultsChan <- WalletResultForTUI{
				Index:      i + 1,
				Address:    result.Wallet.Address,
				PrivateKey: "0x" + result.Wallet.PrivKey,
				Attempts:   result.Attempts,
				Duration:   walletDuration,
				Error:      "",
			}:
			case <-ctx.Done():
			}
		}
	}

	// Generation completed - wait for final stats updates to be sent
	<-statsUpdateDone
	
	// Send final stats update to ensure 100% completion is shown
	sendStatsUpdate()
	
	// Give TUI time to process final updates before closing channels
	finalUpdateTimer := time.NewTimer(2 * time.Second)
	defer finalUpdateTimer.Stop()
	
	// Send final completion notification to TUI
	if statsUpdateChan != nil {
		select {
		case statsUpdateChan <- StatsUpdateForTUI{
			Attempts:         totalAttempts,
			Speed:            float64(totalAttempts) / time.Since(startTime).Seconds(),
			Probability:      100.0, // 100% completed
			ETA:              0,     // No remaining time
			CompletedWallets: completedWallets,
			TotalWallets:     count,
			ProgressPercent:  100.0, // 100% progress
		}:
		case <-finalUpdateTimer.C:
		}
	}
	
	// Wait a bit more to ensure TUI processes the final update
	select {
	case <-finalUpdateTimer.C:
	case <-ctx.Done():
	}

	// Close the channels to signal completion
	if walletResultsChan != nil {
		close(walletResultsChan)
	}
	if statsUpdateChan != nil {
		close(statsUpdateChan)
	}

	return results
}

// displayMultipleWalletsTUI shows a TUI interface for multiple wallet generation
func displayMultipleWalletsTUI(prefix, suffix string, count int, isChecksum bool, walletResultsChan chan WalletResultForTUI, statsUpdateChan chan StatsUpdateForTUI) {
	// Create TUI statistics interface with initial values for the pattern
	stats := newStatistics(prefix, suffix, isChecksum)
	tuiStats := &tui.Statistics{
		Difficulty:      stats.Difficulty,
		Probability50:   stats.Probability50,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         stats.Pattern,
		IsChecksum:      stats.IsChecksum,
	}

	// Create TUI progress model
	tuiManager := tui.NewTUIManager()
	progressModel := tuiManager.CreateProgressModel(tuiStats, nil)

	// Create TUI program
	program := tea.NewProgram(progressModel, tea.WithAltScreen())

	// Start a goroutine to handle both wallet results and stats updates
	go func() {
		walletResultsActive := true
		statsUpdatesActive := true
		generationCompleted := false
		finalDisplayTimer := time.NewTimer(5 * time.Second) // Extended display time for final stats
		finalDisplayTimer.Stop() // Don't start until generation is complete
		
		for walletResultsActive || statsUpdatesActive || !generationCompleted {
			select {
			case result, ok := <-walletResultsChan:
				if !ok {
					walletResultsActive = false
					if !statsUpdatesActive {
						// Both channels closed, start final display timer
						generationCompleted = true
						finalDisplayTimer.Reset(5 * time.Second)
					}
					continue
				}
				// Send wallet result to TUI
				program.Send(tui.WalletResultMsg{
					Result: tui.WalletResult{
						Index:      result.Index,
						Address:    result.Address,
						PrivateKey: result.PrivateKey,
						Attempts:   result.Attempts,
						Time:       result.Duration,
						Error:      result.Error,
					},
				})
			
			case statsUpdate, ok := <-statsUpdateChan:
				if !ok {
					statsUpdatesActive = false
					if !walletResultsActive {
						// Both channels closed, start final display timer
						generationCompleted = true
						finalDisplayTimer.Reset(5 * time.Second)
					}
					continue
				}
				// Send statistics update to TUI
				program.Send(tui.ProgressMsg{
					Attempts:         statsUpdate.Attempts,
					Speed:            statsUpdate.Speed,
					Probability:      statsUpdate.Probability,
					EstimatedTime:    statsUpdate.ETA,
					Difficulty:       tuiStats.Difficulty, // Keep original difficulty
					Pattern:          tuiStats.Pattern,    // Keep original pattern
					CompletedWallets: statsUpdate.CompletedWallets,
					TotalWallets:     statsUpdate.TotalWallets,
					ProgressPercent:  statsUpdate.ProgressPercent,
					IsComplete:       statsUpdate.ProgressPercent >= 100.0 && statsUpdate.CompletedWallets >= count,
				})
				
				// Check if generation is complete (100% progress)
				if statsUpdate.ProgressPercent >= 100.0 && statsUpdate.CompletedWallets >= count {
					if !generationCompleted {
						generationCompleted = true
						finalDisplayTimer.Reset(5 * time.Second)
					}
				}
			
			case <-finalDisplayTimer.C:
				// Final display period is over
				if generationCompleted {
					return
				}
			}
		}
		
		// If we exit the loop without timer expiring, wait for final display
		if generationCompleted {
			select {
			case <-finalDisplayTimer.C:
			case <-time.After(5 * time.Second): // Fallback timeout
			}
		}
	}()

	// Run the TUI program
	program.Run()
}

var (
	prefix       string
	suffix       string
	count        int
	checksum     bool
	showProgress bool
	threads      int
	// Benchmark specific flags
	benchmarkAttempts int64
	benchmarkPattern  string
	compareThreads    bool
)

// Benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run performance benchmark to test address generation speed",
	Long: `Run a comprehensive performance benchmark to test address generation speed
and analyze system performance characteristics.

This command generates addresses continuously and measures detailed performance metrics
including average speed, speed distribution, thread scalability, and system efficiency.
It supports multi-threaded execution with automatic thread optimization and provides
comprehensive scalability analysis.

Benchmark Metrics:
  • Average generation speed (addresses/second)
  • Speed distribution (min, max, median)
  • Thread scalability and efficiency
  • Memory usage and optimization
  • System resource utilization

Performance Analysis:
  • Single-thread baseline measurement
  • Multi-thread scalability testing
  • Amdahl's Law efficiency calculation
  • Thread balance and utilization metrics`,
	Example: `  # Basic benchmark with default pattern 'abc'
  bloco-eth benchmark

  # Benchmark with specific number of attempts
  bloco-eth benchmark --attempts 50000

  # Benchmark with custom pattern
  bloco-eth benchmark --pattern deadbeef --attempts 25000

  # Benchmark with checksum validation (more CPU intensive)
  bloco-eth benchmark --pattern AbCdEf --checksum --attempts 10000

  # Benchmark with specific thread count
  bloco-eth benchmark --threads 8 --attempts 20000

  # Compare performance across different thread counts
  bloco-eth benchmark --compare-threads --attempts 15000

  # Intensive benchmark for performance analysis
  bloco-eth benchmark --pattern cafe --attempts 100000 --compare-threads

  # Quick benchmark for development testing
  bloco-eth benchmark --attempts 5000 --pattern abc`,
	Run: func(cmd *cobra.Command, args []string) {
		if benchmarkPattern == "" {
			// Use default pattern if none specified
			benchmarkPattern = "abc" // 3 hex chars for reasonable benchmark
		}

		// Validate hex characters
		if !isValidHex(benchmarkPattern) {
			fmt.Println("❌ Error: Benchmark pattern contains invalid hex characters")
			os.Exit(1)
		}

		// Validate and set thread count
		validateThreads()

		// Validate thread count doesn't exceed reasonable limits
		maxThreads := detectCPUCount() * 2 // Allow up to 2x CPU cores
		if threads > maxThreads {
			fmt.Printf("⚠️  Warning: Using %d threads on %d CPU cores may not be optimal\n", threads, detectCPUCount())
		}

		// Run the benchmark with the specified parameters
		result := runBenchmark(benchmarkAttempts, benchmarkPattern, checksum)

		// If compare-threads flag is set, run additional benchmarks with different thread counts
		if compareThreads {
			fmt.Printf("\n🔍 Running thread comparison benchmarks...\n")

			// Save original thread count
			originalThreads := threads

			// Define thread counts to compare (1, 2, 4, 8, etc. up to original thread count)
			threadCounts := []int{1}
			for t := 2; t <= originalThreads; t *= 2 {
				if t <= originalThreads {
					threadCounts = append(threadCounts, t)
				}
			}

			// Make sure the original thread count is included if not already
			if threadCounts[len(threadCounts)-1] != originalThreads {
				threadCounts = append(threadCounts, originalThreads)
			}

			// Run benchmarks with different thread counts
			results := make(map[int]*BenchmarkResult)
			results[originalThreads] = result // Store the original result

			// Run benchmarks for other thread counts
			for _, t := range threadCounts {
				if t == originalThreads {
					continue // Skip the original thread count as we already have it
				}

				fmt.Printf("\n📊 Benchmark with %d threads:\n", t)
				threads = t
				results[t] = runBenchmark(benchmarkAttempts/2, benchmarkPattern, checksum) // Use fewer attempts for comparison
			}

			// Display comparison results
			fmt.Printf("\n🧵 Thread Scaling Comparison\n")
			fmt.Printf("═══════════════════════════════════════════════════════════════\n")
			fmt.Printf("%-10s | %-15s | %-15s | %-15s | %-15s\n",
				"Threads", "Speed (addr/s)", "Speedup", "Efficiency", "Balance")
			fmt.Printf("%-10s-+-%-15s-+-%-15s-+-%-15s-+-%-15s\n",
				"----------", "---------------", "---------------", "---------------", "---------------")

			// Use single thread as baseline
			baseSpeed := results[1].AverageSpeed

			// Print results in order of thread count
			for _, t := range threadCounts {
				r := results[t]
				speedup := r.AverageSpeed / baseSpeed
				efficiency := speedup / float64(t) * 100

				fmt.Printf("%-10d | %-15.0f | %-15.2fx | %-14.1f%% | %-14.1f%%\n",
					t, r.AverageSpeed, speedup, efficiency, r.ThreadBalanceScore*100)
			}

			// Calculate and display scalability metrics
			fmt.Printf("\n📈 Scalability Analysis\n")
			fmt.Printf("───────────────────────────────────────────────────────────────\n")

			// Calculate scalability coefficient using Amdahl's Law
			// S(n) = 1 / ((1 - p) + p/n)
			// We can estimate p by fitting the curve to our measurements

			// Use linear regression to estimate parallelizable portion (p)
			var sumX, sumY, sumXY, sumX2 float64
			var n float64

			for _, t := range threadCounts {
				if t == 1 {
					continue // Skip single thread as it doesn't provide scaling info
				}

				x := 1.0 / float64(t)
				y := 1.0 / (results[t].AverageSpeed / baseSpeed)

				sumX += x
				sumY += y
				sumXY += x * y
				sumX2 += x * x
				n++
			}

			// Calculate regression coefficients
			var p float64
			if n > 1 {
				slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
				intercept := (sumY - slope*sumX) / n
				p = slope / (intercept + slope)

				// Clamp p to reasonable values (0.5 to 0.99)
				if p < 0.5 {
					p = 0.5
				} else if p > 0.99 {
					p = 0.99
				}

				fmt.Printf("• Parallelizable code estimate: %.1f%%\n", p*100)
				fmt.Printf("• Sequential code estimate: %.1f%%\n", (1-p)*100)

				// Calculate theoretical maximum speedup with infinite cores
				maxSpeedup := 1.0 / (1.0 - p)
				fmt.Printf("• Theoretical maximum speedup (infinite cores): %.2fx\n", maxSpeedup)

				// Calculate optimal thread count based on efficiency target
				// Efficiency = S(n)/n = 1/(n*((1-p) + p/n))
				// For 90% efficiency: n = 9*p/(1-p)
				optimalThreads90 := 9.0 * p / (1.0 - p)
				fmt.Printf("• Optimal thread count for 90%% efficiency: %.0f threads\n", math.Ceil(optimalThreads90))

				// Project performance at higher thread counts
				fmt.Printf("\n📊 Performance Projection\n")
				fmt.Printf("%-10s | %-15s | %-15s | %-15s\n",
					"Threads", "Projected Speed", "Speedup", "Efficiency")
				fmt.Printf("%-10s-+-%-15s-+-%-15s-+-%-15s\n",
					"----------", "---------------", "---------------", "---------------")

				// Show projections for 2x, 4x, and 8x the current thread count
				maxProjection := originalThreads * 8
				if maxProjection > 128 {
					maxProjection = 128 // Cap at reasonable value
				}

				for t := originalThreads * 2; t <= maxProjection; t *= 2 {
					projectedSpeedup := 1.0 / ((1.0 - p) + p/float64(t))
					projectedSpeed := baseSpeed * projectedSpeedup
					efficiency := projectedSpeedup / float64(t) * 100

					fmt.Printf("%-10d | %-15.0f | %-15.2fx | %-14.1f%%\n",
						t, projectedSpeed, projectedSpeedup, efficiency)
				}
			} else {
				fmt.Printf("• Insufficient data points to calculate scalability metrics\n")
			}

			fmt.Printf("═══════════════════════════════════════════════════════════════\n")

			// Restore original thread count
			threads = originalThreads
		}
	},
}

// Statistics command
// showStatisticsTUI displays statistics using TUI interface
func showStatisticsTUI(stats *Statistics) {
	// Create TUI statistics interface
	tuiStats := &tui.Statistics{
		Difficulty:      stats.Difficulty,
		Probability50:   stats.Probability50,
		CurrentAttempts: 0, // For stats display, this is not relevant
		Speed:           0, // For stats display, this is not relevant
		Probability:     0, // For stats display, this is not relevant
		EstimatedTime:   0, // For stats display, this is not relevant
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         stats.Pattern,
		IsChecksum:      stats.IsChecksum,
	}

	// Create TUI stats model
	tuiManager := tui.NewTUIManager()
	statsModel := tuiManager.CreateStatsModel(tuiStats)

	// Create TUI program
	program := tea.NewProgram(statsModel, tea.WithAltScreen())

	// Run the TUI program
	if _, err := program.Run(); err != nil {
		// If TUI fails, fallback to text mode
		fmt.Printf("TUI failed: %v, falling back to text mode\n", err)
		displayStatisticsText(stats)
	}
}

// displayStatisticsText displays statistics in text mode (fallback)
func displayStatisticsText(stats *Statistics) {
	prefix := ""
	suffix := ""
	
	// Extract prefix and suffix from pattern
	if len(stats.Pattern) > 0 {
		// For simplicity, assume pattern doesn't contain mixed prefix/suffix
		prefix = stats.Pattern
	}

	fmt.Printf("📊 Bloco Address Difficulty Analysis\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("🎯 Pattern: %s%s%s\n", prefix, strings.Repeat("*", 40-len(prefix)-len(suffix)), suffix)
	fmt.Printf("🔧 Checksum: %t\n", stats.IsChecksum)
	fmt.Printf("📏 Pattern length: %d characters\n\n", len(prefix+suffix))

	fmt.Printf("📈 Difficulty Metrics:\n")
	fmt.Printf("   • Base difficulty: %s\n", formatNumber(int64(math.Pow(16, float64(len(prefix+suffix))))))
	fmt.Printf("   • Total difficulty: %s\n", formatNumber(int64(stats.Difficulty)))

	if stats.Probability50 > 0 {
		fmt.Printf("   • 50%% probability: %s attempts\n", formatNumber(stats.Probability50))
	} else {
		fmt.Printf("   • 50%% probability: Nearly impossible\n")
	}

	fmt.Printf("\n⏱️  Time Estimates (at different speeds):\n")
	speeds := []float64{1000, 10000, 50000, 100000}
	for _, speed := range speeds {
		if stats.Probability50 > 0 {
			timeFor50 := time.Duration(float64(stats.Probability50)/speed) * time.Second
			fmt.Printf("   • %s addr/s: %s\n", formatNumber(int64(speed)), formatDuration(timeFor50))
		} else {
			fmt.Printf("   • %s addr/s: Nearly impossible\n", formatNumber(int64(speed)))
		}
	}

	fmt.Printf("\n🎲 Probability Examples:\n")
	attemptExamples := []int64{1000, 10000, 100000, 1000000}
	for _, attempts := range attemptExamples {
		if stats.Difficulty > 0 {
			prob := computeProbability(stats.Difficulty, attempts) * 100
			fmt.Printf("   • After %s attempts: %.4f%%\n", formatNumber(attempts), prob)
		}
	}

	fmt.Printf("\n💡 Recommendations:\n")
	if len(prefix+suffix) <= 3 {
		fmt.Printf("   • ✅ Easy - Should generate quickly\n")
	} else if len(prefix+suffix) <= 5 {
		fmt.Printf("   • ⚠️  Moderate - May take some time\n")
	} else if len(prefix+suffix) <= 7 {
		fmt.Printf("   • 🔥 Hard - Will take considerable time\n")
	} else {
		fmt.Printf("   • 💀 Extremely Hard - May take days/weeks/years\n")
	}

	if stats.IsChecksum {
		fmt.Printf("   • 📝 Checksum enabled - Difficulty increased\n")
	}

	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Analyze difficulty statistics for address patterns",
	Long: `Display comprehensive difficulty analysis and statistics for generating
bloco addresses with the specified prefix and suffix patterns.

This command provides detailed mathematical analysis including difficulty calculations,
probability distributions, time estimates at various speeds, and practical recommendations
for pattern generation. Use this before attempting to generate difficult patterns to
understand the computational requirements.

Statistical Analysis:
  • Base and total difficulty calculations
  • Probability analysis for different attempt counts
  • Time estimates at various generation speeds
  • Pattern complexity assessment and recommendations
  • Checksum impact on difficulty (when enabled)

Difficulty Levels:
  • Easy (1-3 chars): Generates quickly, suitable for testing
  • Moderate (4-5 chars): May take some time, reasonable for production
  • Hard (6-7 chars): Considerable time required, plan accordingly
  • Extreme (8+ chars): May take days/weeks/years, use with caution`,
	Example: `  # Analyze difficulty for prefix pattern
  bloco-eth stats --prefix abc

  # Analyze combined prefix and suffix pattern
  bloco-eth stats --prefix dead --suffix beef

  # Analyze with checksum validation (increases difficulty)
  bloco-eth stats --prefix DeAdBeEf --checksum

  # Check difficulty for suffix-only pattern
  bloco-eth stats --suffix cafe

  # Analyze complex pattern before generation
  bloco-eth stats --prefix 1337 --suffix c0de --checksum

  # Quick difficulty check for development
  bloco-eth stats --prefix ab --suffix cd

  # Analyze very difficult pattern (use with caution)
  bloco-eth stats --prefix abcdef --suffix 123456

  # Compare checksum vs non-checksum difficulty
  bloco-eth stats --prefix AbCd --checksum`,
	Run: func(cmd *cobra.Command, args []string) {
		if prefix == "" && suffix == "" {
			fmt.Println("❌ Error: At least one of --prefix or --suffix must be specified")
			os.Exit(1)
		}

		// Validate hex characters
		if !isValidHex(prefix) || !isValidHex(suffix) {
			fmt.Println("❌ Error: Invalid hex characters in prefix or suffix")
			os.Exit(1)
		}

		stats := newStatistics(prefix, suffix, checksum)
		
		// Check if TUI should be used
		tuiManager := tui.NewTUIManager()
		if tuiManager.ShouldUseTUI() {
			showStatisticsTUI(stats)
		} else {
			displayStatisticsText(stats)
		}
	},
}

var rootCmd = &cobra.Command{
	Use:   "bloco-eth",
	Short: "Generate Ethereum bloco wallets with custom prefix and suffix",
	Long: `A high-performance CLI tool for generating Ethereum bloco wallets with custom 
prefix and suffix patterns.

This tool generates Ethereum wallets where the address starts with a specific prefix
and/or ends with a specific suffix. It supports EIP-55 checksum validation for more
secure bloco addresses and provides detailed statistics and progress information.

Features:
  • Multi-threaded parallel processing for optimal performance
  • Real-time progress tracking with speed metrics
  • EIP-55 checksum validation support
  • Difficulty analysis and time estimation
  • Cross-platform support (Linux, Windows, macOS)
  • Comprehensive benchmarking and statistics

Pattern Format:
  • Prefix: hex characters that the address must start with
  • Suffix: hex characters that the address must end with
  • Valid hex: 0-9, a-f, A-F (case matters for checksum validation)
  • Maximum combined length: 40 characters (full address length)`,
	Example: `  # Generate a single wallet with prefix 'abc'
  bloco-eth --prefix abc

  # Generate 5 wallets with prefix 'dead' and suffix 'beef'
  bloco-eth --prefix dead --suffix beef --count 5

  # Generate with checksum validation (case-sensitive)
  bloco-eth --prefix DeAdBeEf --checksum --count 1

  # Show progress for long-running generation
  bloco-eth --prefix abcdef --progress

  # Use specific number of threads
  bloco-eth --prefix abc --threads 8

  # Generate multiple wallets with progress tracking
  bloco-eth --prefix cafe --suffix babe --count 10 --progress

  # Complex pattern with checksum
  bloco-eth --prefix 1337 --suffix c0de --checksum --progress`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate inputs
		if prefix == "" && suffix == "" {
			fmt.Println("❌ Error: At least one of --prefix or --suffix must be specified")
			fmt.Println("💡 Use --help for usage examples")
			os.Exit(1)
		}

		if count <= 0 {
			fmt.Println("❌ Error: Count must be greater than 0")
			os.Exit(1)
		}

		// Validate and set thread count
		validateThreads()

		// Validate hex characters for prefix and suffix
		if !isValidHex(prefix) || !isValidHex(suffix) {
			fmt.Println("❌ Error: Invalid hex characters in prefix or suffix")
			fmt.Println("💡 Only use hex characters: 0-9, a-f, A-F")
			os.Exit(1)
		}

		// Check if prefix + suffix length doesn't exceed address length
		if len(prefix)+len(suffix) > 40 {
			fmt.Println("❌ Error: Combined length of prefix and suffix cannot exceed 40 characters")
			fmt.Printf("💡 Current length: %d characters (prefix: %d, suffix: %d)\n",
				len(prefix)+len(suffix), len(prefix), len(suffix))
			os.Exit(1)
		}

		// Show warning for difficult patterns
		difficulty := computeDifficulty(prefix, suffix, checksum)
		if difficulty > 1000000 && !showProgress {
			fmt.Println("⚠️  Warning: This pattern may take a long time to generate.")
			fmt.Println("💡 Consider using --progress flag to monitor generation progress")
			fmt.Println("💡 Use 'bloco-eth stats --prefix", prefix, "--suffix", suffix, "' to see difficulty analysis")
			fmt.Println()
		}

		// Start wallet generation
		generateMultipleWallets(prefix, suffix, count, checksum, showProgress)
	},
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(statsCmd)

	// Define command line flags for root command
	rootCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "Prefix for the bloco address (hex characters only)")
	rootCmd.Flags().StringVarP(&suffix, "suffix", "s", "", "Suffix for the bloco address (hex characters only)")
	rootCmd.Flags().IntVarP(&count, "count", "c", 1, "Number of bloco wallets to generate")
	rootCmd.Flags().BoolVar(&checksum, "checksum", false, "Enable EIP-55 checksum validation (case-sensitive)")
	rootCmd.Flags().BoolVar(&showProgress, "progress", false, "Show detailed progress during generation")
	rootCmd.Flags().IntVarP(&threads, "threads", "t", 0, "Number of threads to use (0 = auto-detect, recommended max is 2x CPU cores)")

	// Define flags for benchmark command
	benchmarkCmd.Flags().Int64VarP(&benchmarkAttempts, "attempts", "a", 10000, "Number of attempts for benchmark")
	benchmarkCmd.Flags().StringVarP(&benchmarkPattern, "pattern", "p", "", "Pattern to use for benchmark (default: 'abc')")
	benchmarkCmd.Flags().BoolVar(&checksum, "checksum", false, "Enable checksum validation for benchmark")
	benchmarkCmd.Flags().BoolVar(&compareThreads, "compare-threads", false, "Run additional benchmarks with different thread counts for comparison")
	benchmarkCmd.Flags().IntVarP(&threads, "threads", "t", 0, "Number of threads to use (0 = auto-detect, recommended max is 2x CPU cores)")

	// Define flags for stats command
	statsCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "Prefix for difficulty analysis")
	statsCmd.Flags().StringVarP(&suffix, "suffix", "s", "", "Suffix for difficulty analysis")
	statsCmd.Flags().BoolVar(&checksum, "checksum", false, "Enable checksum validation for analysis")
}

func main() {
	// Initialize object pools for performance optimization
	initializePools()
	
	// Setup global signal handling
	setupSignalHandling()

	if err := fang.Execute(
		globalCtx,
		rootCmd,
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		os.Exit(1)
	}
}
