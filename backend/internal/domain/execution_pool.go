package domain

import (
	"strings"
	"time"
)

type ExecutionPoolState string

const (
	ExecutionPoolAvailable   ExecutionPoolState = "available"
	ExecutionPoolReserved    ExecutionPoolState = "reserved"
	ExecutionPoolAllocated   ExecutionPoolState = "allocated"
	ExecutionPoolReconciling ExecutionPoolState = "reconciling"
	ExecutionPoolRetired     ExecutionPoolState = "retired"
)

type ExecutionPool struct {
	ID                string             `json:"id"`
	PoolKey           string             `json:"pool_key"`
	State             ExecutionPoolState `json:"state"`
	CapacityUnits     int                `json:"capacity_units"`
	AttestationDueAt  time.Time          `json:"attestation_due_at"`
	LastReconciledAt  time.Time          `json:"last_reconciled_at"`
	ReservedRequestID string             `json:"reserved_request_id,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	Version           int64              `json:"version"`
}

func (c ExecutionPool) Validate() error {
	if strings.TrimSpace(c.PoolKey) == "" {
		return FieldError{Field: "pool_key", Message: "is required"}
	}
	if c.CapacityUnits < 100 || c.CapacityUnits > 1000000 {
		return FieldError{Field: "capacity_units", Message: "outside supported range"}
	}
	if c.AttestationDueAt.IsZero() || c.LastReconciledAt.IsZero() {
		return FieldError{Field: "execution_pool", Message: "attestation and reconciliation timestamps are required"}
	}
	switch c.State {
	case ExecutionPoolAvailable, ExecutionPoolReserved, ExecutionPoolAllocated, ExecutionPoolReconciling, ExecutionPoolRetired:
		return nil
	default:
		return FieldError{Field: "execution_pool_state", Message: "is invalid"}
	}
}

func (c ExecutionPool) EligibleFor(plannedStart time.Time, volume int) error {
	if c.State != ExecutionPoolAvailable {
		return ConflictError{Resource: "execution_pool", Reason: "not available"}
	}
	if !c.IsAttestedAt(plannedStart) {
		return ConflictError{Resource: "execution_pool", Reason: "attestation expires before scheduled start"}
	}
	if c.CapacityUnits < volume {
		return ErrCapacityExceeded
	}
	return nil
}

func (c ExecutionPool) IsAttestedAt(at time.Time) bool {
	return c.AttestationDueAt.After(at) && !c.LastReconciledAt.After(at)
}

func (c ExecutionPool) NeedsReconciliation(at time.Time) bool {
	return c.LastReconciledAt.IsZero() || c.LastReconciledAt.Before(at.Add(-72*time.Hour))
}

func (c *ExecutionPool) StartReconciliation(now time.Time) error {
	if c.State != ExecutionPoolAvailable {
		return TransitionError{Entity: "execution_pool", From: string(c.State), To: string(ExecutionPoolReconciling)}
	}
	c.State = ExecutionPoolReconciling
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *ExecutionPool) CompleteReconciliation(now time.Time) error {
	if c.State != ExecutionPoolReconciling {
		return TransitionError{Entity: "execution_pool", From: string(c.State), To: string(ExecutionPoolAvailable)}
	}
	c.State = ExecutionPoolAvailable
	c.LastReconciledAt = now.UTC()
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *ExecutionPool) Retire(now time.Time) error {
	if c.State != ExecutionPoolAvailable && c.State != ExecutionPoolReconciling {
		return ConflictError{Resource: "execution_pool", Reason: "active reservation must be completed before retirement"}
	}
	c.State = ExecutionPoolRetired
	c.ReservedRequestID = ""
	c.UpdatedAt = now.UTC()
	return nil
}
