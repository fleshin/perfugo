package handlers

import (
	"context"
	"net/http"
	"strings"

	templpkg "github.com/a-h/templ"
	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
	"perfugo/models"
)

// Dashboard renders the main application workspace once a user is authenticated.
func Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		applog.Debug(r.Context(), "dashboard access with unsupported method", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	section := pages.NormalizeWorkspaceSection(workspaceSectionFromPath(r.URL.Path))
	applog.Debug(r.Context(), "rendering workspace", "htmx", isHTMX(r), "section", section)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	theme := loadCurrentUserTheme(r)
	applog.Debug(r.Context(), "workspace theme resolved", "theme", theme)
	snapshot := pages.EmptyWorkspaceSnapshot()
	snapshot.Theme = theme
	if database != nil {
		formulas, ingredients, chemicals := loadWorkspaceData(r)
		snapshot = pages.NewWorkspaceSnapshot(formulas, ingredients, chemicals, theme)
	}

	var component templpkg.Component
	if isHTMX(r) {
		applog.Debug(r.Context(), "rendering HTMX workspace partial")
		component = pages.WorkspaceSection(section, snapshot)
	} else {
		applog.Debug(r.Context(), "rendering full workspace page")
		component = pages.Workspace(section, snapshot)
	}

	if err := component.Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render dashboard", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loadWorkspaceData(r *http.Request) ([]models.Formula, []models.FormulaIngredient, []models.AromaChemical) {
	ctx := r.Context()
	formulas := loadFormulas(ctx)
	ingredients := loadFormulaIngredients(ctx)
	chemicals := loadAromaChemicals(ctx)

	applog.Debug(ctx, "workspace dataset loaded",
		"formulas", len(formulas),
		"ingredients", len(ingredients),
		"chemicals", len(chemicals),
	)

	return formulas, ingredients, chemicals
}

func loadFormulas(ctx context.Context) []models.Formula {
	results := []models.Formula{}
	if database == nil {
		return results
	}

	if err := database.WithContext(ctx).
		Preload("Ingredients").
		Preload("Ingredients.AromaChemical").
		Preload("Ingredients.SubFormula").
		Order("name asc").
		Find(&results).Error; err != nil {
		applog.Error(ctx, "failed to load formulas for workspace", "error", err)
	}
	return results
}

func loadFormulaIngredients(ctx context.Context) []models.FormulaIngredient {
	results := []models.FormulaIngredient{}
	if database == nil {
		return results
	}

	if err := database.WithContext(ctx).
		Preload("AromaChemical").
		Preload("SubFormula").
		Preload("Formula").
		Order("formula_id asc").
		Find(&results).Error; err != nil {
		applog.Error(ctx, "failed to load formula ingredients for workspace", "error", err)
	}
	return results
}

func loadAromaChemicals(ctx context.Context) []models.AromaChemical {
	results := []models.AromaChemical{}
	if database == nil {
		return results
	}

	if err := database.WithContext(ctx).
		Model(&models.AromaChemical{}).
		Preload("OtherNames").
		Order("ingredient_name asc").
		Find(&results).Error; err != nil {
		applog.Error(ctx, "failed to load aroma chemicals for workspace", "error", err)
	}
	return results
}

func workspaceSectionFromPath(path string) string {
	trimmed := strings.TrimPrefix(path, "/app")
	trimmed = strings.Trim(trimmed, "/")
	return trimmed
}
