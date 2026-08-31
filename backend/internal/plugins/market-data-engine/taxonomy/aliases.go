package taxonomy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

// AliasManager handles loading and managing skill aliases from various sources
type AliasManager struct {
	aliases map[string]string // alias -> canonical form
	// Reverse map for finding all aliases of a canonical form
	canonicalToAliases map[string][]string
}

// NewAliasManager creates a new AliasManager
func NewAliasManager() *AliasManager {
	return &AliasManager{
		aliases:             make(map[string]string),
		canonicalToAliases:  make(map[string][]string),
	}
}

// LoadFromFile loads aliases from a JSON file
// Expected format: {"canonical_form": ["alias1", "alias2", ...]}
func (am *AliasManager) LoadFromFile(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read alias file %s: %w", filePath, err)
	}

	var aliasMap map[string][]string
	if err := json.Unmarshal(data, &aliasMap); err != nil {
		return fmt.Errorf("failed to parse alias file %s: %w", filePath, err)
	}

	// Clear existing aliases
	am.aliases = make(map[string]string)
	am.canonicalToAliases = make(map[string][]string)

	// Load new aliases
	for canonical, aliases := range aliasMap {
		if strings.TrimSpace(canonical) == "" {
			continue
		}

		// Add the canonical form as an alias of itself (for reverse lookup)
		am.canonicalToAliases[canonical] = append(am.canonicalToAliases[canonical], canonical)

		for _, alias := range aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}

			// Store alias -> canonical mapping
			am.aliases[alias] = canonical

			// Store canonical -> aliases mapping (for reverse lookup)
			am.canonicalToAliases[canonical] = append(am.canonicalToAliases[canonical], alias)
		}

		// Remove duplicates and sort the alias list for each canonical form
		am.canonicalToAliases[canonical] = am.uniqueAndSort(am.canonicalToAliases[canonical])
	}

	return nil
}

// LoadFromDirectory loads all JSON alias files from a directory
func (am *AliasManager) LoadFromDirectory(dirPath string) error {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(dirPath, file.Name())
			if err := am.LoadFromFile(filePath); err != nil {
				return fmt.Errorf("failed to load alias file %s: %w", filePath, err)
			}
		}
	}

	return nil
}

// GetCanonical returns the canonical form for a given alias
func (am *AliasManager) GetCanonical(alias string) (string, bool) {
	if alias == "" {
		return "", false
	}
	canonical, exists := am.aliases[alias]
	return canonical, exists
}

// GetAliases returns all known aliases for a given canonical form
func (am *AliasManager) GetAliases(canonical string) []string {
	if canonical == "" {
		return []string{}
	}
	aliases := am.canonicalToAliases[canonical]
	// Return a copy to prevent external modification
	result := make([]string, len(aliases))
	copy(result, aliases)
	return result
}

// AddAlias adds a new alias-canonical mapping
func (am *AliasManager) AddAlias(alias, canonical string) {
	if alias == "" || canonical == "" {
		return
	}

	// Store alias -> canonical mapping
	am.aliases[alias] = canonical

	// Store canonical -> aliases mapping
	am.canonicalToAliases[canonical] = append(am.canonicalToAliases[canonical], alias)

	// Remove duplicates and sort
	am.canonicalToAliases[canonical] = am.uniqueAndSort(am.canonicalToAliases[canonical])
}

// RemoveAlias removes an alias mapping
func (am *AliasManager) RemoveAlias(alias string) {
	if alias == "" {
		return
	}

	if canonical, exists := am.aliases[alias]; exists {
		delete(am.aliases, alias)

		// Remove from canonical's alias list
		if aliases := am.canonicalToAliases[canonical]; len(aliases) > 0 {
			filtered := make([]string, 0, len(aliases))
			for _, a := range aliases {
				if a != alias {
					filtered = append(filtered, a)
				}
			}
			am.canonicalToAliases[canonical] = am.uniqueAndSort(filtered)
		}
	}
}

// IsAlias checks if a string is a known alias
func (am *AliasManager) IsAlias(alias string) bool {
	if alias == "" {
		return false
	}
	_, exists := am.aliases[alias]
	return exists
}

// IsCanonical checks if a string is a known canonical form
func (am *AliasManager) IsCanonical(canonical string) bool {
	if canonical == "" {
		return false
	}
	_, exists := am.canonicalToAliases[canonical]
	return exists
}

// GetAllAliases returns all known aliases
func (am *AliasManager) GetAllAliases() []string {
	aliases := make([]string, 0, len(am.aliases))
	for alias := range am.aliases {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

// GetAllCanonicals returns all known canonical forms
func (am *AliasManager) GetAllCanonicals() []string {
	canonicals := make([]string, 0, len(am.canonicalToAliases))
	for canonical := range am.canonicalToAliases {
		canonicals = append(canonicals, canonical)
	}
	sort.Strings(canonicals)
	return canonicals
}

// uniqueAndSort removes duplicates from a slice of strings and sorts it
func (am *AliasManager) uniqueAndSort(strings []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strings {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}

// LoadDefaultAliases loads built-in default aliases
func (am *AliasManager) LoadDefaultAliases() {
	defaultAliases := map[string][]string{
		"JavaScript": {"JS", "EcmaScript", "ES6", "ES2015", "ES2016", "ES2017", "ES2018", "ES2019", "ES2020", "ES2021", "ES2022", "ES2023"},
		"TypeScript": {"TS", "Typescript"},
		"Python":     {"Py", "Python3"},
		"Ruby":       {"Rb"},
		"Ruby on Rails": {"RoR", "Rails"},
		"C++":        {"Cpp", "CPPC"},
		"C#":         {"CSharp"},
		".NET":       {"DotNet", "NET"},
		"HTML5":      {"HTML", "HTML 5"},
		"CSS3":       {"CSS", "CSS 3"},
		"React":      {"ReactJS", "React.js", "React JS"},
		"Vue.js":     {"VueJS", "Vue JS"},
		"Angular":    {"AngularJS", "Angular.js", "Angular 2+"},
		"Node.js":    {"nodeJS", "Node"},
		"PostgreSQL": {"Postgres", "Postgre"},
		"MongoDB":    {"Mongo"},
		"Amazon DynamoDB": {"DynamoDB", "Dynamo"},
		"Amazon Web Services": {"AWS", "Amazon Web Service"},
		"Google Cloud Platform": {"GCP", "Google Cloud"},
		"Microsoft Azure": {"Azure", "MS Azure"},
		"Kubernetes":  {"K8s", "Kube", "Kubernets"},
		"Docker":      {"Docker"},
		"Continuous Integration/Continuous Deployment": {"CI/CD", "CICD", "CiCd"},
		"Test-Driven Development": {"TDD"},
		"Behavior-Driven Development": {"BDD"},
		"Agile Methodology": {"Agile"},
		"Scrum Framework": {"Scrum"},
		"Kanban Method": {"Kanban"},
		"JUnit Testing": {"JUnit"},
		"NUnit Testing": {"NUnit"},
		"TestNG Testing": {"TestNG"},
	}

	for canonical, aliases := range defaultAliases {
		// Add the canonical form itself
		am.AddAlias(canonical, canonical)
		// Add all the aliases
		for _, alias := range aliases {
			am.AddAlias(alias, canonical)
		}
	}
}