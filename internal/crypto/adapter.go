package crypto

import (
	"bloco-eth/pkg/wallet"
)

// ChainAdapter defines the interface for blockchain-specific wallet generation
type ChainAdapter interface {
	// GenerateKeyMaterial generates cryptographic key material for the chain
	GenerateKeyMaterial() (*KeyMaterial, error)

	// FormatAddress converts key material to a chain-specific address
	FormatAddress(km *KeyMaterial) (string, error)

	// FormatPrivateKey converts private key bytes to chain-specific format
	FormatPrivateKey(km *KeyMaterial) (string, error)

	// FormatPublicKey converts public key bytes to chain-specific format
	FormatPublicKey(km *KeyMaterial) (string, error)

	// MatchesPattern checks if an address matches the given prefix/suffix
	MatchesPattern(address string, criteria wallet.GenerationCriteria) bool

	// ValidatePattern validates that a prefix/suffix pattern is valid for this chain
	ValidatePattern(prefix, suffix string) error

	// ChainName returns the name of the blockchain
	ChainName() string

	// AddressEncoding returns the encoding used for addresses (hex, base58, etc.)
	AddressEncoding() string
}

// KeyMaterial holds the cryptographic key material for any blockchain
type KeyMaterial struct {
	// PrivateKey raw bytes of the private key
	PrivateKey []byte

	// PublicKey raw bytes of the public key
	PublicKey []byte

	// Metadata for chain-specific data (e.g., compressed pubkey flag, derivation path)
	Metadata map[string]interface{}
}

// NewKeyMaterial creates a new KeyMaterial instance
func NewKeyMaterial(privateKey, publicKey []byte) *KeyMaterial {
	return &KeyMaterial{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Metadata:   make(map[string]interface{}),
	}
}
