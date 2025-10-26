package mock

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"perfugo/models"
)

func TestNewSeedsExpectedRecords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := New(ctx)
	if err != nil {
		t.Fatalf("mock database initialization failed: %v", err)
	}

	var chemicals []models.AromaChemical
	if err := db.WithContext(ctx).Find(&chemicals).Error; err != nil {
		t.Fatalf("query aroma chemicals: %v", err)
	}
	if len(chemicals) == 0 {
		t.Fatal("expected seeded aroma chemicals")
	}

	var ingredients []models.FormulaIngredient
	if err := db.WithContext(ctx).Find(&ingredients).Error; err != nil {
		t.Fatalf("query formula ingredients: %v", err)
	}
	if len(ingredients) == 0 {
		t.Fatal("expected seeded formula ingredients")
	}

	var user models.User
	if err := db.WithContext(ctx).First(&user).Error; err != nil {
		t.Fatalf("query user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("atelier")); err != nil {
		t.Fatalf("unexpected password hash: %v", err)
	}
}
