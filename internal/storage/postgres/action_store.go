package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/jackc/pgx/v5"
)

// ActionStore handles action persistence
type ActionStore struct {
	client *Client
}

// NewActionStore creates a new action store
func NewActionStore(client *Client) *ActionStore {
	return &ActionStore{client: client}
}

// Store persists an action record
func (s *ActionStore) Store(ctx context.Context, action *models.Action) error {
	query := `
		INSERT INTO action_records (
			action_id, decision_id, action_type, action_payload, target_service,
			status, dry_run, scheduled_at, webhook_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	return s.client.Pool().QueryRow(ctx, query,
		action.Type, // Using type as ID for simplicity
		"",          // decision_id would be set properly in real usage
		action.Type,
		action.Payload,
		action.Target,
		"pending",
		false,
		nil,
		"",
	).Scan(nil, nil)
}

// GetByID retrieves an action by ID
func (s *ActionStore) GetByID(ctx context.Context, actionID string) (*ActionRecord, error) {
	query := `
		SELECT id, action_id, decision_id, action_type, action_payload, target_service,
			status, dry_run, scheduled_at, executed_at, completed_at, error_message,
			retry_count, webhook_url, webhook_response, created_at, updated_at
		FROM action_records WHERE action_id = $1`

	var record ActionRecord
	err := s.client.Pool().QueryRow(ctx, query, actionID).Scan(
		&record.ID, &record.ActionID, &record.DecisionID, &record.ActionType,
		&record.ActionPayload, &record.TargetService, &record.Status, &record.DryRun,
		&record.ScheduledAt, &record.ExecutedAt, &record.CompletedAt, &record.ErrorMessage,
		&record.RetryCount, &record.WebhookURL, &record.WebhookResponse,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("action not found: %s", actionID)
		}
		return nil, err
	}

	return &record, nil
}

// ActionRecord represents a stored action
type ActionRecord struct {
	ID              string      `db:"id"`
	ActionID        string      `db:"action_id"`
	DecisionID      string      `db:"decision_id"`
	ActionType      string      `db:"action_type"`
	ActionPayload   interface{} `db:"action_payload"`
	TargetService   string      `db:"target_service"`
	Status          string      `db:"status"`
	DryRun          bool        `db:"dry_run"`
	ScheduledAt     interface{} `db:"scheduled_at"`
	ExecutedAt      interface{} `db:"executed_at"`
	CompletedAt     interface{} `db:"completed_at"`
	ErrorMessage    string      `db:"error_message"`
	RetryCount      int         `db:"retry_count"`
	WebhookURL      string      `db:"webhook_url"`
	WebhookResponse interface{} `db:"webhook_response"`
	CreatedAt       interface{} `db:"created_at"`
	UpdatedAt       interface{} `db:"updated_at"`
}

// ListActions retrieves actions with filters
func (s *ActionStore) ListActions(ctx context.Context, filters ActionFilters) ([]*ActionRecord, error) {
	query := `
		SELECT id, action_id, decision_id, action_type, action_payload, target_service,
			status, dry_run, scheduled_at, executed_at, completed_at, error_message,
			retry_count, created_at, updated_at
		FROM action_records WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	if filters.DecisionID != "" {
		argCount++
		query += fmt.Sprintf(" AND decision_id = $%d", argCount)
		args = append(args, filters.DecisionID)
	}
	if filters.ServiceID != "" {
		argCount++
		query += fmt.Sprintf(" AND target_service = $%d", argCount)
		args = append(args, filters.ServiceID)
	}
	if filters.Status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filters.Status)
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
	}

	rows, err := s.client.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanActionRows(rows)
}

// ActionFilters for listing actions
type ActionFilters struct {
	DecisionID string
	ServiceID  string
	Status     string
	Limit      int
}

// UpdateStatus updates action status
func (s *ActionStore) UpdateStatus(ctx context.Context, actionID, status string, errorMsg string) error {
	query := `
		UPDATE action_records 
		SET status = $1, error_message = $2, updated_at = NOW()
		WHERE action_id = $3`
	_, err := s.client.Pool().Exec(ctx, query, status, errorMsg, actionID)
	return err
}

// MarkExecuted marks an action as executed
func (s *ActionStore) MarkExecuted(ctx context.Context, actionID string, response interface{}) error {
	query := `
		UPDATE action_records 
		SET status = 'completed', executed_at = NOW(), completed_at = NOW(), webhook_response = $1, updated_at = NOW()
		WHERE action_id = $2`
	_, err := s.client.Pool().Exec(ctx, query, response, actionID)
	return err
}

// IncrementRetry increments retry count
func (s *ActionStore) IncrementRetry(ctx context.Context, actionID string) error {
	query := `
		UPDATE action_records 
		SET retry_count = retry_count + 1, updated_at = NOW()
		WHERE action_id = $1`
	_, err := s.client.Pool().Exec(ctx, query, actionID)
	return err
}

// GetPendingActions retrieves pending or scheduled actions
func (s *ActionStore) GetPendingActions(ctx context.Context, limit int) ([]*ActionRecord, error) {
	query := `
		SELECT id, action_id, decision_id, action_type, action_payload, target_service,
			status, dry_run, scheduled_at, retry_count, created_at
		FROM action_records 
		WHERE status IN ('pending', 'scheduled')
			AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := s.client.Pool().Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanActionRows(rows)
}

// GetActionStats returns action statistics
func (s *ActionStore) GetActionStats(ctx context.Context, serviceID string, since time.Time) (map[string]int64, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM action_records 
		WHERE target_service = $1 AND created_at >= $2
		GROUP BY status`

	rows, err := s.client.Pool().Query(ctx, query, serviceID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, rows.Err()
}

func scanActionRows(rows pgx.Rows) ([]*ActionRecord, error) {
	var actions []*ActionRecord
	for rows.Next() {
		var a ActionRecord
		err := rows.Scan(
			&a.ID, &a.ActionID, &a.DecisionID, &a.ActionType, &a.ActionPayload,
			&a.TargetService, &a.Status, &a.DryRun, &a.ScheduledAt, &a.ExecutedAt,
			&a.CompletedAt, &a.ErrorMessage, &a.RetryCount, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		actions = append(actions, &a)
	}
	return actions, rows.Err()
}
