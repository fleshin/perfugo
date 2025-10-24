package models

import (
	"gorm.io/gorm"
)

type Formula struct {
	gorm.Model
	Name            string              `gorm:"not null" json:"name"`
	Notes           string              `gorm:"type:text" json:"notes"`
	Version         int                 `gorm:"not null;default:1" json:"version"`
	IsLatest        bool                `gorm:"not null;default:true" json:"is_latest"`
	ParentFormulaID *uint               `json:"parent_formula_id"`
	Ingredients     []FormulaIngredient `gorm:"foreignKey:FormulaID" json:"ingredients"`
}
