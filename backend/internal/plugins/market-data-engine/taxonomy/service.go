package taxonomy

import (
	"passo/backend/internal/plugins/market-data-engine/models"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// TaxonomyService handles normalization and standardization of extracted data
type TaxonomyService struct {
	skillAliases     map[string]string // alias -> canonical form
	capabilityTaxonomy map[string]*models.Capability
	aliasManager     *AliasManager
}

// NewTaxonomyService creates a new TaxonomyService with predefined aliases and taxonomy
func NewTaxonomyService() *TaxonomyService {
	service := &TaxonomyService{
		skillAliases:     make(map[string]string),
		capabilityTaxonomy: make(map[string]*models.Capability),
		aliasManager:     NewAliasManager(),
	}

	// Initialize skill aliases
	service.initializeSkillAliases()

	// Initialize capability taxonomy (basic version)
	service.initializeCapabilityTaxonomy()

	// Load default aliases into alias manager
	service.aliasManager.LoadDefaultAliases()

	return service
}

// initializeSkillAliases sets up common skill aliases and variations
func (s *TaxonomyService) initializeSkillAliases() {
	// Programming Languages
	s.skillAliases["JS"] = "JavaScript"
	s.skillAliases["EcmaScript"] = "JavaScript"
	s.skillAliases["ES6"] = "JavaScript"
	s.skillAliases["ES2015"] = "JavaScript"
	s.skillAliases["ES2016"] = "JavaScript"
	s.skillAliases["ES2017"] = "JavaScript"
	s.skillAliases["ES2018"] = "JavaScript"
	s.skillAliases["ES2019"] = "JavaScript"
	s.skillAliases["ES2020"] = "JavaScript"
	s.skillAliases["ES2021"] = "JavaScript"
	s.skillAliases["ES2022"] = "JavaScript"
	s.skillAliases["ES2023"] = "JavaScript"

	s.skillAliases["TS"] = "TypeScript"
	s.skillAliases["Typescript"] = "TypeScript"

	s.skillAliases["Py"] = "Python"
	s.skillAliases["Python3"] = "Python"

	s.skillAliases["Rb"] = "Ruby"
	s.skillAliases["RoR"] = "Ruby on Rails"
	s.skillAliases["Rails"] = "Ruby on Rails"

	s.skillAliases["Cpp"] = "C++"
	s.skillAliases["CPPC"] = "C++"
	s.skillAliases["CSharp"] = "C#"
	s.skillAliases["CSharp"] = "C#" // Duplicate intentional for robustness
	s.skillAliases["DotNet"] = ".NET"
	s.skillAliases["NET"] = ".NET"

	// Web Technologies
	s.skillAliases["HTML"] = "HTML5"
	s.skillAliases["HTML 5"] = "HTML5"
	s.skillAliases["CSS"] = "CSS3"
	s.skillAliases["CSS 3"] = "CSS3"

	s.skillAliases["ReactJS"] = "React"
	s.skillAliases["React.js"] = "React"
	s.skillAliases["React JS"] = "React"

	s.skillAliases["VueJS"] = "Vue.js"
	s.skillAliases["Vue.js"] = "Vue.js"
	s.skillAliases["Vue JS"] = "Vue.js"

	s.skillAliases["AngularJS"] = "Angular"
	s.skillAliases["Angular.js"] = "Angular"
	s.skillAliases["Angular 2+"] = "Angular"
	s.skillAliases["AngularJS"] = "Angular" // Duplicate intentional

	s.skillAliases["NodeJS"] = "Node.js"
	s.skillAliases["Node.js"] = "Node.js"
	s.skillAliases["Node"] = "Node.js"

	// Databases
	s.skillAliases["Postgres"] = "PostgreSQL"
	s.skillAliases["Postgre"] = "PostgreSQL"
	s.skillAliases["MySQL"] = "MySQL"
	s.skillAliases["MongoDB"] = "MongoDB"
	s.skillAliases["Mongo"] = "MongoDB"
	s.skillAliases["DynamoDB"] = "Amazon DynamoDB"
	s.skillAliases["Dynamo"] = "Amazon DynamoDB"

	// Cloud Platforms
	s.skillAliases["AWS"] = "Amazon Web Services"
	s.skillAliases["Amazon Web Services"] = "Amazon Web Services"
	s.skillAliases["Amazon Web Service"] = "Amazon Web Services" // Variant
	s.skillAliases["GCP"] = "Google Cloud Platform"
	s.skillAliases["Google Cloud"] = "Google Cloud Platform"
	s.skillAliases["Google Cloud"] = "Google Cloud Platform" // Duplicate
	s.skillAliases["Azure"] = "Microsoft Azure"
	s.skillAliases["MS Azure"] = "Microsoft Azure"

	// DevOps & Tools
	s.skillAliases["K8s"] = "Kubernetes"
	s.skillAliases["Kube"] = "Kubernetes"
	s.skillAliases["Kubernets"] = "Kubernetes" // Common misspelling
	s.skillAliases["Docker"] = "Docker"
	s.skillAliases["CI/CD"] = "Continuous Integration/Continuous Deployment"
	s.skillAliases["CICD"] = "Continuous Integration/Continuous Deployment"
	s.skillAliases["CiCd"] = "Continuous Integration/Continuous Deployment" // Variant

	// Testing
	s.skillAliases["JUnit"] = "JUnit Testing"
	s.skillAliases["NUnit"] = "NUnit Testing"
	s.skillAliases["TestNG"] = "TestNG Testing"

	// Methodologies
	s.skillAliases["TDD"] = "Test-Driven Development"
	s.skillAliases["BDD"] = "Behavior-Driven Development"
	s.skillAliases["Agile"] = "Agile Methodology"
	s.skillAliases["Scrum"] = "Scrum Framework"
	s.skillAliases["Kanban"] = "Kanban Method"
}

// initializeCapabilityTaxonomy sets up a basic capability taxonomy
func (s *TaxonomyService) initializeCapabilityTaxonomy() {
	// This would normally be loaded from a database or configuration file
	// For now, we'll create a basic in-memory taxonomy

	// Programming Language Capabilities
	jsID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["JavaScript"] = &models.Capability{
		ID:          jsID,
		CanonicalName: "JavaScript",
		Category:    "programming_language",
		Description: "High-level, just-in-time compiled language that conforms to the ECMAScript specification",
	}

	tsID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["TypeScript"] = &models.Capability{
		ID:          tsID,
		CanonicalName: "TypeScript",
		Category:    "programming_language",
		Description: "Strict syntactical superset of JavaScript that adds optional static typing",
	}

	pyID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["Python"] = &models.Capability{
		ID:          pyID,
		CanonicalName: "Python",
		Category:    "programming_language",
		Description: "High-level, interpreted programming language known for its readability",
	}

	// Web Development Capabilities
	html5ID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["HTML5"] = &models.Capability{
		ID:          html5ID,
		CanonicalName: "HTML5",
		Category:    "markup_language",
		Description: "Latest version of HyperText Markup Language, the standard markup language for documents designed to be displayed in a web browser",
	}

	css3ID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["CSS3"] = &models.Capability{
		ID:          css3ID,
		CanonicalName: "CSS3",
		Category:    "style_sheet_language",
		Description: "Latest version of Cascading Style Sheets, used for describing the presentation of a document written in HTML or XML",
	}

	reactID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["React"] = &models.Capability{
		ID:          reactID,
		CanonicalName: "React",
		Category:    "frontend_framework",
		Description: "JavaScript library for building user interfaces, maintained by Facebook",
	}

	// Cloud Platform Capabilities
	awsID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["Amazon Web Services"] = &models.Capability{
		ID:          awsID,
		CanonicalName: "Amazon Web Services",
		Category:    "cloud_platform",
		Description: "Comprehensive cloud computing platform provided by Amazon",
	}

	// DevOps Capabilities
	k8sID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["Kubernetes"] = &models.Capability{
		ID:          k8sID,
		CanonicalName: "Kubernetes",
		Category:    "orchestration_platform",
		Description: "Open-source system for automating deployment, scaling, and management of containerized applications",
	}

	dockerID, _ := uuid.NewRandom()
	s.capabilityTaxonomy["Docker"] = &models.Capability{
		ID:          dockerID,
		CanonicalName: "Docker",
		Category:    "containerization_platform",
		Description: "Platform for developing, shipping, and running applications in containers",
	}

	// Build reverse lookup maps for faster alias resolution
	s.buildReverseAliasMaps()
}

// buildReverseAliasMaps builds reverse lookup maps for faster alias resolution
func (s *TaxonomyService) buildReverseAliasMaps() {
	// Clear existing aliases
	s.skillAliases = make(map[string]string)

	// Populate from taxonomy
	for _, capability := range s.capabilityTaxonomy {
		// Add the canonical name as an alias of itself
		s.skillAliases[capability.CanonicalName] = capability.CanonicalName

		// Note: Since the Capability model doesn't have an Aliases field,
		// we rely on the skillAliases map for alias resolution
		// In a real implementation, we would store aliases with each capability
	}
}

// NormalizeSkill takes a skill name and returns its canonical form
func (s *TaxonomyService) NormalizeSkill(skill string) string {
	if skill == "" {
		return ""
	}

	// Trim whitespace
	skill = strings.TrimSpace(skill)

	// Check if it's a known alias
	if canonical, exists := s.skillAliases[skill]; exists {
		return canonical
	}

	// Check if it's already a canonical form in our taxonomy
	if _, exists := s.capabilityTaxonomy[skill]; exists {
		return skill
	}

	// Not found, return as-is (could be a new skill not in our taxonomy)
	return skill
}

// NormalizeSkills takes a slice of skill names and returns normalized, deduplicated slice
func (s *TaxonomyService) NormalizeSkills(skills []string) []string {
	normalizedSet := make(map[string]bool)
	var normalized []string

	for _, skill := range skills {
		normalizedSkill := s.NormalizeSkill(skill)
		if normalizedSkill != "" && !normalizedSet[normalizedSkill] {
			normalizedSet[normalizedSkill] = true
			normalized = append(normalized, normalizedSkill)
		}
	}

	// Sort for consistent output
	sort.Strings(normalized)
	return normalized
}

// GetCapability returns the capability for a given skill name (after normalization)
func (s *TaxonomyService) GetCapability(skill string) *models.Capability {
	normalized := s.NormalizeSkill(skill)
	if capability, exists := s.capabilityTaxonomy[normalized]; exists {
		return capability
	}
	return nil
}

// GetRelatedCapabilities returns capabilities related to the given skill
func (s *TaxonomyService) GetRelatedCapabilities(skill string) []*models.Capability {
	var related []*models.Capability

	capability := s.GetCapability(skill)
	if capability == nil {
		return related
	}

	// Note: Since the Capability model doesn't have ParentID or ChildrenIDs in the expected format,
	// and we don't have a way to traverse the taxonomy hierarchy easily in this basic implementation,
	// we'll return an empty array for related capabilities.
	// In a real implementation with a proper taxonomy structure, we would traverse parent/child relationships.

	return related
}

// IsValidSkill checks if a skill exists in our taxonomy (after normalization)
func (s *TaxonomyService) IsValidSkill(skill string) bool {
	normalized := s.NormalizeSkill(skill)
	_, exists := s.capabilityTaxonomy[normalized]
	return exists
}

// GetAllCapabilities returns all capabilities in the taxonomy
func (s *TaxonomyService) GetAllCapabilities() []*models.Capability {
	capabilities := make([]*models.Capability, 0, len(s.capabilityTaxonomy))
	for _, capability := range s.capabilityTaxonomy {
		capabilities = append(capabilities, capability)
	}

	// Sort by name for consistent output
	sort.Slice(capabilities, func(i, j int) bool {
		return capabilities[i].CanonicalName < capabilities[j].CanonicalName
	})

	return capabilities
}

// GetAliasManager returns the alias manager for external alias management
func (s *TaxonomyService) GetAliasManager() *AliasManager {
	return s.aliasManager
}