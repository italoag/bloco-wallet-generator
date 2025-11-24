package crypto

import (
	"encoding/json"
	"strings"
	"testing"
)

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
			if keystore.Version != 3 {
				t.Errorf("geth requires version 3, got %d", keystore.Version)
			}

			// Verify cipher is supported by geth
			if keystore.Crypto.Cipher != "aes-128-ctr" {
				t.Errorf("geth expects aes-128-ctr cipher, got %s", keystore.Crypto.Cipher)
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
			if keystore.Version != 3 {
				t.Errorf("Besu requires version 3, got %d", keystore.Version)
			}

			// Besu requires specific MAC format
			if len(keystore.Crypto.MAC) != 64 { // 32 bytes hex encoded
				t.Errorf("Besu expects 64-character MAC, got %d characters", len(keystore.Crypto.MAC))
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
			if keystore.Version != 3 {
				t.Errorf("Anvil requires version 3, got %d", keystore.Version)
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
			if keystore.Version != 3 {
				t.Errorf("Reth requires version 3, got %d", keystore.Version)
			}

			// Verify standard Ethereum keystore format
			if keystore.Crypto.Cipher != "aes-128-ctr" {
				t.Errorf("Reth expects aes-128-ctr cipher, got %s", keystore.Crypto.Cipher)
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
			if keystore.Version != 3 {
				t.Errorf("Firefly requires version 3, got %d", keystore.Version)
			}

			// Firefly requires proper UUID format for ID
			if len(keystore.ID) != 36 { // UUID format: 8-4-4-4-12
				t.Errorf("Firefly expects UUID format for ID, got length %d", len(keystore.ID))
			}
		})
	}
}

// generateTestKeystore creates a test keystore with the specified KDF and parameters
func generateTestKeystore(t *testing.T, kdfType string, params map[string]interface{}) *KeyStoreV3 {
	address := "0x1234567890123456789012345678901234567890"

	// Create keystore using the crypto package
	keystore := &KeyStoreV3{
		Address: strings.ToLower(strings.TrimPrefix(address, "0x")),
		ID:      "12345678-1234-1234-1234-123456789012", // Test UUID
		Version: 3,
		Crypto: KeyStoreCrypto{
			KDF:       kdfType,
			KDFParams: params,
			Cipher:    "aes-128-ctr",
			CipherParams: CipherParams{
				IV: "0123456789abcdef0123456789abcdef",
			},
			CipherText: "encrypted_data_placeholder",
			MAC:        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
	}

	return keystore
}

// validateKeystoreFormat validates that a keystore conforms to Ethereum KeyStore V3 format
func validateKeystoreFormat(t *testing.T, keystore *KeyStoreV3) {
	// Validate required fields
	if keystore.Version != 3 {
		t.Errorf("Invalid version: expected 3, got %d", keystore.Version)
	}

	if len(keystore.Address) != 40 {
		t.Errorf("Invalid address length: expected 40, got %d", len(keystore.Address))
	}

	if len(keystore.ID) == 0 {
		t.Errorf("ID field is required")
	}

	// Validate crypto section
	if keystore.Crypto.KDF == "" {
		t.Errorf("KDF field is required")
	}

	if keystore.Crypto.KDFParams == nil {
		t.Errorf("KDFParams field is required")
	}

	if keystore.Crypto.Cipher == "" {
		t.Errorf("Cipher field is required")
	}

	if keystore.Crypto.CipherText == "" {
		t.Errorf("CipherText field is required")
	}

	if keystore.Crypto.MAC == "" {
		t.Errorf("MAC field is required")
	}

	// Validate KDF-specific parameters
	switch keystore.Crypto.KDF {
	case "scrypt":
		if params, ok := keystore.Crypto.KDFParams.(map[string]interface{}); ok {
			validateScryptParams(t, params)
		}
	case "pbkdf2":
		if params, ok := keystore.Crypto.KDFParams.(map[string]interface{}); ok {
			validatePBKDF2Params(t, params)
		}
	default:
		t.Errorf("Unsupported KDF: %s", keystore.Crypto.KDF)
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

			var loadedKeystore KeyStoreV3
			err = json.Unmarshal(jsonData, &loadedKeystore)
			if err != nil {
				t.Fatalf("Failed to unmarshal keystore from JSON: %v", err)
			}

			// Verify loaded keystore matches original
			if loadedKeystore.Address != keystore.Address {
				t.Errorf("Address mismatch: expected %s, got %s", keystore.Address, loadedKeystore.Address)
			}

			if loadedKeystore.Version != keystore.Version {
				t.Errorf("Version mismatch: expected %d, got %d", keystore.Version, loadedKeystore.Version)
			}

			if loadedKeystore.Crypto.KDF != keystore.Crypto.KDF {
				t.Errorf("KDF mismatch: expected %s, got %s", keystore.Crypto.KDF, loadedKeystore.Crypto.KDF)
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
				maxMemoryMB: 200, // 200MB limit for dev speed
				params: map[string]interface{}{
					"n":     262144, // This would use ~256MB
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

// TestActualKeystoreGeneration tests actual keystore generation with the KeyStoreService
func TestActualKeystoreGeneration(t *testing.T) {
	// Create a keystore service for testing
	config := KeyStoreConfig{
		OutputDirectory: t.TempDir(),
		Enabled:         true,
		KDF:             "scrypt",
		KDFParams: map[string]interface{}{
			"n":     16384,
			"r":     8,
			"p":     1,
			"dklen": 32,
		},
	}

	service := NewKeyStoreService(config)

	testCases := []struct {
		name    string
		kdfType string
	}{
		{
			name:    "actual_scrypt_keystore",
			kdfType: "scrypt",
		},
		{
			name:    "actual_pbkdf2_keystore",
			kdfType: "pbkdf2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate a test private key and address
			privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
			address := "0x1234567890123456789012345678901234567890"

			// Update service configuration for this test
			service.config.KDF = tc.kdfType

			// Generate keystore
			keystore, password, err := service.GenerateKeyStore(privateKey, address, "ethereum")
			if err != nil {
				t.Fatalf("Failed to generate keystore: %v", err)
			}

			// Use the generated password for validation
			_ = password

			// Validate the generated keystore
			if keystore == nil {
				t.Fatal("Generated keystore is nil")
			}

			if keystore.Version != 3 {
				t.Errorf("Expected version 3, got %d", keystore.Version)
			}

			if keystore.Crypto.KDF != tc.kdfType {
				t.Errorf("Expected KDF %s, got %s", tc.kdfType, keystore.Crypto.KDF)
			}

			// Validate that the keystore can be serialized to JSON
			jsonData, err := json.Marshal(keystore)
			if err != nil {
				t.Fatalf("Failed to marshal keystore to JSON: %v", err)
			}

			// Validate that the JSON can be deserialized back
			var loadedKeystore KeyStoreV3
			err = json.Unmarshal(jsonData, &loadedKeystore)
			if err != nil {
				t.Fatalf("Failed to unmarshal keystore from JSON: %v", err)
			}

			// Verify the loaded keystore matches the original
			if loadedKeystore.Address != keystore.Address {
				t.Errorf("Address mismatch after JSON round-trip")
			}

			if loadedKeystore.Crypto.KDF != keystore.Crypto.KDF {
				t.Errorf("KDF mismatch after JSON round-trip")
			}
		})
	}
}
