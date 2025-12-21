package enrichment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// Common errors
var (
	ErrNotFound = errors.New("enrichment record not found")
)

// Repository defines the interface for company_enrichment data access
type Repository interface {
	// CRUD operations
	Create(ctx context.Context, companyID uuid.UUID) (*CompanyEnrichment, error)
	GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*CompanyEnrichment, error)

	// Step 1: Basic Info
	SetStep1Processing(ctx context.Context, companyID uuid.UUID) error
	SetStep1Completed(ctx context.Context, companyID uuid.UUID, data *Step1BasicInfo) error
	SetStep1Failed(ctx context.Context, companyID uuid.UUID, errMsg string) error
	ResetStep1(ctx context.Context, companyID uuid.UUID) error

	// Step 2: Business Model
	SetStep2Processing(ctx context.Context, companyID uuid.UUID) error
	SetStep2Completed(ctx context.Context, companyID uuid.UUID, data *Step2BusinessModel) error
	SetStep2Failed(ctx context.Context, companyID uuid.UUID, errMsg string) error
	ResetStep2(ctx context.Context, companyID uuid.UUID) error

	// Step 3: Competitive Intel
	SetStep3Processing(ctx context.Context, companyID uuid.UUID) error
	SetStep3Completed(ctx context.Context, companyID uuid.UUID, data *Step3CompetitiveIntel) error
	SetStep3Failed(ctx context.Context, companyID uuid.UUID, errMsg string) error
	ResetStep3(ctx context.Context, companyID uuid.UUID) error

	// Data retrieval for context building
	GetStep1Data(ctx context.Context, companyID uuid.UUID) (*Step1BasicInfo, error)
	GetStep2Data(ctx context.Context, companyID uuid.UUID) (*Step2BusinessModel, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sqlx.DB) Repository {
	return &PostgresRepository{db: db}
}

// =============================================================================
// CRUD OPERATIONS
// =============================================================================

// Create creates a new enrichment record for a company
func (r *PostgresRepository) Create(ctx context.Context, companyID uuid.UUID) (*CompanyEnrichment, error) {
	enrichment := NewCompanyEnrichment(companyID)

	query := `
		INSERT INTO company_enrichment (
			id, company_id,
			step1_status, step2_status, step3_status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (company_id) DO NOTHING
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, query,
		enrichment.ID, enrichment.CompanyID,
		enrichment.Step1Status, enrichment.Step2Status, enrichment.Step3Status,
		enrichment.CreatedAt, enrichment.UpdatedAt,
	).Scan(&id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Record already exists, fetch it
			return r.GetByCompanyID(ctx, companyID)
		}
		log.Error().Err(err).Str("company_id", companyID.String()).Msg("Failed to create enrichment record")
		return nil, fmt.Errorf("failed to create enrichment record: %w", err)
	}

	enrichment.ID = id
	return enrichment, nil
}

// GetByCompanyID retrieves the enrichment record for a company
func (r *PostgresRepository) GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*CompanyEnrichment, error) {
	query := `
		SELECT
			id, company_id,
			step1_status, step1_completed_at, step1_error, step1_data,
			step2_status, step2_completed_at, step2_error, step2_data,
			step3_status, step3_completed_at, step3_error, step3_data,
			created_at, updated_at
		FROM company_enrichment
		WHERE company_id = $1
	`

	var enrichment CompanyEnrichment
	if err := r.db.GetContext(ctx, &enrichment, query, companyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get enrichment record: %w", err)
	}

	return &enrichment, nil
}

// =============================================================================
// STEP 1: BASIC INFO
// =============================================================================

func (r *PostgresRepository) SetStep1Processing(ctx context.Context, companyID uuid.UUID) error {
	query := `
		UPDATE company_enrichment
		SET step1_status = $2,
			step1_error = NULL,
			updated_at = $3
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusProcessing, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set step1 processing: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetStep1Completed(ctx context.Context, companyID uuid.UUID, data *Step1BasicInfo) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal step1 data: %w", err)
	}

	query := `
		UPDATE company_enrichment
		SET step1_status = $2,
			step1_completed_at = $3,
			step1_error = NULL,
			step1_data = $4,
			updated_at = $5
		WHERE company_id = $1
	`
	now := time.Now()
	_, err = r.db.ExecContext(ctx, query, companyID, StatusCompleted, now, dataJSON, now)
	if err != nil {
		return fmt.Errorf("failed to set step1 completed: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetStep1Failed(ctx context.Context, companyID uuid.UUID, errMsg string) error {
	query := `
		UPDATE company_enrichment
		SET step1_status = $2,
			step1_error = $3,
			updated_at = $4
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusFailed, errMsg, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set step1 failed: %w", err)
	}
	return nil
}

// ResetStep1 resets Step 1 status to pending (for re-enrichment)
func (r *PostgresRepository) ResetStep1(ctx context.Context, companyID uuid.UUID) error {
	query := `
		UPDATE company_enrichment
		SET step1_status = $2,
			step1_completed_at = NULL,
			step1_error = NULL,
			step1_data = NULL,
			updated_at = $3
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusPending, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reset step1: %w", err)
	}
	return nil
}

// =============================================================================
// STEP 2: BUSINESS MODEL
// =============================================================================

func (r *PostgresRepository) SetStep2Processing(ctx context.Context, companyID uuid.UUID) error {
	query := `
		UPDATE company_enrichment
		SET step2_status = $2,
			step2_error = NULL,
			updated_at = $3
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusProcessing, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set step2 processing: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetStep2Completed(ctx context.Context, companyID uuid.UUID, data *Step2BusinessModel) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal step2 data: %w", err)
	}

	query := `
		UPDATE company_enrichment
		SET step2_status = $2,
			step2_completed_at = $3,
			step2_error = NULL,
			step2_data = $4,
			updated_at = $5
		WHERE company_id = $1
	`
	now := time.Now()
	_, err = r.db.ExecContext(ctx, query, companyID, StatusCompleted, now, dataJSON, now)
	if err != nil {
		return fmt.Errorf("failed to set step2 completed: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetStep2Failed(ctx context.Context, companyID uuid.UUID, errMsg string) error {
	query := `
		UPDATE company_enrichment
		SET step2_status = $2,
			step2_error = $3,
			updated_at = $4
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusFailed, errMsg, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set step2 failed: %w", err)
	}
	return nil
}

// ResetStep2 resets Step 2 status to pending (for re-enrichment)
func (r *PostgresRepository) ResetStep2(ctx context.Context, companyID uuid.UUID) error {
	query := `
		UPDATE company_enrichment
		SET step2_status = $2,
			step2_completed_at = NULL,
			step2_error = NULL,
			step2_data = NULL,
			updated_at = $3
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusPending, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reset step2: %w", err)
	}
	return nil
}

// =============================================================================
// STEP 3: COMPETITIVE INTEL
// =============================================================================

func (r *PostgresRepository) SetStep3Processing(ctx context.Context, companyID uuid.UUID) error {
	query := `
		UPDATE company_enrichment
		SET step3_status = $2,
			step3_error = NULL,
			updated_at = $3
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusProcessing, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set step3 processing: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetStep3Completed(ctx context.Context, companyID uuid.UUID, data *Step3CompetitiveIntel) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal step3 data: %w", err)
	}

	query := `
		UPDATE company_enrichment
		SET step3_status = $2,
			step3_completed_at = $3,
			step3_error = NULL,
			step3_data = $4,
			updated_at = $5
		WHERE company_id = $1
	`
	now := time.Now()
	_, err = r.db.ExecContext(ctx, query, companyID, StatusCompleted, now, dataJSON, now)
	if err != nil {
		return fmt.Errorf("failed to set step3 completed: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetStep3Failed(ctx context.Context, companyID uuid.UUID, errMsg string) error {
	query := `
		UPDATE company_enrichment
		SET step3_status = $2,
			step3_error = $3,
			updated_at = $4
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusFailed, errMsg, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set step3 failed: %w", err)
	}
	return nil
}

// ResetStep3 resets Step 3 status to pending (for re-enrichment)
func (r *PostgresRepository) ResetStep3(ctx context.Context, companyID uuid.UUID) error {
	query := `
		UPDATE company_enrichment
		SET step3_status = $2,
			step3_completed_at = NULL,
			step3_error = NULL,
			step3_data = NULL,
			updated_at = $3
		WHERE company_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, companyID, StatusPending, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reset step3: %w", err)
	}
	return nil
}

// =============================================================================
// DATA RETRIEVAL FOR CONTEXT BUILDING
// =============================================================================

// GetStep1Data retrieves Step 1 data for building context in subsequent steps
func (r *PostgresRepository) GetStep1Data(ctx context.Context, companyID uuid.UUID) (*Step1BasicInfo, error) {
	query := `SELECT step1_data FROM company_enrichment WHERE company_id = $1`

	var dataJSON []byte
	if err := r.db.GetContext(ctx, &dataJSON, query, companyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get step1 data: %w", err)
	}

	if dataJSON == nil {
		return nil, nil
	}

	var data Step1BasicInfo
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step1 data: %w", err)
	}

	return &data, nil
}

// GetStep2Data retrieves Step 2 data for building context in Step 3
func (r *PostgresRepository) GetStep2Data(ctx context.Context, companyID uuid.UUID) (*Step2BusinessModel, error) {
	query := `SELECT step2_data FROM company_enrichment WHERE company_id = $1`

	var dataJSON []byte
	if err := r.db.GetContext(ctx, &dataJSON, query, companyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get step2 data: %w", err)
	}

	if dataJSON == nil {
		return nil, nil
	}

	var data Step2BusinessModel
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step2 data: %w", err)
	}

	return &data, nil
}
