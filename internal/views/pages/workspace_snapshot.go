package pages

import (
	"encoding/json"
	"sort"

	"perfugo/models"
)

// WorkspaceSnapshot aggregates relational data required to render the atelier workspace.
type WorkspaceSnapshot struct {
	Formulas           []models.Formula
	FormulaIngredients []models.FormulaIngredient
	AromaChemicals     []models.AromaChemical
}

// NewWorkspaceSnapshot normalises and sorts the data required by the workspace views.
func NewWorkspaceSnapshot(formulas []models.Formula, ingredients []models.FormulaIngredient, chemicals []models.AromaChemical) WorkspaceSnapshot {
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
	}
}

// EmptyWorkspaceSnapshot returns a zero-value snapshot to simplify call sites when no data is available.
func EmptyWorkspaceSnapshot() WorkspaceSnapshot {
	return WorkspaceSnapshot{}
}

// SeedsJSON encodes a subset of the snapshot so that the front-end modules can simulate CRUD interactions.
func (s WorkspaceSnapshot) SeedsJSON() string {
	payload := struct {
		Formulas           []models.Formula           `json:"formulas"`
		FormulaIngredients []models.FormulaIngredient `json:"formula_ingredients"`
		AromaChemicals     []models.AromaChemical     `json:"aroma_chemicals"`
	}{
		Formulas:           s.Formulas,
		FormulaIngredients: s.FormulaIngredients,
		AromaChemicals:     s.AromaChemicals,
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(bytes)
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
