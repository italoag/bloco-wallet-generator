package crypto

import (
	"bytes"
	"strings"
	"testing"
)

func TestEntropyValidator_ValidateEntropy(t *testing.T) {
	validator := NewEntropyValidator()

	tests := []struct {
		name      string
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "empty data",
			data:      []byte{},
			wantError: true,
			errorMsg:  "entropy data cannot be empty",
		},
		{
			name:      "insufficient entropy",
			data:      make([]byte, 16), // Less than minimum 32 bytes
			wantError: true,
			errorMsg:  "insufficient entropy",
		},
		{
			name:      "all zeros",
			data:      make([]byte, 32), // All zeros
			wantError: true,
			errorMsg:  "entropy contains all zeros",
		},
		{
			name:      "valid random data",
			data:      []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEntropy(tt.data)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateEntropy() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateEntropy() error = %v, want error containing %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEntropy() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSecureRandomGenerator_GenerateRandomBytes(t *testing.T) {
	generator := NewSecureRandomGenerator()

	tests := []struct {
		name      string
		length    int
		wantError bool
		errorMsg  string
	}{
		{
			name:      "zero length",
			length:    0,
			wantError: true,
			errorMsg:  "length must be positive",
		},
		{
			name:      "valid medium length",
			length:    32,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.GenerateRandomBytes(tt.length)
			if tt.wantError {
				if err == nil {
					t.Errorf("GenerateRandomBytes() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateRandomBytes() error = %v, want error containing %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateRandomBytes() unexpected error = %v", err)
				}
				if len(result) != tt.length {
					t.Errorf("GenerateRandomBytes() length = %d, want %d", len(result), tt.length)
				}

				// Generate another set and ensure they're different
				if tt.length >= 16 {
					result2, err2 := generator.GenerateRandomBytes(tt.length)
					if err2 != nil {
						t.Errorf("GenerateRandomBytes() second call error = %v", err2)
					}
					if bytes.Equal(result, result2) {
						t.Errorf("GenerateRandomBytes() generated identical bytes twice")
					}
				}
			}
		})
	}
}

func TestMemoryCleaner_ClearBytes(t *testing.T) {
	cleaner := NewMemoryCleaner()

	data := []byte{0x12, 0x34, 0x56, 0x78}
	cleaner.ClearBytes(data)

	// Verify all bytes are zero
	for i, b := range data {
		if b != 0 {
			t.Errorf("ClearBytes() byte at index %d = %d, want 0", i, b)
		}
	}
}

func TestGlobalFunctions(t *testing.T) {
	t.Run("GenerateRandomBytesEnhanced", func(t *testing.T) {
		result, err := GenerateRandomBytesEnhanced(32)
		if err != nil {
			t.Errorf("GenerateRandomBytesEnhanced() error = %v", err)
		}
		if len(result) != 32 {
			t.Errorf("GenerateRandomBytesEnhanced() length = %d, want 32", len(result))
		}
	})

	t.Run("GenerateSecureSalt", func(t *testing.T) {
		salt, err := GenerateSecureSalt(32)
		if err != nil {
			t.Errorf("GenerateSecureSalt() error = %v", err)
		}
		if len(salt) != 32 {
			t.Errorf("GenerateSecureSalt() length = %d, want 32", len(salt))
		}
	})

	t.Run("ClearSensitiveData", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		ClearSensitiveData(data)
		for i, b := range data {
			if b != 0 {
				t.Errorf("ClearSensitiveData() byte at index %d = %d, want 0", i, b)
			}
		}
	})
}
