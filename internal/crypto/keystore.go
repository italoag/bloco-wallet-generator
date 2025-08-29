package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// KeyStoreError represents detailed error information for keystore operations
type KeyStoreError struct {
	Operation   string // The operation that failed (e.g., "encrypt", "save_file", "validate")
	Component   string // The component involved (e.g., "keystore", "password", "directory")
	Address     string // The address being processed (if applicable)
	Path        string // The file path involved (if applicable)
	Underlying  error  // The underlying error
	Recoverable bool   // Whether the error might be recoverable with retry
	UserMessage string // User-friendly error message
}

func (e *KeyStoreError) Error() string {
	if e.UserMessage != "" {
		return e.UserMessage
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("keystore %s failed", e.Operation))

	if e.Component != "" {
		msg.WriteString(fmt.Sprintf(" for %s", e.Component))
	}

	if e.Address != "" {
		msg.WriteString(fmt.Sprintf(" (address: %s)", e.Address))
	}

	if e.Path != "" {
		msg.WriteString(fmt.Sprintf(" (path: %s)", e.Path))
	}

	if e.Underlying != nil {
		msg.WriteString(fmt.Sprintf(": %v", e.Underlying))
	}

	return msg.String()
}

func (e *KeyStoreError) Unwrap() error {
	return e.Underlying
}

// IsRecoverable returns true if the error might be recoverable with retry
func (e *KeyStoreError) IsRecoverable() bool {
	return e.Recoverable
}

// NewKeyStoreError creates a new KeyStoreError with the given parameters
func NewKeyStoreError(operation, component string, err error) *KeyStoreError {
	return &KeyStoreError{
		Operation:  operation,
		Component:  component,
		Underlying: err,
	}
}

// NewKeyStoreErrorWithAddress creates a new KeyStoreError with address information
func NewKeyStoreErrorWithAddress(operation, component, address string, err error) *KeyStoreError {
	return &KeyStoreError{
		Operation:  operation,
		Component:  component,
		Address:    address,
		Underlying: err,
	}
}

// NewKeyStoreErrorWithPath creates a new KeyStoreError with path information
func NewKeyStoreErrorWithPath(operation, component, path string, err error) *KeyStoreError {
	return &KeyStoreError{
		Operation:  operation,
		Component:  component,
		Path:       path,
		Underlying: err,
	}
}

// NewRecoverableKeyStoreError creates a new recoverable KeyStoreError
func NewRecoverableKeyStoreError(operation, component string, err error, userMessage string) *KeyStoreError {
	return &KeyStoreError{
		Operation:   operation,
		Component:   component,
		Underlying:  err,
		Recoverable: true,
		UserMessage: userMessage,
	}
}

// KeyStoreV3 represents the Ethereum KeyStore V3 format
type KeyStoreV3 struct {
	Address string         `json:"address"`
	Crypto  KeyStoreCrypto `json:"crypto"`
	ID      string         `json:"id"`
	Version int            `json:"version"`
}

// KeyStoreCrypto contains the cryptographic parameters for the keystore
type KeyStoreCrypto struct {
	Cipher       string       `json:"cipher"`
	CipherText   string       `json:"ciphertext"`
	CipherParams CipherParams `json:"cipherparams"`
	KDF          string       `json:"kdf"`
	KDFParams    interface{}  `json:"kdfparams"`
	MAC          string       `json:"mac"`
}

// CipherParams contains parameters for the cipher algorithm
type CipherParams struct {
	IV string `json:"iv"`
}

// ScryptParams contains parameters for the scrypt key derivation function
type ScryptParams struct {
	DKLen int    `json:"dklen"`
	N     int    `json:"n"`
	P     int    `json:"p"`
	R     int    `json:"r"`
	Salt  string `json:"salt"`
}

// PBKDF2Params contains parameters for the PBKDF2 key derivation function
type PBKDF2Params struct {
	DKLen int    `json:"dklen"`
	C     int    `json:"c"`
	PRF   string `json:"prf"`
	Salt  string `json:"salt"`
}

// NewKeyStoreV3 creates a new KeyStore V3 structure with default values
func NewKeyStoreV3(address string) *KeyStoreV3 {
	// Remove 0x prefix if present and convert to lowercase
	cleanAddress := strings.ToLower(strings.TrimPrefix(address, "0x"))

	return &KeyStoreV3{
		Address: cleanAddress,
		Crypto: KeyStoreCrypto{
			Cipher: "aes-128-ctr",
			KDF:    "scrypt",
		},
		ID:      uuid.New().String(),
		Version: 3,
	}
}

// SetScryptParams sets the scrypt parameters for the keystore
func (ks *KeyStoreV3) SetScryptParams(n, r, p, dklen int, salt []byte) {
	ks.Crypto.KDF = "scrypt"
	ks.Crypto.KDFParams = ScryptParams{
		DKLen: dklen,
		N:     n,
		P:     p,
		R:     r,
		Salt:  hex.EncodeToString(salt),
	}
}

// SetPBKDF2Params sets the PBKDF2 parameters for the keystore
func (ks *KeyStoreV3) SetPBKDF2Params(c, dklen int, prf string, salt []byte) {
	ks.Crypto.KDF = "pbkdf2"
	ks.Crypto.KDFParams = PBKDF2Params{
		DKLen: dklen,
		C:     c,
		PRF:   prf,
		Salt:  hex.EncodeToString(salt),
	}
}

// SetCipherParams sets the cipher parameters (IV and ciphertext)
func (ks *KeyStoreV3) SetCipherParams(iv, ciphertext []byte) {
	ks.Crypto.CipherParams = CipherParams{
		IV: hex.EncodeToString(iv),
	}
	ks.Crypto.CipherText = hex.EncodeToString(ciphertext)
}

// SetMAC sets the MAC for integrity verification
func (ks *KeyStoreV3) SetMAC(mac []byte) {
	ks.Crypto.MAC = hex.EncodeToString(mac)
}

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// Validate validates the KeyStore V3 structure
func (ks *KeyStoreV3) Validate() error {
	if ks.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if len(ks.Address) != 40 {
		return fmt.Errorf("address must be 40 characters long (without 0x prefix)")
	}

	if ks.Version != 3 {
		return fmt.Errorf("version must be 3")
	}

	if ks.ID == "" {
		return fmt.Errorf("ID cannot be empty")
	}

	if ks.Crypto.Cipher != "aes-128-ctr" {
		return fmt.Errorf("unsupported cipher: %s", ks.Crypto.Cipher)
	}

	if ks.Crypto.KDF != "scrypt" && ks.Crypto.KDF != "pbkdf2" {
		return fmt.Errorf("unsupported KDF: %s", ks.Crypto.KDF)
	}

	if ks.Crypto.CipherText == "" {
		return fmt.Errorf("ciphertext cannot be empty")
	}

	if ks.Crypto.MAC == "" {
		return fmt.Errorf("MAC cannot be empty")
	}

	if ks.Crypto.CipherParams.IV == "" {
		return fmt.Errorf("IV cannot be empty")
	}

	if ks.Crypto.KDFParams == nil {
		return fmt.Errorf("KDF parameters cannot be nil")
	}

	return nil
}

// ToJSON serializes the KeyStore V3 to JSON
func (ks *KeyStoreV3) ToJSON() ([]byte, error) {
	if err := ks.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return json.MarshalIndent(ks, "", "  ")
}

// FromJSON deserializes JSON data into a KeyStore V3 structure
func FromJSON(data []byte) (*KeyStoreV3, error) {
	var ks KeyStoreV3
	if err := json.Unmarshal(data, &ks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := ks.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &ks, nil
}

// GetScryptParams returns the scrypt parameters if KDF is scrypt
func (ks *KeyStoreV3) GetScryptParams() (*ScryptParams, error) {
	if ks.Crypto.KDF != "scrypt" {
		return nil, fmt.Errorf("KDF is not scrypt")
	}

	// Convert interface{} to ScryptParams
	paramsMap, ok := ks.Crypto.KDFParams.(map[string]interface{})
	if !ok {
		// Try direct type assertion
		if params, ok := ks.Crypto.KDFParams.(ScryptParams); ok {
			return &params, nil
		}
		return nil, fmt.Errorf("invalid scrypt parameters format")
	}

	params := &ScryptParams{}
	if dklen, ok := paramsMap["dklen"].(float64); ok {
		params.DKLen = int(dklen)
	}
	if n, ok := paramsMap["n"].(float64); ok {
		params.N = int(n)
	}
	if p, ok := paramsMap["p"].(float64); ok {
		params.P = int(p)
	}
	if r, ok := paramsMap["r"].(float64); ok {
		params.R = int(r)
	}
	if salt, ok := paramsMap["salt"].(string); ok {
		params.Salt = salt
	}

	return params, nil
}

// GetPBKDF2Params returns the PBKDF2 parameters if KDF is pbkdf2
func (ks *KeyStoreV3) GetPBKDF2Params() (*PBKDF2Params, error) {
	if ks.Crypto.KDF != "pbkdf2" {
		return nil, fmt.Errorf("KDF is not pbkdf2")
	}

	// Convert interface{} to PBKDF2Params
	paramsMap, ok := ks.Crypto.KDFParams.(map[string]interface{})
	if !ok {
		// Try direct type assertion
		if params, ok := ks.Crypto.KDFParams.(PBKDF2Params); ok {
			return &params, nil
		}
		return nil, fmt.Errorf("invalid PBKDF2 parameters format")
	}

	params := &PBKDF2Params{}
	if dklen, ok := paramsMap["dklen"].(float64); ok {
		params.DKLen = int(dklen)
	}
	if c, ok := paramsMap["c"].(float64); ok {
		params.C = int(c)
	}
	if prf, ok := paramsMap["prf"].(string); ok {
		params.PRF = prf
	}
	if salt, ok := paramsMap["salt"].(string); ok {
		params.Salt = salt
	}

	return params, nil
}

// DeriveKeyScrypt derives a key using the scrypt key derivation function
func DeriveKeyScrypt(password []byte, salt []byte, n, r, p, dkLen int) ([]byte, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("password cannot be empty")
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}
	if dkLen <= 0 {
		return nil, fmt.Errorf("derived key length must be positive")
	}

	key, err := scrypt.Key(password, salt, n, r, p, dkLen)
	if err != nil {
		return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
	}

	return key, nil
}

// DeriveKeyPBKDF2 derives a key using the PBKDF2 key derivation function with HMAC-SHA256
func DeriveKeyPBKDF2(password []byte, salt []byte, iterations, dkLen int) ([]byte, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("password cannot be empty")
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}
	if iterations <= 0 {
		return nil, fmt.Errorf("iterations must be positive")
	}
	if dkLen <= 0 {
		return nil, fmt.Errorf("derived key length must be positive")
	}

	key := pbkdf2.Key(password, salt, iterations, dkLen, sha256.New)
	return key, nil
}

// EncryptAES128CTR encrypts data using AES-128-CTR mode
func EncryptAES128CTR(plaintext []byte, key []byte, iv []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}
	if len(key) != 16 {
		return nil, fmt.Errorf("key must be 16 bytes for AES-128")
	}
	if len(iv) != 16 {
		return nil, fmt.Errorf("IV must be 16 bytes for AES-128-CTR")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)

	return ciphertext, nil
}

// DecryptAES128CTR decrypts data using AES-128-CTR mode
func DecryptAES128CTR(ciphertext []byte, key []byte, iv []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext cannot be empty")
	}
	if len(key) != 16 {
		return nil, fmt.Errorf("key must be 16 bytes for AES-128")
	}
	if len(iv) != 16 {
		return nil, fmt.Errorf("IV must be 16 bytes for AES-128-CTR")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	plaintext := make([]byte, len(ciphertext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// GenerateMAC generates a MAC using HMAC-SHA256 for integrity verification
func GenerateMAC(derivedKey []byte, ciphertext []byte) ([]byte, error) {
	if len(derivedKey) < 32 {
		return nil, fmt.Errorf("derived key must be at least 32 bytes")
	}
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext cannot be empty")
	}

	// Use the second half of the derived key for MAC generation (Ethereum KeyStore V3 standard)
	macKey := derivedKey[16:32]

	h := hmac.New(sha256.New, macKey)
	h.Write(derivedKey[16:32]) // MAC key
	h.Write(ciphertext)        // Encrypted private key

	return h.Sum(nil), nil
}

// VerifyMAC verifies a MAC using HMAC-SHA256
func VerifyMAC(derivedKey []byte, ciphertext []byte, expectedMAC []byte) (bool, error) {
	if len(expectedMAC) == 0 {
		return false, fmt.Errorf("expected MAC cannot be empty")
	}

	computedMAC, err := GenerateMAC(derivedKey, ciphertext)
	if err != nil {
		return false, fmt.Errorf("failed to compute MAC: %w", err)
	}

	return hmac.Equal(computedMAC, expectedMAC), nil
}

// EncryptPrivateKey encrypts a private key using the specified KDF and AES-128-CTR
func EncryptPrivateKey(privateKeyHex string, password string, kdf string) (*KeyStoreV3, error) {
	if privateKeyHex == "" {
		return nil, fmt.Errorf("private key cannot be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Validate private key format
	if len(privateKeyHex) != 64 {
		return nil, fmt.Errorf("private key must be 64 hex characters")
	}

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key hex: %w", err)
	}

	// Generate random salt and IV
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	iv, err := GenerateRandomBytes(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Derive key using specified KDF
	var derivedKey []byte
	switch kdf {
	case "scrypt":
		derivedKey, err = DeriveKeyScrypt([]byte(password), salt, 262144, 8, 1, 32)
		if err != nil {
			return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
		}
	case "pbkdf2":
		derivedKey, err = DeriveKeyPBKDF2([]byte(password), salt, 262144, 32)
		if err != nil {
			return nil, fmt.Errorf("PBKDF2 key derivation failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported KDF: %s", kdf)
	}

	// Use first 16 bytes of derived key for AES encryption
	encryptionKey := derivedKey[:16]

	// Encrypt private key
	ciphertext, err := EncryptAES128CTR(privateKeyBytes, encryptionKey, iv)
	if err != nil {
		return nil, fmt.Errorf("AES encryption failed: %w", err)
	}

	// Generate MAC for integrity
	mac, err := GenerateMAC(derivedKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("MAC generation failed: %w", err)
	}

	// Derive Ethereum address from private key (placeholder - would need actual implementation)
	// For now, we'll use a placeholder address
	address := "0000000000000000000000000000000000000000"

	// Create KeyStore V3 structure
	ks := NewKeyStoreV3(address)

	// Set KDF parameters
	switch kdf {
	case "scrypt":
		ks.SetScryptParams(262144, 8, 1, 32, salt)
	case "pbkdf2":
		ks.SetPBKDF2Params(262144, 32, "hmac-sha256", salt)
	}

	// Set cipher parameters
	ks.SetCipherParams(iv, ciphertext)
	ks.SetMAC(mac)

	return ks, nil
}

// KeyStoreConfig holds configuration for keystore generation
type KeyStoreConfig struct {
	OutputDirectory string
	Enabled         bool
	Cipher          string // "aes-128-ctr"
	KDF             string // "scrypt" or "pbkdf2"
	MaxRetries      int    // Maximum number of retry attempts for recoverable errors
	RetryDelay      int    // Delay between retries in milliseconds
}

// FileOperationError represents errors that occur during file operations
type FileOperationError struct {
	Operation string
	Path      string
	Err       error
}

func (e *FileOperationError) Error() string {
	return fmt.Sprintf("file operation '%s' failed for path '%s': %v", e.Operation, e.Path, e.Err)
}

func (e *FileOperationError) Unwrap() error {
	return e.Err
}

// ProgressLogger defines the interface for progress logging
type ProgressLogger interface {
	LogInfo(message string)
	LogWarning(message string)
	LogError(message string)
	LogDebug(message string)
}

// DefaultProgressLogger provides a simple console-based progress logger
type DefaultProgressLogger struct {
	VerboseMode bool
}

func (l *DefaultProgressLogger) LogInfo(message string) {
	if l.VerboseMode {
		fmt.Printf("ℹ️  %s\n", message)
	}
}

func (l *DefaultProgressLogger) LogWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

func (l *DefaultProgressLogger) LogError(message string) {
	fmt.Printf("❌ %s\n", message)
}

func (l *DefaultProgressLogger) LogDebug(message string) {
	if l.VerboseMode {
		fmt.Printf("🔍 %s\n", message)
	}
}

// KeyStoreService handles keystore generation and file operations
type KeyStoreService struct {
	config      KeyStoreConfig
	passwordGen *PasswordGenerator
	logger      ProgressLogger
}

// NewKeyStoreService creates a new keystore service with the given configuration
func NewKeyStoreService(config KeyStoreConfig) *KeyStoreService {
	// Set defaults if not specified
	if config.Cipher == "" {
		config.Cipher = "aes-128-ctr"
	}
	if config.KDF == "" {
		config.KDF = "scrypt"
	}
	if config.OutputDirectory == "" {
		config.OutputDirectory = "./keystores"
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 // 100ms default delay
	}

	return &KeyStoreService{
		config:      config,
		passwordGen: NewPasswordGenerator(),
		logger:      &DefaultProgressLogger{VerboseMode: false},
	}
}

// NewKeyStoreServiceWithLogger creates a new keystore service with a custom logger
func NewKeyStoreServiceWithLogger(config KeyStoreConfig, logger ProgressLogger) *KeyStoreService {
	service := NewKeyStoreService(config)
	service.logger = logger
	return service
}

// SetLogger sets a custom progress logger
func (ks *KeyStoreService) SetLogger(logger ProgressLogger) {
	ks.logger = logger
}

// SetVerboseMode enables or disables verbose logging for the default logger
func (ks *KeyStoreService) SetVerboseMode(verbose bool) {
	if defaultLogger, ok := ks.logger.(*DefaultProgressLogger); ok {
		defaultLogger.VerboseMode = verbose
	}
}

// SaveKeyStoreFilesWithRetry saves keystore files with automatic retry for recoverable errors
func (ks *KeyStoreService) SaveKeyStoreFilesWithRetry(privateKeyHex, address string) error {
	if !ks.config.Enabled {
		ks.logger.LogDebug("Keystore generation is disabled, skipping file creation")
		return nil
	}

	var lastErr error
	for attempt := 1; attempt <= ks.config.MaxRetries; attempt++ {
		ks.logger.LogDebug(fmt.Sprintf("Keystore save attempt %d/%d for address %s", attempt, ks.config.MaxRetries, address))

		err := ks.SaveKeyStoreFiles(privateKeyHex, address)
		if err == nil {
			if attempt > 1 {
				ks.logger.LogInfo(fmt.Sprintf("Keystore files saved successfully for address %s after %d attempts", address, attempt))
			}
			return nil
		}

		lastErr = err

		// Check if error is recoverable
		if ksErr, ok := err.(*KeyStoreError); ok && ksErr.IsRecoverable() {
			if attempt < ks.config.MaxRetries {
				ks.logger.LogWarning(fmt.Sprintf("Recoverable error on attempt %d for address %s: %v. Retrying in %dms...",
					attempt, address, err, ks.config.RetryDelay))
				time.Sleep(time.Duration(ks.config.RetryDelay) * time.Millisecond)
				continue
			} else {
				ks.logger.LogError(fmt.Sprintf("Max retries (%d) exceeded for address %s. Last error: %v",
					ks.config.MaxRetries, address, err))
			}
		} else {
			// Non-recoverable error, don't retry
			ks.logger.LogError(fmt.Sprintf("Non-recoverable error for address %s: %v", address, err))
			break
		}
	}

	return lastErr
}

// GenerateKeyStore creates a keystore for the given private key and address
func (ks *KeyStoreService) GenerateKeyStore(privateKeyHex, address string) (*KeyStoreV3, string, error) {
	if !ks.config.Enabled {
		return nil, "", NewKeyStoreError("generate", "service", fmt.Errorf("keystore generation is disabled"))
	}

	if privateKeyHex == "" {
		return nil, "", NewKeyStoreErrorWithAddress("generate", "private_key", address, fmt.Errorf("private key cannot be empty"))
	}

	if address == "" {
		return nil, "", NewKeyStoreError("generate", "address", fmt.Errorf("address cannot be empty"))
	}

	// Generate secure password
	password, err := ks.passwordGen.GenerateSecurePassword()
	if err != nil {
		return nil, "", NewRecoverableKeyStoreError("generate", "password", err,
			"Failed to generate secure password. This might be due to insufficient system entropy. Please try again.")
	}

	// Encrypt private key using the configured KDF
	keystore, err := EncryptPrivateKey(privateKeyHex, password, ks.config.KDF)
	if err != nil {
		return nil, "", NewKeyStoreErrorWithAddress("encrypt", "private_key", address, err)
	}

	// Update the keystore with the correct address
	keystore.Address = strings.ToLower(strings.TrimPrefix(address, "0x"))

	return keystore, password, nil
}

// SaveKeyStoreFiles saves keystore files for a given private key and address (convenience method)
func (ks *KeyStoreService) SaveKeyStoreFiles(privateKeyHex, address string) error {
	if !ks.config.Enabled {
		ks.logger.LogDebug("Keystore generation is disabled, skipping file creation")
		return nil // Silently skip if disabled
	}

	ks.logger.LogDebug(fmt.Sprintf("Starting keystore generation for address %s", address))

	// Generate keystore and password
	keystore, password, err := ks.GenerateKeyStore(privateKeyHex, address)
	if err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to generate keystore for address %s: %v", address, err))
		return fmt.Errorf("failed to generate keystore: %w", err)
	}

	ks.logger.LogDebug(fmt.Sprintf("Keystore generated successfully, saving files for address %s", address))

	// Save the files
	if err := ks.SaveKeyStoreFilesToDisk(address, keystore, password); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to save keystore files for address %s: %v", address, err))
		return err
	}

	ks.logger.LogInfo(fmt.Sprintf("Keystore files saved successfully for address %s", address))
	return nil
}

// SaveKeyStoreFilesToDisk saves both keystore and password files atomically with enhanced error handling
func (ks *KeyStoreService) SaveKeyStoreFilesToDisk(address string, keystore *KeyStoreV3, password string) error {
	if !ks.config.Enabled {
		return NewKeyStoreError("save", "service", fmt.Errorf("keystore generation is disabled"))
	}

	if keystore == nil {
		return NewKeyStoreErrorWithAddress("save", "keystore", address, fmt.Errorf("keystore cannot be nil"))
	}

	if password == "" {
		return NewKeyStoreErrorWithAddress("save", "password", address, fmt.Errorf("password cannot be empty"))
	}

	// Clean address for filename
	cleanAddress := strings.ToLower(strings.TrimPrefix(address, "0x"))
	if len(cleanAddress) != 40 {
		return NewKeyStoreErrorWithAddress("validate", "address", address,
			fmt.Errorf("invalid address length: expected 40 characters, got %d", len(cleanAddress)))
	}

	// Create output directory if it doesn't exist
	ks.logger.LogDebug(fmt.Sprintf("Ensuring output directory exists: %s", ks.config.OutputDirectory))
	if err := ks.ensureOutputDirectory(); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to create directory %s: %v", ks.config.OutputDirectory, err))
		return NewRecoverableKeyStoreError("save", "directory", err,
			fmt.Sprintf("Failed to create keystore directory '%s'. Please check permissions and try again.", ks.config.OutputDirectory))
	}

	// Check directory permissions
	ks.logger.LogDebug(fmt.Sprintf("Checking directory permissions: %s", ks.config.OutputDirectory))
	if err := ks.CheckDirectoryPermissions(); err != nil {
		ks.logger.LogError(fmt.Sprintf("Directory permission check failed for %s: %v", ks.config.OutputDirectory, err))
		return NewRecoverableKeyStoreError("save", "directory", err,
			fmt.Sprintf("Directory '%s' is not writable. Please check permissions and try again.", ks.config.OutputDirectory))
	}

	// Get file paths using helper methods
	keystorePath, err := ks.GetKeystoreFilePath(address)
	if err != nil {
		return NewKeyStoreErrorWithAddress("save", "path", address, err)
	}

	passwordPath, err := ks.GetPasswordFilePath(address)
	if err != nil {
		return NewKeyStoreErrorWithAddress("save", "path", address, err)
	}

	// Check if files already exist and warn (but don't fail)
	if exists, err := ks.FileExists(keystorePath); err != nil {
		return NewRecoverableKeyStoreError("save", "file_check", err,
			"Failed to check if keystore file already exists. Please try again.")
	} else if exists {
		// File exists, we'll overwrite it atomically
	}

	if exists, err := ks.FileExists(passwordPath); err != nil {
		return NewRecoverableKeyStoreError("save", "file_check", err,
			"Failed to check if password file already exists. Please try again.")
	} else if exists {
		// File exists, we'll overwrite it atomically
	}

	// Serialize keystore to JSON
	keystoreJSON, err := keystore.ToJSON()
	if err != nil {
		return NewKeyStoreErrorWithAddress("serialize", "keystore", address, err)
	}

	// Write keystore file atomically with secure permissions (600)
	ks.logger.LogDebug(fmt.Sprintf("Writing keystore file: %s", keystorePath))
	if err := ks.writeFileAtomic(keystorePath, keystoreJSON, 0600); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to write keystore file %s: %v", keystorePath, err))
		return NewRecoverableKeyStoreError("save", "keystore_file", err,
			fmt.Sprintf("Failed to save keystore file to '%s'. Please check disk space and permissions.", keystorePath))
	}
	ks.logger.LogDebug(fmt.Sprintf("Keystore file written successfully: %s", keystorePath))

	// Write password file atomically with secure permissions (600)
	ks.logger.LogDebug(fmt.Sprintf("Writing password file: %s", passwordPath))
	if err := ks.writeFileAtomic(passwordPath, []byte(password), 0600); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to write password file %s: %v", passwordPath, err))
		// If password file fails, try to clean up keystore file
		ks.logger.LogDebug(fmt.Sprintf("Attempting to clean up keystore file: %s", keystorePath))
		if removeErr := os.Remove(keystorePath); removeErr != nil {
			ks.logger.LogError(fmt.Sprintf("Failed to clean up keystore file %s: %v", keystorePath, removeErr))
			// Return compound error with cleanup failure
			return NewKeyStoreErrorWithPath("save", "password_file", passwordPath,
				fmt.Errorf("failed to write password file and cleanup failed: write error: %w, cleanup error: %v", err, removeErr))
		}
		ks.logger.LogDebug(fmt.Sprintf("Keystore file cleaned up successfully: %s", keystorePath))
		return NewRecoverableKeyStoreError("save", "password_file", err,
			fmt.Sprintf("Failed to save password file to '%s'. Please check disk space and permissions.", passwordPath))
	}
	ks.logger.LogDebug(fmt.Sprintf("Password file written successfully: %s", passwordPath))

	// Verify both files were created with correct permissions
	if err := ks.ValidateFilePermissions(keystorePath, 0600); err != nil {
		return NewKeyStoreErrorWithPath("validate", "keystore_permissions", keystorePath, err)
	}

	if err := ks.ValidateFilePermissions(passwordPath, 0600); err != nil {
		return NewKeyStoreErrorWithPath("validate", "password_permissions", passwordPath, err)
	}

	return nil
}

// ensureOutputDirectory creates the output directory if it doesn't exist with proper error handling
func (ks *KeyStoreService) ensureOutputDirectory() error {
	// Check if path is empty
	if ks.config.OutputDirectory == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	// Clean the path to handle any path traversal issues
	cleanPath := filepath.Clean(ks.config.OutputDirectory)

	// Check if directory already exists
	info, err := os.Stat(cleanPath)
	if err == nil {
		// Path exists, check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path %s exists but is not a directory", cleanPath)
		}
		// Directory exists and is valid
		return nil
	}

	// Check if error is something other than "not exists"
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check directory %s: %w", cleanPath, err)
	}

	// Directory doesn't exist, create it with all parent directories
	if err := os.MkdirAll(cleanPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", cleanPath, err)
	}

	// Verify the directory was created successfully
	if info, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("failed to verify created directory %s: %w", cleanPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("created path %s is not a directory", cleanPath)
	}

	return nil
}

// writeFileAtomic writes data to a file atomically using a temporary file with enhanced error handling
func (ks *KeyStoreService) writeFileAtomic(filename string, data []byte, perm os.FileMode) error {
	// Validate inputs
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	if perm == 0 {
		return fmt.Errorf("file permissions cannot be zero")
	}

	// Clean the filename path
	cleanFilename := filepath.Clean(filename)
	dir := filepath.Dir(cleanFilename)

	// Ensure the directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Check if target file already exists and handle accordingly
	if _, err := os.Stat(cleanFilename); err == nil {
		// File exists - we'll overwrite it atomically
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check existing file %s: %w", cleanFilename, err)
	}

	// Create temporary file in the same directory with a secure pattern
	tmpFile, err := os.CreateTemp(dir, ".keystore-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	var writeErr error

	// Ensure cleanup happens regardless of success or failure
	defer func() {
		// Close file if still open
		if tmpFile != nil {
			tmpFile.Close()
		}
		// Remove temp file if write failed
		if writeErr != nil {
			os.Remove(tmpPath)
		}
	}()

	// Set permissions on temp file before writing sensitive data
	if err := tmpFile.Chmod(perm); err != nil {
		writeErr = fmt.Errorf("failed to set permissions on temporary file: %w", err)
		return writeErr
	}

	// Write data to temporary file
	bytesWritten, err := tmpFile.Write(data)
	if err != nil {
		writeErr = fmt.Errorf("failed to write to temporary file: %w", err)
		return writeErr
	}

	// Verify all data was written
	if bytesWritten != len(data) {
		writeErr = fmt.Errorf("incomplete write: wrote %d bytes, expected %d", bytesWritten, len(data))
		return writeErr
	}

	// Sync to ensure data is written to disk before moving
	if err := tmpFile.Sync(); err != nil {
		writeErr = fmt.Errorf("failed to sync temporary file to disk: %w", err)
		return writeErr
	}

	// Close temporary file before rename (required on Windows)
	if err := tmpFile.Close(); err != nil {
		writeErr = fmt.Errorf("failed to close temporary file: %w", err)
		return writeErr
	}
	tmpFile = nil // Mark as closed

	// Verify file permissions are correct
	if info, err := os.Stat(tmpPath); err != nil {
		writeErr = fmt.Errorf("failed to verify temporary file: %w", err)
		return writeErr
	} else if info.Mode().Perm() != perm {
		writeErr = fmt.Errorf("temporary file permissions incorrect: got %o, expected %o", info.Mode().Perm(), perm)
		return writeErr
	}

	// Atomically move temporary file to final location
	if err := os.Rename(tmpPath, cleanFilename); err != nil {
		writeErr = fmt.Errorf("failed to move temporary file to final location %s: %w", cleanFilename, err)
		return writeErr
	}

	// Verify final file exists and has correct permissions
	if info, err := os.Stat(cleanFilename); err != nil {
		return fmt.Errorf("failed to verify final file %s: %w", cleanFilename, err)
	} else if info.Mode().Perm() != perm {
		return fmt.Errorf("final file permissions incorrect: got %o, expected %o", info.Mode().Perm(), perm)
	}

	return nil
}

// GetConfig returns the current keystore service configuration
func (ks *KeyStoreService) GetConfig() KeyStoreConfig {
	return ks.config
}

// SetOutputDirectory updates the output directory for keystore files
func (ks *KeyStoreService) SetOutputDirectory(dir string) {
	ks.config.OutputDirectory = dir
}

// SetEnabled enables or disables keystore generation
func (ks *KeyStoreService) SetEnabled(enabled bool) {
	ks.config.Enabled = enabled
}

// SetKDF sets the key derivation function (scrypt or pbkdf2)
func (ks *KeyStoreService) SetKDF(kdf string) error {
	if kdf != "scrypt" && kdf != "pbkdf2" {
		return fmt.Errorf("unsupported KDF: %s (supported: scrypt, pbkdf2)", kdf)
	}
	ks.config.KDF = kdf
	return nil
}

// CheckDirectoryPermissions verifies that the output directory has appropriate permissions
func (ks *KeyStoreService) CheckDirectoryPermissions() error {
	if !ks.config.Enabled {
		return nil // Skip check if keystore is disabled
	}

	info, err := os.Stat(ks.config.OutputDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileOperationError{
				Operation: "check_permissions",
				Path:      ks.config.OutputDirectory,
				Err:       fmt.Errorf("directory does not exist"),
			}
		}
		return &FileOperationError{
			Operation: "check_permissions",
			Path:      ks.config.OutputDirectory,
			Err:       err,
		}
	}

	if !info.IsDir() {
		return &FileOperationError{
			Operation: "check_permissions",
			Path:      ks.config.OutputDirectory,
			Err:       fmt.Errorf("path is not a directory"),
		}
	}

	// Check if directory is writable by attempting to create a temp file
	tmpFile, err := os.CreateTemp(ks.config.OutputDirectory, ".permission-test-*")
	if err != nil {
		return &FileOperationError{
			Operation: "check_permissions",
			Path:      ks.config.OutputDirectory,
			Err:       fmt.Errorf("directory is not writable: %w", err),
		}
	}

	// Clean up test file
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	return nil
}

// FileExists checks if a file exists at the given path
func (ks *KeyStoreService) FileExists(filename string) (bool, error) {
	if filename == "" {
		return false, fmt.Errorf("filename cannot be empty")
	}

	cleanPath := filepath.Clean(filename)
	_, err := os.Stat(cleanPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, &FileOperationError{
		Operation: "file_exists",
		Path:      cleanPath,
		Err:       err,
	}
}

// GetKeystoreFilePath returns the full path for a keystore file given an address
func (ks *KeyStoreService) GetKeystoreFilePath(address string) (string, error) {
	cleanAddress := strings.ToLower(strings.TrimPrefix(address, "0x"))
	if len(cleanAddress) != 40 {
		return "", fmt.Errorf("invalid address length: expected 40 characters, got %d", len(cleanAddress))
	}

	filename := fmt.Sprintf("0x%s.json", cleanAddress)
	return filepath.Join(ks.config.OutputDirectory, filename), nil
}

// GetPasswordFilePath returns the full path for a password file given an address
func (ks *KeyStoreService) GetPasswordFilePath(address string) (string, error) {
	cleanAddress := strings.ToLower(strings.TrimPrefix(address, "0x"))
	if len(cleanAddress) != 40 {
		return "", fmt.Errorf("invalid address length: expected 40 characters, got %d", len(cleanAddress))
	}

	filename := fmt.Sprintf("0x%s.pwd", cleanAddress)
	return filepath.Join(ks.config.OutputDirectory, filename), nil
}

// RemoveKeystoreFiles removes both keystore and password files for a given address
func (ks *KeyStoreService) RemoveKeystoreFiles(address string) error {
	if !ks.config.Enabled {
		return fmt.Errorf("keystore service is disabled")
	}

	keystorePath, err := ks.GetKeystoreFilePath(address)
	if err != nil {
		return fmt.Errorf("failed to get keystore path: %w", err)
	}

	passwordPath, err := ks.GetPasswordFilePath(address)
	if err != nil {
		return fmt.Errorf("failed to get password path: %w", err)
	}

	// Remove keystore file if it exists
	if exists, err := ks.FileExists(keystorePath); err != nil {
		return fmt.Errorf("failed to check keystore file: %w", err)
	} else if exists {
		if err := os.Remove(keystorePath); err != nil {
			return &FileOperationError{
				Operation: "remove_keystore",
				Path:      keystorePath,
				Err:       err,
			}
		}
	}

	// Remove password file if it exists
	if exists, err := ks.FileExists(passwordPath); err != nil {
		return fmt.Errorf("failed to check password file: %w", err)
	} else if exists {
		if err := os.Remove(passwordPath); err != nil {
			return &FileOperationError{
				Operation: "remove_password",
				Path:      passwordPath,
				Err:       err,
			}
		}
	}

	return nil
}

// ValidateFilePermissions checks if a file has the expected permissions
func (ks *KeyStoreService) ValidateFilePermissions(filename string, expectedPerm os.FileMode) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	info, err := os.Stat(filename)
	if err != nil {
		return &FileOperationError{
			Operation: "validate_permissions",
			Path:      filename,
			Err:       err,
		}
	}

	actualPerm := info.Mode().Perm()
	if actualPerm != expectedPerm {
		return &FileOperationError{
			Operation: "validate_permissions",
			Path:      filename,
			Err:       fmt.Errorf("incorrect permissions: got %o, expected %o", actualPerm, expectedPerm),
		}
	}

	return nil
}

// DecryptPrivateKey decrypts a private key from a KeyStore V3 structure
func DecryptPrivateKey(ks *KeyStoreV3, password string) ([]byte, error) {
	if ks == nil {
		return nil, fmt.Errorf("keystore cannot be nil")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Validate keystore
	if err := ks.Validate(); err != nil {
		return nil, fmt.Errorf("invalid keystore: %w", err)
	}

	// Get cipher parameters
	iv, err := hex.DecodeString(ks.Crypto.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("invalid IV hex: %w", err)
	}

	ciphertext, err := hex.DecodeString(ks.Crypto.CipherText)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext hex: %w", err)
	}

	expectedMAC, err := hex.DecodeString(ks.Crypto.MAC)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC hex: %w", err)
	}

	// Derive key using the same KDF parameters
	var derivedKey []byte
	switch ks.Crypto.KDF {
	case "scrypt":
		params, err := ks.GetScryptParams()
		if err != nil {
			return nil, fmt.Errorf("failed to get scrypt params: %w", err)
		}

		salt, err := hex.DecodeString(params.Salt)
		if err != nil {
			return nil, fmt.Errorf("invalid salt hex: %w", err)
		}

		derivedKey, err = DeriveKeyScrypt([]byte(password), salt, params.N, params.R, params.P, params.DKLen)
		if err != nil {
			return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
		}

	case "pbkdf2":
		params, err := ks.GetPBKDF2Params()
		if err != nil {
			return nil, fmt.Errorf("failed to get PBKDF2 params: %w", err)
		}

		salt, err := hex.DecodeString(params.Salt)
		if err != nil {
			return nil, fmt.Errorf("invalid salt hex: %w", err)
		}

		derivedKey, err = DeriveKeyPBKDF2([]byte(password), salt, params.C, params.DKLen)
		if err != nil {
			return nil, fmt.Errorf("PBKDF2 key derivation failed: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported KDF: %s", ks.Crypto.KDF)
	}

	// Verify MAC
	valid, err := VerifyMAC(derivedKey, ciphertext, expectedMAC)
	if err != nil {
		return nil, fmt.Errorf("MAC verification failed: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("MAC verification failed: incorrect password or corrupted keystore")
	}

	// Decrypt private key
	encryptionKey := derivedKey[:16]
	privateKeyBytes, err := DecryptAES128CTR(ciphertext, encryptionKey, iv)
	if err != nil {
		return nil, fmt.Errorf("AES decryption failed: %w", err)
	}

	return privateKeyBytes, nil
}
