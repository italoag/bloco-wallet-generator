# KDF Compatibility Analyzer

This package provides comprehensive compatibility analysis and security assessment for Key Derivation Functions (KDF) used in Ethereum keystores.

## Features

### KDFCompatibilityAnalyzer

The `KDFCompatibilityAnalyzer` provides:

1. **Keystore Compatibility Analysis** - Analyzes keystore KDF parameters for compatibility with major Ethereum clients
2. **Security Level Assessment** - Evaluates KDF parameters and assigns security levels (Low, Medium, High, Very High)
3. **Client-Specific Compatibility** - Checks compatibility with Besu, geth, Anvil, Reth, and Hyperledger Firefly
4. **Parameter Optimization** - Suggests optimal parameters for different security levels and constraints
5. **Time Estimation** - Estimates key derivation time based on parameters
6. **Detailed Reporting** - Provides comprehensive reports with issues, warnings, and suggestions

## Supported KDF Algorithms

- **Scrypt** - Memory-hard function with GPU resistance
- **PBKDF2** - Password-Based Key Derivation Function 2 with multiple hash functions
  - PBKDF2-SHA256
  - PBKDF2-SHA512

## Usage Example

```go
// Create service and analyzer
service := NewUniversalKDFService()
analyzer := NewKDFCompatibilityAnalyzer(service)

// Analyze keystore compatibility
crypto := &CryptoParams{
    KDF: "scrypt",
    KDFParams: map[string]interface{}{
        "n":     262144,
        "r":     8,
        "p":     1,
        "dklen": 32,
        "salt":  "0123456789abcdef0123456789abcdef",
    },
}

report, err := analyzer.AnalyzeKeystore(crypto)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Compatible: %v\n", report.Compatible)
fmt.Printf("Security Level: %s\n", report.SecurityLevel)
fmt.Printf("Issues: %v\n", report.Issues)
fmt.Printf("Warnings: %v\n", report.Warnings)
fmt.Printf("Suggestions: %v\n", report.Suggestions)
```

## Security Analysis

### Scrypt Security Levels

- **Very High**: N ≥ 262144 (2^18), r ≥ 8, p ≥ 1
- **High**: N ≥ 65536 (2^16), r ≥ 8, p ≥ 1  
- **Medium**: N ≥ 16384 (2^14), r ≥ 8, p ≥ 1
- **Low**: Below medium thresholds

### PBKDF2 Security Levels

- **Very High**: c ≥ 600000 iterations
- **High**: c ≥ 310000 iterations
- **Medium**: c ≥ 120000 iterations
- **Low**: Below medium thresholds

### Client Compatibility

The analyzer checks compatibility with:

- **geth** - Go Ethereum implementation
- **besu** - Hyperledger Besu (Java)
- **anvil** - Foundry's local Ethereum node
- **reth** - Rust Ethereum implementation
- **firefly** - Hyperledger Firefly (enterprise)

## Parameter Optimization

Get optimized parameters for specific security levels:

```go
// Get optimized scrypt parameters for high security with 256MB memory limit
params, err := analyzer.GetOptimizedParams("scrypt", SecurityLevelHigh, 256)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Optimized parameters: %+v\n", params)
```

## Time Estimation

Estimate key derivation time:

```go
duration := analyzer.EstimateDerivationTime("scrypt", params)
fmt.Printf("Estimated derivation time: %v\n", duration)
```

## Requirements Coverage

This implementation satisfies the following requirements:

- **4.1**: Compatibility analysis reporting with detailed compatibility reports
- **4.2**: Security level assessment (Low, Medium, High, Very High) 
- **4.3**: Detailed compatibility reporting with issues, warnings, and suggestions
- **4.4**: Actionable suggestions for parameter improvements

## Testing

The package includes comprehensive tests:

- Unit tests for all analyzer functions
- Integration tests demonstrating complete workflows
- Requirements coverage tests verifying all specifications are met
- Client compatibility validation tests
- Security analysis accuracy tests

Run tests with:
```bash
go test ./internal/crypto/kdf -v
```