package main

import (
	"encoding/hex"
	"math"
	"strings"
	"testing"
)

// init function to initialize pools for testing
func init() {
	initializePools()
}

func TestPrivateToAddress(t *testing.T) {
	// Test with a known private key
	privKeyHex := "c87509a1c067bbde78beb793e6fa76530b6382a4c0241e5e4a9ec0a0f44dc0d3"
	privKey, err := hex.DecodeString(privKeyHex)
	if err != nil {
		t.Fatalf("Failed to decode private key: %v", err)
	}

	address := privateToAddress(privKey)
	if len(address) != 40 {
		t.Errorf("Expected address length 40, got %d", len(address))
	}

	// Address should be valid hex
	_, err = hex.DecodeString(address)
	if err != nil {
		t.Errorf("Generated address is not valid hex: %v", err)
	}
}

func TestGetRandomWallet(t *testing.T) {
	wallet, err := getRandomWallet()
	if err != nil {
		t.Fatalf("Failed to generate random wallet: %v", err)
	}

	// Check private key
	if len(wallet.PrivKey) != 64 {
		t.Errorf("Expected private key length 64, got %d", len(wallet.PrivKey))
	}

	// Check if private key is valid hex
	_, err = hex.DecodeString(wallet.PrivKey)
	if err != nil {
		t.Errorf("Private key is not valid hex: %v", err)
	}

	// Check address
	if len(wallet.Address) != 40 {
		t.Errorf("Expected address length 40, got %d", len(wallet.Address))
	}

	// Check if address is valid hex
	_, err = hex.DecodeString(wallet.Address)
	if err != nil {
		t.Errorf("Address is not valid hex: %v", err)
	}
}

func TestIsValidBlocoAddress(t *testing.T) {
	testCases := []struct {
		address  string
		prefix   string
		suffix   string
		checksum bool
		expected bool
	}{
		// Basic prefix matching
		{
			address:  "abcd1234567890123456789012345678901234ef",
			prefix:   "abcd",
			suffix:   "",
			checksum: false,
			expected: true,
		},
		// Basic suffix matching
		{
			address:  "1234567890123456789012345678901234abcdef",
			prefix:   "",
			suffix:   "cdef",
			checksum: false,
			expected: true,
		},
		// Both prefix and suffix matching
		{
			address:  "abcd567890123456789012345678901234cdef",
			prefix:   "abcd",
			suffix:   "cdef",
			checksum: false,
			expected: true,
		},
		// Case insensitive matching
		{
			address:  "ABCD567890123456789012345678901234cdef",
			prefix:   "abcd",
			suffix:   "CDEF",
			checksum: false,
			expected: true,
		},
		// Non-matching prefix
		{
			address:  "1234567890123456789012345678901234cdef",
			prefix:   "abcd",
			suffix:   "",
			checksum: false,
			expected: false,
		},
		// Non-matching suffix
		{
			address:  "abcd567890123456789012345678901234xyz1",
			prefix:   "",
			suffix:   "cdef",
			checksum: false,
			expected: false,
		},
	}

	for i, tc := range testCases {
		result := isValidBlocoAddress(tc.address, tc.prefix, tc.suffix, tc.checksum)
		if result != tc.expected {
			t.Errorf("Test case %d failed: expected %v, got %v", i+1, tc.expected, result)
			t.Errorf("  Address: %s, Prefix: %s, Suffix: %s", tc.address, tc.prefix, tc.suffix)
		}
	}
}

func TestToChecksumAddress(t *testing.T) {
	// Test with known addresses
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			expected: "5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		},
		{
			input:    "fb6916095ca1df60bb79ce92ce3ea74c37c5d359",
			expected: "fB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		},
	}

	for i, tc := range testCases {
		result := toChecksumAddress(tc.input)
		if !strings.EqualFold(result, tc.expected) {
			t.Errorf("Test case %d failed: expected %s, got %s", i+1, tc.expected, result)
		}
	}
}

func TestGenerateBlocoWallet(t *testing.T) {
	// Test with simple prefix
	result := generateBlocoWallet("a", "", false, false)
	if result.Error != "" {
		t.Fatalf("Unexpected error: %s", result.Error)
	}

	if result.Wallet == nil {
		t.Fatal("Expected wallet to be generated")
	}

	// Check if generated address has the correct prefix
	address := strings.ToLower(result.Wallet.Address[2:]) // Remove 0x prefix
	if !strings.HasPrefix(address, "a") {
		t.Errorf("Generated address doesn't have prefix 'a': %s", address)
	}

	// Check if address is valid
	if len(result.Wallet.Address) != 42 { // 40 chars + "0x"
		t.Errorf("Expected address length 42 (with 0x), got %d", len(result.Wallet.Address))
	}

	// Check if private key is valid
	if len(result.Wallet.PrivKey) != 64 {
		t.Errorf("Expected private key length 64, got %d", len(result.Wallet.PrivKey))
	}

	// Verify attempts counter
	if result.Attempts <= 0 {
		t.Errorf("Expected positive attempts count, got %d", result.Attempts)
	}
}

func BenchmarkGetRandomWallet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := getRandomWallet()
		if err != nil {
			b.Fatalf("Failed to generate wallet: %v", err)
		}
	}
}

func BenchmarkPrivateToAddress(b *testing.B) {
	// Generate a test private key
	privKeyHex := "c87509a1c067bbde78beb793e6fa76530b6382a4c0241e5e4a9ec0a0f44dc0d3"
	privKey, _ := hex.DecodeString(privKeyHex)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		privateToAddress(privKey)
	}
}

func BenchmarkIsValidBlocoAddress(b *testing.B) {
	address := "abcd1234567890123456789012345678901234ef"
	prefix := "abcd"
	suffix := ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isValidBlocoAddress(address, prefix, suffix, false)
	}
}

func BenchmarkToChecksumAddress(b *testing.B) {
	address := "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toChecksumAddress(address)
	}
}

func TestComputeDifficulty(t *testing.T) {
	testCases := []struct {
		prefix   string
		suffix   string
		checksum bool
		expected float64
		name     string
	}{
		{
			prefix:   "a",
			suffix:   "",
			checksum: false,
			expected: 16,
			name:     "single prefix char",
		},
		{
			prefix:   "ab",
			suffix:   "",
			checksum: false,
			expected: 256,
			name:     "two prefix chars",
		},
		{
			prefix:   "",
			suffix:   "12",
			checksum: false,
			expected: 256,
			name:     "two suffix chars",
		},
		{
			prefix:   "a",
			suffix:   "1",
			checksum: false,
			expected: 256,
			name:     "prefix and suffix",
		},
		{
			prefix:   "a",
			suffix:   "",
			checksum: true,
			expected: 32, // 16 * 2^1 (one letter)
			name:     "checksum with letter",
		},
		{
			prefix:   "1",
			suffix:   "",
			checksum: true,
			expected: 16, // 16 * 2^0 (no letters)
			name:     "checksum with number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeDifficulty(tc.prefix, tc.suffix, tc.checksum)
			if result != tc.expected {
				t.Errorf("Expected difficulty %f, got %f", tc.expected, result)
			}
		})
	}
}

func TestComputeProbability(t *testing.T) {
	testCases := []struct {
		difficulty float64
		attempts   int64
		expected   float64
		tolerance  float64
		name       string
	}{
		{
			difficulty: 16,
			attempts:   11, // ~ln(2) * 16
			expected:   0.5,
			tolerance:  0.1,
			name:       "50% probability case",
		},
		{
			difficulty: 16,
			attempts:   0,
			expected:   0,
			tolerance:  0.001,
			name:       "zero attempts",
		},
		{
			difficulty: 16,
			attempts:   16,
			expected:   0.6321, // 1 - 1/e
			tolerance:  0.02,
			name:       "one expected attempt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeProbability(tc.difficulty, tc.attempts)
			if math.Abs(result-tc.expected) > tc.tolerance {
				t.Errorf("Expected probability %f, got %f (tolerance %f)", tc.expected, result, tc.tolerance)
			}
		})
	}
}

func TestComputeProbability50(t *testing.T) {
	testCases := []struct {
		difficulty float64
		expected   int64
		tolerance  int64
		name       string
	}{
		{
			difficulty: 16,
			expected:   11, // ~ln(2) * 16
			tolerance:  2,
			name:       "difficulty 16",
		},
		{
			difficulty: 256,
			expected:   177, // ~ln(2) * 256
			tolerance:  10,
			name:       "difficulty 256",
		},
		{
			difficulty: 0,
			expected:   0,
			tolerance:  0,
			name:       "zero difficulty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeProbability50(tc.difficulty)
			if tc.expected == 0 && result != 0 {
				t.Errorf("Expected 0, got %d", result)
			} else if tc.expected > 0 && math.Abs(float64(result-tc.expected)) > float64(tc.tolerance) {
				t.Errorf("Expected ~%d, got %d (tolerance %d)", tc.expected, result, tc.tolerance)
			}
		})
	}
}

func TestIsValidHex(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
		name     string
	}{
		{
			input:    "abc123",
			expected: true,
			name:     "valid hex lowercase",
		},
		{
			input:    "ABC123",
			expected: true,
			name:     "valid hex uppercase",
		},
		{
			input:    "DeAdBeEf",
			expected: true,
			name:     "valid hex mixed case",
		},
		{
			input:    "",
			expected: true,
			name:     "empty string",
		},
		{
			input:    "xyz",
			expected: false,
			name:     "invalid characters",
		},
		{
			input:    "123g",
			expected: false,
			name:     "invalid character g",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidHex(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %t for input '%s', got %t", tc.expected, tc.input, result)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	testCases := []struct {
		input    int64
		expected string
		name     string
	}{
		{
			input:    123,
			expected: "123",
			name:     "small number",
		},
		{
			input:    1234,
			expected: "1 234",
			name:     "thousands",
		},
		{
			input:    1234567,
			expected: "1 234 567",
			name:     "millions",
		},
		{
			input:    0,
			expected: "0",
			name:     "zero",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatNumber(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestNewStatistics(t *testing.T) {
	stats := newStatistics("abc", "123", false)

	if stats.Pattern != "abc123" {
		t.Errorf("Expected pattern 'abc123', got '%s'", stats.Pattern)
	}

	if stats.Difficulty != 16777216 { // 16^6
		t.Errorf("Expected difficulty 16777216, got %f", stats.Difficulty)
	}

	if stats.IsChecksum != false {
		t.Errorf("Expected checksum false, got %t", stats.IsChecksum)
	}

	if stats.CurrentAttempts != 0 {
		t.Errorf("Expected initial attempts 0, got %d", stats.CurrentAttempts)
	}
}

func TestStatisticsUpdate(t *testing.T) {
	stats := newStatistics("a", "", false)

	// Test update with some attempts
	attempts := int64(100)
	stats.update(attempts)

	if stats.CurrentAttempts != attempts {
		t.Errorf("Expected current attempts %d, got %d", attempts, stats.CurrentAttempts)
	}

	if stats.Probability == 0 {
		t.Error("Expected probability to be calculated and non-zero")
	}

	if stats.Speed == 0 {
		t.Error("Expected speed to be calculated and non-zero")
	}
}

// Benchmark test for the statistics calculations
func BenchmarkComputeDifficulty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		computeDifficulty("deadbeef", "", false)
	}
}

func BenchmarkComputeProbability(b *testing.B) {
	difficulty := float64(16777216)
	attempts := int64(1000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeProbability(difficulty, attempts)
	}
}

func BenchmarkStatisticsUpdate(b *testing.B) {
	stats := newStatistics("abc", "123", false)
	attempts := int64(50000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.update(attempts)
	}
}
