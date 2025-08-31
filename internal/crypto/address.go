package crypto

import (
	"crypto/rand"
	"encoding/hex"

	"bloco-eth/pkg/errors"
	"bloco-eth/pkg/wallet"

	"github.com/ethereum/go-ethereum/crypto"
)

// AddressGenerator handles Ethereum address generation
type AddressGenerator struct {
	poolManager *PoolManager
}

// NewAddressGenerator creates a new address generator
func NewAddressGenerator(poolManager *PoolManager) *AddressGenerator {
	return &AddressGenerator{
		poolManager: poolManager,
	}
}

// GenerateWallet generates a new wallet with private key and address
func (ag *AddressGenerator) GenerateWallet() (*wallet.Wallet, error) {
	// Get private key buffer from pool
	cryptoPool := ag.poolManager.GetCryptoPool()
	privateKey := cryptoPool.GetPrivateKeyBuffer()
	defer cryptoPool.PutPrivateKeyBuffer(privateKey)

	// Generate 32 random bytes for private key
	_, err := rand.Read(privateKey)
	if err != nil {
		return nil, errors.NewCryptoError("generate_wallet",
			"failed to generate random private key", err)
	}

	// Generate address from private key
	address, err := ag.PrivateKeyToAddress(privateKey)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_wallet", "failed to generate address from private key")
	}

	return &wallet.Wallet{
		Address:    address,
		PrivateKey: hex.EncodeToString(privateKey),
	}, nil
}

// PrivateKeyToAddress converts a private key to an Ethereum address
func (ag *AddressGenerator) PrivateKeyToAddress(privateKey []byte) (string, error) {
	if len(privateKey) != 32 {
		return "", errors.NewValidationError("private_key_to_address",
			"private key must be 32 bytes")
	}

	// Get objects from pools
	cryptoPool := ag.poolManager.GetCryptoPool()
	hasherPool := ag.poolManager.GetHasherPool()
	bufferPool := ag.poolManager.GetBufferPool()

	privateKeyInt := cryptoPool.GetBigInt()
	privateKeyECDSA := cryptoPool.GetECDSAKey()
	hasher := hasherPool.GetKeccak()
	publicKeyBytes := cryptoPool.GetPublicKeyBuffer()
	hexBuffer := bufferPool.GetHexBuffer()

	defer func() {
		// Return objects to pools
		cryptoPool.PutBigInt(privateKeyInt)
		cryptoPool.PutECDSAKey(privateKeyECDSA)
		hasherPool.PutKeccak(hasher)
		cryptoPool.PutPublicKeyBuffer(publicKeyBytes)
		bufferPool.PutHexBuffer(hexBuffer)
	}()

	// Convert private key bytes to ECDSA private key
	privateKeyInt.SetBytes(privateKey)
	privateKeyECDSA.D = privateKeyInt
	privateKeyECDSA.Curve = crypto.S256()

	// Calculate public key coordinates
	privateKeyECDSA.X, privateKeyECDSA.Y = crypto.S256().ScalarBaseMult(privateKey)

	// Get uncompressed public key bytes (without 0x04 prefix)
	publicKeyBytes = publicKeyBytes[:0] // Reset but keep capacity

	// Append X and Y coordinates (32 bytes each)
	xBytes := privateKeyECDSA.X.Bytes()
	yBytes := privateKeyECDSA.Y.Bytes()

	// Pad to 32 bytes if necessary
	for len(xBytes) < 32 {
		publicKeyBytes = append(publicKeyBytes, 0)
	}
	publicKeyBytes = append(publicKeyBytes, xBytes...)

	for len(yBytes) < 32 {
		publicKeyBytes = append(publicKeyBytes, 0)
	}
	publicKeyBytes = append(publicKeyBytes, yBytes...)

	// Calculate Keccak256 hash using pooled hasher
	hasher.Reset()
	hasher.Write(publicKeyBytes)
	hash := hasher.Sum(nil)

	// Take the last 20 bytes as the address
	address := hash[len(hash)-20:]

	// Use pre-allocated buffer for hex encoding
	hex.Encode(hexBuffer, address)
	return string(hexBuffer[:40]), nil
}

// GeneratePrivateKey generates a cryptographically secure private key
func (ag *AddressGenerator) GeneratePrivateKey() ([]byte, error) {
	cryptoPool := ag.poolManager.GetCryptoPool()
	privateKey := cryptoPool.GetPrivateKeyBuffer()
	defer cryptoPool.PutPrivateKeyBuffer(privateKey)

	_, err := rand.Read(privateKey)
	if err != nil {
		return nil, errors.NewCryptoError("generate_private_key",
			"failed to generate random bytes", err)
	}

	// Create a copy to return (since we're putting the buffer back in the pool)
	result := make([]byte, 32)
	copy(result, privateKey)
	return result, nil
}

// ValidatePrivateKey validates that a private key is valid
func (ag *AddressGenerator) ValidatePrivateKey(privateKey []byte) error {
	if len(privateKey) != 32 {
		return errors.NewValidationError("validate_private_key",
			"private key must be 32 bytes")
	}

	// Check if private key is zero (invalid)
	isZero := true
	for _, b := range privateKey {
		if b != 0 {
			isZero = false
			break
		}
	}

	if isZero {
		return errors.NewValidationError("validate_private_key",
			"private key cannot be zero")
	}

	// Additional validation could include checking if the key is within
	// the valid range for secp256k1 curve
	return nil
}

// BatchGenerateWallets generates multiple wallets efficiently
func (ag *AddressGenerator) BatchGenerateWallets(count int) ([]*wallet.Wallet, error) {
	if count <= 0 {
		return nil, errors.NewValidationError("batch_generate_wallets",
			"count must be positive")
	}

	wallets := make([]*wallet.Wallet, 0, count)

	for i := 0; i < count; i++ {
		wallet, err := ag.GenerateWallet()
		if err != nil {
			return wallets, errors.WrapError(err, errors.ErrorTypeCrypto,
				"batch_generate_wallets", "failed to generate wallet in batch")
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

// OptimizedAddressGeneration performs optimized address generation for high-throughput scenarios
func (ag *AddressGenerator) OptimizedAddressGeneration(privateKey []byte) (string, error) {
	// This is a more optimized version that reuses objects more efficiently
	// and minimizes allocations for high-performance scenarios

	if len(privateKey) != 32 {
		return "", errors.NewValidationError("optimized_address_generation",
			"private key must be 32 bytes")
	}

	// Get objects from pools
	cryptoPool := ag.poolManager.GetCryptoPool()
	hasherPool := ag.poolManager.GetHasherPool()
	bufferPool := ag.poolManager.GetBufferPool()

	privateKeyInt := cryptoPool.GetBigInt()
	hasher := hasherPool.GetKeccak()
	hexBuffer := bufferPool.GetHexBuffer()

	defer func() {
		cryptoPool.PutBigInt(privateKeyInt)
		hasherPool.PutKeccak(hasher)
		bufferPool.PutHexBuffer(hexBuffer)
	}()

	// Convert private key to big int
	privateKeyInt.SetBytes(privateKey)

	// Calculate public key coordinates directly
	x, y := crypto.S256().ScalarBaseMult(privateKey)

	// Create public key bytes directly without intermediate allocations
	hasher.Reset()

	// Write X coordinate (pad to 32 bytes)
	xBytes := x.Bytes()
	padding := 32 - len(xBytes)
	for i := 0; i < padding; i++ {
		hasher.Write([]byte{0})
	}
	hasher.Write(xBytes)

	// Write Y coordinate (pad to 32 bytes)
	yBytes := y.Bytes()
	padding = 32 - len(yBytes)
	for i := 0; i < padding; i++ {
		hasher.Write([]byte{0})
	}
	hasher.Write(yBytes)

	// Get hash and extract address
	hash := hasher.Sum(nil)
	address := hash[len(hash)-20:]

	// Encode to hex
	hex.Encode(hexBuffer, address)
	return string(hexBuffer[:40]), nil
}

// GetPoolManager returns the pool manager (for testing and optimization)
func (ag *AddressGenerator) GetPoolManager() *PoolManager {
	return ag.poolManager
}
