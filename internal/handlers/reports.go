package handlers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
	"perfugo/models"
)

var (
	errBatchFormulaNotFound   = errors.New("reports: formula not found")
	errBatchInvalidQuantity   = errors.New("reports: invalid target quantity")
	errBatchEmptyComposition  = errors.New("reports: formula has no ingredients")
	errBatchCircularReference = errors.New("reports: circular dependency detected")
	nowFunc                   = time.Now
)

// GenerateBatchProductionReport renders a production-ready batch form for the selected formula.
func GenerateBatchProductionReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid submission.", http.StatusBadRequest)
		return
	}

	formulaID := pages.ParseUint(r.FormValue("formula_id"))
	if formulaID == 0 {
		http.Error(w, "Select a formula before running the report.", http.StatusBadRequest)
		return
	}

	targetQuantity, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("target_quantity")), 64)
	if err != nil || targetQuantity <= 0 {
		http.Error(w, "Provide a positive target quantity.", http.StatusBadRequest)
		return
	}

	report, err := buildBatchProductionReportData(r.Context(), formulaID, targetQuantity)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrInvalidDB):
			http.Error(w, "Reporting is unavailable because no database connection is configured.", http.StatusServiceUnavailable)
		case errors.Is(err, errBatchFormulaNotFound):
			http.Error(w, "The selected formula no longer exists.", http.StatusNotFound)
		case errors.Is(err, errBatchInvalidQuantity):
			http.Error(w, "The target quantity cannot be computed for this formula.", http.StatusBadRequest)
		case errors.Is(err, errBatchEmptyComposition):
			http.Error(w, "The selected formula has no ingredients to report.", http.StatusBadRequest)
		case errors.Is(err, errBatchCircularReference):
			http.Error(w, "The formula has a circular dependency and cannot be expanded.", http.StatusBadRequest)
		default:
			applog.Error(r.Context(), "failed to build batch production report", "error", err, "formulaID", formulaID)
			http.Error(w, "We were unable to generate the batch report. Please try again.", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.BatchProductionReport(report).Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render batch production report", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func buildBatchProductionReportData(ctx context.Context, formulaID uint, targetQuantity float64) (pages.BatchProductionReportData, error) {
	if database == nil {
		return pages.BatchProductionReportData{}, gorm.ErrInvalidDB
	}

	var formula models.Formula
	if err := database.WithContext(ctx).First(&formula, formulaID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pages.BatchProductionReportData{}, errBatchFormulaNotFound
		}
		return pages.BatchProductionReportData{}, err
	}

	var ingredients []models.FormulaIngredient
	if err := database.WithContext(ctx).
		Preload("AromaChemical").
		Preload("SubFormula").
		Find(&ingredients).Error; err != nil {
		return pages.BatchProductionReportData{}, err
	}

	byFormula := make(map[uint][]models.FormulaIngredient)
	for _, ing := range ingredients {
		byFormula[ing.FormulaID] = append(byFormula[ing.FormulaID], ing)
	}

	if len(byFormula[formulaID]) == 0 {
		return pages.BatchProductionReportData{}, errBatchEmptyComposition
	}

	totalsMemo := make(map[uint]float64)
	totalStack := make(map[uint]bool)
	computeTotals := func(id uint) (float64, error) {
		return computeFormulaTotal(id, byFormula, totalsMemo, totalStack)
	}

	baseTotal, err := computeTotals(formulaID)
	if err != nil {
		return pages.BatchProductionReportData{}, err
	}
	if baseTotal <= 0 {
		return pages.BatchProductionReportData{}, errBatchInvalidQuantity
	}

	accumulator := make(map[uint]*reportIngredientTotal)
	traversal := make(map[uint]bool)
	accumulate := func(id uint, factor float64) error {
		return accumulateFormulaIngredients(ctx, id, factor, byFormula, accumulator, computeTotals, traversal)
	}

	if err := accumulate(formulaID, 1.0); err != nil {
		return pages.BatchProductionReportData{}, err
	}

	scale := targetQuantity / baseTotal
	if math.IsNaN(scale) || math.IsInf(scale, 0) {
		return pages.BatchProductionReportData{}, errBatchInvalidQuantity
	}

	reportIngredients := make([]pages.BatchProductionReportIngredient, 0, len(accumulator))
	for _, total := range accumulator {
		if total.Chemical == nil {
			continue
		}
		finalQuantity := total.BaseAmount * scale
		if finalQuantity <= 0 {
			continue
		}
		reportIngredients = append(reportIngredients, pages.BatchProductionReportIngredient{
			IngredientName: total.Chemical.IngredientName,
			CASNumber:      strings.TrimSpace(total.Chemical.CASNumber),
			Pyramid:        pages.CanonicalPyramidPosition(total.Chemical.PyramidPosition),
			PyramidLabel:   pages.PyramidPositionLabel(total.Chemical.PyramidPosition),
			FinalQuantity:  finalQuantity,
			BaseQuantity:   total.BaseAmount,
			Unit:           "g",
		})
	}

	sortBatchProductionIngredients(reportIngredients)

	for idx := range reportIngredients {
		reportIngredients[idx].Order = idx + 1
	}

	runTime := nowFunc().UTC()
	data := pages.BatchProductionReportData{
		FormulaName:       formula.Name,
		FormulaVersion:    int(formula.Version),
		TargetQuantity:    targetQuantity,
		TargetUnit:        "g",
		BaseBatchQuantity: baseTotal,
		BaseBatchUnit:     "g",
		ScaleFactor:       scale,
		LotNumber:         fmt.Sprintf("PERF-%s-%03d", runTime.Format("20060102"), formula.Version),
		RunDate:           runTime,
		Ingredients:       reportIngredients,
	}

	return data, nil
}

type reportIngredientTotal struct {
	Chemical   *models.AromaChemical
	BaseAmount float64
}

func computeFormulaTotal(
	formulaID uint,
	source map[uint][]models.FormulaIngredient,
	memo map[uint]float64,
	stack map[uint]bool,
) (float64, error) {
	if value, ok := memo[formulaID]; ok {
		return value, nil
	}
	if stack[formulaID] {
		return 0, errBatchCircularReference
	}
	stack[formulaID] = true

	ingredients := source[formulaID]
	if len(ingredients) == 0 {
		stack[formulaID] = false
		return 0, errBatchEmptyComposition
	}

	total := 0.0
	for _, ing := range ingredients {
		total += normalizeAmount(ing.Amount, ing.Unit)
	}

	memo[formulaID] = total
	stack[formulaID] = false
	return total, nil
}

func accumulateFormulaIngredients(
	ctx context.Context,
	formulaID uint,
	factor float64,
	source map[uint][]models.FormulaIngredient,
	accumulator map[uint]*reportIngredientTotal,
	totalResolver func(uint) (float64, error),
	path map[uint]bool,
) error {
	if path[formulaID] {
		return errBatchCircularReference
	}
	path[formulaID] = true

	ingredients := source[formulaID]
	if len(ingredients) == 0 {
		path[formulaID] = false
		return errBatchEmptyComposition
	}

	for _, ing := range ingredients {
		amount := normalizeAmount(ing.Amount, ing.Unit) * factor
		if amount <= 0 {
			continue
		}
		if ing.AromaChemicalID != nil {
			chemical := ing.AromaChemical
			if chemical == nil {
				var fetched models.AromaChemical
				if err := database.WithContext(ctx).First(&fetched, *ing.AromaChemicalID).Error; err != nil {
					return err
				}
				chemical = &fetched
			}
			total, ok := accumulator[*ing.AromaChemicalID]
			if !ok {
				total = &reportIngredientTotal{Chemical: chemical}
				accumulator[*ing.AromaChemicalID] = total
			}
			total.BaseAmount += amount
			continue
		}
		if ing.SubFormulaID != nil && *ing.SubFormulaID != 0 {
			subTotal, err := totalResolver(*ing.SubFormulaID)
			if err != nil {
				return err
			}
			if subTotal <= 0 {
				continue
			}
			subFactor := amount / subTotal
			if err := accumulateFormulaIngredients(ctx, *ing.SubFormulaID, subFactor, source, accumulator, totalResolver, path); err != nil {
				return err
			}
		}
	}

	path[formulaID] = false
	return nil
}

func sortBatchProductionIngredients(items []pages.BatchProductionReportIngredient) {
	sort.SliceStable(items, func(i, j int) bool {
		pi := pyramidRank(items[i].Pyramid)
		pj := pyramidRank(items[j].Pyramid)
		if pi != pj {
			return pi < pj
		}
		if !almostEqual(items[i].FinalQuantity, items[j].FinalQuantity) {
			return items[i].FinalQuantity > items[j].FinalQuantity
		}
		return strings.ToLower(items[i].IngredientName) < strings.ToLower(items[j].IngredientName)
	})
}

func pyramidRank(value string) int {
	switch value {
	case "base":
		return 0
	case "heart-base", "base-heart":
		return 1
	case "heart":
		return 2
	case "top-heart", "heart-top":
		return 3
	case "top":
		return 4
	default:
		return 5
	}
}

func normalizeAmount(amount float64, unit string) float64 {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "mg":
		return amount / 1000.0
	case "kg":
		return amount * 1000.0
	case "ml":
		// Assume density ~1 g/mL for production planning purposes.
		return amount
	default:
		return amount
	}
}

func almostEqual(a, b float64) bool {
	const epsilon = 1e-6
	return math.Abs(a-b) <= epsilon
}
