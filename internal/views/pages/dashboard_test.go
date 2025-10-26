package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNormalizeWorkspaceSection(t *testing.T) {
	if got := NormalizeWorkspaceSection("  FORMULAS "); got != "formulas" {
		t.Fatalf("expected normalized section to be 'formulas', got %s", got)
	}
	if got := NormalizeWorkspaceSection("unknown"); got != defaultWorkspaceSection {
		t.Fatalf("expected fallback to default section, got %s", got)
	}
	if got := NormalizeWorkspaceSection(" "); got != defaultWorkspaceSection {
		t.Fatalf("expected fallback for empty section, got %s", got)
	}
}

func TestValidWorkspaceSection(t *testing.T) {
	valid := []string{"ingredients", "formulas", "reports", "preferences"}
	for _, section := range valid {
		if !ValidWorkspaceSection(section) {
			t.Fatalf("expected %s to be valid", section)
		}
	}
	if ValidWorkspaceSection("invalid") {
		t.Fatal("expected invalid section to be rejected")
	}
}

func TestDefaultWorkspaceSection(t *testing.T) {
	if DefaultWorkspaceSection() != defaultWorkspaceSection {
		t.Fatal("expected default workspace section constant to be returned")
	}
}

func TestWorkspaceRendersWithFallbackSection(t *testing.T) {
	snapshot := EmptyWorkspaceSnapshot()
	var buf bytes.Buffer
	if err := Workspace("unknown", snapshot).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "data-module-key=\"ingredients\"") {
		t.Fatalf("expected workspace to render default section content: %s", out)
	}
}

func TestDefaultReportLeadersSeeded(t *testing.T) {
	leaders := defaultReportLeaders()
	if len(leaders) != 3 {
		t.Fatalf("expected three default report leaders, got %d", len(leaders))
	}
}
