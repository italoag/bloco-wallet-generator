---
inclusion: always
---

# Bloco Wallet Generator

## Product Overview
A high-performance CLI tool for generating Ethereum "bloco" wallets with custom prefixes and suffixes. The tool creates Ethereum addresses matching specific hex patterns at the beginning (prefix) and/or end (suffix) of addresses.

## Core Features
- **Pattern Matching**: Generate addresses with custom prefix/suffix combinations
- **Statistical Analysis**: Real-time difficulty calculations and probability estimates
- **Progress Tracking**: Visual progress bars with performance metrics
- **Benchmarking**: Performance testing for address generation speed
- **Checksum Support**: Full EIP-55 compliant validation
- **Multi-threading**: Parallel processing for optimal performance
- **Cross-platform**: Support for Linux, Windows, and macOS

## Architecture Patterns
- **Worker Pool**: Thread-safe parallel processing with work distribution
- **Object Pooling**: Memory optimization with reusable cryptographic objects
- **Progress Management**: Real-time statistics and visual feedback
- **Command Structure**: Cobra-based CLI with stats and benchmark subcommands

## Performance Guidelines
- Optimize critical paths for high-speed address generation (50k+ addr/s)
- Use object pooling for cryptographic operations to reduce GC pressure
- Minimize allocations in high-frequency operations
- Implement thread-safe statistics collection
- Balance thread count with system capabilities

## Security Requirements
- Use crypto/rand for cryptographically secure random number generation
- Implement proper secp256k1 elliptic curve cryptography
- Support EIP-55 checksum validation for all addresses
- Never log or expose private keys in error messages or logs
- Validate all user inputs for proper hex formatting
- Clear sensitive data from memory when returning objects to pools

## Code Conventions
- Follow Go standard formatting (use `go fmt`)
- Use descriptive function and variable names
- Group functions by logical categories (crypto, validation, CLI, etc.)
- Implement comprehensive error handling with user-friendly messages
- Document performance-critical sections with benchmarks
- Use sync.Pool for memory-intensive operations
- Prefer stack allocation over heap allocation where possible

## User Experience
- Provide clear progress indicators for long-running operations
- Display statistical information in human-readable format
- Support both interactive and non-interactive modes
- Include examples in help text for common use cases
- Format large numbers with separators for readability
- Show estimated completion time for long-running operations