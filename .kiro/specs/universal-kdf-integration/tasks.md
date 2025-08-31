# Implementation Plan

- [x] 1. Port Universal KDF core components from bloco-wallet
  - Create new package `internal/crypto/kdf` for Universal KDF system
  - Port `UniversalKDFService`, `KDFHandler` interface, and core types from bloco-wallet
  - Adapt logging interface to work with existing bloco-eth logging system
  - _Requirements: 1.1, 2.1, 3.1_

- [x] 1.1 Create KDF package structure and interfaces
  - Create `internal/crypto/kdf/service.go` with UniversalKDFService
  - Create `internal/crypto/kdf/interfaces.go` with KDFHandler and KDFLogger interfaces
  - Create `internal/crypto/kdf/types.go` with CryptoParams and error types
  - _Requirements: 1.1, 3.1_

- [x] 1.2 Port Scrypt handler with enhanced validation
  - Create `internal/crypto/kdf/scrypt.go` with ScryptHandler implementation
  - Implement parameter validation with power-of-2 check for N parameter
  - Add memory usage validation to prevent system exhaustion
  - Implement secure parameter conversion from various JSON types
  - _Requirements: 1.2, 2.2, 2.5_

- [x] 1.3 Port PBKDF2 handler with multi-hash support
  - Create `internal/crypto/kdf/pbkdf2.go` with PBKDF2Handler implementation
  - Support SHA-256 and SHA-512 hash functions
  - Implement iteration count validation with minimum security thresholds
  - Add PRF (Pseudo Random Function) parameter handling
  - _Requirements: 1.2, 2.3_

- [x] 1.4 Implement KDF name normalization and registration
  - Add case-insensitive KDF name handling (scrypt, Scrypt, SCRYPT)
  - Support KDF aliases (pbkdf2-sha256, pbkdf2_sha256)
  - Implement KDF handler registration system
  - Add default KDF selection logic
  - _Requirements: 3.2_

- [x] 2. Create compatibility analyzer for keystore validation
  - Create `internal/crypto/kdf/analyzer.go` with KDFCompatibilityAnalyzer
  - Implement keystore compatibility analysis against Ethereum clients
  - Add security level assessment (Low, Medium, High, Very High)
  - Create detailed compatibility reporting system
  - _Requirements: 4.1, 4.2, 4.3_

- [x] 2.1 Implement compatibility report generation
  - Create CompatibilityReport struct with detailed analysis fields
  - Implement client-specific compatibility checks for Besu, geth, Anvil, Reth, Firefly
  - Add parameter security analysis based on computational complexity
  - Generate actionable suggestions for parameter improvements
  - _Requirements: 4.1, 4.2, 4.4_

- [x] 2.2 Add security level analysis for KDF parameters
  - Implement computational complexity analysis for scrypt parameters
  - Add iteration count analysis for PBKDF2 parameters
  - Create security level classification algorithm
  - Add recommendations for parameter optimization
  - _Requirements: 4.2, 2.4_

- [x] 3. Enhance existing keystore service with Universal KDF
  - Modify `internal/crypto/keystore.go` to use UniversalKDFService
  - Replace existing DeriveKeyScrypt and DeriveKeyPBKDF2 functions
  - Add compatibility validation before keystore generation
  - Implement enhanced error handling with specific KDF error types
  - _Requirements: 1.1, 1.5, 3.1_

- [x] 3.1 Update KeyStoreService to use Universal KDF
  - Modify `EncryptPrivateKey` function to use UniversalKDFService
  - Add KDF parameter validation before encryption
  - Implement automatic parameter optimization for security
  - Add compatibility analysis integration
  - _Requirements: 1.1, 2.1, 3.1_

- [x] 3.2 Enhance keystore parameter handling
  - Update ScryptParams and PBKDF2Params structures with validation
  - Implement flexible parameter parsing from map[string]interface{}
  - Add salt parameter handling for multiple input formats (hex, bytes, arrays)
  - Ensure proper parameter serialization to JSON
  - _Requirements: 3.3, 3.4_

- [x] 3.3 Add enhanced error handling and reporting
  - Create KDFError type with detailed error information
  - Implement parameter validation error messages with suggested ranges
  - Add recoverable error detection and retry logic
  - Integrate with existing KeyStoreError system
  - _Requirements: 1.5, 4.3, 4.4_

- [x] 4. Implement secure random generation and cryptographic utilities
  - Enhance `GenerateRandomBytes` function with additional entropy validation
  - Add secure salt generation with proper length validation
  - Implement memory clearing for sensitive cryptographic material
  - Add cryptographic parameter validation utilities
  - _Requirements: 5.1, 5.3, 5.4_

- [x] 4.1 Create secure random utilities
  - Create `internal/crypto/random.go` with enhanced random generation
  - Implement entropy validation for cryptographic operations
  - Add secure salt generation with configurable lengths
  - Implement memory clearing utilities for sensitive data
  - _Requirements: 5.1, 5.3, 5.4_

- [x] 4.2 Add cryptographic validation utilities
  - Create parameter validation functions for all supported KDF types
  - Implement range checking with security-based recommendations
  - Add cryptographic strength assessment utilities
  - Create parameter optimization suggestions
  - _Requirements: 2.4, 4.2_

- [x] 5. Update CLI integration and configuration
  - Modify CLI commands to support new KDF options and validation
  - Add compatibility analysis reporting to verbose output
  - Integrate security level warnings into keystore generation
  - Update configuration system to support Universal KDF settings
  - _Requirements: 4.1, 4.2, 4.4_

- [x] 5.1 Update CLI flags and commands
  - Add `--kdf-analysis` flag to show compatibility analysis
  - Add `--security-level` flag to set minimum security requirements
  - Update `--keystore-kdf` flag to support new KDF options
  - Add `--kdf-params` flag for custom parameter specification
  - _Requirements: 4.1, 4.4_

- [x] 5.2 Integrate compatibility reporting in CLI output
  - Add compatibility analysis to verbose keystore generation output
  - Show security level warnings for weak KDF parameters
  - Display client compatibility information when requested
  - Add parameter optimization suggestions to CLI help
  - _Requirements: 4.1, 4.2, 4.4_

- [-] 6. Create comprehensive test suite
  - Write unit tests for all KDF handlers with parameter validation
  - Create integration tests with actual Ethereum client compatibility
  - Add property-based tests for cryptographic correctness
  - Implement performance benchmarks for different KDF parameter sets
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 6.1 Write KDF handler unit tests
  - Test scrypt parameter validation with boundary conditions
  - Test PBKDF2 parameter validation with different hash functions
  - Test key derivation correctness against known test vectors
  - Test error handling for invalid parameters and edge cases
  - _Requirements: 1.2, 1.5_

- [x] 6.2 Create compatibility integration tests
  - Test keystore generation and loading with geth
  - Test keystore compatibility with Besu client
  - Test keystore functionality with Anvil
  - Test keystore integration with Reth and Hyperledger Firefly
  - _Requirements: 1.4_

- [x] 6.3 Add cryptographic property tests
  - Test that different passwords produce different derived keys
  - Verify salt uniqueness produces different keystore outputs
  - Test MAC integrity across different parameter combinations
  - Validate parameter boundary conditions and security properties
  - _Requirements: 5.1, 5.2_

- [x] 7. Update documentation and examples
  - Update README with new KDF options and compatibility information
  - Create examples showing different KDF configurations
  - Document security recommendations for different use cases
  - Add troubleshooting guide for keystore compatibility issues
  - _Requirements: 4.4_

- [x] 7.1 Create KDF configuration examples
  - Document scrypt parameter selection for different security levels
  - Show PBKDF2 configuration examples with different hash functions
  - Provide client-specific configuration recommendations
  - Add performance vs security trade-off examples
  - _Requirements: 2.1, 4.2_

- [x] 7.2 Add compatibility troubleshooting guide
  - Document common keystore compatibility issues and solutions
  - Provide client-specific troubleshooting steps
  - Add parameter optimization guide for different use cases
  - Create FAQ for KDF-related questions
  - _Requirements: 4.3, 4.4_