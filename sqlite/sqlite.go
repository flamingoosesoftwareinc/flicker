package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	_ "modernc.org/sqlite"
)

// StoreOption configures the SQLite store.
type StoreOption func(*Store)

// WithNowFunc sets a custom clock for timestamps. Defaults to time.Now().UTC().
// Use this in tests for deterministic timestamps.
func WithNowFunc(fn func() time.Time) StoreOption {
	return func(s *Store) {
		s.nowFn = fn
	}
}

// Store implements flicker.WorkflowStore using modernc.org/sqlite (pure Go, no CGO).
type Store struct {
	db    *sql.DB
	nowFn func() time.Time
}

// NewStore creates a new SQLite-backed workflow store. The DSN is a SQLite
// connection string (e.g., "file:test.db" or ":memory:").
func NewStore(ctx context.Context, dsn string, opts ...StoreOption) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &Store{
		db:    db,
		nowFn: func() time.Time { return time.Now().UTC() },
	}

	for _, opt := range opts {
		opt(s)
	}

	if err := s.migrate(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS workflows (
			id          TEXT PRIMARY KEY,
			type        TEXT NOT NULL,
			version     TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'pending',
			signal      TEXT NOT NULL DEFAULT '',
			payload     BLOB,
			error       TEXT NOT NULL DEFAULT '',
			retry_after TEXT NOT NULL DEFAULT '',
			attempts    INTEGER NOT NULL DEFAULT 0,
			occ_version INTEGER NOT NULL DEFAULT 1,
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS step_results (
			type        TEXT NOT NULL,
			version     TEXT NOT NULL,
			workflow_id TEXT NOT NULL,
			step_name   TEXT NOT NULL,
			result      BLOB,
			error       TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL,
			PRIMARY KEY (type, version, workflow_id, step_name)
		);

		CREATE TABLE IF NOT EXISTS subscriptions (
			correlation_key TEXT PRIMARY KEY,
			workflow_id     TEXT NOT NULL,
			type            TEXT NOT NULL,
			version         TEXT NOT NULL,
			step_name       TEXT NOT NULL,
			deadline        TEXT NOT NULL,
			created_at      TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) now() time.Time {
	return s.nowFn()
}

func (s *Store) Create(ctx context.Context, record *flicker.WorkflowRecord) error {
	now := s.now()
	record.CreatedAt = now
	record.UpdatedAt = now
	record.OCCVersion = 1

	if record.Status == "" {
		record.Status = flicker.StatusPending
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO workflows (id, type, version, status, signal, payload, error, retry_after, attempts, occ_version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.Type,
		record.Version,
		record.Status,
		record.Signal,
		record.Payload,
		record.Error,
		formatTime(record.RetryAfter),
		record.Attempts,
		record.OCCVersion,
		formatTime(now),
		formatTime(now),
	)
	if err != nil {
		return fmt.Errorf("insert workflow: %w", err)
	}

	return nil
}

func (s *Store) Get(ctx context.Context, id string) (*flicker.WorkflowRecord, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, type, version, status, signal, payload, error, retry_after, attempts, occ_version, created_at, updated_at
		 FROM workflows WHERE id = ?`,
		id,
	)

	return scanWorkflow(row)
}

func (s *Store) UpdateStatus(
	ctx context.Context,
	id string,
	status flicker.Status,
	occVersion int,
) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workflows SET status = ?, occ_version = occ_version + 1, updated_at = ?
		 WHERE id = ? AND occ_version = ?`,
		status, formatTime(s.now()), id, occVersion,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	return checkRowsAffected(res)
}

func (s *Store) SetError(
	ctx context.Context,
	id string,
	status flicker.Status,
	errMsg string,
	occVersion int,
) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workflows SET status = ?, error = ?, occ_version = occ_version + 1, updated_at = ?
		 WHERE id = ? AND occ_version = ?`,
		status, errMsg, formatTime(s.now()), id, occVersion,
	)
	if err != nil {
		return fmt.Errorf("set error: %w", err)
	}

	return checkRowsAffected(res)
}

func (s *Store) SetRetry(
	ctx context.Context,
	id string,
	retryAfter time.Time,
	occVersion int,
) error {
	res, err := s.db.ExecContext(
		ctx,
		`UPDATE workflows SET status = ?, retry_after = ?, attempts = attempts + 1, occ_version = occ_version + 1, updated_at = ?
		 WHERE id = ? AND occ_version = ?`,
		flicker.StatusPending,
		formatTime(retryAfter),
		formatTime(s.now()),
		id,
		occVersion,
	)
	if err != nil {
		return fmt.Errorf("set retry: %w", err)
	}

	return checkRowsAffected(res)
}

func (s *Store) Suspend(
	ctx context.Context,
	id string,
	resumeAt time.Time,
	occVersion int,
) error {
	res, err := s.db.ExecContext(
		ctx,
		`UPDATE workflows SET status = ?, retry_after = ?, occ_version = occ_version + 1, updated_at = ?
		 WHERE id = ? AND occ_version = ?`,
		flicker.StatusSuspended,
		formatTime(resumeAt),
		formatTime(s.now()),
		id,
		occVersion,
	)
	if err != nil {
		return fmt.Errorf("suspend: %w", err)
	}

	return checkRowsAffected(res)
}

func (s *Store) PromoteSuspended(ctx context.Context, now time.Time) (int, error) {
	res, err := s.db.ExecContext(
		ctx,
		`UPDATE workflows SET status = ?, retry_after = '', occ_version = occ_version + 1, updated_at = ?
		 WHERE status = ? AND retry_after != '' AND retry_after <= ?`,
		flicker.StatusPending,
		formatTime(s.now()),
		flicker.StatusSuspended,
		formatTime(now),
	)
	if err != nil {
		return 0, fmt.Errorf("promote suspended: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	return int(n), nil
}

func (s *Store) ListSchedulable(ctx context.Context, limit int) ([]*flicker.WorkflowRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, type, version, status, signal, payload, error, retry_after, attempts, occ_version, created_at, updated_at
		 FROM workflows
		 WHERE status = ? AND (retry_after = '' OR retry_after <= ?)
		 ORDER BY created_at ASC
		 LIMIT ?`,
		flicker.StatusPending,
		formatTime(s.now()),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list schedulable: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []*flicker.WorkflowRecord

	for rows.Next() {
		r, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}

		records = append(records, r)
	}

	return records, rows.Err()
}

func (s *Store) SaveStepResult(ctx context.Context, result *flicker.StepResult) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO step_results (type, version, workflow_id, step_name, result, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		result.Type,
		result.Version,
		result.WorkflowID,
		result.StepName,
		result.Result,
		result.Error,
		formatTime(result.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("save step result: %w", err)
	}

	return nil
}

func (s *Store) GetStepResult(
	ctx context.Context,
	wfType, version, workflowID, stepName string,
) (*flicker.StepResult, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT type, version, workflow_id, step_name, result, error, created_at FROM step_results
		 WHERE type = ? AND version = ? AND workflow_id = ? AND step_name = ?`,
		wfType, version, workflowID, stepName,
	)

	return scanStepResult(row)
}

func (s *Store) ListStepResults(
	ctx context.Context,
	wfType, version, workflowID string,
) ([]*flicker.StepResult, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT type, version, workflow_id, step_name, result, error, created_at FROM step_results
		 WHERE type = ? AND version = ? AND workflow_id = ?
		 ORDER BY created_at ASC`,
		wfType, version, workflowID,
	)
	if err != nil {
		return nil, fmt.Errorf("list step results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*flicker.StepResult

	for rows.Next() {
		r, err := scanStepResult(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// --- Signal management ---

func (s *Store) GetSignal(ctx context.Context, id string) (flicker.Signal, error) {
	var signal flicker.Signal
	err := s.db.QueryRowContext(ctx,
		`SELECT signal FROM workflows WHERE id = ?`, id,
	).Scan(&signal)
	if err != nil {
		return "", fmt.Errorf("get signal: %w", err)
	}

	return signal, nil
}

func (s *Store) SetSignal(ctx context.Context, id string, signal flicker.Signal) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workflows SET signal = ?, updated_at = ? WHERE id = ?`,
		signal, formatTime(s.now()), id,
	)
	if err != nil {
		return fmt.Errorf("set signal: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("workflow %q not found", id)
	}

	return nil
}

// --- Event subscriptions ---

// ErrSubscriptionNotFound is returned when no subscription exists for a
// correlation key.
var ErrSubscriptionNotFound = fmt.Errorf("subscription not found")

func (s *Store) SaveSubscription(ctx context.Context, sub *flicker.Subscription) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO subscriptions (correlation_key, workflow_id, type, version, step_name, deadline, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sub.CorrelationKey,
		sub.WorkflowID,
		sub.Type,
		sub.Version,
		sub.StepName,
		formatTime(sub.Deadline),
		formatTime(sub.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("save subscription: %w", err)
	}

	return nil
}

func (s *Store) ResumeSubscription(
	ctx context.Context,
	correlationKey string,
	payload []byte,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Find the subscription.
	var sub flicker.Subscription
	var deadline, createdAt string
	err = tx.QueryRowContext(ctx,
		`SELECT workflow_id, type, version, step_name, deadline, created_at
		 FROM subscriptions WHERE correlation_key = ?`,
		correlationKey,
	).Scan(&sub.WorkflowID, &sub.Type, &sub.Version, &sub.StepName, &deadline, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSubscriptionNotFound
	}
	if err != nil {
		return fmt.Errorf("find subscription: %w", err)
	}

	// Save the event payload as the step result.
	_, err = tx.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO step_results (type, version, workflow_id, step_name, result, error, created_at)
		 VALUES (?, ?, ?, ?, ?, '', ?)`,
		sub.Type,
		sub.Version,
		sub.WorkflowID,
		sub.StepName,
		payload,
		formatTime(s.now()),
	)
	if err != nil {
		return fmt.Errorf("save event step result: %w", err)
	}

	// Delete the subscription.
	_, err = tx.ExecContext(ctx,
		`DELETE FROM subscriptions WHERE correlation_key = ?`,
		correlationKey,
	)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}

	// Promote the workflow to pending and clear retry_after so it's
	// immediately schedulable.
	_, err = tx.ExecContext(
		ctx,
		`UPDATE workflows SET status = ?, retry_after = '', occ_version = occ_version + 1, updated_at = ?
		 WHERE id = ? AND status = ?`,
		flicker.StatusPending,
		formatTime(s.now()),
		sub.WorkflowID,
		flicker.StatusSuspended,
	)
	if err != nil {
		return fmt.Errorf("promote workflow: %w", err)
	}

	return tx.Commit()
}

func (s *Store) TimeOutSubscriptions(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Find expired subscriptions.
	rows, err := tx.QueryContext(ctx,
		`SELECT correlation_key, workflow_id, type, version, step_name
		 FROM subscriptions WHERE deadline != '' AND deadline <= ?`,
		formatTime(now),
	)
	if err != nil {
		return 0, fmt.Errorf("query expired subscriptions: %w", err)
	}

	type expired struct {
		correlationKey, workflowID, wfType, version, stepName string
	}
	var subs []expired

	for rows.Next() {
		var e expired
		if err := rows.Scan(
			&e.correlationKey,
			&e.workflowID,
			&e.wfType,
			&e.version,
			&e.stepName,
		); err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("scan expired subscription: %w", err)
		}
		subs = append(subs, e)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate expired subscriptions: %w", err)
	}
	_ = rows.Close()

	for _, sub := range subs {
		// Save timeout marker as step result.
		_, err = tx.ExecContext(
			ctx,
			`INSERT OR REPLACE INTO step_results (type, version, workflow_id, step_name, result, error, created_at)
			 VALUES (?, ?, ?, ?, NULL, ?, ?)`,
			sub.wfType,
			sub.version,
			sub.workflowID,
			sub.stepName,
			"event_timeout",
			formatTime(s.now()),
		)
		if err != nil {
			return 0, fmt.Errorf("save timeout marker: %w", err)
		}

		// Delete the subscription.
		_, err = tx.ExecContext(ctx,
			`DELETE FROM subscriptions WHERE correlation_key = ?`,
			sub.correlationKey,
		)
		if err != nil {
			return 0, fmt.Errorf("delete expired subscription: %w", err)
		}

		// Promote the workflow and clear retry_after.
		_, err = tx.ExecContext(
			ctx,
			`UPDATE workflows SET status = ?, retry_after = '', occ_version = occ_version + 1, updated_at = ?
			 WHERE id = ? AND status = ?`,
			flicker.StatusPending,
			formatTime(s.now()),
			sub.workflowID,
			flicker.StatusSuspended,
		)
		if err != nil {
			return 0, fmt.Errorf("promote timed out workflow: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return len(subs), nil
}

func scanStepResult(row scannable) (*flicker.StepResult, error) {
	var (
		r         flicker.StepResult
		createdAt string
	)

	err := row.Scan(
		&r.Type,
		&r.Version,
		&r.WorkflowID,
		&r.StepName,
		&r.Result,
		&r.Error,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, flicker.ErrStepNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan step result: %w", err)
	}

	var parseErr error
	r.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, createdAt)
	if parseErr != nil {
		return nil, fmt.Errorf("parse step result created_at %q: %w", createdAt, parseErr)
	}

	return &r, nil
}

// ErrOCCConflict is returned when an optimistic concurrency check fails.
var ErrOCCConflict = fmt.Errorf("optimistic concurrency conflict: row not updated")

func checkRowsAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if n == 0 {
		return ErrOCCConflict
	}

	return nil
}

// scannable abstracts *sql.Row and *sql.Rows for shared scan logic.
type scannable interface {
	Scan(dest ...any) error
}

func scanWorkflow(row scannable) (*flicker.WorkflowRecord, error) {
	var (
		r          flicker.WorkflowRecord
		retryAfter string
		createdAt  string
		updatedAt  string
	)

	err := row.Scan(
		&r.ID, &r.Type, &r.Version, &r.Status, &r.Signal,
		&r.Payload, &r.Error, &retryAfter, &r.Attempts,
		&r.OCCVersion, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan workflow: %w", err)
	}

	for _, p := range []struct {
		dest *time.Time
		raw  string
		name string
	}{
		{&r.RetryAfter, retryAfter, "retry_after"},
		{&r.CreatedAt, createdAt, "created_at"},
		{&r.UpdatedAt, updatedAt, "updated_at"},
	} {
		if p.raw == "" {
			continue
		}
		t, parseErr := time.Parse(time.RFC3339Nano, p.raw)
		if parseErr != nil {
			return nil, fmt.Errorf("parse workflow %s %q: %w", p.name, p.raw, parseErr)
		}
		*p.dest = t
	}

	return &r, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339Nano)
}
