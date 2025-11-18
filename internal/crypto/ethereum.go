package crypto

import (
	"crypto/ecdsa"
	cryptoRand "crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"bloco-eth/pkg/wallet"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

// EthereumAdapter implements ChainAdapter for Ethereum
type EthereumAdapter struct{}

// NewEthereumAdapter creates a new Ethereum adapter
func NewEthereumAdapter() *EthereumAdapter {
	return &EthereumAdapter{}
}

// GenerateKeyMaterial generates ECDSA secp256k1 key pair for Ethereum
func (e *EthereumAdapter) GenerateKeyMaterial() (*KeyMaterial, error) {
	// Generate ECDSA key pair using secp256k1 curve
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), cryptoRand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Extract private key bytes
	privateKeyBytes := privateKey.D.Bytes()
	// Pad to 32 bytes if needed
	if len(privateKeyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(privateKeyBytes):], privateKeyBytes)
		privateKeyBytes = padded
	}

	// Extract public key bytes (uncompressed)
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	km := NewKeyMaterial(privateKeyBytes, publicKeyBytes)
	km.Metadata["curve"] = "secp256k1"
	
	return km, nil
}

// FormatAddress converts key material to Ethereum address (40 hex chars, no 0x prefix)
func (e *EthereumAdapter) FormatAddress(km *KeyMaterial) (string, error) {
	if km == nil || len(km.PublicKey) == 0 {
		return "", fmt.Errorf("invalid key material")
	}

	// For Ethereum, we hash the public key and take last 20 bytes
	// The public key should be 65 bytes (uncompressed format with 0x04 prefix)
	// We skip the first byte (0x04) and hash the remaining 64 bytes
	pubKeyBytes := km.PublicKey
	if len(pubKeyBytes) == 65 {
		pubKeyBytes = pubKeyBytes[1:] // Skip the 0x04 prefix
	}

	// Keccak256 hash
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKeyBytes)
	addressBytes := hash.Sum(nil)[12:] // Take last 20 bytes

	return hex.EncodeToString(addressBytes), nil
}

// FormatPrivateKey converts private key to hex string (64 chars, no 0x prefix)
func (e *EthereumAdapter) FormatPrivateKey(km *KeyMaterial) (string, error) {
	if km == nil || len(km.PrivateKey) == 0 {
		return "", fmt.Errorf("invalid key material")
	}
	return hex.EncodeToString(km.PrivateKey), nil
}

// FormatPublicKey converts public key to hex string
func (e *EthereumAdapter) FormatPublicKey(km *KeyMaterial) (string, error) {
	if km == nil || len(km.PublicKey) == 0 {
		return "", fmt.Errorf("invalid key material")
	}
	// Return without the 0x04 prefix
	pubKey := km.PublicKey
	if len(pubKey) == 65 && pubKey[0] == 0x04 {
		pubKey = pubKey[1:]
	}
	return hex.EncodeToString(pubKey), nil
}

// MatchesPattern checks if address matches prefix/suffix with optional EIP-55 checksum
func (e *EthereumAdapter) MatchesPattern(address string, criteria wallet.GenerationCriteria) bool {
	// Remove 0x prefix if present
	address = strings.TrimPrefix(address, "0x")
	
	if criteria.IsChecksum {
		// Apply EIP-55 checksum
		checksumAddr := e.applyEIP55Checksum(address)
		
		// Check prefix
		if criteria.Prefix != "" && !strings.HasPrefix(checksumAddr, criteria.Prefix) {
			return false
		}
		
		// Check suffix
		if criteria.Suffix != "" && !strings.HasSuffix(checksumAddr, criteria.Suffix) {
			return false
		}
	} else {
		// Case-insensitive matching
		addressLower := strings.ToLower(address)
		prefixLower := strings.ToLower(criteria.Prefix)
		suffixLower := strings.ToLower(criteria.Suffix)
		
		// Check prefix
		if prefixLower != "" && !strings.HasPrefix(addressLower, prefixLower) {
			return false
		}
		
		// Check suffix
		if suffixLower != "" && !strings.HasSuffix(addressLower, suffixLower) {
			return false
		}
	}
	
	return true
}

// ValidatePattern validates that prefix/suffix contain only hex characters
func (e *EthereumAdapter) ValidatePattern(prefix, suffix string) error {
	// Check hex validity
	if !isValidHex(prefix) {
		return fmt.Errorf("prefix contains invalid hex characters: %s", prefix)
	}
	if !isValidHex(suffix) {
		return fmt.Errorf("suffix contains invalid hex characters: %s", suffix)
	}
	
	// Check combined length
	totalLen := len(prefix) + len(suffix)
	if totalLen > 40 {
		return fmt.Errorf("combined prefix+suffix length (%d) exceeds Ethereum address length (40)", totalLen)
	}
	
	return nil
}

// ChainName returns "ethereum"
func (e *EthereumAdapter) ChainName() string {
	return "ethereum"
}

// AddressEncoding returns "hex"
func (e *EthereumAdapter) AddressEncoding() string {
	return "hex"
}

// applyEIP55Checksum applies EIP-55 checksum encoding to an address
func (e *EthereumAdapter) applyEIP55Checksum(address string) string {
	// Remove 0x prefix if present
	address = strings.TrimPrefix(address, "0x")
	address = strings.ToLower(address)
	
	// Hash the lowercase address
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(address))
	hashBytes := hash.Sum(nil)
	
	// Build checksum address
	result := make([]byte, len(address))
	for i := 0; i < len(address); i++ {
		char := address[i]
		// If it's a letter (a-f) and the hash byte is >= 8, capitalize it
		if char >= 'a' && char <= 'f' {
			// Get the corresponding nibble from hash
			hashByte := hashBytes[i/2]
			var nibble byte
			if i%2 == 0 {
				nibble = hashByte >> 4
			} else {
				nibble = hashByte & 0x0f
			}
			
			if nibble >= 8 {
				result[i] = char - 32 // Convert to uppercase
			} else {
				result[i] = char
			}
		} else {
			result[i] = char
		}
	}
	
	return string(result)
}

// isValidHex checks if a string contains only valid hex characters
func isValidHex(s string) bool {
	if len(s) == 0 {
		return true
	}
	for _, char := range s {
		if (char < '0' || char > '9') &&
			(char < 'a' || char > 'f') &&
			(char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}
