package crypto

import (
	"crypto/rand"
	"fmt"
	"io"
	"runtime"
)

// EntropyValidator provides methods to validate entropy quality
type EntropyValidator struct {
	minEntropyBytes int
}

// NewEntropyValidator creates a new entropy validator with default settings
func NewEntropyValidator() *EntropyValidator {
	return &EntropyValidator{
		minEntropyBytes: 32, // Minimum 256 bits of entropy
	}
}

// ValidateEntropy performs basic entropy validation on random bytes
func (ev *EntropyValidator) ValidateEntropy(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("entropy data cannot be empty")
	}

	if len(data) < ev.minEntropyBytes {
		return fmt.Errorf("insufficient entropy: got %d bytes, minimum %d required", len(data), ev.minEntropyBytes)
	}

	// Basic entropy checks
	if err := ev.checkForPatterns(data); err != nil {
		return fmt.Errorf("entropy validation failed: %w", err)
	}

	return nil
}

// checkForPatterns performs basic pattern detection on random data
func (ev *EntropyValidator) checkForPatterns(data []byte) error {
	if len(data) < 4 {
		return nil // Too small to check patterns
	}

	// Check for all zeros
	allZeros := true
	for _, b := range data {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return fmt.Errorf("entropy contains all zeros")
	}

	// Check for all ones (0xFF)
	allOnes := true
	for _, b := range data {
		if b != 0xFF {
			allOnes = false
			break
		}
	}
	if allOnes {
		return fmt.Errorf("entropy contains all ones")
	}

	// Check for simple repeating patterns
	if len(data) >= 8 {
		if err := ev.checkRepeatingBytes(data); err != nil {
			return err
		}
	}

	return nil
}

// checkRepeatingBytes checks for simple repeating byte patterns
func (ev *EntropyValidator) checkRepeatingBytes(data []byte) error {
	// Check for single byte repetition over significant portion
	for i := 0; i < 256; i++ {
		count := 0
		for _, b := range data {
			if b == byte(i) {
				count++
			}
		}
		// If more than 90% of bytes are the same value, it's suspicious
		if float64(count)/float64(len(data)) > 0.9 {
			return fmt.Errorf("suspicious byte repetition: byte 0x%02x appears %d times out of %d", i, count, len(data))
		}
	}

	return nil
}

// SecureRandomGenerator provides enhanced random generation with validation
type SecureRandomGenerator struct {
	validator *EntropyValidator
	reader    io.Reader
}

// NewSecureRandomGenerator creates a new secure random generator
func NewSecureRandomGenerator() *SecureRandomGenerator {
	return &SecureRandomGenerator{
		validator: NewEntropyValidator(),
		reader:    rand.Reader,
	}
}

// GenerateRandomBytes generates cryptographically secure random bytes with enhanced validation
func (srg *SecureRandomGenerator) GenerateRandomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("length must be positive, got %d", length)
	}

	if length > 1024*1024 { // 1MB limit
		return nil, fmt.Errorf("length too large: %d bytes (maximum: 1MB)", length)
	}

	bytes := make([]byte, length)
	n, err := srg.reader.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	if n != length {
		return nil, fmt.Errorf("insufficient random bytes generated: got %d, expected %d", n, length)
	}

	// Validate entropy quality for cryptographic operations
	if length >= 16 { // Only validate for cryptographically significant lengths
		if err := srg.validator.ValidateEntropy(bytes); err != nil {
			// Try once more before failing
			n, retryErr := srg.reader.Read(bytes)
			if retryErr != nil {
				return nil, fmt.Errorf("failed to generate random bytes on retry: %w", retryErr)
			}
			if n != length {
				return nil, fmt.Errorf("insufficient random bytes on retry: got %d, expected %d", n, length)
			}

			// Validate again
			if validateErr := srg.validator.ValidateEntropy(bytes); validateErr != nil {
				return nil, fmt.Errorf("entropy validation failed after retry: %w", validateErr)
			}
		}
	}

	return bytes, nil
}

// GenerateSecureSalt generates a cryptographically secure salt with proper length validation
func (srg *SecureRandomGenerator) GenerateSecureSalt(length int) ([]byte, error) {
	// Validate salt length according to cryptographic standards
	if length < 8 {
		return nil, fmt.Errorf("salt length too short: %d bytes (minimum: 8)", length)
	}

	if length > 256 {
		return nil, fmt.Errorf("salt length too long: %d bytes (maximum: 256)", length)
	}

	// Recommended salt lengths for different use cases
	// 16: minimum recommended, 32: standard recommended, 64: high security

	salt, err := srg.GenerateRandomBytes(length)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Additional validation for salt uniqueness (basic check)
	if err := srg.validateSaltQuality(salt); err != nil {
		return nil, fmt.Errorf("salt quality validation failed: %w", err)
	}

	return salt, nil
}

// validateSaltQuality performs additional validation specific to salt generation
func (srg *SecureRandomGenerator) validateSaltQuality(salt []byte) error {
	if len(salt) == 0 {
		return fmt.Errorf("salt cannot be empty")
	}

	// Check for weak salt patterns
	if err := srg.validator.checkForPatterns(salt); err != nil {
		return fmt.Errorf("weak salt detected: %w", err)
	}

	return nil
}

// MemoryCleaner provides utilities for securely clearing sensitive data from memory
type MemoryCleaner struct{}

// NewMemoryCleaner creates a new memory cleaner
func NewMemoryCleaner() *MemoryCleaner {
	return &MemoryCleaner{}
}

// ClearBytes securely clears a byte slice by overwriting with zeros
func (mc *MemoryCleaner) ClearBytes(data []byte) {
	if data == nil {
		return
	}

	// Overwrite with zeros
	for i := range data {
		data[i] = 0
	}

	// Force garbage collection to ensure memory is cleared
	runtime.GC()
}

// ClearString securely clears a string by converting to bytes and clearing
// Note: This is best effort as Go strings are immutable
func (mc *MemoryCleaner) ClearString(s *string) {
	if s == nil || *s == "" {
		return
	}

	// Since Go strings are immutable, we can only clear the reference
	// The actual string data in memory may remain until garbage collected
	// This is a limitation of Go's memory model for strings
	*s = ""

	// Force garbage collection to help clear unreferenced string data
	runtime.GC()
}

// ClearByteSlices clears multiple byte slices
func (mc *MemoryCleaner) ClearByteSlices(slices ...[]byte) {
	for _, slice := range slices {
		mc.ClearBytes(slice)
	}
}

// SecureCopy creates a secure copy of sensitive data that can be safely cleared
func (mc *MemoryCleaner) SecureCopy(data []byte) []byte {
	if data == nil {
		return nil
	}

	copy := make([]byte, len(data))
	for i, b := range data {
		copy[i] = b
	}
	return copy
}

// Global instances for convenience
var (
	defaultRandomGenerator = NewSecureRandomGenerator()
	defaultMemoryCleaner   = NewMemoryCleaner()
)

// Enhanced GenerateRandomBytes function that replaces the basic implementation
func GenerateRandomBytesEnhanced(length int) ([]byte, error) {
	return defaultRandomGenerator.GenerateRandomBytes(length)
}

// GenerateSecureSalt generates a cryptographically secure salt with validation
func GenerateSecureSalt(length int) ([]byte, error) {
	return defaultRandomGenerator.GenerateSecureSalt(length)
}

// ClearSensitiveData securely clears sensitive byte data from memory
func ClearSensitiveData(data []byte) {
	defaultMemoryCleaner.ClearBytes(data)
}

// ClearSensitiveString securely clears sensitive string data from memory
func ClearSensitiveString(s *string) {
	defaultMemoryCleaner.ClearString(s)
}

// ClearMultipleSensitiveData clears multiple sensitive byte slices
func ClearMultipleSensitiveData(slices ...[]byte) {
	defaultMemoryCleaner.ClearByteSlices(slices...)
}

// ValidateRandomBytes validates the quality of random bytes
func ValidateRandomBytes(data []byte) error {
	validator := NewEntropyValidator()
	return validator.ValidateEntropy(data)
}
