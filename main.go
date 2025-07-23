package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
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
	TotalAttempts   int64
	TotalDuration   time.Duration
	AverageSpeed    float64
	MinSpeed        float64
	MaxSpeed        float64
	SpeedSamples    []float64
	DurationSamples []time.Duration
}

const (
	// Step defines how many attempts before showing progress
	Step = 500
	// ProgressUpdateInterval defines how often to update progress display
	ProgressUpdateInterval = time.Millisecond * 500
)

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
func privateToAddress(privateKey []byte) string {
	// Convert private key bytes to ECDSA private key
	privateKeyInt := new(big.Int).SetBytes(privateKey)
	privateKeyECDSA := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: crypto.S256(),
		},
		D: privateKeyInt,
	}

	// Calculate public key coordinates
	privateKeyECDSA.PublicKey.X, privateKeyECDSA.PublicKey.Y = crypto.S256().ScalarBaseMult(privateKey)

	// Get uncompressed public key bytes (without 0x04 prefix)
	publicKeyBytes := crypto.FromECDSAPub(&privateKeyECDSA.PublicKey)[1:]

	// Calculate Keccak256 hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(publicKeyBytes)
	hash := hasher.Sum(nil)

	// Take the last 20 bytes as the address
	address := hash[len(hash)-20:]
	return hex.EncodeToString(address)
}

// getRandomWallet generates a random wallet with private key and address
func getRandomWallet() (*Wallet, error) {
	// Generate 32 random bytes for private key
	privateKey := make([]byte, 32)
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
	// Calculate Keccak256 hash of the address
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(address))
	hash := hex.EncodeToString(hasher.Sum(nil))

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

// isValidBlocoAddress checks if a wallet address matches the given constraints
func isValidBlocoAddress(address, prefix, suffix string, isChecksum bool) bool {
	if len(address) != 40 {
		return false
	}

	// Extract prefix and suffix from address
	addressPrefix := address[:len(prefix)]
	addressSuffix := address[40-len(suffix):]

	if !isChecksum {
		return strings.EqualFold(prefix, addressPrefix) && strings.EqualFold(suffix, addressSuffix)
	}

	// For checksum validation, first check if lowercase versions match
	if !strings.EqualFold(prefix, addressPrefix) || !strings.EqualFold(suffix, addressSuffix) {
		return false
	}

	return isValidChecksum(address, prefix, suffix)
}

// toChecksumAddress converts an address to checksum format
func toChecksumAddress(address string) string {
	// Calculate Keccak256 hash of the address
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(address))
	hash := hex.EncodeToString(hasher.Sum(nil))

	result := ""
	for i, char := range address {
		hashChar, _ := strconv.ParseInt(string(hash[i]), 16, 64)
		if hashChar >= 8 {
			result += strings.ToUpper(string(char))
		} else {
			result += string(char)
		}
	}
	return result
}

// generateBlocoWallet generates a wallet that matches the given constraints with statistics
func generateBlocoWallet(prefix, suffix string, isChecksum bool, showProgress bool) *WalletResult {
	attempts := int64(0)
	pre := prefix
	suf := suffix

	if !isChecksum {
		pre = strings.ToLower(prefix)
		suf = strings.ToLower(suffix)
	}

	// Initialize statistics
	var stats *Statistics
	if showProgress {
		stats = newStatistics(prefix, suffix, isChecksum)
		fmt.Printf("\n🎯 Generating bloco wallet with pattern: %s%s%s\n", prefix, strings.Repeat("*", 40-len(prefix)-len(suffix)), suffix)
		fmt.Printf("📊 Difficulty: %s | 50%% probability: %s attempts\n\n",
			formatNumber(int64(stats.Difficulty)),
			formatNumber(stats.Probability50))
	}

	lastProgressUpdate := time.Now()

	for {
		wallet, err := getRandomWallet()
		if err != nil {
			return &WalletResult{
				Error:    err.Error(),
				Attempts: int(attempts),
			}
		}

		attempts++

		// Update progress display
		if showProgress && time.Since(lastProgressUpdate) >= ProgressUpdateInterval {
			stats.update(attempts)
			stats.displayProgress()
			lastProgressUpdate = time.Now()
		}

		if isValidBlocoAddress(wallet.Address, pre, suf, isChecksum) {
			// Final progress update
			if showProgress {
				stats.update(attempts)
				stats.displayProgress()
				fmt.Printf("\n✅ Success! Found matching address in %s attempts\n\n", formatNumber(attempts))
			}

			checksumAddress := "0x" + toChecksumAddress(wallet.Address)
			return &WalletResult{
				Wallet: &Wallet{
					Address: checksumAddress,
					PrivKey: wallet.PrivKey,
				},
				Attempts: int(attempts),
			}
		}
	}
}

// runBenchmark runs a performance benchmark
func runBenchmark(maxAttempts int64, pattern string, isChecksum bool) *BenchmarkResult {
	if maxAttempts == 0 {
		maxAttempts = 10000
	}

	fmt.Printf("🚀 Starting benchmark with pattern '%s' (checksum: %t)\n", pattern, isChecksum)
	fmt.Printf("📈 Target: %s attempts | Step size: %d\n\n", formatNumber(maxAttempts), Step)

	result := &BenchmarkResult{
		SpeedSamples:    make([]float64, 0),
		DurationSamples: make([]time.Duration, 0),
	}

	var attempts int64 = 0

	startTime := time.Now()
	lastStepTime := startTime
	stepAttempts := int64(0)

	for attempts < maxAttempts {
		_, err := getRandomWallet()
		if err != nil {
			fmt.Printf("❌ Error during benchmark: %v\n", err)
			break
		}

		attempts++
		stepAttempts++

		// Check if we completed a step
		if stepAttempts >= Step {
			now := time.Now()
			stepDuration := now.Sub(lastStepTime)
			stepSpeed := float64(stepAttempts) / stepDuration.Seconds()

			result.DurationSamples = append(result.DurationSamples, stepDuration)
			result.SpeedSamples = append(result.SpeedSamples, stepSpeed)

			// Display progress
			progress := float64(attempts) / float64(maxAttempts) * 100
			fmt.Printf("📊 %s/%s (%.1f%%) | %.0f addr/s | Avg: %.0f addr/s\n",
				formatNumber(attempts),
				formatNumber(maxAttempts),
				progress,
				stepSpeed,
				float64(attempts)/time.Since(startTime).Seconds())

			lastStepTime = now
			stepAttempts = 0
		}
	}

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

	// Hardware info
	fmt.Printf("💻 Platform: Go %s\n", strings.TrimPrefix(fmt.Sprintf("%s", "go1.21+"), "go"))
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")

	return result
}
func generateMultipleWallets(prefix, suffix string, count int, isChecksum, showProgress bool) {
	fmt.Printf("Generating %d bloco wallets with prefix '%s' and suffix '%s'\n", count, prefix, suffix)
	fmt.Printf("Checksum validation: %t\n", isChecksum)
	fmt.Println(strings.Repeat("-", 80))

	startTime := time.Now()
	totalAttempts := 0

	for i := 0; i < count; i++ {
		fmt.Printf("\nGenerating wallet %d/%d...\n", i+1, count)

		walletStart := time.Now()
		result := generateBlocoWallet(prefix, suffix, isChecksum, showProgress)
		walletDuration := time.Since(walletStart)

		if result.Error != "" {
			fmt.Printf("Error generating wallet %d: %s\n", i+1, result.Error)
			continue
		}

		totalAttempts += result.Attempts

		fmt.Printf("✓ Wallet %d generated successfully!\n", i+1)
		fmt.Printf("  Address:     %s\n", result.Wallet.Address)
		fmt.Printf("  Private Key: 0x%s\n", result.Wallet.PrivKey)
		fmt.Printf("  Attempts:    %d\n", result.Attempts)
		fmt.Printf("  Time:        %v\n", walletDuration)
	}

	totalDuration := time.Since(startTime)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Generation completed!\n")
	fmt.Printf("Total wallets: %d\n", count)
	fmt.Printf("Total attempts: %d\n", totalAttempts)
	fmt.Printf("Total time: %v\n", totalDuration)
	fmt.Printf("Average attempts per wallet: %.2f\n", float64(totalAttempts)/float64(count))
}

var (
	prefix       string
	suffix       string
	count        int
	checksum     bool
	showProgress bool
	// Benchmark specific flags
	benchmarkAttempts int64
	benchmarkPattern  string
)

// Benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run performance benchmark",
	Long: `Run a performance benchmark to test address generation speed.
	
This command generates addresses continuously and measures performance metrics
including average speed, speed range, and consistency.

Examples:
  bloco-wallet benchmark --attempts 10000 --pattern "fffff"
  bloco-wallet benchmark --attempts 50000 --pattern "abc" --checksum`,
	Run: func(cmd *cobra.Command, args []string) {
		if benchmarkPattern == "" {
			// Use default pattern if none specified
			benchmarkPattern = "fffff" // 5 hex chars for reasonable benchmark
		}

		// Validate hex characters
		if !isValidHex(benchmarkPattern) {
			fmt.Println("❌ Error: Benchmark pattern contains invalid hex characters")
			os.Exit(1)
		}

		runBenchmark(benchmarkAttempts, benchmarkPattern, checksum)
	},
}

// Statistics command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show difficulty statistics for a pattern",
	Long: `Display detailed statistics about the difficulty of generating
a bloco address with the specified pattern.

This includes difficulty calculation, probability analysis, and time estimates.

Examples:
  bloco-wallet stats --prefix abc --suffix 123
  bloco-wallet stats --prefix DeAdBeEf --checksum`,
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

		fmt.Printf("📊 Bloco Address Difficulty Analysis\n")
		fmt.Printf("═══════════════════════════════════════════════════════════════\n")
		fmt.Printf("🎯 Pattern: %s%s%s\n", prefix, strings.Repeat("*", 40-len(prefix)-len(suffix)), suffix)
		fmt.Printf("🔧 Checksum: %t\n", checksum)
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

		if checksum {
			fmt.Printf("   • 📝 Checksum enabled - Difficulty increased\n")
		}

		fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	},
}

var rootCmd = &cobra.Command{
	Use:   "bloco-wallet",
	Short: "Generate Ethereum bloco wallets with custom prefix and suffix",
	Long: `A high-performance CLI tool to generate Ethereum bloco wallets with custom 
prefix and suffix patterns.

This tool generates Ethereum wallets where the address starts with a specific prefix
and/or ends with a specific suffix. It supports checksum validation for more secure
bloco addresses and provides detailed statistics and progress information.

Examples:
  bloco-wallet --prefix abc --suffix 123 --count 5
  bloco-wallet --prefix deadbeef --checksum --count 1 --progress
  bloco-wallet --suffix coffee --count 10
  
Use 'bloco-wallet benchmark' to test performance
Use 'bloco-wallet stats' to analyze pattern difficulty`,
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
			fmt.Println("💡 Use 'bloco-wallet stats --prefix", prefix, "--suffix", suffix, "' to see difficulty analysis")
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

	// Define flags for benchmark command
	benchmarkCmd.Flags().Int64VarP(&benchmarkAttempts, "attempts", "a", 10000, "Number of attempts for benchmark")
	benchmarkCmd.Flags().StringVarP(&benchmarkPattern, "pattern", "p", "", "Pattern to use for benchmark (default: 'fffff')")
	benchmarkCmd.Flags().BoolVar(&checksum, "checksum", false, "Enable checksum validation for benchmark")

	// Define flags for stats command
	statsCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "Prefix for difficulty analysis")
	statsCmd.Flags().StringVarP(&suffix, "suffix", "s", "", "Suffix for difficulty analysis")
	statsCmd.Flags().BoolVar(&checksum, "checksum", false, "Enable checksum validation for analysis")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
