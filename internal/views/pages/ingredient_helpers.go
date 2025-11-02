package pages

import (
	"fmt"
	"sort"
	"strings"

	"perfugo/models"
)

var allowedPyramidPositions = []string{"top", "top-heart", "heart", "heart-base", "base", "all"}

// AllowedPyramidPositions returns the list of canonical pyramid position values.
func AllowedPyramidPositions() []string {
	result := make([]string, len(allowedPyramidPositions))
	copy(result, allowedPyramidPositions)
	return result
}

// WheelPositionOptions returns a sorted list of unique, non-empty wheel positions.
func WheelPositionOptions(chemicals []models.AromaChemical) []string {
	unique := make(map[string]string)
	for _, chemical := range chemicals {
		trimmed := strings.TrimSpace(chemical.WheelPosition)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := unique[key]; !exists {
			unique[key] = trimmed
		}
	}

	options := make([]string, 0, len(unique))
	for _, value := range unique {
		options = append(options, value)
	}
	sort.Slice(options, func(i, j int) bool {
		return strings.ToLower(options[i]) < strings.ToLower(options[j])
	})
	return options
}

// NormalizePyramidPosition converts the supplied value to its canonical representation.
// It returns the normalized value along with a boolean indicating whether the input was valid.
func NormalizePyramidPosition(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", true
	}

	normalized := strings.ToLower(trimmed)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")

	for _, option := range allowedPyramidPositions {
		if normalized == option {
			return option, true
		}
	}
	return "", false
}

// CanonicalPyramidPosition returns the normalized value or an empty string when it cannot be canonicalised.
func CanonicalPyramidPosition(value string) string {
	if normalized, ok := NormalizePyramidPosition(value); ok {
		return normalized
	}
	return ""
}

// PyramidPositionLabel returns a human readable label for the supplied value.
func PyramidPositionLabel(value string) string {
	normalized, _ := NormalizePyramidPosition(value)
	if normalized == "" {
		return DefaultDash("")
	}

	parts := strings.Split(normalized, "-")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "-")
}

func FormatPercentage(value float64) string {
	if value <= 0 {
		return DefaultDash("")
	}
	return fmt.Sprintf("%.2f%%", value)
}

func FormatPricePerMg(value float64) string {
	if value <= 0 {
		return DefaultDash("")
	}
	return fmt.Sprintf("$%.4f", value)
}

func FormatPopularity(value int) string {
	if value <= 0 {
		return DefaultDash("")
	}
	return fmt.Sprintf("%d", value)
}

// IngredientFormAction derives the correct HTMX endpoint for creating or updating ingredients.
func IngredientFormAction(chemical *models.AromaChemical) string {
	if chemical == nil || chemical.ID == 0 {
		return "/app/sections/ingredients/create"
	}
	return "/app/sections/ingredients/update"
}

func FormatFloatInput(value float64, precision int) string {
	if value == 0 {
		return ""
	}
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, value)
}

func FormatIntInput(value int) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%d", value)
}
