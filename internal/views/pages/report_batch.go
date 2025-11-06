package pages

import (
	"fmt"
	"strings"
	"time"
)

// BatchProductionReportIngredient captures the scaled contribution of a single aroma chemical.
type BatchProductionReportIngredient struct {
	Order          int
	IngredientName string
	CASNumber      string
	Pyramid        string
	PyramidLabel   string
	BaseQuantity   float64
	FinalQuantity  float64
	Unit           string
}

// BatchProductionReportData aggregates the metadata required to render the production form.
type BatchProductionReportData struct {
	FormulaName       string
	FormulaVersion    int
	TargetQuantity    float64
	TargetUnit        string
	BaseBatchQuantity float64
	BaseBatchUnit     string
	ScaleFactor       float64
	LotNumber         string
	RunDate           time.Time
	Ingredients       []BatchProductionReportIngredient
}

// FormatReportQuantity renders a quantity using two decimal places and a trailing unit.
func FormatReportQuantity(value float64, unit string) string {
	if strings.EqualFold(unit, "mg") {
		return fmt.Sprintf("%.0f %s", value, unit)
	}
	return fmt.Sprintf("%.2f %s", value, unit)
}

// FormatReportDate renders the supplied time using a production-friendly layout.
func FormatReportDate(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.Format("02 Jan 2006")
}
