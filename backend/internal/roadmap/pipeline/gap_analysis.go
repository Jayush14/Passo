package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"passo/backend/internal/ai/provider/contract"
	"passo/backend/internal/roadmap/matrix"
)

// GapAnalysisInput represents the input to the gap analysis stage.
type GapAnalysisInput struct {
	UserID             int64
	TargetRole         string
	SkillProficiencies map[string]int // skill -> self-rated proficiency (1-5)
	ExperienceYears    int            // optional
	CurrentRole        string         // optional
}

// GapAnalysisOutput represents the output of the gap analysis stage.
type GapAnalysisOutput struct {
	SkillsAlreadyStrong []string   `json:"skills_already_strong"`
	TransferableSkills  []string   `json:"transferable_skills"`
	SkillGaps           []SkillGap `json:"skill_gaps"`
}

// SkillGap represents a single skill gap.
type SkillGap struct {
	Skill         string `json:"skill"`
	CurrentLevel  int    `json:"current_level"`  // 1-5, where 1 is beginner, 5 is expert
	RequiredLevel int    `json:"required_level"` // 1-5
	Priority      string `json:"priority"`       // critical, high, medium, low
}

// GapAnalyzer performs the gap analysis using an AI provider.
type GapAnalyzer struct {
	provider contract.TextGenerator
}

// NewGapAnalyzer creates a new GapAnalyzer with the given AI provider.
func NewGapAnalyzer(provider contract.TextGenerator) *GapAnalyzer {
	return &GapAnalyzer{provider: provider}
}

// Analyze performs the gap analysis.
// If the target role is found in the role matrix and the user has provided skill proficiencies,
// it computes the gap analytically. Otherwise, it falls back to the AI-based analysis.
func (ga *GapAnalyzer) Analyze(ctx context.Context, input GapAnalysisInput) (*GapAnalysisOutput, error) {
	// First, try to use the role matrix for an analytical gap analysis.
	if reqs, exists := matrix.GetRoleRequirements(input.TargetRole); exists && len(input.SkillProficiencies) > 0 {
		return ga.analyzeUsingMatrix(reqs, input)
	}

	// Fall back to AI-based analysis if role not found in matrix or no proficiency data provided.
	return ga.analyzeUsingAI(ctx, input)
}

// analyzeUsingMatrix computes the gap using the role matrix and user's self-assessed proficiencies.
func (ga *GapAnalyzer) analyzeUsingMatrix(reqs *matrix.RoleRequirements, input GapAnalysisInput) (*GapAnalysisOutput, error) {
	var skillsAlreadyStrong []string
	var transferableSkills []string
	var skillGaps []SkillGap

	for _, reqSkill := range reqs.Skills {
		// Get user's proficiency for this skill (default to 0 if not self-assessed)
		currentLevel := input.SkillProficiencies[reqSkill.Skill]
		if currentLevel == 0 {
			// User did not self-assess this skill; assume they don't have it (beginner level)
			currentLevel = 0
		}
		requiredLevel := reqSkill.RequiredLevel

		// Calculate the gap (required - current). Negative means user exceeds requirement.
		gap := requiredLevel - currentLevel

		switch {
		case gap <= 0:
			// User meets or exceeds the requirement
			skillsAlreadyStrong = append(skillsAlreadyStrong, reqSkill.Skill)
		case gap <= 1:
			// Small gap: transferable skill (needs some adaptation)
			transferableSkills = append(transferableSkills, reqSkill.Skill)
			skillGaps = append(skillGaps, SkillGap{
				Skill:         reqSkill.Skill,
				CurrentLevel:  currentLevel,
				RequiredLevel: requiredLevel,
				Priority:      "low",
			})
		case gap <= 2:
			// Medium gap
			skillGaps = append(skillGaps, SkillGap{
				Skill:         reqSkill.Skill,
				CurrentLevel:  currentLevel,
				RequiredLevel: requiredLevel,
				Priority:      "medium",
			})
		default: // gap >= 3
			// Large gap: critical
			skillGaps = append(skillGaps, SkillGap{
				Skill:         reqSkill.Skill,
				CurrentLevel:  currentLevel,
				RequiredLevel: requiredLevel,
				Priority:      "critical",
			})
		}
	}

	return &GapAnalysisOutput{
		SkillsAlreadyStrong: skillsAlreadyStrong,
		TransferableSkills:  transferableSkills,
		SkillGaps:           skillGaps,
	}, nil
}

// analyzeUsingAI performs the original AI-based gap analysis.
func (ga *GapAnalyzer) analyzeUsingAI(ctx context.Context, input GapAnalysisInput) (*GapAnalysisOutput, error) {
	// Create a prompt for the AI to analyze the gap.
// Get the list of current skills from the proficiency map
		var currentSkills []string
		for skill := range input.SkillProficiencies {
			currentSkills = append(currentSkills, skill)
		}
	prompt := fmt.Sprintf(`
Analyze the skill gap between the user's current profile and the target role.

User Profile:
- Current Role: %s
- Years of Experience: %d
- Current Skills: %v

Target Role: %s

Please provide a detailed gap analysis in the following JSON format:
{
  "skills_already_strong": ["skill1", "skill2", ...],
  "transferable_skills": ["skill1", "skill2", ...],
  "skill_gaps": [
    {
      "skill": "skill name",
      "current_level": 1-5,
      "required_level": 1-5,
      "priority": "critical|high|medium|low"
    }
  ]
}

Guidelines:
- skills_already_strong: Skills the user has at a level that meets or exceeds the target role requirements.
- transferable_skills: Skills the user has that are related but may need some adaptation for the target role.
- skill_gaps: Skills the user needs to develop for the target role.
- For each skill gap, assess the user's current level (1-5) and the required level (1-5) for the target role.
- Priority indicates how critical it is to address this gap (critical being highest).
- Be realistic and specific. Focus on skills that are truly relevant to the target role.
- Return ONLY the JSON object, no additional text.
`, input.CurrentRole, input.ExperienceYears, currentSkills, input.TargetRole)

	// Generate the analysis using the AI provider.
	result, err := ga.provider.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate gap analysis: %w", err)
	}

	// Parse the JSON response.
	var output GapAnalysisOutput
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return nil, fmt.Errorf("failed to parse gap analysis response: %w", err)
	}

	return &output, nil
}
