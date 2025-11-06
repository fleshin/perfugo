package handlers

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"perfugo/models"
)

func TestBuildBatchProductionReportDataScalesAndConsolidates(t *testing.T) {
	ctx := context.Background()
	db := newToolsTestDB(t)

	prevDB := database
	database = db
	t.Cleanup(func() { database = prevDB })

	fixedNow := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
	prevNowFunc := nowFunc
	nowFunc = func() time.Time { return fixedNow }
	t.Cleanup(func() { nowFunc = prevNowFunc })

	baseChemical := models.AromaChemical{
		IngredientName:  "Amber Core",
		CASNumber:       "123-45-6",
		PyramidPosition: "base",
		OwnerID:         1,
		Public:          true,
	}
	topChemical := models.AromaChemical{
		IngredientName:  "Citrus Lift",
		CASNumber:       "654-32-1",
		PyramidPosition: "top",
		OwnerID:         1,
		Public:          true,
	}
	if err := db.WithContext(ctx).Create(&baseChemical).Error; err != nil {
		t.Fatalf("create base chemical: %v", err)
	}
	if err := db.WithContext(ctx).Create(&topChemical).Error; err != nil {
		t.Fatalf("create top chemical: %v", err)
	}

	subFormula := models.Formula{Name: "Bridge Accord", Version: 1, IsLatest: true}
	parentFormula := models.Formula{Name: "Auric Essence", Version: 2, IsLatest: true}
	if err := db.WithContext(ctx).Create(&subFormula).Error; err != nil {
		t.Fatalf("create sub formula: %v", err)
	}
	if err := db.WithContext(ctx).Create(&parentFormula).Error; err != nil {
		t.Fatalf("create parent formula: %v", err)
	}

	if err := db.WithContext(ctx).Create(&models.FormulaIngredient{
		FormulaID:       subFormula.ID,
		AromaChemicalID: &baseChemical.ID,
		Amount:          3,
		Unit:            "g",
	}).Error; err != nil {
		t.Fatalf("create sub ingredient base: %v", err)
	}
	if err := db.WithContext(ctx).Create(&models.FormulaIngredient{
		FormulaID:       subFormula.ID,
		AromaChemicalID: &topChemical.ID,
		Amount:          2,
		Unit:            "g",
	}).Error; err != nil {
		t.Fatalf("create sub ingredient top: %v", err)
	}
	if err := db.WithContext(ctx).Create(&models.FormulaIngredient{
		FormulaID:       parentFormula.ID,
		AromaChemicalID: &baseChemical.ID,
		Amount:          10,
		Unit:            "g",
	}).Error; err != nil {
		t.Fatalf("create parent base ingredient: %v", err)
	}
	if err := db.WithContext(ctx).Create(&models.FormulaIngredient{
		FormulaID:    parentFormula.ID,
		SubFormulaID: &subFormula.ID,
		Amount:       5,
		Unit:         "g",
	}).Error; err != nil {
		t.Fatalf("create parent subformula ingredient: %v", err)
	}

	report, err := buildBatchProductionReportData(ctx, parentFormula.ID, 30)
	if err != nil {
		t.Fatalf("buildBatchProductionReportData returned error: %v", err)
	}

	if report.TargetQuantity != 30 {
		t.Fatalf("expected target quantity 30, got %.2f", report.TargetQuantity)
	}
	if !report.RunDate.Equal(fixedNow) {
		t.Fatalf("expected run date %v, got %v", fixedNow, report.RunDate)
	}
	if report.BaseBatchQuantity != 15 {
		t.Fatalf("expected base batch quantity 15, got %.2f", report.BaseBatchQuantity)
	}
	if math.Abs(report.ScaleFactor-2.0) > 1e-6 {
		t.Fatalf("expected scale factor 2, got %.4f", report.ScaleFactor)
	}
	if len(report.Ingredients) != 2 {
		t.Fatalf("expected 2 ingredients, got %d", len(report.Ingredients))
	}
	if report.Ingredients[0].IngredientName != "Amber Core" {
		t.Fatalf("expected Amber Core first, got %s", report.Ingredients[0].IngredientName)
	}
	if math.Abs(report.Ingredients[0].FinalQuantity-26.0) > 1e-6 {
		t.Fatalf("expected Amber quantity 26, got %.4f", report.Ingredients[0].FinalQuantity)
	}
	if report.Ingredients[1].IngredientName != "Citrus Lift" {
		t.Fatalf("expected Citrus Lift second, got %s", report.Ingredients[1].IngredientName)
	}
	if math.Abs(report.Ingredients[1].FinalQuantity-4.0) > 1e-6 {
		t.Fatalf("expected Citrus quantity 4, got %.4f", report.Ingredients[1].FinalQuantity)
	}
	if !strings.HasPrefix(report.LotNumber, "PERF-") {
		t.Fatalf("expected lot number with PERF- prefix, got %s", report.LotNumber)
	}
}
