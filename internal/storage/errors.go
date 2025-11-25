package storage

import "errors"

var (
	ErrTeamExists          = errors.New("team already exists")
	ErrTeamNotFound        = errors.New("team not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrPullRequestExists   = errors.New("pull request already exists")
	ErrPullRequestNotFound = errors.New("pull request not found")
	ErrPullRequestMerged   = errors.New("pull request already merged")
	ErrReviewerNotAssigned = errors.New("reviewer not assigned to pull request")
	ErrNoReviewerCandidate = errors.New("no active reviewer candidates available")
)
