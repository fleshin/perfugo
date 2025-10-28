package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestAromaChemicalDetailRendersHTMLForOwner(t *testing.T) {
	db, cleanupDB := withAromaTestDatabase(t)
	t.Cleanup(cleanupDB)
	sm, cleanupSession := withTestSessionManager(t)
	t.Cleanup(cleanupSession)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}

	chemical := models.AromaChemical{IngredientName: "Galbanum", OwnerID: owner.ID, Public: true, Strength: 4}
	if err := db.Create(&chemical).Error; err != nil {
		t.Fatalf("failed to create chemical: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/app/htmx/ingredients/%d", chemical.ID), nil)
	req = authenticateRequest(t, sm, req, owner.ID)
	w := httptest.NewRecorder()

	AromaChemicalDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "Galbanum") {
		t.Fatalf("expected response to include chemical name, body=%s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Potency") {
		t.Fatalf("expected response to include potency label, body=%s", w.Body.String())
	}
}

func TestAromaChemicalDetailRespectsOwnership(t *testing.T) {
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

	chemical := models.AromaChemical{IngredientName: "Iso E Super", OwnerID: owner.ID, Public: false}
	if err := db.Create(&chemical).Error; err != nil {
		t.Fatalf("failed to create chemical: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/app/htmx/ingredients/%d", chemical.ID), nil)
	req = authenticateRequest(t, sm, req, viewer.ID)
	w := httptest.NewRecorder()

	AromaChemicalDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unauthorized viewer, got %d", w.Code)
	}
}
