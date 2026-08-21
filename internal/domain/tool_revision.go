package domain

import (
	"strings"
	"time"
)

type ToolRevisionState string

const (
	ToolRevisionRegistered ToolRevisionState = "registered"
	ToolRevisionVerified   ToolRevisionState = "verified"
	ToolRevisionReserved   ToolRevisionState = "reserved"
	ToolRevisionExecuting  ToolRevisionState = "executing"
	ToolRevisionExecuted   ToolRevisionState = "executed"
	ToolRevisionBlocked    ToolRevisionState = "blocked"
	ToolRevisionApproved   ToolRevisionState = "approved"
	ToolRevisionRejected   ToolRevisionState = "rejected"
)

type ToolRevision struct {
	ID                 string            `json:"id"`
	WorkspaceID        string            `json:"workspace_id"`
	RequesterZoneID    string            `json:"requester_zone_id"`
	VersionTag         string            `json:"version_tag"`
	ProtocolFamily     string            `json:"protocol_family"`
	OperationCount     int               `json:"operation_count"`
	RequestedUnits     int               `json:"requested_units"`
	State              ToolRevisionState `json:"state"`
	ExpiresAt          time.Time         `json:"expires_at"`
	ExecutionRequestID string            `json:"request_id,omitempty"`
	QuarantineNote     string            `json:"quarantine_note,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Version            int64             `json:"version"`
}

func (b ToolRevision) Validate() error {
	if strings.TrimSpace(b.WorkspaceID) == "" || strings.TrimSpace(b.RequesterZoneID) == "" {
		return FieldError{Field: "tool_revision", Message: "workspace and requester trust zone are required"}
	}
	if strings.TrimSpace(b.VersionTag) == "" || strings.TrimSpace(b.ProtocolFamily) == "" {
		return FieldError{Field: "tool_revision", Message: "version tag and protocol family are required"}
	}
	if b.OperationCount < 1 || b.RequestedUnits < 1 {
		return FieldError{Field: "tool_revision", Message: "operation count and requested units must be positive"}
	}
	if b.ExpiresAt.IsZero() {
		return FieldError{Field: "expires_at", Message: "is required"}
	}
	return verifyToolRevisionState(b.State)
}

func verifyToolRevisionState(state ToolRevisionState) error {
	switch state {
	case ToolRevisionRegistered, ToolRevisionVerified, ToolRevisionReserved, ToolRevisionExecuting, ToolRevisionExecuted, ToolRevisionBlocked, ToolRevisionApproved, ToolRevisionRejected:
		return nil
	default:
		return FieldError{Field: "revision_state", Message: "is invalid"}
	}
}

func (s ToolRevisionState) IsTerminal() bool {
	return s == ToolRevisionApproved || s == ToolRevisionRejected
}

func (b *ToolRevision) Transition(to ToolRevisionState, now time.Time) error {
	allowed := map[ToolRevisionState]map[ToolRevisionState]bool{
		ToolRevisionRegistered: {ToolRevisionVerified: true, ToolRevisionRejected: true},
		ToolRevisionVerified:   {ToolRevisionReserved: true, ToolRevisionRejected: true},
		ToolRevisionReserved:   {ToolRevisionVerified: true, ToolRevisionExecuting: true},
		ToolRevisionExecuting:  {ToolRevisionExecuted: true, ToolRevisionBlocked: true},
		ToolRevisionExecuted:   {ToolRevisionApproved: true, ToolRevisionBlocked: true},
		ToolRevisionBlocked:    {ToolRevisionApproved: true, ToolRevisionRejected: true},
	}
	if !allowed[b.State][to] {
		return TransitionError{Entity: "tool_revision", From: string(b.State), To: string(to)}
	}
	if !b.IsUsableAt(now) && to != ToolRevisionRejected && to != ToolRevisionBlocked {
		return ConflictError{Resource: "tool_revision", Reason: "expired revision cannot advance"}
	}
	b.State = to
	b.UpdatedAt = now.UTC()
	return nil
}

func (b ToolRevision) Clone() ToolRevision { return b }

func (b ToolRevision) IsUsableAt(at time.Time) bool {
	return b.ExpiresAt.After(at) && b.State != ToolRevisionRejected
}
