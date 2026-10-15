package wizard

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Engine struct {
	db *pgxpool.Pool
}

func NewEngine(db *pgxpool.Pool) *Engine {
	// Ensure the wizard_sessions table exists defensively
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS wizard_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		flow_type VARCHAR(50) NOT NULL,
		current_step INT NOT NULL DEFAULT 1,
		total_steps INT NOT NULL DEFAULT 2,
		data JSONB NOT NULL DEFAULT '{}'::jsonb,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_wizard_sessions_user_flow ON wizard_sessions(user_id, flow_type);
	`
	if _, err := db.Exec(ctx, createTableSQL); err != nil {
		slog.Warn("could not auto-ensure wizard_sessions table", "error", err)
	} else {
		slog.Info("wizard_sessions table ready")
	}

	return &Engine{db: db}
}

func (e *Engine) StartOrResume(ctx context.Context, userID uuid.UUID, flow FlowType) (*Session, error) {
	totalSteps := 2
	if flow == FlowPropertyPublish || flow == FlowProperty {
		totalSteps = 3
	}

	// 1. Try to find existing valid session
	query := `
		SELECT id, user_id, flow_type, current_step, total_steps, data, expires_at, created_at, updated_at
		FROM wizard_sessions
		WHERE user_id = $1 AND flow_type = $2 AND expires_at > NOW()
		ORDER BY updated_at DESC
		LIMIT 1
	`
	var s Session
	var dataBytes []byte

	err := e.db.QueryRow(ctx, query, userID, string(flow)).Scan(
		&s.ID,
		&s.UserID,
		&s.FlowType,
		&s.CurrentStep,
		&s.TotalSteps,
		&dataBytes,
		&s.ExpiresAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err == nil {
		s.Data = make(map[string]any)
		if len(dataBytes) > 0 {
			if err := json.Unmarshal(dataBytes, &s.Data); err != nil {
				slog.Warn("could not unmarshal wizard session data", "error", err)
			}
		}
		return &s, nil
	}

	// 2. Otherwise create a new session
	s = Session{
		ID:          uuid.New(),
		UserID:      userID,
		FlowType:    flow,
		CurrentStep: 1,
		TotalSteps:  totalSteps,
		Data:        make(map[string]any),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	insertSQL := `
		INSERT INTO wizard_sessions (id, user_id, flow_type, current_step, total_steps, data, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6, $7, $8)
	`
	_, err = e.db.Exec(ctx, insertSQL, s.ID, s.UserID, string(s.FlowType), s.CurrentStep, s.TotalSteps, s.ExpiresAt, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		slog.Error("failed to create wizard session", "user_id", userID, "flow", flow, "error", err)
		return nil, fmt.Errorf("creating wizard session: %w", err)
	}

	return &s, nil
}

func (e *Engine) Get(ctx context.Context, userID uuid.UUID, flow FlowType) (*Session, error) {
	query := `
		SELECT id, user_id, flow_type, current_step, total_steps, data, expires_at, created_at, updated_at
		FROM wizard_sessions
		WHERE user_id = $1 AND flow_type = $2 AND expires_at > NOW()
		ORDER BY updated_at DESC
		LIMIT 1
	`
	var s Session
	var dataBytes []byte

	err := e.db.QueryRow(ctx, query, userID, string(flow)).Scan(
		&s.ID,
		&s.UserID,
		&s.FlowType,
		&s.CurrentStep,
		&s.TotalSteps,
		&dataBytes,
		&s.ExpiresAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	s.Data = make(map[string]any)
	if len(dataBytes) > 0 {
		if err := json.Unmarshal(dataBytes, &s.Data); err != nil {
			slog.Warn("could not unmarshal wizard data", "error", err)
		}
	}

	return &s, nil
}

func (e *Engine) SaveStep(ctx context.Context, userID uuid.UUID, flow FlowType, step int, stepData map[string]any) (*Session, error) {
	session, err := e.StartOrResume(ctx, userID, flow)
	if err != nil {
		return nil, fmt.Errorf("getting session for step save: %w", err)
	}

	if session.Data == nil {
		session.Data = make(map[string]any)
	}
	for k, v := range stepData {
		session.Data[k] = v
	}

	nextStep := step + 1
	if nextStep > session.TotalSteps {
		nextStep = session.TotalSteps
	}

	dataBytes, err := json.Marshal(session.Data)
	if err != nil {
		return nil, fmt.Errorf("marshaling wizard data: %w", err)
	}

	query := `
		UPDATE wizard_sessions
		SET current_step = $1,
		    data = $2::jsonb,
		    updated_at = NOW()
		WHERE id = $3
	`
	_, err = e.db.Exec(ctx, query, nextStep, dataBytes, session.ID)
	if err != nil {
		return nil, fmt.Errorf("updating wizard step: %w", err)
	}

	session.CurrentStep = nextStep
	return session, nil
}

func (e *Engine) AutoSave(ctx context.Context, userID uuid.UUID, flow FlowType, step int, stepData map[string]any) error {
	session, err := e.StartOrResume(ctx, userID, flow)
	if err != nil {
		return fmt.Errorf("getting session for autosave: %w", err)
	}

	if session.Data == nil {
		session.Data = make(map[string]any)
	}
	for k, v := range stepData {
		session.Data[k] = v
	}

	targetStep := session.CurrentStep
	if step > 0 && step <= session.TotalSteps {
		targetStep = step
	}

	dataBytes, err := json.Marshal(session.Data)
	if err != nil {
		return fmt.Errorf("marshaling autosave data: %w", err)
	}

	query := `
		UPDATE wizard_sessions
		SET current_step = $1,
		    data = $2::jsonb,
		    updated_at = NOW()
		WHERE id = $3
	`
	_, err = e.db.Exec(ctx, query, targetStep, dataBytes, session.ID)
	return err
}

func (e *Engine) Finalize(ctx context.Context, userID uuid.UUID, flow FlowType) error {
	session, err := e.Get(ctx, userID, flow)
	if err != nil {
		return fmt.Errorf("getting session for finalize: %w", err)
	}

	_, err = e.db.Exec(ctx, "DELETE FROM wizard_sessions WHERE id = $1", session.ID)
	return err
}

func (e *Engine) Delete(ctx context.Context, sessionID uuid.UUID) error {
	_, err := e.db.Exec(ctx, "DELETE FROM wizard_sessions WHERE id = $1", sessionID)
	return err
}
