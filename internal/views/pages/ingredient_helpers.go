package pages

import (
	"strings"
)

var allowedPyramidPositions = []string{"top", "top-heart", "heart", "heart-base", "base", "all"}

// AllowedPyramidPositions returns the list of canonical pyramid position values.
func AllowedPyramidPositions() []string {
	result := make([]string, len(allowedPyramidPositions))
	copy(result, allowedPyramidPositions)
	return result
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
