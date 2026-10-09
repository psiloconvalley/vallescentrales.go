package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Engine struct {
	db *pgxpool.Pool
}

func NewEngine(db *pgxpool.Pool) *Engine {
	return &Engine{db: db}
}

func (e *Engine) StartOrResume(ctx context.Context, userID uuid.UUID, flow FlowType) (*Session, error) {
	existing, err := e.Get(ctx, userID, flow)
	if err == nil && existing != nil {
		return existing, nil
	}

	totalSteps := 2
	if flow == FlowPropertyPublish {
		totalSteps = 3
	}

	now := time.Now().UTC()
	session := &Session{
		ID:          uuid.New(),
		UserID:      userID,
		FlowType:    flow,
		CurrentStep: 1,
		TotalSteps:  totalSteps,
		Data:        make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
	}

	dataJSON, _ := json.Marshal(session.Data)
	query := `
		INSERT INTO wizard_sessions (id, user_id, flow_type, current_step, total_steps, data, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = e.db.Exec(ctx, query,
		session.ID, session.UserID, string(session.FlowType),
		session.CurrentStep, session.TotalSteps, dataJSON,
		session.CreatedAt, session.UpdatedAt, session.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("wizard_engine: start: %w", err)
	}

	return session, nil
}

func (e *Engine) Get(ctx context.Context, userID uuid.UUID, flow FlowType) (*Session, error) {
	query := `
		SELECT id, user_id, flow_type, current_step, total_steps, data, created_at, updated_at, expires_at
		FROM wizard_sessions
		WHERE user_id = $1 AND flow_type = $2 AND expires_at > $3
		ORDER BY updated_at DESC
		LIMIT 1
	`
	var s Session
	var flowStr string
	var dataBytes []byte

	err := e.db.QueryRow(ctx, query, userID, string(flow), time.Now().UTC()).Scan(
		&s.ID, &s.UserID, &flowStr, &s.CurrentStep, &s.TotalSteps, &dataBytes,
		&s.CreatedAt, &s.UpdatedAt, &s.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("wizard_engine: get: %w", err)
	}

	s.FlowType = FlowType(flowStr)
	if len(dataBytes) > 0 {
		_ = json.Unmarshal(dataBytes, &s.Data)
	}
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}

	return &s, nil
}

func (e *Engine) SaveStep(ctx context.Context, userID uuid.UUID, flow FlowType, step int, stepData map[string]interface{}) (*Session, error) {
	session, err := e.StartOrResume(ctx, userID, flow)
	if err != nil {
		return nil, err
	}

	for k, v := range stepData {
		session.Data[k] = v
	}

	session.CurrentStep = step
	session.UpdatedAt = time.Now().UTC()

	dataJSON, _ := json.Marshal(session.Data)
	query := `
		UPDATE wizard_sessions
		SET current_step = $1, data = $2, updated_at = $3
		WHERE id = $4
	`
	_, err = e.db.Exec(ctx, query, session.CurrentStep, dataJSON, session.UpdatedAt, session.ID)
	if err != nil {
		return nil, fmt.Errorf("wizard_engine: save step: %w", err)
	}

	return session, nil
}

func (e *Engine) AutoSave(ctx context.Context, userID uuid.UUID, flow FlowType, step int, stepData map[string]interface{}) error {
	_, err := e.SaveStep(ctx, userID, flow, step, stepData)
	return err
}

func (e *Engine) Finalize(ctx context.Context, userID uuid.UUID, flow FlowType) error {
	query := `DELETE FROM wizard_sessions WHERE user_id = $1 AND flow_type = $2`
	_, err := e.db.Exec(ctx, query, userID, string(flow))
	if err != nil {
		return fmt.Errorf("wizard_engine: finalize: %w", err)
	}
	return nil
}
