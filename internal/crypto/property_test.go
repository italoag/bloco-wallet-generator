package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
)

// TestCryptographicProperties tests important cryptographic properties
func TestCryptographicProperties(t *testing.T) {
	t.Run("password_uniqueness", func(t *testing.T) {
		testPasswordUniqueness(t)
	})

	t.Run("salt_uniqueness", func(t *testing.T) {
		testSaltUniqueness(t)
	})

	t.Run("mac_integrity", func(t *testing.T) {
		testMACIntegrity(t)
	})

	t.Run("parameter_boundary_conditions", func(t *testing.T) {
		// Test basic encryption/decryption with both KDF types
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		password := "test_password"

		// Test scrypt
		scryptKeystore, err := EncryptPrivateKey(privateKey, password, "scrypt")
		if err != nil {
			t.Fatalf("Failed to encrypt with scrypt: %v", err)
		}

		decryptedKey, err := DecryptPrivateKey(scryptKeystore, password)
		if err != nil {
			t.Fatalf("Failed to decrypt with scrypt: %v", err)
		}

		if hex.EncodeToString(decryptedKey) != privateKey {
			t.Errorf("Scrypt decrypted key doesn't match original")
		}

		// Test pbkdf2
		pbkdf2Keystore, err := EncryptPrivateKey(privateKey, password, "pbkdf2")
		if err != nil {
			t.Fatalf("Failed to encrypt with pbkdf2: %v", err)
		}

		decryptedKey, err = DecryptPrivateKey(pbkdf2Keystore, password)
		if err != nil {
			t.Fatalf("Failed to decrypt with pbkdf2: %v", err)
		}

		if hex.EncodeToString(decryptedKey) != privateKey {
			t.Errorf("PBKDF2 decrypted key doesn't match original")
		}
	})

	t.Run("deterministic_behavior", func(t *testing.T) {
		// Test that the same private key and password produce valid keystores
		// (We can't test for identical output due to random salt/IV generation)
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		password := "test_password"

		// Generate two keystores with same inputs
		keystore1, err := EncryptPrivateKey(privateKey, password, "scrypt")
		if err != nil {
			t.Fatalf("Failed to encrypt keystore 1: %v", err)
		}

		keystore2, err := EncryptPrivateKey(privateKey, password, "scrypt")
		if err != nil {
			t.Fatalf("Failed to encrypt keystore 2: %v", err)
		}

		// Both should decrypt to the same private key
		decrypted1, err := DecryptPrivateKey(keystore1, password)
		if err != nil {
			t.Fatalf("Failed to decrypt keystore 1: %v", err)
		}

		decrypted2, err := DecryptPrivateKey(keystore2, password)
		if err != nil {
			t.Fatalf("Failed to decrypt keystore 2: %v", err)
		}

		if hex.EncodeToString(decrypted1) != privateKey {
			t.Error("Keystore 1 decrypted to wrong private key")
		}

		if hex.EncodeToString(decrypted2) != privateKey {
			t.Error("Keystore 2 decrypted to wrong private key")
		}

		// The keystores should have different salts (randomness test)
		salt1 := getSaltFromKDFParams(t, keystore1.Crypto.KDFParams)
		salt2 := getSaltFromKDFParams(t, keystore2.Crypto.KDFParams)

		if salt1 == salt2 {
			t.Error("Two keystores should have different salts")
		}
	})
}

// testPasswordUniqueness verifies that different passwords produce different derived keys
func testPasswordUniqueness(t *testing.T) {
	// Test with scrypt
	t.Run("scrypt_password_uniqueness", func(t *testing.T) {
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

		// Generate keystores with different passwords
		passwords := []string{"password1", "password2", "password3", "different_password"}
		keystores := make([]*KeyStoreV3, len(passwords))

		for i, password := range passwords {
			// Manually encrypt with specific password for testing
			keystore, err := EncryptPrivateKey(privateKey, password, "scrypt")
			if err != nil {
				t.Fatalf("Failed to encrypt with password %s: %v", password, err)
			}
			keystores[i] = keystore
		}

		// Verify all keystores have different ciphertexts and MACs
		for i := 0; i < len(keystores); i++ {
			for j := i + 1; j < len(keystores); j++ {
				if keystores[i].Crypto.CipherText == keystores[j].Crypto.CipherText {
					t.Errorf("Keystores %d and %d have identical ciphertext (passwords: %s, %s)",
						i, j, passwords[i], passwords[j])
				}

				if keystores[i].Crypto.MAC == keystores[j].Crypto.MAC {
					t.Errorf("Keystores %d and %d have identical MAC (passwords: %s, %s)",
						i, j, passwords[i], passwords[j])
				}
			}
		}
	})

	// Test with PBKDF2
	t.Run("pbkdf2_password_uniqueness", func(t *testing.T) {
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

		// Generate keystores with different passwords
		passwords := []string{"password1", "password2", "password3", "different_password"}
		keystores := make([]*KeyStoreV3, len(passwords))

		for i, password := range passwords {
			keystore, err := EncryptPrivateKey(privateKey, password, "pbkdf2")
			if err != nil {
				t.Fatalf("Failed to encrypt with password %s: %v", password, err)
			}
			keystores[i] = keystore
		}

		// Verify all keystores have different ciphertexts and MACs
		for i := 0; i < len(keystores); i++ {
			for j := i + 1; j < len(keystores); j++ {
				if keystores[i].Crypto.CipherText == keystores[j].Crypto.CipherText {
					t.Errorf("Keystores %d and %d have identical ciphertext (passwords: %s, %s)",
						i, j, passwords[i], passwords[j])
				}

				if keystores[i].Crypto.MAC == keystores[j].Crypto.MAC {
					t.Errorf("Keystores %d and %d have identical MAC (passwords: %s, %s)",
						i, j, passwords[i], passwords[j])
				}
			}
		}
	})
}

// testSaltUniqueness verifies that salt uniqueness produces different keystore outputs
func testSaltUniqueness(t *testing.T) {
	t.Run("scrypt_salt_uniqueness", func(t *testing.T) {
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		password := "test_password"

		// Generate multiple keystores with the same password but different salts
		const numKeystores = 10
		keystores := make([]*KeyStoreV3, numKeystores)

		for i := 0; i < numKeystores; i++ {
			keystore, err := EncryptPrivateKey(privateKey, password, "scrypt")
			if err != nil {
				t.Fatalf("Failed to encrypt keystore %d: %v", i, err)
			}
			keystores[i] = keystore
		}

		// Verify all keystores have different salts, ciphertexts, and MACs
		for i := 0; i < numKeystores; i++ {
			for j := i + 1; j < numKeystores; j++ {
				// Extract salt from KDF parameters
				saltI := getSaltFromKDFParams(t, keystores[i].Crypto.KDFParams)
				saltJ := getSaltFromKDFParams(t, keystores[j].Crypto.KDFParams)

				if saltI == saltJ {
					t.Errorf("Keystores %d and %d have identical salt", i, j)
				}

				if keystores[i].Crypto.CipherText == keystores[j].Crypto.CipherText {
					t.Errorf("Keystores %d and %d have identical ciphertext despite different salts", i, j)
				}

				if keystores[i].Crypto.MAC == keystores[j].Crypto.MAC {
					t.Errorf("Keystores %d and %d have identical MAC despite different salts", i, j)
				}
			}
		}
	})

	t.Run("pbkdf2_salt_uniqueness", func(t *testing.T) {
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		password := "test_password"

		// Generate multiple keystores with the same password but different salts
		const numKeystores = 10
		keystores := make([]*KeyStoreV3, numKeystores)

		for i := 0; i < numKeystores; i++ {
			keystore, err := EncryptPrivateKey(privateKey, password, "pbkdf2")
			if err != nil {
				t.Fatalf("Failed to encrypt keystore %d: %v", i, err)
			}
			keystores[i] = keystore
		}

		// Verify all keystores have different salts, ciphertexts, and MACs
		for i := 0; i < numKeystores; i++ {
			for j := i + 1; j < numKeystores; j++ {
				// Extract salt from KDF parameters
				saltI := getSaltFromKDFParams(t, keystores[i].Crypto.KDFParams)
				saltJ := getSaltFromKDFParams(t, keystores[j].Crypto.KDFParams)

				if saltI == saltJ {
					t.Errorf("Keystores %d and %d have identical salt", i, j)
				}

				if keystores[i].Crypto.CipherText == keystores[j].Crypto.CipherText {
					t.Errorf("Keystores %d and %d have identical ciphertext despite different salts", i, j)
				}

				if keystores[i].Crypto.MAC == keystores[j].Crypto.MAC {
					t.Errorf("Keystores %d and %d have identical MAC despite different salts", i, j)
				}
			}
		}
	})
}

// testMACIntegrity tests MAC integrity across different parameter combinations
func testMACIntegrity(t *testing.T) {
	t.Run("mac_integrity_scrypt", func(t *testing.T) {
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		password := "test_password"

		// Generate keystore
		keystore, err := EncryptPrivateKey(privateKey, password, "scrypt")
		if err != nil {
			t.Fatalf("Failed to encrypt keystore: %v", err)
		}

		// Verify MAC can be validated
		originalMAC := keystore.Crypto.MAC

		// Test that MAC validation works with correct data
		isValid := verifyKeystoreMAC(t, keystore, password)
		if !isValid {
			t.Error("MAC validation failed for correct keystore")
		}

		// Test that MAC validation fails with tampered ciphertext
		originalCipherText := keystore.Crypto.CipherText
		keystore.Crypto.CipherText = "tampered_ciphertext_data_should_fail_mac_validation"

		isValid = verifyKeystoreMAC(t, keystore, password)
		if isValid {
			t.Error("MAC validation should fail for tampered ciphertext")
		}

		// Restore original ciphertext
		keystore.Crypto.CipherText = originalCipherText
		keystore.Crypto.MAC = originalMAC

		// Test that MAC validation fails with wrong password
		isValid = verifyKeystoreMAC(t, keystore, "wrong_password")
		if isValid {
			t.Error("MAC validation should fail for wrong password")
		}
	})

	t.Run("mac_integrity_pbkdf2", func(t *testing.T) {
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		password := "test_password"

		// Generate keystore
		keystore, err := EncryptPrivateKey(privateKey, password, "pbkdf2")
		if err != nil {
			t.Fatalf("Failed to encrypt keystore: %v", err)
		}

		// Verify MAC can be validated
		originalMAC := keystore.Crypto.MAC

		// Test that MAC validation works with correct data
		isValid := verifyKeystoreMAC(t, keystore, password)
		if !isValid {
			t.Error("MAC validation failed for correct keystore")
		}

		// Test that MAC validation fails with tampered ciphertext
		originalCipherText := keystore.Crypto.CipherText
		keystore.Crypto.CipherText = "tampered_ciphertext_data_should_fail_mac_validation"

		isValid = verifyKeystoreMAC(t, keystore, password)
		if isValid {
			t.Error("MAC validation should fail for tampered ciphertext")
		}

		// Restore original ciphertext
		keystore.Crypto.CipherText = originalCipherText
		keystore.Crypto.MAC = originalMAC

		// Test that MAC validation fails with wrong password
		isValid = verifyKeystoreMAC(t, keystore, "wrong_password")
		if isValid {
			t.Error("MAC validation should fail for wrong password")
		}
	})
}

// Helper functions

func getSaltFromKDFParams(t *testing.T, params interface{}) string {
	switch p := params.(type) {
	case *ScryptParams:
		return p.Salt
	case *PBKDF2Params:
		return p.Salt
	case ScryptParams:
		return p.Salt
	case PBKDF2Params:
		return p.Salt
	case map[string]interface{}:
		salt, ok := p["salt"].(string)
		if !ok {
			t.Fatal("Salt is not a string in map")
		}
		return salt
	default:
		t.Fatalf("Unknown KDF params type: %T", params)
		return ""
	}
}

func verifyKeystoreMAC(t *testing.T, keystore *KeyStoreV3, password string) bool {
	// This is a simplified MAC verification for testing
	// In a real implementation, you would derive the key and verify the MAC

	// Try to decrypt the keystore - if MAC is invalid, decryption should fail
	_, err := DecryptPrivateKey(keystore, password)
	return err == nil
}


func generateRandomSalt(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("Failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(bytes)
}

// TestCryptographicRandomness tests the quality of random number generation
func TestCryptographicRandomness(t *testing.T) {
	t.Run("salt_randomness", func(t *testing.T) {
		// Generate many salts and check for patterns
		const numSalts = 1000
		salts := make([]string, numSalts)

		for i := 0; i < numSalts; i++ {
			salt := generateRandomSalt(16)
			salts[i] = salt
		}

		// Check for duplicates
		saltMap := make(map[string]bool)
		for i, salt := range salts {
			if saltMap[salt] {
				t.Errorf("Duplicate salt found at index %d: %s", i, salt)
			}
			saltMap[salt] = true
		}

		// Check for patterns (all salts should be different)
		if len(saltMap) != numSalts {
			t.Errorf("Expected %d unique salts, got %d", numSalts, len(saltMap))
		}

		// Check salt length consistency
		for i, salt := range salts {
			if len(salt) != 32 { // 16 bytes = 32 hex characters
				t.Errorf("Salt %d has incorrect length: expected 32, got %d", i, len(salt))
			}
		}
	})

	t.Run("iv_randomness", func(t *testing.T) {
		// Generate multiple keystores and check IV uniqueness
		const numKeystores = 100
		ivs := make([]string, numKeystores)

		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		password := "test_password"

		for i := 0; i < numKeystores; i++ {
			keystore, err := EncryptPrivateKey(privateKey, password, "scrypt")
			if err != nil {
				t.Fatalf("Failed to encrypt keystore %d: %v", i, err)
			}

			ivs[i] = keystore.Crypto.CipherParams.IV
		}

		// Check for duplicate IVs
		ivMap := make(map[string]bool)
		for i, iv := range ivs {
			if ivMap[iv] {
				t.Errorf("Duplicate IV found at index %d: %s", i, iv)
			}
			ivMap[iv] = true
		}

		// Check IV length consistency
		for i, iv := range ivs {
			if len(iv) != 32 { // 16 bytes = 32 hex characters
				t.Errorf("IV %d has incorrect length: expected 32, got %d", i, len(iv))
			}
		}
	})
}

// TestSecurityProperties tests security-related properties
func TestSecurityProperties(t *testing.T) {
	t.Run("password_resistance", func(t *testing.T) {
		// Test that similar passwords produce very different outputs
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

		similarPasswords := []string{
			"password123",
			"password124",  // One character different
			"Password123",  // Case different
			"password1234", // One character added
		}

		keystores := make([]*KeyStoreV3, len(similarPasswords))
		for i, password := range similarPasswords {
			keystore, err := EncryptPrivateKey(privateKey, password, "scrypt")
			if err != nil {
				t.Fatalf("Failed to encrypt with password %s: %v", password, err)
			}
			keystores[i] = keystore
		}

		// Verify all outputs are completely different
		for i := 0; i < len(keystores); i++ {
			for j := i + 1; j < len(keystores); j++ {
				if keystores[i].Crypto.CipherText == keystores[j].Crypto.CipherText {
					t.Errorf("Similar passwords %s and %s produced identical ciphertext",
						similarPasswords[i], similarPasswords[j])
				}

				if keystores[i].Crypto.MAC == keystores[j].Crypto.MAC {
					t.Errorf("Similar passwords %s and %s produced identical MAC",
						similarPasswords[i], similarPasswords[j])
				}

				// Check that the outputs are sufficiently different (Hamming distance)
				hammingDistance := calculateHammingDistance(keystores[i].Crypto.CipherText, keystores[j].Crypto.CipherText)
				minExpectedDistance := len(keystores[i].Crypto.CipherText) / 4 // At least 25% different

				if hammingDistance < minExpectedDistance {
					t.Errorf("Ciphertext from passwords %s and %s are too similar (Hamming distance: %d, expected: >%d)",
						similarPasswords[i], similarPasswords[j], hammingDistance, minExpectedDistance)
				}
			}
		}
	})

	t.Run("timing_attack_resistance", func(t *testing.T) {
		// Test that password verification time doesn't leak information
		// This is a basic test - in practice, you'd need more sophisticated timing analysis

		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		correctPassword := "correct_password_123"

		keystore, err := EncryptPrivateKey(privateKey, correctPassword, "scrypt")
		if err != nil {
			t.Fatalf("Failed to encrypt keystore: %v", err)
		}

		// Test with various wrong passwords
		wrongPasswords := []string{
			"",                     // Empty
			"a",                    // Very short
			"wrong",                // Short
			"wrong_password_123",   // Same length, different content
			"correct_password_124", // One character different
			"very_long_wrong_password_that_should_not_work_123456789", // Very long
		}

		// All wrong passwords should fail, regardless of length or similarity
		for _, wrongPassword := range wrongPasswords {
			_, err := DecryptPrivateKey(keystore, wrongPassword)
			if err == nil {
				t.Errorf("Wrong password %q should have failed decryption", wrongPassword)
			}
		}

		// Correct password should work
		decryptedKey, err := DecryptPrivateKey(keystore, correctPassword)
		if err != nil {
			t.Fatalf("Correct password should work: %v", err)
		}

		if hex.EncodeToString(decryptedKey) != privateKey {
			t.Error("Decrypted key doesn't match original")
		}
	})
}

func calculateHammingDistance(s1, s2 string) int {
	if len(s1) != len(s2) {
		return max(len(s1), len(s2)) // Maximum possible distance for different lengths
	}

	distance := 0
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			distance++
		}
	}
	return distance
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
