package pages

import (
	"encoding/json"
	"testing"

	"perfugo/models"

	"gorm.io/gorm"
)

func TestNewWorkspaceSnapshotSortsCollections(t *testing.T) {
	formulas := []models.Formula{{Model: gorm.Model{ID: 2}, Name: "B"}, {Model: gorm.Model{ID: 1}, Name: "A"}}
	ingredients := []models.FormulaIngredient{
		{Model: gorm.Model{ID: 2}, FormulaID: 2},
		{Model: gorm.Model{ID: 1}, FormulaID: 1},
		{Model: gorm.Model{ID: 3}, FormulaID: 1},
	}
	chemicals := []models.AromaChemical{
		{Model: gorm.Model{ID: 2}, IngredientName: "Z"},
		{Model: gorm.Model{ID: 1}, IngredientName: "A"},
	}

	snapshot := NewWorkspaceSnapshot(formulas, ingredients, chemicals, models.ThemeNocturne)

	if snapshot.Formulas[0].Name != "A" {
		t.Fatalf("expected formulas to be sorted by name: %v", snapshot.Formulas)
	}
	if snapshot.FormulaIngredients[0].ID != 1 || snapshot.FormulaIngredients[1].ID != 3 {
		t.Fatalf("expected ingredients sorted by formula then id: %v", snapshot.FormulaIngredients)
	}
	if snapshot.AromaChemicals[0].IngredientName != "A" {
		t.Fatalf("expected aroma chemicals sorted alphabetically: %v", snapshot.AromaChemicals)
	}
}

func TestWorkspaceSnapshotSeedsJSON(t *testing.T) {
	snapshot := WorkspaceSnapshot{
		Formulas:           []models.Formula{{Name: "F"}},
		FormulaIngredients: []models.FormulaIngredient{{Amount: 1}},
		AromaChemicals:     []models.AromaChemical{{IngredientName: "C"}},
	}

	data := snapshot.SeedsJSON()
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		t.Fatalf("expected valid json payload, got %v", err)
	}
	if _, ok := parsed["formulas"]; !ok {
		t.Fatalf("expected formulas key in seeds json: %s", data)
	}
}

func TestEmptyWorkspaceSnapshotUsesDefaultTheme(t *testing.T) {
	snap := EmptyWorkspaceSnapshot()
	if snap.Theme != models.DefaultTheme {
		t.Fatalf("expected default theme %s, got %s", models.DefaultTheme, snap.Theme)
	}
}

func TestIngredientDisplayNamePrefersLinkedEntities(t *testing.T) {
	ingredient := models.FormulaIngredient{AromaChemical: &models.AromaChemical{IngredientName: "Named"}}
	if got := IngredientDisplayName(ingredient); got != "Named" {
		t.Fatalf("expected aroma chemical name, got %s", got)
	}

	ingredient = models.FormulaIngredient{SubFormula: &models.Formula{Name: "Blend"}}
	if got := IngredientDisplayName(ingredient); got != "Blend" {
		t.Fatalf("expected sub-formula name fallback, got %s", got)
	}

	ingredient = models.FormulaIngredient{}
	if got := IngredientDisplayName(ingredient); got != "Unassigned Ingredient" {
		t.Fatalf("expected default label for missing associations, got %s", got)
	}
}

func TestIngredientSourceKind(t *testing.T) {
	ingredient := models.FormulaIngredient{AromaChemicalID: ptr(uint(1))}
	if got := IngredientSourceKind(ingredient); got != "Aroma Chemical" {
		t.Fatalf("expected aroma chemical source, got %s", got)
	}

	ingredient = models.FormulaIngredient{SubFormulaID: ptr(uint(2))}
	if got := IngredientSourceKind(ingredient); got != "Sub-Formula" {
		t.Fatalf("expected sub-formula source, got %s", got)
	}

	if got := IngredientSourceKind(models.FormulaIngredient{}); got != "Unassigned" {
		t.Fatalf("expected unassigned source, got %s", got)
	}
}

func TestFormulaLookupContainsEntries(t *testing.T) {
	formulas := []models.Formula{{Model: gorm.Model{ID: 1}, Name: "First"}}
	snapshot := WorkspaceSnapshot{Formulas: formulas}
	lookup := snapshot.FormulaLookup()
	if lookup[1] != "First" {
		t.Fatalf("expected lookup to return formula name, got %s", lookup[1])
	}
}

func ptr[T any](value T) *T {
	return &value
}
