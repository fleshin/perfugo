package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
	"perfugo/models"
)

// FormulaDetail renders a formula detail card for HTMX interactions.
func FormulaDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if _, ok := currentUserID(r); !ok {
		applog.Debug(r.Context(), "formula detail without authenticated user")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	identifier := strings.TrimPrefix(r.URL.Path, "/app/htmx/formulas")
	identifier = strings.Trim(identifier, "/")
	if identifier == "" {
		renderFormulaDetail(w, r, nil)
		return
	}

	value, err := strconv.ParseUint(identifier, 10, 64)
	if err != nil {
		applog.Debug(r.Context(), "invalid formula identifier", "identifier", identifier, "error", err)
		http.NotFound(w, r)
		return
	}

	formula, err := loadFormulaDetail(r, uint(value))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		applog.Error(r.Context(), "failed to load formula detail", "error", err, "id", value)
		http.Error(w, "unable to load formula", http.StatusInternalServerError)
		return
	}

	renderFormulaDetail(w, r, formula)
}

func loadFormulaDetail(r *http.Request, id uint) (*models.Formula, error) {
	if database == nil {
		return nil, nil
	}

	ctx := r.Context()
	var formula models.Formula
	if err := database.WithContext(ctx).
		Preload("Ingredients").
		Preload("Ingredients.AromaChemical").
		Preload("Ingredients.SubFormula").
		First(&formula, id).Error; err != nil {
		return nil, err
	}

	return &formula, nil
}

func renderFormulaDetail(w http.ResponseWriter, r *http.Request, formula *models.Formula) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.FormulaDetailCard(formula).Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render formula detail", "error", err)
	}
}
