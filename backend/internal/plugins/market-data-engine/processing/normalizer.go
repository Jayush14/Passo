package processing

import (
	"fmt"

	"passo/backend/internal/plugins/market-data-engine/models"
	"passo/backend/internal/plugins/market-data-engine/taxonomy"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Normalizer handles normalization of extracted job data using deterministic matching
// This is the basic normalization service for Phase 3
type Normalizer struct {
	taxonomyService *taxonomy.TaxonomyService
	matcher         *taxonomy.Matcher
}

// NewNormalizer creates a new Normalizer instance
func NewNormalizer(db *pgxpool.Pool) *Normalizer {
	taxonomyService := taxonomy.NewTaxonomyService(db)
	return &Normalizer{
		taxonomyService: taxonomyService,
		matcher:         taxonomy.NewMatcher(taxonomyService),
	}
}

// NormalizeSkills takes extracted skills and returns normalized, standardized skills
func (n *Normalizer) NormalizeSkills(extractedSkills []models.SkillField) []models.NormalizedSkill {
	// Convert SkillField to SkillMatch for the matcher
	var skillMatches []taxonomy.SkillMatch
	for _, skill := range extractedSkills {
		skillMatches = append(skillMatches, taxonomy.SkillMatch{
			RawName:           skill.Name,
			AlternativeNames:  []string{}, // In a real implementation, these would come from context
			RequirementLevel:  skill.Requirement,
			Evidence:          skill.Evidence,
		})
	}

	// Use matcher to normalize skills
	normalized := n.matcher.MatchSkills(skillMatches)

	// Convert back to the expected format
	var result []models.NormalizedSkill
	for _, normSkill := range normalized {
		var extCap *models.ExtractedCapability
		if normSkill.Capability != nil {
			// Convert from *models.Capability to *models.ExtractedCapability
			extCap = &models.ExtractedCapability{
				ID:              normSkill.Capability.ID,
				CanonicalName:   normSkill.Capability.Name,
				Category:        normSkill.Capability.Category,
				Subcategory:     "", // Not in base Capability model
				Description:     normSkill.Capability.Description,
				ParentID:        nil, // Not in base Capability model
				LevelDefinition: nil, // Not in base Capability model
				Status:          "active", // Default status
				TaxonomyVersion: normSkill.Capability.TaxonomyVersion.String(),
				CreatedAt:       normSkill.Capability.CreatedAt,
				UpdatedAt:       normSkill.Capability.UpdatedAt,
			}
		}
		result = append(result, models.NormalizedSkill{
			RawName:           normSkill.RawName,
			NormalizedName:    normSkill.NormalizedName,
			Capability:        extCap,
			Confidence:        normSkill.Confidence,
			RequirementLevel:  normSkill.RequirementLevel,
			Evidence:          normSkill.Evidence,
			AlternativeNames:  normSkill.AlternativeNames,
		})
	}

	return result
}

// NormalizeSkill takes a single skill and returns its normalized form
func (n *Normalizer) NormalizeSkill(skill string) string {
	return n.taxonomyService.NormalizeSkill(skill)
}

// GetSkillCapability returns the capability associated with a skill
func (n *Normalizer) GetSkillCapability(skill string) *models.Capability {
	return n.taxonomyService.GetCapability(skill)
}

// GetRelatedSkills returns skills related to the given skill based on taxonomy
func (n *Normalizer) GetRelatedSkills(skill string) []*models.Capability {
	return n.taxonomyService.GetRelatedCapabilities(skill)
}

// IsValidSkill checks if a skill is valid according to the taxonomy
func (n *Normalizer) IsValidSkill(skill string) bool {
	return n.taxonomyService.IsValidSkill(skill)
}

// NormalizeExperienceLevel normalizes experience level strings (e.g., "3-5 years" -> "Mid-Level")
// This is a basic implementation - could be enhanced with more sophisticated logic
func (n *Normalizer) NormalizeExperienceLevel(experience string) string {
	if experience == "" {
		return "Unknown"
	}

	expLower := strings.ToLower(experience)

	// Extract years if mentioned
	var years float64

	switch {
	case strings.Contains(expLower, "year") || strings.Contains(expLower, "yr"):
		// Try to extract a number
		var num float64
		fmt.Sscanf(experience, "%f", &num)
		years = num
	case strings.Contains(expLower, "month") || strings.Contains(expLower, "mo"):
		// Convert months to years
		var num float64
		fmt.Sscanf(experience, "%f", &num)
		years = num / 12.0
	default:
		// No time unit found, treat as years if it's a number
		var num float64
		if _, err := fmt.Sscanf(experience, "%f", &num); err == nil {
			years = num
		} else {
			return experience // Return as-is if we can't parse it
		}
	}

	// Normalize based on years of experience
	switch {
	case years < 1:
		return "Entry-Level"
	case years < 3:
		return "Junior-Level"
	case years < 5:
		return "Mid-Level"
	case years < 7:
		return "Senior-Level"
	case years < 10:
		return "Lead-Level"
	default:
		return "Principal-Level"
	}
}

// NormalizeJobTitle normalizes job titles using known patterns and taxonomy
func (n *Normalizer) NormalizeJobTitle(title string) string {
	if title == "" {
		return ""
	}

	// Trim whitespace
	title = strings.TrimSpace(title)

	// Common title normalizations
	title = strings.ReplaceAll(title, "Sr.", "Senior")
	title = strings.ReplaceAll(title, "Jr.", "Junior")
	title = strings.ReplaceAll(title, "Lead ", "Lead ")
	title = strings.ReplaceAll(title, "Mgr.", "Manager")
	title = strings.ReplaceAll(title, "Manager", "Manager")
	title = strings.ReplaceAll(title, "Director", "Director")
	title = strings.ReplaceAll(title, "Head of", "Head of")
	title = strings.ReplaceAll(title, "VP", "Vice President")
	title = strings.ReplaceAll(title, "SVP", "Senior Vice President")
	title = strings.ReplaceAll(title, "C-", "Chief ")
	title = strings.ReplaceAll(title, "Chief ", "Chief ")

	// Ensure proper spacing
	title = strings.Join(strings.Fields(title), " ")

	return title
}

// NormalizeCompanyName normalizes company names
func (n *Normalizer) NormalizeCompanyName(company string) string {
	if company == "" {
		return ""
	}

	// Trim whitespace
	company = strings.TrimSpace(company)

	// Remove common suffixes/prefixes that don't add value
	company = strings.TrimSuffix(company, " Inc")
	company = strings.TrimSuffix(company, " Inc.")
	company = strings.TrimSuffix(company, " LLC")
	company = strings.TrimSuffix(company, " LLC.")
	company = strings.TrimSuffix(company, " Ltd")
	company = strings.TrimSuffix(company, " Ltd.")
	company = strings.TrimSuffix(company, " Limited")
	company = strings.TrimSuffix(company, " Corporation")
	company = strings.TrimSuffix(company, " Corp")
	company = strings.TrimSuffix(company, " Corp.")
	company = strings.TrimSuffix(company, " Company")
	company = strings.TrimSuffix(company, " Co")
	company = strings.TrimSuffix(company, " Co.")

	// Ensure proper spacing
	company = strings.Join(strings.Fields(company), " ")

	return company
}

// NormalizeLocation normalizes location strings (city, state, country)
func (n *Normalizer) NormalizeLocation(location string) string {
	if location == "" {
		return ""
	}

	// Trim whitespace
	location = strings.TrimSpace(location)

	// Split by common delimiters
	parts := []string{}
	if strings.Contains(location, ",") {
		parts = strings.Split(location, ",")
	} else if strings.Contains(location, "-") {
		parts = strings.Split(location, "-")
	} else {
		parts = []string{location}
	}

	// Trim each part
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// Rejoin with commas
	return strings.Join(parts, ", ")
}

// NormalizeJob takes extracted job data and returns normalized job data
func (n *Normalizer) NormalizeJob(extractedJob models.ExtractedJobData) models.NormalizedJobData {
	// Normalize basic fields
	normalizedTitle := n.NormalizeJobTitle(extractedJob.Title.Value)
	normalizedCompany := n.NormalizeCompanyName(func() string { if extractedJob.Company.Value == nil { return "" }; return extractedJob.Company.Value.(string) }())
	normalizedLocation := n.NormalizeLocation(extractedJob.Location.City + ", " + extractedJob.Location.State + ", " + extractedJob.Location.Country)

	// Normalize skills
	normalizedSkills := n.NormalizeSkills(extractedJob.Skills)

	// Normalize responsibilities (basic text normalization - keep categories as-is for now)
	var normalizedResp []models.NormalizedResponsibility
	for _, resp := range extractedJob.Responsibilities {
		normalizedResp = append(normalizedResp, models.NormalizedResponsibility{
			Description: strings.TrimSpace(resp.Responsibility),
			Category:    resp.Category, // Keep existing category - in a full implementation, this could be enhanced with taxonomy matching
		})
	}

	// Normalize soft skills
	var normalizedSoftSkills []models.NormalizedSoftSkill
	for _, softSkill := range extractedJob.SoftSkills {
		normalizedSoftSkills = append(normalizedSoftSkills, models.NormalizedSoftSkill{
			Name:  n.NormalizeSkill(softSkill.Name),
			Evidence: strings.TrimSpace(softSkill.Evidence),
		})
	}

	return models.NormalizedJobData{
		Title: models.TitleField{
			Value:   normalizedTitle,
			Confidence: extractedJob.Title.Confidence,
			Evidence:   normalizedTitle,
		},
		Company: models.CompanyField{
			Value:     func() interface{} { if normalizedCompany == "" { return nil }; return normalizedCompany }(),
			Confidence: extractedJob.Company.Confidence,
			Evidence:  func() interface{} { if normalizedCompany == "" { return nil }; return normalizedCompany }(),
		},
		Location: models.LocationField{
			City:      splitLocation(normalizedLocation, 0),
			State:     splitLocation(normalizedLocation, 1),
			Country:   splitLocation(normalizedLocation, 2),
			Confidence: extractedJob.Location.Confidence,
			Evidence:   normalizedLocation,
		},
		EmploymentType: models.EmploymentTypeField{
			Value:     extractedJob.EmploymentType.Value,
			Confidence: extractedJob.EmploymentType.Confidence,
			Evidence:  extractedJob.EmploymentType.Evidence,
		},
		WorkMode: models.WorkModeField{
			Value:     extractedJob.WorkMode.Value,
			Confidence: extractedJob.WorkMode.Confidence,
			Evidence:  extractedJob.WorkMode.Evidence,
		},
		Seniority: models.SeniorityField{
			Value:     n.NormalizeJobTitle(extractedJob.Seniority.Value), // Normalize seniority title
			Confidence: extractedJob.Seniority.Confidence,
			Evidence:   extractedJob.Seniority.Evidence,
		},
		Experience: models.ExperienceField{
			MinimumYears: extractedJob.Experience.MinimumYears,
			MaximumYears: extractedJob.Experience.MaximumYears,
			Confidence:   extractedJob.Experience.Confidence,
			Evidence:     extractedJob.Experience.Evidence,
		},
		Education: func() interface{} { if extractedJob.Education == nil { return nil }; return extractedJob.Education }(),
		Skills:      normalizedSkills,
		Responsibilities: normalizedResp,
		SoftSkills:  normalizedSoftSkills,
	}
}

// splitLocation splits a location string by commas and returns the specified part
func splitLocation(location string, partIndex int) string {
	parts := strings.Split(location, ",")
	if partIndex >= 0 && partIndex < len(parts) {
		return strings.TrimSpace(parts[partIndex])
	}
	return ""
}