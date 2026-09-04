package processing

import (
	"passo/backend/internal/plugins/market-data-engine/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pipeline coordinates the extraction and normalization stages
type Pipeline struct {
	extractor *Extractor
	normalizer *Normalizer
	db        *pgxpool.Pool
}

// NewPipeline creates a new processing pipeline
func NewPipeline(db *pgxpool.Pool) *Pipeline {
	return &Pipeline{
		extractor: NewExtractor(),
		normalizer: NewNormalizer(db),
		db:        db,
	}
}

// Process takes raw content and returns fully normalized job data
func (p *Pipeline) Process(content string) (models.NormalizedJobData, error) {
	// Stage 1: Extraction (Phase 2)
	extractedJob, err := p.extractor.ExtractJobData(content)
	if err != nil {
		return models.NormalizedJobData{}, err
	}

	// Stage 2: Normalization & Taxonomy (Phase 3)
	normalizedJob := p.normalizer.NormalizeJob(*extractedJob)

	return normalizedJob, nil
}

// ProcessSkillsOnly takes extracted skills and returns normalized skills
// Useful for testing or partial processing
func (p *Pipeline) ProcessSkillsOnly(extractedSkills []models.SkillField) []models.NormalizedSkill {
	return p.normalizer.NormalizeSkills(extractedSkills)
}