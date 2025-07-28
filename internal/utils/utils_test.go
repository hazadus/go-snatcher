package utils

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00:00"},
		{59 * time.Second, "00:00:59"},
		{60 * time.Second, "00:01:00"},
		{61*time.Minute + 1*time.Second, "01:01:01"},
		{2*time.Hour + 3*time.Minute + 1*time.Second, "02:03:01"},
		{25*time.Hour + 45*time.Minute + 30*time.Second, "25:45:30"},
	}

	for _, test := range tests {
		result := FormatDuration(test.duration)
		if result != test.expected {
			t.Errorf("FormatDuration(%v) = %s; expected %s", test.duration, result, test.expected)
		}
	}
}

func TestFormatDurationFromSeconds(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "00:00:00"},
		{59, "00:00:59"},
		{60, "00:01:00"},
		{3661, "01:01:01"},
		{7381, "02:03:01"},
	}

	for _, test := range tests {
		result := FormatDurationFromSeconds(test.seconds)
		if result != test.expected {
			t.Errorf("FormatDurationFromSeconds(%d) = %s; expected %s", test.seconds, result, test.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10", 10, "exactly10"},
		{"this is a very long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"abcde", 4, "a..."},
	}

	for _, test := range tests {
		result := TruncateString(test.input, test.maxLen)
		if result != test.expected {
			t.Errorf("TruncateString(%s, %d) = %s; expected %s", test.input, test.maxLen, result, test.expected)
		}
	}
}
