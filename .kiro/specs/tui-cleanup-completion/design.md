# Design Document

## Overview

This design addresses the final cleanup and stabilization of the TUI integration in bloco-eth. The primary focus is on resolving test failures, consolidating duplicate TUI implementations, and ensuring seamless integration between the TUI components and the existing Fang CLI framework.

## Architecture

### Current State Analysis

The application currently has:
- A working `internal/tui/` directory with core TUI components
- A separate `tui/` directory with additional TUI implementations and tests
- Some test failures due to missing dependencies and undefined references
- Successful compilation of the main binary
- Working TUI functionality for basic operations

### Target Architecture

```
bloco-eth/
├── cmd/bloco-eth/main.go              # Fang integration point
├── internal/
│   ├── cli/commands.go                # CLI command handlers with TUI integration
│   ├── tui/                           # Consolidated TUI components
│   │   ├── manager.go                 # TUI capability detection and management
│   │   ├── integration.go             # Bubbletea/Bubbles/Lipgloss integration
│   │   ├── models.go                  # TUI data models
│   │   └── progress.go                # Progress display components
│   └── ...
└── tui/                               # To be consolidated into internal/tui/
```

## Components and Interfaces

### 1. TUI Manager Component

**Purpose**: Central management of TUI capabilities and lifecycle

**Key Interfaces**:
```go
type TUIManager interface {
    ShouldUseTUI() bool
    DetectCapabilities() TUICapabilities
    CreateProgressModel(stats *wallet.GenerationStats, statsManager StatsManager) tea.Model
    CreateBenchmarkModel() tea.Model
}
```

**Responsibilities**:
- Terminal capability detection
- TUI/CLI mode decision making
- Model factory methods
- Integration with Fang signal handling

### 2. Integrated TUI Models

**Purpose**: Bubbletea models that integrate with the existing worker system

**Key Components**:
- `IntegratedTUIModel`: Main TUI model with bubbletea/bubbles/lipgloss
- Progress tracking with animated progress bars
- Statistics display with formatted tables
- Benchmark results with interactive navigation

### 3. CLI Integration Layer

**Purpose**: Seamless integration between CLI commands and TUI components

**Key Functions**:
```go
func (app *Application) generateWithTUI(ctx context.Context, ...) error
func (app *Application) generateMultipleWithTUI(ctx context.Context, ...) error
func (app *Application) runBenchmarkWithTUI(ctx context.Context, ...) error
```

**Integration Points**:
- Command flag processing
- Context cancellation handling
- Error propagation and display
- Fallback to CLI mode

## Data Models

### TUI State Management

```go
type IntegratedTUIModel struct {
    // Core components
    stats            *wallet.GenerationStats
    statsCollector   *worker.StatsCollector
    criteria         wallet.GenerationCriteria
    totalWallets     int
    completedWallets int

    // Bubbletea components
    progress     progress.Model
    resultsTable table.Model

    // Lipgloss styles
    headerStyle    lipgloss.Style
    progressStyle  lipgloss.Style
    statsStyle     lipgloss.Style
    successStyle   lipgloss.Style
    errorStyle     lipgloss.Style

    // State management
    width        int
    height       int
    quitting     bool
    isComplete   bool
    showResults  bool
}
```

### Message Types

```go
type tickMsg time.Time
type WalletResultMsg struct {
    Result WalletResult
}
type QuitMsg struct{}
```

## Error Handling

### TUI Initialization Errors

1. **Terminal Detection Failures**: Fall back to CLI mode with informative message
2. **Dependency Missing**: Graceful degradation to text-based output
3. **Capability Insufficient**: Automatic fallback with user notification

### Runtime Error Handling

1. **TUI Component Failures**: Catch panics and fall back to CLI
2. **Signal Handling Conflicts**: Coordinate with Fang's signal management
3. **Context Cancellation**: Proper cleanup of TUI resources

### Error Logging Strategy

```go
// Debug mode logging
if os.Getenv("BLOCO_DEBUG") != "" {
    log.Printf("TUI: %s", debugMessage)
}

// User-facing error messages
fmt.Fprintf(os.Stderr, "TUI unavailable, using CLI mode: %v\n", err)
```

## Testing Strategy

### Test Consolidation Plan

1. **Identify Duplicate Tests**: Compare `internal/tui/` and `tui/` test files
2. **Merge Test Coverage**: Combine the best tests from both locations
3. **Remove Problematic Dependencies**: Eliminate undefined references
4. **Simplify Test Setup**: Use mocks for complex dependencies

### Test Categories

1. **Unit Tests**: Individual TUI component testing
2. **Integration Tests**: CLI-TUI interaction testing
3. **Capability Tests**: Terminal detection and fallback testing
4. **Signal Handling Tests**: Interrupt and cleanup testing

### Test Environment Setup

```go
// Test helper for TUI testing
func setupTestTUI() (*TUIManager, *MockStatsCollector) {
    manager := NewTUIManager()
    mockStats := &MockStatsCollector{}
    return manager, mockStats
}
```

## Implementation Phases

### Phase 1: Test Stabilization
- Identify and fix all test failures
- Remove undefined references
- Consolidate duplicate test files
- Ensure `go test ./...` passes completely

### Phase 2: Directory Consolidation
- Analyze both TUI directories for best components
- Move all working components to `internal/tui/`
- Remove duplicate and broken implementations
- Update import paths throughout the codebase

### Phase 3: Integration Refinement
- Ensure Fang integration works seamlessly
- Test all CLI commands with TUI enabled/disabled
- Verify signal handling coordination
- Test fallback mechanisms

### Phase 4: Documentation and Polish
- Update help text to mention TUI features
- Add environment variable documentation
- Create troubleshooting guide
- Verify all examples work correctly

## Performance Considerations

### TUI Performance Impact
- TUI components should not impact core generation performance
- Progress updates should be throttled to avoid excessive redraws
- Memory usage should be minimal for TUI state management

### Fallback Performance
- CLI fallback should have zero performance overhead
- TUI detection should be fast and cached
- Error handling should not slow down normal operations

## Security Considerations

### Terminal Security
- No sensitive data should be logged in TUI debug mode
- Terminal escape sequences should be properly sanitized
- TUI should not expose additional attack surfaces

### Signal Handling Security
- Signal handlers should not interfere with security-critical operations
- Cleanup should be thorough to prevent resource leaks
- Context cancellation should be respected for security timeouts