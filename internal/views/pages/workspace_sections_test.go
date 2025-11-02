package pages

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"
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
	chemicals := []models.AromaChemical{
		{IngredientName: "Alpha", CASNumber: "111", PyramidPosition: "Top", WheelPosition: "Citrus"},
		{IngredientName: "Beta", Type: "Base", PyramidPosition: "Base", WheelPosition: "Woody"},
	}
	f := IngredientFilters{Query: "beta"}
	filtered := FilterAromaChemicals(chemicals, f)
	if len(filtered) != 1 || filtered[0].IngredientName != "Beta" {
		t.Fatalf("expected Beta chemical, got %+v", filtered)
	}

	f = IngredientFilters{Pyramid: "base"}
	filtered = FilterAromaChemicals(chemicals, f)
	if len(filtered) != 1 || filtered[0].PyramidPosition != "Base" {
		t.Fatalf("expected pyramid filter to match Base, got %+v", filtered)
	}

	f = IngredientFilters{Wheel: "woody"}
	filtered = FilterAromaChemicals(chemicals, f)
	if len(filtered) != 1 || strings.ToLower(filtered[0].WheelPosition) != "woody" {
		t.Fatalf("expected wheel filter to match Woody, got %+v", filtered)
	}

	f = IngredientFilters{Pyramid: "heart", Wheel: "citrus"}
	filtered = FilterAromaChemicals(chemicals, f)
	if len(filtered) != 0 {
		t.Fatalf("expected no results for unmatched combo, got %+v", filtered)
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

func TestNextCopiedFormulaName(t *testing.T) {
	base := "Aurora"
	existing := []models.Formula{
		{Name: "Aurora"},
		{Name: "Aurora (Copy)"},
		{Name: "Aurora (Copy 2)"},
	}
	if got := NextCopiedFormulaName(existing, base); got != "Aurora (Copy 3)" {
		t.Fatalf("expected unique copy name, got %s", got)
	}

	if got := NextCopiedFormulaName(existing, "Celeste"); got != "Celeste (Copy)" {
		t.Fatalf("expected first copy suffix, got %s", got)
	}

	if got := NextCopiedFormulaName(existing, "   "); got != "Untitled Formula" {
		t.Fatalf("expected fallback to untitled, got %s", got)
	}
}

func TestNormalizePyramidPosition(t *testing.T) {
	cases := map[string]struct {
		input string
		want  string
		valid bool
	}{
		"blank":      {input: "", want: "", valid: true},
		"canonical":  {input: "heart", want: "heart", valid: true},
		"uppercase":  {input: "TOP", want: "top", valid: true},
		"spaced":     {input: "Heart Base", want: "heart-base", valid: true},
		"underscore": {input: "top_heart", want: "top-heart", valid: true},
		"invalid":    {input: "mid", want: "", valid: false},
	}

	for name, tc := range cases {
		got, ok := NormalizePyramidPosition(tc.input)
		if got != tc.want || ok != tc.valid {
			t.Fatalf("%s: expected (%q,%t), got (%q,%t)", name, tc.want, tc.valid, got, ok)
		}
	}
}

func TestPyramidPositionLabel(t *testing.T) {
	if got := PyramidPositionLabel("heart-base"); got != "Heart-Base" {
		t.Fatalf("expected Heart-Base, got %s", got)
	}
	if got := PyramidPositionLabel(""); got != "—" {
		t.Fatalf("expected em dash for blank, got %s", got)
	}
	if got := PyramidPositionLabel("unknown"); got != "—" {
		t.Fatalf("expected em dash for invalid, got %s", got)
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := FormatPercentage(0); got != "—" {
		t.Fatalf("expected dash for zero percentage, got %s", got)
	}
	if got := FormatPercentage(12.345); got != "12.35%" {
		t.Fatalf("expected formatted percentage, got %s", got)
	}
	if got := FormatPricePerMg(0); got != "—" {
		t.Fatalf("expected dash for zero price, got %s", got)
	}
	if got := FormatPricePerMg(1.2345); got != "$1.2345" {
		t.Fatalf("expected formatted price, got %s", got)
	}
	if got := FormatPopularity(0); got != "—" {
		t.Fatalf("expected dash for zero popularity, got %s", got)
	}
	if got := FormatPopularity(7); got != "7" {
		t.Fatalf("expected numeric popularity, got %s", got)
	}
	if got := FormatFloatInput(0, 2); got != "" {
		t.Fatalf("expected empty input for zero float, got %q", got)
	}
	if got := FormatFloatInput(1.2345, 2); got != "1.23" {
		t.Fatalf("expected formatted float input, got %s", got)
	}
	if got := FormatIntInput(0); got != "" {
		t.Fatalf("expected empty input for zero int, got %q", got)
	}
	if got := FormatIntInput(3); got != "3" {
		t.Fatalf("expected formatted int input, got %s", got)
	}
	if got := IngredientFormAction(nil); got != "/app/sections/ingredients/create" {
		t.Fatalf("expected create action, got %s", got)
	}
	if got := IngredientFormAction(&models.AromaChemical{Model: gorm.Model{ID: 5}}); got != "/app/sections/ingredients/update" {
		t.Fatalf("expected update action, got %s", got)
	}
}

func TestFormulaEditorSelectsEachIngredientSource(t *testing.T) {
	u := func(v uint) *uint { return &v }

	formula := models.Formula{Model: gorm.Model{ID: 42}, Name: "Celestial Blend"}
	chemicals := []models.AromaChemical{
		{Model: gorm.Model{ID: 1}, IngredientName: "Ambroxan"},
		{Model: gorm.Model{ID: 2}, IngredientName: "Bergamot"},
	}
	ingredients := []models.FormulaIngredient{
		{
			Model:           gorm.Model{ID: 101},
			FormulaID:       formula.ID,
			AromaChemicalID: u(1),
			AromaChemical:   &chemicals[0],
			Amount:          10,
			Unit:            "g",
		},
		{
			Model:           gorm.Model{ID: 102},
			FormulaID:       formula.ID,
			AromaChemicalID: u(2),
			AromaChemical:   &chemicals[1],
			Amount:          5,
			Unit:            "g",
		},
	}

	var buf bytes.Buffer
	err := FormulaEditor(&formula, ingredients, chemicals, []models.Formula{formula}, "").Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render formula editor: %v", err)
	}

	html := buf.String()
	if strings.Count(html, `value="chem:1" selected`) != 1 {
		t.Fatalf("expected chem:1 option selected once, got html: %s", html)
	}
	if strings.Count(html, `value="chem:2" selected`) != 1 {
		t.Fatalf("expected chem:2 option selected once, got html: %s", html)
	}
}
