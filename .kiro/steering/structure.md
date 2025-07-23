---
inclusion: always
---

# Project Structure & Organization

## Architecture Overview
- **Single-File Architecture**: Monolithic design with all core functionality in `main.go`
- **Testing**: Comprehensive test suite in `main_test.go`
- **Build System**: Makefile-based with cross-platform support

## Code Organization

### Core Components in `main.go`
- **Data Structures**: `Wallet`, `WalletResult`, `Statistics`, `BenchmarkResult`
- **Cryptographic Functions**: `privateToAddress()`, `getRandomWallet()`, `toChecksumAddress()`
- **Pattern Validation**: `isValidBlocoAddress()`, `isValidChecksum()`, `isValidHex()`
- **Statistical Analysis**: `computeDifficulty()`, `computeProbability()`, `computeProbability50()`
- **Generation Logic**: `generateBlocoWallet()`, `generateMultipleWallets()`, `runBenchmark()`
- **CLI Framework**: `rootCmd`, `statsCmd`, `benchmarkCmd` (using Cobra)

### Function Organization
1. Data structures and constants
2. Utility functions (validation, formatting)
3. Cryptographic functions
4. Statistical calculations
5. Core generation logic
6. CLI command definitions
7. Main function and initialization

## Code Style Guidelines

### Go Conventions
- Use Go standard formatting (`go fmt`)
- Group imports: standard library → third-party → local packages
- Implement comprehensive error handling with user-friendly messages
- Use descriptive function and variable names
- Document complex algorithms and crypto operations with comments

### Performance Considerations
- Optimize critical paths for high-speed address generation
- Use memory-efficient random number generation
- Include benchmarks for performance-critical sections
- Minimize allocations in high-frequency operations

## Testing Requirements
- **Unit Tests**: Cover all core cryptographic and validation functions
- **Benchmark Tests**: Measure performance of key operations
- **Integration Tests**: Verify end-to-end workflows
- **Property-Based Tests**: Validate cryptographic function properties

## Documentation Structure
- **README.md**: Primary documentation (English)
- **SUMMARY.md**: Enhanced features summary (Portuguese)
- **Code Comments**: Document complex algorithms and security considerations

## Security Guidelines
- Use crypto/rand for secure random number generation
- Implement proper secp256k1 elliptic curve cryptography
- Support EIP-55 checksum validation
- Never expose private keys in logs or error messages
- Validate all user inputs for proper hex formatting