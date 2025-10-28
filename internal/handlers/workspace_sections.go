package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	templpkg "github.com/a-h/templ"
	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
	"perfugo/models"
)

// IngredientTable handles HTMX requests for the ingredient ledger.
func IngredientTable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	filters := pages.IngredientFiltersFromRequest(r)
	chemicals := pages.FilterAromaChemicals(snapshot.AromaChemicals, filters)

	renderComponent(w, r, pages.IngredientTable(chemicals, filters, len(snapshot.AromaChemicals)))
}

// IngredientDetail renders the detail card for a single aroma chemical.
func IngredientDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	id := pages.ParseUint(r.URL.Query().Get("id"))
	chemical := pages.FindAromaChemical(snapshot.AromaChemicals, id)

	renderComponent(w, r, pages.IngredientDetail(chemical))
}

// IngredientEdit renders the edit form for a selected aroma chemical.
func IngredientEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	id := pages.ParseUint(r.URL.Query().Get("id"))
	chemical := pages.FindAromaChemical(snapshot.AromaChemicals, id)

	renderComponent(w, r, pages.IngredientEditor(chemical, ""))
}

// IngredientUpdate processes updates submitted from the aroma chemical edit form.
func IngredientUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		applog.Error(r.Context(), "failed to parse ingredient form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := pages.ParseUint(r.FormValue("id"))
	if id == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	chemical := pages.FindAromaChemical(snapshot.AromaChemicals, id)
	if chemical == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := strings.TrimSpace(r.FormValue("ingredient_name"))
	if name == "" {
		renderComponent(w, r, pages.IngredientEditor(chemical, "Ingredient name is required."))
		return
	}

	strengthInput := strings.TrimSpace(r.FormValue("strength"))
	strengthValue := chemical.Strength
	var strengthErr error
	if strengthInput == "" {
		strengthValue = 0
	} else {
		parsed, err := strconv.Atoi(strengthInput)
		if err != nil {
			strengthErr = err
		} else {
			strengthValue = parsed
		}
	}

	if database == nil {
		message := "Editing is unavailable because no database connection is configured."
		chemical.IngredientName = name
		chemical.CASNumber = strings.TrimSpace(r.FormValue("cas_number"))
		chemical.Type = strings.TrimSpace(r.FormValue("type"))
		chemical.PyramidPosition = strings.TrimSpace(r.FormValue("pyramid_position"))
		chemical.WheelPosition = strings.TrimSpace(r.FormValue("wheel_position"))
		chemical.Duration = strings.TrimSpace(r.FormValue("duration"))
		chemical.Notes = strings.TrimSpace(r.FormValue("notes"))
		chemical.Usage = strings.TrimSpace(r.FormValue("usage"))
		if strengthErr == nil {
			chemical.Strength = strengthValue
		}
		renderComponent(w, r, pages.IngredientEditor(chemical, message))
		return
	}

	userID, ok := currentUserID(r)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	ctx := r.Context()
	var stored models.AromaChemical
	if err := database.WithContext(ctx).First(&stored, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		applog.Error(ctx, "failed to load ingredient for update", "error", err, "ingredientID", id)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if stored.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updates := map[string]interface{}{
		"ingredient_name":  name,
		"cas_number":       strings.TrimSpace(r.FormValue("cas_number")),
		"type":             strings.TrimSpace(r.FormValue("type")),
		"pyramid_position": strings.TrimSpace(r.FormValue("pyramid_position")),
		"wheel_position":   strings.TrimSpace(r.FormValue("wheel_position")),
		"duration":         strings.TrimSpace(r.FormValue("duration")),
		"notes":            strings.TrimSpace(r.FormValue("notes")),
		"usage":            strings.TrimSpace(r.FormValue("usage")),
	}

	if strengthErr == nil {
		updates["strength"] = strengthValue
	}

	if err := database.WithContext(ctx).Model(&stored).Updates(updates).Error; err != nil {
		applog.Error(ctx, "failed to update ingredient", "error", err, "ingredientID", id)
		renderComponent(w, r, pages.IngredientEditor(chemical, "We couldn't save your changes. Please try again."))
		return
	}

	if err := database.WithContext(ctx).First(&stored, id).Error; err != nil {
		applog.Error(ctx, "failed to reload ingredient after update", "error", err, "ingredientID", id)
		renderComponent(w, r, pages.IngredientEditor(chemical, "The ingredient was updated, but we couldn't refresh the latest data."))
		return
	}

	status := "Ingredient updated successfully."
	if strengthErr != nil {
		status = "Ingredient updated, but the strength value must be a whole number. The previous value was kept."
	}

	renderComponent(w, r, pages.IngredientEditor(&stored, status))
}

// FormulaList handles HTMX requests for the formula library listings.
func FormulaList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	filters := pages.FormulaFiltersFromRequest(r)
	formulas := pages.FilterFormulas(snapshot.Formulas, filters)

	renderComponent(w, r, pages.FormulaList(formulas, filters, len(snapshot.Formulas)))
}

// FormulaDetail renders the selected formula and its composition.
func FormulaDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	id := pages.ParseUint(r.URL.Query().Get("id"))
	formula := pages.FindFormula(snapshot.Formulas, id)
	ingredients := pages.FormulaIngredientsFor(snapshot.FormulaIngredients, id)

	renderComponent(w, r, pages.FormulaDetail(formula, ingredients))
}

func renderComponent(w http.ResponseWriter, r *http.Request, component templpkg.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render workspace fragment", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
