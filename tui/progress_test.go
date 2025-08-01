package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// MockStatsManager implements StatsManager for testing
type MockStatsManager struct {
	metrics   ThreadMetrics
	peakSpeed float64
}

func (m *MockStatsManager) GetMetrics() ThreadMetrics {
	return m.metrics
}

func (m *MockStatsManager) GetPeakSpeed() float64 {
	return m.peakSpeed
}

func TestNewProgressModel(t *testing.T) {
	stats := &Statistics{
		Difficulty:      1000000,
		Probability50:   693147,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         "abc",
		IsChecksum:      false,
	}

	statsManager := &MockStatsManager{
		metrics: ThreadMetrics{
			EfficiencyRatio: 0.85,
			TotalSpeed:      5000,
			ThreadCount:     4,
		},
		peakSpeed: 6000,
	}

	model := NewProgressModel(stats, statsManager)

	// Verify model initialization
	if model.stats != stats {
		t.Error("Expected stats to be set correctly")
	}

	if model.statsManager == nil {
		t.Error("Expected statsManager to be set correctly")
	}

	if model.quitting {
		t.Error("Expected quitting to be false initially")
	}

	if model.width <= 0 || model.height <= 0 {
		t.Error("Expected positive terminal dimensions")
	}

	// Verify progress bar is initialized (we can't compare directly due to lipgloss.Style)
	// Just check that it's not nil by checking if it has a reasonable width
	if model.progress.Width < 0 {
		t.Error("Expected progress bar to be initialized")
	}

	// Verify style manager is initialized
	if model.styleManager == nil {
		t.Error("Expected style manager to be initialized")
	}
}

func TestProgressModel_Init(t *testing.T) {
	stats := &Statistics{
		Pattern: "test",
	}
	model := NewProgressModel(stats, nil)

	cmd := model.Init()
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}

	// Test that Init returns a batch command
	// We can't easily test the exact commands without executing them,
	// but we can verify a command is returned
}

func TestProgressModel_Update_WindowSizeMsg(t *testing.T) {
	stats := &Statistics{Pattern: "test"}
	model := NewProgressModel(stats, nil)

	// Test window resize
	newModel, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updatedModel := newModel.(ProgressModel)

	if updatedModel.width != 120 {
		t.Errorf("Expected width to be 120, got %d", updatedModel.width)
	}

	if updatedModel.height != 30 {
		t.Errorf("Expected height to be 30, got %d", updatedModel.height)
	}

	if cmd != nil {
		t.Error("Expected no command from window resize")
	}
}

func TestProgressModel_Update_KeyMsg(t *testing.T) {
	stats := &Statistics{Pattern: "test"}
	model := NewProgressModel(stats, nil)

	testCases := []struct {
		name       string
		key        string
		shouldQuit bool
	}{
		{"Ctrl+C", "ctrl+c", true},
		{"Q key", "q", true},
		{"Escape", "esc", true},
		{"Other key", "a", true}, // Any key should quit in the new implementation
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset model state
			model.quitting = false

			var keyMsg tea.KeyMsg
			if tc.key == "ctrl+c" {
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else if tc.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				keyMsg = tea.KeyMsg{
					Type:  tea.KeyRunes,
					Runes: []rune(tc.key),
				}
			}

			newModel, cmd := model.Update(keyMsg)
			updatedModel := newModel.(ProgressModel)

			if updatedModel.quitting != tc.shouldQuit {
				t.Errorf("Expected quitting to be %v for key %s, got %v", tc.shouldQuit, tc.key, updatedModel.quitting)
			}

			if tc.shouldQuit && cmd == nil {
				t.Errorf("Expected command for quit key %s", tc.key)
			}
		})
	}
}

func TestProgressModel_Update_TickMsg(t *testing.T) {
	stats := &Statistics{Pattern: "test"}
	model := NewProgressModel(stats, nil)

	// Test normal tick
	tickMsg := TickMsg(time.Now())
	newModel, cmd := model.Update(tickMsg)
	updatedModel := newModel.(ProgressModel)

	if updatedModel.quitting {
		t.Error("Expected model not to be quitting after normal tick")
	}

	if cmd == nil {
		t.Error("Expected tick command to be returned")
	}

	// Test tick when quitting
	model.quitting = true
	newModel, cmd = model.Update(tickMsg)
	updatedModel = newModel.(ProgressModel)

	if cmd == nil {
		t.Error("Expected quit command when quitting")
	}
}

func TestProgressModel_Update_ProgressMsg(t *testing.T) {
	stats := &Statistics{
		Pattern:         "test",
		CurrentAttempts: 1000,
		Speed:           500,
		Probability:     25.5,
	}
	model := NewProgressModel(stats, nil)

	progressMsg := ProgressMsg{
		Attempts:      2000,
		Speed:         1000,
		Probability:   45.2,
		EstimatedTime: time.Minute * 5,
		Difficulty:    1000000,
		Pattern:       "updated",
	}

	newModel, cmd := model.Update(progressMsg)
	updatedModel := newModel.(ProgressModel)

	// Verify statistics were updated
	if updatedModel.stats.CurrentAttempts != 2000 {
		t.Errorf("Expected attempts to be 2000, got %d", updatedModel.stats.CurrentAttempts)
	}

	if updatedModel.stats.Speed != 1000 {
		t.Errorf("Expected speed to be 1000, got %f", updatedModel.stats.Speed)
	}

	if updatedModel.stats.Probability != 45.2 {
		t.Errorf("Expected probability to be 45.2, got %f", updatedModel.stats.Probability)
	}

	if updatedModel.stats.EstimatedTime != time.Minute*5 {
		t.Errorf("Expected estimated time to be 5 minutes, got %v", updatedModel.stats.EstimatedTime)
	}

	if updatedModel.stats.Difficulty != 1000000 {
		t.Errorf("Expected difficulty to be 1000000, got %f", updatedModel.stats.Difficulty)
	}

	if updatedModel.stats.Pattern != "updated" {
		t.Errorf("Expected pattern to be 'updated', got %s", updatedModel.stats.Pattern)
	}

	if cmd != nil {
		t.Error("Expected no command from progress update")
	}
}

func TestProgressModel_Update_QuitMsg(t *testing.T) {
	stats := &Statistics{Pattern: "test"}
	model := NewProgressModel(stats, nil)

	quitMsg := QuitMsg{}
	newModel, cmd := model.Update(quitMsg)
	updatedModel := newModel.(ProgressModel)

	if !updatedModel.quitting {
		t.Error("Expected model to be quitting after QuitMsg")
	}

	if cmd == nil {
		t.Error("Expected command from QuitMsg")
	}
}

func TestProgressModel_View(t *testing.T) {
	stats := &Statistics{
		Difficulty:      1000000,
		Probability50:   693147,
		CurrentAttempts: 50000,
		Speed:           1500,
		Probability:     35.5,
		EstimatedTime:   time.Minute * 10,
		Pattern:         "abc123",
		IsChecksum:      true,
	}

	statsManager := &MockStatsManager{
		metrics: ThreadMetrics{
			EfficiencyRatio: 0.85,
			TotalSpeed:      5000,
			ThreadCount:     4,
		},
		peakSpeed: 6000,
	}

	model := NewProgressModel(stats, statsManager)

	view := model.View()

	// Check that key information is present in the view
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Check for pattern information
	if !contains(view, "abc123") {
		t.Error("Expected pattern to be displayed in view")
	}

	// Check for checksum indicator
	if !contains(view, "checksum") {
		t.Error("Expected checksum indicator in view")
	}

	// Check for statistics
	if !contains(view, "50 000") { // Formatted attempts
		t.Error("Expected formatted attempts in view")
	}

	if !contains(view, "1500") { // Speed
		t.Error("Expected speed in view")
	}

	if !contains(view, "35.5") { // Probability
		t.Error("Expected probability in view")
	}

	// Check for thread information
	if !contains(view, "4 threads") {
		t.Error("Expected thread count in view")
	}

	if !contains(view, "85.0%") { // Efficiency
		t.Error("Expected efficiency in view")
	}

	// Check for help text
	if !contains(view, "Press") && !contains(view, "quit") {
		t.Error("Expected help text in view")
	}
}

func TestProgressModel_View_Quitting(t *testing.T) {
	stats := &Statistics{Pattern: "test"}
	model := NewProgressModel(stats, nil)
	model.quitting = true

	view := model.View()

	// When quitting, the view should still show the normal content
	// since the program will exit via tea.Quit command
	if !contains(view, "Bloco Wallet Generation") {
		t.Error("Expected title in view even when quitting")
	}
}

func TestProgressModel_View_NoStats(t *testing.T) {
	model := NewProgressModel(nil, nil)

	view := model.View()

	if !contains(view, "No statistics") {
		t.Error("Expected error message when no statistics available")
	}
}

func TestUpdateProgress(t *testing.T) {
	cmd := UpdateProgress(1000, 500.5, 25.0, time.Minute*5, 1000000, "test")

	if cmd == nil {
		t.Error("Expected UpdateProgress to return a command")
	}

	// Execute the command to get the message
	msg := cmd()
	progressMsg, ok := msg.(ProgressMsg)
	if !ok {
		t.Error("Expected ProgressMsg from UpdateProgress command")
	}

	if progressMsg.Attempts != 1000 {
		t.Errorf("Expected attempts to be 1000, got %d", progressMsg.Attempts)
	}

	if progressMsg.Speed != 500.5 {
		t.Errorf("Expected speed to be 500.5, got %f", progressMsg.Speed)
	}

	if progressMsg.Probability != 25.0 {
		t.Errorf("Expected probability to be 25.0, got %f", progressMsg.Probability)
	}

	if progressMsg.EstimatedTime != time.Minute*5 {
		t.Errorf("Expected estimated time to be 5 minutes, got %v", progressMsg.EstimatedTime)
	}

	if progressMsg.Difficulty != 1000000 {
		t.Errorf("Expected difficulty to be 1000000, got %f", progressMsg.Difficulty)
	}

	if progressMsg.Pattern != "test" {
		t.Errorf("Expected pattern to be 'test', got %s", progressMsg.Pattern)
	}
}

func TestSendQuit(t *testing.T) {
	cmd := SendQuit()

	if cmd == nil {
		t.Error("Expected SendQuit to return a command")
	}

	// Execute the command to get the message
	msg := cmd()
	_, ok := msg.(QuitMsg)
	if !ok {
		t.Error("Expected QuitMsg from SendQuit command")
	}
}

func TestProgressModel_StateTransitions(t *testing.T) {
	stats := &Statistics{
		Pattern:         "test",
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
	}
	model := NewProgressModel(stats, nil)

	// Test initial state
	if model.quitting {
		t.Error("Expected initial state to not be quitting")
	}

	// Test transition to quitting state
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("q"),
	}
	newModel, _ := model.Update(keyMsg)
	updatedModel := newModel.(ProgressModel)

	if !updatedModel.quitting {
		t.Error("Expected state to transition to quitting")
	}

	// Test that quitting state persists
	tickMsg := TickMsg(time.Now())
	newModel, cmd := updatedModel.Update(tickMsg)
	finalModel := newModel.(ProgressModel)

	if !finalModel.quitting {
		t.Error("Expected quitting state to persist")
	}

	if cmd == nil {
		t.Error("Expected quit command when in quitting state")
	}
}

func TestProgressModel_DisplayFormatting(t *testing.T) {
	stats := &Statistics{
		Difficulty:      1234567,
		Probability50:   987654,
		CurrentAttempts: 123456,
		Speed:           1500.75,
		Probability:     67.89,
		EstimatedTime:   time.Hour*2 + time.Minute*30,
		Pattern:         "abc",
		IsChecksum:      false,
	}

	model := NewProgressModel(stats, nil)
	view := model.View()

	// Test number formatting
	if !contains(view, "1 234 567") {
		t.Error("Expected formatted difficulty in view")
	}

	if !contains(view, "123 456") {
		t.Error("Expected formatted attempts in view")
	}

	if !contains(view, "987 654") {
		t.Error("Expected formatted probability50 in view")
	}

	// Test percentage formatting
	if !contains(view, "67.89%") {
		t.Error("Expected formatted percentage in view")
	}

	// Test speed formatting
	if !contains(view, "1501") { // Rounded speed
		t.Error("Expected formatted speed in view")
	}

	// Test duration formatting
	if !contains(view, "2.5h") {
		t.Error("Expected formatted duration in view")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
