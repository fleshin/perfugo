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

// AromaChemicalDetail renders an aroma chemical detail card for HTMX interactions.
func AromaChemicalDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, ok := currentUserID(r)
	if !ok {
		applog.Debug(r.Context(), "aroma chemical detail without authenticated user")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	identifier := strings.TrimPrefix(r.URL.Path, "/app/htmx/ingredients")
	identifier = strings.Trim(identifier, "/")
	if identifier == "" {
		renderIngredientDetail(w, r, nil)
		return
	}

	value, err := strconv.ParseUint(identifier, 10, 64)
	if err != nil {
		applog.Debug(r.Context(), "invalid aroma chemical identifier", "identifier", identifier, "error", err)
		http.NotFound(w, r)
		return
	}

	chemical, err := loadAromaChemicalDetail(r, uint(value), userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		applog.Error(r.Context(), "failed to load aroma chemical detail", "error", err, "id", value)
		http.Error(w, "unable to load aroma chemical", http.StatusInternalServerError)
		return
	}

	renderIngredientDetail(w, r, chemical)
}

func loadAromaChemicalDetail(r *http.Request, id, userID uint) (*models.AromaChemical, error) {
	if database == nil {
		return nil, nil
	}

	ctx := r.Context()
	var chemical models.AromaChemical
	if err := database.WithContext(ctx).Preload("OtherNames").First(&chemical, id).Error; err != nil {
		return nil, err
	}

	if chemical.OwnerID != userID && !chemical.Public {
		applog.Debug(ctx, "aroma chemical access denied", "id", id, "owner", chemical.OwnerID, "user", userID)
		return nil, gorm.ErrRecordNotFound
	}

	return &chemical, nil
}

func renderIngredientDetail(w http.ResponseWriter, r *http.Request, chemical *models.AromaChemical) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.IngredientDetailCard(chemical).Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render ingredient detail", "error", err)
	}
}
