package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/idempotency"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type ExecutionService struct{ dependencies }

type PlanExecutionRequestInput struct {
	WorkspaceID      string
	RequesterZoneID  string
	ExecutionZoneID  string
	ExecutionPoolID  string
	RequestKey       string
	ToolRevisionIDs  []string
	ScheduledStartAt time.Time
	ExpectedFinishAt time.Time
	IdempotencyKey   string
}

func (s *ExecutionService) PlanExecutionRequest(ctx context.Context, input PlanExecutionRequestInput) (domain.ExecutionRequest, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.ExecutionRequest{}, err
	}
	if len(input.ToolRevisionIDs) == 0 {
		return domain.ExecutionRequest{}, domain.FieldError{Field: "tool_revision_ids", Message: "at least one revision is required"}
	}
	if err := domain.ValidateIdempotencyKey(input.IdempotencyKey); err != nil {
		return domain.ExecutionRequest{}, err
	}
	hash, err := idempotency.Hash(input)
	if err != nil {
		return domain.ExecutionRequest{}, err
	}
	var run domain.ExecutionRequest
	err = s.store.WithTx(ctx, func(tx repository.Tx) error {
		if existing, err := tx.GetIdempotency(ctx, "submit-execution", input.IdempotencyKey); err == nil {
			if existing.RequestHash != hash {
				return domain.ConflictError{Resource: "idempotency_key", Reason: "request payload differs"}
			}
			return json.Unmarshal(existing.ResponseBody, &run)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		workspace, err := tx.GetWorkspace(ctx, input.WorkspaceID)
		if err != nil {
			return err
		}
		if !workspace.CanAcceptExecutionRequests() {
			return domain.ConflictError{Resource: "workspace", Reason: "workspace is not active"}
		}
		source, err := tx.GetTrustZone(ctx, input.RequesterZoneID)
		if err != nil {
			return err
		}
		target, err := tx.GetTrustZone(ctx, input.ExecutionZoneID)
		if err != nil {
			return err
		}
		if source.Status != domain.TrustZoneActive || target.Status != domain.TrustZoneActive {
			return domain.ConflictError{Resource: "trust_zone", Reason: "source and execution trust zones must be active"}
		}
		if err := domain.ValidateRoute(source, target); err != nil {
			return err
		}
		businessDay, err := source.BusinessDay(input.ScheduledStartAt)
		if err != nil {
			return err
		}
		count, err := tx.CountTrustZoneExecutionRequestsForBusinessDay(ctx, source.ID, businessDay)
		if err != nil {
			return err
		}
		if count >= source.DailyLimit {
			return domain.ConflictError{Resource: "trust_zone", Reason: "daily run limit reached"}
		}
		execution_pool, err := tx.GetExecutionPool(ctx, input.ExecutionPoolID)
		if err != nil {
			return err
		}
		if err := execution_pool.EligibleFor(input.ScheduledStartAt, 1); err != nil {
			return err
		}
		now := s.clock.Now()
		run = domain.ExecutionRequest{ID: identity.New("run"), WorkspaceID: input.WorkspaceID, RequesterZoneID: input.RequesterZoneID,
			ExecutionZoneID: input.ExecutionZoneID, ExecutionPoolID: input.ExecutionPoolID, RequestKey: strings.TrimSpace(input.RequestKey),
			State: domain.ExecutionRequestSubmitted, ScheduledStartAt: input.ScheduledStartAt.UTC(), ExpectedFinishAt: input.ExpectedFinishAt.UTC(), Version: 1, CreatedAt: now, UpdatedAt: now}
		volume := 0
		batches := make([]domain.ToolRevision, 0, len(input.ToolRevisionIDs))
		seen := make(map[string]struct{}, len(input.ToolRevisionIDs))
		for _, batchID := range input.ToolRevisionIDs {
			if _, exists := seen[batchID]; exists {
				return domain.ConflictError{Resource: "tool_revision", Reason: "duplicate revision in request"}
			}
			seen[batchID] = struct{}{}
			batch, err := tx.GetToolRevision(ctx, batchID)
			if err != nil {
				return err
			}
			if batch.WorkspaceID != workspace.ID || batch.RequesterZoneID != source.ID {
				return domain.ConflictError{Resource: "tool_revision", Reason: "revision belongs to another workspace or requester trust zone"}
			}
			if err := batch.Transition(domain.ToolRevisionReserved, now); err != nil {
				return err
			}
			volume += batch.RequestedUnits
			batches = append(batches, batch)
		}
		if err := (domain.ExecutionWindow{StartAt: input.ScheduledStartAt.UTC(), FinishAt: input.ExpectedFinishAt.UTC()}).Validate(workspace, batches, now); err != nil {
			return err
		}
		run.TotalRequestedUnits = volume
		if err := execution_pool.EligibleFor(input.ScheduledStartAt, volume); err != nil {
			return err
		}
		if err := run.Validate(); err != nil {
			return err
		}
		if err := tx.InsertExecutionRequest(ctx, run); err != nil {
			return err
		}
		for _, batch := range batches {
			batch.ExecutionRequestID = run.ID
			if err := tx.UpdateToolRevision(ctx, batch, batch.Version); err != nil {
				return err
			}
			if err := tx.InsertExecutionRequestInput(ctx, domain.ExecutionRequestInput{ExecutionRequestID: run.ID, ToolRevisionID: batch.ID, AddedAt: now}); err != nil {
				return err
			}
		}
		execution_pool.State = domain.ExecutionPoolReserved
		execution_pool.ReservedRequestID = run.ID
		execution_pool.UpdatedAt = now
		if err := tx.UpdateExecutionPool(ctx, execution_pool, execution_pool.Version); err != nil {
			return err
		}
		body, err := idempotency.Encode(run)
		if err != nil {
			return err
		}
		if err := tx.PutIdempotency(ctx, repository.IdempotencyRecord{Scope: "submit-execution", Key: input.IdempotencyKey, RequestHash: hash, ResponseCode: 201, ResponseBody: body, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now}); err != nil {
			return err
		}
		if err := tx.InsertJob(ctx, domain.OutboxJob{ID: identity.New("job"), Kind: "execution_request_submitted", AggregateID: run.ID, Payload: body, Status: domain.JobPending, MaxAttempts: 5, AvailableAt: now, CreatedAt: now, UpdatedAt: now}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "execution_request_submitted", "execution_request", run.ID, "success", map[string]string{"revision_count": fmt.Sprint(len(batches))})
	})
	return run, err
}

func (s *ExecutionService) AuthorizeExecutionRequest(ctx context.Context, requestID string) (domain.ExecutionRequest, error) {
	return s.transition(ctx, requestID, domain.ExecutionRequestAuthorized, domain.RoleAgentDeveloper, "execution_request_authorized")
}

func (s *ExecutionService) BeginExecutionRequest(ctx context.Context, requestID string) (domain.ExecutionRequest, error) {
	return s.transitionAny(ctx, requestID, domain.ExecutionRequestExecuting, []domain.Role{domain.RoleToolOperator, domain.RoleAgentDeveloper}, "execution_request_started")
}

func (s *ExecutionService) CompleteExecutionRequest(ctx context.Context, requestID string) (domain.ExecutionRequest, error) {
	return s.transitionAny(ctx, requestID, domain.ExecutionRequestCompleted, []domain.Role{domain.RoleToolOperator, domain.RoleAgentDeveloper}, "execution_request_completed")
}

func (s *ExecutionService) ArchiveExecutionRequest(ctx context.Context, requestID string) (domain.ExecutionRequest, error) {
	return s.transitionAny(ctx, requestID, domain.ExecutionRequestArchived, []domain.Role{domain.RoleAgentDeveloper}, "execution_request_archived")
}

func (s *ExecutionService) CancelExecutionRequest(ctx context.Context, requestID string, note string) (domain.ExecutionRequest, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.ExecutionRequest{}, err
	}
	var result domain.ExecutionRequest
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetExecutionRequest(ctx, requestID)
		if err != nil {
			return err
		}
		if run.State != domain.ExecutionRequestSubmitted && run.State != domain.ExecutionRequestAuthorized {
			return domain.TransitionError{Entity: "execution_request", From: string(run.State), To: string(domain.ExecutionRequestCancelled)}
		}
		now := s.clock.Now()
		items, err := tx.ListExecutionRequestInputs(ctx, run.ID)
		if err != nil {
			return err
		}
		if err := run.Transition(domain.ExecutionRequestCancelled, now); err != nil {
			return err
		}
		for _, batch := range items {
			if err := batch.Transition(domain.ToolRevisionVerified, now); err != nil {
				return err
			}
			batch.ExecutionRequestID = ""
			if err := tx.UpdateToolRevision(ctx, batch, batch.Version); err != nil {
				return err
			}
		}
		execution_pool, err := tx.GetExecutionPool(ctx, run.ExecutionPoolID)
		if err != nil {
			return err
		}
		execution_pool.State = domain.ExecutionPoolAvailable
		execution_pool.ReservedRequestID = ""
		execution_pool.UpdatedAt = now
		if err := tx.UpdateExecutionPool(ctx, execution_pool, execution_pool.Version); err != nil {
			return err
		}
		if err := tx.UpdateExecutionRequest(ctx, run, run.Version); err != nil {
			return err
		}
		result = run
		return s.audit.Record(ctx, tx, "execution_request_cancelled", "run", run.ID, "success", map[string]string{"note": strings.TrimSpace(note)})
	})
	return result, err
}

func (s *ExecutionService) transition(ctx context.Context, requestID string, target domain.ExecutionRequestState, role domain.Role, action string) (domain.ExecutionRequest, error) {
	return s.transitionAny(ctx, requestID, target, []domain.Role{role}, action)
}

func (s *ExecutionService) transitionAny(ctx context.Context, requestID string, target domain.ExecutionRequestState, roles []domain.Role, action string) (domain.ExecutionRequest, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, roles...); err != nil {
		return domain.ExecutionRequest{}, err
	}
	var result domain.ExecutionRequest
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetExecutionRequest(ctx, requestID)
		if err != nil {
			return err
		}
		if err := run.Transition(target, s.clock.Now()); err != nil {
			return err
		}
		now := s.clock.Now()
		items, err := tx.ListExecutionRequestInputs(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, batch := range items {
			switch target {
			case domain.ExecutionRequestExecuting:
				if err := batch.Transition(domain.ToolRevisionExecuting, now); err != nil {
					return err
				}
			case domain.ExecutionRequestCompleted:
				if batch.State != domain.ToolRevisionBlocked && batch.State != domain.ToolRevisionRejected && batch.State != domain.ToolRevisionApproved {
					if err := batch.Transition(domain.ToolRevisionExecuted, now); err != nil {
						return err
					}
				}
			case domain.ExecutionRequestArchived:
				if batch.State != domain.ToolRevisionApproved && batch.State != domain.ToolRevisionRejected && batch.State != domain.ToolRevisionExecuted {
					return domain.ConflictError{Resource: "tool_revision", Reason: "all revisions must be resolved before archiving"}
				}
			}
			if target == domain.ExecutionRequestExecuting || target == domain.ExecutionRequestCompleted {
				if err := tx.UpdateToolRevision(ctx, batch, batch.Version); err != nil {
					return err
				}
			}
		}
		execution_pool, err := tx.GetExecutionPool(ctx, run.ExecutionPoolID)
		if err != nil {
			return err
		}
		switch target {
		case domain.ExecutionRequestExecuting:
			execution_pool.State = domain.ExecutionPoolAllocated
		case domain.ExecutionRequestArchived, domain.ExecutionRequestCancelled:
			execution_pool.State = domain.ExecutionPoolAvailable
			execution_pool.ReservedRequestID = ""
		}
		execution_pool.UpdatedAt = now
		if err := tx.UpdateExecutionPool(ctx, execution_pool, execution_pool.Version); err != nil {
			return err
		}
		if err := tx.UpdateExecutionRequest(ctx, run, run.Version); err != nil {
			return err
		}
		result = run
		return s.audit.Record(ctx, tx, action, "execution_request", run.ID, "success", nil)
	})
	return result, err
}
