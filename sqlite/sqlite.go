package sqlite

import (
	"context"
	"database/sql"
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
			workflow_id TEXT NOT NULL,
			step_name   TEXT NOT NULL,
			result      BLOB,
			error       TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL,
			PRIMARY KEY (workflow_id, step_name)
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
	res, err := s.db.ExecContext(ctx,
		`UPDATE workflows SET status = ?, occ_version = occ_version + 1, updated_at = ?
		 WHERE status = ? AND retry_after != '' AND retry_after <= ?`,
		flicker.StatusPending, formatTime(s.now()), flicker.StatusSuspended, formatTime(now),
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
	result.CreatedAt = s.now()

	_, err := s.db.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO step_results (workflow_id, step_name, result, error, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
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
	workflowID, stepName string,
) (*flicker.StepResult, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT workflow_id, step_name, result, error, created_at FROM step_results
		 WHERE workflow_id = ? AND step_name = ?`,
		workflowID, stepName,
	)

	var (
		r         flicker.StepResult
		createdAt string
	)

	err := row.Scan(&r.WorkflowID, &r.StepName, &r.Result, &r.Error, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("get step result: %w", err)
	}

	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)

	return &r, nil
}

func (s *Store) ListStepResults(
	ctx context.Context,
	workflowID string,
) ([]*flicker.StepResult, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT workflow_id, step_name, result, error, created_at FROM step_results
		 WHERE workflow_id = ? ORDER BY step_name ASC`,
		workflowID,
	)
	if err != nil {
		return nil, fmt.Errorf("list step results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*flicker.StepResult

	for rows.Next() {
		var (
			r         flicker.StepResult
			createdAt string
		)

		if err := rows.Scan(
			&r.WorkflowID,
			&r.StepName,
			&r.Result,
			&r.Error,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan step result: %w", err)
		}

		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		results = append(results, &r)
	}

	return results, rows.Err()
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

	r.RetryAfter, _ = time.Parse(time.RFC3339Nano, retryAfter)
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &r, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339Nano)
}
