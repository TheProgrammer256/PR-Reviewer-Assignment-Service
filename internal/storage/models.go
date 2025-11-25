package storage

import "time"

type TeamMember struct {
	ID       string
	Username string
	IsActive bool
}

type Team struct {
	Name    string
	Members []TeamMember
}

type User struct {
	ID       string
	Name     string
	IsActive bool
	TeamName string
}

type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	TeamID            int64
	TeamName          string
	Status            string
	CreatedAt         time.Time
	MergedAt          *time.Time
	AssignedReviewers []string
}

type PullRequestShort struct {
	ID        string
	Name      string
	AuthorID  string
	Status    string
	CreatedAt time.Time
}
