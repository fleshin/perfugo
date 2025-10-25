package theme

import "strings"

// Option represents a selectable theme exposed to the UI.
type Option struct {
	Value string
	Label string
}

// WorkspaceTheme contains resolved styling primitives for the application shell.
type WorkspaceTheme struct {
	Key                   string
	BodyClass             string
	ShellClass            string
	PanelSurfaceClass     string
	PanelSoftSurfaceClass string
	BorderStrongClass     string
	BorderSoftClass       string
	AccentTextClass       string
	MutedTextClass        string
	SubtleTextClass       string
}

const (
	// DefaultKey defines the fallback theme when no user preference exists.
	DefaultKey = "nocturne"
)

var catalogue = map[string]WorkspaceTheme{
	"nocturne": {
		Key:                   "nocturne",
		BodyClass:             "min-h-screen bg-slate-950 text-slate-100",
		ShellClass:            "workspace-shell dark",
		PanelSurfaceClass:     "workspace-surface",
		PanelSoftSurfaceClass: "workspace-surface-soft",
		BorderStrongClass:     "workspace-border-strong",
		BorderSoftClass:       "workspace-border-soft",
		AccentTextClass:       "workspace-accent",
		MutedTextClass:        "workspace-muted",
		SubtleTextClass:       "workspace-subtle",
	},
	"atelier_ivory": {
		Key:                   "atelier_ivory",
		BodyClass:             "min-h-screen bg-stone-50 text-stone-900",
		ShellClass:            "workspace-shell light",
		PanelSurfaceClass:     "workspace-surface",
		PanelSoftSurfaceClass: "workspace-surface-soft",
		BorderStrongClass:     "workspace-border-strong",
		BorderSoftClass:       "workspace-border-soft",
		AccentTextClass:       "workspace-accent",
		MutedTextClass:        "workspace-muted",
		SubtleTextClass:       "workspace-subtle",
	},
	"midnight_draft": {
		Key:                   "midnight_draft",
		BodyClass:             "min-h-screen bg-slate-950 text-slate-100",
		ShellClass:            "workspace-shell twilight",
		PanelSurfaceClass:     "workspace-surface",
		PanelSoftSurfaceClass: "workspace-surface-soft",
		BorderStrongClass:     "workspace-border-strong",
		BorderSoftClass:       "workspace-border-soft",
		AccentTextClass:       "workspace-accent",
		MutedTextClass:        "workspace-muted",
		SubtleTextClass:       "workspace-subtle",
	},
}

var options = []Option{
	{Value: "nocturne", Label: "Nocturne (Dark)"},
	{Value: "atelier_ivory", Label: "Atelier Ivory (Light)"},
	{Value: "midnight_draft", Label: "Midnight Draft (Blue)"},
}

// Resolve returns the registered theme configuration for the provided key.
func Resolve(key string) WorkspaceTheme {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if value, ok := catalogue[normalized]; ok {
		return value
	}
	return catalogue[DefaultKey]
}

// Options exposes the available theme selections for rendering in a form control.
func Options() []Option {
	return options
}
