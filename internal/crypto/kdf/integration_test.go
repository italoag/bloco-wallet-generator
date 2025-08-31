package kdf

import (
	"testing"
)

// TestKDFCompatibilityAnalyzer_Integration demonstrates the complete workflow
// of analyzing keystore compatibility and security
func TestKDFCompatibilityAnalyzer_Integration(t *testing.T) {
	// Create service and analyzer
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test Case 1: High-security scrypt configuration
	t.Run("HighSecurityScrypt", func(t *testing.T) {
		crypto := &CryptoParams{
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     262144, // 2^18 - Very High security
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		}

		report, err := analyzer.AnalyzeKeystore(crypto)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify compatibility report structure
		if !report.Compatible {
			t.Error("Expected high-security scrypt to be compatible")
		}

		if report.SecurityLevel != SecurityLevelVeryHigh {
			t.Errorf("Expected Very High security level, got %s", report.SecurityLevel)
		}

		if report.KDFType != "scrypt" {
			t.Errorf("Expected KDF type 'scrypt', got %s", report.KDFType)
		}

		if report.NormalizedKDF != "scrypt" {
			t.Errorf("Expected normalized KDF 'scrypt', got %s", report.NormalizedKDF)
		}

		// Should have minimal issues/warnings for high-security config
		if len(report.Issues) > 0 {
			t.Errorf("Expected no issues for high-security config, got: %v", report.Issues)
		}

		t.Logf("High-security scrypt report: Compatible=%v, Security=%s, Issues=%d, Warnings=%d",
			report.Compatible, report.SecurityLevel, len(report.Issues), len(report.Warnings))
	})

	// Test Case 2: Low-security configuration with recommendations
	t.Run("LowSecurityScrypt", func(t *testing.T) {
		crypto := &CryptoParams{
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     1024, // Very low
				"r":     4,    // Low
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		}

		report, err := analyzer.AnalyzeKeystore(crypto)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should still be compatible but with low security
		if !report.Compatible {
			t.Error("Expected low-security scrypt to still be compatible")
		}

		if report.SecurityLevel != SecurityLevelLow {
			t.Errorf("Expected Low security level, got %s", report.SecurityLevel)
		}

		// Should have warnings and suggestions
		if len(report.Warnings) == 0 {
			t.Error("Expected warnings for low-security configuration")
		}

		if len(report.Suggestions) == 0 {
			t.Error("Expected suggestions for low-security configuration")
		}

		t.Logf("Low-security scrypt report: Compatible=%v, Security=%s, Warnings=%d, Suggestions=%d",
			report.Compatible, report.SecurityLevel, len(report.Warnings), len(report.Suggestions))
	})

	// Test Case 3: PBKDF2 with deprecated hash function
	t.Run("PBKDF2WithDeprecatedHash", func(t *testing.T) {
		crypto := &CryptoParams{
			KDF: "pbkdf2",
			KDFParams: map[string]interface{}{
				"c":     600000, // High iteration count
				"dklen": 32,
				"prf":   "hmac-sha1", // Deprecated hash function
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		}

		report, err := analyzer.AnalyzeKeystore(crypto)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should be incompatible due to client compatibility issues with SHA-1
		if report.Compatible {
			t.Error("Expected PBKDF2 with SHA-1 to be incompatible due to client issues")
		}

		// Security level should be downgraded due to weak hash
		if report.SecurityLevel == SecurityLevelVeryHigh {
			t.Error("Expected security level to be downgraded due to weak hash function")
		}

		// Should have recommendations about SHA-1
		foundSHA1Warning := false
		for _, suggestion := range report.Suggestions {
			if suggestion == "SHA-1 is deprecated, consider using SHA-256 or SHA-512" {
				foundSHA1Warning = true
				break
			}
		}

		if !foundSHA1Warning {
			t.Error("Expected recommendation about deprecated SHA-1")
		}

		t.Logf("PBKDF2 SHA-1 report: Compatible=%v, Security=%s, Suggestions=%v",
			report.Compatible, report.SecurityLevel, report.Suggestions)
	})

	// Test Case 4: Client compatibility analysis
	t.Run("ClientCompatibilityAnalysis", func(t *testing.T) {
		// Test parameters that should be incompatible with some clients
		crypto := &CryptoParams{
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     1048576, // Very high N
				"r":     16,      // High r
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		}

		report, err := analyzer.AnalyzeKeystore(crypto)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should have client compatibility warnings
		foundClientWarning := false
		for _, warning := range report.Warnings {
			if len(warning) > 0 && warning[:13] == "Incompatible " {
				foundClientWarning = true
				break
			}
		}

		// High memory usage should cause some client incompatibility
		if !foundClientWarning {
			t.Error("Expected client compatibility warnings for high memory usage")
		}

		t.Logf("High memory scrypt report: Warnings=%v", report.Warnings)
	})

	// Test Case 5: Parameter optimization
	t.Run("ParameterOptimization", func(t *testing.T) {
		// Test getting optimized parameters for different security levels
		securityLevels := []SecurityLevel{
			SecurityLevelLow,
			SecurityLevelMedium,
			SecurityLevelHigh,
			SecurityLevelVeryHigh,
		}

		for _, level := range securityLevels {
			params, err := analyzer.GetOptimizedParams("scrypt", level, 256) // 256MB limit
			if err != nil {
				t.Fatalf("Failed to get optimized params for %s: %v", level, err)
			}

			// Verify parameters are valid
			n, ok := params["n"].(int)
			if !ok || n < 1024 {
				t.Errorf("Invalid N parameter for %s: %v", level, params["n"])
			}

			r, ok := params["r"].(int)
			if !ok || r < 1 {
				t.Errorf("Invalid r parameter for %s: %v", level, params["r"])
			}

			// Check memory constraint
			memoryUsage := 128 * n * r
			maxMemory := 256 * 1024 * 1024 // 256MB
			if memoryUsage > maxMemory {
				t.Errorf("Memory usage %d exceeds limit %d for %s", memoryUsage, maxMemory, level)
			}

			t.Logf("Optimized %s params: n=%d, r=%d, memory=%s",
				level, n, r, analyzer.formatBytes(int64(memoryUsage)))
		}
	})

	// Test Case 6: Time estimation
	t.Run("TimeEstimation", func(t *testing.T) {
		testCases := []struct {
			name   string
			kdf    string
			params map[string]interface{}
		}{
			{
				name: "Fast scrypt",
				kdf:  "scrypt",
				params: map[string]interface{}{
					"n": 16384,
					"r": 8,
					"p": 1,
				},
			},
			{
				name: "Slow scrypt",
				kdf:  "scrypt",
				params: map[string]interface{}{
					"n": 262144,
					"r": 8,
					"p": 1,
				},
			},
			{
				name: "Fast PBKDF2",
				kdf:  "pbkdf2",
				params: map[string]interface{}{
					"c":   100000,
					"prf": "hmac-sha256",
				},
			},
			{
				name: "Slow PBKDF2",
				kdf:  "pbkdf2",
				params: map[string]interface{}{
					"c":   1000000,
					"prf": "hmac-sha256",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				duration := analyzer.EstimateDerivationTime(tc.kdf, tc.params)
				if duration <= 0 {
					t.Errorf("Expected positive duration for %s, got %v", tc.name, duration)
				}

				t.Logf("%s estimated time: %v", tc.name, duration)
			})
		}
	})
}

// TestRequirementsCoverage verifies that all requirements are met
func TestRequirementsCoverage(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Requirement 4.1: Compatibility analysis reporting
	t.Run("Requirement_4_1_CompatibilityReporting", func(t *testing.T) {
		crypto := &CryptoParams{
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     65536,
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		}

		report, err := analyzer.AnalyzeKeystore(crypto)
		if err != nil {
			t.Fatalf("Failed to analyze keystore: %v", err)
		}

		// Verify report structure matches requirements
		if report.KDFType == "" {
			t.Error("Report missing KDF type")
		}
		if report.NormalizedKDF == "" {
			t.Error("Report missing normalized KDF")
		}
		if report.Parameters == nil {
			t.Error("Report missing parameters")
		}
		if report.SecurityLevel == "" {
			t.Error("Report missing security level")
		}

		t.Logf("✓ Requirement 4.1: Compatibility report generated with all required fields")
	})

	// Requirement 4.2: Security level assessment
	t.Run("Requirement_4_2_SecurityAssessment", func(t *testing.T) {
		testCases := []struct {
			name     string
			params   map[string]interface{}
			expected SecurityLevel
		}{
			{
				name:     "Very High Security",
				params:   map[string]interface{}{"n": 262144, "r": 8, "p": 1},
				expected: SecurityLevelVeryHigh,
			},
			{
				name:     "Low Security",
				params:   map[string]interface{}{"n": 1024, "r": 4, "p": 1},
				expected: SecurityLevelLow,
			},
		}

		for _, tc := range testCases {
			crypto := &CryptoParams{
				KDF:       "scrypt",
				KDFParams: tc.params,
			}

			report, err := analyzer.AnalyzeKeystore(crypto)
			if err != nil {
				t.Fatalf("Failed to analyze %s: %v", tc.name, err)
			}

			if report.SecurityLevel != tc.expected {
				t.Errorf("%s: expected %s, got %s", tc.name, tc.expected, report.SecurityLevel)
			}
		}

		t.Logf("✓ Requirement 4.2: Security level assessment working correctly")
	})

	// Requirement 4.3: Detailed compatibility reporting
	t.Run("Requirement_4_3_DetailedReporting", func(t *testing.T) {
		// Test with invalid parameters to generate issues
		crypto := &CryptoParams{
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     0, // Invalid
				"r":     0, // Invalid
				"p":     0, // Invalid
				"dklen": 0, // Invalid
			},
		}

		report, err := analyzer.AnalyzeKeystore(crypto)
		if err != nil {
			t.Fatalf("Failed to analyze invalid keystore: %v", err)
		}

		if len(report.Issues) == 0 {
			t.Error("Expected issues for invalid parameters")
		}

		if len(report.Suggestions) == 0 {
			t.Error("Expected suggestions for invalid parameters")
		}

		t.Logf("✓ Requirement 4.3: Detailed reporting with %d issues and %d suggestions",
			len(report.Issues), len(report.Suggestions))
	})
}
