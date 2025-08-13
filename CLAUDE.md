# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a high-performance Ethereum bloco wallet generator built in Go. It generates Ethereum wallets with custom prefixes and/or suffixes using multi-threaded parallel processing for optimal performance.

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

# Analyze pattern difficulty
./bloco-eth stats --prefix deadbeef --suffix 123

# Run performance benchmark
./bloco-eth benchmark --attempts 10000 --pattern "abc" --threads 8
```

## Architecture Overview

### Core Components

1. **Multi-threaded Architecture**
   - `WorkerPool` (worker_pool.go): Manages multiple worker threads for parallel processing
   - `Worker` (worker.go): Individual worker thread with dedicated crypto resources
   - `StatsManager` (stats_manager.go): Thread-safe statistics aggregation across workers
   - `ProgressManager` (progress_manager.go): Real-time progress display management
   - `ThreadMetrics` (thread_metrics.go): Thread performance monitoring and efficiency calculations
   - `ThreadValidation` (thread_validation.go): Thread count validation and CPU detection

2. **Object Pooling System**
   - `CryptoPool`: Reuses cryptographic structures (private keys, ECDSA keys, big.Int)
   - `HasherPool`: Reuses Keccak256 hash instances
   - `BufferPool`: Reuses byte buffers and string builders
   - All pools implement secure cleanup of sensitive data

3. **Terminal User Interface (TUI) System**
   - `TUIManager` (tui/manager.go): Terminal capability detection and TUI coordination
   - `ProgressDisplay` (tui/progress.go): Advanced progress visualization with Bubble Tea
   - `StatsDisplay` (tui/stats.go): Real-time statistics rendering with styling
   - `Styles` (tui/styles.go): Consistent visual theming and color schemes
   - `Utils` (tui/utils.go): TUI utility functions and helpers

4. **Core Generation Logic**
   - `generateBlocoWallet()`: Main wallet generation with statistics
   - `privateToAddress()`: Optimized private key to address conversion using pools
   - `isValidBlocoAddress()`: Pattern matching with checksum validation
   - `toChecksumAddress()`: EIP-55 checksum formatting

5. **Statistics and Analysis**
   - `computeDifficulty()`: Calculates generation difficulty based on pattern length and checksum
   - `computeProbability()`: Probability calculations for success estimates
   - `Statistics` struct: Real-time progress tracking with ETA calculations
   - `BenchmarkResult`: Performance metrics collection

### File Structure
- `main.go`: CLI interface, main application logic, and core algorithms
- `worker_pool.go`: Multi-threaded worker pool implementation
- `worker.go`: Individual worker thread implementation  
- `stats_manager.go`: Thread-safe statistics aggregation
- `progress_manager.go`: Progress display management
- `thread_metrics.go`: Thread performance monitoring
- `thread_validation.go`: Thread count validation
- `tui/`: Terminal User Interface components with Bubble Tea integration
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
- **Object Pooling**: Minimizes garbage collection pressure and memory allocations
- **Thread-safe Operations**: Proper synchronization without performance penalties
- **CPU Auto-detection**: Automatically uses all available CPU cores by default

### Thread Safety
- All cryptographic operations use worker-local object pools
- Statistics are aggregated through thread-safe channels
- Graceful shutdown coordination when wallet is found
- No shared mutable state between workers

## Testing Strategy

### Test Categories
- **Unit Tests**: Individual function testing (main_test.go)
- **Integration Tests**: End-to-end wallet generation (worker_test.go)
- **Performance Tests**: Multi-threading efficiency (pool_test.go)
- **Statistics Tests**: Progress and metrics accuracy (stats_manager_test.go, progress_manager_test.go)
- **TUI Tests**: Terminal interface components (tui/*_test.go)
- **Thread Tests**: Metrics and validation testing (thread_*_test.go)

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
- EIP-55 checksum validation support
- Secure cleanup of private key data in object pools
- No shared cryptographic state between workers

## Common Patterns

### Adding New CLI Commands
1. Define command variable with `&cobra.Command{}`
2. Add to `rootCmd` in `init()` function
3. Define flags specific to the command
4. Implement validation and core logic in `Run` function

### Extending Statistics
1. Add fields to `Statistics` struct
2. Update `newStatistics()` constructor
3. Modify `update()` method for real-time calculations
4. Update display functions for new metrics

### Performance Optimization
- Always use object pools for frequently allocated structures
- Implement worker-local resources to avoid contention
- Use channels for thread-safe communication
- Profile with `go tool pprof` when making performance changes

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