package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"perfugo/models"
)

func withFormulaTestDatabase(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	original := database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Formula{}, &models.FormulaIngredient{}, &models.AromaChemical{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}
	database = db
	return db, func() {
		database = original
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

func TestFormulaDetailRendersComposition(t *testing.T) {
	db, cleanupDB := withFormulaTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	user := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	formula := models.Formula{Name: "Aurum", Version: 2, IsLatest: true}
	if err := db.Create(&formula).Error; err != nil {
		t.Fatalf("failed to create formula: %v", err)
	}
	aroma := models.AromaChemical{IngredientName: "Ambroxan", OwnerID: user.ID, Public: true}
	if err := db.Create(&aroma).Error; err != nil {
		t.Fatalf("failed to create aroma chemical: %v", err)
	}
	ingredient := models.FormulaIngredient{FormulaID: formula.ID, AromaChemicalID: &aroma.ID, Amount: 12.5, Unit: "g"}
	if err := db.Create(&ingredient).Error; err != nil {
		t.Fatalf("failed to create ingredient: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/app/htmx/formulas/%d", formula.ID), nil)
	req = authenticateRequest(t, sm, req, user.ID)
	w := httptest.NewRecorder()

	FormulaDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Aurum") {
		t.Fatalf("expected formula name in response, body=%s", body)
	}
	if !strings.Contains(body, "Ambroxan") {
		t.Fatalf("expected ingredient name in response, body=%s", body)
	}
	if !strings.Contains(body, "12.50 g") {
		t.Fatalf("expected formatted amount in response, body=%s", body)
	}
}

func TestFormulaDetailMissingRecord(t *testing.T) {
	_, cleanupDB := withFormulaTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	req := httptest.NewRequest(http.MethodGet, "/app/htmx/formulas/999", nil)
	req = authenticateRequest(t, sm, req, 1)
	w := httptest.NewRecorder()

	FormulaDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing formula, got %d", w.Code)
	}
}
