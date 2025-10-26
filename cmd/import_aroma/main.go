package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"perfugo/internal/config"
	"perfugo/internal/db"
	"perfugo/models"

	"gorm.io/gorm"
)

var (
	bracketPattern  = regexp.MustCompile(`\[[^\]]*\]`)
	numberPattern   = regexp.MustCompile(`[-+]?\d*\.?\d+`)
	cleanWhitespace = regexp.MustCompile(`\s+`)
	slugPattern     = regexp.MustCompile(`[^a-z0-9]+`)
)

func main() {
	csvPath := "master ingredients list - master.csv"
	if len(os.Args) > 1 {
		csvPath = os.Args[1]
	}

	if err := run(csvPath); err != nil {
		fmt.Fprintf(os.Stderr, "import failed: %v\n", err)
		os.Exit(1)
	}
}

func run(csvPath string) error {
	if strings.TrimSpace(csvPath) == "" {
		return fmt.Errorf("csv path must not be empty")
	}

	if _, err := os.Stat(csvPath); err != nil {
		return fmt.Errorf("locate csv: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	database, err := db.Initialize(cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(database); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	records, err := readCSV(csvPath)
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}

	ownerID, err := resolveImportOwner(database)
	if err != nil {
		return fmt.Errorf("resolve owner: %w", err)
	}

	imported := 0
	for idx, record := range records {
		if err := database.Transaction(func(tx *gorm.DB) error {
			chemical := buildAromaChemical(record)
			chemical.OwnerID = ownerID

			var existing models.AromaChemical
			foundByName := false
			foundByCAS := false

			err := tx.Where("ingredient_name = ?", chemical.IngredientName).First(&existing).Error
			if err == nil {
				foundByName = true
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("find aroma chemical by name %q: %w", chemical.IngredientName, err)
			}

			if !foundByName && chemical.CASNumber != "" {
				err = tx.Where("cas_number = ?", chemical.CASNumber).First(&existing).Error
				if err == nil {
					foundByCAS = true
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("find aroma chemical by CAS %q (%s): %w", chemical.CASNumber, chemical.IngredientName, err)
				}
			}

			canonicalName := chemical.IngredientName
			var extraAliases []string

			if !foundByName && !foundByCAS {
				if err := tx.Create(&chemical).Error; err != nil {
					return fmt.Errorf("create aroma chemical %q: %w", chemical.IngredientName, err)
				}
			} else {
				updates := map[string]any{
					"notes":                chemical.Notes,
					"wheel_position":       chemical.WheelPosition,
					"pyramid_position":     chemical.PyramidPosition,
					"type":                 chemical.Type,
					"strength":             chemical.Strength,
					"recommended_dilution": chemical.RecommendedDilution,
					"dilution_percentage":  chemical.DilutionPercentage,
					"max_ifra_percentage":  chemical.MaxIFRAPercentage,
					"duration":             chemical.Duration,
					"historic_role":        chemical.HistoricRole,
					"popularity":           chemical.Popularity,
					"usage":                chemical.Usage,
				}

				if chemical.CASNumber != "" {
					updates["cas_number"] = chemical.CASNumber
				}

				if foundByCAS && !strings.EqualFold(existing.IngredientName, chemical.IngredientName) {
					canonicalName = existing.IngredientName
					extraAliases = append(extraAliases, chemical.IngredientName)
				} else {
					updates["ingredient_name"] = chemical.IngredientName
					canonicalName = chemical.IngredientName
				}

				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return fmt.Errorf("update aroma chemical %q: %w", canonicalName, err)
				}

				chemical.ID = existing.ID
				chemical.OwnerID = existing.OwnerID
			}

			if chemical.ID == 0 {
				return fmt.Errorf("missing primary key for %q after upsert", canonicalName)
			}

			combinedNames, err := aggregateOtherNames(tx, chemical.ID, canonicalName, chemical.OtherNames, extraAliases)
			if err != nil {
				return fmt.Errorf("prepare other names for %q: %w", canonicalName, err)
			}

			target := models.AromaChemical{}
			target.ID = chemical.ID

			if len(combinedNames) > 0 {
				if err := tx.Model(&target).Association("OtherNames").Replace(combinedNames); err != nil {
					return fmt.Errorf("replace other names for %q: %w", canonicalName, err)
				}
			} else {
				if err := tx.Model(&target).Association("OtherNames").Clear(); err != nil {
					return fmt.Errorf("clear other names for %q: %w", canonicalName, err)
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf("record %d (%s): %w", idx+1, record["Ingredient Name"], err)
		}
		imported++
	}

	fmt.Fprintf(os.Stdout, "Imported %d aroma chemicals from %s\n", imported, filepath.Base(csvPath))
	return nil
}

func resolveImportOwner(db *gorm.DB) (uint, error) {
	if db == nil {
		return 0, fmt.Errorf("database handle is nil")
	}

	ctx := context.Background()
	email := strings.TrimSpace(os.Getenv("PERFUGO_AROMA_OWNER_EMAIL"))
	if email != "" {
		var user models.User
		if err := db.WithContext(ctx).Where("lower(email) = ?", strings.ToLower(email)).First(&user).Error; err != nil {
			return 0, fmt.Errorf("find owner by email %q: %w", strings.ToLower(email), err)
		}
		return user.ID, nil
	}

	var user models.User
	if err := db.WithContext(ctx).Order("id asc").First(&user).Error; err != nil {
		return 0, fmt.Errorf("find default owner: %w", err)
	}
	return user.ID, nil
}

func readCSV(path string) ([]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errors.New("csv is empty")
	}

	header := rows[0]
	records := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}

		record := make(map[string]string, len(header))
		for idx, key := range header {
			if idx >= len(row) {
				continue
			}
			value := strings.TrimSpace(row[idx])
			record[key] = value
		}
		records = append(records, record)
	}

	return records, nil
}

func buildAromaChemical(row map[string]string) models.AromaChemical {
	name := strings.TrimSpace(row["Ingredient Name"])
	casNumber := normalizeCAS(row["CAS Number"], name)

	chemical := models.AromaChemical{
		IngredientName:      name,
		CASNumber:           casNumber,
		Notes:               normalizeText(row["Notes"]),
		WheelPosition:       normalizeValue(row["Wheel Position"]),
		PyramidPosition:     normalizeValue(row["Pyramid Position"]),
		Type:                normalizeValue(row["Type"]),
		Strength:            mapStrength(row["Strength"]),
		RecommendedDilution: parseFirstNumber(row["Recommended Dilution"]),
		DilutionPercentage:  parseFirstNumber(row["Recommended Dilution"]),
		MaxIFRAPercentage:   parseFirstNumber(row["Max % in Concentrate (IFRA Cat. 4)"]),
		Duration:            normalizeValue(row["Duration (on blotter)"]),
		HistoricRole:        normalizeValue(row["Historic Role"]),
		Popularity:          mapPopularity(row["Popularity"]),
		Usage:               normalizeText(row["Usage"]),
	}

	chemical.OtherNames = buildOtherNames(row["Other Names"])

	return chemical
}

func normalizeValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return ""
	}
	return value
}

func normalizeText(value string) string {
	value = normalizeValue(value)
	if value == "" {
		return value
	}
	value = cleanWhitespace.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func parseFirstNumber(value string) float64 {
	value = normalizeValue(value)
	if value == "" {
		return 0
	}

	matches := numberPattern.FindString(value)
	if matches == "" {
		return 0
	}

	parsed, err := strconv.ParseFloat(matches, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func mapStrength(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "very low":
		return 1
	case "low":
		return 2
	case "low-medium":
		return 3
	case "medium":
		return 4
	case "medium-high":
		return 5
	case "high", "high (in effect)":
		return 6
	case "very high":
		return 7
	case "extreme":
		return 8
	default:
		return 0
	}
}

func mapPopularity(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "low (perfumery)", "specialist":
		return 1
	case "medium", "niche":
		return 2
	case "high", "foundational", "restricted", "restricted/banned", "banned/restricted", "high (endangered)":
		return 3
	case "high impact":
		return 4
	default:
		return 0
	}
}

func buildOtherNames(value string) []models.OtherName {
	value = normalizeValue(value)
	if value == "" {
		return nil
	}

	parts := splitOtherNames(value)
	names := make([]models.OtherName, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		clean := strings.TrimSpace(part)
		if clean == "" {
			continue
		}
		clean = stripFootnotes(clean)
		clean = strings.Trim(clean, ";,")
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}
		if _, ok := seen[strings.ToLower(clean)]; ok {
			continue
		}
		seen[strings.ToLower(clean)] = struct{}{}
		names = append(names, models.OtherName{Name: clean})
	}
	return names
}

func splitOtherNames(value string) []string {
	value = strings.ReplaceAll(value, ";", ",")
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, strings.TrimSpace(part))
	}
	return result
}

func stripFootnotes(value string) string {
	return strings.TrimSpace(bracketPattern.ReplaceAllString(value, ""))
}

func aggregateOtherNames(tx *gorm.DB, chemicalID uint, canonical string, newNames []models.OtherName, extra []string) ([]models.OtherName, error) {
	var current []models.OtherName
	if err := tx.Where("aroma_chemical_id = ?", chemicalID).Find(&current).Error; err != nil {
		return nil, err
	}

	nameMap := make(map[string]string)

	addName := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if strings.EqualFold(value, canonical) {
			return
		}
		key := strings.ToLower(value)
		if _, ok := nameMap[key]; !ok {
			nameMap[key] = value
		}
	}

	for _, entry := range current {
		addName(entry.Name)
	}

	for _, entry := range newNames {
		addName(entry.Name)
	}

	for _, alias := range extra {
		addName(alias)
	}

	if len(nameMap) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(nameMap))
	for key := range nameMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	combined := make([]models.OtherName, 0, len(keys))
	for _, key := range keys {
		combined = append(combined, models.OtherName{
			Name:            nameMap[key],
			AromaChemicalID: chemicalID,
		})
	}

	return combined, nil
}

func normalizeCAS(raw string, ingredient string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return syntheticCAS("UNASSIGNED", ingredient)
	}

	upper := strings.ToUpper(value)
	switch {
	case upper == "N/A", upper == "NA", upper == "NOT APPLICABLE", upper == "NOT ASSIGNED", upper == "UNKNOWN", upper == "NONE":
		return syntheticCAS("UNASSIGNED", ingredient)
	case strings.Contains(upper, "MIXTURE"):
		return syntheticCAS("MIXTURE", ingredient)
	case strings.Contains(upper, "BLEND"):
		return syntheticCAS("BLEND", ingredient)
	}

	return value
}

func syntheticCAS(prefix, ingredient string) string {
	slug := slugify(ingredient)
	if slug == "" {
		slug = "component"
	}
	return fmt.Sprintf("%s-%s", prefix, slug)
}

func slugify(value string) string {
	value = strings.ToLower(value)
	value = slugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}
