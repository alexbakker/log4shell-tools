package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	schema = `
CREATE TABLE IF NOT EXISTS test (
	id UUID NOT NULL,
	created timestamp NOT NULL DEFAULT timezone('utc'::text, CURRENT_TIMESTAMP),
	finished timestamp,
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS test_result (
	id BIGSERIAL NOT NULL,
	test_id UUID NOT NULL,
	created timestamp NOT NULL DEFAULT timezone('utc'::text, CURRENT_TIMESTAMP),
	type TEXT NOT NULL,
	addr TEXT,
	ptr TEXT,
	PRIMARY KEY (id),
	FOREIGN KEY (test_id) REFERENCES test (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS test_result_test_id_idx ON test_result (test_id);
`
)

type DB struct {
	p *pgxpool.Pool
}

func NewDB(connStr string) (*DB, error) {
	p, err := pgxpool.Connect(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("db connect: %s", err)
	}

	if _, err = p.Exec(context.Background(), schema); err != nil {
		return nil, fmt.Errorf("schema init: %s", err)
	}

	return &DB{p: p}, nil
}

func (db *DB) Close() {
	db.p.Close()
}

func (db *DB) Test(ctx context.Context, id uuid.UUID) (*Test, error) {
	row := db.p.QueryRow(ctx, `SELECT id, created, finished FROM test WHERE id = $1`, id.String())

	var test Test
	if err := row.Scan(
		&test.ID,
		&test.Created,
		&test.Finished); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &test, nil
}

func (db *DB) InsertTest(ctx context.Context, id uuid.UUID) error {
	_, err := db.p.Exec(ctx, "INSERT INTO test (id) VALUES($1)", id)
	return err
}

func (db *DB) InsertTestResult(ctx context.Context, t *Test, resultType string, addr string, ptr *string) error {
	_, err := db.p.Exec(ctx, `
		INSERT INTO test_result (test_id, type, addr, ptr)
		VALUES($1, $2, $3, $4)
	`, t.ID, resultType, addr, ptr)
	return err
}

func (db *DB) TestResults(ctx context.Context, t *Test) ([]*TestResult, error) {
	rows, err := db.p.Query(ctx, `
		SELECT created, type, addr, ptr
		FROM test_result
		WHERE test_id = $1
		ORDER BY created ASC
	`, t.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*TestResult
	for rows.Next() {
		var res TestResult
		if err = rows.Scan(&res.Created, &res.Type, &res.Addr, &res.Ptr); err != nil {
			return nil, err
		}

		results = append(results, &res)
	}

	return results, nil
}

func (db *DB) PruneTestResults(ctx context.Context) (int64, error) {
	res, err := db.p.Exec(ctx, `
		DELETE FROM test
		WHERE created < timezone('utc'::text, CURRENT_TIMESTAMP) - '1 day'::interval
	`)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected(), nil
}

func (db *DB) FinishTest(ctx context.Context, t *Test) error {
	_, err := db.p.Exec(ctx, `
		UPDATE test
		SET finished = timezone('utc'::text, CURRENT_TIMESTAMP)
		WHERE id = $1
	`, t.ID)
	return err
}

func (db *DB) ActiveTests(ctx context.Context, timeout time.Duration) (int64, error) {
	var count int64
	row := db.p.QueryRow(ctx, `
		SELECT count(*)
		FROM test
		WHERE finished IS NULL
			AND created > timezone('utc'::text, CURRENT_TIMESTAMP) - ('1 minute'::interval * $1);
	`, int64(timeout.Minutes()))

	err := row.Scan(&count)
	return count, err
}
