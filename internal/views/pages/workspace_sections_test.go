package pages

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"perfugo/models"
)

func TestDefaultDash(t *testing.T) {
	if DefaultDash("value") != "value" {
		t.Fatal("expected non-empty value to pass through")
	}
	if DefaultDash("   ") != "—" {
		t.Fatal("expected whitespace value to produce em dash")
	}
}

func TestAromaChemicalPotencyLabel(t *testing.T) {
	cases := map[int]string{
		8: "Powerful",
		5: "Strong",
		3: "Moderate",
		1: "Delicate",
		0: "Unknown",
	}
	for strength, want := range cases {
		if got := AromaChemicalPotencyLabel(strength); got != want {
			t.Fatalf("potency label for %d: expected %s, got %s", strength, want, got)
		}
	}
}

func TestFilterAromaChemicals(t *testing.T) {
	chemicals := []models.AromaChemical{{IngredientName: "Alpha", CASNumber: "111"}, {IngredientName: "Beta", Type: "Base"}}
	filters := IngredientFilters{Query: "beta"}
	filtered := FilterAromaChemicals(chemicals, filters)
	if len(filtered) != 1 || filtered[0].IngredientName != "Beta" {
		t.Fatalf("expected Beta chemical, got %+v", filtered)
	}
}

func TestFilterFormulas(t *testing.T) {
	formulas := []models.Formula{{Name: "Dawn", Notes: "citrus"}, {Name: "Dusk"}}
	filters := FormulaFilters{Query: "cit"}
	filtered := FilterFormulas(formulas, filters)
	if len(filtered) != 1 || filtered[0].Name != "Dawn" {
		t.Fatalf("expected Dawn formula, got %+v", filtered)
	}
}

func TestFormulaIngredientsFor(t *testing.T) {
	ingredients := []models.FormulaIngredient{{FormulaID: 1, Amount: 2}, {FormulaID: 2, Amount: 3}}
	matches := FormulaIngredientsFor(ingredients, 1)
	if len(matches) != 1 || matches[0].Amount != 2 {
		t.Fatalf("expected ingredient for formula 1, got %+v", matches)
	}
}

func TestPreferenceStatusTemplateRenders(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := PreferenceStatus("Saved").Render(context.Background(), recorder); err != nil {
		t.Fatalf("expected status template to render: %v", err)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "Saved") {
		t.Fatalf("expected rendered status to contain message, got %s", body)
	}
}

func TestNextUntitledFormulaName(t *testing.T) {
	cases := []struct {
		name     string
		existing []models.Formula
		want     string
	}{
		{
			name:     "no existing",
			existing: nil,
			want:     "Untitled Formula",
		},
		{
			name: "fills gaps",
			existing: []models.Formula{
				{Name: "Untitled Formula"},
				{Name: "Untitled Formula 2"},
				{Name: "Citrus Bloom"},
			},
			want: "Untitled Formula 3",
		},
		{
			name: "ignores casing",
			existing: []models.Formula{
				{Name: "untitled formula"},
			},
			want: "Untitled Formula 2",
		},
	}

	for _, tc := range cases {
		if got := NextUntitledFormulaName(tc.existing); got != tc.want {
			t.Fatalf("%s: expected %s, got %s", tc.name, tc.want, got)
		}
	}
}
