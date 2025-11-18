package crypto

import (
	"testing"

	"bloco-eth/pkg/wallet"
)

func TestEthereumAdapter_GenerateKeyMaterial(t *testing.T) {
	adapter := NewEthereumAdapter()

	km, err := adapter.GenerateKeyMaterial()
	if err != nil {
		t.Fatalf("GenerateKeyMaterial() failed: %v", err)
	}

	if km == nil {
		t.Fatal("GenerateKeyMaterial() returned nil")
	}

	if len(km.PrivateKey) != 32 {
		t.Errorf("Private key length = %d, want 32", len(km.PrivateKey))
	}

	if len(km.PublicKey) == 0 {
		t.Error("Public key is empty")
	}

	if km.Metadata == nil {
		t.Error("Metadata is nil")
	}

	if km.Metadata["curve"] != "secp256k1" {
		t.Errorf("Metadata curve = %v, want secp256k1", km.Metadata["curve"])
	}
}

func TestEthereumAdapter_FormatAddress(t *testing.T) {
	adapter := NewEthereumAdapter()

	// Generate key material first
	km, err := adapter.GenerateKeyMaterial()
	if err != nil {
		t.Fatalf("GenerateKeyMaterial() failed: %v", err)
	}

	address, err := adapter.FormatAddress(km)
	if err != nil {
		t.Fatalf("FormatAddress() failed: %v", err)
	}

	if len(address) != 40 {
		t.Errorf("Address length = %d, want 40", len(address))
	}

	// Check if address is valid hex
	for _, char := range address {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			t.Errorf("Address contains non-hex character: %c", char)
		}
	}
}

func TestEthereumAdapter_FormatPrivateKey(t *testing.T) {
	adapter := NewEthereumAdapter()

	km, err := adapter.GenerateKeyMaterial()
	if err != nil {
		t.Fatalf("GenerateKeyMaterial() failed: %v", err)
	}

	privKey, err := adapter.FormatPrivateKey(km)
	if err != nil {
		t.Fatalf("FormatPrivateKey() failed: %v", err)
	}

	if len(privKey) != 64 {
		t.Errorf("Private key string length = %d, want 64", len(privKey))
	}

	// Check if private key is valid hex
	for _, char := range privKey {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			t.Errorf("Private key contains non-hex character: %c", char)
		}
	}
}

func TestEthereumAdapter_MatchesPattern(t *testing.T) {
	adapter := NewEthereumAdapter()

	tests := []struct {
		name     string
		address  string
		criteria wallet.GenerationCriteria
		want     bool
	}{
		{
			name:    "matches prefix",
			address: "abcdef1234567890abcdef1234567890abcdef12",
			criteria: wallet.GenerationCriteria{
				Prefix: "abc",
			},
			want: true,
		},
		{
			name:    "matches suffix",
			address: "1234567890abcdef1234567890abcdef12abcdef",
			criteria: wallet.GenerationCriteria{
				Suffix: "cdef",
			},
			want: true,
		},
		{
			name:    "matches prefix and suffix",
			address: "abc4567890abcdef1234567890abcdef12abcdef",
			criteria: wallet.GenerationCriteria{
				Prefix: "abc",
				Suffix: "cdef",
			},
			want: true,
		},
		{
			name:    "doesn't match prefix",
			address: "1234567890abcdef1234567890abcdef12abcdef",
			criteria: wallet.GenerationCriteria{
				Prefix: "abc",
			},
			want: false,
		},
		{
			name:    "doesn't match suffix",
			address: "abc4567890abcdef1234567890abcdef12123456",
			criteria: wallet.GenerationCriteria{
				Suffix: "cdef",
			},
			want: false,
		},
		{
			name:    "case insensitive without checksum",
			address: "ABC4567890abcdef1234567890abcdef12abcdef",
			criteria: wallet.GenerationCriteria{
				Prefix:     "abc",
				IsChecksum: false,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.MatchesPattern(tt.address, tt.criteria)
			if got != tt.want {
				t.Errorf("MatchesPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEthereumAdapter_ValidatePattern(t *testing.T) {
	adapter := NewEthereumAdapter()

	tests := []struct {
		name    string
		prefix  string
		suffix  string
		wantErr bool
	}{
		{
			name:    "valid patterns",
			prefix:  "abc",
			suffix:  "def",
			wantErr: false,
		},
		{
			name:    "empty patterns",
			prefix:  "",
			suffix:  "",
			wantErr: false,
		},
		{
			name:    "invalid hex in prefix",
			prefix:  "xyz",
			suffix:  "def",
			wantErr: true,
		},
		{
			name:    "invalid hex in suffix",
			prefix:  "abc",
			suffix:  "xyz",
			wantErr: true,
		},
		{
			name:    "too long combined",
			prefix:  "abcdef1234567890abcdef12",
			suffix:  "1234567890abcdef12abcdef",
			wantErr: true,
		},
		{
			name:    "max length allowed",
			prefix:  "abcdef1234567890abcdef",
			suffix:  "1234567890abcdef12",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.ValidatePattern(tt.prefix, tt.suffix)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePattern() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEthereumAdapter_ChainName(t *testing.T) {
	adapter := NewEthereumAdapter()

	if got := adapter.ChainName(); got != "ethereum" {
		t.Errorf("ChainName() = %v, want ethereum", got)
	}
}

func TestEthereumAdapter_AddressEncoding(t *testing.T) {
	adapter := NewEthereumAdapter()

	if got := adapter.AddressEncoding(); got != "hex" {
		t.Errorf("AddressEncoding() = %v, want hex", got)
	}
}

func TestEthereumAdapter_EIP55Checksum(t *testing.T) {
	adapter := NewEthereumAdapter()

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "all lowercase remains lowercase",
			address:  "0123456789012345678901234567890123456789",
			expected: "0123456789012345678901234567890123456789",
		},
		{
			name:     "mixed case gets checksummed",
			address:  "abcdefabcdefabcdefabcdefabcdefabcdefabcd",
			expected: adapter.applyEIP55Checksum("abcdefabcdefabcdefabcdefabcdefabcdefabcd"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.applyEIP55Checksum(tt.address)
			if got != tt.expected {
				t.Errorf("applyEIP55Checksum() = %v, want %v", got, tt.expected)
			}
		})
	}
}
