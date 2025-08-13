package wallet

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// WalletLogger handles logging wallets to file
type WalletLogger struct {
	filename string
	file     *os.File
	mu       sync.Mutex
}

// NewWalletLogger creates a new wallet logger with timestamp (daily file)
func NewWalletLogger() (*WalletLogger, error) {
	timestamp := time.Now().Format("20060102")
	filename := fmt.Sprintf("wallets-%s.log", timestamp)
	
	// Check if file exists to determine if we need to write header
	_, fileExists := os.Stat(filename)
	isNewFile := os.IsNotExist(fileExists)
	
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet log file: %w", err)
	}
	
	logger := &WalletLogger{
		filename: filename,
		file:     file,
	}
	
	// Write header only if it's a new file
	if isNewFile {
		header := fmt.Sprintf("# Wallet Generation Log - Started at %s\n", time.Now().Format(time.RFC3339))
		header += "# Format: TIMESTAMP | ADDRESS | PUBLIC_KEY | PRIVATE_KEY | ATTEMPTS | DURATION\n"
		header += "# =====================================================================================\n"
		
		if _, err := file.WriteString(header); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to write header: %w", err)
		}
	}
	
	return logger, nil
}

// LogWallet logs a wallet to the file
func (wl *WalletLogger) LogWallet(result *GenerationResult) error {
	if result == nil || result.Wallet == nil {
		return fmt.Errorf("invalid wallet result")
	}
	
	wl.mu.Lock()
	defer wl.mu.Unlock()
	
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s | %s | %s | %s | %d | %v\n",
		timestamp,
		result.Wallet.Address,
		result.Wallet.PublicKey,
		result.Wallet.PrivateKey,
		result.Attempts,
		result.Duration,
	)
	
	if _, err := wl.file.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write wallet log: %w", err)
	}
	
	// Flush to ensure data is written immediately
	return wl.file.Sync()
}

// Close closes the log file
func (wl *WalletLogger) Close() error {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	
	if wl.file != nil {
		// Sync any pending writes
		wl.file.Sync()
		return wl.file.Close()
	}
	return nil
}

// GetFilename returns the log filename
func (wl *WalletLogger) GetFilename() string {
	return wl.filename
}