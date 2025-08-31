package kdf

import (
	"fmt"
	"time"
)

// KDFCompatibilityAnalyzer analyzes keystore compatibility and provides security assessments
type KDFCompatibilityAnalyzer struct {
	service *UniversalKDFService
}

// NewKDFCompatibilityAnalyzer creates a new compatibility analyzer
func NewKDFCompatibilityAnalyzer(service *UniversalKDFService) *KDFCompatibilityAnalyzer {
	return &KDFCompatibilityAnalyzer{
		service: service,
	}
}

// AnalyzeKeystore analyzes a keystore's KDF parameters for compatibility and security
func (analyzer *KDFCompatibilityAnalyzer) AnalyzeKeystore(crypto *CryptoParams) (*CompatibilityReport, error) {
	if crypto == nil {
		return nil, NewKDFError("validation", "", "", nil, nil, "crypto parameters cannot be nil")
	}

	// Normalize KDF name
	normalizedKDF := analyzer.service.normalizeKDFName(crypto.KDF)

	// Check if KDF is supported
	if !analyzer.service.IsKDFSupported(crypto.KDF) {
		return &CompatibilityReport{
			Compatible:    false,
			KDFType:       crypto.KDF,
			NormalizedKDF: normalizedKDF,
			Parameters:    crypto.KDFParams,
			SecurityLevel: SecurityLevelLow,
			Issues:        []string{fmt.Sprintf("Unsupported KDF type: %s", crypto.KDF)},
			Warnings:      []string{},
			Suggestions:   []string{"Use 'scrypt', 'pbkdf2', 'pbkdf2-sha256', or 'pbkdf2-sha512'"},
		}, nil
	}

	// Validate parameters
	var issues []string
	var warnings []string
	var suggestions []string

	if err := analyzer.service.ValidateParams(crypto.KDF, crypto.KDFParams); err != nil {
		if kdfErr, ok := err.(*KDFError); ok {
			issues = append(issues, kdfErr.Message)
			suggestions = append(suggestions, kdfErr.GetSuggestions()...)
		} else {
			issues = append(issues, err.Error())
		}
	}

	// Perform security analysis
	securityAnalysis := analyzer.analyzeSecurityLevel(normalizedKDF, crypto.KDFParams)

	// Check client compatibility
	clientCompatibility := analyzer.analyzeClientCompatibility(normalizedKDF, crypto.KDFParams)

	// Add security-based warnings and suggestions
	securityWarnings, securitySuggestions := analyzer.generateSecurityRecommendations(securityAnalysis)
	warnings = append(warnings, securityWarnings...)
	suggestions = append(suggestions, securitySuggestions...)

	// Add client-specific warnings
	clientWarnings := analyzer.generateClientWarnings(clientCompatibility)
	warnings = append(warnings, clientWarnings...)

	return &CompatibilityReport{
		Compatible:    len(issues) == 0,
		KDFType:       crypto.KDF,
		NormalizedKDF: normalizedKDF,
		Parameters:    crypto.KDFParams,
		SecurityLevel: securityAnalysis.Level,
		Issues:        issues,
		Warnings:      warnings,
		Suggestions:   suggestions,
	}, nil
}

// analyzeSecurityLevel performs security analysis of KDF parameters
func (analyzer *KDFCompatibilityAnalyzer) analyzeSecurityLevel(kdfType string, params map[string]interface{}) *SecurityAnalysis {
	switch kdfType {
	case "scrypt":
		return analyzer.analyzeScryptSecurity(params)
	case "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512":
		return analyzer.analyzePBKDF2Security(kdfType, params)
	default:
		return &SecurityAnalysis{
			Level:               SecurityLevelLow,
			ComputationalCost:   0,
			MemoryUsage:         0,
			Recommendations:     []string{"Use a supported KDF algorithm"},
			ClientCompatibility: make(map[string]bool),
		}
	}
}

// analyzeScryptSecurity analyzes scrypt parameters for security level
func (analyzer *KDFCompatibilityAnalyzer) analyzeScryptSecurity(params map[string]interface{}) *SecurityAnalysis {
	// Extract parameters with defaults
	n := analyzer.getIntParam(params, "n", 16384)
	r := analyzer.getIntParam(params, "r", 8)
	p := analyzer.getIntParam(params, "p", 1)

	// Calculate computational cost (operations)
	computationalCost := float64(n * r * p)

	// Calculate memory usage (bytes)
	// Scrypt memory usage: 128 * N * r bytes
	memoryUsage := int64(128 * n * r)

	// Determine security level based on parameters
	var level SecurityLevel
	var recommendations []string

	// Security thresholds based on industry standards
	if n >= 262144 && r >= 8 && p >= 1 { // N >= 2^18
		level = SecurityLevelVeryHigh
	} else if n >= 65536 && r >= 8 && p >= 1 { // N >= 2^16
		level = SecurityLevelHigh
	} else if n >= 16384 && r >= 8 && p >= 1 { // N >= 2^14
		level = SecurityLevelMedium
	} else {
		level = SecurityLevelLow
		recommendations = append(recommendations, "Increase N parameter to at least 16384 for better security")
	}

	// Add specific recommendations
	if n < 16384 {
		recommendations = append(recommendations, fmt.Sprintf("N parameter (%d) is too low, recommend at least 16384", n))
	}
	if r < 8 {
		recommendations = append(recommendations, fmt.Sprintf("r parameter (%d) is too low, recommend at least 8", r))
	}
	if p < 1 {
		recommendations = append(recommendations, fmt.Sprintf("p parameter (%d) is too low, recommend at least 1", p))
	}

	// Memory usage warnings
	if memoryUsage > 2*1024*1024*1024 { // > 2GB
		recommendations = append(recommendations, "Memory usage is very high, consider reducing N or r parameters")
	} else if memoryUsage > 512*1024*1024 { // > 512MB
		recommendations = append(recommendations, "Memory usage is high, monitor system resources")
	}

	// Check if N is power of 2
	if n&(n-1) != 0 {
		recommendations = append(recommendations, "N parameter should be a power of 2 for optimal performance")
	}

	return &SecurityAnalysis{
		Level:               level,
		ComputationalCost:   computationalCost,
		MemoryUsage:         memoryUsage,
		Recommendations:     recommendations,
		ClientCompatibility: analyzer.getScryptClientCompatibility(n, r, p),
	}
}

// analyzePBKDF2Security analyzes PBKDF2 parameters for security level
func (analyzer *KDFCompatibilityAnalyzer) analyzePBKDF2Security(kdfType string, params map[string]interface{}) *SecurityAnalysis {
	// Extract parameters with defaults
	c := analyzer.getIntParam(params, "c", 100000)
	prf := analyzer.getStringParam(params, "prf", "hmac-sha256")

	// Calculate computational cost (iterations)
	computationalCost := float64(c)

	// PBKDF2 has minimal memory usage
	memoryUsage := int64(1024) // ~1KB

	// Determine security level based on iteration count and PRF
	var level SecurityLevel
	var recommendations []string

	// Security thresholds based on OWASP recommendations
	if c >= 600000 {
		level = SecurityLevelVeryHigh
	} else if c >= 310000 {
		level = SecurityLevelHigh
	} else if c >= 120000 {
		level = SecurityLevelMedium
	} else {
		level = SecurityLevelLow
		recommendations = append(recommendations, "Increase iteration count to at least 120000 for better security")
	}

	// Add specific recommendations
	if c < 100000 {
		recommendations = append(recommendations, fmt.Sprintf("Iteration count (%d) is too low, recommend at least 100000", c))
	}

	// PRF-specific recommendations
	switch prf {
	case "hmac-sha1":
		recommendations = append(recommendations, "SHA-1 is deprecated, consider using SHA-256 or SHA-512")
		if level > SecurityLevelMedium {
			level = SecurityLevelMedium // Downgrade due to weak hash
		}
	case "hmac-sha256":
		// Good choice, no additional recommendations
	case "hmac-sha512":
		// Excellent choice, no additional recommendations
	default:
		recommendations = append(recommendations, fmt.Sprintf("Unknown PRF '%s', recommend 'hmac-sha256' or 'hmac-sha512'", prf))
	}

	return &SecurityAnalysis{
		Level:               level,
		ComputationalCost:   computationalCost,
		MemoryUsage:         memoryUsage,
		Recommendations:     recommendations,
		ClientCompatibility: analyzer.getPBKDF2ClientCompatibility(c, prf),
	}
}

// analyzeClientCompatibility checks compatibility with different Ethereum clients
func (analyzer *KDFCompatibilityAnalyzer) analyzeClientCompatibility(kdfType string, params map[string]interface{}) map[string]bool {
	switch kdfType {
	case "scrypt":
		n := analyzer.getIntParam(params, "n", 16384)
		r := analyzer.getIntParam(params, "r", 8)
		p := analyzer.getIntParam(params, "p", 1)
		return analyzer.getScryptClientCompatibility(n, r, p)
	case "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512":
		c := analyzer.getIntParam(params, "c", 100000)
		prf := analyzer.getStringParam(params, "prf", "hmac-sha256")
		return analyzer.getPBKDF2ClientCompatibility(c, prf)
	default:
		// Unknown KDF, assume incompatible with all clients
		return map[string]bool{
			"geth":    false,
			"besu":    false,
			"anvil":   false,
			"reth":    false,
			"firefly": false,
		}
	}
}

// getScryptClientCompatibility returns client compatibility for scrypt parameters
func (analyzer *KDFCompatibilityAnalyzer) getScryptClientCompatibility(n, r, p int) map[string]bool {
	// Most clients support standard scrypt parameters
	// Some have limitations on very high memory usage
	memoryUsage := 128 * n * r

	return map[string]bool{
		"geth":    n >= 1024 && n <= 1048576 && r >= 1 && r <= 32 && p >= 1 && p <= 16,
		"besu":    n >= 1024 && n <= 1048576 && r >= 1 && r <= 32 && p >= 1 && p <= 16,
		"anvil":   n >= 1024 && n <= 262144 && r >= 1 && r <= 16 && p >= 1 && p <= 8, // More conservative
		"reth":    n >= 1024 && n <= 1048576 && r >= 1 && r <= 32 && p >= 1 && p <= 16,
		"firefly": memoryUsage <= 512*1024*1024, // 512MB limit for enterprise use
	}
}

// getPBKDF2ClientCompatibility returns client compatibility for PBKDF2 parameters
func (analyzer *KDFCompatibilityAnalyzer) getPBKDF2ClientCompatibility(c int, prf string) map[string]bool {
	// Check PRF support
	prfSupported := map[string]map[string]bool{
		"geth":    {"hmac-sha256": true, "hmac-sha512": true, "hmac-sha1": true},
		"besu":    {"hmac-sha256": true, "hmac-sha512": true, "hmac-sha1": false},
		"anvil":   {"hmac-sha256": true, "hmac-sha512": false, "hmac-sha1": false},
		"reth":    {"hmac-sha256": true, "hmac-sha512": true, "hmac-sha1": false},
		"firefly": {"hmac-sha256": true, "hmac-sha512": true, "hmac-sha1": false},
	}

	compatibility := make(map[string]bool)
	for client, prfs := range prfSupported {
		// Check iteration count limits (most clients support up to 10M iterations)
		iterationOk := c >= 1000 && c <= 10000000
		prfOk := prfs[prf]
		compatibility[client] = iterationOk && prfOk
	}

	return compatibility
}

// generateSecurityRecommendations generates warnings and suggestions based on security analysis
func (analyzer *KDFCompatibilityAnalyzer) generateSecurityRecommendations(analysis *SecurityAnalysis) ([]string, []string) {
	var warnings []string
	var suggestions []string

	// Security level warnings
	switch analysis.Level {
	case SecurityLevelLow:
		warnings = append(warnings, "Security level is LOW - parameters provide insufficient protection")
		suggestions = append(suggestions, "Increase KDF parameters for better security")
	case SecurityLevelMedium:
		warnings = append(warnings, "Security level is MEDIUM - consider stronger parameters for sensitive applications")
	}

	// Memory usage warnings
	if analysis.MemoryUsage > 1024*1024*1024 { // > 1GB
		warnings = append(warnings, fmt.Sprintf("High memory usage: %s", analyzer.formatBytes(analysis.MemoryUsage)))
	}

	// Add analysis-specific recommendations
	suggestions = append(suggestions, analysis.Recommendations...)

	return warnings, suggestions
}

// generateClientWarnings generates warnings based on client compatibility
func (analyzer *KDFCompatibilityAnalyzer) generateClientWarnings(compatibility map[string]bool) []string {
	var warnings []string
	var incompatibleClients []string

	for client, compatible := range compatibility {
		if !compatible {
			incompatibleClients = append(incompatibleClients, client)
		}
	}

	if len(incompatibleClients) > 0 {
		warnings = append(warnings, fmt.Sprintf("Incompatible with clients: %v", incompatibleClients))
	}

	return warnings
}

// Helper functions for parameter extraction
func (analyzer *KDFCompatibilityAnalyzer) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if val, exists := params[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			// Try to parse string as int
			if parsed, err := fmt.Sscanf(v, "%d", &defaultValue); err == nil && parsed == 1 {
				return defaultValue
			}
		}
	}
	return defaultValue
}

func (analyzer *KDFCompatibilityAnalyzer) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if val, exists := params[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// formatBytes formats byte count as human-readable string
func (analyzer *KDFCompatibilityAnalyzer) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// EstimateDerivationTime estimates the time required for key derivation
func (analyzer *KDFCompatibilityAnalyzer) EstimateDerivationTime(kdfType string, params map[string]interface{}) time.Duration {
	switch kdfType {
	case "scrypt":
		n := analyzer.getIntParam(params, "n", 16384)
		r := analyzer.getIntParam(params, "r", 8)
		p := analyzer.getIntParam(params, "p", 1)

		// Rough estimation based on typical hardware performance
		// These are conservative estimates for average consumer hardware
		operations := float64(n * r * p)
		operationsPerSecond := 50000.0 // Conservative estimate

		seconds := operations / operationsPerSecond
		return time.Duration(seconds * float64(time.Second))

	case "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512":
		c := analyzer.getIntParam(params, "c", 100000)

		// PBKDF2 performance varies by hash function
		prf := analyzer.getStringParam(params, "prf", "hmac-sha256")
		var iterationsPerSecond float64

		switch prf {
		case "hmac-sha256":
			iterationsPerSecond = 1000000.0 // ~1M iterations/second
		case "hmac-sha512":
			iterationsPerSecond = 500000.0 // ~500K iterations/second
		default:
			iterationsPerSecond = 800000.0 // Conservative default
		}

		seconds := float64(c) / iterationsPerSecond
		return time.Duration(seconds * float64(time.Second))

	default:
		return time.Second // Unknown KDF, return default estimate
	}
}

// GetOptimizedParams suggests optimized parameters for a given security level and constraints
func (analyzer *KDFCompatibilityAnalyzer) GetOptimizedParams(kdfType string, securityLevel SecurityLevel, maxMemoryMB int64) (map[string]interface{}, error) {
	normalizedKDF := analyzer.service.normalizeKDFName(kdfType)

	switch normalizedKDF {
	case "scrypt":
		return analyzer.getOptimizedScryptParams(securityLevel, maxMemoryMB), nil
	case "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512":
		return analyzer.getOptimizedPBKDF2Params(normalizedKDF, securityLevel), nil
	default:
		return nil, NewKDFError("compatibility", kdfType, "", kdfType, "supported KDF type",
			fmt.Sprintf("Cannot optimize parameters for unsupported KDF: %s", kdfType))
	}
}

// getOptimizedScryptParams returns optimized scrypt parameters
func (analyzer *KDFCompatibilityAnalyzer) getOptimizedScryptParams(securityLevel SecurityLevel, maxMemoryMB int64) map[string]interface{} {
	maxMemoryBytes := maxMemoryMB * 1024 * 1024

	var n, r, p int

	switch securityLevel {
	case SecurityLevelLow:
		n, r, p = 16384, 8, 1 // 16MB memory
	case SecurityLevelMedium:
		n, r, p = 65536, 8, 1 // 64MB memory
	case SecurityLevelHigh:
		n, r, p = 262144, 8, 1 // 256MB memory
	case SecurityLevelVeryHigh:
		n, r, p = 1048576, 8, 1 // 1GB memory
	default:
		n, r, p = 65536, 8, 1 // Default to medium
	}

	// Adjust for memory constraints
	memoryUsage := int64(128 * n * r)
	if maxMemoryBytes > 0 && memoryUsage > maxMemoryBytes {
		// Reduce N to fit memory constraint while maintaining power of 2
		for memoryUsage > maxMemoryBytes && n > 1024 {
			n /= 2
			memoryUsage = int64(128 * n * r)
		}
	}

	return map[string]interface{}{
		"n":     n,
		"r":     r,
		"p":     p,
		"dklen": 32,
	}
}

// getOptimizedPBKDF2Params returns optimized PBKDF2 parameters
func (analyzer *KDFCompatibilityAnalyzer) getOptimizedPBKDF2Params(kdfType string, securityLevel SecurityLevel) map[string]interface{} {
	var c int
	var prf string

	// Set PRF based on KDF type
	switch kdfType {
	case "pbkdf2-sha256":
		prf = "hmac-sha256"
	case "pbkdf2-sha512":
		prf = "hmac-sha512"
	default:
		prf = "hmac-sha256" // Default
	}

	// Set iteration count based on security level
	switch securityLevel {
	case SecurityLevelLow:
		c = 120000
	case SecurityLevelMedium:
		c = 310000
	case SecurityLevelHigh:
		c = 600000
	case SecurityLevelVeryHigh:
		c = 1000000
	default:
		c = 310000 // Default to medium
	}

	return map[string]interface{}{
		"c":     c,
		"dklen": 32,
		"prf":   prf,
	}
}
