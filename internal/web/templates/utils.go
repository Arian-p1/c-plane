package templates

import (
	"fmt"
	"strings"
	"time"
)

// timeAgo returns a human-readable string representing how long ago the given time was
func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}

	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "Just now"
	case duration < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

// formatBytes returns a human-readable string representing the given number of bytes
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return "..."
	}
	return s[:length-3] + "..."
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// formatDuration returns a human-readable duration string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// formatPercentage formats a float as a percentage
func formatPercentage(value, total float64) string {
	if total == 0 {
		return "0%"
	}
	percentage := (value / total) * 100
	return fmt.Sprintf("%.1f%%", percentage)
}

// safeString returns a safe string for HTML output, handling nil pointers
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// formatTimestamp formats a timestamp for display
func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatDate formats a date for display
func formatDate(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}
	return t.Format("2006-01-02")
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}
	return t.Format("15:04:05")
}

// pluralize returns the singular or plural form based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, separator string) string {
	return strings.Join(strs, separator)
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// clamp constrains a value between min and max
func clamp(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

// defaultString returns the default value if the string is empty
func defaultString(s, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}

// yesNo returns "Yes" or "No" based on a boolean value
func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// enabledDisabled returns "Enabled" or "Disabled" based on a boolean value
func enabledDisabled(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}

// activeInactive returns "Active" or "Inactive" based on a boolean value
func activeInactive(b bool) string {
	if b {
		return "Active"
	}
	return "Inactive"
}
