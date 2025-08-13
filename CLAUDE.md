# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a high-performance Ethereum bloco wallet generator built in Go. It generates Ethereum wallets with custom prefixes and/or suffixes using multi-threaded parallel processing for optimal performance. Features automatic wallet logging, EIP-55 checksum support, and comprehensive statistics collection.

## Key Commands

### Development Commands
```bash
# Build the application
make build

# Run all tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run benchmarks
make bench

# Format and vet code
make fmt && make vet

# Complete development workflow (format, vet, test, build)
make dev

# CI workflow (deps, format, vet, race tests)
make ci

# Additional useful commands
make clean      # Clean build artifacts
make install    # Install to GOPATH/bin
make lint       # Lint code (requires golangci-lint)
make security   # Security checks (requires gosec)
make demo       # Run comprehensive demo
make examples   # Run usage examples
make stats-test # Test statistics functionality
```

### Application Usage
```bash
# Generate wallet with prefix (auto-detects CPU cores)
./bloco-eth --prefix abc --progress

# Generate wallet with specific thread count
./bloco-eth --prefix abc --threads 8

# Generate wallet with EIP-55 checksum validation
./bloco-eth --prefix ABC --checksum --threads 4

# Analyze pattern difficulty
./bloco-eth stats --prefix deadbeef --suffix 123

# Run performance benchmark
./bloco-eth benchmark --attempts 10000 --pattern "abc" --threads 8

# All generated wallets are automatically logged to wallets-YYYYMMDD.log
```

## Architecture Overview

### Core Components

1. **Multi-threaded Architecture**
   - `Pool` (pool.go): Streamlined worker pool with integrated statistics collection
   - `StatsCollector` (stats.go): Thread-safe statistics aggregation across workers
   - `ProgressManager` (progress_manager.go): Real-time progress display management
   - `WalletLogger` (logger.go): Automatic wallet logging to daily files
   - `ThreadMetrics`: Thread performance monitoring and efficiency calculations
   - `ThreadValidation`: Thread count validation and CPU detection

2. **Wallet Logging System**
   - `WalletLogger`: Automatic logging of all generated wallets
   - Daily log files with timestamp (`wallets-YYYYMMDD.log`)
   - Comprehensive data: address, public key, private key, attempts, duration
   - Thread-safe logging with proper file handling and cleanup
   - Incremental logging for multiple sessions per day

3. **Terminal User Interface (TUI) System**
   - `TUIManager` (tui/manager.go): Terminal capability detection and TUI coordination
   - `ProgressDisplay` (tui/progress.go): Advanced progress visualization with Bubble Tea
   - `StatsDisplay` (tui/stats.go): Real-time statistics rendering with styling
   - `Styles` (tui/styles.go): Consistent visual theming and color schemes
   - `Utils` (tui/utils.go): TUI utility functions and helpers

4. **Core Generation Logic**
   - `GenerateWalletWithContext()`: Main wallet generation with context cancellation
   - `isValidBlocoAddress()`: Pattern matching with checksum validation
   - `toChecksumAddress()`: Complete EIP-55 checksum implementation
   - `isEIP55Checksum()`: Validates EIP-55 checksum patterns
   - Integrated statistics collection during generation

5. **Statistics and Analysis**
   - `StatsCollector`: Real-time statistics collection from workers
   - `WorkerStats`: Individual worker performance metrics
   - `AggregatedStats`: Combined statistics across all threads
   - `PerformanceMetrics`: Detailed performance analysis and efficiency
   - `computeDifficulty()`: Calculates generation difficulty based on pattern length and checksum
   - `computeProbability()`: Probability calculations for success estimates

### File Structure
- `main.go`: CLI interface and main application entry point
- `internal/worker/pool.go`: Streamlined worker pool implementation
- `internal/worker/stats.go`: Comprehensive statistics collection system
- `internal/worker/interface.go`: Worker pool interface definitions
- `pkg/wallet/types.go`: Wallet data structures and validation
- `pkg/wallet/logger.go`: Automatic wallet logging system
- `internal/cli/commands.go`: CLI command implementations
- `internal/tui/`: Terminal User Interface components with Bubble Tea integration
- `*_test.go`: Comprehensive test suite for all components

## Technical Details

### Dependencies
- `github.com/spf13/cobra`: CLI framework
- `github.com/ethereum/go-ethereum/crypto`: Ethereum cryptographic functions
- `golang.org/x/crypto/sha3`: Keccak-256 hashing
- `crypto/rand`: Secure random number generation
- `github.com/charmbracelet/bubbletea`: Terminal User Interface framework
- `golang.org/x/term`: Terminal capability detection

### Performance Optimizations
- **Multi-threading**: Linear scaling with CPU cores (up to 8x performance improvement)
- **Real-time Statistics**: Worker performance monitoring with 100ms updates
- **Thread-safe Operations**: Proper synchronization without performance penalties
- **CPU Auto-detection**: Automatically uses all available CPU cores by default
- **EIP-55 Checksum**: Optimized checksum validation and generation
- **Automatic Logging**: Non-blocking wallet logging with minimal performance impact

### Thread Safety
- All cryptographic operations are worker-local
- Statistics are aggregated through thread-safe channels
- Graceful shutdown coordination when wallet is found
- Thread-safe wallet logging with mutex protection
- Context-based cancellation for clean shutdown
- No shared mutable state between workers

## Testing Strategy

### Test Categories
- **Unit Tests**: Individual function testing (pool_test.go)
- **Integration Tests**: End-to-end wallet generation and logging
- **Performance Tests**: Multi-threading efficiency and statistics collection
- **Statistics Tests**: Real-time metrics accuracy and aggregation
- **TUI Tests**: Terminal interface components (tui/*_test.go)
- **Checksum Tests**: EIP-55 validation and generation accuracy
- **Logging Tests**: Wallet logging functionality and file handling

### Running Tests
```bash
# Run all tests
make test

# Run with race detection
make test-race

# Generate coverage report
make test-coverage

# Run benchmarks
make bench

# Run specific test categories
make test-unit      # Unit tests only
make perf-test      # Performance testing
make benchmark-test # Benchmark testing
```

## Security Considerations

- Uses cryptographically secure random number generation (`crypto/rand`)
- Proper secp256k1 elliptic curve cryptography
- Complete EIP-55 checksum validation and generation
- Secure wallet logging with proper file permissions
- No shared cryptographic state between workers
- Thread-safe operations prevent race conditions
- Context cancellation prevents resource leaks

## Common Patterns

### Adding New CLI Commands
1. Define command variable with `&cobra.Command{}`
2. Add to `rootCmd` in `init()` function
3. Define flags specific to the command
4. Implement validation and core logic in `Run` function

### Extending Statistics
1. Add fields to `WorkerStats` or `AggregatedStats` struct
2. Update worker statistics collection in `pool.go`
3. Modify `StatsCollector` aggregation logic
4. Update display functions for new metrics
5. Ensure thread-safe access to new statistics

### Performance Optimization
- Use worker-local resources to avoid contention
- Implement non-blocking statistics collection
- Use channels for thread-safe communication
- Monitor worker efficiency and thread utilization
- Profile with `go tool pprof` when making performance changes
- Optimize EIP-55 checksum calculations for performance

### Working with TUI Components
- TUI system automatically detects terminal capabilities
- Use `TUIManager` for capability detection and coordination
- Styling is centralized in `tui/styles.go` for consistency
- All TUI components are fully tested with mock terminal support
- Graceful fallback to basic text output when TUI is not supported

## Building and Deployment

### Build Targets
```bash
# Local build
make build

# Cross-platform builds
make build-all  # Builds for Linux, Windows, macOS (Intel + ARM)

# Individual platform builds
make build-linux
make build-windows  
make build-darwin
make build-darwin-arm64
```

### Release Process
```bash
# Prepare release (includes clean, CI, and all platform builds)
make release
```

## Troubleshooting

### Common Issues
- **Build failures**: Run `go mod tidy` to resolve dependencies
- **Performance issues**: Verify thread count with `--threads` flag
- **Memory issues**: Check object pool usage and cleanup
- **Test failures**: Use `make test-race` to detect race conditions

### Debugging Multi-threading
- Use `--threads 1` to test single-threaded behavior
- Monitor thread utilization statistics in benchmark output
- Check worker efficiency ratios for load balancing issues

## Guidelines

- Never use the `timeout` command, Mac OS don't have this command. 