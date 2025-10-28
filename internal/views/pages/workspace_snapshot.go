package pages

import (
	"sort"

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

// ChemicalByID returns the aroma chemical with the provided identifier if present.
func (s WorkspaceSnapshot) ChemicalByID(id uint) *models.AromaChemical {
	if id == 0 {
		return nil
	}
	for i := range s.AromaChemicals {
		if uint(s.AromaChemicals[i].ID) == id {
			return &s.AromaChemicals[i]
		}
	}
	return nil
}

// FormulaByID returns the formula with the provided identifier if present.
func (s WorkspaceSnapshot) FormulaByID(id uint) *models.Formula {
	if id == 0 {
		return nil
	}
	for i := range s.Formulas {
		if uint(s.Formulas[i].ID) == id {
			return &s.Formulas[i]
		}
	}
	return nil
}

// IngredientsForFormula returns ingredients that belong to the provided formula.
func (s WorkspaceSnapshot) IngredientsForFormula(id uint) []models.FormulaIngredient {
	if id == 0 {
		return nil
	}
	results := make([]models.FormulaIngredient, 0)
	for _, ingredient := range s.FormulaIngredients {
		if uint(ingredient.FormulaID) == id {
			results = append(results, ingredient)
		}
	}
	return results
}
