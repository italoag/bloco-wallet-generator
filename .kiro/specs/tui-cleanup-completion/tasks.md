# Implementation Plan

- [x] 1. Analyze and resolve current test failures
  - Run `go test ./...` to identify all failing tests and build errors
  - Document each failure with root cause analysis
  - Identify missing dependencies and undefined references in TUI components
  - Create a prioritized list of fixes needed for test stabilization
  - Fix undefined references like `Statistics`, `NewStatsModel`, and missing imports
  - _Requirements: 1.1, 1.3_

- [ ] 2. Consolidate duplicate TUI directories and implementations
  - Compare `internal/tui/` and `tui/` directories to identify overlapping functionality
  - Determine which implementation of each component is more complete and stable
  - Move all working components to `internal/tui/` as the single source of truth
  - Remove duplicate files and update all import paths throughout the codebase
  - Ensure no functionality is lost during consolidation process
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 3. Fix TUI component dependencies and missing interfaces
  - Implement missing interfaces like `StatsManager` that TUI components depend on
  - Add proper imports for all TUI dependencies (bubbletea, bubbles, lipgloss)
  - Create mock implementations for testing complex dependencies
  - Ensure all TUI models implement required bubbletea interfaces correctly
  - Add proper error handling for missing or failed dependencies
  - _Requirements: 1.3, 1.4, 5.3_

- [ ] 4. Stabilize TUI integration with CLI commands
  - Test all CLI commands with TUI enabled: `./bloco-eth --prefix a --progress`
  - Verify TUI fallback works correctly when terminal doesn't support TUI
  - Ensure `BLOCO_TUI=true/false` environment variable controls work properly
  - Test signal handling (Ctrl+C) works correctly in both TUI and CLI modes
  - Validate that all existing CLI functionality remains unchanged
  - _Requirements: 3.1, 3.2, 3.4_

- [ ] 5. Enhance Fang CLI framework integration
  - Ensure TUI components work seamlessly with Fang's command execution
  - Coordinate signal handling between Fang and TUI components
  - Test that Fang's graceful shutdown integrates properly with TUI cleanup
  - Verify that Fang's animation and progress features don't conflict with TUI
  - Add proper context cancellation support throughout TUI components
  - _Requirements: 4.1, 4.2, 4.3_

- [ ] 6. Implement comprehensive error handling and logging
  - Add proper error handling for TUI initialization failures
  - Implement graceful fallback to CLI mode when TUI is unavailable
  - Add debug logging for TUI decision-making and error diagnosis
  - Create user-friendly error messages for common TUI issues
  - Ensure application never crashes due to TUI component failures
  - _Requirements: 5.1, 5.2, 5.3_

- [ ] 7. Create consolidated test suite for TUI components
  - Merge the best tests from both TUI directories into `internal/tui/`
  - Remove problematic tests that depend on undefined components
  - Create unit tests for all TUI models and manager components
  - Add integration tests for CLI-TUI interaction scenarios
  - Implement tests for terminal capability detection and fallback logic
  - _Requirements: 1.1, 1.2, 1.5_

- [ ] 8. Update documentation and help text for TUI features
  - Update `./bloco-eth --help` to mention TUI capabilities and environment variables
  - Add examples showing both CLI and TUI usage in command help text
  - Document the `BLOCO_TUI` environment variable and its effects
  - Create troubleshooting section for common TUI issues
  - Update README.md with TUI feature documentation and screenshots
  - _Requirements: 6.1, 6.2, 6.3_

- [ ] 9. Perform comprehensive testing and validation
  - Run full test suite: `make test`, `make test-race`, `make bench`
  - Test all CLI commands in both TUI and CLI modes
  - Verify performance benchmarks show no regression from TUI integration
  - Test across different terminal emulators and terminal sizes
  - Validate signal handling and graceful shutdown in all scenarios
  - _Requirements: 1.1, 1.2, 3.5_

- [ ] 10. Final cleanup and optimization
  - Remove any remaining unused or duplicate code from TUI integration
  - Optimize TUI performance to ensure minimal impact on core functionality
  - Clean up import statements and remove unused dependencies
  - Ensure code follows Go best practices and project conventions
  - Update build system to properly handle TUI dependencies
  - _Requirements: 2.5, 5.4, 5.5_

## Testing Checklist

Before considering any task complete, ensure the following tests pass:

### Basic Compilation and Test Suite
```bash
# Core functionality must work
go build ./cmd/bloco-eth                    # Must compile successfully
go test ./...                               # All tests must pass
make test && make test-race && make bench   # Full test suite must pass
```

### TUI Functionality Tests
```bash
# TUI mode testing
./bloco-eth --prefix a --progress                           # Should show TUI or CLI progress
BLOCO_TUI=true ./bloco-eth --prefix ab --count 3 --progress # Should force TUI mode
BLOCO_TUI=false ./bloco-eth benchmark --attempts 1000      # Should force CLI mode
```

### Fallback and Error Handling Tests
```bash
# Terminal compatibility
TERM=dumb ./bloco-eth --prefix a --progress                 # Should fall back to CLI
./bloco-eth --prefix a --progress < /dev/null              # Should detect non-interactive
```

### Signal Handling Tests
```bash
# Interrupt handling (manual tests)
./bloco-eth --prefix abcd --count 10 --progress            # Press Ctrl+C during operation
./bloco-eth benchmark --attempts 50000                     # Press Ctrl+C during benchmark
```

### Integration Tests
```bash
# Existing functionality preservation
./bloco-eth --help                                         # Should show enhanced help
./bloco-eth stats --prefix abc                            # Should work in appropriate mode
./bloco-eth benchmark --attempts 1000 --detailed          # Should show results correctly
```

### Performance Validation
```bash
# No performance regression
make perf-test                                             # Should show no significant slowdown
time ./bloco-eth --prefix a --count 1                     # Should be fast without TUI
time BLOCO_TUI=true ./bloco-eth --prefix a --count 1      # TUI overhead should be minimal
```

All tests must pass and functionality must be preserved before considering the cleanup complete.