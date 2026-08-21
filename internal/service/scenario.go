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

type ScenarioService struct{ dependencies }

type CreateScenarioInput struct {
	Name            string
	ProtocolFamily  string
	OperationCount  int
	RequestedUnits  int
	DurationMinutes int
	IdempotencyKey  string
}

type ScenarioResult struct {
	Workspace    domain.Workspace        `json:"workspace"`
	ToolRevision domain.ToolRevision     `json:"tool_revision"`
	Request      domain.ExecutionRequest `json:"request"`
}

func (s *ScenarioService) Create(ctx context.Context, input CreateScenarioInput) (ScenarioResult, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return ScenarioResult{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	input.ProtocolFamily = strings.TrimSpace(input.ProtocolFamily)
	if input.Name == "" || len(input.Name) > 120 {
		return ScenarioResult{}, domain.FieldError{Field: "name", Message: "must be between 1 and 120 characters"}
	}
	if input.ProtocolFamily == "" || len(input.ProtocolFamily) > 64 {
		return ScenarioResult{}, domain.FieldError{Field: "protocol_family", Message: "must be between 1 and 64 characters"}
	}
	if input.OperationCount < 1 || input.OperationCount > 5000 {
		return ScenarioResult{}, domain.FieldError{Field: "operation_count", Message: "must be between 1 and 5000"}
	}
	if input.RequestedUnits < 1 || input.RequestedUnits > 100000 {
		return ScenarioResult{}, domain.FieldError{Field: "requested_units", Message: "must be between 1 and 100000"}
	}
	if input.DurationMinutes < 15 || input.DurationMinutes > 480 {
		return ScenarioResult{}, domain.FieldError{Field: "duration_minutes", Message: "must be between 15 and 480"}
	}
	if err := domain.ValidateIdempotencyKey(input.IdempotencyKey); err != nil {
		return ScenarioResult{}, err
	}

	payload := struct {
		Name            string
		ProtocolFamily  string
		OperationCount  int
		RequestedUnits  int
		DurationMinutes int
	}{input.Name, input.ProtocolFamily, input.OperationCount, input.RequestedUnits, input.DurationMinutes}
	hash, err := idempotency.Hash(payload)
	if err != nil {
		return ScenarioResult{}, err
	}

	var result ScenarioResult
	err = s.store.WithTx(ctx, func(tx repository.Tx) error {
		if existing, err := tx.GetIdempotency(ctx, "create-training-scenario", input.IdempotencyKey); err == nil {
			if existing.RequestHash != hash {
				return domain.ConflictError{Resource: "idempotency_key", Reason: "request payload differs"}
			}
			return json.Unmarshal(existing.ResponseBody, &result)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		now := s.clock.Now().UTC()
		startAt := now.Add(15 * time.Minute)
		finishAt := startAt.Add(time.Duration(input.DurationMinutes) * time.Minute)
		suffix := scenarioSuffix(identity.New("scenario"))
		workspace := domain.Workspace{
			ID: identity.New("workspace"), Code: "LAB-" + suffix, Name: input.Name, Status: domain.WorkspaceActive,
			RiskScore: domain.RiskRange{Minimum: 800, Maximum: 990}, MaxExecution: 8 * time.Hour,
			ReviewDeadline: 4 * time.Hour, BusinessTimezone: "Asia/Shanghai", Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		requesterZone := domain.TrustZone{ID: identity.New("trust_zone"), Code: "DEV-" + suffix, Name: input.Name + " 开发区", Timezone: "Asia/Shanghai", Status: domain.TrustZoneActive, DailyLimit: 100, CutoffHour: 4, Version: 1, CreatedAt: now, UpdatedAt: now}
		executionZone := domain.TrustZone{ID: identity.New("trust_zone"), Code: "RUN-" + suffix, Name: input.Name + " 执行区", Timezone: "Asia/Shanghai", Status: domain.TrustZoneActive, DailyLimit: 100, CutoffHour: 4, Version: 1, CreatedAt: now, UpdatedAt: now}
		run := domain.ExecutionRequest{
			ID: identity.New("run"), WorkspaceID: workspace.ID, RequesterZoneID: requesterZone.ID, ExecutionZoneID: executionZone.ID,
			RequestKey: "RUN-" + suffix, State: domain.ExecutionRequestSubmitted, ScheduledStartAt: startAt,
			ExpectedFinishAt: finishAt, TotalRequestedUnits: input.RequestedUnits, Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		pool := domain.ExecutionPool{
			ID: identity.New("pool"), PoolKey: "POOL-" + suffix, State: domain.ExecutionPoolReserved,
			CapacityUnits: max(100, input.RequestedUnits), AttestationDueAt: now.Add(30 * 24 * time.Hour),
			LastReconciledAt: now, ReservedRequestID: run.ID, Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		run.ExecutionPoolID = pool.ID
		revision := domain.ToolRevision{
			ID: identity.New("revision"), WorkspaceID: workspace.ID, RequesterZoneID: requesterZone.ID,
			VersionTag: "v1-" + strings.ToLower(suffix), ProtocolFamily: input.ProtocolFamily,
			OperationCount: input.OperationCount, RequestedUnits: input.RequestedUnits, State: domain.ToolRevisionReserved,
			ExpiresAt: now.Add(7 * 24 * time.Hour), ExecutionRequestID: run.ID, Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		if err := domain.ValidateRoute(requesterZone, executionZone); err != nil {
			return err
		}
		if err := (domain.ExecutionWindow{StartAt: startAt, FinishAt: finishAt}).Validate(workspace, []domain.ToolRevision{revision}, now); err != nil {
			return err
		}
		for _, value := range []struct {
			name string
			fn   func() error
		}{
			{"workspace", func() error { return tx.InsertWorkspace(ctx, workspace) }},
			{"requester trust zone", func() error { return tx.InsertTrustZone(ctx, requesterZone) }},
			{"execution trust zone", func() error { return tx.InsertTrustZone(ctx, executionZone) }},
			{"execution pool", func() error { return tx.InsertExecutionPool(ctx, pool) }},
			{"tool revision", func() error { return tx.InsertToolRevision(ctx, revision) }},
			{"execution request", func() error { return tx.InsertExecutionRequest(ctx, run) }},
		} {
			if err := value.fn(); err != nil {
				return fmt.Errorf("create scenario %s: %w", value.name, err)
			}
		}
		if err := tx.InsertExecutionRequestInput(ctx, domain.ExecutionRequestInput{ExecutionRequestID: run.ID, ToolRevisionID: revision.ID, AddedAt: now}); err != nil {
			return fmt.Errorf("link scenario revision: %w", err)
		}
		result = ScenarioResult{Workspace: workspace, ToolRevision: revision, Request: run}
		body, err := idempotency.Encode(result)
		if err != nil {
			return err
		}
		if err := tx.PutIdempotency(ctx, repository.IdempotencyRecord{Scope: "create-training-scenario", Key: input.IdempotencyKey, RequestHash: hash, ResponseCode: 201, ResponseBody: body, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now}); err != nil {
			return err
		}
		if err := tx.InsertJob(ctx, domain.OutboxJob{ID: identity.New("job"), Kind: "execution_request_submitted", AggregateID: run.ID, Payload: body, Status: domain.JobPending, MaxAttempts: 5, AvailableAt: now, CreatedAt: now, UpdatedAt: now}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "training_scenario_created", "execution_request", run.ID, "success", map[string]string{"workspace_id": workspace.ID, "revision_id": revision.ID})
	})
	return result, err
}

func scenarioSuffix(id string) string {
	if index := strings.LastIndexByte(id, '_'); index >= 0 {
		id = id[index+1:]
	}
	if len(id) > 12 {
		id = id[:12]
	}
	return strings.ToUpper(id)
}
