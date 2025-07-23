---
inclusion: always
---

# Bloco Wallet Generator

## Product Overview
A high-performance CLI tool for generating Ethereum "bloco" wallets with custom prefixes and suffixes. The tool creates vanity Ethereum addresses matching specific hex patterns at the beginning (prefix) and/or end (suffix) of addresses.

## Core Features
- **Pattern Matching**: Generate addresses with custom prefix/suffix combinations
- **Statistical Analysis**: Real-time difficulty calculations and probability estimates
- **Progress Tracking**: Visual progress bars with performance metrics
- **Benchmarking**: Performance testing for address generation speed
- **Checksum Support**: Full EIP-55 compliant validation
- **Cross-platform**: Support for Linux, Windows, and macOS

## Development Guidelines
- **Command Structure**: Use Cobra framework for all CLI commands
- **Error Handling**: Provide user-friendly error messages with suggestions
- **Performance**: Optimize for high-speed address generation (50k+ addr/s)
- **Testing**: Include comprehensive tests for all cryptographic functions
- **Documentation**: Document complex algorithms and cryptographic operations

## Security Requirements
- Use crypto/rand for cryptographically secure random number generation
- Implement proper secp256k1 elliptic curve cryptography
- Support EIP-55 checksum validation for all addresses
- Never log or expose private keys in error messages or logs
- Validate all user inputs for proper hex formatting

## Code Conventions
- Follow Go standard formatting (use `go fmt`)
- Use descriptive function and variable names
- Group functions by logical categories (crypto, validation, CLI, etc.)
- Implement comprehensive error handling with user-friendly messages
- Document performance-critical sections with benchmarks

## User Experience
- Provide clear progress indicators for long-running operations
- Display statistical information in human-readable format
- Support both interactive and non-interactive modes
- Include examples in help text for common use cases