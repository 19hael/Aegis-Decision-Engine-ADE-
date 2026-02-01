package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/jackc/pgx/v5"
)

// FeatureStore handles feature snapshot persistence
type FeatureStore struct {
	client *Client
}

// NewFeatureStore creates a new feature store
func NewFeatureStore(client *Client) *FeatureStore {
	return &FeatureStore{client: client}
}

// Store persists a feature snapshot
func (s *FeatureStore) Store(ctx context.Context, snapshot *models.FeatureSnapshot) error {
	query := `
		INSERT INTO feature_snapshots (snapshot_id, service_id, features, valid_until, event_ids)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, calculated_at, created_at`

	err := s.client.Pool().QueryRow(ctx, query,
		snapshot.SnapshotID,
		snapshot.ServiceID,
		snapshot.Features,
		snapshot.ValidUntil,
		snapshot.EventIDs,
	).Scan(&snapshot.ID, &snapshot.CalculatedAt, &snapshot.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to store feature snapshot: %w", err)
	}

	return nil
}

// GetLatest retrieves the most recent feature snapshot for a service
func (s *FeatureStore) GetLatest(ctx context.Context, serviceID string) (*models.FeatureSnapshot, error) {
	query := `
		SELECT id, snapshot_id, service_id, features, calculated_at, valid_until, event_ids, created_at
		FROM feature_snapshots 
		WHERE service_id = $1 
			AND (valid_until IS NULL OR valid_until > NOW())
		ORDER BY calculated_at DESC 
		LIMIT 1`

	var snapshot models.FeatureSnapshot
	err := s.client.Pool().QueryRow(ctx, query, serviceID).Scan(
		&snapshot.ID, &snapshot.SnapshotID, &snapshot.ServiceID,
		&snapshot.Features, &snapshot.CalculatedAt, &snapshot.ValidUntil,
		&snapshot.EventIDs, &snapshot.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get feature snapshot: %w", err)
	}

	return &snapshot, nil
}

// GetByID retrieves a feature snapshot by its ID
func (s *FeatureStore) GetByID(ctx context.Context, snapshotID string) (*models.FeatureSnapshot, error) {
	query := `
		SELECT id, snapshot_id, service_id, features, calculated_at, valid_until, event_ids, created_at
		FROM feature_snapshots WHERE snapshot_id = $1`

	var snapshot models.FeatureSnapshot
	err := s.client.Pool().QueryRow(ctx, query, snapshotID).Scan(
		&snapshot.ID, &snapshot.SnapshotID, &snapshot.ServiceID,
		&snapshot.Features, &snapshot.CalculatedAt, &snapshot.ValidUntil,
		&snapshot.EventIDs, &snapshot.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get feature snapshot: %w", err)
	}

	return &snapshot, nil
}

// GetServiceFeatures retrieves the latest features for a service as a typed struct
func (s *FeatureStore) GetServiceFeatures(ctx context.Context, serviceID string) (*models.ServiceFeatures, error) {
	snapshot, err := s.GetLatest(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	var features models.ServiceFeatures
	if err := json.Unmarshal(snapshot.Features, &features); err != nil {
		return nil, fmt.Errorf("failed to unmarshal features: %w", err)
	}

	return &features, nil
}

// ListByService retrieves recent feature snapshots for a service
func (s *FeatureStore) ListByService(ctx context.Context, serviceID string, limit int) ([]models.FeatureSnapshot, error) {
	query := `
		SELECT id, snapshot_id, service_id, features, calculated_at, valid_until, event_ids, created_at
		FROM feature_snapshots 
		WHERE service_id = $1 
		ORDER BY calculated_at DESC 
		LIMIT $2`

	rows, err := s.client.Pool().Query(ctx, query, serviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query feature snapshots: %w", err)
	}
	defer rows.Close()

	return scanFeatureSnapshots(rows)
}

// Invalidate marks old snapshots as invalid
func (s *FeatureStore) Invalidate(ctx context.Context, serviceID string, olderThan time.Time) error {
	query := `
		UPDATE feature_snapshots 
		SET valid_until = NOW() 
		WHERE service_id = $1 AND calculated_at < $2`
	
	_, err := s.client.Pool().Exec(ctx, query, serviceID, olderThan)
	return err
}

func scanFeatureSnapshots(rows pgx.Rows) ([]models.FeatureSnapshot, error) {
	var snapshots []models.FeatureSnapshot
	for rows.Next() {
		var snapshot models.FeatureSnapshot
		err := rows.Scan(
			&snapshot.ID, &snapshot.SnapshotID, &snapshot.ServiceID,
			&snapshot.Features, &snapshot.CalculatedAt, &snapshot.ValidUntil,
			&snapshot.EventIDs, &snapshot.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}
