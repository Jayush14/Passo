package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// SkillRelationshipRepository handles storage and retrieval of skill relationships
type SkillRelationshipRepository struct {
	db *pgxpool.Pool
}

// NewSkillRelationshipRepository creates a new SkillRelationshipRepository
func NewSkillRelationshipRepository(db *pgxpool.Pool) *SkillRelationshipRepository {
	return &SkillRelationshipRepository{db: db}
}

// Insert adds a new skill relationship and returns its ID
func (r *SkillRelationshipRepository) Insert(ctx context.Context, relationship *models.SkillRelationship) (uuid.UUID, error) {
	query := `
		INSERT INTO skill_relationships
			(id, source_skill_id, target_skill_id, relationship_type, strength, taxonomy_version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		relationship.ID,
		relationship.SourceSkillID,
		relationship.TargetSkillID,
		relationship.RelationshipType,
		relationship.Strength,
		relationship.TaxonomyVersion,
		relationship.CreatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a skill relationship by its ID
func (r *SkillRelationshipRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SkillRelationship, error) {
	query := `
		SELECT id, source_skill_id, target_skill_id, relationship_type, strength, taxonomy_version, created_at
		FROM skill_relationships
		WHERE id = $1
	`

	var relationship models.SkillRelationship
	err := r.db.QueryRow(ctx, query, id).Scan(
		&relationship.ID,
		&relationship.SourceSkillID,
		&relationship.TargetSkillID,
		&relationship.RelationshipType,
		&relationship.Strength,
		&relationship.TaxonomyVersion,
		&relationship.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &relationship, nil
}

// ListBySourceSkillID returns all relationships where the given skill is the source
func (r *SkillRelationshipRepository) ListBySourceSkillID(ctx context.Context, sourceSkillID uuid.UUID) ([]*models.SkillRelationship, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, source_skill_id, target_skill_id, relationship_type, strength, taxonomy_version, created_at
		FROM skill_relationships
		WHERE source_skill_id = $1
		ORDER BY relationship_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []*models.SkillRelationship
	for rows.Next() {
		var relationship models.SkillRelationship
		if err := rows.Scan(
			&relationship.ID,
			&relationship.SourceSkillID,
			&relationship.TargetSkillID,
			&relationship.RelationshipType,
			&relationship.Strength,
			&relationship.TaxonomyVersion,
			&relationship.CreatedAt,
		); err != nil {
			return nil, err
		}
		relationships = append(relationships, &relationship)
	}

	return relationships, rows.Err()
}

// ListByTargetSkillID returns all relationships where the given skill is the target
func (r *SkillRelationshipRepository) ListByTargetSkillID(ctx context.Context, targetSkillID uuid.UUID) ([]*models.SkillRelationship, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, source_skill_id, target_skill_id, relationship_type, strength, taxonomy_version, created_at
		FROM skill_relationships
		WHERE target_skill_id = $1
		ORDER BY relationship_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []*models.SkillRelationship
	for rows.Next() {
		var relationship models.SkillRelationship
		if err := rows.Scan(
			&relationship.ID,
			&relationship.SourceSkillID,
			&relationship.TargetSkillID,
			&relationship.RelationshipType,
			&relationship.Strength,
			&relationship.TaxonomyVersion,
			&relationship.CreatedAt,
		); err != nil {
			return nil, err
		}
		relationships = append(relationships, &relationship)
	}

	return relationships, rows.Err()
}

// Update updates an existing skill relationship
func (r *SkillRelationshipRepository) Update(ctx context.Context, relationship *models.SkillRelationship) error {
	query := `
		UPDATE skill_relationships
		SET source_skill_id = $2,
		    target_skill_id = $3,
		    relationship_type = $4,
		    strength = $5,
		    taxonomy_version = $6
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		relationship.ID,
		relationship.SourceSkillID,
		relationship.TargetSkillID,
		relationship.RelationshipType,
		relationship.Strength,
		relationship.TaxonomyVersion,
	)

	return err
}

// Delete removes a skill relationship by its ID
func (r *SkillRelationshipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM skill_relationships
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}