package layout

import (
	"sort"

	"perfugo/models"
)

// ThemeDefinition describes a visual theme that can be applied to the workspace layout.
type ThemeDefinition struct {
	ID          string
	Label       string
	Description string
}

var themeRegistry = map[string]ThemeDefinition{
	models.ThemeNocturne: {
		ID:          models.ThemeNocturne,
		Label:       "Nocturne",
		Description: "Dark mode with soft contrast and cyan highlights.",
	},
	models.ThemeAtelierIvory: {
		ID:          models.ThemeAtelierIvory,
		Label:       "Atelier Ivory",
		Description: "Warm ivory canvas with charcoal typography.",
	},
	models.ThemeMidnightDraft: {
		ID:          models.ThemeMidnightDraft,
		Label:       "Midnight Draft",
		Description: "Muted blue workspace with indigo accents.",
	},
}

// ThemeByID returns a definition for the provided identifier, falling back to the default theme.
func ThemeByID(id string) ThemeDefinition {
	if def, ok := themeRegistry[id]; ok {
		return def
	}
	return themeRegistry[models.DefaultTheme]
}

// ThemeOptions exposes all theme definitions sorted by label for form rendering.
func ThemeOptions() []ThemeDefinition {
	options := make([]ThemeDefinition, 0, len(themeRegistry))
	for _, def := range themeRegistry {
		options = append(options, def)
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})
	return options
}
