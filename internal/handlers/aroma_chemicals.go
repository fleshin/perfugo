package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/models"
)

type aromaChemicalResponse struct {
	ID                  uint      `json:"id"`
	IngredientName      string    `json:"ingredient_name"`
	CASNumber           string    `json:"cas_number"`
	Notes               string    `json:"notes"`
	WheelPosition       string    `json:"wheel_position"`
	PyramidPosition     string    `json:"pyramid_position"`
	Type                string    `json:"type"`
	Strength            int       `json:"strength"`
	RecommendedDilution float64   `json:"recommended_dilution"`
	DilutionPercentage  float64   `json:"dilution_percentage"`
	MaxIFRAPercentage   float64   `json:"max_ifra_percentage"`
	PricePerMg          float64   `json:"price_per_mg"`
	Duration            string    `json:"duration"`
	HistoricRole        string    `json:"historic_role"`
	Popularity          int       `json:"popularity"`
	Usage               string    `json:"usage"`
	OwnerID             uint      `json:"owner_id"`
	Public              bool      `json:"public"`
	OtherNames          []string  `json:"other_names"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	CanEdit             bool      `json:"can_edit"`
	CanDelete           bool      `json:"can_delete"`
	CanCopy             bool      `json:"can_copy"`
}

type aromaChemicalUpdateRequest struct {
	IngredientName      string   `json:"ingredient_name"`
	CASNumber           string   `json:"cas_number"`
	Notes               string   `json:"notes"`
	WheelPosition       string   `json:"wheel_position"`
	PyramidPosition     string   `json:"pyramid_position"`
	Type                string   `json:"type"`
	Strength            int      `json:"strength"`
	RecommendedDilution float64  `json:"recommended_dilution"`
	DilutionPercentage  float64  `json:"dilution_percentage"`
	MaxIFRAPercentage   float64  `json:"max_ifra_percentage"`
	PricePerMg          float64  `json:"price_per_mg"`
	Duration            string   `json:"duration"`
	HistoricRole        string   `json:"historic_role"`
	Popularity          int      `json:"popularity"`
	Usage               string   `json:"usage"`
	Public              bool     `json:"public"`
	OtherNames          []string `json:"other_names"`
}

// AromaChemicalResource handles REST-style interactions for aroma chemical records.
func AromaChemicalResource(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		applog.Debug(r.Context(), "aroma chemical request without database")
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	userID, ok := currentUserID(r)
	if !ok {
		applog.Debug(r.Context(), "aroma chemical request missing authenticated user")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/app/api/aroma-chemicals")
	path = strings.Trim(path, "/")

	if path == "" {
		if r.Method == http.MethodGet {
			listAromaChemicals(w, r, userID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	segments := strings.Split(path, "/")
	identifier := segments[0]
	idValue, err := strconv.ParseUint(identifier, 10, 64)
	if err != nil {
		applog.Debug(r.Context(), "invalid aroma chemical identifier", "identifier", identifier, "error", err)
		http.NotFound(w, r)
		return
	}
	chemicalID := uint(idValue)

	if len(segments) > 1 && segments[1] == "copy" {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		copyAromaChemical(w, r, chemicalID, userID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		showAromaChemical(w, r, chemicalID, userID)
	case http.MethodPut:
		updateAromaChemical(w, r, chemicalID, userID)
	case http.MethodDelete:
		deleteAromaChemical(w, r, chemicalID, userID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func listAromaChemicals(w http.ResponseWriter, r *http.Request, userID uint) {
	ctx := r.Context()
	var results []models.AromaChemical
	query := database.WithContext(ctx).
		Preload("OtherNames").
		Order("ingredient_name asc")
	query = query.Where("owner_id = ? OR public = ?", userID, true)
	if err := query.Find(&results).Error; err != nil {
		applog.Error(ctx, "failed to list aroma chemicals", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "unable to load aroma chemicals")
		return
	}

	responses := make([]aromaChemicalResponse, 0, len(results))
	for _, chemical := range results {
		responses = append(responses, projectAromaChemical(chemical, userID))
	}
	writeJSON(w, http.StatusOK, responses)
}

func showAromaChemical(w http.ResponseWriter, r *http.Request, chemicalID, userID uint) {
	ctx := r.Context()
	var chemical models.AromaChemical
	if err := database.WithContext(ctx).Preload("OtherNames").First(&chemical, chemicalID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			applog.Debug(ctx, "aroma chemical not found", "id", chemicalID)
			http.NotFound(w, r)
			return
		}
		applog.Error(ctx, "failed to load aroma chemical", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load aroma chemical")
		return
	}

	if chemical.OwnerID != userID && !chemical.Public {
		applog.Debug(ctx, "aroma chemical access denied", "id", chemicalID, "owner", chemical.OwnerID, "user", userID)
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, projectAromaChemical(chemical, userID))
}

func updateAromaChemical(w http.ResponseWriter, r *http.Request, chemicalID, userID uint) {
	ctx := r.Context()
	var chemical models.AromaChemical
	if err := database.WithContext(ctx).Preload("OtherNames").Where("id = ? AND owner_id = ?", chemicalID, userID).First(&chemical).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			applog.Debug(ctx, "update denied: aroma chemical not found or not owned", "id", chemicalID, "user", userID)
			http.NotFound(w, r)
			return
		}
		applog.Error(ctx, "failed to load aroma chemical for update", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load aroma chemical")
		return
	}

	var payload aromaChemicalUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		applog.Debug(ctx, "invalid aroma chemical update payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	trimmedName := strings.TrimSpace(payload.IngredientName)
	if trimmedName == "" {
		writeJSONError(w, http.StatusBadRequest, "ingredient_name is required")
		return
	}

	updates := map[string]any{
		"ingredient_name":      trimmedName,
		"cas_number":           strings.TrimSpace(payload.CASNumber),
		"notes":                strings.TrimSpace(payload.Notes),
		"wheel_position":       strings.TrimSpace(payload.WheelPosition),
		"pyramid_position":     strings.TrimSpace(payload.PyramidPosition),
		"type":                 strings.TrimSpace(payload.Type),
		"strength":             payload.Strength,
		"recommended_dilution": payload.RecommendedDilution,
		"dilution_percentage":  payload.DilutionPercentage,
		"max_ifra_percentage":  payload.MaxIFRAPercentage,
		"price_per_mg":         payload.PricePerMg,
		"duration":             strings.TrimSpace(payload.Duration),
		"historic_role":        strings.TrimSpace(payload.HistoricRole),
		"popularity":           payload.Popularity,
		"usage":                strings.TrimSpace(payload.Usage),
		"public":               payload.Public,
	}

	if err := database.WithContext(ctx).Model(&chemical).Updates(updates).Error; err != nil {
		applog.Error(ctx, "failed to update aroma chemical", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("update failed: %v", err))
		return
	}

	if err := replaceOtherNames(ctx, &chemical, payload.OtherNames); err != nil {
		applog.Error(ctx, "failed to update aroma chemical other names", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to update other names")
		return
	}

	if err := database.WithContext(ctx).Preload("OtherNames").First(&chemical, chemicalID).Error; err != nil {
		applog.Error(ctx, "failed to reload aroma chemical after update", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load updated record")
		return
	}

	writeJSON(w, http.StatusOK, projectAromaChemical(chemical, userID))
}

func deleteAromaChemical(w http.ResponseWriter, r *http.Request, chemicalID, userID uint) {
	ctx := r.Context()
	var chemical models.AromaChemical
	if err := database.WithContext(ctx).Where("id = ? AND owner_id = ?", chemicalID, userID).First(&chemical).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			applog.Debug(ctx, "delete denied: aroma chemical not found or not owned", "id", chemicalID, "user", userID)
			http.NotFound(w, r)
			return
		}
		applog.Error(ctx, "failed to load aroma chemical for delete", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load aroma chemical")
		return
	}

	if err := database.WithContext(ctx).Delete(&chemical).Error; err != nil {
		applog.Error(ctx, "failed to soft delete aroma chemical", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to delete aroma chemical")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func copyAromaChemical(w http.ResponseWriter, r *http.Request, chemicalID, userID uint) {
	ctx := r.Context()
	var source models.AromaChemical
	if err := database.WithContext(ctx).Preload("OtherNames").First(&source, chemicalID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			applog.Debug(ctx, "copy failed: aroma chemical not found", "id", chemicalID)
			http.NotFound(w, r)
			return
		}
		applog.Error(ctx, "failed to load aroma chemical for copy", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load aroma chemical")
		return
	}

	if source.OwnerID != userID && !source.Public {
		applog.Debug(ctx, "copy denied: aroma chemical not accessible", "id", chemicalID, "owner", source.OwnerID, "user", userID)
		http.NotFound(w, r)
		return
	}

	clone := source
	clone.Model = gorm.Model{}
	clone.OwnerID = userID
	clone.Public = false
	clone.CASNumber = strings.TrimSpace(clone.CASNumber)

	// reset CAS if duplicate would violate unique constraint
	if clone.CASNumber != "" {
		var count int64
		if err := database.WithContext(ctx).Model(&models.AromaChemical{}).Where("cas_number = ?", clone.CASNumber).Count(&count).Error; err == nil && count > 0 {
			clone.CASNumber = ""
		}
	}

	clone.IngredientName = nextAvailableName(ctx, source.IngredientName)

	if err := database.WithContext(ctx).Create(&clone).Error; err != nil {
		applog.Error(ctx, "failed to create aroma chemical copy", "error", err, "id", chemicalID)
		writeJSONError(w, http.StatusInternalServerError, "unable to copy aroma chemical")
		return
	}

	names := make([]models.OtherName, 0, len(source.OtherNames))
	for _, alias := range source.OtherNames {
		trimmed := strings.TrimSpace(alias.Name)
		if trimmed == "" {
			continue
		}
		names = append(names, models.OtherName{Name: trimmed})
	}

	if len(names) > 0 {
		if err := database.WithContext(ctx).Model(&clone).Association("OtherNames").Replace(names); err != nil {
			applog.Error(ctx, "failed to copy aroma chemical other names", "error", err, "id", chemicalID)
			writeJSONError(w, http.StatusInternalServerError, "unable to copy aroma chemical")
			return
		}
	}

	if err := database.WithContext(ctx).Preload("OtherNames").First(&clone, clone.ID).Error; err != nil {
		applog.Error(ctx, "failed to reload copied aroma chemical", "error", err, "id", clone.ID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load copied aroma chemical")
		return
	}

	writeJSON(w, http.StatusCreated, projectAromaChemical(clone, userID))
}

func replaceOtherNames(ctx context.Context, chemical *models.AromaChemical, names []string) error {
	sanitized := make([]models.OtherName, 0, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		sanitized = append(sanitized, models.OtherName{Name: trimmed})
	}

	assoc := database.WithContext(ctx).Model(chemical).Association("OtherNames")
	if len(sanitized) == 0 {
		return assoc.Clear()
	}
	return assoc.Replace(sanitized)
}

func projectAromaChemical(chemical models.AromaChemical, userID uint) aromaChemicalResponse {
	names := make([]string, 0, len(chemical.OtherNames))
	for _, entry := range chemical.OtherNames {
		trimmed := strings.TrimSpace(entry.Name)
		if trimmed == "" {
			continue
		}
		names = append(names, trimmed)
	}

	canEdit := chemical.OwnerID == userID
	canCopy := canEdit || chemical.Public

	return aromaChemicalResponse{
		ID:                  chemical.ID,
		IngredientName:      chemical.IngredientName,
		CASNumber:           chemical.CASNumber,
		Notes:               chemical.Notes,
		WheelPosition:       chemical.WheelPosition,
		PyramidPosition:     chemical.PyramidPosition,
		Type:                chemical.Type,
		Strength:            chemical.Strength,
		RecommendedDilution: chemical.RecommendedDilution,
		DilutionPercentage:  chemical.DilutionPercentage,
		MaxIFRAPercentage:   chemical.MaxIFRAPercentage,
		PricePerMg:          chemical.PricePerMg,
		Duration:            chemical.Duration,
		HistoricRole:        chemical.HistoricRole,
		Popularity:          chemical.Popularity,
		Usage:               chemical.Usage,
		OwnerID:             chemical.OwnerID,
		Public:              chemical.Public,
		OtherNames:          names,
		CreatedAt:           chemical.CreatedAt,
		UpdatedAt:           chemical.UpdatedAt,
		CanEdit:             canEdit,
		CanDelete:           canEdit,
		CanCopy:             canCopy,
	}
}

func nextAvailableName(ctx context.Context, base string) string {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		trimmed = "Unnamed Ingredient"
	}

	candidate := fmt.Sprintf("%s (Copy)", trimmed)
	suffix := 2

	for {
		var count int64
		if err := database.WithContext(ctx).Model(&models.AromaChemical{}).Where("ingredient_name = ?", candidate).Count(&count).Error; err != nil {
			applog.Error(ctx, "failed to check aroma chemical name availability", "error", err, "candidate", candidate)
			return fmt.Sprintf("%s (Copy %d)", trimmed, time.Now().Unix())
		}
		if count == 0 {
			return candidate
		}
		candidate = fmt.Sprintf("%s (Copy %d)", trimmed, suffix)
		suffix++
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		applog.Error(context.Background(), "failed to encode json response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
