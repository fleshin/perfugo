package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/models"
)

type formulaIngredientRequest struct {
	FormulaID       uint    `json:"formula_id"`
	Amount          float64 `json:"amount"`
	Unit            string  `json:"unit"`
	AromaChemicalID *uint   `json:"aroma_chemical_id"`
	SubFormulaID    *uint   `json:"sub_formula_id"`
}

type ingredientChemicalSummary struct {
	ID             uint   `json:"id"`
	IngredientName string `json:"ingredient_name"`
	Version        int    `json:"version"`
}

type ingredientFormulaSummary struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type formulaIngredientResponse struct {
	ID              uint                       `json:"id"`
	FormulaID       uint                       `json:"formula_id"`
	Amount          float64                    `json:"amount"`
	Unit            string                     `json:"unit"`
	AromaChemicalID *uint                      `json:"aroma_chemical_id,omitempty"`
	SubFormulaID    *uint                      `json:"sub_formula_id,omitempty"`
	AromaChemical   *ingredientChemicalSummary `json:"aroma_chemical,omitempty"`
	SubFormula      *ingredientFormulaSummary  `json:"sub_formula,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

// FormulaIngredientResource handles CRUD interactions for formula ingredient records.
func FormulaIngredientResource(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		applog.Debug(r.Context(), "formula ingredient request without database")
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	if _, ok := currentUserID(r); !ok {
		applog.Debug(r.Context(), "formula ingredient request without authenticated user")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/app/api/formula-ingredients")
	path = strings.Trim(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			listFormulaIngredients(w, r)
		case http.MethodPost:
			createFormulaIngredient(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	idValue, err := strconv.ParseUint(path, 10, 64)
	if err != nil {
		applog.Debug(r.Context(), "invalid formula ingredient identifier", "identifier", path, "error", err)
		http.NotFound(w, r)
		return
	}
	ingredientID := uint(idValue)

	switch r.Method {
	case http.MethodGet:
		showFormulaIngredient(w, r, ingredientID)
	case http.MethodPut:
		updateFormulaIngredient(w, r, ingredientID)
	case http.MethodDelete:
		deleteFormulaIngredient(w, r, ingredientID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func listFormulaIngredients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var results []models.FormulaIngredient

	query := database.WithContext(ctx).
		Preload("AromaChemical").
		Preload("SubFormula").
		Order("formula_id asc, id asc")

	if formulaParam := strings.TrimSpace(r.URL.Query().Get("formula_id")); formulaParam != "" {
		if idValue, err := strconv.ParseUint(formulaParam, 10, 64); err == nil {
			query = query.Where("formula_id = ?", uint(idValue))
		}
	}

	if err := query.Find(&results).Error; err != nil {
		applog.Error(ctx, "failed to list formula ingredients", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "unable to load formula ingredients")
		return
	}

	responses := make([]formulaIngredientResponse, 0, len(results))
	for _, ingredient := range results {
		responses = append(responses, projectFormulaIngredient(ingredient))
	}

	writeJSON(w, http.StatusOK, responses)
}

func showFormulaIngredient(w http.ResponseWriter, r *http.Request, ingredientID uint) {
	ctx := r.Context()
	var ingredient models.FormulaIngredient
	if err := database.WithContext(ctx).
		Preload("AromaChemical").
		Preload("SubFormula").
		First(&ingredient, ingredientID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
			return
		}
		applog.Error(ctx, "failed to load formula ingredient", "error", err, "id", ingredientID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load formula ingredient")
		return
	}

	writeJSON(w, http.StatusOK, projectFormulaIngredient(ingredient))
}

func createFormulaIngredient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload formulaIngredientRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		applog.Debug(ctx, "invalid formula ingredient create payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := validateFormulaIngredientPayload(payload); err != nil {
		applog.Debug(ctx, "formula ingredient validation failed", "error", err)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ingredient := models.FormulaIngredient{
		FormulaID:       payload.FormulaID,
		Amount:          payload.Amount,
		Unit:            normalizedUnit(payload.Unit),
		AromaChemicalID: payload.AromaChemicalID,
		SubFormulaID:    payload.SubFormulaID,
	}

	if err := database.WithContext(ctx).Create(&ingredient).Error; err != nil {
		applog.Error(ctx, "failed to create formula ingredient", "error", err)
		writeJSONError(w, http.StatusBadRequest, "unable to create formula ingredient")
		return
	}

	if err := database.WithContext(ctx).
		Preload("AromaChemical").
		Preload("SubFormula").
		First(&ingredient, ingredient.ID).Error; err != nil {
		applog.Error(ctx, "failed to reload created formula ingredient", "error", err, "id", ingredient.ID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load formula ingredient")
		return
	}

	writeJSON(w, http.StatusCreated, projectFormulaIngredient(ingredient))
}

func updateFormulaIngredient(w http.ResponseWriter, r *http.Request, ingredientID uint) {
	ctx := r.Context()
	var existing models.FormulaIngredient
	if err := database.WithContext(ctx).First(&existing, ingredientID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
			return
		}
		applog.Error(ctx, "failed to load formula ingredient for update", "error", err, "id", ingredientID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load formula ingredient")
		return
	}

	var payload formulaIngredientRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		applog.Debug(ctx, "invalid formula ingredient update payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := validateFormulaIngredientPayload(payload); err != nil {
		applog.Debug(ctx, "formula ingredient update validation failed", "error", err)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	updates := map[string]any{
		"formula_id":        payload.FormulaID,
		"amount":            payload.Amount,
		"unit":              normalizedUnit(payload.Unit),
		"aroma_chemical_id": payload.AromaChemicalID,
		"sub_formula_id":    payload.SubFormulaID,
	}

	if err := database.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		applog.Error(ctx, "failed to update formula ingredient", "error", err, "id", ingredientID)
		writeJSONError(w, http.StatusBadRequest, "unable to update formula ingredient")
		return
	}

	if err := database.WithContext(ctx).
		Preload("AromaChemical").
		Preload("SubFormula").
		First(&existing, ingredientID).Error; err != nil {
		applog.Error(ctx, "failed to reload updated formula ingredient", "error", err, "id", ingredientID)
		writeJSONError(w, http.StatusInternalServerError, "unable to load formula ingredient")
		return
	}

	writeJSON(w, http.StatusOK, projectFormulaIngredient(existing))
}

func deleteFormulaIngredient(w http.ResponseWriter, r *http.Request, ingredientID uint) {
	ctx := r.Context()
	if err := database.WithContext(ctx).Delete(&models.FormulaIngredient{}, ingredientID).Error; err != nil {
		applog.Error(ctx, "failed to delete formula ingredient", "error", err, "id", ingredientID)
		writeJSONError(w, http.StatusInternalServerError, "unable to delete formula ingredient")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func projectFormulaIngredient(ingredient models.FormulaIngredient) formulaIngredientResponse {
	response := formulaIngredientResponse{
		ID:              ingredient.ID,
		FormulaID:       ingredient.FormulaID,
		Amount:          ingredient.Amount,
		Unit:            ingredient.Unit,
		AromaChemicalID: ingredient.AromaChemicalID,
		SubFormulaID:    ingredient.SubFormulaID,
		CreatedAt:       ingredient.CreatedAt,
		UpdatedAt:       ingredient.UpdatedAt,
	}

	if ingredient.AromaChemical != nil {
		response.AromaChemical = &ingredientChemicalSummary{
			ID:             ingredient.AromaChemical.ID,
			IngredientName: strings.TrimSpace(ingredient.AromaChemical.IngredientName),
			Version:        1,
		}
	}

	if ingredient.SubFormula != nil {
		response.SubFormula = &ingredientFormulaSummary{
			ID:      ingredient.SubFormula.ID,
			Name:    strings.TrimSpace(ingredient.SubFormula.Name),
			Version: ingredient.SubFormula.Version,
		}
	}

	return response
}

func validateFormulaIngredientPayload(payload formulaIngredientRequest) error {
	if payload.FormulaID == 0 {
		return errors.New("formula_id is required")
	}
	if payload.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}

	hasChemical := payload.AromaChemicalID != nil && *payload.AromaChemicalID != 0
	hasSubFormula := payload.SubFormulaID != nil && *payload.SubFormulaID != 0

	if hasChemical && hasSubFormula {
		return errors.New("only one of aroma_chemical_id or sub_formula_id may be set")
	}
	if !hasChemical && !hasSubFormula {
		return errors.New("either aroma_chemical_id or sub_formula_id must be provided")
	}
	return nil
}

func normalizedUnit(unit string) string {
	trimmed := strings.TrimSpace(unit)
	if trimmed == "" {
		return "g"
	}
	return trimmed
}
