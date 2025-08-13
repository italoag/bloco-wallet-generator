package utils

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// FormatLargeNumber formats large numbers with space separators for thousands
func FormatLargeNumber(num int64) string {
	if num == 0 {
		return "0"
	}

	str := strconv.FormatInt(num, 10)
	result := ""

	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += " "
		}
		result += string(char)
	}

	return result
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "Nearly impossible"
	}

	seconds := d.Seconds()

	// If more than 200 years, return "Thousands of years"
	if seconds > 200*365.25*24*3600 {
		return "Thousands of years"
	}

	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%.1fm", seconds/60)
	} else if seconds < 86400 {
		return fmt.Sprintf("%.1fh", seconds/3600)
	} else if seconds < 31536000 {
		return fmt.Sprintf("%.1fd", seconds/86400)
	} else {
		return fmt.Sprintf("%.1fy", seconds/31536000)
	}
}

// FormatSpeed formats speed in addr/s with appropriate units
func FormatSpeed(speed float64) string {
	if speed < 1000 {
		return fmt.Sprintf("%.0f addr/s", speed)
	} else if speed < 1000000 {
		return fmt.Sprintf("%.1fk addr/s", speed/1000)
	} else {
		return fmt.Sprintf("%.1fM addr/s", speed/1000000)
	}
}

// FormatPercentage formats a percentage with appropriate precision
func FormatPercentage(value float64) string {
	if value < 0.01 {
		return fmt.Sprintf("%.4f%%", value)
	} else if value < 1 {
		return fmt.Sprintf("%.2f%%", value)
	} else {
		return fmt.Sprintf("%.1f%%", value)
	}
}

// FormatBytes formats byte counts in human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatBool formats a boolean as a human-readable string
func FormatBool(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}

// FormatDifficulty formats difficulty with appropriate scale
func FormatDifficulty(difficulty float64) string {
	if difficulty < 1000 {
		return fmt.Sprintf("%.0f", difficulty)
	} else if difficulty < 1000000 {
		return fmt.Sprintf("%.1fK", difficulty/1000)
	} else if difficulty < 1000000000 {
		return fmt.Sprintf("%.1fM", difficulty/1000000)
	} else if difficulty < 1000000000000 {
		return fmt.Sprintf("%.1fB", difficulty/1000000000)
	} else {
		return fmt.Sprintf("%.1fT", difficulty/1000000000000)
	}
}

// FormatProgress creates a progress bar string
func FormatProgress(percentage float64, width int) string {
	if width <= 0 {
		width = 20
	}

	filled := int(percentage * float64(width) / 100.0)
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < width; i++ {
		bar += "░"
	}

	return fmt.Sprintf("[%s] %.1f%%", bar, percentage)
}

// FormatProgressASCII creates an ASCII progress bar for terminals without Unicode support
func FormatProgressASCII(percentage float64, width int) string {
	if width <= 0 {
		width = 20
	}

	filled := int(percentage * float64(width) / 100.0)
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "#"
	}
	for i := filled; i < width; i++ {
		bar += "-"
	}

	return fmt.Sprintf("[%s] %.1f%%", bar, percentage)
}

// TruncateString truncates a string to the specified length with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}

// PadRight pads a string with spaces to the specified width
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}

	padding := make([]byte, width-len(s))
	for i := range padding {
		padding[i] = ' '
	}

	return s + string(padding)
}

// PadLeft pads a string with spaces to the specified width (left-aligned)
func PadLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}

	padding := make([]byte, width-len(s))
	for i := range padding {
		padding[i] = ' '
	}

	return string(padding) + s
}

// Center centers a string within the specified width
func Center(s string, width int) string {
	if len(s) >= width {
		return s
	}

	totalPadding := width - len(s)
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding

	left := make([]byte, leftPadding)
	right := make([]byte, rightPadding)

	for i := range left {
		left[i] = ' '
	}
	for i := range right {
		right[i] = ' '
	}

	return string(left) + s + string(right)
}

// FormatTable formats data as a simple table
func FormatTable(headers []string, rows [][]string, padding int) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Add padding
	for i := range colWidths {
		colWidths[i] += padding * 2
	}

	result := ""

	// Format header
	for i, header := range headers {
		result += PadRight(header, colWidths[i])
		if i < len(headers)-1 {
			result += "|"
		}
	}
	result += "\n"

	// Add separator
	for i, width := range colWidths {
		for j := 0; j < width; j++ {
			result += "-"
		}
		if i < len(colWidths)-1 {
			result += "+"
		}
	}
	result += "\n"

	// Format rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				result += PadRight(cell, colWidths[i])
				if i < len(colWidths)-1 {
					result += "|"
				}
			}
		}
		result += "\n"
	}

	return result
}

// CalculateDifficulty calculates the difficulty of finding a bloco address
func CalculateDifficulty(prefix, suffix string, isChecksum bool) float64 {
	pattern := prefix + suffix
	baseDifficulty := math.Pow(16, float64(len(pattern)))

	if !isChecksum {
		return baseDifficulty
	}

	// Count letters (a-f, A-F) in the pattern for checksum calculation
	letterCount := 0
	for _, char := range pattern {
		if (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F') {
			letterCount++
		}
	}

	return baseDifficulty * math.Pow(2, float64(letterCount))
}

// CalculateProbability calculates the probability of finding an address after N attempts
func CalculateProbability(difficulty float64, attempts int64) float64 {
	if difficulty <= 0 {
		return 0
	}
	return 1 - math.Pow(1-1/difficulty, float64(attempts))
}

// CalculateProbability50 calculates how many attempts are needed for 50% probability
func CalculateProbability50(difficulty float64) int64 {
	if difficulty <= 0 {
		return 0
	}
	result := math.Log(0.5) / math.Log(1-1/difficulty)
	if math.IsInf(result, 0) || result < 0 {
		return -1 // Nearly impossible
	}
	return int64(math.Floor(result))
}

// IsValidHex checks if a string contains only valid hex characters
func IsValidHex(hex string) bool {
	if len(hex) == 0 {
		return true
	}
	for _, char := range hex {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'f') ||
			(char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
