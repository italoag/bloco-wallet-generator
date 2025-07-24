# Design Document

## Overview

This design document outlines the integration of Charmbracelet Bubbletea TUI components into the bloco-eth application. The integration will enhance user experience by replacing text-based output with interactive, styled interfaces while maintaining full backward compatibility with existing CLI functionality.

The design follows a modular approach where TUI components are separate from core business logic, allowing for easy maintenance and graceful degradation when TUI features are not available.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Layer (Cobra)                        │
├─────────────────────────────────────────────────────────────┤
│                  TUI Detection Layer                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  Benchmark TUI  │  │ Statistics TUI  │  │ Progress TUI │ │
│  │   Components    │  │   Components    │  │  Components  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│              Bubbletea Framework Layer                      │
├─────────────────────────────────────────────────────────────┤
│                Core Business Logic                          │
│  (Wallet Generation, Statistics, Benchmarking)              │
└─────────────────────────────────────────────────────────────┘
```

### Component Architecture

The TUI integration will be organized into the following main components:

1. **TUI Manager**: Detects terminal capabilities and manages TUI/text mode switching
2. **Progress Components**: Animated progress bars and real-time statistics display
3. **Table Components**: Formatted data presentation for benchmarks and statistics
4. **Style Manager**: Consistent styling and theming across all TUI components
5. **Event Handlers**: Keyboard input and signal handling for interactive features

## Components and Interfaces

### TUI Manager

```go
type TUIManager struct {
    enabled       bool
    terminalWidth int
    terminalHeight int
    colorSupport  bool
}

type TUICapabilities struct {
    SupportsColor    bool
    SupportsUnicode  bool
    TerminalWidth    int
    TerminalHeight   int
    SupportsResize   bool
}

func NewTUIManager() *TUIManager
func (tm *TUIManager) DetectCapabilities() TUICapabilities
func (tm *TUIManager) ShouldUseTUI() bool
func (tm *TUIManager) CreateProgressModel() tea.Model
func (tm *TUIManager) CreateBenchmarkModel() tea.Model
func (tm *TUIManager) CreateStatsModel() tea.Model
```

### Progress TUI Component

```go
type ProgressModel struct {
    progress     progress.Model
    stats        *Statistics
    statsManager *StatsManager
    width        int
    height       int
    quitting     bool
}

type ProgressMsg struct {
    Attempts      int64
    Speed         float64
    Probability   float64
    EstimatedTime time.Duration
}

func NewProgressModel(stats *Statistics, statsManager *StatsManager) ProgressModel
func (m ProgressModel) Init() tea.Cmd
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m ProgressModel) View() string
```

### Benchmark TUI Component

```go
type BenchmarkModel struct {
    table        table.Model
    progress     progress.Model
    results      *BenchmarkResult
    running      bool
    width        int
    height       int
    quitting     bool
}

type BenchmarkUpdateMsg struct {
    Results *BenchmarkResult
    Running bool
}

func NewBenchmarkModel() BenchmarkModel
func (m BenchmarkModel) Init() tea.Cmd
func (m BenchmarkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m BenchmarkModel) View() string
```

### Statistics TUI Component

```go
type StatsModel struct {
    table    table.Model
    stats    *Statistics
    width    int
    height   int
    quitting bool
}

type StatsData struct {
    Pattern       string
    Difficulty    string
    Probability50 string
    IsChecksum    bool
}

func NewStatsModel(stats *Statistics) StatsModel
func (m StatsModel) Init() tea.Cmd
func (m StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m StatsModel) View() string
```

### Style Manager

```go
type StyleManager struct {
    baseStyle      lipgloss.Style
    headerStyle    lipgloss.Style
    tableStyle     lipgloss.Style
    progressStyle  lipgloss.Style
    successStyle   lipgloss.Style
    errorStyle     lipgloss.Style
    helpStyle      lipgloss.Style
}

func NewStyleManager() *StyleManager
func (sm *StyleManager) GetProgressStyle() lipgloss.Style
func (sm *StyleManager) GetTableStyle() lipgloss.Style
func (sm *StyleManager) GetHeaderStyle() lipgloss.Style
func (sm *StyleManager) GetSuccessStyle() lipgloss.Style
func (sm *StyleManager) GetErrorStyle() lipgloss.Style
func (sm *StyleManager) AdaptToTerminal(capabilities TUICapabilities)
```

## Data Models

### Enhanced Statistics Structure

```go
type TUIStatistics struct {
    *Statistics
    DisplayMode    string // "progress", "table", "compact"
    LastUIUpdate   time.Time
    UIUpdateRate   time.Duration
    FormattedData  map[string]string
}

func (ts *TUIStatistics) FormatForDisplay() map[string]string
func (ts *TUIStatistics) ShouldUpdate() bool
func (ts *TUIStatistics) GetTableRows() []table.Row
```

### Benchmark Results for TUI

```go
type TUIBenchmarkResult struct {
    *BenchmarkResult
    TableData     []table.Row
    ProgressData  ProgressMsg
    FormattedTime string
    FormattedSpeed string
}

func (tbr *TUIBenchmarkResult) ToTableRows() []table.Row
func (tbr *TUIBenchmarkResult) GetProgressData() ProgressMsg
func (tbr *TUIBenchmarkResult) FormatMetrics() map[string]string
```

## Error Handling

### TUI Error Management

1. **Graceful Degradation**: If TUI initialization fails, automatically fall back to text mode
2. **Terminal Compatibility**: Detect terminal capabilities and disable unsupported features
3. **Resize Handling**: Handle terminal resize events gracefully without crashing
4. **Signal Handling**: Properly handle Ctrl+C and other signals in TUI mode
5. **Error Display**: Show errors in styled format when in TUI mode, plain text otherwise

### Error Recovery Strategies

```go
type TUIErrorHandler struct {
    fallbackMode bool
    lastError    error
}

func (teh *TUIErrorHandler) HandleTUIError(err error) bool
func (teh *TUIErrorHandler) ShouldFallback() bool
func (teh *TUIErrorHandler) RecoverFromError() error
```

## Testing Strategy

### Unit Testing

1. **Component Testing**: Test each TUI component independently with mock data
2. **Style Testing**: Verify styling works correctly across different terminal capabilities
3. **Event Testing**: Test keyboard and signal handling in TUI components
4. **Fallback Testing**: Ensure graceful degradation when TUI features are unavailable

### Integration Testing

1. **End-to-End TUI**: Test complete workflows with TUI enabled
2. **Terminal Compatibility**: Test across different terminal emulators and capabilities
3. **Performance Testing**: Ensure TUI doesn't significantly impact performance
4. **Regression Testing**: Verify existing CLI functionality remains intact

### Test Structure

```go
func TestProgressModel_Update(t *testing.T)
func TestBenchmarkModel_TableGeneration(t *testing.T)
func TestStatsModel_DataFormatting(t *testing.T)
func TestTUIManager_CapabilityDetection(t *testing.T)
func TestStyleManager_TerminalAdaptation(t *testing.T)
func TestTUIFallback_GracefulDegradation(t *testing.T)
```

## Implementation Details

### Dependencies

The implementation will require adding the following Bubbletea ecosystem dependencies:

```go
require (
    github.com/charmbracelet/bubbletea v1.2.4
    github.com/charmbracelet/bubbles v0.20.0
    github.com/charmbracelet/lipgloss v1.0.0
)
```

### Integration Points

1. **Main Function**: Modify main function to detect TUI capabilities and choose appropriate mode
2. **Command Handlers**: Update benchmark, stats, and progress handlers to use TUI when available
3. **Signal Handling**: Integrate TUI signal handling with existing Fang signal management
4. **Progress Updates**: Replace current progress display with TUI progress components
5. **Output Formatting**: Route output through TUI components when enabled

### File Structure

```
├── tui/
│   ├── manager.go          # TUI capability detection and management
│   ├── progress.go         # Progress bar TUI component
│   ├── benchmark.go        # Benchmark results TUI component
│   ├── stats.go            # Statistics display TUI component
│   ├── styles.go           # Styling and theming
│   └── utils.go            # TUI utility functions
├── main.go                 # Updated with TUI integration
└── *_test.go               # Enhanced tests for TUI components
```

### Performance Considerations

1. **Update Frequency**: Limit TUI updates to reasonable intervals (e.g., 100ms) to avoid performance impact
2. **Memory Usage**: Use object pooling for TUI components to minimize allocations
3. **CPU Usage**: Ensure TUI rendering doesn't significantly impact wallet generation performance
4. **Terminal I/O**: Optimize terminal output to minimize flickering and improve responsiveness

### Backward Compatibility

1. **CLI Flags**: All existing CLI flags and functionality remain unchanged
2. **Output Format**: Provide option to disable TUI and use original text output
3. **Environment Variables**: Support environment variables to control TUI behavior
4. **Fallback Mode**: Automatically detect when TUI is not suitable and fall back to text mode

The design ensures that the TUI integration enhances the user experience while maintaining the robustness and performance characteristics of the existing application.