package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNewBenchmarkModel tests the creation of a new benchmark model
func TestNewBenchmarkModel(t *testing.T) {
	model := NewBenchmarkModel()

	// Check initial state
	if model.state != BenchmarkStateProgress {
		t.Errorf("Expected initial state to be BenchmarkStateProgress, got %v", model.state)
	}

	if !model.running {
		t.Error("Expected model to be in running state initially")
	}

	if model.quitting {
		t.Error("Expected model not to be quitting initially")
	}

	// Check that required components are initialized
	if model.styleManager == nil {
		t.Error("Expected styleManager to be initialized")
	}
}

// TestBenchmarkModel_Init tests the Init method
func TestBenchmarkModel_Init(t *testing.T) {
	model := NewBenchmarkModel()
	cmd := model.Init()

	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

// TestBenchmarkModel_UpdateWindowSize tests window size updates
func TestBenchmarkModel_UpdateWindowSize(t *testing.T) {
	model := NewBenchmarkModel()
	
	// Send window size message
	msg := tea.WindowSizeMsg{Width: 120, Height: 30}
	updatedModel, _ := model.Update(msg)
	
	benchmarkModel := updatedModel.(BenchmarkModel)
	if benchmarkModel.width != 120 {
		t.Errorf("Expected width to be 120, got %d", benchmarkModel.width)
	}
	
	if benchmarkModel.height != 30 {
		t.Errorf("Expected height to be 30, got %d", benchmarkModel.height)
	}
}

// TestBenchmarkModel_UpdateKeyHandling tests keyboard input handling
func TestBenchmarkModel_UpdateKeyHandling(t *testing.T) {
	model := NewBenchmarkModel()
	
	// Test quit key
	quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := model.Update(quitMsg)
	
	benchmarkModel := updatedModel.(BenchmarkModel)
	if !benchmarkModel.quitting {
		t.Error("Expected model to be quitting after 'q' key press")
	}
	
	if cmd == nil {
		t.Error("Expected quit command to be returned")
	}
}

// TestBenchmarkModel_UpdateBenchmarkMessage tests benchmark update messages
func TestBenchmarkModel_UpdateBenchmarkMessage(t *testing.T) {
	model := NewBenchmarkModel()
	
	// Create benchmark update message
	progressMsg := ProgressMsg{
		Attempts:      1000,
		Speed:         2500.0,
		Pattern:       "abc",
		Difficulty:    12.5,
		EstimatedTime: 30 * time.Second,
	}
	
	results := &BenchmarkResult{
		TotalAttempts:         5000,
		AverageSpeed:          2400.0,
		ThreadCount:           4,
		ScalabilityEfficiency: 0.85,
		TotalDuration:         2 * time.Minute,
	}
	
	updateMsg := BenchmarkUpdateMsg{
		Results:  results,
		Running:  true,
		Progress: progressMsg,
	}
	
	updatedModel, _ := model.Update(updateMsg)
	benchmarkModel := updatedModel.(BenchmarkModel)
	
	// Check that progress message was updated
	if benchmarkModel.progressMsg.Attempts != 1000 {
		t.Errorf("Expected attempts to be 1000, got %d", benchmarkModel.progressMsg.Attempts)
	}
	
	if benchmarkModel.progressMsg.Speed != 2500.0 {
		t.Errorf("Expected speed to be 2500.0, got %f", benchmarkModel.progressMsg.Speed)
	}
	
	// Check that results were updated
	if benchmarkModel.results == nil {
		t.Error("Expected results to be set")
	} else if benchmarkModel.results.TotalAttempts != 5000 {
		t.Errorf("Expected total attempts to be 5000, got %d", benchmarkModel.results.TotalAttempts)
	}
}

// TestBenchmarkModel_StateTransition tests state transitions
func TestBenchmarkModel_StateTransition(t *testing.T) {
	model := NewBenchmarkModel()
	
	// Start in progress state
	if model.state != BenchmarkStateProgress {
		t.Errorf("Expected initial state to be Progress, got %v", model.state)
	}
	
	// Send completion message
	results := &BenchmarkResult{
		TotalAttempts:         10000,
		AverageSpeed:          3000.0,
		ThreadCount:           8,
		ScalabilityEfficiency: 0.90,
		TotalDuration:         3 * time.Minute,
	}
	
	completeMsg := BenchmarkCompleteMsg{Results: results}
	updatedModel, _ := model.Update(completeMsg)
	benchmarkModel := updatedModel.(BenchmarkModel)
	
	// Should transition to transitioning state
	if benchmarkModel.state != BenchmarkStateTransitioning {
		t.Errorf("Expected state to be Transitioning after completion, got %v", benchmarkModel.state)
	}
	
	if benchmarkModel.running {
		t.Error("Expected model not to be running after completion")
	}
	
	// Simulate tick message to complete transition
	benchmarkModel.transitionTime = time.Now().Add(-time.Second) // Force transition
	tickMsg := TickMsg(time.Now())
	finalModel, _ := benchmarkModel.Update(tickMsg)
	finalBenchmarkModel := finalModel.(BenchmarkModel)
	
	if finalBenchmarkModel.state != BenchmarkStateResults {
		t.Errorf("Expected final state to be Results, got %v", finalBenchmarkModel.state)
	}
}

// TestBenchmarkModel_ViewProgressState tests progress view rendering
func TestBenchmarkModel_ViewProgressState(t *testing.T) {
	model := NewBenchmarkModel()
	model.state = BenchmarkStateProgress
	model.progressMsg = ProgressMsg{
		Attempts:      2500,
		Speed:         1800.0,
		Pattern:       "test",
		Difficulty:    8.2,
		EstimatedTime: 45 * time.Second,
	}
	
	view := model.View()
	
	// Check that progress view contains expected elements
	if !strings.Contains(view, "Benchmark Running") {
		t.Error("Expected progress view to contain 'Benchmark Running'")
	}
	
	if !strings.Contains(view, "Current Performance") {
		t.Error("Expected progress view to contain 'Current Performance'")
	}
	
	if !strings.Contains(view, "test") {
		t.Error("Expected progress view to contain pattern 'test'")
	}
	
	if !strings.Contains(view, "Press q to quit") {
		t.Error("Expected progress view to contain help text")
	}
}

// TestBenchmarkModel_ViewResultsState tests results view rendering
func TestBenchmarkModel_ViewResultsState(t *testing.T) {
	model := NewBenchmarkModel()
	model.state = BenchmarkStateResults
	model.results = &BenchmarkResult{
		TotalAttempts:         15000,
		AverageSpeed:          2200.0,
		ThreadCount:           6,
		ScalabilityEfficiency: 0.75,
		TotalDuration:         4 * time.Minute,
		MinSpeed:              1800.0,
		MaxSpeed:              2600.0,
		SingleThreadSpeed:     400.0,
	}
	
	// Set up table rows
	model.table.SetRows(model.generateResultsRows())
	
	view := model.View()
	
	// Check that results view contains expected elements
	if !strings.Contains(view, "Benchmark Results") {
		t.Error("Expected results view to contain 'Benchmark Results'")
	}
	
	if !strings.Contains(view, "Summary") {
		t.Error("Expected results view to contain 'Summary'")
	}
	
	if !strings.Contains(view, "Navigate") {
		t.Error("Expected results view to contain navigation help")
	}
}

// TestGenerateResultsRows tests results table generation
func TestGenerateResultsRows(t *testing.T) {
	model := NewBenchmarkModel()
	model.results = &BenchmarkResult{
		TotalAttempts:         8000,
		AverageSpeed:          1500.0,
		ThreadCount:           4,
		ScalabilityEfficiency: 0.80,
		TotalDuration:         2 * time.Minute,
		MinSpeed:              1200.0,
		MaxSpeed:              1800.0,
		SingleThreadSpeed:     375.0,
	}
	
	rows := model.generateResultsRows()
	
	// Check that we have the expected number of rows
	expectedRows := 9 // Based on the implementation
	if len(rows) != expectedRows {
		t.Errorf("Expected %d rows, got %d", expectedRows, len(rows))
	}
	
	// Check that rows contain expected data
	found := false
	for _, row := range rows {
		if len(row) >= 2 && row[0] == "Total Attempts" {
			found = true
			if !strings.Contains(row[1], "8000") && !strings.Contains(row[1], "8K") && !strings.Contains(row[1], "8 000") {
				t.Errorf("Expected total attempts row to contain 8000, 8K, or 8 000, got %s", row[1])
			}
		}
	}
	
	if !found {
		t.Error("Expected to find 'Total Attempts' row")
	}
}

// TestBenchmarkModel_EmptyResults tests handling of nil results
func TestBenchmarkModel_EmptyResults(t *testing.T) {
	model := NewBenchmarkModel()
	model.results = nil
	
	rows := model.generateResultsRows()
	if len(rows) != 0 {
		t.Errorf("Expected 0 rows for nil results, got %d", len(rows))
	}
	
	// Test view with nil results
	model.state = BenchmarkStateResults
	view := model.View()
	
	// Should not panic and should render basic structure
	if view == "" {
		t.Error("Expected non-empty view even with nil results")
	}
}

// TestFormatSpeed tests the speed formatting function
func TestFormatSpeed(t *testing.T) {
	tests := map[float64]string{
		0:        "0",
		500:      "500",
		1000:     "1.0K",
		1500:     "1.5K",
		1000000:  "1.0M",
		1500000:  "1.5M",
		2500000:  "2.5M",
	}
	
	for input, expected := range tests {
		result := formatSpeed(input)
		if result != expected {
			t.Errorf("formatSpeed(%f): expected %s, got %s", input, expected, result)
		}
	}
}

// BenchmarkBenchmarkModel_View benchmarks the View method
func BenchmarkBenchmarkModel_View(b *testing.B) {
	model := NewBenchmarkModel()
	model.results = &BenchmarkResult{
		TotalAttempts:         10000,
		AverageSpeed:          2000.0,
		ThreadCount:           4,
		ScalabilityEfficiency: 0.85,
		TotalDuration:         2 * time.Minute,
	}
	model.table.SetRows(model.generateResultsRows())
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

// BenchmarkGenerateResultsRows benchmarks the table generation
func BenchmarkGenerateResultsRows(b *testing.B) {
	model := NewBenchmarkModel()
	model.results = &BenchmarkResult{
		TotalAttempts:         50000,
		AverageSpeed:          3500.0,
		ThreadCount:           8,
		ScalabilityEfficiency: 0.90,
		TotalDuration:         5 * time.Minute,
		MinSpeed:              3000.0,
		MaxSpeed:              4000.0,
		SingleThreadSpeed:     450.0,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.generateResultsRows()
	}
}