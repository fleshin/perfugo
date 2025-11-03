package handlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"perfugo/internal/ai"
	"perfugo/models"
)

func newToolsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:tools-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.AromaChemical{},
		&models.OtherName{},
		&models.Formula{},
		&models.FormulaIngredient{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestReplaceOtherNamesReplacesExistingEntries(t *testing.T) {
	ctx := context.Background()
	db := newToolsTestDB(t)

	chemical := models.AromaChemical{
		IngredientName: "Test Aromatic",
		OwnerID:        777,
	}
	if err := db.WithContext(ctx).Create(&chemical).Error; err != nil {
		t.Fatalf("create chemical: %v", err)
	}

	initial := []models.OtherName{
		{Name: "Alpha", AromaChemicalID: chemical.ID},
		{Name: "Beta", AromaChemicalID: chemical.ID},
	}
	if err := db.WithContext(ctx).Create(&initial).Error; err != nil {
		t.Fatalf("seed other names: %v", err)
	}

	if err := replaceOtherNames(ctx, db, chemical.ID, []string{" Gamma ", "delta", "gamma"}); err != nil {
		t.Fatalf("replace other names: %v", err)
	}

	var stored []models.OtherName
	if err := db.WithContext(ctx).
		Where("aroma_chemical_id = ?", chemical.ID).
		Find(&stored).Error; err != nil {
		t.Fatalf("load other names: %v", err)
	}

	if len(stored) != 2 {
		t.Fatalf("expected 2 other names, got %d", len(stored))
	}

	expected := map[string]struct{}{
		"Gamma": {},
		"delta": {},
	}
	for _, name := range stored {
		if _, ok := expected[name.Name]; !ok {
			t.Fatalf("unexpected other name stored: %q", name.Name)
		}
		delete(expected, name.Name)
	}
	if len(expected) != 0 {
		t.Fatalf("expected other names missing from results: %v", expected)
	}
}

func TestPersistAromaProfileCanonicalisesData(t *testing.T) {
	ctx := context.Background()
	db := newToolsTestDB(t)
	Configure(nil, db)
	t.Cleanup(func() {
		database = nil
		sessionManager = nil
	})

	ownerID := uint(4242)
	profile := ai.Profile{
		IngredientName:      "Celestial Musk",
		CASNumber:           "999-99-9",
		PyramidPosition:     "Heart Base",
		WheelPosition:       " Floral ",
		OtherNames:          []string{" Aurora ", "aurora"},
		Strength:            4,
		RecommendedDilution: 12.5,
	}

	record, created, err := persistAromaProfile(ctx, profile, ownerID)
	if err != nil {
		t.Fatalf("persist profile: %v", err)
	}
	if !created {
		t.Fatalf("expected new record to be created")
	}
	if record.OwnerID != ownerID {
		t.Fatalf("expected owner id %d, got %d", ownerID, record.OwnerID)
	}
	if record.Public {
		t.Fatalf("expected private record")
	}
	if record.PyramidPosition != "heart-base" {
		t.Fatalf("expected canonical pyramid position, got %q", record.PyramidPosition)
	}
	if record.WheelPosition != "Floral" {
		t.Fatalf("expected trimmed wheel position, got %q", record.WheelPosition)
	}
	if len(record.OtherNames) != 1 || record.OtherNames[0].Name != "Aurora" {
		t.Fatalf("expected deduplicated other names, got %+v", record.OtherNames)
	}
	if record.RecommendedDilution != 12.5 {
		t.Fatalf("expected recommended dilution 12.5, got %.2f", record.RecommendedDilution)
	}
}
