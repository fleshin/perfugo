package handlers

import (
	"net/http"

	templpkg "github.com/a-h/templ"

	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
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
