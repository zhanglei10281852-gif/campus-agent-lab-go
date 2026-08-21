package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type ReceiptService struct{ dependencies }

type RecordReceiptInput struct {
	ExecutionRequestID string
	SignalKey          string
	Sequence           int64
	RiskScore          domain.MilliRiskScore
	RecordedAt         time.Time
}

func (s *ReceiptService) RecordReceipt(ctx context.Context, input RecordReceiptInput) (domain.ExecutionReceipt, *domain.PolicyIncident, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleToolOperator, domain.RoleAgentDeveloper); err != nil {
		return domain.ExecutionReceipt{}, nil, err
	}
	now := s.clock.Now()
	receipt := domain.ExecutionReceipt{ID: identity.New("obs"), ExecutionRequestID: input.ExecutionRequestID, SignalKey: input.SignalKey, Sequence: input.Sequence, RiskScore: input.RiskScore, RecordedAt: input.RecordedAt.UTC(), ReceivedAt: now}
	if err := receipt.Validate(); err != nil {
		return domain.ExecutionReceipt{}, nil, err
	}
	var policy_incident *domain.PolicyIncident
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetExecutionRequest(ctx, input.ExecutionRequestID)
		if err != nil {
			return err
		}
		workspace, err := tx.GetWorkspace(ctx, run.WorkspaceID)
		if err != nil {
			return err
		}
		if run.State != domain.ExecutionRequestExecuting && run.State != domain.ExecutionRequestCompleted {
			return domain.ConflictError{Resource: "execution_request", Reason: "execution receipts require active execution"}
		}
		if err := tx.InsertReceipt(ctx, receipt); err != nil {
			return err
		}
		if workspace.RiskScore.Contains(receipt.RiskScore) {
			return s.audit.Record(ctx, tx, "execution_receipt_recorded", "execution_request", run.ID, "success", map[string]string{"in_range": "true"})
		}
		active, err := tx.GetActivePolicyIncident(ctx, run.ID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if errors.Is(err, domain.ErrNotFound) {
			active = domain.PolicyIncident{ID: identity.New("drift"), ExecutionRequestID: run.ID, Status: domain.PolicyIncidentOpen, ReviewDueAt: now.Add(workspace.ReviewDeadline), Version: 1, CreatedAt: now, UpdatedAt: now}
			active.Include(receipt, now)
			if err := tx.InsertPolicyIncident(ctx, active); err != nil {
				return err
			}
		} else {
			before := active.Version
			active.Include(receipt, now)
			if err := tx.UpdatePolicyIncident(ctx, active, before); err != nil {
				return err
			}
		}
		items, err := tx.ListExecutionRequestInputs(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, batch := range items {
			if batch.State == domain.ToolRevisionExecuting || batch.State == domain.ToolRevisionExecuted {
				batch.State = domain.ToolRevisionBlocked
				batch.QuarantineNote = fmt.Sprintf("quality policy incident %s", active.ID)
				batch.UpdatedAt = now
				if err := tx.UpdateToolRevision(ctx, batch, batch.Version); err != nil {
					return err
				}
			}
		}
		payload := []byte(active.ID)
		if err := tx.InsertJob(ctx, domain.OutboxJob{ID: identity.New("job"), Kind: "policy_incident_review", AggregateID: active.ID, Payload: payload, Status: domain.JobPending, MaxAttempts: 5, AvailableAt: now, CreatedAt: now, UpdatedAt: now}); err != nil {
			return err
		}
		policy_incident = &active
		return s.audit.Record(ctx, tx, "execution_policy_incident_opened", "policy_incident", active.ID, "success", map[string]string{"request_id": run.ID})
	})
	return receipt, policy_incident, err
}
