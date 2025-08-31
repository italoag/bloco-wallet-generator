package kdf

import (
	"fmt"
	"strings"
	"time"
)

// UniversalKDFService provides a unified interface for all supported KDF algorithms
// It supports scrypt and PBKDF2 with comprehensive parameter validation and normalization
type UniversalKDFService struct {
	supportedKDFs map[string]KDFHandler
	logger        KDFLogger
}

// NewUniversalKDFService creates a new Universal KDF service with default handlers
func NewUniversalKDFService() *UniversalKDFService {
	service := &UniversalKDFService{
		supportedKDFs: make(map[string]KDFHandler),
		logger:        &NoOpKDFLogger{}, // Default to no-op logger
	}

	// Register default KDF handlers
	service.RegisterKDF("scrypt", NewScryptHandler())
	service.RegisterKDF("pbkdf2", NewPBKDF2Handler())
	service.RegisterKDF("pbkdf2-sha256", NewPBKDF2SHA256Handler())
	service.RegisterKDF("pbkdf2-sha512", NewPBKDF2SHA512Handler())

	return service
}

// NewUniversalKDFServiceWithLogger creates a new Universal KDF service with a custom logger
func NewUniversalKDFServiceWithLogger(logger KDFLogger) *UniversalKDFService {
	service := NewUniversalKDFService()
	service.logger = logger
	return service
}

// RegisterKDF registers a new KDF handler with the service
func (uks *UniversalKDFService) RegisterKDF(name string, handler KDFHandler) {
	if handler == nil {
		return
	}
	uks.supportedKDFs[strings.ToLower(name)] = handler
}

// DeriveKey derives a key using the specified KDF and parameters
func (uks *UniversalKDFService) DeriveKey(password string, crypto *CryptoParams) ([]byte, error) {
	if crypto == nil {
		return nil, NewKDFError("validation", "", "", nil, nil, "crypto parameters cannot be nil")
	}

	kdfName := crypto.KDF
	if kdfName == "" {
		return nil, NewKDFError("validation", "", "kdf", "", "non-empty string", "KDF type is required")
	}

	// Normalize KDF name (case-insensitive, handle aliases)
	normalizedKDF := uks.normalizeKDFName(kdfName)

	handler, exists := uks.supportedKDFs[normalizedKDF]
	if !exists {
		return nil, NewKDFError("compatibility", kdfName, "", kdfName, "supported KDF type",
			fmt.Sprintf("KDF not supported: %s (normalized: %s)", kdfName, normalizedKDF)).
			WithSuggestions("Use 'scrypt', 'pbkdf2', 'pbkdf2-sha256', or 'pbkdf2-sha512'")
	}

	// Log the attempt
	uks.logger.LogKDFAttempt(normalizedKDF, crypto.KDFParams)

	// Validate parameters before derivation
	if err := handler.ValidateParams(crypto.KDFParams); err != nil {
		kdfErr := &KDFError{
			Type:        "validation",
			KDFType:     normalizedKDF,
			Message:     err.Error(),
			Recoverable: true,
		}
		uks.logger.LogKDFError(normalizedKDF, kdfErr)
		return nil, kdfErr
	}

	// Derive the key
	start := time.Now()
	derivedKey, err := handler.DeriveKey(password, crypto.KDFParams)
	duration := time.Since(start)

	if err != nil {
		kdfErr := &KDFError{
			Type:        "derivation",
			KDFType:     normalizedKDF,
			Message:     err.Error(),
			Recoverable: false,
		}
		uks.logger.LogKDFError(normalizedKDF, kdfErr)
		return nil, kdfErr
	}

	uks.logger.LogKDFSuccess(normalizedKDF, duration)
	return derivedKey, nil
}

// normalizeKDFName normalizes KDF names to handle case variations and aliases
func (uks *UniversalKDFService) normalizeKDFName(kdf string) string {
	// Convert to lowercase and trim whitespace for case-insensitive matching
	kdfLower := strings.ToLower(strings.TrimSpace(kdf))

	// Handle common variations and aliases
	kdfMap := map[string]string{
		// Scrypt variations
		"scrypt": "scrypt",
		"Scrypt": "scrypt",
		"SCRYPT": "scrypt",

		// PBKDF2 variations
		"pbkdf2":        "pbkdf2",
		"PBKDF2":        "pbkdf2",
		"pbkdf2-sha256": "pbkdf2-sha256",
		"PBKDF2-SHA256": "pbkdf2-sha256",
		"pbkdf2-sha512": "pbkdf2-sha512",
		"PBKDF2-SHA512": "pbkdf2-sha512",

		// Underscore variants (common in some implementations)
		"pbkdf2_sha256": "pbkdf2-sha256",
		"PBKDF2_SHA256": "pbkdf2-sha256",
		"pbkdf2_sha512": "pbkdf2-sha512",
		"PBKDF2_SHA512": "pbkdf2-sha512",

		// Alternative naming conventions
		"pbkdf2sha256": "pbkdf2-sha256",
		"PBKDF2SHA256": "pbkdf2-sha256",
		"pbkdf2sha512": "pbkdf2-sha512",
		"PBKDF2SHA512": "pbkdf2-sha512",

		// Hash function specific aliases
		"pbkdf2-256": "pbkdf2-sha256",
		"pbkdf2-512": "pbkdf2-sha512",
		"pbkdf2_256": "pbkdf2-sha256",
		"pbkdf2_512": "pbkdf2-sha512",
	}

	if normalized, exists := kdfMap[kdfLower]; exists {
		return normalized
	}

	// Return lowercase version if no specific mapping found
	return kdfLower
}

// GetSupportedKDFs returns a list of supported KDF types
func (uks *UniversalKDFService) GetSupportedKDFs() []string {
	kdfs := make([]string, 0, len(uks.supportedKDFs))
	for kdf := range uks.supportedKDFs {
		kdfs = append(kdfs, kdf)
	}
	return kdfs
}

// GetDefaultParams returns default parameters for a specific KDF type
func (uks *UniversalKDFService) GetDefaultParams(kdfType string) (map[string]interface{}, error) {
	normalizedKDF := uks.normalizeKDFName(kdfType)

	handler, exists := uks.supportedKDFs[normalizedKDF]
	if !exists {
		return nil, NewKDFError("compatibility", kdfType, "", kdfType, "supported KDF type",
			fmt.Sprintf("KDF not supported: %s", kdfType))
	}

	return handler.GetDefaultParams(), nil
}

// ValidateParams validates parameters for a specific KDF type
func (uks *UniversalKDFService) ValidateParams(kdfType string, params map[string]interface{}) error {
	normalizedKDF := uks.normalizeKDFName(kdfType)

	handler, exists := uks.supportedKDFs[normalizedKDF]
	if !exists {
		return NewKDFError("compatibility", kdfType, "", kdfType, "supported KDF type",
			fmt.Sprintf("KDF not supported: %s", kdfType))
	}

	return handler.ValidateParams(params)
}

// GetParamRange returns the valid range for a parameter of a specific KDF type
func (uks *UniversalKDFService) GetParamRange(kdfType, param string) (min, max interface{}, err error) {
	normalizedKDF := uks.normalizeKDFName(kdfType)

	handler, exists := uks.supportedKDFs[normalizedKDF]
	if !exists {
		return nil, nil, NewKDFError("compatibility", kdfType, "", kdfType, "supported KDF type",
			fmt.Sprintf("KDF not supported: %s", kdfType))
	}

	min, max = handler.GetParamRange(param)
	if min == nil && max == nil {
		return nil, nil, NewKDFError("validation", normalizedKDF, param, param, "valid parameter name",
			fmt.Sprintf("Parameter not supported: %s", param))
	}

	return min, max, nil
}

// SetLogger sets the logger for the service
func (uks *UniversalKDFService) SetLogger(logger KDFLogger) {
	if logger != nil {
		uks.logger = logger
	}
}

// IsKDFSupported checks if a KDF type is supported
func (uks *UniversalKDFService) IsKDFSupported(kdfType string) bool {
	normalizedKDF := uks.normalizeKDFName(kdfType)
	_, exists := uks.supportedKDFs[normalizedKDF]
	return exists
}

// GetDefaultKDF returns the default KDF type
func (uks *UniversalKDFService) GetDefaultKDF() string {
	return "scrypt" // scrypt is the default due to better GPU resistance
}

// GetRecommendedKDF returns the recommended KDF type for a given security level
func (uks *UniversalKDFService) GetRecommendedKDF(securityLevel SecurityLevel) string {
	switch securityLevel {
	case SecurityLevelLow:
		return "pbkdf2" // Faster but less secure
	case SecurityLevelMedium:
		return "scrypt" // Good balance
	case SecurityLevelHigh, SecurityLevelVeryHigh:
		return "scrypt" // Best GPU resistance
	default:
		return uks.GetDefaultKDF()
	}
}

// UnregisterKDF removes a KDF handler from the service
func (uks *UniversalKDFService) UnregisterKDF(name string) {
	normalizedName := strings.ToLower(name)
	delete(uks.supportedKDFs, normalizedName)
}

// GetKDFAliases returns all known aliases for a KDF type
func (uks *UniversalKDFService) GetKDFAliases(kdfType string) []string {
	normalizedKDF := uks.normalizeKDFName(kdfType)

	// Build reverse mapping to find all aliases
	aliases := make([]string, 0)

	// All possible variations we support
	allVariations := map[string]string{
		"scrypt":        "scrypt",
		"Scrypt":        "scrypt",
		"SCRYPT":        "scrypt",
		"pbkdf2":        "pbkdf2",
		"PBKDF2":        "pbkdf2",
		"pbkdf2-sha256": "pbkdf2-sha256",
		"PBKDF2-SHA256": "pbkdf2-sha256",
		"pbkdf2-sha512": "pbkdf2-sha512",
		"PBKDF2-SHA512": "pbkdf2-sha512",
		"pbkdf2_sha256": "pbkdf2-sha256",
		"PBKDF2_SHA256": "pbkdf2-sha256",
		"pbkdf2_sha512": "pbkdf2-sha512",
		"PBKDF2_SHA512": "pbkdf2-sha512",
		"pbkdf2sha256":  "pbkdf2-sha256",
		"PBKDF2SHA256":  "pbkdf2-sha256",
		"pbkdf2sha512":  "pbkdf2-sha512",
		"PBKDF2SHA512":  "pbkdf2-sha512",
		"pbkdf2-256":    "pbkdf2-sha256",
		"pbkdf2-512":    "pbkdf2-sha512",
		"pbkdf2_256":    "pbkdf2-sha256",
		"pbkdf2_512":    "pbkdf2-sha512",
	}

	for alias, normalized := range allVariations {
		if normalized == normalizedKDF {
			aliases = append(aliases, alias)
		}
	}

	return aliases
}

// NoOpKDFLogger is a no-operation logger that discards all log messages
type NoOpKDFLogger struct{}

// LogKDFAttempt does nothing
func (l *NoOpKDFLogger) LogKDFAttempt(kdf string, params map[string]interface{}) {}

// LogKDFSuccess does nothing
func (l *NoOpKDFLogger) LogKDFSuccess(kdf string, duration time.Duration) {}

// LogKDFError does nothing
func (l *NoOpKDFLogger) LogKDFError(kdf string, err error) {}

// LogParamValidation does nothing
func (l *NoOpKDFLogger) LogParamValidation(param string, value interface{}, valid bool) {}
