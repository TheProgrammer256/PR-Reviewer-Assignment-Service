package service

import (
	"context"
	"errors"
	"net/http"

	openapi "github.com/TheProgrammer256/PR-Reviewer-Assignment-Service/go"

	"github.com/avito/pr-reviewer-assignment-service/internal/apperr"
	"github.com/avito/pr-reviewer-assignment-service/internal/storage"
)

type APIService struct {
	repo *storage.Repository
}

var _ openapi.PullRequestsAPIServicer = (*APIService)(nil)
var _ openapi.TeamsAPIServicer = (*APIService)(nil)
var _ openapi.UsersAPIServicer = (*APIService)(nil)

func New(repo *storage.Repository) *APIService {
	return &APIService{repo: repo}
}

// POST /team/add
func (s *APIService) CreateTeam(ctx context.Context, team openapi.Team) (openapi.ImplResponse, error) {
	members := make([]storage.TeamMember, 0, len(team.Members))
	for _, member := range team.Members {
		members = append(members, storage.TeamMember{
			ID:       member.UserId,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	created, err := s.repo.CreateTeam(ctx, team.TeamName, members)
	if err != nil {
		return s.fail(err)
	}

	resp := openapi.CreateTeam201Response{
		Team: teamToAPI(created),
	}
	return openapi.Response(http.StatusCreated, resp), nil
}

// GET /team/get
func (s *APIService) GetTeam(ctx context.Context, teamName string) (openapi.ImplResponse, error) {
	team, err := s.repo.GetTeam(ctx, teamName)
	if err != nil {
		return s.fail(err)
	}
	return openapi.Response(http.StatusOK, teamToAPI(team)), nil
}

// POST /users/setIsActive
func (s *APIService) UpdateActiveFlag(ctx context.Context, req openapi.UpdateActiveFlagRequest) (openapi.ImplResponse, error) {
	user, err := s.repo.UpdateUserActive(ctx, req.UserId, req.IsActive)
	if err != nil {
		return s.fail(err)
	}
	resp := openapi.UpdateActiveFlag200Response{
		User: userToAPI(user),
	}
	return openapi.Response(http.StatusOK, resp), nil
}

// POST /pullRequest/create
func (s *APIService) CreatePullRequestAndAssign(ctx context.Context, req openapi.CreatePullRequestAndAssignRequest) (openapi.ImplResponse, error) {
	pr, err := s.repo.CreatePullRequest(ctx, req.PullRequestId, req.PullRequestName, req.AuthorId)
	if err != nil {
		return s.fail(err)
	}
	resp := openapi.CreatePullRequestAndAssign201Response{
		Pr: prToAPI(pr),
	}
	return openapi.Response(http.StatusCreated, resp), nil
}

// POST /pullRequest/merge
func (s *APIService) UpdateMergedFlag(ctx context.Context, req openapi.UpdateMergedFlagRequest) (openapi.ImplResponse, error) {
	pr, err := s.repo.UpdatePullRequestMerged(ctx, req.PullRequestId)
	if err != nil {
		return s.fail(err)
	}
	resp := openapi.CreatePullRequestAndAssign201Response{
		Pr: prToAPI(pr),
	}
	return openapi.Response(http.StatusOK, resp), nil
}

// POST /pullRequest/reassign
func (s *APIService) ReassignUserOnPullRequest(ctx context.Context, req openapi.ReassignUserOnPullRequestRequest) (openapi.ImplResponse, error) {
	pr, replacement, err := s.repo.ReassignReviewer(ctx, req.PullRequestId, req.OldUserId)
	if err != nil {
		return s.fail(err)
	}
	resp := openapi.ReassignUserOnPullRequest200Response{
		Pr:         prToAPI(pr),
		ReplacedBy: replacement,
	}
	return openapi.Response(http.StatusOK, resp), nil
}

// GET /users/getReview
func (s *APIService) GetPullRequestsByUser(ctx context.Context, userID string) (openapi.ImplResponse, error) {
	prs, err := s.repo.ListPullRequestsByReviewer(ctx, userID)
	if err != nil {
		return s.fail(err)
	}
	resp := openapi.GetPullRequestsByUser200Response{
		UserId:       userID,
		PullRequests: make([]openapi.PullRequestShort, 0, len(prs)),
	}
	for _, pr := range prs {
		resp.PullRequests = append(resp.PullRequests, prShortToAPI(pr))
	}
	return openapi.Response(http.StatusOK, resp), nil
}

func (s *APIService) fail(err error) (openapi.ImplResponse, error) {
	if apiErr := mapError(err); apiErr != nil {
		return openapi.Response(apiErr.Status, apiErr.Response()), apiErr
	}
	return openapi.Response(http.StatusInternalServerError, nil), err
}

func mapError(err error) *apperr.APIError {
	switch {
	case errors.Is(err, storage.ErrTeamExists):
		return apperr.New(http.StatusBadRequest, "TEAM_EXISTS", "team already exists")
	case errors.Is(err, storage.ErrTeamNotFound):
		return apperr.New(http.StatusNotFound, "NOT_FOUND", "team not found")
	case errors.Is(err, storage.ErrUserNotFound):
		return apperr.New(http.StatusNotFound, "NOT_FOUND", "user not found")
	case errors.Is(err, storage.ErrPullRequestExists):
		return apperr.New(http.StatusConflict, "PR_EXISTS", "pull request already exists")
	case errors.Is(err, storage.ErrPullRequestNotFound):
		return apperr.New(http.StatusNotFound, "NOT_FOUND", "pull request not found")
	case errors.Is(err, storage.ErrPullRequestMerged):
		return apperr.New(http.StatusConflict, "PR_MERGED", "pull request already merged")
	case errors.Is(err, storage.ErrReviewerNotAssigned):
		return apperr.New(http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this pull request")
	case errors.Is(err, storage.ErrNoReviewerCandidate):
		return apperr.New(http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate available")
	default:
		return nil
	}
}

func teamToAPI(team storage.Team) openapi.Team {
	resp := openapi.Team{
		TeamName: team.Name,
		Members:  make([]openapi.TeamMember, 0, len(team.Members)),
	}
	for _, member := range team.Members {
		resp.Members = append(resp.Members, openapi.TeamMember{
			UserId:   member.ID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}
	return resp
}

func userToAPI(user storage.User) openapi.User {
	return openapi.User{
		UserId:   user.ID,
		Username: user.Name,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func prToAPI(pr storage.PullRequest) openapi.PullRequest {
	apiPR := openapi.PullRequest{
		PullRequestId:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorId:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: append([]string(nil), pr.AssignedReviewers...),
	}
	if !pr.CreatedAt.IsZero() {
		created := pr.CreatedAt.UTC()
		apiPR.CreatedAt = &created
	}
	if pr.MergedAt != nil {
		merged := pr.MergedAt.UTC()
		apiPR.MergedAt = &merged
	}
	return apiPR
}

func prShortToAPI(pr storage.PullRequestShort) openapi.PullRequestShort {
	return openapi.PullRequestShort{
		PullRequestId:   pr.ID,
		PullRequestName: pr.Name,
		AuthorId:        pr.AuthorID,
		Status:          pr.Status,
	}
}
