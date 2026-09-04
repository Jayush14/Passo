package taxonomy

import (
	"passo/backend/internal/plugins/market-data-engine/models"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Matcher handles exact and fuzzy matching of skills and capabilities
type Matcher struct {
	taxonomyService *TaxonomyService
}

// NewMatcher creates a new Matcher instance
func NewMatcher(taxonomyService *TaxonomyService) *Matcher {
	return &Matcher{
		taxonomyService: taxonomyService,
	}
}

// ExactMatch checks if two skill strings match exactly after normalization
func (m *Matcher) ExactMatch(skill1, skill2 string) bool {
	return m.taxonomyService.NormalizeSkill(skill1) == m.taxonomyService.NormalizeSkill(skill2)
}

// FuzzyMatch performs fuzzy matching between two skill strings
// Returns a score between 0.0 and 1.0 where 1.0 is an exact match
func (m *Matcher) FuzzyMatch(skill1, skill2 string) float64 {
	if skill1 == "" || skill2 == "" {
		return 0.0
	}

	// Normalize both skills
	norm1 := m.taxonomyService.NormalizeSkill(skill1)
	norm2 := m.taxonomyService.NormalizeSkill(skill2)

	// Exact match after normalization
	if norm1 == norm2 {
		return 1.0
	}

	// Check if one is an alias of the other
	if m.taxonomyService.skillAliases[norm1] == norm2 || m.taxonomyService.skillAliases[norm2] == norm1 {
		return 0.95
	}

	// Check if they share a common parent in taxonomy
	cap1 := m.taxonomyService.GetCapability(norm1)
	cap2 := m.taxonomyService.GetCapability(norm2)
	if cap1 != nil && cap2 != nil {
		// Since we don't have parent/children relationships properly modeled in the basic capability struct,
		// we'll skip the ancestor/descendant checks for now and rely on string similarity
		// In a real implementation with a full taxonomy, we would check parent/child relationships here
	}

	// String similarity based on longest common subsequence (simplified)
	return m.stringSimilarity(norm1, norm2)
}

// isAncestor checks if cap1 is an ancestor of cap2 in the taxonomy
func (m *Matcher) isAncestor(ancestor, descendant *models.Capability) bool {
	if ancestor == nil || descendant == nil {
		return false
	}

	// Note: Since the Capability model in this basic implementation doesn't properly support
	// hierarchical taxonomy (no easy way to traverse parent/child relationships),
	// we'll return false for ancestor checks.
	// In a real implementation with a full taxonomy structure, we would walk up the parent chain.

	return false
}

// stringSimilarity calculates a simple similarity score between two strings
// Based on longest common subsequence (simplified version)
func (m *Matcher) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Convert to lowercase for case-insensitive comparison
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// Remove non-alphanumeric characters for better matching
	s1 = m.removeNonAlphanumeric(s1)
	s2 = m.removeNonAlphanumeric(s2)

	if s1 == s2 {
		return 0.9
	}

	// Check if one contains the other
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		// Calculate ratio of shorter to longer length
		len1, len2 := float64(len(s1)), float64(len(s2))
		if len1 < len2 {
			return 0.8 * (len1 / len2)
		}
		return 0.8 * (len2 / len1)
	}

	// Calculate similarity based on common prefixes/suffixes
	commonPrefix := m.commonPrefixLength(s1, s2)
	commonSuffix := m.commonSuffixLength(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))

	if maxLen == 0 {
		return 0.0
	}

	similarity := (float64(commonPrefix+commonSuffix) / (2 * maxLen)) * 0.6
	return similarity
}

// removeNonAlphanumeric removes non-alphanumeric characters from a string
func (m *Matcher) removeNonAlphanumeric(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// commonPrefixLength returns the length of the common prefix between two strings
func (m *Matcher) commonPrefixLength(s1, s2 string) int {
	minLen := min(len(s1), len(s2))
	for i := 0; i < minLen; i++ {
		if s1[i] != s2[i] {
			return i
		}
	}
	return minLen
}

// commonSuffixLength returns the length of the common suffix between two strings
func (m *Matcher) commonSuffixLength(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)
	minLen := min(len1, len2)
	for i := 0; i < minLen; i++ {
		if s1[len1-1-i] != s2[len2-1-i] {
			return i
		}
	}
	return minLen
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FindBestMatch finds the best matching capability from a list of candidates
func (m *Matcher) FindBestMatch(target string, candidates []string) (*models.Capability, float64) {
	if target == "" || len(candidates) == 0 {
		return nil, 0.0
	}

	// First try exact match through taxonomy
	if cap := m.taxonomyService.GetCapability(target); cap != nil {
		return cap, 1.0
	}

	bestMatch := m.taxonomyService.GetCapability(target)
	bestScore := 0.0

	// Check against candidates
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		score := m.FuzzyMatch(target, candidate)
		if score > bestScore {
			bestScore = score
			if cap := m.taxonomyService.GetCapability(candidate); cap != nil {
				bestMatch = cap
			}
		}
	}

	// Also check against all known capabilities if no good candidate match
	if bestScore < 0.8 {
		for _, capability := range m.taxonomyService.GetAllCapabilities() {
			// Check canonical name
			score := m.FuzzyMatch(target, capability.Name)
			if score > bestScore {
				bestScore = score
				bestMatch = capability
			}

			// Note: Since the Capability model doesn't have an Aliases field in this basic implementation,
			// we cannot check aliases here. In a real implementation, we would check against known aliases.
			// For now, we rely on the taxonomyService's skillAliases map for alias resolution elsewhere.
		}
	}

	// If still no good match, return the normalized form as a capability with low confidence
	if bestMatch == nil && bestScore < 0.3 {
		normalized := m.taxonomyService.NormalizeSkill(target)
		if normalized != "" {
			// Create a temporary capability for unknown skills
			// Note: We don't have models.NilUUID, so we'll create a new UUID using NewRandom()
			tempID, _ := uuid.NewRandom()
			bestMatch = &models.Capability{
				ID:              tempID,
				Name:            normalized,
				Category:        "unknown",
				Description:     "Temporary capability for unknown skill",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				TaxonomyVersion: uuid.Nil,
			}
			bestScore = 0.3 // Low confidence for unknown skill
		}
	}

	return bestMatch, bestScore
}

// MatchSkills normalizes a slice of skill matches using the matcher's capabilities
func (m *Matcher) MatchSkills(skillMatches []SkillMatch) []NormalizedSkill {
	var normalizedSkills []NormalizedSkill

	for _, skillMatch := range skillMatches {
		if skillMatch.RawName == "" {
			continue
		}

		// Prepare candidates: AlternativeNames plus the RawName itself
		var candidates []string
		candidates = append(candidates, skillMatch.RawName)
		candidates = append(candidates, skillMatch.AlternativeNames...)

		// Find the best matching capability
		bestCap, confidence := m.FindBestMatch(skillMatch.RawName, candidates)

		// Build the normalized skill
		normalizedSkill := NormalizedSkill{
			RawName:          skillMatch.RawName,
			NormalizedName:   "", // Will be set below
			Capability:       bestCap,
			Confidence:       confidence,
			RequirementLevel: skillMatch.RequirementLevel,
			Evidence:         skillMatch.Evidence,
			AlternativeNames: skillMatch.AlternativeNames,
		}

		// Set the normalized name based on the capability or fallback to raw name
		if bestCap != nil {
			normalizedSkill.NormalizedName = bestCap.Name
		} else {
			// If no capability found, use the normalized form of the raw name
			normalizedSkill.NormalizedName = m.taxonomyService.NormalizeSkill(skillMatch.RawName)
		}

		normalizedSkills = append(normalizedSkills, normalizedSkill)
	}

	return normalizedSkills
}

// SkillMatch represents a skill as extracted from raw data
type SkillMatch struct {
	RawName          string   // Original skill name as found in text
	AlternativeNames []string // Alternative names found in context
	RequirementLevel string   // "required" or "preferred"
	Evidence         string   // Evidence text supporting this skill
}

// NormalizedSkill represents a skill after normalization and taxonomy matching
type NormalizedSkill struct {
	RawName          string             // Original skill name as found in text
	NormalizedName   string             // Standardized skill name
	Capability       *models.Capability // Matched capability from taxonomy
	Confidence       float64            // Confidence in the match (0.0-1.0)
	RequirementLevel string             // "required" or "preferred"
	Evidence         string             // Evidence text supporting this skill
	AlternativeNames []string           // Alternative names found in context
}
