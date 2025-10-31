package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	rawPyramid := r.FormValue("pyramid_position")
	pyramidValue, pyramidOK := pages.NormalizePyramidPosition(rawPyramid)
	if !pyramidOK {
		chemical.PyramidPosition = strings.TrimSpace(rawPyramid)
		renderComponent(w, r, pages.IngredientEditor(chemical, "Select a valid pyramid position."))
		return
	}
	chemical.PyramidPosition = pyramidValue

	if database == nil {
		message := "Editing is unavailable because no database connection is configured."
		chemical.IngredientName = name
		chemical.CASNumber = strings.TrimSpace(r.FormValue("cas_number"))
		chemical.Type = strings.TrimSpace(r.FormValue("type"))
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
		"pyramid_position": pyramidValue,
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

// FormulaCreate initialises a new, empty formula and opens it in the editor.
func FormulaCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	filters := pages.FormulaFiltersFromRequest(r)
	snapshot := buildWorkspaceSnapshot(r)
	filtered := pages.FilterFormulas(snapshot.Formulas, filters)
	total := len(snapshot.Formulas)

	if database == nil {
		message := "Creating formulas is unavailable because no database connection is configured."
		renderComponent(w, r, pages.FormulaCreationError(message, filtered, filters, total))
		return
	}

	name := pages.NextUntitledFormulaName(snapshot.Formulas)
	record := models.Formula{
		Name:    name,
		Version: 1,
	}

	ctx := r.Context()
	if err := database.WithContext(ctx).Create(&record).Error; err != nil {
		applog.Error(ctx, "failed to create formula", "error", err)
		renderComponent(w, r, pages.FormulaCreationError("We couldn't start a new formula. Please try again.", filtered, filters, total))
		return
	}

	refreshed := buildWorkspaceSnapshot(r)
	created := pages.FindFormula(refreshed.Formulas, record.ID)
	if created == nil {
		created = &record
	}
	ingredients := pages.FormulaIngredientsFor(refreshed.FormulaIngredients, created.ID)

	refreshedFiltered := pages.FilterFormulas(refreshed.Formulas, filters)
	if filters.Query != "" {
		found := false
		for _, formula := range refreshedFiltered {
			if formula.ID == created.ID {
				found = true
				break
			}
		}
		if !found {
			refreshedFiltered = append([]models.Formula{*created}, refreshedFiltered...)
		}
	}

	status := "New formula created. Give it a name and start composing."
	renderComponent(w, r, pages.FormulaCreationSuccess(
		created,
		ingredients,
		refreshed.AromaChemicals,
		refreshed.Formulas,
		refreshedFiltered,
		filters,
		len(refreshed.Formulas),
		status,
	))
}

// FormulaEdit renders the edit form for a selected formula.
func FormulaEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	id := pages.ParseUint(r.URL.Query().Get("id"))
	formula := pages.FindFormula(snapshot.Formulas, id)
	ingredients := pages.FormulaIngredientsFor(snapshot.FormulaIngredients, id)

	renderComponent(w, r, pages.FormulaEditor(formula, ingredients, snapshot.AromaChemicals, snapshot.Formulas, ""))
}

type formulaIngredientUpdate struct {
	ID              uint
	Amount          float64
	Unit            string
	AromaChemicalID *uint
	SubFormulaID    *uint
	RowKey          string
}

func parseIngredientSource(value string) (*uint, *uint, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil, errors.New("ingredient source missing")
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid ingredient source: %s", trimmed)
	}
	id := pages.ParseUint(parts[1])
	if id == 0 {
		return nil, nil, fmt.Errorf("invalid ingredient identifier: %s", trimmed)
	}
	switch parts[0] {
	case "chem":
		return &id, nil, nil
	case "formula":
		return nil, &id, nil
	default:
		return nil, nil, fmt.Errorf("unknown ingredient source prefix: %s", parts[0])
	}
}

// FormulaUpdate processes edits submitted from the formula editor.
func FormulaUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		applog.Error(r.Context(), "failed to parse formula form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := pages.ParseUint(r.FormValue("id"))
	if id == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	formula := pages.FindFormula(snapshot.Formulas, id)
	if formula == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	currentIngredients := pages.FormulaIngredientsFor(snapshot.FormulaIngredients, id)

	name := strings.TrimSpace(r.FormValue("formula_name"))
	if name == "" {
		renderComponent(w, r, pages.FormulaEditor(formula, currentIngredients, snapshot.AromaChemicals, snapshot.Formulas, "Formula name is required."))
		return
	}

	notes := strings.TrimSpace(r.FormValue("notes"))

	action := strings.TrimSpace(r.FormValue("form_action"))
	if action == "" {
		action = "update"
	}

	versionValue := formula.Version
	if action == "new_version" {
		versionValue = formula.Version + 1
	}

	filters := pages.FormulaFiltersFromRequest(r)

	removals := map[string]struct{}{}
	for _, key := range r.Form["ingredient_remove"] {
		removals[strings.TrimSpace(key)] = struct{}{}
	}

	rowKeys := r.Form["ingredient_row_key"]
	entryIDs := r.Form["ingredient_entry_id"]
	sources := r.Form["ingredient_source"]
	amounts := r.Form["ingredient_amount"]
	units := r.Form["ingredient_unit"]

	if len(rowKeys) != len(entryIDs) || len(rowKeys) != len(sources) || len(rowKeys) != len(amounts) || len(rowKeys) != len(units) {
		applog.Error(r.Context(), "formula ingredient arrays misaligned",
			"rowKeys", len(rowKeys),
			"entryIDs", len(entryIDs),
			"sources", len(sources),
			"amounts", len(amounts),
			"units", len(units),
		)
		renderComponent(w, r, pages.FormulaEditor(formula, currentIngredients, snapshot.AromaChemicals, snapshot.Formulas, "We couldn't process the ingredient list. Please try again."))
		return
	}

	updates := make([]formulaIngredientUpdate, 0, len(rowKeys))
	deletes := make([]uint, 0)
	updatedIngredients := make([]models.FormulaIngredient, 0, len(rowKeys))

	for i := range rowKeys {
		rowKey := strings.TrimSpace(rowKeys[i])
		entryID := pages.ParseUint(entryIDs[i])
		source := strings.TrimSpace(sources[i])
		amountInput := strings.TrimSpace(amounts[i])
		unit := strings.TrimSpace(units[i])

		if _, marked := removals[rowKey]; marked {
			if entryID > 0 {
				deletes = append(deletes, entryID)
			}
			continue
		}

		if source == "" && amountInput == "" && unit == "" && entryID == 0 {
			continue
		}

		chemID, subID, err := parseIngredientSource(source)
		if err != nil {
			renderComponent(w, r, pages.FormulaEditor(formula, currentIngredients, snapshot.AromaChemicals, snapshot.Formulas, "Select an ingredient for each composition row before saving."))
			return
		}

		if subID != nil && *subID == formula.ID {
			renderComponent(w, r, pages.FormulaEditor(formula, currentIngredients, snapshot.AromaChemicals, snapshot.Formulas, "A formula cannot include itself as a sub-formula."))
			return
		}

		amountValue := 0.0
		if amountInput != "" {
			parsedAmount, err := strconv.ParseFloat(amountInput, 64)
			if err != nil {
				renderComponent(w, r, pages.FormulaEditor(formula, currentIngredients, snapshot.AromaChemicals, snapshot.Formulas, "Ingredient amounts must be numbers."))
				return
			}
			amountValue = parsedAmount
		}

		update := formulaIngredientUpdate{
			ID:              entryID,
			Amount:          amountValue,
			Unit:            unit,
			AromaChemicalID: chemID,
			SubFormulaID:    subID,
			RowKey:          rowKey,
		}
		updates = append(updates, update)

		ingredientRecord := models.FormulaIngredient{
			FormulaID:       formula.ID,
			Amount:          amountValue,
			Unit:            unit,
			AromaChemicalID: chemID,
			SubFormulaID:    subID,
		}
		ingredientRecord.ID = update.ID
		if chemID != nil {
			ingredientRecord.AromaChemical = pages.FindAromaChemical(snapshot.AromaChemicals, *chemID)
		}
		if subID != nil {
			ingredientRecord.SubFormula = pages.FindFormula(snapshot.Formulas, *subID)
		}
		updatedIngredients = append(updatedIngredients, ingredientRecord)
	}

	status := "Formula updated successfully."

	if database == nil {
		formula.Name = name
		formula.Notes = notes
		if action == "new_version" {
			formula.Version = versionValue
		}
		renderComponent(w, r, pages.FormulaEditor(formula, updatedIngredients, snapshot.AromaChemicals, snapshot.Formulas, "Editing is unavailable because no database connection is configured."))
		return
	}

	ctx := r.Context()

	if action == "save_as" {
		copyName := pages.NextCopiedFormulaName(snapshot.Formulas, name)
		newFormula := models.Formula{
			Name:     copyName,
			Notes:    notes,
			Version:  1,
			IsLatest: true,
		}

		err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&newFormula).Error; err != nil {
				return err
			}
			for _, update := range updates {
				record := models.FormulaIngredient{
					FormulaID:       newFormula.ID,
					Amount:          update.Amount,
					Unit:            update.Unit,
					AromaChemicalID: update.AromaChemicalID,
					SubFormulaID:    update.SubFormulaID,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			applog.Error(ctx, "failed to save formula copy", "error", err, "formulaID", id)
			renderComponent(w, r, pages.FormulaEditor(formula, currentIngredients, snapshot.AromaChemicals, snapshot.Formulas, "We couldn't create a copy of this formula. Please try again."))
			return
		}

		refreshed := buildWorkspaceSnapshot(r)
		created := pages.FindFormula(refreshed.Formulas, newFormula.ID)
		if created == nil {
			created = &newFormula
		}
		newComposition := pages.FormulaIngredientsFor(refreshed.FormulaIngredients, newFormula.ID)
		refreshedFiltered := pages.FilterFormulas(refreshed.Formulas, filters)
		if filters.Query != "" {
			found := false
			for _, candidate := range refreshedFiltered {
				if candidate.ID == created.ID {
					found = true
					break
				}
			}
			if !found {
				refreshedFiltered = append([]models.Formula{*created}, refreshedFiltered...)
			}
		}

		statusCopy := fmt.Sprintf("Saved copy as %s.", created.Name)
		renderComponent(w, r, pages.FormulaCreationSuccess(
			created,
			newComposition,
			refreshed.AromaChemicals,
			refreshed.Formulas,
			refreshedFiltered,
			filters,
			len(refreshed.Formulas),
			statusCopy,
		))
		return
	}

	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updatesMap := map[string]interface{}{
			"name":  name,
			"notes": notes,
		}
		if versionValue > 0 {
			updatesMap["version"] = versionValue
		}
		if err := tx.Model(&models.Formula{}).Where("id = ?", id).Updates(updatesMap).Error; err != nil {
			return err
		}

		if len(deletes) > 0 {
			if err := tx.Where("id IN ?", deletes).Delete(&models.FormulaIngredient{}).Error; err != nil {
				return err
			}
		}

		for _, update := range updates {
			if update.ID > 0 {
				if err := tx.Model(&models.FormulaIngredient{}).
					Where("id = ?", update.ID).
					Updates(map[string]interface{}{
						"amount":            update.Amount,
						"unit":              update.Unit,
						"aroma_chemical_id": update.AromaChemicalID,
						"sub_formula_id":    update.SubFormulaID,
					}).Error; err != nil {
					return err
				}
			} else {
				record := models.FormulaIngredient{
					FormulaID:       id,
					Amount:          update.Amount,
					Unit:            update.Unit,
					AromaChemicalID: update.AromaChemicalID,
					SubFormulaID:    update.SubFormulaID,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		applog.Error(ctx, "failed to update formula", "error", err, "formulaID", id)
		renderComponent(w, r, pages.FormulaEditor(formula, currentIngredients, snapshot.AromaChemicals, snapshot.Formulas, "We couldn't save your changes. Please try again."))
		return
	}

	refreshed := buildWorkspaceSnapshot(r)
	updatedFormula := pages.FindFormula(refreshed.Formulas, id)
	if updatedFormula == nil {
		updatedFormula = formula
	}
	updatedComposition := pages.FormulaIngredientsFor(refreshed.FormulaIngredients, id)
	refreshedFiltered := pages.FilterFormulas(refreshed.Formulas, filters)
	if filters.Query != "" && updatedFormula != nil {
		found := false
		for _, candidate := range refreshedFiltered {
			if candidate.ID == updatedFormula.ID {
				found = true
				break
			}
		}
		if !found {
			refreshedFiltered = append([]models.Formula{*updatedFormula}, refreshedFiltered...)
		}
	}
	if action == "new_version" {
		status = fmt.Sprintf("Version bumped to %d and saved.", versionValue)
	}

	renderComponent(w, r, pages.FormulaCreationSuccess(
		updatedFormula,
		updatedComposition,
		refreshed.AromaChemicals,
		refreshed.Formulas,
		refreshedFiltered,
		filters,
		len(refreshed.Formulas),
		status,
	))
}

// FormulaIngredientRow returns an editable composition row for the formula editor.
func FormulaIngredientRow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		applog.Error(r.Context(), "failed to parse ingredient row request", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	formulaID := pages.ParseUint(r.FormValue("formula_id"))
	if formulaID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)
	formula := pages.FindFormula(snapshot.Formulas, formulaID)
	if formula == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	position := len(r.Form["ingredient_row_key"])
	rowKey := fmt.Sprintf("new-%d", time.Now().UnixNano())
	renderComponent(w, r, pages.FormulaIngredientEditorRow(rowKey, position, nil, snapshot.AromaChemicals, snapshot.Formulas, formula.ID))
}

// FormulaDelete removes a formula record and refreshes the list/detail panes.
func FormulaDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		applog.Error(r.Context(), "failed to parse formula delete form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := pages.ParseUint(r.FormValue("id"))
	if id == 0 {
		id = pages.ParseUint(r.URL.Query().Get("id"))
	}
	if id == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filters := pages.FormulaFiltersFromRequest(r)

	if database == nil {
		snapshot := buildWorkspaceSnapshot(r)
		filtered := pages.FilterFormulas(snapshot.Formulas, filters)
		message := "Deleting formulas is unavailable because no database connection is configured."
		renderComponent(w, r, pages.FormulaDeletionResult(message, filtered, filters, len(snapshot.Formulas)))
		return
	}

	ctx := r.Context()
	var formula models.Formula
	if err := database.WithContext(ctx).First(&formula, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		applog.Error(ctx, "failed to load formula for deletion", "error", err, "formulaID", id)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var inUse int64
	if err := database.WithContext(ctx).
		Model(&models.FormulaIngredient{}).
		Where("sub_formula_id = ?", id).
		Count(&inUse).Error; err != nil {
		applog.Error(ctx, "failed to count formula references", "error", err, "formulaID", id)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if inUse > 0 {
		snapshot := buildWorkspaceSnapshot(r)
		filtered := pages.FilterFormulas(snapshot.Formulas, filters)
		message := "This formula is used as a sub-formula in other compositions. Remove those references before deleting."
		renderComponent(w, r, pages.FormulaDeletionResult(message, filtered, filters, len(snapshot.Formulas)))
		return
	}

	if err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("formula_id = ?", id).Delete(&models.FormulaIngredient{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", id).Delete(&models.Formula{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		applog.Error(ctx, "failed to delete formula", "error", err, "formulaID", id)
		snapshot := buildWorkspaceSnapshot(r)
		filtered := pages.FilterFormulas(snapshot.Formulas, filters)
		renderComponent(w, r, pages.FormulaDeletionResult("We couldn't delete this formula. Please try again.", filtered, filters, len(snapshot.Formulas)))
		return
	}

	refreshed := buildWorkspaceSnapshot(r)
	filtered := pages.FilterFormulas(refreshed.Formulas, filters)
	message := fmt.Sprintf("\"%s\" deleted successfully.", formula.Name)
	renderComponent(w, r, pages.FormulaDeletionResult(message, filtered, filters, len(refreshed.Formulas)))
}

// IngredientDelete removes an aroma chemical owned by the current user when it is not referenced.
func IngredientDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		applog.Error(r.Context(), "failed to parse ingredient delete form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := pages.ParseUint(r.FormValue("id"))
	if id == 0 {
		id = pages.ParseUint(r.URL.Query().Get("id"))
	}
	if id == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filters := pages.IngredientFiltersFromRequest(r)

	if database == nil {
		snapshot := buildWorkspaceSnapshot(r)
		filtered := pages.FilterAromaChemicals(snapshot.AromaChemicals, filters)
		message := "Deleting ingredients is unavailable because no database connection is configured."
		renderComponent(w, r, pages.IngredientDeletionResult(message, filtered, filters, len(snapshot.AromaChemicals)))
		return
	}

	userID, ok := currentUserID(r)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	ctx := r.Context()
	var chemical models.AromaChemical
	if err := database.WithContext(ctx).First(&chemical, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		applog.Error(ctx, "failed to load ingredient for deletion", "error", err, "ingredientID", id)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if chemical.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var inUse int64
	if err := database.WithContext(ctx).
		Model(&models.FormulaIngredient{}).
		Where("aroma_chemical_id = ?", id).
		Count(&inUse).Error; err != nil {
		applog.Error(ctx, "failed to count ingredient references", "error", err, "ingredientID", id)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if inUse > 0 {
		snapshot := buildWorkspaceSnapshot(r)
		filtered := pages.FilterAromaChemicals(snapshot.AromaChemicals, filters)
		message := "This ingredient is used in one or more formulas. Remove those references before deleting."
		renderComponent(w, r, pages.IngredientDeletionResult(message, filtered, filters, len(snapshot.AromaChemicals)))
		return
	}

	if err := database.WithContext(ctx).Delete(&models.AromaChemical{}, id).Error; err != nil {
		applog.Error(ctx, "failed to delete ingredient", "error", err, "ingredientID", id)
		snapshot := buildWorkspaceSnapshot(r)
		filtered := pages.FilterAromaChemicals(snapshot.AromaChemicals, filters)
		renderComponent(w, r, pages.IngredientDeletionResult("We couldn't delete this ingredient. Please try again.", filtered, filters, len(snapshot.AromaChemicals)))
		return
	}

	refreshed := buildWorkspaceSnapshot(r)
	filtered := pages.FilterAromaChemicals(refreshed.AromaChemicals, filters)
	message := fmt.Sprintf("\"%s\" deleted successfully.", chemical.IngredientName)
	renderComponent(w, r, pages.IngredientDeletionResult(message, filtered, filters, len(refreshed.AromaChemicals)))
}

func renderComponent(w http.ResponseWriter, r *http.Request, component templpkg.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render workspace fragment", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
