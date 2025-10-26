package models

import (
	"gorm.io/gorm"
)

type AromaChemical struct {
	gorm.Model
	IngredientName      string      `gorm:"uniqueIndex;not null" json:"ingredient_name"`
	CASNumber           string      `gorm:"uniqueIndex" json:"cas_number"`
	OtherNames          []OtherName `gorm:"foreignKey:AromaChemicalID" json:"other_names"`
	Notes               string      `gorm:"type:text" json:"notes"`
	WheelPosition       string      `json:"wheel_position"`
	PyramidPosition     string      `json:"pyramid_position"`
	Type                string      `json:"type"`
	Strength            int         `json:"strength"`
	RecommendedDilution float64     `json:"recommended_dilution"`
	DilutionPercentage  float64     `json:"dilution_percentage"`
	MaxIFRAPercentage   float64     `json:"max_ifra_percentage"`
	PricePerMg          float64     `json:"price_per_mg"`
	Duration            string      `json:"duration"`
	HistoricRole        string      `json:"historic_role"`
	Popularity          int         `json:"popularity"`
	Usage               string      `gorm:"type:text" json:"usage"`
	OwnerID             uint        `gorm:"not null" json:"owner_id"`
	Owner               *User       `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Public              bool        `gorm:"not null;default:false" json:"public"`
}

// OtherName holds an alternative name for an AromaChemical.
type OtherName struct {
	gorm.Model
	Name            string `gorm:"not null" json:"name"`
	AromaChemicalID uint
}
