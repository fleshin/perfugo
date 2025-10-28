package layout

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"perfugo/models"
)

func TestThemeByIDReturnsDefinition(t *testing.T) {
	def := ThemeByID(models.ThemeNocturne)
	if def.ID != models.ThemeNocturne {
		t.Fatalf("expected to retrieve definition for %s", models.ThemeNocturne)
	}
}

func TestThemeByIDFallsBackToDefault(t *testing.T) {
	def := ThemeByID("unknown")
	if def.ID != models.DefaultTheme {
		t.Fatalf("expected fallback to default theme, got %s", def.ID)
	}
}

func TestThemeOptionsAreSortedByLabel(t *testing.T) {
	options := ThemeOptions()
	if len(options) < 2 {
		t.Fatal("expected multiple theme options")
	}
	for i := 1; i < len(options); i++ {
		if options[i-1].Label > options[i].Label {
			t.Fatalf("expected options to be sorted alphabetically by label: %v", options)
		}
	}
}

func TestLayoutRendersProvidedContent(t *testing.T) {
	sidebar := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("<aside>sidebar</aside>"))
		return err
	})
	content := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("<main>content</main>"))
		return err
	})

	var buf bytes.Buffer
	err := Layout("Workspace", sidebar, content, true, ThemeByID(models.DefaultTheme)).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render layout: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<title>Workspace</title>") {
		t.Fatalf("expected document title to be rendered: %s", out)
	}
	if !strings.Contains(out, "sidebar") || !strings.Contains(out, "content") {
		t.Fatalf("expected sidebar and content sections in output: %s", out)
	}
}

func TestBodyWrapperClassReflectsSidebarState(t *testing.T) {
	if bodyWrapperClass(true) == bodyWrapperClass(false) {
		t.Fatal("expected different body wrapper class depending on sidebar state")
	}
	if mainClass(true) == mainClass(false) {
		t.Fatal("expected different main class depending on sidebar state")
	}
}
