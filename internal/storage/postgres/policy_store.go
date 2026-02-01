package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PolicyStore handles policy persistence
type PolicyStore struct {
	client *Client
}

// NewPolicyStore creates a new policy store
func NewPolicyStore(client *Client) *PolicyStore {
	return &PolicyStore{client: client}
}

// PolicyRecord represents a stored policy
type PolicyRecord struct {
	ID            string          `db:"id"`
	PolicyID      string          `db:"policy_id"`
	Version       string          `db:"version"`
	Name          string          `db:"name"`
	Description   string          `db:"description"`
	DSL           string          `db:"dsl_yaml"`
	EffectiveFrom interface{}     `db:"effective_from"`
	EffectiveUntil interface{}    `db:"effective_until"`
	IsActive      bool            `db:"is_active"`
	CreatedBy     string          `db:"created_by"`
	CreatedAt     interface{}     `db:"created_at"`
}

// Store persists a policy
func (s *PolicyStore) Store(ctx context.Context, record *PolicyRecord) error {
	query := `
		INSERT INTO policies (policy_id, version, name, description, dsl_yaml, effective_from, effective_until, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (policy_id, version) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			dsl_yaml = EXCLUDED.dsl_yaml,
			effective_until = EXCLUDED.effective_until,
			is_active = EXCLUDED.is_active
		RETURNING id, created_at`

	return s.client.Pool().QueryRow(ctx, query,
		record.PolicyID,
		record.Version,
		record.Name,
		record.Description,
		record.DSL,
		record.EffectiveFrom,
		record.EffectiveUntil,
		record.IsActive,
		record.CreatedBy,
	).Scan(&record.ID, &record.CreatedAt)
}

// GetActivePolicy retrieves the active policy by ID
func (s *PolicyStore) GetActivePolicy(ctx context.Context, policyID string) (*PolicyRecord, error) {
	query := `
		SELECT id, policy_id, version, name, description, dsl_yaml, effective_from, effective_until, is_active, created_by, created_at
		FROM policies
		WHERE policy_id = $1 
			AND is_active = true
			AND effective_from <= NOW()
			AND (effective_until IS NULL OR effective_until > NOW())
		ORDER BY effective_from DESC
		LIMIT 1`

	var record PolicyRecord
	err := s.client.Pool().QueryRow(ctx, query, policyID).Scan(
		&record.ID, &record.PolicyID, &record.Version, &record.Name,
		&record.Description, &record.DSL, &record.EffectiveFrom,
		&record.EffectiveUntil, &record.IsActive, &record.CreatedBy, &record.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("policy not found: %s", policyID)
		}
		return nil, err
	}

	return &record, nil
}

// GetPolicyVersion retrieves a specific policy version
func (s *PolicyStore) GetPolicyVersion(ctx context.Context, policyID, version string) (*PolicyRecord, error) {
	query := `
		SELECT id, policy_id, version, name, description, dsl_yaml, effective_from, effective_until, is_active, created_by, created_at
		FROM policies
		WHERE policy_id = $1 AND version = $2`

	var record PolicyRecord
	err := s.client.Pool().QueryRow(ctx, query, policyID, version).Scan(
		&record.ID, &record.PolicyID, &record.Version, &record.Name,
		&record.Description, &record.DSL, &record.EffectiveFrom,
		&record.EffectiveUntil, &record.IsActive, &record.CreatedBy, &record.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("policy version not found: %s@%s", policyID, version)
		}
		return nil, err
	}

	return &record, nil
}

// ListPolicies lists all policies
func (s *PolicyStore) ListPolicies(ctx context.Context, activeOnly bool) ([]*PolicyRecord, error) {
	query := `
		SELECT id, policy_id, version, name, description, dsl_yaml, effective_from, effective_until, is_active, created_by, created_at
		FROM policies`
	
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	
	query += ` ORDER BY policy_id, version`

	rows, err := s.client.Pool().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPolicyRows(rows)
}

// GetPolicyVersions gets all versions of a policy
func (s *PolicyStore) GetPolicyVersions(ctx context.Context, policyID string) ([]*PolicyRecord, error) {
	query := `
		SELECT id, policy_id, version, name, description, dsl_yaml, effective_from, effective_until, is_active, created_by, created_at
		FROM policies
		WHERE policy_id = $1
		ORDER BY effective_from DESC`

	rows, err := s.client.Pool().Query(ctx, query, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPolicyRows(rows)
}

// DeactivatePolicy deactivates a policy
func (s *PolicyStore) DeactivatePolicy(ctx context.Context, policyID, version string) error {
	query := `UPDATE policies SET is_active = false WHERE policy_id = $1 AND version = $2`
	_, err := s.client.Pool().Exec(ctx, query, policyID, version)
	return err
}

func scanPolicyRows(rows pgx.Rows) ([]*PolicyRecord, error) {
	var policies []*PolicyRecord
	for rows.Next() {
		var p PolicyRecord
		err := rows.Scan(
			&p.ID, &p.PolicyID, &p.Version, &p.Name,
			&p.Description, &p.DSL, &p.EffectiveFrom,
			&p.EffectiveUntil, &p.IsActive, &p.CreatedBy, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		policies = append(policies, &p)
	}
	return policies, rows.Err()
}

// Helper to convert map to JSONB
func toJSONB(v interface{}) interface{} {
	b, _ := json.Marshal(v)
	return b
}
