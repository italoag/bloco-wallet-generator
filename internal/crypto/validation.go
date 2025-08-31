package crypto

import (
	"fmt"
	"math"
	"time"
)

// SecurityLevel represents the security strength of cryptographic parameters
type SecurityLevel int

const (
	SecurityLevelLow SecurityLevel = iota
	SecurityLevelMedium
	SecurityLevelHigh
	SecurityLevelVeryHigh
)

// String returns the string representation of the security level
func (sl SecurityLevel) String() string {
	switch sl {
	case SecurityLevelLow:
		return "Low"
	case SecurityLevelMedium:
		return "Medium"
	case SecurityLevelHigh:
		return "High"
	case SecurityLevelVeryHigh:
		return "Very High"
	default:
		return "Unknown"
	}
}

// ParameterRange defines the valid range for a cryptographic parameter
type ParameterRange struct {
	Min         interface{} `json:"min"`
	Max         interface{} `json:"max"`
	Recommended interface{} `json:"recommended"`
	Description string      `json:"description"`
}

// ValidationResult contains the result of parameter validation
type ValidationResult struct {
	Valid         bool                      `json:"valid"`
	SecurityLevel SecurityLevel             `json:"security_level"`
	Issues        []string                  `json:"issues"`
	Warnings      []string                  `json:"warnings"`
	Suggestions   []string                  `json:"suggestions"`
	Ranges        map[string]ParameterRange `json:"ranges"`
	Analysis      *CryptographicAnalysis    `json:"analysis,omitempty"`
}

// CryptographicAnalysis provides detailed analysis of cryptographic strength
type CryptographicAnalysis struct {
	ComputationalCost float64       `json:"computational_cost"` // Estimated operations
	MemoryUsage       int64         `json:"memory_usage"`       // Bytes
	TimeEstimate      time.Duration `json:"time_estimate"`      // At standard hardware
	BitsOfSecurity    int           `json:"bits_of_security"`   // Equivalent security bits
	ResistanceProfile string        `json:"resistance_profile"` // Attack resistance description
}

// CryptographicValidator provides validation for different KDF types
type CryptographicValidator struct {
	scryptRanges map[string]ParameterRange
	pbkdf2Ranges map[string]ParameterRange
}

// NewCryptographicValidator creates a new cryptographic validator
func NewCryptographicValidator() *CryptographicValidator {
	return &CryptographicValidator{
		scryptRanges: initScryptRanges(),
		pbkdf2Ranges: initPBKDF2Ranges(),
	}
}

// initScryptRanges initializes the parameter ranges for scrypt
func initScryptRanges() map[string]ParameterRange {
	return map[string]ParameterRange{
		"n": {
			Min:         1024,
			Max:         67108864, // 2^26
			Recommended: 262144,   // 2^18
			Description: "CPU/memory cost parameter (must be power of 2)",
		},
		"r": {
			Min:         1,
			Max:         1024,
			Recommended: 8,
			Description: "Block size parameter",
		},
		"p": {
			Min:         1,
			Max:         16,
			Recommended: 1,
			Description: "Parallelization parameter",
		},
		"dklen": {
			Min:         16,
			Max:         128,
			Recommended: 32,
			Description: "Derived key length in bytes",
		},
	}
}

// initPBKDF2Ranges initializes the parameter ranges for PBKDF2
func initPBKDF2Ranges() map[string]ParameterRange {
	return map[string]ParameterRange{
		"c": {
			Min:         1000,
			Max:         10000000,
			Recommended: 100000,
			Description: "Iteration count",
		},
		"dklen": {
			Min:         16,
			Max:         128,
			Recommended: 32,
			Description: "Derived key length in bytes",
		},
	}
}

// ValidateScryptParams validates scrypt parameters and provides security analysis
func (cv *CryptographicValidator) ValidateScryptParams(params map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:         true,
		SecurityLevel: SecurityLevelMedium,
		Issues:        []string{},
		Warnings:      []string{},
		Suggestions:   []string{},
		Ranges:        cv.scryptRanges,
	}

	// Extract parameters
	n, nOk := getIntParam(params, "n")
	r, rOk := getIntParam(params, "r")
	p, pOk := getIntParam(params, "p")
	dklen, dklenOk := getIntParam(params, "dklen")

	// Check required parameters
	if !nOk {
		result.Valid = false
		result.Issues = append(result.Issues, "missing required parameter: n")
	}
	if !rOk {
		result.Valid = false
		result.Issues = append(result.Issues, "missing required parameter: r")
	}
	if !pOk {
		result.Valid = false
		result.Issues = append(result.Issues, "missing required parameter: p")
	}
	if !dklenOk {
		result.Valid = false
		result.Issues = append(result.Issues, "missing required parameter: dklen")
	}

	if !result.Valid {
		return result
	}

	// Validate N parameter (must be power of 2)
	if n <= 0 {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("N parameter must be positive, got %d", n))
	} else if n&(n-1) != 0 {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("N parameter must be a power of 2, got %d", n))
	} else {
		cv.validateIntRange(result, "n", n, cv.scryptRanges["n"])
	}

	// Validate other parameters
	cv.validateIntRange(result, "r", r, cv.scryptRanges["r"])
	cv.validateIntRange(result, "p", p, cv.scryptRanges["p"])
	cv.validateIntRange(result, "dklen", dklen, cv.scryptRanges["dklen"])

	// Check memory usage
	memoryUsage := int64(128) * int64(r) * int64(n) * int64(p)
	if memoryUsage > 2*1024*1024*1024 { // 2GB limit
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("memory usage too high: %d bytes (limit: 2GB)", memoryUsage))
		result.Suggestions = append(result.Suggestions, "reduce N, r, or p parameters to lower memory usage")
	} else if memoryUsage > 256*1024*1024 { // 256MB warning
		result.Warnings = append(result.Warnings, fmt.Sprintf("high memory usage: %d bytes", memoryUsage))
	}

	// Perform security analysis
	if result.Valid {
		result.Analysis = cv.analyzeScryptSecurity(n, r, p, dklen)
		result.SecurityLevel = cv.determineScryptSecurityLevel(n, r, p)

		// Add security-based suggestions
		switch result.SecurityLevel {
		case SecurityLevelLow:
			result.Suggestions = append(result.Suggestions, "increase N parameter for better security (recommended: 262144)")
		case SecurityLevelMedium:
			result.Suggestions = append(result.Suggestions, "consider increasing N parameter for high security applications")
		}
	}

	return result
}

// ValidatePBKDF2Params validates PBKDF2 parameters and provides security analysis
func (cv *CryptographicValidator) ValidatePBKDF2Params(params map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:         true,
		SecurityLevel: SecurityLevelMedium,
		Issues:        []string{},
		Warnings:      []string{},
		Suggestions:   []string{},
		Ranges:        cv.pbkdf2Ranges,
	}

	// Extract parameters
	c, cOk := getIntParam(params, "c")
	dklen, dklenOk := getIntParam(params, "dklen")
	prf, prfOk := getStringParam(params, "prf")

	// Check required parameters
	if !cOk {
		result.Valid = false
		result.Issues = append(result.Issues, "missing required parameter: c")
	}
	if !dklenOk {
		result.Valid = false
		result.Issues = append(result.Issues, "missing required parameter: dklen")
	}

	if !result.Valid {
		return result
	}

	// Validate parameters
	cv.validateIntRange(result, "c", c, cv.pbkdf2Ranges["c"])
	cv.validateIntRange(result, "dklen", dklen, cv.pbkdf2Ranges["dklen"])

	// Validate PRF if provided
	if prfOk && prf != "" {
		validPRFs := map[string]bool{
			"hmac-sha256": true,
			"hmac-sha512": true,
		}
		if !validPRFs[prf] {
			result.Valid = false
			result.Issues = append(result.Issues, fmt.Sprintf("invalid PRF: %s (supported: hmac-sha256, hmac-sha512)", prf))
		}
	}

	// Perform security analysis
	if result.Valid {
		hashFunc := "sha256"
		if prfOk && prf == "hmac-sha512" {
			hashFunc = "sha512"
		}
		result.Analysis = cv.analyzePBKDF2Security(c, dklen, hashFunc)
		result.SecurityLevel = cv.determinePBKDF2SecurityLevel(c)

		// Add security-based suggestions
		if result.SecurityLevel == SecurityLevelLow {
			result.Suggestions = append(result.Suggestions, "increase iteration count for better security (recommended: 100000)")
		} else if result.SecurityLevel == SecurityLevelMedium && c < 200000 {
			result.Suggestions = append(result.Suggestions, "consider increasing iteration count for high security applications")
		}
	}

	return result
}

// validateIntRange validates an integer parameter against its range
func (cv *CryptographicValidator) validateIntRange(result *ValidationResult, paramName string, value int, paramRange ParameterRange) {
	min := paramRange.Min.(int)
	max := paramRange.Max.(int)
	recommended := paramRange.Recommended.(int)

	if value < min {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("%s parameter too low: %d (minimum: %d)", paramName, value, min))
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("set %s to at least %d", paramName, min))
	} else if value > max {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("%s parameter too high: %d (maximum: %d)", paramName, value, max))
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("set %s to at most %d", paramName, max))
	} else if value < recommended {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s parameter below recommended value: %d (recommended: %d)", paramName, value, recommended))
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("consider setting %s to %d for better security", paramName, recommended))
	}
}

// analyzeScryptSecurity performs detailed security analysis for scrypt parameters
func (cv *CryptographicValidator) analyzeScryptSecurity(n, r, p, dklen int) *CryptographicAnalysis {
	// Calculate computational cost (approximate operations)
	computationalCost := float64(n) * float64(r) * float64(p) * 2.0

	// Calculate memory usage
	memoryUsage := int64(128) * int64(r) * int64(n) * int64(p)

	// Estimate time on standard hardware (rough approximation)
	// Assuming ~1M scrypt operations per second on standard CPU
	timeEstimate := time.Duration(computationalCost/1000000) * time.Second

	// Calculate equivalent security bits (logarithmic scale)
	bitsOfSecurity := int(math.Log2(computationalCost) / 2)
	if bitsOfSecurity > 128 {
		bitsOfSecurity = 128 // Cap at 128 bits
	}

	// Determine resistance profile
	resistanceProfile := "Memory-hard function resistant to ASIC attacks"
	if memoryUsage < 32*1024*1024 { // Less than 32MB
		resistanceProfile += " (low memory usage may reduce ASIC resistance)"
	}

	return &CryptographicAnalysis{
		ComputationalCost: computationalCost,
		MemoryUsage:       memoryUsage,
		TimeEstimate:      timeEstimate,
		BitsOfSecurity:    bitsOfSecurity,
		ResistanceProfile: resistanceProfile,
	}
}

// analyzePBKDF2Security performs detailed security analysis for PBKDF2 parameters
func (cv *CryptographicValidator) analyzePBKDF2Security(c, dklen int, hashFunc string) *CryptographicAnalysis {
	// Calculate computational cost (iterations * hash operations)
	computationalCost := float64(c) * 2.0 // Approximate hash operations per iteration

	// PBKDF2 has minimal memory usage
	memoryUsage := int64(dklen + 64) // Output + hash state

	// Estimate time on standard hardware
	// Assuming ~100K PBKDF2 iterations per second on standard CPU
	timeEstimate := time.Duration(c/100000) * time.Second

	// Calculate equivalent security bits
	bitsOfSecurity := int(math.Log2(float64(c)) * 0.8) // PBKDF2 is less secure than scrypt
	if bitsOfSecurity > 80 {
		bitsOfSecurity = 80 // PBKDF2 practical limit
	}

	// Determine resistance profile
	resistanceProfile := fmt.Sprintf("CPU-bound function using %s", hashFunc)
	if c < 10000 {
		resistanceProfile += " (vulnerable to GPU/ASIC attacks)"
	} else if c < 100000 {
		resistanceProfile += " (moderate GPU resistance)"
	} else {
		resistanceProfile += " (good GPU resistance)"
	}

	return &CryptographicAnalysis{
		ComputationalCost: computationalCost,
		MemoryUsage:       memoryUsage,
		TimeEstimate:      timeEstimate,
		BitsOfSecurity:    bitsOfSecurity,
		ResistanceProfile: resistanceProfile,
	}
}

// determineScryptSecurityLevel determines the security level based on scrypt parameters
func (cv *CryptographicValidator) determineScryptSecurityLevel(n, r, p int) SecurityLevel {
	// Calculate total work factor
	workFactor := float64(n) * float64(r) * float64(p)

	if workFactor < 16384 { // Very weak
		return SecurityLevelLow
	} else if workFactor < 1048576 { // Weak to moderate
		return SecurityLevelMedium
	} else if workFactor < 8388608 { // Strong
		return SecurityLevelHigh
	} else { // Very strong
		return SecurityLevelVeryHigh
	}
}

// determinePBKDF2SecurityLevel determines the security level based on PBKDF2 parameters
func (cv *CryptographicValidator) determinePBKDF2SecurityLevel(c int) SecurityLevel {
	if c < 10000 {
		return SecurityLevelLow
	} else if c < 200000 {
		return SecurityLevelMedium
	} else if c < 500000 {
		return SecurityLevelHigh
	} else {
		return SecurityLevelVeryHigh
	}
}

// GetParameterOptimizationSuggestions provides optimization suggestions for parameters
func (cv *CryptographicValidator) GetParameterOptimizationSuggestions(kdfType string, currentParams map[string]interface{}, targetSecurity SecurityLevel) []string {
	suggestions := []string{}

	switch kdfType {
	case "scrypt":
		n, _ := getIntParam(currentParams, "n")
		r, _ := getIntParam(currentParams, "r")
		p, _ := getIntParam(currentParams, "p")

		currentLevel := cv.determineScryptSecurityLevel(n, r, p)
		if currentLevel < targetSecurity {
			switch targetSecurity {
			case SecurityLevelMedium:
				suggestions = append(suggestions, "increase N to 16384 for medium security")
			case SecurityLevelHigh:
				suggestions = append(suggestions, "increase N to 262144 for high security")
			case SecurityLevelVeryHigh:
				suggestions = append(suggestions, "increase N to 1048576 for very high security")
			}
		}

	case "pbkdf2":
		c, _ := getIntParam(currentParams, "c")
		currentLevel := cv.determinePBKDF2SecurityLevel(c)
		if currentLevel < targetSecurity {
			switch targetSecurity {
			case SecurityLevelMedium:
				suggestions = append(suggestions, "increase iterations to 50000 for medium security")
			case SecurityLevelHigh:
				suggestions = append(suggestions, "increase iterations to 200000 for high security")
			case SecurityLevelVeryHigh:
				suggestions = append(suggestions, "increase iterations to 500000 for very high security")
			}
		}
	}

	return suggestions
}

// Helper functions for parameter extraction
func getIntParam(params map[string]interface{}, key string) (int, bool) {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		case float32:
			return int(v), true
		}
	}
	return 0, false
}

func getStringParam(params map[string]interface{}, key string) (string, bool) {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

// Global validator instance
var defaultValidator = NewCryptographicValidator()

// ValidateKDFParameters validates KDF parameters using the default validator
func ValidateKDFParameters(kdfType string, params map[string]interface{}) *ValidationResult {
	switch kdfType {
	case "scrypt":
		return defaultValidator.ValidateScryptParams(params)
	case "pbkdf2":
		return defaultValidator.ValidatePBKDF2Params(params)
	default:
		return &ValidationResult{
			Valid:  false,
			Issues: []string{fmt.Sprintf("unsupported KDF type: %s", kdfType)},
		}
	}
}

// GetParameterRanges returns the valid parameter ranges for a KDF type
func GetParameterRanges(kdfType string) map[string]ParameterRange {
	switch kdfType {
	case "scrypt":
		return defaultValidator.scryptRanges
	case "pbkdf2":
		return defaultValidator.pbkdf2Ranges
	default:
		return nil
	}
}

// OptimizeParameters provides parameter optimization suggestions
func OptimizeParameters(kdfType string, currentParams map[string]interface{}, targetSecurity SecurityLevel) []string {
	return defaultValidator.GetParameterOptimizationSuggestions(kdfType, currentParams, targetSecurity)
}
