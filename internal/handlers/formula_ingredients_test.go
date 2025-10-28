package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"perfugo/models"
)

func withFormulaIngredientTestDatabase(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	original := database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AromaChemical{}, &models.Formula{}, &models.FormulaIngredient{}); err != nil {
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

func TestFormulaIngredientCRUD(t *testing.T) {
	db, cleanupDB := withFormulaIngredientTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	user := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	formula := models.Formula{Name: "Test Formula", Notes: "Notes", Version: 1, IsLatest: true}
	if err := db.Create(&formula).Error; err != nil {
		t.Fatalf("failed to create formula: %v", err)
	}

	chemical := models.AromaChemical{IngredientName: "Citrus", OwnerID: user.ID, Public: true}
	if err := db.Create(&chemical).Error; err != nil {
		t.Fatalf("failed to create aroma chemical: %v", err)
	}

	// Create ingredient
	createPayload := map[string]any{
		"formula_id":        formula.ID,
		"aroma_chemical_id": chemical.ID,
		"amount":            2.5,
		"unit":              "g",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/app/api/formula-ingredients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = authenticateRequest(t, sm, req, user.ID)
	w := httptest.NewRecorder()
	FormulaIngredientResource(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var created formulaIngredientResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.FormulaID != formula.ID || created.AromaChemicalID == nil || *created.AromaChemicalID != chemical.ID {
		t.Fatalf("unexpected create response: %+v", created)
	}
	if created.AromaChemical == nil || created.AromaChemical.IngredientName != chemical.IngredientName {
		t.Fatalf("expected aroma chemical details in response: %+v", created)
	}

	// List ingredients
	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/app/api/formula-ingredients?formula_id=%d", formula.ID), nil)
	listReq = authenticateRequest(t, sm, listReq, user.ID)
	listW := httptest.NewRecorder()
	FormulaIngredientResource(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for list, got %d", listW.Code)
	}
	var listResponse []formulaIngredientResponse
	if err := json.Unmarshal(listW.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(listResponse) != 1 || listResponse[0].ID != created.ID {
		t.Fatalf("expected one ingredient in list, got %+v", listResponse)
	}

	// Update ingredient
	updatePayload := map[string]any{
		"formula_id":        formula.ID,
		"aroma_chemical_id": chemical.ID,
		"amount":            4.0,
		"unit":              "ml",
	}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/app/api/formula-ingredients/%d", created.ID), bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = authenticateRequest(t, sm, updateReq, user.ID)
	updateW := httptest.NewRecorder()
	FormulaIngredientResource(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for update, got %d", updateW.Code)
	}
	var updated formulaIngredientResponse
	if err := json.Unmarshal(updateW.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode update response: %v", err)
	}
	if updated.Amount != 4.0 || updated.Unit != "ml" {
		t.Fatalf("expected updated amount/unit, got %+v", updated)
	}

	// Delete ingredient
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/app/api/formula-ingredients/%d", created.ID), nil)
	deleteReq = authenticateRequest(t, sm, deleteReq, user.ID)
	deleteW := httptest.NewRecorder()
	FormulaIngredientResource(deleteW, deleteReq)
	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for delete, got %d", deleteW.Code)
	}

	var count int64
	if err := db.Model(&models.FormulaIngredient{}).Where("id = ?", created.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count ingredients: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected ingredient to be deleted, count=%d", count)
	}
}
