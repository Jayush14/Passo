package models

import (
	"time"
	"github.com/google/uuid"
)

// RawMarketData represents immutable raw input data
type RawMarketData struct {
	ID             uuid.UUID    `json:"id"`
	DataType       string       `json:"data_type"`
	RawContent     string       `json:"raw_content"`
	Metadata       interface{}  `json:"metadata,omitempty"`
	ContentHash    string       `json:"content_hash"`
	IngestionBatchID *uuid.UUID  `json:"ingestion_batch_id,omitempty"`
	ReceivedAt     time.Time    `json:"received_at"`
	Status         string       `json:"status"`
	ProcessedAt    *time.Time   `json:"processed_at,omitempty"`
	ErrorMessage   *string      `json:"error_message,omitempty"`
}

// ProcessingJob tracks the processing state of raw data
type ProcessingJob struct {
	ID           uuid.UUID `json:"id"`
	RawDataID    uuid.UUID `json:"raw_data_id"`
	Status       string    `json:"status"`
	Attempts     int       `json:"attempts"`
	LastError    *string   `json:"last_error,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ExtractedJobData represents the structured output from LLM extraction (Phase 2)
type ExtractedJobData struct {
	Title           TitleField           `json:"title"`
	Company         CompanyField         `json:"company"`
	Location        LocationField        `json:"location"`
	EmploymentType  EmploymentTypeField  `json:"employment_type"`
	WorkMode        WorkModeField        `json:"work_mode"`
	Seniority       SeniorityField       `json:"seniority"`
	Experience      ExperienceField      `json:"experience"`
	Education       interface{}          `json:"education,omitempty"`
	Skills          []SkillField         `json:"skills"`
	Responsibilities []ResponsibilityField `json:"responsibilities"`
	SoftSkills      []SoftSkillField     `json:"soft_skills"`
}

// NormalizedJobData represents the standardized output after taxonomy normalization (Phase 3)
type NormalizedJobData struct {
	Title           TitleField           `json:"title"`
	Company         CompanyField         `json:"company"`
	Location        LocationField        `json:"location"`
	EmploymentType  EmploymentTypeField  `json:"employment_type"`
	WorkMode        WorkModeField        `json:"work_mode"`
	Seniority       SeniorityField       `json:"seniority"`
	Experience      ExperienceField      `json:"experience"`
	Education       interface{}          `json:"education,omitempty"`
	Skills          []NormalizedSkill    `json:"skills"`
	Responsibilities []NormalizedResponsibility `json:"responsibilities"`
	SoftSkills      []NormalizedSoftSkill `json:"soft_skills"`
}

// TitleField represents a job title extraction
type TitleField struct {
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}

// CompanyField represents a company extraction
type CompanyField struct {
	Value     interface{} `json:"value,omitempty"`
	Confidence float64   `json:"confidence"`
	Evidence  interface{} `json:"evidence,omitempty"`
}

// LocationField represents a location extraction
type LocationField struct {
	City      string  `json:"city,omitempty"`
	State     string  `json:"state,omitempty"`
	Country   string  `json:"country,omitempty"`
	Confidence float64 `json:"confidence"`
	Evidence  string  `json:"evidence,omitempty"`
}

// EmploymentTypeField represents employment type extraction
type EmploymentTypeField struct {
	Value     string  `json:"value,omitempty"`
	Confidence float64 `json:"confidence"`
	Evidence  string  `json:"evidence,omitempty"`
}

// WorkModeField represents work mode extraction
type WorkModeField struct {
	Value     string  `json:"value,omitempty"`
	Confidence float64 `json:"confidence"`
	Evidence  string  `json:"evidence,omitempty"`
}

// SeniorityField represents a seniority level extraction
type SeniorityField struct {
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}

// ExperienceField represents experience requirements
type ExperienceField struct {
	MinimumYears *int    `json:"minimum_years,omitempty"`
	MaximumYears *int    `json:"maximum_years,omitempty"`
	Confidence   float64 `json:"confidence"`
	Evidence     string  `json:"evidence,omitempty"`
}

// SkillField represents a skill requirement
type SkillField struct {
	Name       string  `json:"name"`
	Requirement string  `json:"requirement"` // required, preferred, etc.
	Importance float64 `json:"importance"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}

// ResponsibilityField represents a responsibility
type ResponsibilityField struct {
	Responsibility string  `json:"responsibility"`
	Category       string  `json:"category,omitempty"`
	Importance     float64 `json:"importance"`
	Confidence     float64 `json:"confidence"`
	Evidence       string  `json:"evidence,omitempty"`
}

// SoftSkillField represents a soft skill
type SoftSkillField struct {
	Name   string  `json:"name"`
	Evidence string  `json:"evidence,omitempty"`
}

// NormalizedSkill represents a skill after normalization and taxonomy matching
type NormalizedSkill struct {
	RawName           string  `json:"raw_name"`
	NormalizedName    string  `json:"normalized_name"`
	Capability        *ExtractedCapability `json:"capability,omitempty"`
	Confidence        float64 `json:"confidence"`
	RequirementLevel  string  `json:"requirement_level"` // "required" or "preferred"
	Evidence          string  `json:"evidence,omitempty"`
	AlternativeNames  []string `json:"alternative_names,omitempty"`
}

// NormalizedResponsibility represents a responsibility after normalization
type NormalizedResponsibility struct {
	Description string  `json:"description"`
	Category    string  `json:"category,omitempty"`
}

// NormalizedSoftSkill represents a soft skill after normalization
type NormalizedSoftSkill struct {
	Name   string  `json:"name"`
	Evidence string  `json:"evidence,omitempty"`
}

// ExtractedCapability represents a skill or competency from the extraction process
type ExtractedCapability struct {
	ID              uuid.UUID `json:"id"`
	CanonicalName   string    `json:"canonical_name"`
	Category        string    `json:"category"`
	Subcategory     string    `json:"subcategory,omitempty"`
	Description     string    `json:"description,omitempty"`
	ParentID        *uuid.UUID `json:"parent_id,omitempty"`
	LevelDefinition interface{} `json:"level_definition,omitempty"`
	Status          string    `json:"status"`
	TaxonomyVersion string    `json:"taxonomy_version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ExtractedCapabilityAlias represents an alternative name for a capability from extraction
type ExtractedCapabilityAlias struct {
	ID        uuid.UUID `json:"id"`
	CapabilityID uuid.UUID `json:"capability_id"`
	Alias       string    `json:"alias"`
	MatchType   string    `json:"match_type"`
	Confidence  float64   `json:"confidence"`
}

// ExtractedJobCapability represents the relationship between a job and a required capability from extraction
type ExtractedJobCapability struct {
	ID              uuid.UUID `json:"id"`
	JobID           uuid.UUID `json:"job_id"`
	CapabilityID    uuid.UUID `json:"capability_id"`
	RequirementType string    `json:"requirement_type"`
	Importance      float64   `json:"importance"`
	Confidence      float64   `json:"confidence"`
	EvidenceText    string    `json:"evidence_text"`
	SourceLocation  string    `json:"source_location"`
	ExtractionVersion string   `json:"extraction_version"`
	TaxonomyVersion string   `json:"taxonomy_version"`
	CreatedAt       time.Time `json:"created_at"`
}

// ExtractedJobResponsibility represents a responsibility extracted from a job description
type ExtractedJobResponsibility struct {
	ID          uuid.UUID `json:"id"`
	JobID       uuid.UUID `json:"job_id"`
	Responsibility string  `json:"responsibility"`
	Category      string    `json:"category"`
	Importance    float64   `json:"importance"`
	Confidence    float64   `json:"confidence"`
	EvidenceText  string    `json:"evidence_text"`
	CreatedAt     time.Time `json:"created_at"`
}

// ExtractedRoleProfile represents an aggregated profile for a specific role from extraction
type ExtractedRoleProfile struct {
	ID          uuid.UUID `json:"id"`
	RoleName    string    `json:"role_name"`
	Seniority   string    `json:"seniority"`
	ProfileVersion string  `json:"profile_version"`
	SampleSize    int       `json:"sample_size"`
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	CreatedAt     time.Time `json:"created_at"`
}

// ExtractedRoleProfileCapability represents the relationship between a role profile and a capability from extraction
type ExtractedRoleProfileCapability struct {
	ID          uuid.UUID `json:"id"`
	RoleProfileID uuid.UUID `json:"role_profile_id"`
	CapabilityID  uuid.UUID `json:"capability_id"`
	Frequency     float64   `json:"frequency"`
	RequiredRate  float64   `json:"required_rate"`
	PreferredRate float64   `json:"preferred_rate"`
	AverageImportance float64 `json:"average_importance"`
	TrendScore    float64   `json:"trend_score"`
	Confidence    float64   `json:"confidence"`
	CreatedAt     time.Time `json:"created_at"`
}

// ExtractedMarketTrend represents time-series data on capability demand from extraction
type ExtractedMarketTrend struct {
	ID          uuid.UUID `json:"id"`
	CapabilityID uuid.UUID `json:"capability_id"`
	RoleID      uuid.UUID `json:"role_id"`
	Period      string    `json:"period"`
	JobCount    int       `json:"job_count"`
	MentionCount int      `json:"mention_count"`
	RequiredCount int     `json:"required_count"`
	Frequency     float64   `json:"frequency"`
	GrowthRate    float64   `json:"growth_rate"`
	TrendDirection string  `json:"trend_direction"`
	CreatedAt     time.Time `json:"created_at"`
}

// ExtractedExtractionResult represents the structured output from LLM extraction
type ExtractedExtractionResult struct {
	Title           *TitleField           `json:"title,omitempty"`
	Seniority       *SeniorityField       `json:"seniority,omitempty"`
	Experience      *ExperienceField      `json:"experience,omitempty"`
	Education       *ExtractedEducationField  `json:"education,omitempty"`
	Certifications  []ExtractedCertificationField  `json:"certificates,omitempty"`
	Skills          []SkillField          `json:"skills,omitempty"`
	Responsibilities []ResponsibilityField `json:"responsibilities,omitempty"`
	Leadership      []ExtractedLeadershipField     `json:"leadership,omitempty"`
	Domain          []ExtractedDomainField         `json:"domain,omitempty"`
}

// ExtractedEducationField represents education requirements from extraction
type ExtractedEducationField struct {
	Degree     string  `json:"degree,omitempty"`
	FieldOfStudy string `json:"field_of_study,omitempty"`
	Confidence float64 `json:"confidence"`
}

// ExtractedCertificationField represents a certification requirement from extraction
type ExtractedCertificationField struct {
	Name       string  `json:"name,omitempty"`
	Authority  string  `json:"authority,omitempty"`
	Confidence float64 `json:"confidence"`
}

// ExtractedLeadershipField represents leadership expectations from extraction
type ExtractedLeadershipField struct {
	Leadership   string  `json:"leadership"`
	Importance   float64 `json:"importance"`
	Confidence   float64 `json:"confidence"`
	Evidence     string  `json:"evidence,omitempty"`
}

// ExtractedDomainField represents domain/industry knowledge from extraction
type ExtractedDomainField struct {
	Domain     string  `json:"domain"`
	Importance float64 `json:"importance"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}

// ExtractedJob represents a normalized and validated job posting from extraction
type ExtractedJob struct {
	ID              uuid.UUID `json:"id"`
	RawDataID       uuid.UUID `json:"raw_data_id"`
	Title           string    `json:"title"`
	NormalizedTitle string    `json:"normalized_title"`
	Seniority       string    `json:"seniority"`
	Company         string    `json:"company"`
	Location        string    `json:"location"`
	RemoteType      string    `json:"remote_type"`
	EmploymentType  string    `json:"employment_type"`
	MinExperience   *int      `json:"min_experience,omitempty"`
	MaxExperience   *int      `json:"max_experience,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}