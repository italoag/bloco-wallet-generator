# Gemini Project Guidelines: Bloco Wallet Generator

## 1. Product Overview

**Bloco Wallet Generator** is a high-performance CLI tool for generating Ethereum "bloco" wallets with custom prefixes and suffixes. It creates Ethereum addresses matching specific hex patterns at the beginning (prefix) and/or end (suffix) of the address.

### Core Features
- **Pattern Matching**: Generate addresses with custom prefix/suffix combinations.
- **Statistical Analysis**: Real-time difficulty calculations and probability estimates.
- **Progress Tracking**: Visual progress bars with performance metrics.
- **Benchmarking**: Performance testing for address generation speed.
- **Checksum Support**: Full EIP-55 compliant validation.
- **Multi-threading**: Parallel processing for optimal performance.
- **Cross-platform**: Support for Linux, Windows, and macOS.

### User Experience
- Clear progress indicators for long-running operations.
- Human-readable statistical information.
- Support for both interactive and non-interactive modes.
- Examples in help text for common use cases.
- Formatted large numbers for readability.
- Estimated completion time for long-running tasks.

## 2. Project Structure

### Architecture
- **Monolithic Design**: All core functionality is currently in `main.go`.
- **Testing**: A comprehensive test suite is located in `main_test.go`.
- **Build System**: A `Makefile` is used for building, testing, and cross-platform compilation.

### Code Organization (`main.go`)
1.  **Data Structures**: `Wallet`, `WalletResult`, `Statistics`, `BenchmarkResult`.
2.  **Utility Functions**: Validation, formatting.
3.  **Cryptographic Functions**: `privateToAddress()`, `getRandomWallet()`, `toChecksumAddress()`.
4.  **Statistical Calculations**: `computeDifficulty()`, `computeProbability()`.
5.  **Core Generation Logic**: `generateBlocoWallet()`, `generateMultipleWallets()`.
6.  **CLI Commands**: `rootCmd`, `statsCmd`, `benchmarkCmd` (using Cobra).
7.  **Main Function**: Initialization and execution.

## 3. Tech Stack

### Core Technologies
- **Go**: Version 1.24.5+
- **Go Modules**: For dependency management.

### Key Dependencies
- **Cobra (`v1.9.1`)**: CLI framework.
- **Go-Ethereum (`v1.16.1`)**: Cryptographic functions.
- **x/crypto (`v0.40.0`)**: Keccak-256 hashing.
- **dcrd/secp256k1 (`v4`)**: Elliptic curve operations.

### Build System (`Makefile`)
- `make build`: Build the main binary.
- `make test`: Run all tests.
- `make clean`: Clean build artifacts.
- `make build-all`: Build for all supported platforms (Linux, Windows, macOS).
- `make bench`: Run benchmarks.

## 4. Testing Guidelines

### Core Principles
- **Test-First**: Run `make test` before making changes.
- **Incremental Testing**: Add tests for new functionality as it's implemented.
- **Regression Prevention**: Ensure new changes do not break existing tests.

### Required Commands
- **Basic Tests**: `make test`
- **Race Detection**: `make test-race`
- **Benchmarks**: `make bench`

### Test Coverage
- All new functions must have corresponding unit tests.
- Error handling paths must be tested.
- Backward compatibility must be verified.

### Security Testing
- Manually verify that private keys are never logged or exposed.
- Test input validation with invalid hex characters and lengths.
- Ensure checksum validation works correctly.
