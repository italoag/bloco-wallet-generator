package kdf

import (
	"testing"
)

func TestUniversalKDFService_Basic(t *testing.T) {
	service := NewUniversalKDFService()

	// Test that default KDFs are registered
	supportedKDFs := service.GetSupportedKDFs()
	if len(supportedKDFs) == 0 {
		t.Error("Expected supported KDFs to be registered")
	}

	// Test KDF support check
	if !service.IsKDFSupported("scrypt") {
		t.Error("Expected scrypt to be supported")
	}

	if !service.IsKDFSupported("pbkdf2") {
		t.Error("Expected pbkdf2 to be supported")
	}

	if service.IsKDFSupported("unsupported") {
		t.Error("Expected unsupported KDF to return false")
	}
}

func TestUniversalKDFService_KDFNormalization(t *testing.T) {
	service := NewUniversalKDFService()

	tests := []struct {
		input    string
		expected string
	}{
		{"scrypt", "scrypt"},
		{"Scrypt", "scrypt"},
		{"SCRYPT", "scrypt"},
		{"pbkdf2", "pbkdf2"},
		{"PBKDF2", "pbkdf2"},
		{"pbkdf2-sha256", "pbkdf2-sha256"},
		{"PBKDF2-SHA256", "pbkdf2-sha256"},
		{"pbkdf2_sha256", "pbkdf2-sha256"},
		{"pbkdf2sha256", "pbkdf2-sha256"},
		{"unknown", "unknown"},
	}

	for _, test := range tests {
		result := service.normalizeKDFName(test.input)
		if result != test.expected {
			t.Errorf("normalizeKDFName(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestUniversalKDFService_DefaultParams(t *testing.T) {
	service := NewUniversalKDFService()

	// Test scrypt default params
	params, err := service.GetDefaultParams("scrypt")
	if err != nil {
		t.Errorf("GetDefaultParams(scrypt) failed: %v", err)
	}
	if params == nil {
		t.Error("Expected default params for scrypt")
	}

	// Test pbkdf2 default params
	params, err = service.GetDefaultParams("pbkdf2")
	if err != nil {
		t.Errorf("GetDefaultParams(pbkdf2) failed: %v", err)
	}
	if params == nil {
		t.Error("Expected default params for pbkdf2")
	}

	// Test unsupported KDF
	_, err = service.GetDefaultParams("unsupported")
	if err == nil {
		t.Error("Expected error for unsupported KDF")
	}
}

func TestUniversalKDFService_ParamValidation(t *testing.T) {
	service := NewUniversalKDFService()

	// Test scrypt param validation with valid params
	validScryptParams := map[string]interface{}{
		"n":     262144,
		"r":     8,
		"p":     1,
		"dklen": 32,
		"salt":  "0123456789abcdef0123456789abcdef",
	}

	err := service.ValidateParams("scrypt", validScryptParams)
	if err != nil {
		t.Errorf("ValidateParams(scrypt) with valid params failed: %v", err)
	}

	// Test scrypt param validation with invalid N (not power of 2)
	invalidScryptParams := map[string]interface{}{
		"n":     100000, // Not a power of 2
		"r":     8,
		"p":     1,
		"dklen": 32,
		"salt":  "0123456789abcdef0123456789abcdef",
	}

	err = service.ValidateParams("scrypt", invalidScryptParams)
	if err == nil {
		t.Error("Expected validation error for invalid N parameter")
	}
}

func TestUniversalKDFService_GetParamRange(t *testing.T) {
	service := NewUniversalKDFService()

	// Test scrypt parameter ranges
	min, max, err := service.GetParamRange("scrypt", "n")
	if err != nil {
		t.Errorf("GetParamRange(scrypt, n) failed: %v", err)
	}
	if min == nil || max == nil {
		t.Error("Expected min and max values for scrypt n parameter")
	}

	// Test invalid parameter
	_, _, err = service.GetParamRange("scrypt", "invalid")
	if err == nil {
		t.Error("Expected error for invalid parameter")
	}

	// Test unsupported KDF
	_, _, err = service.GetParamRange("unsupported", "n")
	if err == nil {
		t.Error("Expected error for unsupported KDF")
	}
}

func TestUniversalKDFService_GetKDFAliases(t *testing.T) {
	service := NewUniversalKDFService()

	// Test scrypt aliases
	aliases := service.GetKDFAliases("scrypt")
	if len(aliases) == 0 {
		t.Error("Expected aliases for scrypt")
	}

	// Check that all aliases normalize to scrypt
	for _, alias := range aliases {
		normalized := service.normalizeKDFName(alias)
		if normalized != "scrypt" {
			t.Errorf("Alias %s should normalize to scrypt, got %s", alias, normalized)
		}
	}

	// Test pbkdf2-sha256 aliases
	aliases = service.GetKDFAliases("pbkdf2-sha256")
	if len(aliases) == 0 {
		t.Error("Expected aliases for pbkdf2-sha256")
	}
}

func TestUniversalKDFService_GetRecommendedKDF(t *testing.T) {
	service := NewUniversalKDFService()

	tests := []struct {
		level    SecurityLevel
		expected string
	}{
		{SecurityLevelLow, "pbkdf2"},
		{SecurityLevelMedium, "scrypt"},
		{SecurityLevelHigh, "scrypt"},
		{SecurityLevelVeryHigh, "scrypt"},
	}

	for _, test := range tests {
		result := service.GetRecommendedKDF(test.level)
		if result != test.expected {
			t.Errorf("GetRecommendedKDF(%s) = %s, expected %s", test.level, result, test.expected)
		}
	}
}

func TestKDFError(t *testing.T) {
	err := NewKDFError("validation", "scrypt", "n", 100, "power of 2", "N must be power of 2")

	if err.Type != "validation" {
		t.Errorf("Expected Type 'validation', got %s", err.Type)
	}

	if err.KDFType != "scrypt" {
		t.Errorf("Expected KDFType 'scrypt', got %s", err.KDFType)
	}

	if !err.IsRecoverable() {
		t.Error("Expected validation error to be recoverable")
	}

	// Test with suggestions
	err = err.WithSuggestions("Use power of 2", "Try 262144")
	if len(err.GetSuggestions()) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(err.GetSuggestions()))
	}

	// Test error message
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
}
