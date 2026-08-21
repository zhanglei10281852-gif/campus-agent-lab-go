package service

import (
	"context"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type ApprovalService struct {
	dependencies
	approval_taskTTL time.Duration
}

type CreateApprovalTaskInput struct {
	ExecutionRequestID string
	RequesterID        string
	ReviewerID         string
	ReviewQueue        string
}

func (s *ApprovalService) CreateApprovalTask(ctx context.Context, input CreateApprovalTaskInput) (domain.ApprovalTask, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleToolOperator, domain.RoleAgentDeveloper); err != nil {
		return domain.ApprovalTask{}, err
	}
	if strings.TrimSpace(input.RequesterID) == "" {
		input.RequesterID = principal.UserID
	}
	var result domain.ApprovalTask
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetExecutionRequest(ctx, input.ExecutionRequestID)
		if err != nil {
			return err
		}
		if run.State != domain.ExecutionRequestExecuting && run.State != domain.ExecutionRequestCompleted {
			return domain.ConflictError{Resource: "run", Reason: "approval_task requires an active run"}
		}
		if _, err := tx.GetPendingApprovalTask(ctx, run.ID); err == nil {
			return domain.ConflictError{Resource: "approval_task", Reason: "run already has a pending approval_task"}
		} else if !isNotFound(err) {
			return err
		}
		now := s.clock.Now()
		approval_task := domain.ApprovalTask{ID: identity.New("approval_task"), ExecutionRequestID: run.ID, RequesterID: strings.TrimSpace(input.RequesterID), ReviewerID: strings.TrimSpace(input.ReviewerID), ReviewQueue: strings.TrimSpace(input.ReviewQueue), Status: domain.ApprovalTaskPending, ExpiresAt: now.Add(s.approval_taskTTL), Version: 1, CreatedAt: now, UpdatedAt: now}
		if err := approval_task.Validate(); err != nil {
			return err
		}
		if err := tx.InsertApprovalTask(ctx, approval_task); err != nil {
			return err
		}
		result = approval_task
		return s.audit.Record(ctx, tx, "approval_task_created", "approval_task", approval_task.ID, "success", nil)
	})
	return result, err
}

func (s *ApprovalService) ResolveApprovalTask(ctx context.Context, approval_taskID string, accepted bool, note string) (domain.ApprovalTask, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleToolOperator, domain.RoleAgentDeveloper); err != nil {
		return domain.ApprovalTask{}, err
	}
	var result domain.ApprovalTask
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		approval_task, err := tx.GetApprovalTask(ctx, approval_taskID)
		if err != nil {
			return err
		}
		if approval_task.ReviewerID != principal.UserID && !principal.Can(domain.RoleAgentDeveloper) {
			return domain.ConflictError{Resource: "approval_task", Reason: "only the receiving custodian may resolve it"}
		}
		status := domain.ApprovalTaskRejected
		if accepted {
			status = domain.ApprovalTaskAccepted
		}
		if err := approval_task.Resolve(status, note, s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdateApprovalTask(ctx, approval_task, approval_task.Version); err != nil {
			return err
		}
		result = approval_task
		return s.audit.Record(ctx, tx, "approval_task_resolved", "approval_task", approval_task.ID, "success", map[string]string{"status": string(status)})
	})
	return result, err
}
