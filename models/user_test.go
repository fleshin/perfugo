package models

import "testing"

func TestValidTheme(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{"nocturne", ThemeNocturne, true},
		{"atelier", ThemeAtelierIvory, true},
		{"midnight", ThemeMidnightDraft, true},
		{"unknown", "galaxy", false},
		{"empty", "", false},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidTheme(tt.value); got != tt.want {
				t.Fatalf("ValidTheme(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeTheme(t *testing.T) {
	t.Parallel()

	if got := NormalizeTheme(ThemeAtelierIvory); got != ThemeAtelierIvory {
		t.Fatalf("NormalizeTheme returned %q, want %q", got, ThemeAtelierIvory)
	}

	if got := NormalizeTheme("  invalid  "); got != DefaultTheme {
		t.Fatalf("NormalizeTheme returned %q, want %q", got, DefaultTheme)
	}
}
