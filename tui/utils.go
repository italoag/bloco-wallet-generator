package tui

import (
	"fmt"
	"strconv"
	"time"
)

// formatLargeNumber formats large numbers with space separators
func formatLargeNumber(num int64) string {
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

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
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
