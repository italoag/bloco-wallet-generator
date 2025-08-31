package kdf

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// PBKDF2Handler implements the KDFHandler interface for PBKDF2 KDF
type PBKDF2Handler struct {
	hashFunc func() hash.Hash // Optional fixed hash function
}

// NewPBKDF2Handler creates a new PBKDF2 handler with default SHA-256
func NewPBKDF2Handler() KDFHandler {
	return &PBKDF2Handler{}
}

// NewPBKDF2SHA256Handler creates a new PBKDF2 handler with SHA-256
func NewPBKDF2SHA256Handler() KDFHandler {
	return &PBKDF2Handler{
		hashFunc: sha256.New,
	}
}

// NewPBKDF2SHA512Handler creates a new PBKDF2 handler with SHA-512
func NewPBKDF2SHA512Handler() KDFHandler {
	return &PBKDF2Handler{
		hashFunc: sha512.New,
	}
}

// DeriveKey derives a key using PBKDF2 with the given password and parameters
func (ph *PBKDF2Handler) DeriveKey(password string, params map[string]interface{}) ([]byte, error) {
	// Extract parameters with secure defaults
	iterations := ph.getIntParam(params, []string{"c", "iter", "iterations", "rounds"}, 262144)
	dklen := ph.getIntParam(params, []string{"dklen", "dkLen", "keylen", "length"}, 32)

	// Extract salt
	salt, err := ph.getSaltParam(params)
	if err != nil {
		return nil, fmt.Errorf("failed to extract salt: %w", err)
	}

	// Determine hash function
	hashFunc := ph.getHashFunction(params)

	// Derive key using PBKDF2
	return pbkdf2.Key([]byte(password), salt, iterations, dklen, hashFunc), nil
}

// ValidateParams validates PBKDF2 parameters for correctness and security
func (ph *PBKDF2Handler) ValidateParams(params map[string]interface{}) error {
	if params == nil {
		return NewKDFError("validation", "pbkdf2", "", nil, "non-nil map", "parameters cannot be nil")
	}

	// Validate iteration count
	iterations := ph.getIntParam(params, []string{"c", "iter", "iterations"}, 262144)
	if err := ph.validateIterations(iterations); err != nil {
		return err
	}

	// Validate derived key length
	dklen := ph.getIntParam(params, []string{"dklen", "dkLen"}, 32)
	if err := ph.validateDKLen(dklen); err != nil {
		return err
	}

	// Validate salt exists and is valid
	if _, err := ph.getSaltParam(params); err != nil {
		return NewKDFError("validation", "pbkdf2", "salt", nil, "valid salt",
			fmt.Sprintf("salt validation failed: %v", err)).
			WithSuggestions("Provide salt as hex string, byte array, or number array")
	}

	// Validate PRF (Pseudo Random Function) if specified
	if err := ph.validatePRF(params); err != nil {
		return err
	}

	return nil
}

// validateIterations validates the iteration count parameter
func (ph *PBKDF2Handler) validateIterations(iterations int) error {
	if iterations < 1000 {
		return NewKDFError("validation", "pbkdf2", "c", iterations, "≥ 1000",
			"Iteration count too low for modern security standards").
			WithSuggestions(
				"Use at least 100,000 iterations for good security",
				"Recommended: 262,144 iterations",
				"Consider migrating to scrypt for better GPU resistance",
			)
	}

	if iterations > 10000000 {
		return NewKDFError("validation", "pbkdf2", "c", iterations, "≤ 10,000,000",
			"Iteration count extremely high, may cause performance issues").
			WithSuggestions(
				"Use ≤ 10,000,000 iterations",
				"Consider scrypt for better memory-hard properties",
			)
	}

	// Warn about low iteration counts (but don't fail)
	if iterations < 100000 {
		// This would be logged as a warning by the logger
		return NewKDFError("validation", "pbkdf2", "c", iterations, "≥ 100,000",
			"Iteration count below recommended minimum").
			WithSuggestions(
				"Use at least 100,000 iterations for good security",
				"Modern recommendation: 262,144+ iterations",
			).WithRecoverable(true)
	}

	return nil
}

// validateDKLen validates the derived key length parameter
func (ph *PBKDF2Handler) validateDKLen(dklen int) error {
	if dklen < 16 {
		return NewKDFError("validation", "pbkdf2", "dklen", dklen, "≥ 16",
			"Derived key length too short for security").
			WithSuggestions("Use dklen ≥ 16 bytes", "Recommended: dklen = 32 bytes")
	}

	if dklen > 128 {
		return NewKDFError("validation", "pbkdf2", "dklen", dklen, "≤ 128",
			"Derived key length unnecessarily long").
			WithSuggestions("Use dklen ≤ 128 bytes", "Recommended: dklen = 32 bytes")
	}

	return nil
}

// validatePRF validates the Pseudo Random Function parameter
func (ph *PBKDF2Handler) validatePRF(params map[string]interface{}) error {
	prfNames := []string{"prf", "PRF", "hash", "hashFunc"}

	for _, name := range prfNames {
		if value, exists := params[name]; exists {
			if str, ok := value.(string); ok {
				normalizedPRF := strings.ToLower(strings.TrimSpace(str))
				switch normalizedPRF {
				case "hmac-sha256", "sha256", "sha-256":
					return nil
				case "hmac-sha512", "sha512", "sha-512":
					return nil
				default:
					return NewKDFError("validation", "pbkdf2", "prf", str, "hmac-sha256 or hmac-sha512",
						fmt.Sprintf("Unsupported PRF: %s", str)).
						WithSuggestions(
							"Use 'hmac-sha256' for SHA-256",
							"Use 'hmac-sha512' for SHA-512",
							"Leave empty for default SHA-256",
						)
				}
			} else {
				return NewKDFError("validation", "pbkdf2", "prf", value, "string",
					"PRF parameter must be a string")
			}
		}
	}

	return nil
}

// GetDefaultParams returns the default PBKDF2 parameters
func (ph *PBKDF2Handler) GetDefaultParams() map[string]interface{} {
	prf := "hmac-sha256"
	if ph.hashFunc != nil {
		// Determine PRF based on fixed hash function
		testHash := ph.hashFunc()
		switch testHash.Size() {
		case 32: // SHA-256
			prf = "hmac-sha256"
		case 64: // SHA-512
			prf = "hmac-sha512"
		}
	}

	return map[string]interface{}{
		"c":     262144, // Good balance of security and performance
		"dklen": 32,     // 256-bit key
		"prf":   prf,    // Pseudo Random Function
	}
}

// GetParamRange returns the valid range for a PBKDF2 parameter
func (ph *PBKDF2Handler) GetParamRange(param string) (min, max interface{}) {
	ranges := map[string][2]int{
		"c":     {1000, 10000000}, // 1K to 10M iterations
		"dklen": {16, 128},        // 16 to 128 bytes
	}

	if r, exists := ranges[param]; exists {
		return r[0], r[1]
	}
	return nil, nil
}

// getHashFunction determines the hash function to use based on parameters
func (ph *PBKDF2Handler) getHashFunction(params map[string]interface{}) func() hash.Hash {
	// If handler has a fixed hash function, use it
	if ph.hashFunc != nil {
		return ph.hashFunc
	}

	// Check PRF parameter
	prfNames := []string{"prf", "PRF", "hash", "hashFunc"}
	for _, name := range prfNames {
		if value, exists := params[name]; exists {
			if str, ok := value.(string); ok {
				normalizedPRF := strings.ToLower(strings.TrimSpace(str))
				switch normalizedPRF {
				case "hmac-sha256", "sha256", "sha-256":
					return sha256.New
				case "hmac-sha512", "sha512", "sha-512":
					return sha512.New
				}
			}
		}
	}

	// Default to SHA-256
	return sha256.New
}

// getIntParam extracts an integer parameter with multiple possible names
func (ph *PBKDF2Handler) getIntParam(params map[string]interface{}, names []string, defaultValue int) int {
	for _, name := range names {
		if value, exists := params[name]; exists {
			return ph.convertToInt(value, defaultValue)
		}
	}
	return defaultValue
}

// convertToInt converts various JSON types to int
func (ph *PBKDF2Handler) convertToInt(value interface{}, defaultValue int) int {
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
func (ph *PBKDF2Handler) getSaltParam(params map[string]interface{}) ([]byte, error) {
	saltNames := []string{"salt", "Salt", "SALT"}

	for _, name := range saltNames {
		if value, exists := params[name]; exists {
			return ph.convertToBytes(value)
		}
	}

	return nil, fmt.Errorf("salt parameter not found")
}

// convertToBytes converts various types to []byte
func (ph *PBKDF2Handler) convertToBytes(value interface{}) ([]byte, error) {
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
