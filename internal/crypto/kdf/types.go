package kdf

import (
	"fmt"
)

// CryptoParams represents the crypto section parameters for KDF operations
// This matches the Ethereum KeyStore V3 specification
type CryptoParams struct {
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams map[string]interface{} `json:"cipherparams"`
	MAC          string                 `json:"mac"`
}

// KDFError represents errors that occur during KDF operations
type KDFError struct {
	Type        string      `json:"type"`        // validation, derivation, compatibility
	KDFType     string      `json:"kdf_type"`    // scrypt, pbkdf2, etc.
	Parameter   string      `json:"parameter"`   // specific parameter that caused error
	Value       interface{} `json:"value"`       // parameter value that caused error
	Expected    interface{} `json:"expected"`    // expected value or range
	Message     string      `json:"message"`     // human-readable error message
	Suggestions []string    `json:"suggestions"` // suggested fixes
	Recoverable bool        `json:"recoverable"` // whether error can be recovered from
}

// Error implements the error interface
func (e *KDFError) Error() string {
	if e.Parameter != "" {
		return fmt.Sprintf("KDF %s error in parameter %s: %s", e.KDFType, e.Parameter, e.Message)
	}
	return fmt.Sprintf("KDF %s error: %s", e.KDFType, e.Message)
}

// IsRecoverable returns whether the error can be recovered from
func (e *KDFError) IsRecoverable() bool {
	return e.Recoverable
}

// GetSuggestions returns suggested fixes for the error
func (e *KDFError) GetSuggestions() []string {
	return e.Suggestions
}

// NewKDFError creates a new KDF error
func NewKDFError(errorType, kdfType, parameter string, value, expected interface{}, message string) *KDFError {
	return &KDFError{
		Type:        errorType,
		KDFType:     kdfType,
		Parameter:   parameter,
		Value:       value,
		Expected:    expected,
		Message:     message,
		Suggestions: make([]string, 0),
		Recoverable: errorType == "validation", // validation errors are typically recoverable
	}
}

// WithSuggestions adds suggestions to the error
func (e *KDFError) WithSuggestions(suggestions ...string) *KDFError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// WithRecoverable sets whether the error is recoverable
func (e *KDFError) WithRecoverable(recoverable bool) *KDFError {
	e.Recoverable = recoverable
	return e
}

// ScryptParams represents scrypt KDF parameters
type ScryptParams struct {
	DKLen int    `json:"dklen"` // Derived key length
	N     int    `json:"n"`     // CPU/memory cost parameter (must be power of 2)
	P     int    `json:"p"`     // Parallelization parameter
	R     int    `json:"r"`     // Block size parameter
	Salt  string `json:"salt"`  // Hex-encoded salt
}

// PBKDF2Params represents PBKDF2 KDF parameters
type PBKDF2Params struct {
	DKLen int    `json:"dklen"` // Derived key length
	C     int    `json:"c"`     // Iteration count
	PRF   string `json:"prf"`   // Pseudo-random function (hmac-sha256, hmac-sha512)
	Salt  string `json:"salt"`  // Hex-encoded salt
}

// SecurityLevel represents the security level of KDF parameters
type SecurityLevel string

const (
	SecurityLevelLow      SecurityLevel = "Low"
	SecurityLevelMedium   SecurityLevel = "Medium"
	SecurityLevelHigh     SecurityLevel = "High"
	SecurityLevelVeryHigh SecurityLevel = "Very High"
)

// SecurityAnalysis represents security analysis of KDF parameters
type SecurityAnalysis struct {
	Level               SecurityLevel   `json:"level"`                // Security level assessment
	ComputationalCost   float64         `json:"computational_cost"`   // Estimated operations
	MemoryUsage         int64           `json:"memory_usage"`         // Memory usage in bytes
	Recommendations     []string        `json:"recommendations"`      // Security recommendations
	ClientCompatibility map[string]bool `json:"client_compatibility"` // Client -> compatible
}

// CompatibilityReport represents compatibility analysis results
type CompatibilityReport struct {
	Compatible    bool                   `json:"compatible"`     // Overall compatibility
	KDFType       string                 `json:"kdf_type"`       // Original KDF type
	NormalizedKDF string                 `json:"normalized_kdf"` // Normalized KDF name
	Parameters    map[string]interface{} `json:"parameters"`     // KDF parameters
	SecurityLevel SecurityLevel          `json:"security_level"` // Security assessment
	Issues        []string               `json:"issues"`         // Compatibility issues
	Warnings      []string               `json:"warnings"`       // Warnings
	Suggestions   []string               `json:"suggestions"`    // Improvement suggestions
}

// KDFType represents supported KDF types
type KDFType string

const (
	KDFTypeScrypt       KDFType = "scrypt"
	KDFTypePBKDF2       KDFType = "pbkdf2"
	KDFTypePBKDF2SHA256 KDFType = "pbkdf2-sha256"
	KDFTypePBKDF2SHA512 KDFType = "pbkdf2-sha512"
)

// String returns the string representation of the KDF type
func (k KDFType) String() string {
	return string(k)
}

// IsValid checks if the KDF type is valid
func (k KDFType) IsValid() bool {
	switch k {
	case KDFTypeScrypt, KDFTypePBKDF2, KDFTypePBKDF2SHA256, KDFTypePBKDF2SHA512:
		return true
	default:
		return false
	}
}

// DefaultKDFParams provides default parameters for each KDF type
var DefaultKDFParams = map[KDFType]map[string]interface{}{
	KDFTypeScrypt: {
		"n":     262144, // 2^18
		"r":     8,
		"p":     1,
		"dklen": 32,
	},
	KDFTypePBKDF2: {
		"c":     262144,
		"dklen": 32,
		"prf":   "hmac-sha256",
	},
	KDFTypePBKDF2SHA256: {
		"c":     262144,
		"dklen": 32,
		"prf":   "hmac-sha256",
	},
	KDFTypePBKDF2SHA512: {
		"c":     262144,
		"dklen": 32,
		"prf":   "hmac-sha512",
	},
}

// KDFParamRanges defines valid parameter ranges for each KDF type
var KDFParamRanges = map[KDFType]map[string][2]interface{}{
	KDFTypeScrypt: {
		"n":     {1024, 67108864}, // 2^10 to 2^26
		"r":     {1, 1024},        // 1 to 1024
		"p":     {1, 16},          // 1 to 16
		"dklen": {16, 128},        // 16 to 128 bytes
	},
	KDFTypePBKDF2: {
		"c":     {1000, 10000000}, // 1K to 10M iterations
		"dklen": {16, 128},        // 16 to 128 bytes
	},
	KDFTypePBKDF2SHA256: {
		"c":     {1000, 10000000}, // 1K to 10M iterations
		"dklen": {16, 128},        // 16 to 128 bytes
	},
	KDFTypePBKDF2SHA512: {
		"c":     {1000, 10000000}, // 1K to 10M iterations
		"dklen": {16, 128},        // 16 to 128 bytes
	},
}
