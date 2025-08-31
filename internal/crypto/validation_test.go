package crypto

import (
	"strings"
	"testing"
)

func TestSecurityLevel_String(t *testing.T) {
	tests := []struct {
		level    SecurityLevel
		expected string
	}{
		{SecurityLevelLow, "Low"},
		{SecurityLevelMedium, "Medium"},
		{SecurityLevelHigh, "High"},
		{SecurityLevelVeryHigh, "Very High"},
		{SecurityLevel(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("SecurityLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCryptographicValidator_ValidateScryptParams(t *testing.T) {
	validator := NewCryptographicValidator()

	tests := []struct {
		name           string
		params         map[string]interface{}
		expectValid    bool
		expectSecurity SecurityLevel
		expectIssues   []string
		expectWarnings []string
	}{
		{
			name: "valid standard parameters",
			params: map[string]interface{}{
				"n":     262144,
				"r":     8,
				"p":     1,
				"dklen": 32,
			},
			expectValid:    true,
			expectSecurity: SecurityLevelHigh,
		},
		{
			name: "missing n parameter",
			params: map[string]interface{}{
				"r":     8,
				"p":     1,
				"dklen": 32,
			},
			expectValid:  false,
			expectIssues: []string{"missing required parameter: n"},
		},
		{
			name: "n not power of 2",
			params: map[string]interface{}{
				"n":     1000,
				"r":     8,
				"p":     1,
				"dklen": 32,
			},
			expectValid:  false,
			expectIssues: []string{"N parameter must be a power of 2"},
		},
		{
			name: "n too low",
			params: map[string]interface{}{
				"n":     512,
				"r":     8,
				"p":     1,
				"dklen": 32,
			},
			expectValid:  false,
			expectIssues: []string{"n parameter too low"},
		},
		{
			name: "high memory usage warning",
			params: map[string]interface{}{
				"n":     65536, // High N
				"r":     16,    // High r
				"p":     4,     // High p
				"dklen": 32,
			},
			expectValid:    true,
			expectSecurity: SecurityLevelHigh,
			expectWarnings: []string{"high memory usage"},
		},
		{
			name: "excessive memory usage",
			params: map[string]interface{}{
				"n":     16777216, // Very high N
				"r":     32,       // Very high r
				"p":     8,        // Very high p
				"dklen": 32,
			},
			expectValid:  false,
			expectIssues: []string{"memory usage too high"},
		},
		{
			name: "low security parameters",
			params: map[string]interface{}{
				"n":     1024,
				"r":     1,
				"p":     1,
				"dklen": 32,
			},
			expectValid:    true,
			expectSecurity: SecurityLevelLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateScryptParams(tt.params)

			if result.Valid != tt.expectValid {
				t.Errorf("ValidateScryptParams() valid = %v, want %v", result.Valid, tt.expectValid)
			}

			if tt.expectValid && result.SecurityLevel != tt.expectSecurity {
				t.Errorf("ValidateScryptParams() security level = %v, want %v", result.SecurityLevel, tt.expectSecurity)
			}

			for _, expectedIssue := range tt.expectIssues {
				found := false
				for _, issue := range result.Issues {
					if strings.Contains(issue, expectedIssue) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateScryptParams() missing expected issue: %s", expectedIssue)
				}
			}

			for _, expectedWarning := range tt.expectWarnings {
				found := false
				for _, warning := range result.Warnings {
					if strings.Contains(warning, expectedWarning) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateScryptParams() missing expected warning: %s", expectedWarning)
				}
			}

			// Check that ranges are provided
			if len(result.Ranges) == 0 {
				t.Errorf("ValidateScryptParams() should provide parameter ranges")
			}

			// Check that analysis is provided for valid parameters
			if result.Valid && result.Analysis == nil {
				t.Errorf("ValidateScryptParams() should provide analysis for valid parameters")
			}
		})
	}
}

func TestCryptographicValidator_ValidatePBKDF2Params(t *testing.T) {
	validator := NewCryptographicValidator()

	tests := []struct {
		name           string
		params         map[string]interface{}
		expectValid    bool
		expectSecurity SecurityLevel
		expectIssues   []string
	}{
		{
			name: "valid standard parameters",
			params: map[string]interface{}{
				"c":     100000,
				"dklen": 32,
				"prf":   "hmac-sha256",
			},
			expectValid:    true,
			expectSecurity: SecurityLevelMedium,
		},
		{
			name: "missing c parameter",
			params: map[string]interface{}{
				"dklen": 32,
			},
			expectValid:  false,
			expectIssues: []string{"missing required parameter: c"},
		},
		{
			name: "c too low",
			params: map[string]interface{}{
				"c":     500,
				"dklen": 32,
			},
			expectValid:  false,
			expectIssues: []string{"c parameter too low"},
		},
		{
			name: "invalid PRF",
			params: map[string]interface{}{
				"c":     100000,
				"dklen": 32,
				"prf":   "md5",
			},
			expectValid:  false,
			expectIssues: []string{"invalid PRF"},
		},
		{
			name: "high security parameters",
			params: map[string]interface{}{
				"c":     500000,
				"dklen": 32,
				"prf":   "hmac-sha512",
			},
			expectValid:    true,
			expectSecurity: SecurityLevelVeryHigh,
		},
		{
			name: "low security parameters",
			params: map[string]interface{}{
				"c":     5000,
				"dklen": 32,
			},
			expectValid:    true,
			expectSecurity: SecurityLevelLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidatePBKDF2Params(tt.params)

			if result.Valid != tt.expectValid {
				t.Errorf("ValidatePBKDF2Params() valid = %v, want %v", result.Valid, tt.expectValid)
			}

			if tt.expectValid && result.SecurityLevel != tt.expectSecurity {
				t.Errorf("ValidatePBKDF2Params() security level = %v, want %v", result.SecurityLevel, tt.expectSecurity)
			}

			for _, expectedIssue := range tt.expectIssues {
				found := false
				for _, issue := range result.Issues {
					if strings.Contains(issue, expectedIssue) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidatePBKDF2Params() missing expected issue: %s", expectedIssue)
				}
			}

			// Check that analysis is provided for valid parameters
			if result.Valid && result.Analysis == nil {
				t.Errorf("ValidatePBKDF2Params() should provide analysis for valid parameters")
			}
		})
	}
}

func TestCryptographicValidator_GetParameterOptimizationSuggestions(t *testing.T) {
	validator := NewCryptographicValidator()

	tests := []struct {
		name              string
		kdfType           string
		currentParams     map[string]interface{}
		targetSecurity    SecurityLevel
		expectSuggestions bool
	}{
		{
			name:    "scrypt low to medium",
			kdfType: "scrypt",
			currentParams: map[string]interface{}{
				"n": 1024,
				"r": 1,
				"p": 1,
			},
			targetSecurity:    SecurityLevelMedium,
			expectSuggestions: true,
		},
		{
			name:    "pbkdf2 low to high",
			kdfType: "pbkdf2",
			currentParams: map[string]interface{}{
				"c": 5000,
			},
			targetSecurity:    SecurityLevelHigh,
			expectSuggestions: true,
		},
		{
			name:    "already high security",
			kdfType: "scrypt",
			currentParams: map[string]interface{}{
				"n": 262144,
				"r": 8,
				"p": 1,
			},
			targetSecurity:    SecurityLevelMedium,
			expectSuggestions: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := validator.GetParameterOptimizationSuggestions(tt.kdfType, tt.currentParams, tt.targetSecurity)

			if tt.expectSuggestions && len(suggestions) == 0 {
				t.Errorf("GetParameterOptimizationSuggestions() expected suggestions but got none")
			}

			if !tt.expectSuggestions && len(suggestions) > 0 {
				t.Errorf("GetParameterOptimizationSuggestions() expected no suggestions but got %d", len(suggestions))
			}
		})
	}
}

func TestAnalyzeScryptSecurity(t *testing.T) {
	validator := NewCryptographicValidator()

	analysis := validator.analyzeScryptSecurity(262144, 8, 1, 32)

	if analysis == nil {
		t.Fatal("analyzeScryptSecurity() returned nil")
	}

	if analysis.ComputationalCost <= 0 {
		t.Errorf("analyzeScryptSecurity() computational cost = %f, want > 0", analysis.ComputationalCost)
	}

	if analysis.MemoryUsage <= 0 {
		t.Errorf("analyzeScryptSecurity() memory usage = %d, want > 0", analysis.MemoryUsage)
	}

	if analysis.BitsOfSecurity <= 0 {
		t.Errorf("analyzeScryptSecurity() bits of security = %d, want > 0", analysis.BitsOfSecurity)
	}

	if analysis.ResistanceProfile == "" {
		t.Errorf("analyzeScryptSecurity() resistance profile should not be empty")
	}
}

func TestAnalyzePBKDF2Security(t *testing.T) {
	validator := NewCryptographicValidator()

	analysis := validator.analyzePBKDF2Security(100000, 32, "sha256")

	if analysis == nil {
		t.Fatal("analyzePBKDF2Security() returned nil")
	}

	if analysis.ComputationalCost <= 0 {
		t.Errorf("analyzePBKDF2Security() computational cost = %f, want > 0", analysis.ComputationalCost)
	}

	if analysis.MemoryUsage <= 0 {
		t.Errorf("analyzePBKDF2Security() memory usage = %d, want > 0", analysis.MemoryUsage)
	}

	if analysis.BitsOfSecurity <= 0 {
		t.Errorf("analyzePBKDF2Security() bits of security = %d, want > 0", analysis.BitsOfSecurity)
	}

	if !strings.Contains(analysis.ResistanceProfile, "sha256") {
		t.Errorf("analyzePBKDF2Security() resistance profile should mention hash function")
	}
}

func TestGlobalValidationFunctions(t *testing.T) {
	t.Run("ValidateKDFParameters scrypt", func(t *testing.T) {
		params := map[string]interface{}{
			"n":     262144,
			"r":     8,
			"p":     1,
			"dklen": 32,
		}
		result := ValidateKDFParameters("scrypt", params)
		if !result.Valid {
			t.Errorf("ValidateKDFParameters() valid = %v, want true", result.Valid)
		}
	})

	t.Run("ValidateKDFParameters pbkdf2", func(t *testing.T) {
		params := map[string]interface{}{
			"c":     100000,
			"dklen": 32,
		}
		result := ValidateKDFParameters("pbkdf2", params)
		if !result.Valid {
			t.Errorf("ValidateKDFParameters() valid = %v, want true", result.Valid)
		}
	})

	t.Run("ValidateKDFParameters unsupported", func(t *testing.T) {
		params := map[string]interface{}{}
		result := ValidateKDFParameters("unsupported", params)
		if result.Valid {
			t.Errorf("ValidateKDFParameters() valid = %v, want false", result.Valid)
		}
	})

	t.Run("GetParameterRanges scrypt", func(t *testing.T) {
		ranges := GetParameterRanges("scrypt")
		if ranges == nil {
			t.Errorf("GetParameterRanges() returned nil for scrypt")
		}
		if _, ok := ranges["n"]; !ok {
			t.Errorf("GetParameterRanges() missing 'n' parameter for scrypt")
		}
	})

	t.Run("GetParameterRanges unsupported", func(t *testing.T) {
		ranges := GetParameterRanges("unsupported")
		if ranges != nil {
			t.Errorf("GetParameterRanges() should return nil for unsupported KDF")
		}
	})

	t.Run("OptimizeParameters", func(t *testing.T) {
		params := map[string]interface{}{
			"n": 1024,
			"r": 1,
			"p": 1,
		}
		suggestions := OptimizeParameters("scrypt", params, SecurityLevelHigh)
		if len(suggestions) == 0 {
			t.Errorf("OptimizeParameters() expected suggestions for low security parameters")
		}
	})
}

func TestParameterHelpers(t *testing.T) {
	t.Run("getIntParam", func(t *testing.T) {
		params := map[string]interface{}{
			"int":     42,
			"int32":   int32(42),
			"int64":   int64(42),
			"float64": float64(42),
			"float32": float32(42),
			"string":  "not a number",
		}

		// Test valid conversions
		validKeys := []string{"int", "int32", "int64", "float64", "float32"}
		for _, key := range validKeys {
			if val, ok := getIntParam(params, key); !ok || val != 42 {
				t.Errorf("getIntParam(%s) = %d, %v, want 42, true", key, val, ok)
			}
		}

		// Test invalid conversion
		if val, ok := getIntParam(params, "string"); ok {
			t.Errorf("getIntParam(string) = %d, %v, want _, false", val, ok)
		}

		// Test missing key
		if val, ok := getIntParam(params, "missing"); ok {
			t.Errorf("getIntParam(missing) = %d, %v, want _, false", val, ok)
		}
	})

	t.Run("getStringParam", func(t *testing.T) {
		params := map[string]interface{}{
			"string": "test",
			"int":    42,
		}

		// Test valid conversion
		if val, ok := getStringParam(params, "string"); !ok || val != "test" {
			t.Errorf("getStringParam(string) = %s, %v, want test, true", val, ok)
		}

		// Test invalid conversion
		if val, ok := getStringParam(params, "int"); ok {
			t.Errorf("getStringParam(int) = %s, %v, want _, false", val, ok)
		}

		// Test missing key
		if val, ok := getStringParam(params, "missing"); ok {
			t.Errorf("getStringParam(missing) = %s, %v, want _, false", val, ok)
		}
	})
}

// Benchmark tests
func BenchmarkValidateScryptParams(b *testing.B) {
	validator := NewCryptographicValidator()
	params := map[string]interface{}{
		"n":     262144,
		"r":     8,
		"p":     1,
		"dklen": 32,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateScryptParams(params)
	}
}

func BenchmarkValidatePBKDF2Params(b *testing.B) {
	validator := NewCryptographicValidator()
	params := map[string]interface{}{
		"c":     100000,
		"dklen": 32,
		"prf":   "hmac-sha256",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidatePBKDF2Params(params)
	}
}
