package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// DomainRepository handles storage and retrieval of domains
type DomainRepository struct {
	db *pgxpool.Pool
}

// NewDomainRepository creates a new DomainRepository
func NewDomainRepository(db *pgxpool.Pool) *DomainRepository {
	return &DomainRepository{db: db}
}

// Insert adds a new domain and returns its ID
func (r *DomainRepository) Insert(ctx context.Context, domain *models.Domain) (uuid.UUID, error) {
	query := `
		INSERT INTO domains
			(id, name, parent_id, category, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		domain.ID,
		domain.Name,
		domain.ParentID,
		domain.Category,
		domain.TaxonomyVersion,
		domain.CreatedAt,
		domain.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a domain by its ID
func (r *DomainRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Domain, error) {
	query := `
		SELECT id, name, parent_id, category, taxonomy_version, created_at, updated_at
		FROM domains
		WHERE id = $1
	`

	var domain models.Domain
	err := r.db.QueryRow(ctx, query, id).Scan(
		&domain.ID,
		&domain.Name,
		&domain.ParentID,
		&domain.Category,
		&domain.TaxonomyVersion,
		&domain.CreatedAt,
		&domain.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &domain, nil
}

// GetByName retrieves a domain by its name (returns first match if multiple exist)
func (r *DomainRepository) GetByName(ctx context.Context, name string) (*models.Domain, error) {
	query := `
		SELECT id, name, parent_id, category, taxonomy_version, created_at, updated_at
		FROM domains
		WHERE name = $1
		LIMIT 1
	`

	var domain models.Domain
	err := r.db.QueryRow(ctx, query, name).Scan(
		&domain.ID,
		&domain.Name,
		&domain.ParentID,
		&domain.Category,
		&domain.TaxonomyVersion,
		&domain.CreatedAt,
		&domain.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &domain, nil
}

// List returns all domains
func (r *DomainRepository) List(ctx context.Context) ([]*models.Domain, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, parent_id, category, taxonomy_version, created_at, updated_at
		FROM domains
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*models.Domain
	for rows.Next() {
		var domain models.Domain
		if err := rows.Scan(
			&domain.ID,
			&domain.Name,
			&domain.ParentID,
			&domain.Category,
			&domain.TaxonomyVersion,
			&domain.CreatedAt,
			&domain.UpdatedAt,
		); err != nil {
			return nil, err
		}
		domains = append(domains, &domain)
	}

	return domains, rows.Err()
}

// Update updates an existing domain
func (r *DomainRepository) Update(ctx context.Context, domain *models.Domain) error {
	query := `
		UPDATE domains
		SET name = $2,
		    parent_id = $3,
		    category = $4,
		    taxonomy_version = $5,
		    updated_at = $6
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		domain.ID,
		domain.Name,
		domain.ParentID,
		domain.Category,
		domain.TaxonomyVersion,
		domain.UpdatedAt,
	)

	return err
}

// Delete removes a domain by its ID
func (r *DomainRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM domains
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}