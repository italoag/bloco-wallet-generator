package kdf

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// ScryptHandler implements the KDFHandler interface for scrypt KDF
type ScryptHandler struct{}

// NewScryptHandler creates a new scrypt handler
func NewScryptHandler() KDFHandler {
	return &ScryptHandler{}
}

// DeriveKey derives a key using scrypt with the given password and parameters
func (sh *ScryptHandler) DeriveKey(password string, params map[string]interface{}) ([]byte, error) {
	// Extract parameters with fallbacks to secure defaults
	n := sh.getIntParam(params, []string{"n", "N", "cost"}, 262144)
	r := sh.getIntParam(params, []string{"r", "R", "blocksize"}, 8)
	p := sh.getIntParam(params, []string{"p", "P", "parallel"}, 1)
	dklen := sh.getIntParam(params, []string{"dklen", "dkLen", "keylen", "length"}, 32)

	// Extract salt
	salt, err := sh.getSaltParam(params)
	if err != nil {
		return nil, fmt.Errorf("failed to extract salt: %w", err)
	}

	// Derive key using scrypt
	return scrypt.Key([]byte(password), salt, n, r, p, dklen)
}

// ValidateParams validates scrypt parameters for correctness and security
func (sh *ScryptHandler) ValidateParams(params map[string]interface{}) error {
	if params == nil {
		return NewKDFError("validation", "scrypt", "", nil, "non-nil map", "parameters cannot be nil")
	}

	// Validate N parameter (must be power of 2)
	n := sh.getIntParam(params, []string{"n", "N", "cost"}, 262144)
	if err := sh.validateNParameter(n); err != nil {
		return err
	}

	// Validate R parameter
	r := sh.getIntParam(params, []string{"r", "R", "blocksize"}, 8)
	if err := sh.validateRParameter(r); err != nil {
		return err
	}

	// Validate P parameter
	p := sh.getIntParam(params, []string{"p", "P", "parallel"}, 1)
	if err := sh.validatePParameter(p); err != nil {
		return err
	}

	// Validate dklen parameter
	dklen := sh.getIntParam(params, []string{"dklen", "dkLen", "keylen"}, 32)
	if err := sh.validateDKLenParameter(dklen); err != nil {
		return err
	}

	// Validate salt exists and is valid
	if _, err := sh.getSaltParam(params); err != nil {
		return NewKDFError("validation", "scrypt", "salt", nil, "valid salt",
			fmt.Sprintf("salt validation failed: %v", err)).
			WithSuggestions("Provide salt as hex string, byte array, or number array")
	}

	// Validate memory usage to prevent system exhaustion
	if err := sh.validateMemoryUsage(n, r); err != nil {
		return err
	}

	return nil
}

// validateNParameter validates the N parameter (CPU/memory cost)
func (sh *ScryptHandler) validateNParameter(n int) error {
	if n < 1024 {
		return NewKDFError("validation", "scrypt", "n", n, "≥ 1024",
			"N parameter too low for security").
			WithSuggestions("Use N ≥ 1024 for basic security", "Recommended: N = 262144 (2^18)")
	}

	if n > 67108864 { // 2^26
		return NewKDFError("validation", "scrypt", "n", n, "≤ 67108864",
			"N parameter too high, may cause memory exhaustion").
			WithSuggestions("Use N ≤ 67108864 (2^26)", "Consider N = 262144 (2^18) for good security")
	}

	if !sh.isPowerOfTwo(n) {
		return NewKDFError("validation", "scrypt", "n", n, "power of 2",
			"N parameter must be a power of 2").
			WithSuggestions("Use powers of 2: 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, etc.")
	}

	return nil
}

// validateRParameter validates the R parameter (block size)
func (sh *ScryptHandler) validateRParameter(r int) error {
	if r < 1 {
		return NewKDFError("validation", "scrypt", "r", r, "≥ 1",
			"R parameter must be positive").
			WithSuggestions("Use r ≥ 1", "Recommended: r = 8")
	}

	if r > 1024 {
		return NewKDFError("validation", "scrypt", "r", r, "≤ 1024",
			"R parameter too high, may cause excessive memory usage").
			WithSuggestions("Use r ≤ 1024", "Recommended: r = 8")
	}

	return nil
}

// validatePParameter validates the P parameter (parallelization)
func (sh *ScryptHandler) validatePParameter(p int) error {
	if p < 1 {
		return NewKDFError("validation", "scrypt", "p", p, "≥ 1",
			"P parameter must be positive").
			WithSuggestions("Use p ≥ 1", "Recommended: p = 1")
	}

	if p > 16 {
		return NewKDFError("validation", "scrypt", "p", p, "≤ 16",
			"P parameter too high, may not provide additional security benefit").
			WithSuggestions("Use p ≤ 16", "Recommended: p = 1")
	}

	return nil
}

// validateDKLenParameter validates the derived key length parameter
func (sh *ScryptHandler) validateDKLenParameter(dklen int) error {
	if dklen < 16 {
		return NewKDFError("validation", "scrypt", "dklen", dklen, "≥ 16",
			"Derived key length too short for security").
			WithSuggestions("Use dklen ≥ 16 bytes", "Recommended: dklen = 32 bytes")
	}

	if dklen > 128 {
		return NewKDFError("validation", "scrypt", "dklen", dklen, "≤ 128",
			"Derived key length unnecessarily long").
			WithSuggestions("Use dklen ≤ 128 bytes", "Recommended: dklen = 32 bytes")
	}

	return nil
}

// validateMemoryUsage validates that memory usage won't exceed system limits
func (sh *ScryptHandler) validateMemoryUsage(n, r int) error {
	// Calculate memory usage: 128 * N * r bytes
	memoryUsage := int64(128 * n * r)
	maxMemory := int64(2 * 1024 * 1024 * 1024) // 2GB limit

	if memoryUsage > maxMemory {
		return NewKDFError("validation", "scrypt", "memory", memoryUsage,
			fmt.Sprintf("≤ %d bytes", maxMemory),
			fmt.Sprintf("Memory usage too high: %d bytes (max: %d bytes)", memoryUsage, maxMemory)).
			WithSuggestions(
				"Reduce N parameter to lower memory usage",
				"Reduce r parameter to lower memory usage",
				fmt.Sprintf("Try N=%d, r=%d for ~256MB usage", 32768, 8),
			)
	}

	return nil
}

// GetDefaultParams returns the default scrypt parameters
func (sh *ScryptHandler) GetDefaultParams() map[string]interface{} {
	return map[string]interface{}{
		"n":     262144, // 2^18 - good balance of security and performance
		"r":     8,      // Standard block size
		"p":     1,      // Single thread
		"dklen": 32,     // 256-bit key
	}
}

// GetParamRange returns the valid range for a scrypt parameter
func (sh *ScryptHandler) GetParamRange(param string) (min, max interface{}) {
	ranges := map[string][2]int{
		"n":     {1024, 67108864}, // 2^10 to 2^26
		"r":     {1, 1024},        // 1 to 1024
		"p":     {1, 16},          // 1 to 16
		"dklen": {16, 128},        // 16 to 128 bytes
	}

	if r, exists := ranges[param]; exists {
		return r[0], r[1]
	}
	return nil, nil
}

// getIntParam extracts an integer parameter with multiple possible names
func (sh *ScryptHandler) getIntParam(params map[string]interface{}, names []string, defaultValue int) int {
	for _, name := range names {
		if value, exists := params[name]; exists {
			return sh.convertToInt(value, defaultValue)
		}
	}
	return defaultValue
}

// convertToInt converts various JSON types to int
func (sh *ScryptHandler) convertToInt(value interface{}, defaultValue int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	}
	return defaultValue
}

// getSaltParam extracts salt from various formats
func (sh *ScryptHandler) getSaltParam(params map[string]interface{}) ([]byte, error) {
	saltNames := []string{"salt", "Salt", "SALT"}

	for _, name := range saltNames {
		if value, exists := params[name]; exists {
			return sh.convertToBytes(value)
		}
	}

	return nil, fmt.Errorf("salt parameter not found")
}

// convertToBytes converts various types to []byte
func (sh *ScryptHandler) convertToBytes(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		// Try hex decode first (remove 0x prefix if present)
		hexStr := strings.TrimPrefix(v, "0x")
		if len(hexStr)%2 == 0 && len(hexStr) > 0 {
			if bytes, err := hex.DecodeString(hexStr); err == nil {
				return bytes, nil
			}
		}
		// Fallback to string bytes
		return []byte(v), nil

	case []byte:
		return v, nil

	case []interface{}:
		// Array of numbers (common in JSON)
		bytes := make([]byte, len(v))
		for i, item := range v {
			switch num := item.(type) {
			case float64:
				if num < 0 || num > 255 {
					return nil, fmt.Errorf("salt array item %d out of byte range: %v", i, num)
				}
				bytes[i] = byte(num)
			case int:
				if num < 0 || num > 255 {
					return nil, fmt.Errorf("salt array item %d out of byte range: %v", i, num)
				}
				bytes[i] = byte(num)
			default:
				return nil, fmt.Errorf("salt array item %d invalid type: %T", i, item)
			}
		}
		return bytes, nil

	default:
		return nil, fmt.Errorf("unsupported salt type: %T", value)
	}
}

// isPowerOfTwo checks if a number is a power of 2
func (sh *ScryptHandler) isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}
