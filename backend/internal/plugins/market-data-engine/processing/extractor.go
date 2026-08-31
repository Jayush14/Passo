package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// Extractor handles extraction of structured data from raw job descriptions
// In a real implementation, this would interface with LLM providers
type Extractor struct {
	// LLM configuration
	APIKey     string
	Provider   string // e.g., "gemini", "openai"
	Model      string
	Endpoint   string // API endpoint
	useLLM     bool   // whether to use LLM extraction
}

// NewExtractor creates a new Extractor instance
func NewExtractor() *Extractor {
	// Read configuration from environment variables
	apiKey := os.Getenv("GEMINI_API_KEY")
	provider := os.Getenv("LLM_PROVIDER")
	model := os.Getenv("LLM_MODEL")
	endpoint := os.Getenv("LLM_ENDPOINT")
	useLLM := apiKey != "" && provider != ""

	return &Extractor{
		APIKey:   apiKey,
		Provider: strings.ToLower(provider),
		Model:    model,
		Endpoint: endpoint,
		useLLM:   useLLM,
	}
}

// ExtractJobData extracts structured job data from raw content
// Uses LLM if configured, otherwise falls back to rule-based extraction
func (e *Extractor) ExtractJobData(content string) (*models.ExtractedJobData, error) {
	if content == "" {
		return nil, fmt.Errorf("empty content provided")
	}

	// If LLM is configured, try to use it
	if e.useLLM {
		extracted, err := e.extractWithLLM(content)
		if err == nil {
			return extracted, nil
		}
		// If LLM fails, fall back to rule-based extraction
		// log.Printf("LLM extraction failed: %v, falling back to rule-based", err)
	}

	// Fallback to rule-based extraction (original implementation)
	return e.extractRuleBased(content)
}

// extractWithLLM uses the configured LLM provider to extract job data
func (e *Extractor) extractWithLLM(content string) (*models.ExtractedJobData, error) {
	// Build prompt for LLM
	prompt := e.buildExtractionPrompt(content)

	// Call LLM based on provider
	var response string
	var err error

	ctx := context.Background()
	switch e.Provider {
	case "gemini":
		response, err = e.callGeminiAPI(ctx, prompt)
	case "openai":
		response, err = e.callOpenAIAPI(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", e.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %v", err)
	}

	// Parse JSON response
	var extracted models.ExtractedJobData
	if err := json.Unmarshal([]byte(response), &extracted); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %v", err)
	}

	// Validate essential fields
	if extracted.Title.Value == "" {
		return nil, fmt.Errorf("LLM failed to extract job title")
	}

	return &extracted, nil
}

// buildExtractionPrompt creates a prompt for the LLM to extract job data
func (e *Extractor) buildExtractionPrompt(jobDescription string) string {
	return `
You are an expert HR technician tasked with extracting structured information from job descriptions for skills gap analysis.

Given the following job description, extract the information in the exact JSON format specified below.

Job Description:
"` + jobDescription + `"

Extract and return ONLY a valid JSON object with this exact structure (no additional text before or after):
{
  "title": {
    "value": "job title string",
    "confidence": float between 0 and 1,
    "evidence": "supporting text from job description for the title"
  },
  "company": {
    "value": "company name string or null if not found",
    "confidence": float between 0 and 1,
    "evidence": "supporting text from job description for the company or null if not found"
  },
  "location": {
    "city": "city string or null if not found",
    "state": "state string or null if not found",
    "country": "country string or null if not found",
    "confidence": float between 0 and 1,
    "evidence": "supporting text from job description for the location"
  },
  "employment_type": {
    "value": "employment type string (full_time, part_time, contract, temporary, internship, etc.) or null if not found",
    "confidence": float between 0 and 1,
    "evidence": "supporting text from job description for employment type or null if not found"
  },
  "work_mode": {
    "value": "work mode string (remote, hybrid, onsite) or null if not found",
    "confidence": float between 0 and 1,
    "evidence": "supporting text from job description for work mode or null if not found"
  },
  "seniority": {
    "value": "seniority level string (entry, junior, mid, senior, lead, staff, principal, executive) or null if not found",
    "confidence": float between 0 and 1,
    "evidence": "supporting text from job description for seniority or null if not found"
  },
  "experience": {
    "minimum_years": integer or null if not specified,
    "maximum_years": integer or null if not specified,
    "confidence": float between 0 and 1,
    "evidence": "supporting text from job description for experience requirements"
  },
  "skills": [
    {
      "name": "skill name string (e.g., \"JavaScript\", \"React\", \"SQL\", \"Docker\", \"Go\", \"Java\")",
      "requirement": "required, preferred, or nice_to_have",
      "confidence": float between 0 and 1,
      "evidence": "supporting text from job description for this skill"
    }
  ],
  "responsibilities": [
    {
      "description": "responsibility string (action-oriented phrase from responsibilities section)",
      "category": "engineering, design, collaboration, quality_assurance, problem_solving, general, or null"
    }
  ],
  "soft_skills": [
    {
      "name": "soft skill name string (e.g., \"Communication\", \"Teamwork\", \"Leadership\", \"Problem Solving\")",
      "evidence": "supporting text from job description for this soft skill"
    }
  ]
}

Critical extraction guidelines:
1. Extract ALL mentioned technical skills, programming languages, frameworks, tools, technologies, and methodologies
2. Correctly classify requirement levels:
   - Keywords like "required", "must have", "essential", "mandatory" → "required"
   - Keywords like "preferred", "nice to have", "bonus", "plus", "desirable" → "preferred"
   - Look for explicit section headers like "Requirements", "Qualifications", "Nice to have", "Preferred qualifications"
   - If a skill is mentioned in a "Nice to have" section, classify it as "nice_to_have"
3. For experience, extract numbers from phrases like:
   - "3+ years of experience" → minimum_years: 3
   - "2-5 years experience" → minimum_years: 2, maximum_years: 5
   - "5 years experience required" → minimum_years: 5, maximum_years: 5
   - If only a maximum is stated (e.g., "up to 3 years"), set minimum_years: null
   - If only a minimum is stated (e.g., "at least 2 years"), set maximum_years: null
4. Location should be parsed from formats like "City, State, Country" or "City, Country"
5. Employment types: full_time, part-time, contract, temporary, internship, freelance
6. Work modes: remote, hybrid, onsite
7. If information cannot be found or is ambiguous, use null rather than guessing
8. The company name should be extracted from phrases like "We are [Company]", "[Company] is hiring", "at [Company]", etc.

Return ONLY the JSON object. Do not include any explanations, markdown formatting, or additional text.
`
}

// callGeminiAPI calls the Gemini API to generate text
func (e *Extractor) callGeminiAPI(ctx context.Context, prompt string) (string, error) {
	// Use Gemini 1.5 Flash model if not specified
	model := e.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, e.APIKey)

	// Request body
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.1,
			"topK":        1,
			"topP":        0.8,
			"maxOutputTokens": 2048,
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqJSON)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	// Parse response
	var geminiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %v", err)
	}

	// Extract text from candidates
	candidates, ok := geminiResp["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}
	firstCandidate := candidates[0].(map[string]interface{})
	contentMap, ok := firstCandidate["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no content in candidate")
	}
	parts := contentMap["parts"].([]interface{})
	if len(parts) == 0 {
		return "", fmt.Errorf("no parts in content")
	}
	firstPart := parts[0].(map[string]interface{})
	text, ok := firstPart["text"].(string)
	if !ok {
		return "", fmt.Errorf("no text in part")
	}

	return text, nil
}

// callOpenAIAPI calls the OpenAI API to generate text
func (e *Extractor) callOpenAIAPI(ctx context.Context, prompt string) (string, error) {
	// Use GPT-3.5-turbo if not specified
	model := e.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	url := "https://api.openai.com/v1/chat/completions"

	// Request body
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.1,
		"max_tokens":  2048,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqJSON)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	// Parse response
	var openaiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %v", err)
	}

	// Extract text from choices
	choices, ok := openaiResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	firstChoice := choices[0].(map[string]interface{})
	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no message in choice")
	}
	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("no content in message")
	}

	return content, nil
}

// extractRuleBased is the original rule-based extraction logic (kept as fallback)
func (e *Extractor) extractRuleBased(content string) (*models.ExtractedJobData, error) {
	// Parse location from the known format in the data
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	city, state, country := "", "", ""
	employmentType := ""
	workMode := ""

	if len(lines) > 2 {
		locationLine := strings.TrimSpace(lines[2])
		if locationLine != "" {
			// Format: "Bengaluru, Karnataka, India · 2 days ago"
			locationParts := strings.Split(locationLine, "·")
			if len(locationParts) > 0 {
				locationStr := strings.TrimSpace(locationParts[0])
				parts := strings.Split(locationStr, ",")
				if len(parts) >= 1 {
					city = strings.TrimSpace(parts[0])
				}
				if len(parts) >= 2 {
					state = strings.TrimSpace(parts[1])
				}
				if len(parts) >= 3 {
					country = strings.TrimSpace(parts[2])
				}
			}

			// Parse employment info from the same line or next line
			if len(locationParts) > 1 {
				secondPart := strings.TrimSpace(locationParts[1])
				// Check for time ago pattern
				if strings.Contains(strings.ToLower(secondPart), "ago") ||
				   strings.Contains(strings.ToLower(secondPart), "day") ||
				   strings.Contains(strings.ToLower(secondPart), "hour") ||
				   strings.Contains(strings.ToLower(secondPart), "week") ||
				   strings.Contains(strings.ToLower(secondPart), "month") {
					// Time indicator present, employment info should be in next line(s)
					if len(lines) > 3 {
						empLine := strings.TrimSpace(lines[3])
						if empLine != "" {
							empParts := strings.Split(empLine, "·")
							if len(empParts) >= 1 {
								workModeStr := strings.TrimSpace(empParts[0])
								if workModeStr == "Hybrid" || workModeStr == "hybrid" {
									workMode = "hybrid"
								} else if workModeStr == "Remote" || workModeStr == "remote" {
									workMode = "remote"
								} else if workModeStr == "Onsite" || workModeStr == "onsite" ||
								    workModeStr == "Office" || workModeStr == "office" {
									workMode = "onsite"
								}
							}
							if len(empParts) >= 2 {
								empTypeStr := strings.TrimSpace(empParts[1])
								if empTypeStr == "Full-time" || empTypeStr == "full_time" {
									employmentType = "full_time"
								} else if empTypeStr == "Part-time" || empTypeStr == "part_time" {
									employmentType = "part_time"
								} else if empTypeStr == "Contract" || empTypeStr == "contract" {
									employmentType = "contract"
								}
							}
						}
					}
				} else {
					// No time indicator, second part might be employment type
					if secondPart == "Full-time" || secondPart == "full_time" {
						employmentType = "full_time"
					} else if secondPart == "Part-time" || secondPart == "part_time" {
						employmentType = "part_time"
					} else if secondPart == "Contract" || secondPart == "contract" {
						employmentType = "contract"
					}
				}
			}
		}
	}

	// Fallback parsing for employment info
	if employmentType == "" && len(lines) > 3 {
		workLine := strings.TrimSpace(lines[3])
		if workLine != "" {
			parts := strings.Split(workLine, "·")
			if len(parts) >= 1 {
				workModeStr := strings.TrimSpace(parts[0])
				if workModeStr == "Hybrid" || workModeStr == "hybrid" {
					workMode = "hybrid"
				} else if workModeStr == "Remote" || workModeStr == "remote" {
					workMode = "remote"
				} else if workModeStr == "Onsite" || workModeStr == "onsite" ||
				    workModeStr == "Office" || workModeStr == "office" {
					workMode = "onsite"
				}
			}
			if len(parts) >= 2 {
				empTypeStr := strings.TrimSpace(parts[1])
				if empTypeStr == "Full-time" || empTypeStr == "full_time" {
					employmentType = "full_time"
				} else if empTypeStr == "Part-time" || empTypeStr == "part_time" {
					employmentType = "part_time"
				} else if empTypeStr == "Contract" || empTypeStr == "contract" {
					employmentType = "contract"
				}
			}
		}
	}

	// Extract title
	title := ""
	if len(lines) > 0 {
		titleLine := strings.TrimSpace(lines[0])
		if titleLine != "" {
			title = titleLine
		}
	}

	// Extract company (simplified)
	company := ""
	// Look for patterns like "We are [Company]" or "at [Company]" or "[Company] is hiring"
	patterns := []string{
		`(?i)we\s+are\s+([A-Z][a-zA-Z0-9\s&\.]+?)(?:\s+is|\s+to|\s+looking|\s+hiring|\.|,)`,
		`(?i)at\s+([A-Z][a-zA-Z0-9\s&\.]+?)(?:\s+is|\s+to|\s+looking|\s+hiring|\.|,|$)`,
		`(?i)^([A-Z][a-zA-Z0-9\s&\.]+?)\s+is\s+hiring`,
		`(?i)([A-Z][a-zA-Z0-9\s&\.]+?)\s+is\s+looking\s+for`,
		`(?i)([A-Z][a-zA-Z0-9\s&\.]+?)\s+are\s+hiring`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 && matches[1] != "" {
			company := strings.TrimSpace(matches[1])
			// Filter out false positives and clean up
			company = strings.TrimRight(company, ".")
			if !strings.Contains(strings.ToLower(company), "the ") &&
			   !strings.Contains(strings.ToLower(company), "a ") &&
			   !strings.Contains(strings.ToLower(company), "an ") &&
			   len(company) > 1 && len(company) < 100 &&
			   !strings.Contains(strings.ToLower(company), "role") &&
			   !strings.Contains(strings.ToLower(company), "position") &&
			   !strings.Contains(strings.ToLower(company), "engineer") &&
			   !strings.Contains(strings.ToLower(company), "developer") &&
			   !strings.Contains(strings.ToLower(company), "manager") &&
			   !strings.Contains(strings.ToLower(company), "analyst") &&
			   !strings.Contains(strings.ToLower(company), "specialist") &&
			   !strings.Contains(strings.ToLower(company), "you will") &&
			   !strings.Contains(strings.ToLower(company), "what you") &&
			   !strings.Contains(strings.ToLower(company), "responsibilit") {
				break
			}
		}
	}

	// For now, return empty skills, responsibilities, and softskills as placeholders
	// The rule-based extraction for skills is complex and we'd rather rely on LLM
	// In a real implementation, we would restore the previous skill extraction logic
	return &models.ExtractedJobData{
		Title: models.TitleField{
			Value:   title,
			Confidence: 0.5, // Low confidence for rule-based title
			Evidence:  title,
		},
		Company: models.CompanyField{
			Value:     func() interface{} { if company == "" { return nil }; return company }(),
			Confidence: func() float64 { if company == "" { return 0 }; return 0.5 }(),
			Evidence:  func() interface{} { if company == "" { return nil }; return company }(),
		},
		Location: models.LocationField{
			City:      city,
			State:     state,
			Country:   country,
			Confidence: 0.5,
			Evidence:  strings.TrimSpace(lines[2]),
		},
		EmploymentType: models.EmploymentTypeField{
			Value:     employmentType,
			Confidence: func() float64 { if employmentType == "" { return 0 }; return 0.5 }(),
			Evidence:  func() string { if employmentType == "" { return "" }; return employmentType }(),
		},
		WorkMode: models.WorkModeField{
			Value:     workMode,
			Confidence: func() float64 { if workMode == "" { return 0 }; return 0.5 }(),
			Evidence:  func() string { if workMode == "" { return "" }; return workMode }(),
		},
		Seniority: models.SeniorityField{
			Value:     "", // Leave empty for rule-based fallback
			Confidence: 0.0,
			Evidence:    "",
		},
		Experience: models.ExperienceField{
			MinimumYears: nil,
			MaximumYears: nil,
			Confidence:   0.0,
			Evidence:     "",
		},
		Education: nil,
		Skills:    []models.SkillField{}, // Empty for now
		Responsibilities: []models.ResponsibilityField{},
		SoftSkills:  []models.SoftSkillField{},
	}, nil
}