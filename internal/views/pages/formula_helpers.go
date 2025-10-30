package pages

import (
	"fmt"
	"strings"

	"perfugo/models"
)

// NextUntitledFormulaName returns a human-friendly default name for a new formula.
func NextUntitledFormulaName(existing []models.Formula) string {
	const base = "Untitled Formula"
	used := make(map[string]struct{}, len(existing))
	for _, formula := range existing {
		name := strings.TrimSpace(formula.Name)
		if name == "" {
			continue
		}
		used[strings.ToLower(name)] = struct{}{}
	}

	if _, ok := used[strings.ToLower(base)]; !ok {
		return base
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s %d", base, i)
		if _, ok := used[strings.ToLower(candidate)]; !ok {
			return candidate
		}
	}
}

// NextCopiedFormulaName generates a non-conflicting name when duplicating a formula.
func NextCopiedFormulaName(existing []models.Formula, base string) string {
	baseTrim := strings.TrimSpace(base)
	if baseTrim == "" {
		return NextUntitledFormulaName(existing)
	}

	used := make(map[string]struct{}, len(existing))
	for _, formula := range existing {
		name := strings.TrimSpace(formula.Name)
		if name == "" {
			continue
		}
		used[strings.ToLower(name)] = struct{}{}
	}

	candidate := fmt.Sprintf("%s (Copy)", baseTrim)
	if _, ok := used[strings.ToLower(candidate)]; !ok {
		return candidate
	}

	for i := 2; ; i++ {
		candidate = fmt.Sprintf("%s (Copy %d)", baseTrim, i)
		if _, ok := used[strings.ToLower(candidate)]; !ok {
			return candidate
		}
	}
}
