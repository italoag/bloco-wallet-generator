package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"bloco-eth/tui"
)

// TestTUIProgressModel tests that the TUI progress model handles wallet results correctly
func TestTUIProgressModel(t *testing.T) {
	// Create statistics
	stats := &tui.Statistics{
		Difficulty:      16,
		Probability50:   10,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         "A*",
		IsChecksum:      false,
	}

	// Create progress model
	model := tui.NewProgressModel(stats, nil)

	// Test initial state - should show progress info but no wallet results
	initialView := model.View()
	if initialView == "" {
		t.Error("Expected non-empty view initially")
	}
	
	// Initial view should contain progress information
	if !contains(initialView, "Pattern: A*") {
		t.Error("Expected initial view to contain pattern information")
	}
	if !contains(initialView, "📊 Statistics") {
		t.Error("Expected initial view to contain statistics section")
	}
	if contains(initialView, "Generated Wallets") {
		t.Error("Expected initial view to NOT contain wallet results section")
	}

	// Test wallet result message
	walletResult := tui.WalletResult{
		Index:      1,
		Address:    "0xA247Fe4d8bFF4181dC1f0e0E79BC4103480637Ae",
		PrivateKey: "0x19a254e7b7b1e6a0b14e7009c1d2a7aff126323d6fceaddc3861ff9b194bd22c",
		Attempts:   8,
		Time:       500 * time.Microsecond,
		Error:      "",
	}

	// Send wallet result to model
	updatedModel, _ := model.Update(tui.WalletResultMsg{Result: walletResult})
	progressModel := updatedModel.(tui.ProgressModel)

	// Verify that the model now shows results
	view := progressModel.View()
	if view == "" {
		t.Error("Expected non-empty view after wallet result")
	}

	// Check that the view STILL contains progress information (should always be there)
	if !contains(view, "Pattern: A*") {
		t.Error("Expected view to still contain pattern information")
	}
	if !contains(view, "📊 Statistics") {
		t.Error("Expected view to still contain statistics section")
	}
	
	// Check that the view NOW contains wallet results (added below progress info)
	if !contains(view, "Generated Wallets") {
		t.Error("Expected view to contain 'Generated Wallets' section")
	}

	if !contains(view, walletResult.Address) {
		t.Error("Expected view to contain wallet address")
	}
	
	// Verify that progress information comes BEFORE wallet results
	progressIndex := getIndexOf(view, "📊 Statistics")
	resultsIndex := getIndexOf(view, "Generated Wallets")
	
	if progressIndex == -1 {
		t.Error("Expected to find statistics section")
	}
	if resultsIndex == -1 {
		t.Error("Expected to find wallet results section")
	}
	if progressIndex >= resultsIndex {
		t.Error("Expected progress information to appear BEFORE wallet results")
	}
}

// TestTUILayoutWithMultipleWallets tests the layout with many wallet results (scroll scenario)
func TestTUILayoutWithMultipleWallets(t *testing.T) {
	// Create statistics
	stats := &tui.Statistics{
		Difficulty:      16,
		Probability50:   10,
		CurrentAttempts: 100,
		Speed:           50.5,
		Probability:     25.3,
		EstimatedTime:   2 * time.Second,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         "ABC*",
		IsChecksum:      true,
	}

	// Create progress model
	model := tui.NewProgressModel(stats, nil)

	// Add multiple wallet results (more than table height of 8)
	for i := 1; i <= 12; i++ {
		walletResult := tui.WalletResult{
			Index:      i,
			Address:    fmt.Sprintf("0xABC%037d", i),
			PrivateKey: fmt.Sprintf("0x%064d", i),
			Attempts:   i * 10,
			Time:       time.Duration(i*100) * time.Microsecond,
			Error:      "",
		}
		
		updatedModel, _ := model.Update(tui.WalletResultMsg{Result: walletResult})
		model = updatedModel.(tui.ProgressModel)
	}
	
	view := model.View()
	
	// Should still contain all progress information
	if !contains(view, "Pattern: ABC* (checksum)") {
		t.Error("Expected view to contain pattern with checksum info")
	}
	if !contains(view, "📊 Statistics") {
		t.Error("Expected view to contain statistics section")
	}
	// Should contain progress or probability information (depends on whether totalWallets is set)
	if !contains(view, "25.30%") && !contains(view, "wallets") {
		t.Error("Expected view to contain progress or probability information")
	}
	
	// Should contain wallet results section
	if !contains(view, "Generated Wallets (12)") {
		t.Error("Expected view to show count of 12 generated wallets")
	}
	
	// Should contain scroll instructions since we have more than 8 results
	if !contains(view, "scroll") {
		t.Error("Expected view to contain scroll instructions")
	}
	
	// Test keyboard navigation
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if updatedModel == nil {
		t.Error("Expected model to handle keyboard navigation")
	}
}

// TestTUIStatsUpdates tests that the TUI correctly handles real-time statistics updates
func TestTUIStatsUpdates(t *testing.T) {
	// Create initial statistics
	stats := &tui.Statistics{
		Difficulty:      256,
		Probability50:   100,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         "ABCD*",
		IsChecksum:      false,
	}

	// Create progress model
	model := tui.NewProgressModel(stats, nil)

	// Test initial state
	initialView := model.View()
	if !contains(initialView, "0 addr/s") {
		t.Error("Expected initial view to show 0 speed")
	}
	if !contains(initialView, "0 attempts") {
		t.Error("Expected initial view to show 0 attempts")  
	}

	// Send a statistics update
	statsUpdate := tui.ProgressMsg{
		Attempts:      1500,
		Speed:         75.5,
		Probability:   15.8,
		EstimatedTime: 5 * time.Second,
		Difficulty:    256,
		Pattern:       "ABCD*",
	}

	updatedModel, _ := model.Update(statsUpdate)
	progressModel := updatedModel.(tui.ProgressModel)

	// Check that statistics were updated
	updatedView := progressModel.View()
	t.Logf("Updated view content: %q", updatedView)
	
	// Should show updated attempts (formatted with spaces: "1 500")
	if !contains(updatedView, "1 500") {
		t.Error("Expected view to show updated attempt count")
	}
	
	// Should show updated speed (75.5 rounded to 76)
	if !contains(updatedView, "76 addr/s") {
		t.Error("Expected view to show updated speed")
	}
	
	// Should show updated probability
	if !contains(updatedView, "15.80%") {
		t.Error("Expected view to show updated probability")
	}

	// Test multiple updates to ensure real-time functionality
	secondUpdate := tui.ProgressMsg{
		Attempts:      3000,
		Speed:         120.2,
		Probability:   25.4,
		EstimatedTime: 3 * time.Second,
		Difficulty:    256,
		Pattern:       "ABCD*",
	}

	updatedModel2, _ := progressModel.Update(secondUpdate)
	progressModel2 := updatedModel2.(tui.ProgressModel)
	
	secondView := progressModel2.View()
	
	// Should show the most recent statistics (formatted with spaces: "3 000")
	if !contains(secondView, "3 000") {
		t.Error("Expected view to show second updated attempt count")
	}
	
	if !contains(secondView, "120 addr/s") {
		t.Error("Expected view to show second updated speed")  
	}
}

// TestTUIProgressBarCalculation tests that the progress bar reflects wallet completion correctly
func TestTUIProgressBarCalculation(t *testing.T) {
	// Create initial statistics
	stats := &tui.Statistics{
		Difficulty:      16,
		Probability50:   10,
		CurrentAttempts: 0,
		Speed:           0,
		Probability:     0,
		EstimatedTime:   0,
		StartTime:       time.Now(),
		LastUpdate:      time.Now(),
		Pattern:         "A*",
		IsChecksum:      false,
	}

	// Create progress model
	model := tui.NewProgressModel(stats, nil)

	// Test initial state - 0% progress
	initialView := model.View()
	if !contains(initialView, "0.00% probability") {
		t.Error("Expected initial view to show 0% probability")
	}

	// Simulate progress: 2 out of 5 wallets completed (40% progress)
	progressUpdate := tui.ProgressMsg{
		Attempts:         500,
		Speed:            25.5,
		Probability:      40.0, // This should now represent actual progress percentage
		EstimatedTime:    3 * time.Second,
		Difficulty:       16,
		Pattern:          "A*",
		CompletedWallets: 2,      // 2 wallets done
		TotalWallets:     5,      // out of 5 total
		ProgressPercent:  40.0,   // 40% progress for progress bar
	}

	updatedModel, _ := model.Update(progressUpdate)
	progressModel := updatedModel.(tui.ProgressModel)

	updatedView := progressModel.View()
	t.Logf("Progress view content: %q", updatedView)
	
	// Should show wallets completed
	if !contains(updatedView, "2/5 wallets completed") {
		t.Error("Expected view to show '2/5 wallets completed'")
	}
	
	// Should show correct percentage (40.0%)
	if !contains(updatedView, "40.0%") {
		t.Error("Expected view to show 40.0% progress")
	}

	// Test completion: 5 out of 5 wallets (100% progress)
	completeUpdate := tui.ProgressMsg{
		Attempts:         1250,
		Speed:            35.8,
		Probability:      100.0,
		EstimatedTime:    0,
		Difficulty:       16,
		Pattern:          "A*",
		CompletedWallets: 5,      // All 5 wallets done
		TotalWallets:     5,      // out of 5 total  
		ProgressPercent:  100.0,  // 100% progress for progress bar
	}

	finalModel, _ := progressModel.Update(completeUpdate)
	finalProgressModel := finalModel.(tui.ProgressModel)

	finalView := finalProgressModel.View()
	
	// Should show completion
	if !contains(finalView, "5/5 wallets completed") {
		t.Error("Expected view to show '5/5 wallets completed'")
	}
	
	// Should show 100%
	if !contains(finalView, "100.0%") {
		t.Error("Expected view to show 100.0% progress")
	}
}

// TestTUIWalletResultChannel tests the channel communication for wallet results
func TestTUIWalletResultChannel(t *testing.T) {
	// Create a channel for wallet results
	walletResultsChan := make(chan WalletResultForTUI, 1)

	// Send a test result
	testResult := WalletResultForTUI{
		Index:      1,
		Address:    "0xA247Fe4d8bFF4181dC1f0e0E79BC4103480637Ae",
		PrivateKey: "0x19a254e7b7b1e6a0b14e7009c1d2a7aff126323d6fceaddc3861ff9b194bd22c",
		Attempts:   8,
		Duration:   500 * time.Microsecond,
		Error:      "",
	}

	// Send the result
	select {
	case walletResultsChan <- testResult:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to send wallet result to channel")
	}

	// Receive the result
	select {
	case received := <-walletResultsChan:
		if received.Address != testResult.Address {
			t.Errorf("Expected address %s, got %s", testResult.Address, received.Address)
		}
		if received.Index != testResult.Index {
			t.Errorf("Expected index %d, got %d", testResult.Index, received.Index)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to receive wallet result from channel")
	}

	close(walletResultsChan)
}

// TestTUITableCreation tests that the Bubbletea table is created correctly
func TestTUITableCreation(t *testing.T) {
	// Create the columns as defined in NewProgressModel
	columns := []table.Column{
		{Title: "№", Width: 3},
		{Title: "Address", Width: 42},
		{Title: "Private Key", Width: 66},
		{Title: "Attempts", Width: 8},
		{Title: "Time", Width: 8},
	}

	// Create table
	tbl := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	// Test that table was created
	if tbl.View() == "" {
		t.Error("Expected non-empty table view")
	}

	// Add a test row
	rows := []table.Row{
		{"1", "0xA247Fe4d8bFF4181dC1f0e0E79BC4103480637Ae", "0x19a254e7b7b1e6a0b14e7009c1d2a7aff126323d6fceaddc3861ff9b194bd22c", "8", "500µs"},
	}
	tbl.SetRows(rows)

	// Verify table has content
	tableView := tbl.View()
	if tableView == "" {
		t.Error("Expected non-empty table view after adding rows")
	}

	// Check that table contains the test data
	if !contains(tableView, "0xA247Fe4d8bFF4181dC1f0e0E79BC4103480637Ae") {
		t.Error("Expected table to contain test address")
	}
}

// TestWalletResultChannel tests the channel communication for wallet results
func TestWalletResultChannelCommunication(t *testing.T) {
	// Test the channel communication pattern used in TUI
	walletResultsChan := make(chan WalletResultForTUI, 5)
	
	// Simulate background wallet generation sending results to TUI
	go func() {
		for i := 1; i <= 3; i++ {
			result := WalletResultForTUI{
				Index:      i,
				Address:    fmt.Sprintf("0xA%040d", i),
				PrivateKey: fmt.Sprintf("0x%064d", i),
				Attempts:   i * 10,
				Duration:   time.Duration(i) * time.Millisecond,
				Error:      "",
			}
			walletResultsChan <- result
		}
		close(walletResultsChan)
	}()
	
	// Simulate TUI receiving results (like in displayMultipleWalletsTUI)
	receivedCount := 0
	for result := range walletResultsChan {
		receivedCount++
		
		// Verify result structure
		if result.Index != receivedCount {
			t.Errorf("Expected index %d, got %d", receivedCount, result.Index)
		}
		
		if result.Address == "" {
			t.Error("Expected non-empty address")
		}
		
		if result.PrivateKey == "" {
			t.Error("Expected non-empty private key") 
		}
		
		if result.Attempts <= 0 {
			t.Error("Expected positive attempt count")
		}
	}
	
	if receivedCount != 3 {
		t.Errorf("Expected 3 results, got %d", receivedCount)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to find the index of a substring
func getIndexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}