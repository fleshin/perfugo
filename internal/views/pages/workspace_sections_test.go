package pages

import (
	"testing"
)

func TestThemeButtonState(t *testing.T) {
	if themeButtonState("a", "a") != "active" {
		t.Fatal("expected active state when theme matches")
	}
	if themeButtonState("a", "b") != "inactive" {
		t.Fatal("expected inactive state when theme differs")
	}
}

func TestFormatAuditDate(t *testing.T) {
	if got := formatAuditDate("2024-03-01"); got != "01 Mar 2024" {
		t.Fatalf("expected formatted date, got %s", got)
	}
	if got := formatAuditDate(""); got != "—" {
		t.Fatalf("expected em dash for empty string, got %s", got)
	}
	if got := formatAuditDate("invalid"); got != "invalid" {
		t.Fatalf("expected original value on parse failure, got %s", got)
	}
}

func TestDefaultDash(t *testing.T) {
	if DefaultDash("value") != "value" {
		t.Fatal("expected non-empty value to pass through")
	}
	if DefaultDash("   ") != "—" {
		t.Fatal("expected whitespace value to produce em dash")
	}
}

func TestAromaChemicalPotencyLabel(t *testing.T) {
	cases := map[int]string{
		8: "Powerful",
		5: "Strong",
		3: "Moderate",
		1: "Delicate",
		0: "Unknown",
	}
	for strength, want := range cases {
		if got := AromaChemicalPotencyLabel(strength); got != want {
			t.Fatalf("potency label for %d: expected %s, got %s", strength, want, got)
		}
	}
}
