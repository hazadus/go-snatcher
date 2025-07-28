// Package utils содержит утилитарные функции, используемые в разных частях приложения
package utils

import (
	"fmt"
	"time"
)

// FormatDuration форматирует time.Duration в формат HH:MM:SS
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// FormatDurationFromSeconds форматирует продолжительность в секундах в формат HH:MM:SS
func FormatDurationFromSeconds(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// TruncateString обрезает строку до указанной длины, добавляя "..." если строка длиннее
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
