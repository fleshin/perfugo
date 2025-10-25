package mock

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	applog "perfugo/internal/log"
	"perfugo/models"
)

// New returns an in-memory sqlite database seeded with representative atelier data.
func New(ctx context.Context) (*gorm.DB, error) {
	applog.Debug(ctx, "initialising mock database")

	db, err := gorm.Open(sqlite.Open("file:perfugo-mock?mode=memory&cache=shared"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		PrepareStmt:                              true,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.AromaChemical{},
		&models.OtherName{},
		&models.Formula{},
		&models.FormulaIngredient{},
		&models.User{},
	); err != nil {
		return nil, err
	}

	if err := seed(ctx, db); err != nil {
		return nil, err
	}

	applog.Debug(ctx, "mock database ready")
	return db, nil
}

func seed(ctx context.Context, db *gorm.DB) error {
	applog.Debug(ctx, "seeding mock database")

	password, err := bcrypt.GenerateFromPassword([]byte("atelier"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		Name:         "Avery Studio",
		Email:        "avery@perfugo.app",
		PasswordHash: string(password),
	}
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}

	bergamot := models.AromaChemical{
		IngredientName:      "Bergamot Essential",
		CASNumber:           "8007-75-8",
		Notes:               "Cold-pressed citrus brightness harvested from Calabria groves.",
		Type:                "Top Note",
		Strength:            3,
		RecommendedDilution: 0.1,
	}

	iris := models.AromaChemical{
		IngredientName:      "Iris Pallida Butter",
		CASNumber:           "8002-65-1",
		Notes:               "Velvety floral heart with powdery texture and persistence.",
		Type:                "Heart Note",
		Strength:            4,
		RecommendedDilution: 0.05,
	}

	ambroxan := models.AromaChemical{
		IngredientName:      "Ambroxan",
		CASNumber:           "6790-58-5",
		Notes:               "Modern ambergris profile delivering warmth and diffusion.",
		Type:                "Base Note",
		Strength:            5,
		RecommendedDilution: 0.02,
	}

	chemicals := []*models.AromaChemical{&bergamot, &iris, &ambroxan}
	for _, chemical := range chemicals {
		if err := db.WithContext(ctx).Create(chemical).Error; err != nil {
			return err
		}
	}

	aurum := models.Formula{
		Name:     "Aurum Nocturne",
		Notes:    "Resinous amber core balanced with luminous citrus facets.",
		Version:  1,
		IsLatest: true,
	}

	lumen := models.Formula{
		Name:     "Lumen Céleste",
		Notes:    "Radiant iris halo with cool musk trails for longevity.",
		Version:  2,
		IsLatest: true,
	}

	if err := db.WithContext(ctx).Create(&aurum).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Create(&lumen).Error; err != nil {
		return err
	}

	ingredients := []models.FormulaIngredient{
		{
			FormulaID:       aurum.ID,
			Amount:          18.0,
			Unit:            "g",
			AromaChemicalID: &bergamot.ID,
			AromaChemical:   &bergamot,
		},
		{
			FormulaID:       aurum.ID,
			Amount:          12.5,
			Unit:            "g",
			AromaChemicalID: &ambroxan.ID,
			AromaChemical:   &ambroxan,
		},
		{
			FormulaID:       lumen.ID,
			Amount:          9.2,
			Unit:            "g",
			AromaChemicalID: &iris.ID,
			AromaChemical:   &iris,
		},
		{
			FormulaID:       lumen.ID,
			Amount:          4.8,
			Unit:            "g",
			AromaChemicalID: &bergamot.ID,
			AromaChemical:   &bergamot,
		},
	}

	for _, ingredient := range ingredients {
		ingredientCopy := ingredient
		if err := db.WithContext(ctx).Create(&ingredientCopy).Error; err != nil {
			return err
		}
	}

	applog.Debug(ctx, "mock database seeded")
	return nil
}
