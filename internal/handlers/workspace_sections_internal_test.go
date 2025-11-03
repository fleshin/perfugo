package handlers

import (
	"testing"

	"gorm.io/gorm"

	"perfugo/models"
)

func TestWouldCreateFormulaCycle(t *testing.T) {
	u := func(v uint) *uint { return &v }

	formulaA := models.Formula{Model: gorm.Model{ID: 1}, Name: "Formula A"}
	formulaB := models.Formula{
		Model: gorm.Model{ID: 2},
		Name:  "Formula B",
		Ingredients: []models.FormulaIngredient{
			{Model: gorm.Model{ID: 20}, FormulaID: 2, SubFormulaID: u(1)},
		},
	}
	formulaC := models.Formula{
		Model: gorm.Model{ID: 3},
		Name:  "Formula C",
		Ingredients: []models.FormulaIngredient{
			{Model: gorm.Model{ID: 30}, FormulaID: 3, SubFormulaID: u(2)},
		},
	}
	formulaD := models.Formula{Model: gorm.Model{ID: 4}, Name: "Formula D"}

	graph := buildFormulaDependencyGraph([]models.Formula{formulaA, formulaB, formulaC, formulaD})

	if !wouldCreateFormulaCycle(graph, 1, 2) {
		t.Fatalf("expected cycle when adding formula B (contains A) to formula A")
	}
	if !wouldCreateFormulaCycle(graph, 1, 3) {
		t.Fatalf("expected cycle when adding formula C (contains B -> A) to formula A")
	}
	if wouldCreateFormulaCycle(graph, 1, 4) {
		t.Fatalf("did not expect cycle when adding unrelated formula D to formula A")
	}
	if !wouldCreateFormulaCycle(graph, 1, 1) {
		t.Fatalf("expected cycle when referencing the same formula")
	}
}
