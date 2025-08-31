package kdf

import (
	"testing"
	"time"
)

func TestNewKDFCompatibilityAnalyzer(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	if analyzer == nil {
		t.Fatal("Expected analyzer to be created, got nil")
	}

	if analyzer.service != service {
		t.Error("Expected analyzer to use provided service")
	}
}

func TestAnalyzeKeystore_NilCrypto(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	report, err := analyzer.AnalyzeKeystore(nil)

	if err == nil {
		t.Fatal("Expected error for nil crypto params")
	}

	if report != nil {
		t.Error("Expected nil report for nil crypto params")
	}
}

func TestAnalyzeKeystore_UnsupportedKDF(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	crypto := &CryptoParams{
		KDF:       "unsupported-kdf",
		KDFParams: map[string]interface{}{},
	}

	report, err := analyzer.AnalyzeKeystore(crypto)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if report.Compatible {
		t.Error("Expected incompatible report for unsupported KDF")
	}

	if report.SecurityLevel != SecurityLevelLow {
		t.Errorf("Expected security level Low, got %s", report.SecurityLevel)
	}

	if len(report.Issues) == 0 {
		t.Error("Expected issues for unsupported KDF")
	}
}

func TestAnalyzeKeystore_ValidScrypt(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	crypto := &CryptoParams{
		KDF: "scrypt",
		KDFParams: map[string]interface{}{
			"n":     262144,
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

	if !report.Compatible {
		t.Error("Expected compatible report for valid scrypt params")
	}

	if report.SecurityLevel != SecurityLevelVeryHigh {
		t.Errorf("Expected security level Very High, got %s", report.SecurityLevel)
	}

	if report.KDFType != "scrypt" {
		t.Errorf("Expected KDF type 'scrypt', got %s", report.KDFType)
	}

	if report.NormalizedKDF != "scrypt" {
		t.Errorf("Expected normalized KDF 'scrypt', got %s", report.NormalizedKDF)
	}
}

func TestAnalyzeKeystore_ValidPBKDF2(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	crypto := &CryptoParams{
		KDF: "pbkdf2",
		KDFParams: map[string]interface{}{
			"c":     600000,
			"dklen": 32,
			"prf":   "hmac-sha256",
			"salt":  "0123456789abcdef0123456789abcdef",
		},
	}

	report, err := analyzer.AnalyzeKeystore(crypto)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !report.Compatible {
		t.Error("Expected compatible report for valid PBKDF2 params")
	}

	if report.SecurityLevel != SecurityLevelVeryHigh {
		t.Errorf("Expected security level Very High, got %s", report.SecurityLevel)
	}
}

func TestAnalyzeScryptSecurity_VeryHighSecurity(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	params := map[string]interface{}{
		"n": 262144, // 2^18
		"r": 8,
		"p": 1,
	}

	analysis := analyzer.analyzeScryptSecurity(params)

	if analysis.Level != SecurityLevelVeryHigh {
		t.Errorf("Expected Very High security level, got %s", analysis.Level)
	}

	expectedMemory := int64(128 * 262144 * 8) // 268MB
	if analysis.MemoryUsage != expectedMemory {
		t.Errorf("Expected memory usage %d, got %d", expectedMemory, analysis.MemoryUsage)
	}

	expectedCost := float64(262144 * 8 * 1)
	if analysis.ComputationalCost != expectedCost {
		t.Errorf("Expected computational cost %f, got %f", expectedCost, analysis.ComputationalCost)
	}
}

func TestAnalyzeScryptSecurity_LowSecurity(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	params := map[string]interface{}{
		"n": 1024, // Very low
		"r": 4,    // Low
		"p": 1,
	}

	analysis := analyzer.analyzeScryptSecurity(params)

	if analysis.Level != SecurityLevelLow {
		t.Errorf("Expected Low security level, got %s", analysis.Level)
	}

	if len(analysis.Recommendations) == 0 {
		t.Error("Expected recommendations for low security parameters")
	}
}

func TestAnalyzeScryptSecurity_NonPowerOfTwo(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	params := map[string]interface{}{
		"n": 65535, // Not power of 2
		"r": 8,
		"p": 1,
	}

	analysis := analyzer.analyzeScryptSecurity(params)

	// Check if recommendation about power of 2 is included
	found := false
	for _, rec := range analysis.Recommendations {
		if rec == "N parameter should be a power of 2 for optimal performance" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected recommendation about power of 2 for N parameter")
	}
}

func TestAnalyzePBKDF2Security_VeryHighSecurity(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	params := map[string]interface{}{
		"c":   600000,
		"prf": "hmac-sha256",
	}

	analysis := analyzer.analyzePBKDF2Security("pbkdf2", params)

	if analysis.Level != SecurityLevelVeryHigh {
		t.Errorf("Expected Very High security level, got %s", analysis.Level)
	}

	expectedCost := float64(600000)
	if analysis.ComputationalCost != expectedCost {
		t.Errorf("Expected computational cost %f, got %f", expectedCost, analysis.ComputationalCost)
	}
}

func TestAnalyzePBKDF2Security_WeakPRF(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	params := map[string]interface{}{
		"c":   600000,
		"prf": "hmac-sha1", // Weak PRF
	}

	analysis := analyzer.analyzePBKDF2Security("pbkdf2", params)

	// Security level should be downgraded due to weak PRF
	if analysis.Level == SecurityLevelVeryHigh {
		t.Error("Expected security level to be downgraded due to weak PRF")
	}

	// Should have recommendation about SHA-1
	found := false
	for _, rec := range analysis.Recommendations {
		if rec == "SHA-1 is deprecated, consider using SHA-256 or SHA-512" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected recommendation about deprecated SHA-1")
	}
}

func TestGetScryptClientCompatibility(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test standard parameters - should be compatible with most clients
	compatibility := analyzer.getScryptClientCompatibility(65536, 8, 1)

	expectedClients := []string{"geth", "besu", "reth"}
	for _, client := range expectedClients {
		if !compatibility[client] {
			t.Errorf("Expected %s to be compatible with standard scrypt parameters", client)
		}
	}

	// Test very high memory usage - should be incompatible with Firefly
	highMemoryCompatibility := analyzer.getScryptClientCompatibility(1048576, 16, 1)
	if highMemoryCompatibility["firefly"] {
		t.Error("Expected Firefly to be incompatible with very high memory usage")
	}
}

func TestGetPBKDF2ClientCompatibility(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test SHA-256 - should be compatible with all clients
	compatibility := analyzer.getPBKDF2ClientCompatibility(100000, "hmac-sha256")

	for client, compatible := range compatibility {
		if !compatible {
			t.Errorf("Expected %s to be compatible with PBKDF2 SHA-256", client)
		}
	}

	// Test SHA-1 - should be incompatible with most modern clients
	sha1Compatibility := analyzer.getPBKDF2ClientCompatibility(100000, "hmac-sha1")

	incompatibleClients := []string{"besu", "anvil", "reth", "firefly"}
	for _, client := range incompatibleClients {
		if sha1Compatibility[client] {
			t.Errorf("Expected %s to be incompatible with PBKDF2 SHA-1", client)
		}
	}
}

func TestEstimateDerivationTime(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test scrypt estimation
	scryptParams := map[string]interface{}{
		"n": 16384,
		"r": 8,
		"p": 1,
	}

	scryptTime := analyzer.EstimateDerivationTime("scrypt", scryptParams)
	if scryptTime <= 0 {
		t.Error("Expected positive time estimate for scrypt")
	}

	// Test PBKDF2 estimation
	pbkdf2Params := map[string]interface{}{
		"c":   100000,
		"prf": "hmac-sha256",
	}

	pbkdf2Time := analyzer.EstimateDerivationTime("pbkdf2", pbkdf2Params)
	if pbkdf2Time <= 0 {
		t.Error("Expected positive time estimate for PBKDF2")
	}

	// Test unknown KDF
	unknownTime := analyzer.EstimateDerivationTime("unknown", map[string]interface{}{})
	if unknownTime != time.Second {
		t.Errorf("Expected 1 second default for unknown KDF, got %v", unknownTime)
	}
}

func TestGetOptimizedParams_Scrypt(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test high security with memory constraint
	params, err := analyzer.GetOptimizedParams("scrypt", SecurityLevelHigh, 128) // 128MB limit

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	n, ok := params["n"].(int)
	if !ok {
		t.Fatal("Expected n parameter to be int")
	}

	r, ok := params["r"].(int)
	if !ok {
		t.Fatal("Expected r parameter to be int")
	}

	// Check memory usage is within constraint
	memoryUsage := 128 * n * r
	maxMemory := 128 * 1024 * 1024 // 128MB in bytes
	if memoryUsage > maxMemory {
		t.Errorf("Memory usage %d exceeds constraint %d", memoryUsage, maxMemory)
	}

	// Check that N is power of 2
	if n&(n-1) != 0 {
		t.Errorf("N parameter %d is not power of 2", n)
	}
}

func TestGetOptimizedParams_PBKDF2(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test PBKDF2-SHA256
	params, err := analyzer.GetOptimizedParams("pbkdf2-sha256", SecurityLevelHigh, 0)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	prf, ok := params["prf"].(string)
	if !ok {
		t.Fatal("Expected prf parameter to be string")
	}

	if prf != "hmac-sha256" {
		t.Errorf("Expected PRF to be hmac-sha256, got %s", prf)
	}

	c, ok := params["c"].(int)
	if !ok {
		t.Fatal("Expected c parameter to be int")
	}

	if c < 600000 {
		t.Errorf("Expected high security iteration count >= 600000, got %d", c)
	}
}

func TestGetOptimizedParams_UnsupportedKDF(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	params, err := analyzer.GetOptimizedParams("unsupported", SecurityLevelHigh, 0)

	if err == nil {
		t.Fatal("Expected error for unsupported KDF")
	}

	if params != nil {
		t.Error("Expected nil params for unsupported KDF")
	}
}

func TestFormatBytes(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, test := range tests {
		result := analyzer.formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestGetIntParam(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	tests := []struct {
		name     string
		params   map[string]interface{}
		key      string
		defVal   int
		expected int
	}{
		{
			name:     "int value",
			params:   map[string]interface{}{"test": 42},
			key:      "test",
			defVal:   10,
			expected: 42,
		},
		{
			name:     "int64 value",
			params:   map[string]interface{}{"test": int64(42)},
			key:      "test",
			defVal:   10,
			expected: 42,
		},
		{
			name:     "float64 value",
			params:   map[string]interface{}{"test": 42.0},
			key:      "test",
			defVal:   10,
			expected: 42,
		},
		{
			name:     "string value",
			params:   map[string]interface{}{"test": "42"},
			key:      "test",
			defVal:   10,
			expected: 42,
		},
		{
			name:     "missing key",
			params:   map[string]interface{}{},
			key:      "test",
			defVal:   10,
			expected: 10,
		},
		{
			name:     "invalid string",
			params:   map[string]interface{}{"test": "invalid"},
			key:      "test",
			defVal:   10,
			expected: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzer.getIntParam(test.params, test.key, test.defVal)
			if result != test.expected {
				t.Errorf("Expected %d, got %d", test.expected, result)
			}
		})
	}
}

func TestGetStringParam(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	tests := []struct {
		name     string
		params   map[string]interface{}
		key      string
		defVal   string
		expected string
	}{
		{
			name:     "string value",
			params:   map[string]interface{}{"test": "value"},
			key:      "test",
			defVal:   "default",
			expected: "value",
		},
		{
			name:     "missing key",
			params:   map[string]interface{}{},
			key:      "test",
			defVal:   "default",
			expected: "default",
		},
		{
			name:     "non-string value",
			params:   map[string]interface{}{"test": 42},
			key:      "test",
			defVal:   "default",
			expected: "default",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzer.getStringParam(test.params, test.key, test.defVal)
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestGenerateSecurityRecommendations(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test low security level
	lowSecurityAnalysis := &SecurityAnalysis{
		Level:           SecurityLevelLow,
		MemoryUsage:     1024,
		Recommendations: []string{"Increase parameters"},
	}

	warnings, suggestions := analyzer.generateSecurityRecommendations(lowSecurityAnalysis)

	if len(warnings) == 0 {
		t.Error("Expected warnings for low security level")
	}

	if len(suggestions) == 0 {
		t.Error("Expected suggestions for low security level")
	}

	// Test high memory usage
	highMemoryAnalysis := &SecurityAnalysis{
		Level:           SecurityLevelHigh,
		MemoryUsage:     2 * 1024 * 1024 * 1024, // 2GB
		Recommendations: []string{},
	}

	warnings, _ = analyzer.generateSecurityRecommendations(highMemoryAnalysis)

	found := false
	for _, warning := range warnings {
		if warning == "High memory usage: 2.0 GB" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected warning about high memory usage")
	}
}

func TestGenerateClientWarnings(t *testing.T) {
	service := NewUniversalKDFService()
	analyzer := NewKDFCompatibilityAnalyzer(service)

	// Test with some incompatible clients
	compatibility := map[string]bool{
		"geth":    true,
		"besu":    false,
		"anvil":   false,
		"reth":    true,
		"firefly": true,
	}

	warnings := analyzer.generateClientWarnings(compatibility)

	if len(warnings) == 0 {
		t.Error("Expected warnings for incompatible clients")
	}

	// Check that warning mentions incompatible clients
	found := false
	for _, warning := range warnings {
		if warning == "Incompatible with clients: [besu anvil]" || warning == "Incompatible with clients: [anvil besu]" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected warning about incompatible clients, got: %v", warnings)
	}
}
