package models

import (
	"gorm.io/gorm"
)

type FormulaIngredient struct {
	gorm.Model
	FormulaID uint    `gorm:"not null" json:"formula_id"` // Parent Formula
	Amount    float64 `gorm:"not null" json:"amount"`
	Unit      string  `gorm:"not null" json:"unit"`

	// --- Ingredient Link ---
	// One of these will be non-null, the other will be null.
	AromaChemicalID *uint `json:"aroma_chemical_id,omitempty"`
	SubFormulaID    *uint `json:"sub_formula_id,omitempty"`

	// --- Preloadable Data ---
	// These allow GORM to fetch the ingredient's details.
	// We use pointers so they can be null.
	AromaChemical *AromaChemical `gorm:"foreignKey:AromaChemicalID" json:"aroma_chemical,omitempty"`
	SubFormula    *Formula       `gorm:"foreignKey:SubFormulaID" json:"sub_formula,omitempty"`
}
