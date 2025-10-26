package main

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"perfugo/internal/db/mock"
	"perfugo/models"
)

func TestMockImporterSeedsWorkspaceData(t *testing.T) {
	ctx := context.Background()
	db, err := mock.New(ctx)
	if err != nil {
		t.Fatalf("mock.New returned error: %v", err)
	}

	var formulaCount int64
	if err := db.Model(&models.Formula{}).Count(&formulaCount).Error; err != nil {
		t.Fatalf("count formulas: %v", err)
	}
	if formulaCount == 0 {
		t.Fatal("expected mock database to seed formulas")
	}

	var ingredientCount int64
	if err := db.Model(&models.FormulaIngredient{}).Count(&ingredientCount).Error; err != nil {
		t.Fatalf("count ingredients: %v", err)
	}
	if ingredientCount == 0 {
		t.Fatal("expected mock database to seed formula ingredients")
	}

	var user models.User
	if err := db.First(&user).Error; err != nil {
		t.Fatalf("fetch user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("atelier")); err != nil {
		t.Fatalf("seeded user password hash mismatch: %v", err)
	}
}
