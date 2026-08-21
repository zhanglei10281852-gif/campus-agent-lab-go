package domain

import (
	"strings"
	"time"
)

type ExecutionWindow struct {
	StartAt  time.Time
	FinishAt time.Time
}

func (w ExecutionWindow) Duration() time.Duration {
	return w.FinishAt.Sub(w.StartAt)
}

func (w ExecutionWindow) Validate(workspace Workspace, revisions []ToolRevision, now time.Time) error {
	if w.StartAt.IsZero() || w.FinishAt.IsZero() {
		return FieldError{Field: "execution_window", Message: "start and finish are required"}
	}
	if !w.FinishAt.After(w.StartAt) {
		return FieldError{Field: "finish_at", Message: "must be after start"}
	}
	if w.Duration() > workspace.MaxExecution {
		return ConflictError{Resource: "run", Reason: "execution exceeds workspace limit"}
	}
	if w.StartAt.Before(now.Add(-15 * time.Minute)) {
		return ConflictError{Resource: "execution_request", Reason: "execution window is already closed"}
	}
	for _, revision := range revisions {
		if !revision.ExpiresAt.After(w.FinishAt) {
			return ConflictError{Resource: "tool_revision", Reason: "revision expires before expected finish"}
		}
	}
	return nil
}

func ValidateRoute(source, target TrustZone) error {
	if strings.TrimSpace(source.ID) == "" || strings.TrimSpace(target.ID) == "" {
		return FieldError{Field: "route", Message: "source and execution trust zones are required"}
	}
	if source.ID == target.ID {
		return FieldError{Field: "route", Message: "source and execution trust zones must differ"}
	}
	if source.Status != TrustZoneActive || target.Status != TrustZoneActive {
		return ConflictError{Resource: "route", Reason: "both trust zones must be active"}
	}
	return nil
}
