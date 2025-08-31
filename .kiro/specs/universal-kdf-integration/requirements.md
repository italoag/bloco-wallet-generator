# Requirements Document

## Introduction

This feature integrates the Universal KDF (Key Derivation Function) system from the bloco-wallet project into bloco-eth to ensure maximum compatibility with Ethereum clients like Besu, Go Ethereum, Anvil, Reth, and Hyperledger Firefly. The current keystore implementation in bloco-eth may have compatibility issues due to insufficient KDF parameter validation and limited support for different KDF variations. The Universal KDF system provides comprehensive support for multiple KDF algorithms with proper parameter validation, normalization, and compatibility analysis.

## Requirements

### Requirement 1

**User Story:** As a developer using bloco-eth generated keystores, I want the keystore files to be compatible with all major Ethereum clients, so that I can use the same keystore files across different blockchain infrastructure tools.

#### Acceptance Criteria

1. WHEN a keystore is generated THEN the system SHALL use the Universal KDF service for key derivation
2. WHEN using scrypt KDF THEN the system SHALL validate parameters according to industry standards (N must be power of 2, proper ranges for r, p, dklen)
3. WHEN using PBKDF2 KDF THEN the system SHALL support multiple hash functions (SHA-256, SHA-512) and validate iteration counts
4. WHEN generating keystores THEN the system SHALL be compatible with Besu, Go Ethereum, Anvil, Reth, and Hyperledger Firefly clients
5. WHEN keystore generation fails THEN the system SHALL provide detailed error messages indicating the specific KDF validation issue

### Requirement 2

**User Story:** As a user generating multiple keystores, I want the system to automatically select optimal KDF parameters based on security requirements, so that my keystores have appropriate security levels without manual configuration.

#### Acceptance Criteria

1. WHEN no KDF is specified THEN the system SHALL default to scrypt with secure parameters (N=262144, r=8, p=1, dklen=32)
2. WHEN scrypt is selected THEN the system SHALL validate that N is a power of 2 and within acceptable ranges (1024 to 67108864)
3. WHEN PBKDF2 is selected THEN the system SHALL use a minimum of 100000 iterations for security
4. WHEN generating keystores THEN the system SHALL analyze parameter security levels and warn about weak configurations
5. WHEN memory usage would exceed 2GB THEN the system SHALL reject the parameters and suggest alternatives

### Requirement 3

**User Story:** As a system administrator deploying blockchain infrastructure, I want keystore files to follow the exact Ethereum KeyStore V3 specification, so that they work seamlessly with existing Ethereum tooling and infrastructure.

#### Acceptance Criteria

1. WHEN generating keystores THEN the system SHALL produce files that strictly conform to Ethereum KeyStore V3 format
2. WHEN using different KDF algorithms THEN the system SHALL normalize KDF names to handle case variations and aliases
3. WHEN processing salt parameters THEN the system SHALL handle multiple input formats (hex strings, byte arrays, number arrays)
4. WHEN generating MAC values THEN the system SHALL use the correct portion of derived keys according to Ethereum standards
5. WHEN validating keystores THEN the system SHALL verify all cryptographic parameters before file generation

### Requirement 4

**User Story:** As a developer debugging keystore issues, I want detailed compatibility analysis and validation reporting, so that I can identify and resolve keystore compatibility problems quickly.

#### Acceptance Criteria

1. WHEN keystore generation completes THEN the system SHALL provide a compatibility report indicating supported clients
2. WHEN KDF parameters are suboptimal THEN the system SHALL provide security level analysis (Low, Medium, High, Very High)
3. WHEN validation fails THEN the system SHALL provide specific parameter ranges and suggested corrections
4. WHEN using verbose mode THEN the system SHALL log KDF operations without exposing sensitive data
5. WHEN compatibility issues are detected THEN the system SHALL suggest specific parameter adjustments for better compatibility

### Requirement 5

**User Story:** As a security-conscious user, I want the keystore generation to use cryptographically secure random values and proper key derivation, so that my private keys are protected with industry-standard security practices.

#### Acceptance Criteria

1. WHEN generating salt values THEN the system SHALL use cryptographically secure random number generation
2. WHEN deriving keys THEN the system SHALL use the exact algorithms and parameters specified in the keystore
3. WHEN generating multiple keystores THEN the system SHALL use unique salt values for each keystore
4. WHEN handling sensitive data THEN the system SHALL clear temporary cryptographic material from memory
5. WHEN logging operations THEN the system SHALL never log private keys, passwords, or derived key material