package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/flamingoosesoftwareinc/flicker/engine"
	_ "modernc.org/sqlite"
)

// Store implements engine.WorkflowStore using modernc.org/sqlite (pure Go, no CGO).
type Store struct {
	db *sql.DB
}

// NewStore creates a new SQLite-backed workflow store. The DSN is a SQLite
// connection string (e.g., "file:test.db" or ":memory:").
func NewStore(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &Store{db: db}
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

func (s *Store) Create(ctx context.Context, record *engine.WorkflowRecord) error {
	now := time.Now().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now
	record.OCCVersion = 1

	if record.Status == "" {
		record.Status = engine.StatusPending
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

func (s *Store) Get(ctx context.Context, id string) (*engine.WorkflowRecord, error) {
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
	status engine.Status,
	occVersion int,
) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workflows SET status = ?, occ_version = occ_version + 1, updated_at = ?
		 WHERE id = ? AND occ_version = ?`,
		status, formatTime(time.Now().UTC()), id, occVersion,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	return checkRowsAffected(res)
}

func (s *Store) SetError(
	ctx context.Context,
	id string,
	status engine.Status,
	errMsg string,
	occVersion int,
) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workflows SET status = ?, error = ?, occ_version = occ_version + 1, updated_at = ?
		 WHERE id = ? AND occ_version = ?`,
		status, errMsg, formatTime(time.Now().UTC()), id, occVersion,
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
		engine.StatusPending,
		formatTime(retryAfter),
		formatTime(time.Now().UTC()),
		id,
		occVersion,
	)
	if err != nil {
		return fmt.Errorf("set retry: %w", err)
	}

	return checkRowsAffected(res)
}

func (s *Store) ListSchedulable(ctx context.Context, limit int) ([]*engine.WorkflowRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, type, version, status, signal, payload, error, retry_after, attempts, occ_version, created_at, updated_at
		 FROM workflows
		 WHERE status = ? AND (retry_after = '' OR retry_after <= ?)
		 ORDER BY created_at ASC
		 LIMIT ?`,
		engine.StatusPending,
		formatTime(time.Now().UTC()),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list schedulable: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []*engine.WorkflowRecord

	for rows.Next() {
		r, err := scanWorkflowRows(rows)
		if err != nil {
			return nil, err
		}

		records = append(records, r)
	}

	return records, rows.Err()
}

func (s *Store) SaveStepResult(ctx context.Context, result *engine.StepResult) error {
	result.CreatedAt = time.Now().UTC()

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
) (*engine.StepResult, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT workflow_id, step_name, result, error, created_at FROM step_results
		 WHERE workflow_id = ? AND step_name = ?`,
		workflowID, stepName,
	)

	var (
		r         engine.StepResult
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
) ([]*engine.StepResult, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT workflow_id, step_name, result, error, created_at FROM step_results
		 WHERE workflow_id = ? ORDER BY step_name ASC`,
		workflowID,
	)
	if err != nil {
		return nil, fmt.Errorf("list step results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*engine.StepResult

	for rows.Next() {
		var (
			r         engine.StepResult
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

type scannable interface {
	Scan(dest ...any) error
}

func scanWorkflow(row scannable) (*engine.WorkflowRecord, error) {
	var (
		r          engine.WorkflowRecord
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

func scanWorkflowRows(rows *sql.Rows) (*engine.WorkflowRecord, error) {
	var (
		r          engine.WorkflowRecord
		retryAfter string
		createdAt  string
		updatedAt  string
	)

	err := rows.Scan(
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
