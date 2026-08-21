package domain

import (
	"strings"
	"time"
)

type ExecutionRequestState string

const (
	ExecutionRequestSubmitted  ExecutionRequestState = "submitted"
	ExecutionRequestAuthorized ExecutionRequestState = "authorized"
	ExecutionRequestExecuting  ExecutionRequestState = "executing"
	ExecutionRequestCompleted  ExecutionRequestState = "completed"
	ExecutionRequestArchived   ExecutionRequestState = "archived"
	ExecutionRequestCancelled  ExecutionRequestState = "cancelled"
)

type ExecutionRequest struct {
	ID                  string                `json:"id"`
	WorkspaceID         string                `json:"workspace_id"`
	RequesterZoneID     string                `json:"requester_zone_id"`
	ExecutionZoneID     string                `json:"execution_zone_id"`
	ExecutionPoolID     string                `json:"execution_pool_id"`
	RequestKey          string                `json:"request_key"`
	State               ExecutionRequestState `json:"state"`
	ScheduledStartAt    time.Time             `json:"scheduled_start_at"`
	ExpectedFinishAt    time.Time             `json:"expected_finish_at"`
	StartedAt           *time.Time            `json:"started_at,omitempty"`
	CompletedAt         *time.Time            `json:"completed_at,omitempty"`
	ArchivedAt          *time.Time            `json:"archived_at,omitempty"`
	TotalRequestedUnits int                   `json:"total_requested_units"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
	Version             int64                 `json:"version"`
}

type ExecutionRequestInput struct {
	ExecutionRequestID string
	ToolRevisionID     string
	AddedAt            time.Time
}

func (s ExecutionRequest) Validate() error {
	if strings.TrimSpace(s.WorkspaceID) == "" || strings.TrimSpace(s.RequesterZoneID) == "" || strings.TrimSpace(s.ExecutionZoneID) == "" {
		return FieldError{Field: "execution_request", Message: "workspace, requester trust zone and execution trust zone are required"}
	}
	if s.RequesterZoneID == s.ExecutionZoneID {
		return FieldError{Field: "execution_zone_id", Message: "must differ from requester trust zone"}
	}
	if strings.TrimSpace(s.RequestKey) == "" || strings.TrimSpace(s.ExecutionPoolID) == "" {
		return FieldError{Field: "execution_request", Message: "request key and execution pool are required"}
	}
	if !s.ExpectedFinishAt.After(s.ScheduledStartAt) {
		return FieldError{Field: "expected_finish_at", Message: "must be after scheduled start"}
	}
	if s.TotalRequestedUnits < 1 {
		return FieldError{Field: "total_requested_units", Message: "must be positive"}
	}
	return validateExecutionRequestState(s.State)
}

func validateExecutionRequestState(state ExecutionRequestState) error {
	switch state {
	case ExecutionRequestSubmitted, ExecutionRequestAuthorized, ExecutionRequestExecuting, ExecutionRequestCompleted, ExecutionRequestArchived, ExecutionRequestCancelled:
		return nil
	default:
		return FieldError{Field: "request_state", Message: "is invalid"}
	}
}

func (s ExecutionRequestState) IsTerminal() bool {
	return s == ExecutionRequestArchived || s == ExecutionRequestCancelled
}

func (s *ExecutionRequest) Transition(to ExecutionRequestState, now time.Time) error {
	allowed := map[ExecutionRequestState]map[ExecutionRequestState]bool{
		ExecutionRequestSubmitted:  {ExecutionRequestAuthorized: true, ExecutionRequestCancelled: true},
		ExecutionRequestAuthorized: {ExecutionRequestExecuting: true, ExecutionRequestCancelled: true},
		ExecutionRequestExecuting:  {ExecutionRequestCompleted: true},
		ExecutionRequestCompleted:  {ExecutionRequestArchived: true},
	}
	if !allowed[s.State][to] {
		return TransitionError{Entity: "execution_request", From: string(s.State), To: string(to)}
	}
	now = now.UTC()
	switch to {
	case ExecutionRequestExecuting:
		s.StartedAt = &now
	case ExecutionRequestCompleted:
		s.CompletedAt = &now
	case ExecutionRequestArchived:
		s.ArchivedAt = &now
	}
	s.State = to
	s.UpdatedAt = now
	return nil
}
