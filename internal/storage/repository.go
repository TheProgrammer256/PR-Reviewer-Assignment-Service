package storage

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
	rng  *rand.Rand
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *Repository) CreateTeam(ctx context.Context, name string, members []TeamMember) (Team, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Team{}, err
	}
	defer tx.Rollback(ctx)

	var teamID int64
	if err := tx.QueryRow(ctx, `INSERT INTO teams (name) VALUES ($1) RETURNING id`, name).Scan(&teamID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Team{}, ErrTeamExists
		}
		return Team{}, err
	}

	for _, m := range members {
		if m.ID == "" {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO users (id, username, is_active, team_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE
			SET username = EXCLUDED.username,
			    team_id = EXCLUDED.team_id,
			    is_active = EXCLUDED.is_active,
			    updated_at = NOW()`,
			m.ID, m.Username, m.IsActive, teamID)
		if err != nil {
			return Team{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Team{}, err
	}

	return r.GetTeam(ctx, name)
}

func (r *Repository) GetTeam(ctx context.Context, name string) (Team, error) {
	var (
		teamID int64
	)
	if err := r.pool.QueryRow(ctx, `SELECT id FROM teams WHERE name = $1`, name).Scan(&teamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Team{}, ErrTeamNotFound
		}
		return Team{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, username, is_active
		FROM users
		WHERE team_id = $1
		ORDER BY username`, teamID)
	if err != nil {
		return Team{}, err
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var tm TeamMember
		if err := rows.Scan(&tm.ID, &tm.Username, &tm.IsActive); err != nil {
			return Team{}, err
		}
		members = append(members, tm)
	}

	return Team{Name: name, Members: members}, rows.Err()
}

func (r *Repository) UpdateUserActive(ctx context.Context, userID string, active bool) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET is_active = $2,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, username, is_active,
			(SELECT name FROM teams WHERE teams.id = users.team_id) AS team_name`,
		userID, active,
	).Scan(&u.ID, &u.Name, &u.IsActive, &u.TeamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (r *Repository) CreatePullRequest(ctx context.Context, id, name, authorID string) (PullRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return PullRequest{}, err
	}
	defer tx.Rollback(ctx)

	var (
		teamID   int64
		teamName string
	)
	err = tx.QueryRow(ctx, `
		SELECT t.id, t.name
		FROM users u
		JOIN teams t ON t.id = u.team_id
		WHERE u.id = $1`,
		authorID,
	).Scan(&teamID, &teamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PullRequest{}, ErrUserNotFound
		}
		return PullRequest{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO pull_requests (id, name, author_id, team_id, status)
		VALUES ($1, $2, $3, $4, 'OPEN')`,
		id, name, authorID, teamID,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return PullRequest{}, ErrPullRequestExists
		}
		return PullRequest{}, err
	}

	reviewerRows, err := tx.Query(ctx, `
		SELECT u.id
		FROM users u
		WHERE u.team_id = $1
		  AND u.is_active
		  AND u.id <> $2
		ORDER BY random()
		LIMIT 2`,
		teamID, authorID,
	)
	if err != nil {
		return PullRequest{}, err
	}
	defer reviewerRows.Close()

	var reviewers []string
	for reviewerRows.Next() {
		var reviewerID string
		if err := reviewerRows.Scan(&reviewerID); err != nil {
			return PullRequest{}, err
		}
		reviewers = append(reviewers, reviewerID)
	}
	if err := reviewerRows.Err(); err != nil {
		return PullRequest{}, err
	}

	for _, reviewerID := range reviewers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
			VALUES ($1, $2)`,
			id, reviewerID,
		); err != nil {
			return PullRequest{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PullRequest{}, err
	}

	return r.GetPullRequest(ctx, id)
}

func (r *Repository) UpdatePullRequestMerged(ctx context.Context, id string) (PullRequest, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE pull_requests
		SET status = 'MERGED',
		    merged_at = COALESCE(merged_at, NOW())
		WHERE id = $1`,
		id,
	)
	if err != nil {
		return PullRequest{}, err
	}

	pr, err := r.GetPullRequest(ctx, id)
	if errors.Is(err, ErrPullRequestNotFound) {
		return PullRequest{}, ErrPullRequestNotFound
	}
	return pr, err
}

func (r *Repository) ReassignReviewer(ctx context.Context, pullRequestID, oldReviewerID string) (PullRequest, string, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return PullRequest{}, "", err
	}
	defer tx.Rollback(ctx)

	var pr PullRequest
	err = tx.QueryRow(ctx, `
		SELECT pr.id, pr.name, pr.author_id, pr.team_id, t.name, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		JOIN teams t ON t.id = pr.team_id
		WHERE pr.id = $1`,
		pullRequestID,
	).Scan(
		&pr.ID,
		&pr.Name,
		&pr.AuthorID,
		&pr.TeamID,
		&pr.TeamName,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PullRequest{}, "", ErrPullRequestNotFound
		}
		return PullRequest{}, "", err
	}

	if pr.Status == "MERGED" {
		return PullRequest{}, "", ErrPullRequestMerged
	}

	var assignedCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM pull_request_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2`,
		pullRequestID, oldReviewerID,
	).Scan(&assignedCount)
	if err != nil {
		return PullRequest{}, "", err
	}
	if assignedCount == 0 {
		return PullRequest{}, "", ErrReviewerNotAssigned
	}

	reviewerRows, err := tx.Query(ctx, `
		SELECT reviewer_id
		FROM pull_request_reviewers
		WHERE pull_request_id = $1`,
		pullRequestID,
	)
	if err != nil {
		return PullRequest{}, "", err
	}
	defer reviewerRows.Close()

	currentReviewers := map[string]struct{}{}
	for reviewerRows.Next() {
		var reviewerID string
		if err := reviewerRows.Scan(&reviewerID); err != nil {
			return PullRequest{}, "", err
		}
		currentReviewers[reviewerID] = struct{}{}
	}
	if err := reviewerRows.Err(); err != nil {
		return PullRequest{}, "", err
	}

	candidateRows, err := tx.Query(ctx, `
		SELECT u.id
		FROM users u
		WHERE u.team_id = $1
		  AND u.is_active
		  AND u.id <> $2`,
		pr.TeamID, pr.AuthorID,
	)
	if err != nil {
		return PullRequest{}, "", err
	}
	defer candidateRows.Close()

	var candidates []string
	for candidateRows.Next() {
		var candidateID string
		if err := candidateRows.Scan(&candidateID); err != nil {
			return PullRequest{}, "", err
		}
		if _, already := currentReviewers[candidateID]; already {
			continue
		}
		candidates = append(candidates, candidateID)
	}
	if err := candidateRows.Err(); err != nil {
		return PullRequest{}, "", err
	}

	if len(candidates) == 0 {
		return PullRequest{}, "", ErrNoReviewerCandidate
	}

	newReviewer := candidates[r.rng.Intn(len(candidates))]
	_, err = tx.Exec(ctx, `
		UPDATE pull_request_reviewers
		SET reviewer_id = $3,
		    assigned_at = NOW()
		WHERE pull_request_id = $1 AND reviewer_id = $2`,
		pullRequestID, oldReviewerID, newReviewer,
	)
	if err != nil {
		return PullRequest{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return PullRequest{}, "", err
	}

	updatedPR, err := r.GetPullRequest(ctx, pullRequestID)
	return updatedPR, newReviewer, err
}

func (r *Repository) GetPullRequest(ctx context.Context, id string) (PullRequest, error) {
	var pr PullRequest
	err := r.pool.QueryRow(ctx, `
		SELECT pr.id, pr.name, pr.author_id, pr.team_id, t.name, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		JOIN teams t ON t.id = pr.team_id
		WHERE pr.id = $1`,
		id,
	).Scan(
		&pr.ID,
		&pr.Name,
		&pr.AuthorID,
		&pr.TeamID,
		&pr.TeamName,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PullRequest{}, ErrPullRequestNotFound
		}
		return PullRequest{}, err
	}

	reviewerRows, err := r.pool.Query(ctx, `
		SELECT reviewer_id
		FROM pull_request_reviewers
		WHERE pull_request_id = $1
		ORDER BY assigned_at`,
		id,
	)
	if err != nil {
		return PullRequest{}, err
	}
	defer reviewerRows.Close()

	for reviewerRows.Next() {
		var reviewerID string
		if err := reviewerRows.Scan(&reviewerID); err != nil {
			return PullRequest{}, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}
	if err := reviewerRows.Err(); err != nil {
		return PullRequest{}, err
	}

	return pr, nil
}

func (r *Repository) ListPullRequestsByReviewer(ctx context.Context, reviewerID string) ([]PullRequestShort, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pr.id, pr.name, pr.author_id, pr.status, pr.created_at
		FROM pull_requests pr
		JOIN pull_request_reviewers rvr ON rvr.pull_request_id = pr.id
		WHERE rvr.reviewer_id = $1
		ORDER BY pr.created_at DESC`,
		reviewerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []PullRequestShort
	for rows.Next() {
		var pr PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	return prs, rows.Err()
}
