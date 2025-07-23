---
inclusion: always
---

# Technology Stack & Build System

## Core Technology
- **Go 1.24.5+** - Primary programming language
- **Module**: `bloco-wallet` (Go modules for dependency management)
- **Architecture**: Single-file monolithic design with all core functionality in `main.go`
- **Binary**: `bloco-eth` (Cross-platform naming: `bloco-eth-{os}-{arch}[.exe]`)

## Key Dependencies
- **github.com/spf13/cobra v1.9.1** - CLI framework for command structure
- **github.com/ethereum/go-ethereum v1.16.1** - Ethereum cryptographic functions
- **golang.org/x/crypto v0.40.0** - Keccak-256 hashing implementation
- **github.com/decred/dcrd/dcrec/secp256k1/v4** - secp256k1 elliptic curve operations

## Cryptographic Implementation
- **secp256k1** elliptic curve for private/public key generation
- **Keccak-256** hashing for Ethereum address derivation
- **crypto/rand** for cryptographically secure random number generation
- **EIP-55** checksum validation for address verification

## Architecture Patterns
- **Command Structure**: Root command with `stats` and `benchmark` subcommands
- **Core Components**:
  1. Cryptographic Engine: Private key generation and address derivation
  2. Pattern Matching: Prefix/suffix validation with checksum support
  3. Statistics Engine: Real-time difficulty and probability calculations
  4. Progress System: Visual progress bars and performance metrics
  5. Benchmark Suite: Performance measurement and analysis

## Performance Requirements
- Optimized for high-speed address generation (50k+ addr/s typical)
- Memory-efficient random number generation
- Real-time statistics with minimal overhead
- Minimize allocations in high-frequency operations
- Configurable progress update intervals

## Build System (Makefile)
```bash
# Essential Commands
make init          # Initialize Go module and download dependencies
make build         # Build the main binary (bloco-eth)
make test          # Run all tests
make clean         # Clean build artifacts
make dev           # Format, vet, test, and build

# Cross-Platform Builds
make build-all     # Build for all platforms
make build-linux   # Linux AMD64
make build-windows # Windows AMD64  
make build-darwin  # macOS AMD64
make build-darwin-arm64 # macOS ARM64 (M1/M2)

# Testing & Quality
make test-race     # Run tests with race detection
make test-coverage # Generate coverage report
make bench         # Run benchmarks
make lint          # Run golangci-lint (if installed)

# Demo & Examples
make demo          # Run comprehensive demo
make examples      # Run various usage examples
make perf-test     # Performance tests with different complexities
```

## Code Style & Conventions
- Use Go standard formatting (`go fmt`)
- Group imports: standard library → third-party → local packages
- Implement comprehensive error handling with user-friendly messages
- Use descriptive function and variable names
- Document complex algorithms and crypto operations with comments
- Group functions by logical categories (crypto, validation, CLI, etc.)

## Security Guidelines
- Use crypto/rand for secure random number generation
- Implement proper secp256k1 elliptic curve cryptography
- Support EIP-55 checksum validation
- Never expose private keys in logs or error messages
- Validate all user inputs for proper hex formatting