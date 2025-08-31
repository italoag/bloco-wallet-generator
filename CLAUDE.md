# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bloco-ETH is a high-performance CLI tool for generating Ethereum wallets with custom address patterns (prefixes/suffixes). It features multi-threaded generation, TUI interface, keystore generation, benchmarking, and statistical analysis.

## Build & Development Commands

### Essential Commands
- `make build` - Build the main binary (`bloco-eth`)
- `make test` - Run all tests with verbose output
- `make test-race` - Run tests with race detection
- `make clean` - Clean build artifacts
- `make run` - Build and run with default parameters (`--prefix abc --count 1`)

### Testing Commands
- `make test-unit` - Run unit tests only (skip integration tests)
- `make test-coverage` - Generate coverage report (creates coverage.html)
- `make bench` - Run benchmarks

### Quality Commands  
- `make fmt` - Format code with `go fmt`
- `make vet` - Run `go vet` for static analysis
- `make lint` - Run golangci-lint (if installed)
- `make dev` - Complete development workflow (fmt, vet, test, build)

### Cross-Platform Builds
- `make build-all` - Build for all platforms (Linux, Windows, macOS)
- `make build-linux`, `make build-windows`, `make build-darwin` - Platform-specific builds

### Application Commands
- `./bloco-eth --help` - Show complete CLI help
- `./bloco-eth stats --prefix abc` - Show pattern difficulty statistics
- `./bloco-eth benchmark --attempts 5000` - Run performance benchmark
- `BLOCO_TUI=false ./bloco-eth --prefix a --count 1` - Force text mode (no TUI)
- `BLOCO_DEBUG=1 ./bloco-eth --prefix a` - Enable debug output

## Architecture Overview

### Modular Design
The codebase has been refactored from a monolithic structure to a well-organized modular architecture:

**Core Packages:**
- `cmd/bloco-eth/` - Main application entry point
- `internal/cli/` - CLI command handling and application logic
- `internal/worker/` - Worker pool management and generation logic
- `internal/crypto/` - Cryptographic operations, keystore generation
- `internal/tui/` - Terminal User Interface components
- `internal/config/` - Configuration management
- `internal/validation/` - Address validation strategies
- `pkg/wallet/`, `pkg/errors/`, `pkg/utils/` - Shared types and utilities

### Key Components
- **Worker Pool**: Uses `ants` library for efficient goroutine management
- **TUI System**: Bubble Tea-based interface with fallback to text mode
- **Keystore Generation**: Supports scrypt/PBKDF2 with KeyStore V3 format
- **Statistics Engine**: Real-time difficulty calculations and performance metrics
- **Secure Logging**: Never logs private keys or sensitive data

### TUI Behavior
- TUI automatically enables when supported and `--progress` flag is used
- Environment variable `BLOCO_TUI=false` forces text mode
- Environment variable `BLOCO_TUI=force` forces TUI mode (for testing)
- Graceful fallback to text mode if TUI fails

### Threading
- Auto-detects CPU cores by default (`--threads 0`)
- Worker pool uses ants for efficient goroutine management
- Real-time performance monitoring with thread balance scoring

## Configuration System

### Environment Variables
- `BLOCO_DEBUG=1` - Enable debug output and detailed logging
- `BLOCO_TUI=false/true/force` - Control TUI behavior
- Standard Go environment variables (GOOS, GOARCH) for cross-compilation

### Command-Line Structure
The CLI uses Cobra framework with these main commands:
- Root command: Generate wallets with patterns
- `stats` - Analyze pattern difficulty statistics
- `benchmark` - Run performance benchmarks  
- `version` - Show version information

### Keystore Support
- Automatic KeyStore V3 file generation compatible with MetaMask/geth
- Support for scrypt and PBKDF2 key derivation functions
- Configurable security levels and KDF parameters
- Compatibility analysis and security recommendations

## Development Guidelines

### Code Quality
- Always run `make dev` before committing (formats, vets, tests, builds)
- Maintain test coverage for new functionality
- Follow existing patterns for error handling using `pkg/errors`
- Never log or expose private keys in any output

### Testing Strategy
- Unit tests for individual components (`*_test.go` files)
- Integration tests for cross-component functionality  
- Property-based tests for cryptographic functions
- Benchmark tests for performance validation
- Use `make test-race` to detect race conditions

### Modular Development
- Keep packages focused and well-separated
- Use interfaces for testability (see `worker.WorkerPool` interface)
- Maintain clean dependencies between internal packages
- Follow Go best practices for package organization

## Security Considerations

### Critical Security Rules
- Private keys are NEVER logged, printed, or stored in plain text
- All logging is designed to be secure by default
- Keystore passwords use cryptographically secure random generation
- KDF parameters are validated for security compliance
- All cryptographic operations use established libraries (go-ethereum, x/crypto)

### Validation
- All user inputs are validated before processing
- Address checksum validation follows EIP-55 standard
- Pattern validation ensures only valid hex characters
- KDF parameter validation prevents weak configurations

This architecture supports high-performance wallet generation while maintaining security, modularity, and excellent user experience through both CLI and TUI interfaces.