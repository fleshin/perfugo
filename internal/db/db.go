package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"perfugo/internal/config"
	applog "perfugo/internal/log"
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

	applog.Debug(context.Background(), "initializing database connection",
		"urlConfigured", strings.TrimSpace(cfg.URL) != "",
		"maxIdleConns", cfg.MaxIdleConns,
		"maxOpenConns", cfg.MaxOpenConns,
		"connMaxLifetime", cfg.ConnMaxLifetime.String(),
		"connMaxIdleTime", cfg.ConnMaxIdleTime.String(),
	)

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

	applog.Debug(context.Background(), "database connection established")

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	applog.Debug(context.Background(), "configuring database connection pool")

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

	applog.Debug(context.Background(), "database connection pool configured")

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}

	applog.Debug(context.Background(), "running database migrations")

	return db.AutoMigrate(
		&models.AromaChemical{},
		&models.OtherName{},
		&models.Formula{},
		&models.FormulaIngredient{},
		&models.User{},
	)
}

func Configure(cfg config.DatabaseConfig) (*gorm.DB, error) {
	applog.Debug(context.Background(), "configuring database from application settings")
	database, err := Initialize(cfg)
	if err != nil {
		return nil, err
	}

	if err := AutoMigrate(database); err != nil {
		return nil, err
	}

	applog.Debug(context.Background(), "database configured and migrated")

	DB = database

	return database, nil
}

func MustConfigure(cfg config.DatabaseConfig) *gorm.DB {
	applog.Debug(context.Background(), "MustConfigure invoked for database")
	database, err := Configure(cfg)
	if err != nil {
		panic(err)
	}

	applog.Debug(context.Background(), "MustConfigure completed successfully")
	return database
}

func Get() *gorm.DB {
	return DB
}
