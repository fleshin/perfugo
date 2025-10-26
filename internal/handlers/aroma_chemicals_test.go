package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"perfugo/models"
)

func withAromaTestDatabase(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	original := database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AromaChemical{}, &models.OtherName{}); err != nil {
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

func authenticateRequest(t *testing.T, sm *scs.SessionManager, req *http.Request, userID uint) *http.Request {
	t.Helper()
	ctx, err := sm.Load(req.Context(), "")
	if err != nil {
		t.Fatalf("failed to load session context: %v", err)
	}
	req = req.WithContext(ctx)
	sm.Put(req.Context(), sessionUserIDKey, int(userID))
	sm.Put(req.Context(), sessionAuthenticatedKey, true)
	return req
}

func TestAromaChemicalShowAccessControl(t *testing.T) {
	db, cleanupDB := withAromaTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	viewer := models.User{Email: "viewer@example.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	if err := db.Create(&viewer).Error; err != nil {
		t.Fatalf("failed to create viewer: %v", err)
	}

	chemical := models.AromaChemical{IngredientName: "Test Ingredient", OwnerID: owner.ID, Public: false}
	if err := db.Create(&chemical).Error; err != nil {
		t.Fatalf("failed to create chemical: %v", err)
	}

	// owner can view
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/app/api/aroma-chemicals/%d", chemical.ID), nil)
	req = authenticateRequest(t, sm, req, owner.ID)
	w := httptest.NewRecorder()
	AromaChemicalResource(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for owner, got %d", w.Code)
	}
	var response aromaChemicalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !response.CanEdit || !response.CanDelete {
		t.Fatalf("expected owner to have edit/delete permissions: %+v", response)
	}

	// viewer cannot view private chemical
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/app/api/aroma-chemicals/%d", chemical.ID), nil)
	req = authenticateRequest(t, sm, req, viewer.ID)
	w = httptest.NewRecorder()
	AromaChemicalResource(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unauthorized viewer, got %d", w.Code)
	}

	// mark chemical public and ensure viewer can see but not edit
	if err := db.Model(&chemical).Update("public", true).Error; err != nil {
		t.Fatalf("failed to update chemical public flag: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/app/api/aroma-chemicals/%d", chemical.ID), nil)
	req = authenticateRequest(t, sm, req, viewer.ID)
	w = httptest.NewRecorder()
	AromaChemicalResource(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for public chemical, got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode viewer response: %v", err)
	}
	if response.CanEdit || response.CanDelete {
		t.Fatalf("expected viewer to lack edit/delete permissions: %+v", response)
	}
	if !response.CanCopy {
		t.Fatalf("expected viewer to be able to copy public chemical")
	}
}

func TestAromaChemicalUpdate(t *testing.T) {
	db, cleanupDB := withAromaTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	chemical := models.AromaChemical{IngredientName: "Updatable", OwnerID: owner.ID, Strength: 1}
	if err := db.Create(&chemical).Error; err != nil {
		t.Fatalf("failed to create chemical: %v", err)
	}

	payload := aromaChemicalUpdateRequest{
		IngredientName:      "Updated Name",
		CASNumber:           "123-45-6",
		Notes:               "Updated",
		WheelPosition:       "Citrus",
		PyramidPosition:     "Top",
		Type:                "Synthetic",
		Strength:            4,
		RecommendedDilution: 0.5,
		DilutionPercentage:  0.3,
		MaxIFRAPercentage:   0.2,
		PricePerMg:          0.01,
		Duration:            "Long",
		HistoricRole:        "Classic",
		Popularity:          9,
		Usage:               "Use sparingly",
		Public:              true,
		OtherNames:          []string{"Alias One", "Alias Two"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/app/api/aroma-chemicals/%d", chemical.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = authenticateRequest(t, sm, req, owner.ID)
	w := httptest.NewRecorder()
	AromaChemicalResource(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for update, got %d", w.Code)
	}

	var response aromaChemicalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode update response: %v", err)
	}
	if response.IngredientName != payload.IngredientName || response.CASNumber != payload.CASNumber {
		t.Fatalf("expected updated name/cas, got %+v", response)
	}
	if len(response.OtherNames) != 2 {
		t.Fatalf("expected two other names, got %+v", response.OtherNames)
	}

	var stored models.AromaChemical
	if err := db.Preload("OtherNames").First(&stored, chemical.ID).Error; err != nil {
		t.Fatalf("failed to reload chemical: %v", err)
	}
	if stored.Strength != payload.Strength || stored.Public != payload.Public {
		t.Fatalf("expected stored fields to update, got %+v", stored)
	}
	if len(stored.OtherNames) != 2 {
		t.Fatalf("expected stored other names to update, got %d", len(stored.OtherNames))
	}
}

func TestAromaChemicalUpdateUnauthorized(t *testing.T) {
	db, cleanupDB := withAromaTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	viewer := models.User{Email: "viewer@example.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	if err := db.Create(&viewer).Error; err != nil {
		t.Fatalf("failed to create viewer: %v", err)
	}

	chemical := models.AromaChemical{IngredientName: "Locked", OwnerID: owner.ID}
	if err := db.Create(&chemical).Error; err != nil {
		t.Fatalf("failed to create chemical: %v", err)
	}

	payload := aromaChemicalUpdateRequest{IngredientName: "New Name"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/app/api/aroma-chemicals/%d", chemical.ID),
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req = authenticateRequest(t, sm, req, viewer.ID)

	w := httptest.NewRecorder()
	AromaChemicalResource(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unauthorized update, got %d", w.Code)
	}
}

func TestAromaChemicalCopyAndDelete(t *testing.T) {
	db, cleanupDB := withAromaTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	other := models.User{Email: "other@example.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	chemical := models.AromaChemical{IngredientName: "CopyMe", OwnerID: owner.ID, Public: true, CASNumber: "111-11-1"}
	if err := db.Create(&chemical).Error; err != nil {
		t.Fatalf("failed to create chemical: %v", err)
	}
	if err := db.Model(&chemical).Association("OtherNames").Replace([]models.OtherName{{Name: "Alias"}}); err != nil {
		t.Fatalf("failed to seed other names: %v", err)
	}

	// copy as other user
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/app/api/aroma-chemicals/%d/copy", chemical.ID), nil)
	req = authenticateRequest(t, sm, req, other.ID)
	w := httptest.NewRecorder()
	AromaChemicalResource(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for copy, got %d", w.Code)
	}

	var response aromaChemicalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode copy response: %v", err)
	}
	if response.OwnerID != other.ID {
		t.Fatalf("expected clone owner to match copier, got %d", response.OwnerID)
	}
	if response.Public {
		t.Fatalf("expected clone to be private by default")
	}
	if response.CASNumber != "" {
		t.Fatalf("expected CAS to be cleared to avoid conflicts, got %s", response.CASNumber)
	}
	if len(response.OtherNames) != 1 {
		t.Fatalf("expected copied other names, got %+v", response.OtherNames)
	}

	// delete as owner
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/app/api/aroma-chemicals/%d", chemical.ID), nil)
	deleteReq = authenticateRequest(t, sm, deleteReq, owner.ID)
	deleteW := httptest.NewRecorder()
	AromaChemicalResource(deleteW, deleteReq)
	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for delete, got %d", deleteW.Code)
	}

	// ensure soft deleted
	var count int64
	if err := db.Model(&models.AromaChemical{}).Where("id = ?", chemical.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count chemicals: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted chemical to be excluded from default queries")
	}
	if err := db.WithContext(context.Background()).Unscoped().Model(&models.AromaChemical{}).Where("id = ?", chemical.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count unscoped chemicals: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unscoped count to include deleted record")
	}
}
