package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLinkState(t *testing.T) {
	if got := linkState("features", "features"); got != "active" {
		t.Fatalf("expected active state when sections match, got %q", got)
	}
	if got := linkState("formulas", "features"); got != "inactive" {
		t.Fatalf("expected inactive state when sections differ, got %q", got)
	}
}

func TestStatCardRendersValues(t *testing.T) {
	var buf bytes.Buffer
	err := StatCard("Completed", "12", "+5%", "Week over week").Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render stat card: %v", err)
	}
	output := buf.String()
	for _, token := range []string{"Completed", "12", "+5%", "Week over week"} {
		if !strings.Contains(output, token) {
			t.Fatalf("expected output to contain %q: %s", token, output)
		}
	}
}

func TestActivityTableRendersEntries(t *testing.T) {
	entries := []ActivityEntry{{Name: "Batch 01", Reference: "REF-1", Quantity: "10kg", Progress: "25%", UpdatedAt: "today", Status: "Mixing"}}
	var buf bytes.Buffer
	if err := ActivityTable(entries).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render activity table: %v", err)
	}
	if !strings.Contains(buf.String(), "Batch 01") {
		t.Fatalf("expected rendered table to include entry name")
	}
}

func TestSidebarRendersActiveSection(t *testing.T) {
	data := SidebarData{
		Active: "ingredients",
		Features: []SidebarLink{{
			Label:   "Ingredients",
			Path:    "/workspace/ingredients",
			Section: "ingredients",
		}},
	}
	var buf bytes.Buffer
	if err := Sidebar(data).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render sidebar: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "data-state=\"active\"") {
		t.Fatalf("expected active data-state attribute in sidebar output: %s", out)
	}
	if !strings.Contains(out, "data-nav-section=\"ingredients\"") {
		t.Fatalf("expected data-nav-section attribute for active link: %s", out)
	}
}
