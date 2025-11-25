package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS teams (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS pull_requests (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		author_id TEXT NOT NULL REFERENCES users(id),
		team_id INTEGER NOT NULL REFERENCES teams(id),
		status TEXT NOT NULL CHECK (status IN ('OPEN','MERGED')),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		merged_at TIMESTAMPTZ
	)`,
	`CREATE TABLE IF NOT EXISTS pull_request_reviewers (
		pull_request_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
		reviewer_id TEXT NOT NULL REFERENCES users(id),
		assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (pull_request_id, reviewer_id)
	)`,

	`CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_id, is_active)`,
	`CREATE INDEX IF NOT EXISTS idx_pull_request_reviewers_reviewer ON pull_request_reviewers(reviewer_id)`,
}

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	for _, stmt := range schemaStatements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
