package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// CompanyRepository handles storage and retrieval of companies
type CompanyRepository struct {
	db *pgxpool.Pool
}

// NewCompanyRepository creates a new CompanyRepository
func NewCompanyRepository(db *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{db: db}
}

// Insert adds a new company and returns its ID
func (r *CompanyRepository) Insert(ctx context.Context, company *models.Company) (uuid.UUID, error) {
	query := `
		INSERT INTO companies
			(id, canonical_name, normalized_name, website, industry, size,
			 taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		company.ID,
		company.CanonicalName,
		company.NormalizedName,
		company.Website,
		company.Industry,
		company.Size,
		company.TaxonomyVersion,
		company.CreatedAt,
		company.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a company by its ID
func (r *CompanyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Company, error) {
	query := `
		SELECT id, canonical_name, normalized_name, website, industry, size,
		       taxonomy_version, created_at, updated_at
		FROM companies
		WHERE id = $1
	`

	var company models.Company
	err := r.db.QueryRow(ctx, query, id).Scan(
		&company.ID,
		&company.CanonicalName,
		&company.NormalizedName,
		&company.Website,
		&company.Industry,
		&company.Size,
		&company.TaxonomyVersion,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &company, nil
}

// GetByCanonicalName retrieves a company by its canonical name
func (r *CompanyRepository) GetByCanonicalName(ctx context.Context, canonicalName string) (*models.Company, error) {
	query := `
		SELECT id, canonical_name, normalized_name, website, industry, size,
		       taxonomy_version, created_at, updated_at
		FROM companies
		WHERE canonical_name = $1
	`

	var company models.Company
	err := r.db.QueryRow(ctx, query, canonicalName).Scan(
		&company.ID,
		&company.CanonicalName,
		&company.NormalizedName,
		&company.Website,
		&company.Industry,
		&company.Size,
		&company.TaxonomyVersion,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &company, nil
}

// List returns all companies
func (r *CompanyRepository) List(ctx context.Context) ([]*models.Company, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, canonical_name, normalized_name, website, industry, size,
		       taxonomy_version, created_at, updated_at
		FROM companies
		ORDER BY canonical_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []*models.Company
	for rows.Next() {
		var company models.Company
		if err := rows.Scan(
			&company.ID,
			&company.CanonicalName,
			&company.NormalizedName,
			&company.Website,
			&company.Industry,
			&company.Size,
			&company.TaxonomyVersion,
			&company.CreatedAt,
			&company.UpdatedAt,
		); err != nil {
			return nil, err
		}
		companies = append(companies, &company)
	}

	return companies, rows.Err()
}

// Update updates an existing company
func (r *CompanyRepository) Update(ctx context.Context, company *models.Company) error {
	query := `
		UPDATE companies
		SET canonical_name = $2,
		    normalized_name = $3,
		    website = $4,
		    industry = $5,
		    size = $6,
		    taxonomy_version = $7,
		    updated_at = $8
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		company.ID,
		company.CanonicalName,
		company.NormalizedName,
		company.Website,
		company.Industry,
		company.Size,
		company.TaxonomyVersion,
		company.UpdatedAt,
	)

	return err
}

// Delete removes a company by its ID
func (r *CompanyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM companies
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}