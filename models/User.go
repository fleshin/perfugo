package models

import "gorm.io/gorm"

const (
	// ThemeNocturne represents the dark studio palette.
	ThemeNocturne = "nocturne"
	// ThemeAtelierIvory is a light, warm palette for high contrast.
	ThemeAtelierIvory = "atelier_ivory"
	// ThemeMidnightDraft offers a muted blue workspace.
	ThemeMidnightDraft = "midnight_draft"
	// DefaultTheme is applied when no explicit preference has been saved.
	DefaultTheme = ThemeNocturne
)

// ValidTheme reports whether the provided identifier maps to a supported theme.
func ValidTheme(value string) bool {
	switch value {
	case ThemeNocturne, ThemeAtelierIvory, ThemeMidnightDraft:
		return true
	default:
		return false
	}
}

// NormalizeTheme coerces a user-provided theme to a supported value, falling back to the default.
func NormalizeTheme(value string) string {
	if ValidTheme(value) {
		return value
	}
	return DefaultTheme
}

// User represents an application account that can authenticate with the platform.
type User struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Name         string
	Theme        string `gorm:"not null;default:nocturne"`
}
