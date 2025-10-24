package db

import (
	"fmt"
	"strings"
	"time"

	"perfugo/internal/config"
	"perfugo/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

func Initialize(cfg config.DatabaseConfig) (*gorm.DB, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("database URL must not be empty")
	}

	gormCfg := &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Warn),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	db, err := gorm.Open(postgres.Open(cfg.URL), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}

	return db.AutoMigrate(
		&models.AromaChemical{},
		&models.OtherName{},
		&models.Formula{},
		&models.FormulaIngredient{},
	)
}

func Configure(cfg config.DatabaseConfig) (*gorm.DB, error) {
	database, err := Initialize(cfg)
	if err != nil {
		return nil, err
	}

	if err := AutoMigrate(database); err != nil {
		return nil, err
	}

	DB = database

	return database, nil
}

func MustConfigure(cfg config.DatabaseConfig) *gorm.DB {
	database, err := Configure(cfg)
	if err != nil {
		panic(err)
	}

	return database
}

func Get() *gorm.DB {
	return DB
}
