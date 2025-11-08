package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"gorm.io/gorm"

	"perfugo/internal/ai"
	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
	"perfugo/models"
)

const (
	maxFormulaUploadSize = 5 << 20 // 5 MiB
	targetFormulaTotalMG = 1000.0
)

type formulaImportIngredient struct {
	Name       string
	OtherNames []string
	QuantityMG float64
}

type resolvedIngredient struct {
	Chemical *models.AromaChemical
	AmountMG float64
}

// ToolsImportFormula handles AI-assisted formula ingestion.
func ToolsImportFormula(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)

	if openAIClient == nil {
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "AI integration is not configured. Set OPENAI_API_KEY to enable this tool."))
		return
	}

	userID, ok := currentUserID(r)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(maxFormulaUploadSize); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		applog.Error(r.Context(), "failed to parse formula import form", "error", err)
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "Upload is too large or invalid. Please retry with a smaller file."))
		return
	}

	nameHint := strings.TrimSpace(r.FormValue("formula_name_hint"))
	rawText := strings.TrimSpace(r.FormValue("formula_text"))

	fileName, fileBytes, fileType, err := readFormulaUpload(r)
	if err != nil {
		applog.Error(r.Context(), "formula upload read failed", "error", err)
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "Unable to read the uploaded file. Please try again."))
		return
	}

	var base64Payload string
	if len(fileBytes) > 0 {
		processed, encoded, convErr := deriveTextFromUpload(fileBytes, fileType)
		if convErr != nil {
			applog.Error(r.Context(), "failed to extract formula text", "error", convErr, "mime", fileType)
			renderComponent(w, r, pages.ToolsPanel(snapshot, "", "We couldn't interpret the uploaded document. Try a different format."))
			return
		}
		if strings.TrimSpace(processed) != "" {
			if rawText != "" {
				rawText += "\n\n"
			}
			rawText += processed
		} else if encoded != "" {
			base64Payload = encoded
		}
	}

	if strings.TrimSpace(rawText) == "" && base64Payload == "" {
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "Provide formula text or upload a document before running the import."))
		return
	}

	ctx := r.Context()
	aiResult, err := openAIClient.ExtractFormula(ctx, ai.FormulaImportInput{
		NameHint:   nameHint,
		RawText:    rawText,
		Base64File: base64Payload,
		FileName:   fileName,
		FileType:   fileType,
	})
	if err != nil {
		applog.Error(ctx, "formula extraction failed", "error", err)
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "We couldn't interpret that formula. Please refine the input and try again."))
		return
	}

	scaled, err := scaleFormulaComponents(aiResult.Ingredients, targetFormulaTotalMG)
	if err != nil {
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", err.Error()))
		return
	}

	chemicals := snapshotChemicalPointers(snapshot.AromaChemicals)
	resolved, warnings, err := resolveFormulaIngredients(ctx, userID, scaled, chemicals)
	if err != nil {
		applog.Error(ctx, "resolve ingredients failed", "error", err)
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "Unable to map ingredients to the catalog. Please review the names and retry."))
		return
	}

	formulaName := determineFormulaName(snapshot.Formulas, aiResult.FormulaName)
	formula, err := persistImportedFormula(ctx, formulaName, aiResult.Notes, resolved)
	if err != nil {
		applog.Error(ctx, "persist imported formula failed", "error", err)
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "We couldn't save the imported formula. Please try again."))
		return
	}

	snapshot = buildWorkspaceSnapshot(r)
	message := fmt.Sprintf("Imported formula \"%s\" with %d ingredients.", formula.Name, len(resolved))
	if len(warnings) > 0 {
		message = fmt.Sprintf("%s %s", message, strings.Join(warnings, " "))
	}
	renderComponent(w, r, pages.ToolsPanel(snapshot, message, ""))
}

func readFormulaUpload(r *http.Request) (string, []byte, string, error) {
	file, header, err := r.FormFile("formula_file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil, "", nil
		}
		return "", nil, "", err
	}
	defer file.Close()

	if header.Size > maxFormulaUploadSize {
		return "", nil, "", fmt.Errorf("file exceeds %d bytes", maxFormulaUploadSize)
	}

	buf := bytes.NewBuffer(make([]byte, 0, header.Size))
	if _, err := io.Copy(buf, file); err != nil {
		return "", nil, "", err
	}

	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = mimeTypeFromName(header.Filename)
	}

	return header.Filename, buf.Bytes(), mime, nil
}

func deriveTextFromUpload(data []byte, mime string) (string, string, error) {
	lower := strings.ToLower(mime)
	switch {
	case strings.Contains(lower, "pdf"):
		text, err := extractTextFromPDF(data)
		if err != nil {
			return "", "", err
		}
		return text, "", nil
	case strings.HasPrefix(lower, "text/") || strings.Contains(lower, "json"):
		return string(data), "", nil
	case strings.HasPrefix(lower, "image/"):
		return "", base64.StdEncoding.EncodeToString(data), nil
	default:
		return string(data), "", nil
	}
}

func extractTextFromPDF(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	numPages := reader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			return "", err
		}
		builder.WriteString(text)
		builder.WriteString("\n")
	}
	return builder.String(), nil
}

func mimeTypeFromName(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func scaleFormulaComponents(ingredients []ai.FormulaImportIngredient, target float64) ([]formulaImportIngredient, error) {
	scaled := make([]formulaImportIngredient, 0, len(ingredients))
	total := 0.0
	for _, item := range ingredients {
		amount := item.QuantityMG
		if amount <= 0 {
			continue
		}
		total += amount
		scaled = append(scaled, formulaImportIngredient{
			Name:       strings.TrimSpace(item.IngredientName),
			OtherNames: item.OtherNames,
			QuantityMG: amount,
		})
	}
	if len(scaled) == 0 || total <= 0 {
		return nil, errors.New("no valid ingredients were extracted from the formula reference")
	}

	factor := target / total
	for i := range scaled {
		scaled[i].QuantityMG = math.Round(scaled[i].QuantityMG*factor*1000) / 1000
		if scaled[i].QuantityMG <= 0 {
			scaled[i].QuantityMG = 0.1
		}
	}
	return scaled, nil
}

func snapshotChemicalPointers(source []models.AromaChemical) []*models.AromaChemical {
	result := make([]*models.AromaChemical, 0, len(source))
	for i := range source {
		result = append(result, &source[i])
	}
	return result
}

func resolveFormulaIngredients(ctx context.Context, userID uint, candidates []formulaImportIngredient, chemicals []*models.AromaChemical) ([]resolvedIngredient, []string, error) {
	resolved := make([]resolvedIngredient, 0, len(candidates))
	warnings := []string{}
	for _, candidate := range candidates {
		match := matchChemicalByAliases(chemicals, candidate.Name, candidate.OtherNames)
		if match == nil {
			profile, err := openAIClient.FetchAromaProfile(ctx, candidate.Name, ai.FetchOptions{})
			if err != nil {
				return nil, nil, err
			}
			record, _, warning, err := persistAromaProfile(ctx, profile, userID)
			if err != nil {
				return nil, nil, err
			}
			if strings.TrimSpace(warning) != "" {
				warnings = append(warnings, warning)
			}
			chemicals = append(chemicals, record)
			match = record
		}
		resolved = append(resolved, resolvedIngredient{
			Chemical: match,
			AmountMG: candidate.QuantityMG,
		})
	}
	return resolved, warnings, nil
}

func determineFormulaName(existing []models.Formula, requested string) string {
	trimmed := strings.TrimSpace(requested)
	if trimmed == "" {
		return pages.NextUntitledFormulaName(existing)
	}
	for _, f := range existing {
		if strings.EqualFold(strings.TrimSpace(f.Name), trimmed) {
			return pages.NextCopiedFormulaName(existing, trimmed)
		}
	}
	return trimmed
}

func matchChemicalByAliases(chemicals []*models.AromaChemical, primary string, aliases []string) *models.AromaChemical {
	targets := uniqueAliases(append([]string{primary}, aliases...))
	for _, chem := range chemicals {
		if chem == nil {
			continue
		}
		if aliasMatches(chem, targets) {
			return chem
		}
	}
	return nil
}

func aliasMatches(chemical *models.AromaChemical, targets []string) bool {
	if chemical == nil {
		return false
	}
	candidates := uniqueAliases([]string{chemical.IngredientName})
	for _, other := range chemical.OtherNames {
		candidates = append(candidates, normalizeIngredientName(other.Name))
	}
	for _, target := range targets {
		for _, candidate := range candidates {
			if candidate == target || similarAlias(candidate, target) {
				return true
			}
		}
	}
	return false
}

func uniqueAliases(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		norm := normalizeIngredientName(value)
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		result = append(result, norm)
	}
	return result
}

func normalizeIngredientName(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	cleaned := replacer.Replace(trimmed)
	cleaned = lettersOnly(cleaned)
	return cleaned
}

func lettersOnly(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func similarAlias(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	dist := levenshteinDistance(a, b)
	limit := 1
	if len(a) >= 8 || len(b) >= 8 {
		limit = 2
	}
	if len(a) >= 12 || len(b) >= 12 {
		limit = 3
	}
	return dist <= limit
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = minInt(
				curr[j-1]+1,
				minInt(prev[j]+1, prev[j-1]+cost),
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func persistImportedFormula(ctx context.Context, name, notes string, entries []resolvedIngredient) (*models.Formula, error) {
	if database == nil {
		return nil, gorm.ErrInvalidDB
	}
	formula := models.Formula{
		Name:  strings.TrimSpace(name),
		Notes: strings.TrimSpace(notes),
	}
	if formula.Name == "" {
		formula.Name = "Imported Formula"
	}

	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&formula).Error; err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Chemical == nil || entry.AmountMG <= 0 {
				continue
			}
			chemID := entry.Chemical.ID
			ing := models.FormulaIngredient{
				FormulaID:       formula.ID,
				Amount:          entry.AmountMG,
				Unit:            "mg",
				AromaChemicalID: &chemID,
			}
			if err := tx.Create(&ing).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &formula, nil
}
