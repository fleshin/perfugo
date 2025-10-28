package pages

import (
	"strings"
	"time"
)

// DefaultDash returns an em dash when the provided value is empty or whitespace.
func DefaultDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

// AromaChemicalPotencyLabel converts a numeric strength into a descriptive label.
func AromaChemicalPotencyLabel(strength int) string {
	switch {
	case strength >= 7:
		return "Powerful"
	case strength >= 5:
		return "Strong"
	case strength >= 3:
		return "Moderate"
	case strength > 0:
		return "Delicate"
	default:
		return "Unknown"
	}
}

// formatAuditDate renders YYYY-MM-DD timestamps in a friendly day month year format.
func formatAuditDate(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("02 Jan 2006")
}

// PreferenceStatusMessage normalises the text displayed in the preferences status banner.
func PreferenceStatusMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "Pick a theme and save to update your atelier."
	}
	return trimmed
}
