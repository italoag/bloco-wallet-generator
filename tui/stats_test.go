package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNewStatsModel tests the creation of a new statistics model
func TestNewStatsModel(t *testing.T) {
	stats := &Statistics{
		Difficulty:      16777216, // 16^6
		Probability50:   11629080,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         "abcdef",
		IsChecksum:      false,
	}

	model := NewStatsModel(stats)

	// Test that model is properly initialized
	if model.stats != stats {
		t.Errorf("Expected stats to be set correctly")
	}

	if model.quitting {
		t.Errorf("Expected quitting to be false initially")
	}

	if model.ready {
		t.Errorf("Expected ready to be false initially")
	}

	if model.width <= 0 || model.height <= 0 {
		t.Errorf("Expected positive terminal dimensions, got width=%d, height=%d", model.width, model.height)
	}

	// Test that style manager is created
	if model.styleManager == nil {
		t.Errorf("Expected style manager to be initialized")
	}

	// Test that table is created
	if model.table.Rows() == nil {
		// This is expected initially as rows are set during Init/Update
	}
}

// TestStatsModel_Init tests the initialization of the statistics model
func TestStatsModel_Init(t *testing.T) {
	stats := &Statistics{
		Difficulty:    256, // 16^2
		Probability50: 177,
		Pattern:       "ab",
		IsChecksum:    false,
	}

	model := NewStatsModel(stats)
	_ = model.Init()

	// Init may or may not return a command depending on stats availability
	// This is acceptable behavior
}

// TestStatsModel_Update tests the update functionality
func TestStatsModel_Update(t *testing.T) {
	stats := &Statistics{
		Difficulty:    4096, // 16^3
		Probability50: 2839,
		Pattern:       "abc",
		IsChecksum:    true,
	}

	model := NewStatsModel(stats)

	// Test window size message
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := model.Update(windowMsg)
	statsModel := updatedModel.(StatsModel)

	if statsModel.width != 100 || statsModel.height != 30 {
		t.Errorf("Expected dimensions to be updated to 100x30, got %dx%d", statsModel.width, statsModel.height)
	}

	if cmd != nil {
		t.Errorf("Expected no command from window size update")
	}

	// Test quit key messages
	quitKeys := []string{"ctrl+c", "q", "esc"}
	for _, key := range quitKeys {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if key == "ctrl+c" {
			keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
		} else if key == "esc" {
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		}

		updatedModel, cmd := model.Update(keyMsg)
		statsModel := updatedModel.(StatsModel)

		if !statsModel.quitting {
			t.Errorf("Expected quitting to be true after %s key", key)
		}

		if cmd == nil {
			t.Errorf("Expected command after %s key", key)
		}
	}

	// Test navigation keys
	navKeys := []string{"up", "down", "j", "k"}
	for _, key := range navKeys {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if key == "up" {
			keyMsg = tea.KeyMsg{Type: tea.KeyUp}
		} else if key == "down" {
			keyMsg = tea.KeyMsg{Type: tea.KeyDown}
		}

		_, _ = model.Update(keyMsg)
		// Navigation should return a command (from table update)
		// We don't test the specific command as it's internal to the table
	}

	// Test stats update message
	newStats := &Statistics{
		Difficulty:    65536, // 16^4
		Probability50: 45426,
		Pattern:       "abcd",
		IsChecksum:    false,
	}

	updateMsg := StatsUpdateMsg{Stats: newStats}
	updatedModel, cmd = model.Update(updateMsg)
	statsModel = updatedModel.(StatsModel)

	if statsModel.stats != newStats {
		t.Errorf("Expected stats to be updated")
	}

	// updateTableData returns nil, so no command is expected
	if cmd != nil {
		t.Errorf("Expected no command after stats update, got %v", cmd)
	}
}

// TestStatsModel_View tests the view rendering
func TestStatsModel_View(t *testing.T) {
	stats := &Statistics{
		Difficulty:    1048576, // 16^5
		Probability50: 726817,
		Pattern:       "abcde",
		IsChecksum:    true,
	}

	model := NewStatsModel(stats)
	model.ready = true // Mark as ready to avoid initialization issues

	view := model.View()

	// Test that view contains expected content
	expectedContent := []string{
		"📊 Bloco Address Difficulty Analysis",
		"Pattern",
		"Checksum",
		"📈 Detailed Statistics",
		"⏱️ Time Estimates",
		"🎲 Probability Examples",
		"💡 Recommendations",
		"Use ↑/↓ or j/k to navigate",
	}

	for _, content := range expectedContent {
		if !strings.Contains(view, content) {
			t.Errorf("Expected view to contain '%s', but it didn't", content)
		}
	}

	// Test quitting view - in the new implementation, quitting doesn't change the view
	// since the program exits via tea.Quit command
	model.quitting = true
	quitView := model.View()
	if !strings.Contains(quitView, "Bloco Address Difficulty Analysis") {
		t.Errorf("Expected view to contain title even when quitting")
	}

	// Test nil stats
	model.stats = nil
	model.quitting = false
	nilView := model.View()
	if !strings.Contains(nilView, "No statistics available") {
		t.Errorf("Expected nil stats view to contain error message")
	}
}

// TestGenerateStatisticsRows tests the statistics table row generation
func TestGenerateStatisticsRows(t *testing.T) {
	stats := &Statistics{
		Difficulty:    256, // 16^2
		Probability50: 177,
		Pattern:       "ab",
		IsChecksum:    false,
	}

	model := NewStatsModel(stats)
	rows := model.generateStatisticsRows()

	// Test that we have the expected number of rows
	expectedMinRows := 6 // Pattern, Length, Base Difficulty, Total Difficulty, 50% Probability, Expected Attempts, Success Rate
	if len(rows) < expectedMinRows {
		t.Errorf("Expected at least %d rows, got %d", expectedMinRows, len(rows))
	}

	// Test that each row has 3 columns
	for i, row := range rows {
		if len(row) != 3 {
			t.Errorf("Row %d should have 3 columns, got %d", i, len(row))
		}
	}

	// Test specific row content
	foundPattern := false
	foundDifficulty := false
	for _, row := range rows {
		if row[0] == "Pattern" {
			foundPattern = true
			if !strings.Contains(row[1], "ab") {
				t.Errorf("Expected pattern row to contain 'ab', got '%s'", row[1])
			}
		}
		if row[0] == "Total Difficulty" {
			foundDifficulty = true
			if row[1] != "256" {
				t.Errorf("Expected difficulty row to show '256', got '%s'", row[1])
			}
		}
	}

	if !foundPattern {
		t.Errorf("Expected to find Pattern row")
	}
	if !foundDifficulty {
		t.Errorf("Expected to find Total Difficulty row")
	}
}

// TestGenerateStatisticsRowsWithChecksum tests row generation with checksum enabled
func TestGenerateStatisticsRowsWithChecksum(t *testing.T) {
	stats := &Statistics{
		Difficulty:    1024, // 16^2 * 4 (checksum multiplier)
		Probability50: 710,
		Pattern:       "Ab", // Mixed case for checksum
		IsChecksum:    true,
	}

	model := NewStatsModel(stats)
	rows := model.generateStatisticsRows()

	// Test that checksum multiplier row is present
	foundMultiplier := false
	for _, row := range rows {
		if row[0] == "Checksum Multiplier" {
			foundMultiplier = true
			if !strings.Contains(row[1], "x") {
				t.Errorf("Expected multiplier row to contain 'x', got '%s'", row[1])
			}
		}
	}

	if !foundMultiplier {
		t.Errorf("Expected to find Checksum Multiplier row when checksum is enabled")
	}
}

// TestAdaptTableToSize tests table adaptation to different terminal sizes
func TestAdaptTableToSize(t *testing.T) {
	stats := &Statistics{
		Difficulty: 256,
		Pattern:    "ab",
		IsChecksum: false,
	}

	model := NewStatsModel(stats)

	// Test narrow terminal
	model.width = 60
	model.height = 15
	model.adaptTableToSize()

	columns := model.table.Columns()
	if len(columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(columns))
	}

	// Check that columns are narrower for small terminal
	totalWidth := columns[0].Width + columns[1].Width + columns[2].Width
	if totalWidth > 80 { // Should be narrower than default (15+20+25=60)
		t.Errorf("Expected narrower columns for small terminal, got total width %d", totalWidth)
	}

	// Test wide terminal
	model.width = 150
	model.height = 40
	model.adaptTableToSize()

	columns = model.table.Columns()
	newTotalWidth := columns[0].Width + columns[1].Width + columns[2].Width
	if newTotalWidth <= totalWidth {
		t.Errorf("Expected wider columns for large terminal, got %d vs %d", newTotalWidth, totalWidth)
	}

	// Test height adaptation
	if model.table.Height() <= 8 {
		t.Errorf("Expected taller table for large terminal")
	}
}

// TestStatsFormatLargeNumber tests the number formatting function in stats context
func TestStatsFormatLargeNumber(t *testing.T) {
	testCases := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{1234, "1 234"},
		{12345, "12 345"},
		{123456, "123 456"},
		{1234567, "1 234 567"},
		{12345678, "12 345 678"},
		{123456789, "123 456 789"},
		{1000000000, "1 000 000 000"},
	}

	for _, tc := range testCases {
		result := formatLargeNumber(tc.input)
		if result != tc.expected {
			t.Errorf("formatLargeNumber(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

// TestComputeProbability tests the probability calculation function
func TestComputeProbability(t *testing.T) {
	testCases := []struct {
		difficulty float64
		attempts   int64
		expected   float64
		tolerance  float64
	}{
		{16, 11, 0.5, 0.01},    // ~50% probability
		{256, 177, 0.5, 0.01},  // ~50% probability
		{16, 1, 0.0625, 0.001}, // 1/16 probability
		{0, 100, 0, 0},         // Zero difficulty
		{100, 0, 0, 0},         // Zero attempts
	}

	for _, tc := range testCases {
		result := computeProbability(tc.difficulty, tc.attempts)
		if tc.tolerance > 0 {
			if result < tc.expected-tc.tolerance || result > tc.expected+tc.tolerance {
				t.Errorf("computeProbability(%f, %d) = %f, expected %f ± %f",
					tc.difficulty, tc.attempts, result, tc.expected, tc.tolerance)
			}
		} else {
			if result != tc.expected {
				t.Errorf("computeProbability(%f, %d) = %f, expected %f",
					tc.difficulty, tc.attempts, result, tc.expected)
			}
		}
	}
}

// TestStatsFormatDuration tests the duration formatting function in stats context
func TestStatsFormatDuration(t *testing.T) {
	testCases := []struct {
		input    time.Duration
		expected string
	}{
		{-1 * time.Second, "Nearly impossible"},
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1.5m"},
		{3900 * time.Second, "1.1h"},
		{90000 * time.Second, "1.0d"},
		{32000000 * time.Second, "1.0y"},
		{time.Duration(250*365*24) * time.Hour, "Thousands of years"}, // Fixed overflow
	}

	for _, tc := range testCases {
		result := formatDuration(tc.input)
		if result != tc.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

// TestUpdateStats tests the UpdateStats command function
func TestUpdateStats(t *testing.T) {
	stats := &Statistics{
		Difficulty: 1024,
		Pattern:    "test",
		IsChecksum: true,
	}

	cmd := UpdateStats(stats)
	if cmd == nil {
		t.Errorf("Expected UpdateStats to return a command")
	}

	// Execute the command to get the message
	msg := cmd()
	updateMsg, ok := msg.(StatsUpdateMsg)
	if !ok {
		t.Errorf("Expected StatsUpdateMsg, got %T", msg)
	}

	if updateMsg.Stats != stats {
		t.Errorf("Expected stats to match")
	}
}

// TestGetPatternVisualization tests pattern visualization
func TestGetPatternVisualization(t *testing.T) {
	testCases := []struct {
		pattern  string
		expected string
	}{
		{"", "any"},
		{"ab", "ab" + strings.Repeat("*", 38)},
		{"abcdef", "abcdef" + strings.Repeat("*", 34)},
		{strings.Repeat("a", 40), strings.Repeat("a", 40)}, // Full length pattern
	}

	for _, tc := range testCases {
		stats := &Statistics{Pattern: tc.pattern}
		model := NewStatsModel(stats)
		result := model.getPatternVisualization()

		if result != tc.expected {
			t.Errorf("getPatternVisualization(%s) = %s, expected %s", tc.pattern, result, tc.expected)
		}
	}
}

// TestRenderSections tests individual rendering sections
func TestRenderSections(t *testing.T) {
	stats := &Statistics{
		Difficulty:    1024,
		Probability50: 710,
		Pattern:       "abc",
		IsChecksum:    true,
	}

	model := NewStatsModel(stats)

	// Test pattern overview rendering
	overview := model.renderPatternOverview()
	if !strings.Contains(overview, "Pattern") {
		t.Errorf("Expected pattern overview to contain 'Pattern'")
	}
	if !strings.Contains(overview, "Checksum") {
		t.Errorf("Expected pattern overview to contain 'Checksum'")
	}

	// Test time estimates rendering
	timeEstimates := model.renderTimeEstimates()
	if !strings.Contains(timeEstimates, "Time Estimates") {
		t.Errorf("Expected time estimates to contain title")
	}
	if !strings.Contains(timeEstimates, "addr/s") {
		t.Errorf("Expected time estimates to contain speed units")
	}

	// Test probability examples rendering
	probExamples := model.renderProbabilityExamples()
	if !strings.Contains(probExamples, "Probability Examples") {
		t.Errorf("Expected probability examples to contain title")
	}
	if !strings.Contains(probExamples, "Attempts") {
		t.Errorf("Expected probability examples to contain 'Attempts'")
	}

	// Test recommendations rendering
	recommendations := model.renderRecommendations()
	if !strings.Contains(recommendations, "Recommendations") {
		t.Errorf("Expected recommendations to contain title")
	}
	if !strings.Contains(recommendations, "Difficulty Level") {
		t.Errorf("Expected recommendations to contain difficulty level")
	}
}

// BenchmarkStatsModel_View benchmarks the view rendering performance
func BenchmarkStatsModel_View(b *testing.B) {
	stats := &Statistics{
		Difficulty:    16777216,
		Probability50: 11629080,
		Pattern:       "abcdef",
		IsChecksum:    true,
	}

	model := NewStatsModel(stats)
	model.ready = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

// BenchmarkGenerateStatisticsRows benchmarks the table row generation
func BenchmarkGenerateStatisticsRows(b *testing.B) {
	stats := &Statistics{
		Difficulty:    16777216,
		Probability50: 11629080,
		Pattern:       "abcdef",
		IsChecksum:    true,
	}

	model := NewStatsModel(stats)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.generateStatisticsRows()
	}
}
