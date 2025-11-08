package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultModel       = "gpt-4.1-mini"
	defaultBaseURL     = "https://api.openai.com/v1"
	defaultTemperature = 0.2
	defaultTimeout     = 90 * time.Second
)

// Config describes how the OpenAI client should be initialised.
type Config struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// Client offers a thin wrapper around the OpenAI Chat Completions API.
type Client struct {
	apiKey      string
	model       string
	baseURL     string
	temperature float64
	httpClient  *http.Client
}

// FetchOptions control per-request overrides.
type FetchOptions struct {
	ModelOverride string
}

// Profile captures the normalised aroma chemical data returned by OpenAI.
type Profile struct {
	IngredientName      string
	CASNumber           string
	OtherNames          []string
	Notes               string
	WheelPosition       string
	PyramidPosition     string
	Type                string
	Strength            int
	RecommendedDilution float64
	DilutionPercentage  float64
	MaxIFRAPercentage   float64
	Duration            string
	HistoricRole        string
	Popularity          int
	Usage               string
}

// NewClient builds a Client that can query OpenAI for aroma data.
func NewClient(cfg Config) (*Client, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, errors.New("ai: api key must not be empty")
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	temp := cfg.Temperature
	if temp <= 0 {
		temp = defaultTemperature
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	return &Client{
		apiKey:      apiKey,
		model:       model,
		baseURL:     strings.TrimRight(baseURL, "/"),
		temperature: temp,
		httpClient:  httpClient,
	}, nil
}

// FetchAromaProfile contacts OpenAI and returns a normalised aroma profile.
func (c *Client) FetchAromaProfile(ctx context.Context, ingredient string, opts FetchOptions) (Profile, error) {
	ingredient = strings.TrimSpace(ingredient)
	if ingredient == "" {
		return Profile{}, errors.New("ai: ingredient name must not be empty")
	}

	payload := map[string]any{
		"model":       c.effectiveModel(opts),
		"temperature": c.temperature,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are an expert perfumery researcher. Provide compact, fact-checked ingredient data in JSON only.",
			},
			{
				"role":    "user",
				"content": buildPrompt(ingredient),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Profile{}, fmt.Errorf("ai: encode request: %w", err)
	}

	content, err := c.performChatCompletion(ctx, payload, body)
	if err != nil {
		return Profile{}, err
	}
	var parsed aiAromaResponse
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return Profile{}, fmt.Errorf("ai: parse JSON payload: %w", err)
	}

	return normaliseAromaData(ingredient, parsed)
}

func (c *Client) effectiveModel(opts FetchOptions) string {
	model := strings.TrimSpace(opts.ModelOverride)
	if model != "" {
		return model
	}
	return c.model
}

func buildPrompt(ingredient string) string {
	return fmt.Sprintf(`Return JSON describing the aroma chemical "%s". Fields:
{
  "ingredient_name": string,
  "cas_number": string | "",
  "other_names": string[] (omit unverified aliases),
  "notes": short string,
  "wheel_position": string (taxonomy category similar to the scent wheel),
  "pyramid_position": string (Top, Heart, Base, etc.),
  "type": string (e.g., "Aroma Chemical (Lactone)"),
  "strength_label": string from {Very Low, Low, Low-Medium, Medium, Medium-High, High, Very High, Extreme},
  "recommended_dilution_percent": number (percent of concentrate, 0-100) or null if unknown,
  "dilution_percent": number (same as recommended_dilution_percent),
  "max_ifra_cat4_percent": number (0-100) or null if not restricted,
  "duration_description": string (e.g., "45 hours"),
  "historic_role": string,
  "popularity_label": string from {Low, Medium, High, High Impact, Specialist, Niche, Foundational, Restricted},
  "usage": concise guidance string
}
Strict rules: respond with raw JSON, no Markdown, no comments. Use empty string instead of unknown text fields. Use empty list for other_names if none.`, ingredient)
}

type aiAromaResponse struct {
	IngredientName         string            `json:"ingredient_name"`
	CASNumber              string            `json:"cas_number"`
	OtherNames             any               `json:"other_names"`
	Notes                  string            `json:"notes"`
	WheelPosition          string            `json:"wheel_position"`
	PyramidPosition        string            `json:"pyramid_position"`
	Type                   string            `json:"type"`
	StrengthLabel          string            `json:"strength_label"`
	RecommendedDilution    any               `json:"recommended_dilution_percent"`
	DilutionPercentage     any               `json:"dilution_percent"`
	MaxIFRAPercentage      any               `json:"max_ifra_cat4_percent"`
	Duration               string            `json:"duration_description"`
	HistoricRole           string            `json:"historic_role"`
	PopularityLabel        string            `json:"popularity_label"`
	Usage                  string            `json:"usage"`
	AdditionalInstructions map[string]string `json:"_note,omitempty"`
}

func normaliseAromaData(requestedName string, aiData aiAromaResponse) (Profile, error) {
	name := strings.TrimSpace(aiData.IngredientName)
	if name == "" {
		name = strings.TrimSpace(requestedName)
	}
	if name == "" {
		return Profile{}, errors.New("ai: ingredient name missing from response")
	}

	result := Profile{
		IngredientName:      normaliseText(name),
		CASNumber:           normaliseValue(aiData.CASNumber),
		Notes:               normaliseText(aiData.Notes),
		WheelPosition:       normaliseValue(aiData.WheelPosition),
		PyramidPosition:     normaliseValue(aiData.PyramidPosition),
		Type:                normaliseValue(aiData.Type),
		Duration:            normaliseValue(aiData.Duration),
		HistoricRole:        normaliseValue(aiData.HistoricRole),
		Usage:               normaliseText(aiData.Usage),
		Strength:            mapStrength(aiData.StrengthLabel),
		Popularity:          mapPopularity(aiData.PopularityLabel),
		RecommendedDilution: parseNumeric(aiData.RecommendedDilution),
		DilutionPercentage:  parseNumeric(aiData.DilutionPercentage),
		MaxIFRAPercentage:   parseNumeric(aiData.MaxIFRAPercentage),
		OtherNames:          sanitiseOtherNames(aiData.OtherNames, name),
	}

	if result.RecommendedDilution == 0 {
		result.RecommendedDilution = result.DilutionPercentage
	}
	if result.DilutionPercentage == 0 {
		result.DilutionPercentage = result.RecommendedDilution
	}

	return result, nil
}

func normaliseValue(value string) string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "n/a", "na", "none":
		return ""
	default:
		return value
	}
}

func normaliseText(value string) string {
	value = normaliseValue(value)
	if value == "" {
		return ""
	}
	return strings.Join(strings.Fields(value), " ")
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

func parseNumeric(value any) float64 {
	switch v := value.(type) {
	case nil:
		return 0
	case float64:
		return v
	case json.Number:
		parsed, err := strconv.ParseFloat(v.String(), 64)
		if err != nil {
			return 0
		}
		return parsed
	case string:
		return parseFirstNumber(v)
	default:
		return 0
	}
}

func parseFirstNumber(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	match := numberPattern.FindString(value)
	if match == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0
	}
	return parsed
}

var numberPattern = regexp.MustCompile(`[-+]?\d*\.?\d+`)

func sanitiseOtherNames(raw any, canonical string) []string {
	canonical = strings.ToLower(strings.TrimSpace(canonical))
	unique := make(map[string]struct{})
	result := []string{}

	add := func(value string) {
		value = normaliseValue(value)
		if value == "" {
			return
		}
		if strings.ToLower(value) == canonical {
			return
		}
		key := strings.ToLower(value)
		if _, ok := unique[key]; ok {
			return
		}
		unique[key] = struct{}{}
		result = append(result, value)
	}

	switch values := raw.(type) {
	case []any:
		for _, entry := range values {
			if s, ok := entry.(string); ok {
				add(s)
			}
		}
	case []string:
		for _, entry := range values {
			add(entry)
		}
	case nil:
	default:
		if s, ok := values.(string); ok {
			for _, part := range strings.Split(s, ",") {
				add(part)
			}
		}
	}

	return result
}

func (c *Client) performChatCompletion(ctx context.Context, payload map[string]any, preEncoded ...[]byte) (string, error) {
	var body []byte
	var err error
	if len(preEncoded) > 0 && preEncoded[0] != nil {
		body = preEncoded[0]
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("ai: encode request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai: call openai: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("ai: openai returned status %s", resp.Status)
	}

	var responseData struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return "", fmt.Errorf("ai: decode response: %w", err)
	}

	if len(responseData.Choices) == 0 {
		return "", errors.New("ai: openai returned no choices")
	}

	content := strings.TrimSpace(responseData.Choices[0].Message.Content)
	content = strings.Trim(content, "`")
	return strings.TrimSpace(content), nil
}
