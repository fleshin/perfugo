package db

import (
	"testing"

	"perfugo/internal/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitializeRequiresURL(t *testing.T) {
	t.Parallel()

	db, err := Initialize(config.DatabaseConfig{URL: ""})
	if err == nil {
		t.Fatal("expected error when database URL is empty")
	}
	if db != nil {
		t.Fatal("expected returned db handle to be nil on error")
	}
}

func TestAutoMigrateRejectsNilDatabase(t *testing.T) {
	t.Parallel()

	if err := AutoMigrate(nil); err == nil {
		t.Fatal("expected error when database handle is nil")
	}
}

func TestAutoMigrateWithSQLite(t *testing.T) {
	t.Parallel()

	sqliteDB, err := gorm.Open(sqlite.Open("file:memdb?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}

	if err := AutoMigrate(sqliteDB); err != nil {
		t.Fatalf("automigrate sqlite database: %v", err)
	}
}

func TestConfigurePropagatesInitializationError(t *testing.T) {
	t.Parallel()

	if _, err := Configure(config.DatabaseConfig{}); err == nil {
		t.Fatal("expected configuration error when initialize fails")
	}
}

func TestMustConfigurePanicsOnError(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when configuration fails")
		}
	}()

	MustConfigure(config.DatabaseConfig{})
}
