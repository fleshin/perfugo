package handlers

import (
	"context"
	"net/http"
	"testing"

	"perfugo/internal/db/mock"
)

func TestLoadWorkspaceDataReturnsChemicals(t *testing.T) {
	db, err := mock.New(context.Background())
	if err != nil {
		t.Fatalf("mock database: %v", err)
	}
	Configure(nil, db)
	t.Cleanup(func() { database = nil })

	req, err := http.NewRequest(http.MethodGet, "/app", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	_, _, chemicals := loadWorkspaceData(req, 0)
	if len(chemicals) == 0 {
		t.Fatalf("expected chemicals from workspace load, got none")
	}
}
