package ai

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
)

// FormulaImportInput describes the source provided by the user when importing formulas.
type FormulaImportInput struct {
    NameHint   string
    RawText    string
    Base64File string
    FileName   string
    FileType   string
}

// FormulaImportResult captures the parsed formula returned by the AI service.
type FormulaImportResult struct {
    FormulaName string                    `json:"formula_name"`
    Notes       string                    `json:"notes"`
    Ingredients []FormulaImportIngredient `json:"ingredients"`
}

// FormulaImportIngredient represents one ingredient entry from the AI response.
type FormulaImportIngredient struct {
    IngredientName string   `json:"ingredient_name"`
    OtherNames     []string `json:"other_names"`
    QuantityMG     float64  `json:"quantity_mg"`
    Notes          string   `json:"notes"`
}

// ExtractFormula asks the AI model to parse the provided material into a structured formula.
func (c *Client) ExtractFormula(ctx context.Context, input FormulaImportInput) (FormulaImportResult, error) {
    trimmedText := strings.TrimSpace(input.RawText)
    if trimmedText == "" && strings.TrimSpace(input.Base64File) == "" {
        return FormulaImportResult{}, errors.New("ai: formula import requires text or file content")
    }

    if input.Base64File != "" {
        if _, err := base64.StdEncoding.DecodeString(input.Base64File); err != nil {
            return FormulaImportResult{}, fmt.Errorf("ai: invalid base64 payload: %w", err)
        }
    }

    systemPrompt := `You are an assistant who converts perfumery formula references into precise JSON.
- Always OCR any provided base64 file content before analysing the formula data.
- If both text and files are provided, treat the text as authoritative while using the file for clarification.
- Extract the formula name, optional notes, and every ingredient mentioned.
- Convert all ingredient quantities to milligrams. If percentages are supplied, assume the sum represents 100 percent before converting to milligrams.
- When quantities are implicit, infer practical mg values that maintain the relative proportions.
- For each ingredient, include helpful alternate names detected in the source.
- Respond with strictly valid JSON using this schema:
{
  "formula_name": string,
  "notes": string,
  "ingredients": [
    {
      "ingredient_name": string,
      "other_names": [string],
      "quantity_mg": number,
      "notes": string
    }
  ]
}
- Never include explanations, markdown, or commentary outside of the JSON payload.`

    var builder strings.Builder
    if hint := strings.TrimSpace(input.NameHint); hint != "" {
        builder.WriteString("Formula hint: ")
        builder.WriteString(hint)
        builder.WriteString("\n\n")
    }
    if trimmedText != "" {
        builder.WriteString("Formula text:\n")
        builder.WriteString(trimmedText)
        builder.WriteString("\n\n")
    }
    if strings.TrimSpace(input.Base64File) != "" {
        builder.WriteString("Base64 file metadata: ")
        if input.FileName != "" {
            builder.WriteString(fmt.Sprintf("name=%s ", input.FileName))
        }
        if input.FileType != "" {
            builder.WriteString(fmt.Sprintf("type=%s ", input.FileType))
        }
        builder.WriteString("\n")
        builder.WriteString(input.Base64File)
    }

    payload := map[string]any{
        "model":       c.model,
        "temperature": c.temperature,
        "messages": []map[string]string{
            {
                "role":    "system",
                "content": systemPrompt,
            },
            {
                "role":    "user",
                "content": builder.String(),
            },
        },
    }

    content, err := c.performChatCompletion(ctx, payload)
    if err != nil {
        return FormulaImportResult{}, err
    }

    content = strings.TrimSpace(strings.Trim(content, "`"))

    var result FormulaImportResult
    if err := json.NewDecoder(strings.NewReader(content)).Decode(&result); err != nil {
        return FormulaImportResult{}, fmt.Errorf("ai: parse formula payload: %w", err)
    }

    return result, nil
}
