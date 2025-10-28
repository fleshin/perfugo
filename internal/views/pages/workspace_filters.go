package pages

import (
	"net/http"
	"strconv"
	"strings"

	"perfugo/models"
)

// IngredientFilters capture the client-driven state for aroma chemical lookups.
type IngredientFilters struct {
	Query string
}

// IngredientFiltersFromRequest extracts filter inputs from an HTTP request.
func IngredientFiltersFromRequest(r *http.Request) IngredientFilters {
	filters := IngredientFilters{}
	if err := r.ParseForm(); err != nil {
		return filters
	}
	filters.Query = strings.TrimSpace(r.FormValue("q"))
	return filters
}

// FilterAromaChemicals applies the provided filters to a list of aroma chemicals.
func FilterAromaChemicals(all []models.AromaChemical, filters IngredientFilters) []models.AromaChemical {
	if filters.Query == "" {
		return all
	}
	query := strings.ToLower(filters.Query)
	filtered := make([]models.AromaChemical, 0, len(all))
	for _, chemical := range all {
		if containsFold(chemical.IngredientName, query) ||
			containsFold(chemical.CASNumber, query) ||
			containsFold(chemical.Type, query) {
			filtered = append(filtered, chemical)
		}
	}
	return filtered
}

// FindAromaChemical returns the first aroma chemical matching the requested identifier.
func FindAromaChemical(all []models.AromaChemical, id uint) *models.AromaChemical {
	for i := range all {
		if all[i].ID == id {
			return &all[i]
		}
	}
	return nil
}

// FormulaFilters capture the client-driven state for formula lookups.
type FormulaFilters struct {
	Query string
}

// FormulaFiltersFromRequest extracts filter inputs for formula listings.
func FormulaFiltersFromRequest(r *http.Request) FormulaFilters {
	filters := FormulaFilters{}
	if err := r.ParseForm(); err != nil {
		return filters
	}
	filters.Query = strings.TrimSpace(r.FormValue("q"))
	return filters
}

// FilterFormulas applies the provided filters to a list of formulas.
func FilterFormulas(all []models.Formula, filters FormulaFilters) []models.Formula {
	if filters.Query == "" {
		return all
	}
	query := strings.ToLower(filters.Query)
	filtered := make([]models.Formula, 0, len(all))
	for _, formula := range all {
		if containsFold(formula.Name, query) || containsFold(formula.Notes, query) {
			filtered = append(filtered, formula)
		}
	}
	return filtered
}

// FindFormula returns the first formula matching the requested identifier.
func FindFormula(all []models.Formula, id uint) *models.Formula {
	for i := range all {
		if all[i].ID == id {
			return &all[i]
		}
	}
	return nil
}

// FormulaIngredientsFor collects ingredients linked to the supplied formula identifier.
func FormulaIngredientsFor(all []models.FormulaIngredient, formulaID uint) []models.FormulaIngredient {
	if formulaID == 0 {
		return nil
	}
	matches := make([]models.FormulaIngredient, 0)
	for _, ingredient := range all {
		if ingredient.FormulaID == formulaID {
			matches = append(matches, ingredient)
		}
	}
	return matches
}

// ParseUint extracts a uint from the provided string, returning zero on failure.
func ParseUint(value string) uint {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return 0
	}
	return uint(parsed)
}

func containsFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), needle)
}
