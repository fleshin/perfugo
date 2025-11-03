# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server/` – Go entrypoint that wires configuration, database, and HTTP server.
- `internal/handlers/` – Request handlers, domain logic, and shared utilities.
- `internal/views/` – Templ templates (`.templ`) and generated Go views (`*_templ.go`).
- `internal/views/pages/ingredient_helpers.go` – Helper functions shared by templates.
- `web/static/` – Static assets served by the application.
- `tests/` – Unit and integration tests live beside source files; snapshots reside under the same package.

## Build, Test, and Development Commands
- `go run ./cmd/server` – Start the development server on the configured address.
- `go test ./...` – Execute all Go unit tests; run after any library or handler changes.
- `templ generate ./...` – Regenerate Go view files from `.templ` sources; rerun after editing templates.
- `gofmt -w <files>` – Format Go files; required before commits.
- `golangci-lint run` (if installed) – Run static analysis; resolves most CI lint checks.

## Coding Style & Naming Conventions
- Go code: follow `gofmt`, use mixedCaps for exported identifiers and short, descriptive names for locals.
- Templates: keep HTML indentation at two spaces; prefer helper functions (see `ingredient_helpers.go`) for repeated formatting.
- Routes and handlers: keep files grouped by feature (ingredients, formulas) and name HTTP handlers with verb suffixes (`IngredientUpdate`, `FormulaDelete`).

## Testing Guidelines
- Use Go’s `testing` package; place `_test.go` files alongside implementation files.
- Name tests with the `TestXxx` pattern and use table-driven tests for variations.
- Optional test helpers should live in the same package (`workspace_sections_test.go` demonstrates this).
- Run `go test ./...` before pushing; ensure new handlers include coverage for success and failure paths.

## Commit & Pull Request Guidelines
- Commit messages follow an imperative mood (`Add ingredient helpers`, `Fix formula delete`) and focus on a single change set.
- Squash trivial fix-up commits before submitting a PR.
- Pull requests should describe the change, list test commands executed, and note any follow-up tasks or known gaps.
- Attach screenshots or GIFs when adjusting UI (templates or styling) to simplify review.

