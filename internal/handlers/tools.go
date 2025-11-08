package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"perfugo/internal/ai"
	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
	"perfugo/models"
)

var openAIClient *ai.Client

// ConfigureAI installs the OpenAI client used by tooling endpoints.
func ConfigureAI(client *ai.Client) {
	openAIClient = client
}

// ToolsImportIngredient handles the AI-assisted ingredient import workflow.
func ToolsImportIngredient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := buildWorkspaceSnapshot(r)

	ingredientName := strings.TrimSpace(r.FormValue("ingredient_name"))
	if ingredientName == "" {
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "Provide an ingredient name before requesting an import."))
		return
	}

	userID, ok := currentUserID(r)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if openAIClient == nil {
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "AI integration is not configured. Set OPENAI_API_KEY to enable this tool."))
		return
	}

	ctx := r.Context()
	profile, err := openAIClient.FetchAromaProfile(ctx, ingredientName, ai.FetchOptions{})
	if err != nil {
		applog.Error(ctx, "ai fetch failed", "error", err)
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", fmt.Sprintf("We couldn't fetch data for %q. Please try again shortly.", ingredientName)))
		return
	}

	record, created, warning, err := persistAromaProfile(ctx, profile, userID)
	if err != nil {
		applog.Error(ctx, "persist ai aroma", "error", err)
		renderComponent(w, r, pages.ToolsPanel(snapshot, "", "We couldn't store the generated ingredient. Please try again."))
		return
	}

	snapshot = buildWorkspaceSnapshot(r)
	message := fmt.Sprintf("Added %s to your private library.", record.IngredientName)
	if !created {
		message = fmt.Sprintf("Updated %s with the latest AI profile.", record.IngredientName)
	}
	if strings.TrimSpace(warning) != "" {
		message = fmt.Sprintf("%s %s", message, warning)
	}

	renderComponent(w, r, pages.ToolsPanel(snapshot, message, ""))
}

func persistAromaProfile(ctx context.Context, profile ai.Profile, ownerID uint) (*models.AromaChemical, bool, string, error) {
	if database == nil {
		return nil, false, "", gorm.ErrInvalidDB
	}

	profile.IngredientName = fallbackIngredientName(profile.IngredientName)
	profile.PyramidPosition = pages.CanonicalPyramidPosition(profile.PyramidPosition)
	profile.WheelPosition = strings.TrimSpace(profile.WheelPosition)

	var result models.AromaChemical
	warnings := []string{}
	created := false

	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Attempt to locate an existing record by name.
		existing, err := findChemicalByName(ctx, tx, profile.IngredientName)
		if err != nil {
			return err
		}

		// Attempt CAS lookup when name is unique.
		if existing == nil && strings.TrimSpace(profile.CASNumber) != "" {
			existing, err = findChemicalByCAS(ctx, tx, profile.CASNumber)
			if err != nil {
				return err
			}
			if strings.TrimSpace(profile.CASNumber) != "" {
				casMatch, err := findChemicalByCAS(ctx, tx, profile.CASNumber)
				if err != nil {
					return err
				}
				if casMatch != nil {
					if existing == nil && casMatch.OwnerID == ownerID {
						existing = casMatch
					} else if casMatch.OwnerID != ownerID {
						warnings = append(warnings, fmt.Sprintf("CAS %s already exists as %s. Review for duplicates.", strings.TrimSpace(profile.CASNumber), casMatch.IngredientName))
					}
				}
			}
		}

		if existing != nil && existing.OwnerID == ownerID {
			if err := applyProfileToChemical(ctx, tx, existing, profile, ownerID); err != nil {
				return err
			}
			result = *existing
			created = false
			return nil
		}

		if existing != nil && existing.OwnerID != ownerID {
			uniqueName, err := generatePrivateName(ctx, tx, profile.IngredientName)
			if err != nil {
				return err
			}
			profile.IngredientName = uniqueName
		}

		record, err := createChemicalFromProfile(ctx, tx, profile, ownerID)
		if err != nil {
			return err
		}
		result = *record
		created = true
		return nil
	})
	if err != nil {
		return nil, false, strings.Join(warnings, " "), err
	}
	return &result, created, strings.Join(warnings, " "), nil
}

func findChemicalByName(ctx context.Context, tx *gorm.DB, name string) (*models.AromaChemical, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	var existing models.AromaChemical
	err := tx.WithContext(ctx).Where("lower(ingredient_name) = ?", strings.ToLower(strings.TrimSpace(name))).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &existing, nil
}

func findChemicalByCAS(ctx context.Context, tx *gorm.DB, cas string) (*models.AromaChemical, error) {
	cas = strings.TrimSpace(cas)
	if cas == "" {
		return nil, nil
	}
	var existing models.AromaChemical
	err := tx.WithContext(ctx).Where("cas_number = ?", cas).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &existing, nil
}

func createChemicalFromProfile(ctx context.Context, tx *gorm.DB, profile ai.Profile, ownerID uint) (*models.AromaChemical, error) {
	canonicalPyramid := pages.CanonicalPyramidPosition(profile.PyramidPosition)

	record := models.AromaChemical{
		IngredientName:      profile.IngredientName,
		CASNumber:           strings.TrimSpace(profile.CASNumber),
		Notes:               profile.Notes,
		WheelPosition:       profile.WheelPosition,
		PyramidPosition:     canonicalPyramid,
		Type:                profile.Type,
		Strength:            profile.Strength,
		RecommendedDilution: profile.RecommendedDilution,
		DilutionPercentage:  profile.DilutionPercentage,
		MaxIFRAPercentage:   profile.MaxIFRAPercentage,
		Duration:            profile.Duration,
		HistoricRole:        profile.HistoricRole,
		Popularity:          profile.Popularity,
		Usage:               profile.Usage,
		OwnerID:             ownerID,
		Public:              false,
	}

	if err := tx.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}

	if err := replaceOtherNames(ctx, tx, record.ID, profile.OtherNames); err != nil {
		return nil, err
	}

	if err := tx.WithContext(ctx).Preload("OtherNames").First(&record, record.ID).Error; err != nil {
		return nil, err
	}

	return &record, nil
}

func applyProfileToChemical(ctx context.Context, tx *gorm.DB, existing *models.AromaChemical, profile ai.Profile, ownerID uint) error {
	canonicalPyramid := pages.CanonicalPyramidPosition(profile.PyramidPosition)

	updates := map[string]any{
		"ingredient_name":      profile.IngredientName,
		"notes":                profile.Notes,
		"wheel_position":       profile.WheelPosition,
		"pyramid_position":     canonicalPyramid,
		"type":                 profile.Type,
		"strength":             profile.Strength,
		"recommended_dilution": profile.RecommendedDilution,
		"dilution_percentage":  profile.DilutionPercentage,
		"max_ifra_percentage":  profile.MaxIFRAPercentage,
		"duration":             profile.Duration,
		"historic_role":        profile.HistoricRole,
		"popularity":           profile.Popularity,
		"usage":                profile.Usage,
		"owner_id":             ownerID,
		"public":               false,
	}

	cas := strings.TrimSpace(profile.CASNumber)
	if cas != "" {
		updates["cas_number"] = cas
	}
	if err := tx.WithContext(ctx).Model(existing).Updates(updates).Error; err != nil {
		return err
	}

	if err := replaceOtherNames(ctx, tx, existing.ID, profile.OtherNames); err != nil {
		return err
	}

	return tx.WithContext(ctx).Preload("OtherNames").First(existing, existing.ID).Error
}

func generatePrivateName(ctx context.Context, tx *gorm.DB, base string) (string, error) {
	base = fallbackIngredientName(base)
	candidate := fmt.Sprintf("%s (Private)", base)
	suffix := 2
	for {
		var count int64
		if err := tx.WithContext(ctx).Model(&models.AromaChemical{}).
			Where("lower(ingredient_name) = ?", strings.ToLower(candidate)).
			Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s (Private %d)", base, suffix)
		suffix++
	}
}

func fallbackIngredientName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return "Untitled Ingredient"
}

func replaceOtherNames(ctx context.Context, tx *gorm.DB, chemicalID uint, names []string) error {
	if err := tx.WithContext(ctx).Where("aroma_chemical_id = ?", chemicalID).Delete(&models.OtherName{}).Error; err != nil {
		return err
	}

	if len(names) == 0 {
		return nil
	}

	entries := make([]models.OtherName, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		entries = append(entries, models.OtherName{
			Name:            trimmed,
			AromaChemicalID: chemicalID,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	return tx.WithContext(ctx).Create(&entries).Error
}
