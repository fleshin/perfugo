package pages

import (
	"fmt"
	"sort"
	"strconv"

	"perfugo/models"
)

// WorkspaceSnapshot aggregates relational data required to render the atelier workspace.
type WorkspaceSnapshot struct {
	Formulas           []models.Formula
	FormulaIngredients []models.FormulaIngredient
	AromaChemicals     []models.AromaChemical
	Theme              string
	UserID             uint
}

// NewWorkspaceSnapshot normalises and sorts the data required by the workspace views.
func NewWorkspaceSnapshot(formulas []models.Formula, ingredients []models.FormulaIngredient, chemicals []models.AromaChemical, theme string, userID uint) WorkspaceSnapshot {
	sort.SliceStable(formulas, func(i, j int) bool {
		return formulas[i].Name < formulas[j].Name
	})

	sort.SliceStable(ingredients, func(i, j int) bool {
		if ingredients[i].FormulaID == ingredients[j].FormulaID {
			return ingredients[i].ID < ingredients[j].ID
		}
		return ingredients[i].FormulaID < ingredients[j].FormulaID
	})

	sort.SliceStable(chemicals, func(i, j int) bool {
		return chemicals[i].IngredientName < chemicals[j].IngredientName
	})

	return WorkspaceSnapshot{
		Formulas:           formulas,
		FormulaIngredients: ingredients,
		AromaChemicals:     chemicals,
		Theme:              theme,
		UserID:             userID,
	}
}

// EmptyWorkspaceSnapshot returns a zero-value snapshot to simplify call sites when no data is available.
func EmptyWorkspaceSnapshot() WorkspaceSnapshot {
	return WorkspaceSnapshot{Theme: models.DefaultTheme}
}

// IngredientDisplayName returns a user-friendly label for the formula ingredient's source.
func IngredientDisplayName(ingredient models.FormulaIngredient) string {
	if ingredient.AromaChemical != nil && ingredient.AromaChemical.IngredientName != "" {
		return ingredient.AromaChemical.IngredientName
	}
	if ingredient.SubFormula != nil && ingredient.SubFormula.Name != "" {
		return ingredient.SubFormula.Name
	}
	return "Unassigned Ingredient"
}

// IngredientSourceKind indicates whether an ingredient links to an aroma chemical or a sub-formula.
func IngredientSourceKind(ingredient models.FormulaIngredient) string {
	if ingredient.AromaChemicalID != nil {
		return "Aroma Chemical"
	}
	if ingredient.SubFormulaID != nil {
		return "Sub-Formula"
	}
	return "Unassigned"
}

// FormulaLookup builds a map of formula IDs to formula names to speed up template rendering.
func (s WorkspaceSnapshot) FormulaLookup() map[uint]string {
	lookup := make(map[uint]string, len(s.Formulas))
	for _, formula := range s.Formulas {
		lookup[uint(formula.ID)] = formula.Name
	}
	return lookup
}

// FormulaIngredientRowKey generates a stable identifier for an ingredient row within the editor form.
func FormulaIngredientRowKey(ingredient models.FormulaIngredient, index int) string {
	if ingredient.ID == 0 {
		return fmt.Sprintf("new-%d", index)
	}
	return fmt.Sprintf("existing-%d", ingredient.ID)
}

// FormulaIngredientEntryID returns the identifier for the ingredient row, defaulting to zero for new entries.
func FormulaIngredientEntryID(ingredient *models.FormulaIngredient) string {
	if ingredient == nil || ingredient.ID == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", ingredient.ID)
}

// FormulaIngredientSourceValue returns the encoded select value for the ingredient's source.
func FormulaIngredientSourceValue(ingredient *models.FormulaIngredient) string {
	if ingredient == nil {
		return ""
	}
	if ingredient.AromaChemicalID != nil {
		return fmt.Sprintf("chem:%d", *ingredient.AromaChemicalID)
	}
	if ingredient.SubFormulaID != nil {
		return fmt.Sprintf("formula:%d", *ingredient.SubFormulaID)
	}
	return ""
}

// FormulaIngredientAmountValue formats the ingredient amount for display in the editor.
func FormulaIngredientAmountValue(ingredient *models.FormulaIngredient) string {
	if ingredient == nil {
		return ""
	}
	return strconv.FormatFloat(ingredient.Amount, 'f', -1, 64)
}

// FormulaIngredientUnitValue returns the unit text for the ingredient row.
func FormulaIngredientUnitValue(ingredient *models.FormulaIngredient) string {
	if ingredient == nil {
		return ""
	}
	return ingredient.Unit
}
