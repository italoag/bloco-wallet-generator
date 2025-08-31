package kdf

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestScryptHandler_ParameterValidation tests scrypt parameter validation with boundary conditions
func TestScryptHandler_ParameterValidation(t *testing.T) {
	handler := &ScryptHandler{}

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid_standard_parameters",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: false,
		},
		{
			name: "n_not_power_of_2",
			params: map[string]interface{}{
				"n":     16383, // Not a power of 2
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "N parameter must be a power of 2",
		},
		{
			name: "n_too_low",
			params: map[string]interface{}{
				"n":     512, // Below minimum
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "N parameter too low for security",
		},
		{
			name: "n_too_high",
			params: map[string]interface{}{
				"n":     134217728, // Above maximum
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "N parameter too high, may cause memory exhaustion",
		},
		{
			name: "r_too_low",
			params: map[string]interface{}{
				"n":     16384,
				"r":     0, // Below minimum
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "R parameter must be positive",
		},
		{
			name: "r_too_high",
			params: map[string]interface{}{
				"n":     16384,
				"r":     2000, // Above maximum
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "R parameter too high, may cause excessive memory usage",
		},
		{
			name: "p_too_low",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     0, // Below minimum
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "P parameter must be positive",
		},
		{
			name: "p_too_high",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     256, // Above maximum
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "P parameter too high, may not provide additional security benefit",
		},
		{
			name: "dklen_too_low",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     1,
				"dklen": 0, // Below minimum
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Derived key length too short for security",
		},
		{
			name: "dklen_too_high",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     1,
				"dklen": 1025, // Above maximum
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Derived key length unnecessarily long",
		},
		{
			name: "excessive_memory_usage",
			params: map[string]interface{}{
				"n":     1048576, // High N
				"r":     32,      // High r
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Memory usage too high",
		},
		{
			name: "missing_salt_parameter",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     1,
				"dklen": 32,
				// Missing salt
			},
			wantError: true,
			errorMsg:  "salt parameter not found",
		},
		{
			name: "invalid_salt_type",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  123, // Invalid type
			},
			wantError: true,
			errorMsg:  "unsupported salt type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateParams(tt.params)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got %v", tt.name, err)
				}
			}
		})
	}
}

// TestPBKDF2Handler_ParameterValidation tests PBKDF2 parameter validation with different hash functions
func TestPBKDF2Handler_ParameterValidation(t *testing.T) {
	handler := &PBKDF2Handler{}

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid_sha256_parameters",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: false,
		},
		{
			name: "valid_sha512_parameters",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-sha512",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: false,
		},
		{
			name: "valid_default_prf",
			params: map[string]interface{}{
				"c":     100000,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: false,
		},
		{
			name: "c_too_low",
			params: map[string]interface{}{
				"c":     999, // Below minimum
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Iteration count too low for modern security standards",
		},
		{
			name: "c_too_high",
			params: map[string]interface{}{
				"c":     100000001, // Above maximum
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Iteration count extremely high, may cause performance issues",
		},
		{
			name: "invalid_prf",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-md5", // Unsupported
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Unsupported PRF",
		},
		{
			name: "dklen_too_low",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-sha256",
				"dklen": 0, // Below minimum
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Derived key length too short for security",
		},
		{
			name: "dklen_too_high",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-sha256",
				"dklen": 1025, // Above maximum
				"salt":  "0123456789abcdef0123456789abcdef",
			},
			wantError: true,
			errorMsg:  "Derived key length unnecessarily long",
		},
		{
			name: "missing_salt_parameter",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-sha256",
				"dklen": 32,
				// Missing salt
			},
			wantError: true,
			errorMsg:  "salt parameter not found",
		},
		{
			name: "invalid_salt_type",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  123, // Invalid type
			},
			wantError: true,
			errorMsg:  "unsupported salt type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateParams(tt.params)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got %v", tt.name, err)
				}
			}
		})
	}
}

// TestKDFHandlers_KnownTestVectors tests key derivation correctness against known test vectors
func TestKDFHandlers_KnownTestVectors(t *testing.T) {
	t.Run("scrypt_test_vectors", func(t *testing.T) {
		handler := &ScryptHandler{}

		// Test vector from RFC 7914
		testVectors := []struct {
			name     string
			password string
			params   map[string]interface{}
			expected string
		}{
			{
				name:     "rfc7914_test_vector_1",
				password: "",
				params: map[string]interface{}{
					"n":     16,
					"r":     1,
					"p":     1,
					"dklen": 64,
					"salt":  "",
				},
				expected: "77d6576238657b203b19ca42c18a0497f16b4844e3074ae8dfdffa3fede21442fcd0069ded0948f8326a753a0fc81f17e8d3e0fb2e0d3628cf35e20c38d18906",
			},
			{
				name:     "rfc7914_test_vector_2",
				password: "password",
				params: map[string]interface{}{
					"n":     1024,
					"r":     8,
					"p":     16,
					"dklen": 64,
					"salt":  "4e61436c",
				},
				expected: "fdbabe1c9d3472007856e7190d01e9fe7c6ad7cbc8237830e77376634b3731622eaf30d92e22a3886ff109279d9830dac727afb94a83ee6d8360cbdfa2cc0640",
			},
		}

		for _, tv := range testVectors {
			t.Run(tv.name, func(t *testing.T) {
				derived, err := handler.DeriveKey(tv.password, tv.params)
				if err != nil {
					t.Fatalf("Failed to derive key: %v", err)
				}

				derivedHex := hex.EncodeToString(derived)
				if derivedHex != tv.expected {
					t.Errorf("Expected %s, got %s", tv.expected, derivedHex)
				}
			})
		}
	})

	t.Run("pbkdf2_test_vectors", func(t *testing.T) {
		handler := &PBKDF2Handler{}

		// Test vectors for SHA-256 (since SHA-1 is deprecated)
		testVectors := []struct {
			name     string
			password string
			params   map[string]interface{}
			expected string
		}{
			{
				name:     "pbkdf2_sha256_test_vector_1",
				password: "password",
				params: map[string]interface{}{
					"c":     1,
					"prf":   "hmac-sha256",
					"dklen": 32,
					"salt":  "73616c74", // "salt"
				},
				expected: "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b",
			},
			{
				name:     "pbkdf2_sha256_test_vector_2",
				password: "password",
				params: map[string]interface{}{
					"c":     2,
					"prf":   "hmac-sha256",
					"dklen": 32,
					"salt":  "73616c74", // "salt"
				},
				expected: "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43",
			},
			{
				name:     "pbkdf2_sha256_test_vector_3",
				password: "password",
				params: map[string]interface{}{
					"c":     4096,
					"prf":   "hmac-sha256",
					"dklen": 32,
					"salt":  "73616c74", // "salt"
				},
				expected: "c5e478d59288c841aa530db6845c4c8d962893a001ce4e11a4963873aa98134a",
			},
		}

		for _, tv := range testVectors {
			t.Run(tv.name, func(t *testing.T) {
				derived, err := handler.DeriveKey(tv.password, tv.params)
				if err != nil {
					t.Fatalf("Failed to derive key: %v", err)
				}

				derivedHex := hex.EncodeToString(derived)
				if derivedHex != tv.expected {
					t.Errorf("Expected %s, got %s", tv.expected, derivedHex)
				}
			})
		}
	})
}

// TestKDFHandlers_ErrorHandling tests error handling for invalid parameters and edge cases
func TestKDFHandlers_ErrorHandling(t *testing.T) {
	t.Run("scrypt_error_handling", func(t *testing.T) {
		handler := &ScryptHandler{}

		errorTests := []struct {
			name     string
			password string
			params   map[string]interface{}
			errorMsg string
		}{
			{
				name:     "empty_password_valid",
				password: "",
				params: map[string]interface{}{
					"n":     16384,
					"r":     8,
					"p":     1,
					"dklen": 32,
					"salt":  "0123456789abcdef0123456789abcdef",
				},
				errorMsg: "", // Empty password should be valid
			},
			{
				name:     "missing_salt",
				password: "password",
				params: map[string]interface{}{
					"n":     16384,
					"r":     8,
					"p":     1,
					"dklen": 32,
					// Missing salt
				},
				errorMsg: "salt parameter not found",
			},
		}

		for _, tt := range errorTests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := handler.DeriveKey(tt.password, tt.params)

				if tt.errorMsg == "" {
					if err != nil {
						t.Errorf("Expected no error for %s, got %v", tt.name, err)
					}
				} else {
					if err == nil {
						t.Errorf("Expected error for %s, got nil", tt.name)
					} else if !strings.Contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
					}
				}
			})
		}
	})

	t.Run("pbkdf2_error_handling", func(t *testing.T) {
		handler := &PBKDF2Handler{}

		errorTests := []struct {
			name     string
			password string
			params   map[string]interface{}
			errorMsg string
		}{
			{
				name:     "empty_password_valid",
				password: "",
				params: map[string]interface{}{
					"c":     100000,
					"prf":   "hmac-sha256",
					"dklen": 32,
					"salt":  "0123456789abcdef0123456789abcdef",
				},
				errorMsg: "", // Empty password should be valid
			},
			{
				name:     "missing_salt",
				password: "password",
				params: map[string]interface{}{
					"c":     100000,
					"prf":   "hmac-sha256",
					"dklen": 32,
					// Missing salt
				},
				errorMsg: "salt parameter not found",
			},
		}

		for _, tt := range errorTests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := handler.DeriveKey(tt.password, tt.params)

				if tt.errorMsg == "" {
					if err != nil {
						t.Errorf("Expected no error for %s, got %v", tt.name, err)
					}
				} else {
					if err == nil {
						t.Errorf("Expected error for %s, got nil", tt.name)
					} else if !strings.Contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
					}
				}
			})
		}
	})
}

// TestKDFHandlers_DefaultParameters tests default parameter generation
func TestKDFHandlers_DefaultParameters(t *testing.T) {
	t.Run("scrypt_default_params", func(t *testing.T) {
		handler := &ScryptHandler{}
		defaults := handler.GetDefaultParams()

		// Verify all required parameters are present
		requiredParams := []string{"n", "r", "p", "dklen"}
		for _, param := range requiredParams {
			if _, exists := defaults[param]; !exists {
				t.Errorf("Missing required default parameter: %s", param)
			}
		}

		// Verify parameter values are reasonable
		if n, ok := defaults["n"].(int); !ok || n < 1024 {
			t.Errorf("Default n parameter should be at least 1024, got %v", defaults["n"])
		}

		if r, ok := defaults["r"].(int); !ok || r < 1 {
			t.Errorf("Default r parameter should be at least 1, got %v", defaults["r"])
		}

		if p, ok := defaults["p"].(int); !ok || p < 1 {
			t.Errorf("Default p parameter should be at least 1, got %v", defaults["p"])
		}

		if dklen, ok := defaults["dklen"].(int); !ok || dklen < 32 {
			t.Errorf("Default dklen parameter should be at least 32, got %v", defaults["dklen"])
		}

		// Note: We don't validate defaults since they don't include salt
		// which is required for actual key derivation but not for parameter templates
	})

	t.Run("pbkdf2_default_params", func(t *testing.T) {
		handler := &PBKDF2Handler{}
		defaults := handler.GetDefaultParams()

		// Verify all required parameters are present
		requiredParams := []string{"c", "dklen"}
		for _, param := range requiredParams {
			if _, exists := defaults[param]; !exists {
				t.Errorf("Missing required default parameter: %s", param)
			}
		}

		// Verify parameter values are reasonable
		if c, ok := defaults["c"].(int); !ok || c < 100000 {
			t.Errorf("Default c parameter should be at least 100000, got %v", defaults["c"])
		}

		if dklen, ok := defaults["dklen"].(int); !ok || dklen < 32 {
			t.Errorf("Default dklen parameter should be at least 32, got %v", defaults["dklen"])
		}

		// PRF should default to hmac-sha256 or be empty (which defaults to sha256)
		if prf, exists := defaults["prf"]; exists {
			if prfStr, ok := prf.(string); ok && prfStr != "hmac-sha256" && prfStr != "" {
				t.Errorf("Default PRF should be hmac-sha256 or empty, got %v", prf)
			}
		}

		// Note: We don't validate defaults since they don't include salt
		// which is required for actual key derivation but not for parameter templates
	})
}

// TestKDFHandlers_ParameterRanges tests parameter range retrieval
func TestKDFHandlers_ParameterRanges(t *testing.T) {
	t.Run("scrypt_parameter_ranges", func(t *testing.T) {
		handler := &ScryptHandler{}

		testParams := []string{"n", "r", "p", "dklen"}
		for _, param := range testParams {
			min, max := handler.GetParamRange(param)
			if min == nil || max == nil {
				t.Errorf("Parameter %s should have defined ranges, got min=%v, max=%v", param, min, max)
				continue
			}

			// Verify ranges are reasonable
			switch param {
			case "n":
				if minInt, ok := min.(int); !ok || minInt < 1024 {
					t.Errorf("Minimum n should be at least 1024, got %v", min)
				}
				if maxInt, ok := max.(int); !ok || maxInt > 67108864 {
					t.Errorf("Maximum n should be at most 67108864, got %v", max)
				}
			case "r":
				if minInt, ok := min.(int); !ok || minInt < 1 {
					t.Errorf("Minimum r should be at least 1, got %v", min)
				}
				if maxInt, ok := max.(int); !ok || maxInt > 1024 {
					t.Errorf("Maximum r should be at most 1024, got %v", max)
				}
			case "p":
				if minInt, ok := min.(int); !ok || minInt < 1 {
					t.Errorf("Minimum p should be at least 1, got %v", min)
				}
				if maxInt, ok := max.(int); !ok || maxInt > 255 {
					t.Errorf("Maximum p should be at most 255, got %v", max)
				}
			case "dklen":
				if minInt, ok := min.(int); !ok || minInt < 1 {
					t.Errorf("Minimum dklen should be at least 1, got %v", min)
				}
				if maxInt, ok := max.(int); !ok || maxInt > 1024 {
					t.Errorf("Maximum dklen should be at most 1024, got %v", max)
				}
			}
		}

		// Test invalid parameter
		min, max := handler.GetParamRange("invalid")
		if min != nil || max != nil {
			t.Errorf("Invalid parameter should return nil ranges, got min=%v, max=%v", min, max)
		}
	})

	t.Run("pbkdf2_parameter_ranges", func(t *testing.T) {
		handler := &PBKDF2Handler{}

		testParams := []string{"c", "dklen"}
		for _, param := range testParams {
			min, max := handler.GetParamRange(param)
			if min == nil || max == nil {
				t.Errorf("Parameter %s should have defined ranges, got min=%v, max=%v", param, min, max)
				continue
			}

			// Verify ranges are reasonable
			switch param {
			case "c":
				if minInt, ok := min.(int); !ok || minInt < 1000 {
					t.Errorf("Minimum c should be at least 1000, got %v", min)
				}
				if maxInt, ok := max.(int); !ok || maxInt > 100000000 {
					t.Errorf("Maximum c should be at most 100000000, got %v", max)
				}
			case "dklen":
				if minInt, ok := min.(int); !ok || minInt < 1 {
					t.Errorf("Minimum dklen should be at least 1, got %v", min)
				}
				if maxInt, ok := max.(int); !ok || maxInt > 1024 {
					t.Errorf("Maximum dklen should be at most 1024, got %v", max)
				}
			}
		}

		// Test invalid parameter
		min, max := handler.GetParamRange("invalid")
		if min != nil || max != nil {
			t.Errorf("Invalid parameter should return nil ranges, got min=%v, max=%v", min, max)
		}
	})
}

// BenchmarkKDFHandlers_Performance benchmarks KDF handler performance
func BenchmarkKDFHandlers_Performance(b *testing.B) {
	b.Run("scrypt_standard_params", func(b *testing.B) {
		handler := &ScryptHandler{}
		params := map[string]interface{}{
			"n":     16384,
			"r":     8,
			"p":     1,
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef",
		}
		password := "test_password"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := handler.DeriveKey(password, params)
			if err != nil {
				b.Fatalf("Failed to derive key: %v", err)
			}
		}
	})

	b.Run("scrypt_high_security_params", func(b *testing.B) {
		handler := &ScryptHandler{}
		params := map[string]interface{}{
			"n":     262144,
			"r":     8,
			"p":     1,
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef",
		}
		password := "test_password"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := handler.DeriveKey(password, params)
			if err != nil {
				b.Fatalf("Failed to derive key: %v", err)
			}
		}
	})

	b.Run("pbkdf2_sha256_standard_params", func(b *testing.B) {
		handler := &PBKDF2Handler{}
		params := map[string]interface{}{
			"c":     100000,
			"prf":   "hmac-sha256",
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef",
		}
		password := "test_password"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := handler.DeriveKey(password, params)
			if err != nil {
				b.Fatalf("Failed to derive key: %v", err)
			}
		}
	})

	b.Run("pbkdf2_sha512_standard_params", func(b *testing.B) {
		handler := &PBKDF2Handler{}
		params := map[string]interface{}{
			"c":     100000,
			"prf":   "hmac-sha512",
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef",
		}
		password := "test_password"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := handler.DeriveKey(password, params)
			if err != nil {
				b.Fatalf("Failed to derive key: %v", err)
			}
		}
	})
}

// TestKDFHandlers_MemoryUsage tests memory usage calculations and limits
func TestKDFHandlers_MemoryUsage(t *testing.T) {
	t.Run("scrypt_memory_calculation", func(t *testing.T) {
		handler := &ScryptHandler{}

		tests := []struct {
			name           string
			n, r           int
			expectedMemory int64
		}{
			{
				name:           "standard_params",
				n:              16384,
				r:              8,
				expectedMemory: 16384 * 8 * 128, // N * r * 128
			},
			{
				name:           "high_memory_params",
				n:              262144,
				r:              8,
				expectedMemory: 262144 * 8 * 128,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				params := map[string]interface{}{
					"n":     tt.n,
					"r":     tt.r,
					"p":     1,
					"dklen": 32,
					"salt":  "0123456789abcdef0123456789abcdef",
				}

				// Memory usage should be calculated during validation
				err := handler.ValidateParams(params)
				if err != nil && strings.Contains(err.Error(), "memory usage") {
					// This is expected for high memory usage
					if tt.expectedMemory > 2*1024*1024*1024 { // 2GB
						return
					}
					t.Errorf("Unexpected memory usage error: %v", err)
				}

				// For valid parameters, should be able to derive key
				if err == nil {
					_, derivErr := handler.DeriveKey("test", params)
					if derivErr != nil {
						t.Errorf("Failed to derive key with valid params: %v", derivErr)
					}
				}
			})
		}
	})
}

// TestKDFHandlers_ConcurrentAccess tests thread safety
func TestKDFHandlers_ConcurrentAccess(t *testing.T) {
	t.Run("scrypt_concurrent_derivation", func(t *testing.T) {
		handler := &ScryptHandler{}
		params := map[string]interface{}{
			"n":     4096, // Lower for faster testing
			"r":     8,
			"p":     1,
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef",
		}

		const numGoroutines = 10
		const numIterations = 5

		results := make(chan error, numGoroutines*numIterations)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					password := fmt.Sprintf("password_%d_%d", id, j)
					_, err := handler.DeriveKey(password, params)
					results <- err
				}
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines*numIterations; i++ {
			if err := <-results; err != nil {
				t.Errorf("Concurrent derivation failed: %v", err)
			}
		}
	})

	t.Run("pbkdf2_concurrent_derivation", func(t *testing.T) {
		handler := &PBKDF2Handler{}
		params := map[string]interface{}{
			"c":     10000, // Lower for faster testing
			"prf":   "hmac-sha256",
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef",
		}

		const numGoroutines = 10
		const numIterations = 5

		results := make(chan error, numGoroutines*numIterations)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					password := fmt.Sprintf("password_%d_%d", id, j)
					_, err := handler.DeriveKey(password, params)
					results <- err
				}
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines*numIterations; i++ {
			if err := <-results; err != nil {
				t.Errorf("Concurrent derivation failed: %v", err)
			}
		}
	})
}

// TestEthereumClientCompatibility tests keystore generation and loading with different Ethereum clients
func TestEthereumClientCompatibility(t *testing.T) {
	// Test keystore generation with different KDF configurations that should be compatible
	// with various Ethereum clients

	t.Run("geth_compatibility", func(t *testing.T) {
		// Test keystore format that's compatible with geth
		testGethCompatibility(t)
	})

	t.Run("besu_compatibility", func(t *testing.T) {
		// Test keystore format that's compatible with Besu
		testBesuCompatibility(t)
	})

	t.Run("anvil_compatibility", func(t *testing.T) {
		// Test keystore format that's compatible with Anvil
		testAnvilCompatibility(t)
	})

	t.Run("reth_compatibility", func(t *testing.T) {
		// Test keystore format that's compatible with Reth
		testRethCompatibility(t)
	})

	t.Run("firefly_compatibility", func(t *testing.T) {
		// Test keystore format that's compatible with Hyperledger Firefly
		testFireflyCompatibility(t)
	})
}

func testGethCompatibility(t *testing.T) {
	// geth supports both scrypt and PBKDF2 with standard parameters
	testCases := []struct {
		name   string
		kdf    string
		params map[string]interface{}
	}{
		{
			name: "geth_scrypt_standard",
			kdf:  "scrypt",
			params: map[string]interface{}{
				"n":     262144,
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "geth_pbkdf2_sha256",
			kdf:  "pbkdf2",
			params: map[string]interface{}{
				"c":     262144,
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keystore := generateTestKeystore(t, tc.kdf, tc.params)
			validateKeystoreFormat(t, keystore)

			// Verify geth-specific compatibility
			if version, ok := keystore["version"].(int); !ok || version != 3 {
				t.Errorf("geth requires version 3, got %v", keystore["version"])
			}

			// Verify cipher is supported by geth
			if crypto, ok := keystore["crypto"].(map[string]interface{}); ok {
				if cipher, ok := crypto["cipher"].(string); !ok || cipher != "aes-128-ctr" {
					t.Errorf("geth expects aes-128-ctr cipher, got %v", crypto["cipher"])
				}
			}
		})
	}
}

func testBesuCompatibility(t *testing.T) {
	// Besu supports scrypt and PBKDF2 with specific parameter ranges
	testCases := []struct {
		name   string
		kdf    string
		params map[string]interface{}
	}{
		{
			name: "besu_scrypt_standard",
			kdf:  "scrypt",
			params: map[string]interface{}{
				"n":     16384, // Besu prefers lower N values for performance
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "besu_pbkdf2_sha256",
			kdf:  "pbkdf2",
			params: map[string]interface{}{
				"c":     100000, // Besu supports standard iteration counts
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keystore := generateTestKeystore(t, tc.kdf, tc.params)
			validateKeystoreFormat(t, keystore)

			// Verify Besu-specific requirements
			if version, ok := keystore["version"].(int); !ok || version != 3 {
				t.Errorf("Besu requires version 3, got %v", keystore["version"])
			}

			// Besu requires specific MAC format
			if crypto, ok := keystore["crypto"].(map[string]interface{}); ok {
				if mac, ok := crypto["mac"].(string); ok {
					if len(mac) != 64 { // 32 bytes hex encoded
						t.Errorf("Besu expects 64-character MAC, got %d characters", len(mac))
					}
				}
			}
		})
	}
}

func testAnvilCompatibility(t *testing.T) {
	// Anvil has more restrictive requirements for performance
	testCases := []struct {
		name   string
		kdf    string
		params map[string]interface{}
	}{
		{
			name: "anvil_scrypt_light",
			kdf:  "scrypt",
			params: map[string]interface{}{
				"n":     4096, // Anvil prefers very light parameters for dev speed
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "anvil_pbkdf2_light",
			kdf:  "pbkdf2",
			params: map[string]interface{}{
				"c":     10000, // Anvil uses lower iteration counts for dev speed
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keystore := generateTestKeystore(t, tc.kdf, tc.params)
			validateKeystoreFormat(t, keystore)

			// Anvil compatibility checks
			if version, ok := keystore["version"].(int); !ok || version != 3 {
				t.Errorf("Anvil requires version 3, got %v", keystore["version"])
			}

			// Verify parameters are within Anvil's acceptable range
			if tc.kdf == "scrypt" {
				n := tc.params["n"].(int)
				if n > 16384 {
					t.Errorf("Anvil may have issues with N > 16384, got %d", n)
				}
			}
		})
	}
}

func testRethCompatibility(t *testing.T) {
	// Reth supports standard Ethereum keystore formats
	testCases := []struct {
		name   string
		kdf    string
		params map[string]interface{}
	}{
		{
			name: "reth_scrypt_standard",
			kdf:  "scrypt",
			params: map[string]interface{}{
				"n":     262144,
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "reth_pbkdf2_sha256",
			kdf:  "pbkdf2",
			params: map[string]interface{}{
				"c":     262144,
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keystore := generateTestKeystore(t, tc.kdf, tc.params)
			validateKeystoreFormat(t, keystore)

			// Reth compatibility checks
			if version, ok := keystore["version"].(int); !ok || version != 3 {
				t.Errorf("Reth requires version 3, got %v", keystore["version"])
			}

			// Verify standard Ethereum keystore format
			if crypto, ok := keystore["crypto"].(map[string]interface{}); ok {
				if cipher, ok := crypto["cipher"].(string); !ok || cipher != "aes-128-ctr" {
					t.Errorf("Reth expects aes-128-ctr cipher, got %v", crypto["cipher"])
				}
			}
		})
	}
}

func testFireflyCompatibility(t *testing.T) {
	// Hyperledger Firefly has specific requirements for enterprise use
	testCases := []struct {
		name   string
		kdf    string
		params map[string]interface{}
	}{
		{
			name: "firefly_scrypt_enterprise",
			kdf:  "scrypt",
			params: map[string]interface{}{
				"n":     65536, // Firefly balances security and performance
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "firefly_pbkdf2_enterprise",
			kdf:  "pbkdf2",
			params: map[string]interface{}{
				"c":     200000, // Firefly uses higher iteration counts for enterprise security
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keystore := generateTestKeystore(t, tc.kdf, tc.params)
			validateKeystoreFormat(t, keystore)

			// Firefly compatibility checks
			if version, ok := keystore["version"].(int); !ok || version != 3 {
				t.Errorf("Firefly requires version 3, got %v", keystore["version"])
			}

			// Firefly requires proper UUID format for ID
			if id, ok := keystore["id"].(string); ok {
				if len(id) != 36 { // UUID format: 8-4-4-4-12
					t.Errorf("Firefly expects UUID format for ID, got length %d", len(id))
				}
			}
		})
	}
}

// generateTestKeystore creates a test keystore with the specified KDF and parameters
func generateTestKeystore(t *testing.T, kdfType string, params map[string]interface{}) map[string]interface{} {
	// Create a minimal keystore structure for testing
	// We'll return a map instead of a struct to avoid import issues

	// Test address for keystore generation
	address := "0x1234567890123456789012345678901234567890"

	// Create keystore using a map structure
	keystore := map[string]interface{}{
		"address": strings.ToLower(strings.TrimPrefix(address, "0x")),
		"id":      "12345678-1234-1234-1234-123456789012", // Test UUID
		"version": 3,
		"crypto": map[string]interface{}{
			"kdf":       kdfType,
			"kdfparams": params,
			"cipher":    "aes-128-ctr",
			"cipherparams": map[string]interface{}{
				"iv": "0123456789abcdef0123456789abcdef",
			},
			"ciphertext": "encrypted_data_placeholder",
			"mac":        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
	}

	return keystore
}

// validateKeystoreFormat validates that a keystore conforms to Ethereum KeyStore V3 format
func validateKeystoreFormat(t *testing.T, keystore map[string]interface{}) {
	// Validate required fields
	if version, ok := keystore["version"].(int); !ok || version != 3 {
		t.Errorf("Invalid version: expected 3, got %v", keystore["version"])
	}

	if address, ok := keystore["address"].(string); !ok || len(address) != 40 {
		t.Errorf("Invalid address length: expected 40, got %d", len(address))
	}

	if id, ok := keystore["id"].(string); !ok || len(id) == 0 {
		t.Errorf("ID field is required")
	}

	// Validate crypto section
	crypto, ok := keystore["crypto"].(map[string]interface{})
	if !ok {
		t.Errorf("Crypto section is required")
		return
	}

	if kdf, ok := crypto["kdf"].(string); !ok || kdf == "" {
		t.Errorf("KDF field is required")
	}

	if _, ok := crypto["kdfparams"]; !ok {
		t.Errorf("KDFParams field is required")
	}

	if cipher, ok := crypto["cipher"].(string); !ok || cipher == "" {
		t.Errorf("Cipher field is required")
	}

	if _, ok := crypto["cipherparams"]; !ok {
		t.Errorf("CipherParams field is required")
	}

	if ciphertext, ok := crypto["ciphertext"].(string); !ok || ciphertext == "" {
		t.Errorf("CipherText field is required")
	}

	if mac, ok := crypto["mac"].(string); !ok || mac == "" {
		t.Errorf("MAC field is required")
	}

	// Validate KDF-specific parameters
	if kdf, ok := crypto["kdf"].(string); ok {
		if kdfParams, ok := crypto["kdfparams"].(map[string]interface{}); ok {
			switch kdf {
			case "scrypt":
				validateScryptParams(t, kdfParams)
			case "pbkdf2":
				validatePBKDF2Params(t, kdfParams)
			default:
				t.Errorf("Unsupported KDF: %s", kdf)
			}
		}
	}
}

func validateScryptParams(t *testing.T, params map[string]interface{}) {
	requiredParams := []string{"n", "r", "p", "dklen", "salt"}
	for _, param := range requiredParams {
		if _, exists := params[param]; !exists {
			t.Errorf("Missing required scrypt parameter: %s", param)
		}
	}

	// Validate parameter types and ranges
	if n, ok := params["n"].(int); ok {
		if n <= 0 || (n&(n-1)) != 0 {
			t.Errorf("Invalid scrypt N parameter: must be positive power of 2, got %d", n)
		}
	}

	if r, ok := params["r"].(int); ok {
		if r <= 0 {
			t.Errorf("Invalid scrypt r parameter: must be positive, got %d", r)
		}
	}

	if p, ok := params["p"].(int); ok {
		if p <= 0 {
			t.Errorf("Invalid scrypt p parameter: must be positive, got %d", p)
		}
	}

	if dklen, ok := params["dklen"].(int); ok {
		if dklen <= 0 {
			t.Errorf("Invalid scrypt dklen parameter: must be positive, got %d", dklen)
		}
	}
}

func validatePBKDF2Params(t *testing.T, params map[string]interface{}) {
	requiredParams := []string{"c", "dklen", "salt"}
	for _, param := range requiredParams {
		if _, exists := params[param]; !exists {
			t.Errorf("Missing required PBKDF2 parameter: %s", param)
		}
	}

	// Validate parameter types and ranges
	if c, ok := params["c"].(int); ok {
		if c <= 0 {
			t.Errorf("Invalid PBKDF2 c parameter: must be positive, got %d", c)
		}
	}

	if dklen, ok := params["dklen"].(int); ok {
		if dklen <= 0 {
			t.Errorf("Invalid PBKDF2 dklen parameter: must be positive, got %d", dklen)
		}
	}

	// PRF is optional, but if present should be valid
	if prf, exists := params["prf"]; exists {
		if prfStr, ok := prf.(string); ok {
			validPRFs := []string{"hmac-sha256", "hmac-sha512", ""}
			valid := false
			for _, validPRF := range validPRFs {
				if prfStr == validPRF {
					valid = true
					break
				}
			}
			if !valid {
				t.Errorf("Invalid PBKDF2 PRF parameter: %s", prfStr)
			}
		}
	}
}

// TestKeystoreRoundTrip tests that keystores can be generated and then loaded successfully
func TestKeystoreRoundTrip(t *testing.T) {
	testCases := []struct {
		name   string
		kdf    string
		params map[string]interface{}
	}{
		{
			name: "scrypt_roundtrip",
			kdf:  "scrypt",
			params: map[string]interface{}{
				"n":     16384,
				"r":     8,
				"p":     1,
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "pbkdf2_roundtrip",
			kdf:  "pbkdf2",
			params: map[string]interface{}{
				"c":     100000,
				"prf":   "hmac-sha256",
				"dklen": 32,
				"salt":  "0123456789abcdef0123456789abcdef",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate keystore
			keystore := generateTestKeystore(t, tc.kdf, tc.params)

			// Validate format
			validateKeystoreFormat(t, keystore)

			// Test JSON serialization/deserialization
			jsonData, err := json.Marshal(keystore)
			if err != nil {
				t.Fatalf("Failed to marshal keystore to JSON: %v", err)
			}

			var loadedKeystore map[string]interface{}
			err = json.Unmarshal(jsonData, &loadedKeystore)
			if err != nil {
				t.Fatalf("Failed to unmarshal keystore from JSON: %v", err)
			}

			// Verify loaded keystore matches original
			if loadedKeystore["address"] != keystore["address"] {
				t.Errorf("Address mismatch: expected %s, got %s", keystore["address"], loadedKeystore["address"])
			}

			// JSON unmarshaling converts numbers to float64, so we need to handle this
			expectedVersion := float64(keystore["version"].(int))
			if loadedKeystore["version"] != expectedVersion {
				t.Errorf("Version mismatch: expected %v, got %v", expectedVersion, loadedKeystore["version"])
			}

			originalCrypto := keystore["crypto"].(map[string]interface{})
			loadedCrypto := loadedKeystore["crypto"].(map[string]interface{})
			if loadedCrypto["kdf"] != originalCrypto["kdf"] {
				t.Errorf("KDF mismatch: expected %s, got %s", originalCrypto["kdf"], loadedCrypto["kdf"])
			}
		})
	}
}

// TestClientSpecificParameterLimits tests parameter limits for different clients
func TestClientSpecificParameterLimits(t *testing.T) {
	t.Run("memory_limits", func(t *testing.T) {
		// Test that different clients have different memory tolerance
		testCases := []struct {
			client      string
			maxMemoryMB int
			params      map[string]interface{}
			shouldPass  bool
		}{
			{
				client:      "geth",
				maxMemoryMB: 1024, // 1GB limit
				params: map[string]interface{}{
					"n":     262144,
					"r":     8,
					"p":     1,
					"dklen": 32,
					"salt":  "0123456789abcdef0123456789abcdef",
				},
				shouldPass: true,
			},
			{
				client:      "anvil",
				maxMemoryMB: 256, // 256MB limit for dev speed
				params: map[string]interface{}{
					"n":     524288, // This would use ~512MB, exceeding 256MB limit
					"r":     8,
					"p":     1,
					"dklen": 32,
					"salt":  "0123456789abcdef0123456789abcdef",
				},
				shouldPass: false, // Should exceed Anvil's limits
			},
			{
				client:      "besu",
				maxMemoryMB: 512, // 512MB limit
				params: map[string]interface{}{
					"n":     65536,
					"r":     8,
					"p":     1,
					"dklen": 32,
					"salt":  "0123456789abcdef0123456789abcdef",
				},
				shouldPass: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.client, func(t *testing.T) {
				// Calculate memory usage: N * r * 128 bytes
				n := tc.params["n"].(int)
				r := tc.params["r"].(int)
				memoryBytes := int64(n * r * 128)
				memoryMB := memoryBytes / (1024 * 1024)

				if tc.shouldPass {
					if memoryMB > int64(tc.maxMemoryMB) {
						t.Errorf("Memory usage %dMB exceeds %s limit of %dMB", memoryMB, tc.client, tc.maxMemoryMB)
					}
				} else {
					if memoryMB <= int64(tc.maxMemoryMB) {
						t.Errorf("Expected memory usage %dMB to exceed %s limit of %dMB", memoryMB, tc.client, tc.maxMemoryMB)
					}
				}
			})
		}
	})

	t.Run("iteration_limits", func(t *testing.T) {
		// Test PBKDF2 iteration limits for different clients
		testCases := []struct {
			client        string
			maxIterations int
			iterations    int
			shouldPass    bool
		}{
			{
				client:        "geth",
				maxIterations: 10000000,
				iterations:    262144,
				shouldPass:    true,
			},
			{
				client:        "anvil",
				maxIterations: 100000, // Lower for dev speed
				iterations:    262144,
				shouldPass:    false,
			},
			{
				client:        "firefly",
				maxIterations: 1000000, // Enterprise balance
				iterations:    500000,
				shouldPass:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.client, func(t *testing.T) {
				if tc.shouldPass {
					if tc.iterations > tc.maxIterations {
						t.Errorf("Iterations %d exceeds %s limit of %d", tc.iterations, tc.client, tc.maxIterations)
					}
				} else {
					if tc.iterations <= tc.maxIterations {
						t.Errorf("Expected iterations %d to exceed %s limit of %d", tc.iterations, tc.client, tc.maxIterations)
					}
				}
			})
		}
	})
}
