package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/clock"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/requestmeta"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/storage/sqlite"
)

type serviceFixture struct {
	t                 *testing.T
	ctx               context.Context
	store             *sqlite.Store
	services          *Services
	clock             *clock.Fixed
	agent_developer   domain.Principal
	tool_operator     domain.Principal
	security_reviewer domain.Principal
	workspace         domain.Workspace
	origin            domain.TrustZone
	destination       domain.TrustZone
	execution_pool    domain.ExecutionPool
	batch             domain.ToolRevision
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	fixed := clock.NewFixed(now)
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "service.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	services := New(store, fixed, 4*time.Hour, 30*time.Minute)
	users := []struct {
		email string
		name  string
		role  domain.Role
	}{
		{"ops@example.test", "Ops", domain.RoleAgentDeveloper},
		{"tool_operator@example.test", "Tool Operator", domain.RoleToolOperator},
		{"security_reviewer@example.test", "Reviewer", domain.RoleSecurityReviewer},
	}
	principals := make([]domain.Principal, 0, len(users))
	for _, user := range users {
		created, err := services.Auth.CreateUser(ctx, user.email, user.name, "very-secure-password", user.role)
		if err != nil {
			t.Fatalf("create user %s: %v", user.email, err)
		}
		login, err := services.Auth.Login(ctx, LoginInput{Email: user.email, Password: "very-secure-password"})
		if err != nil {
			t.Fatalf("login %s: %v", user.email, err)
		}
		if login.Principal.UserID != created.ID {
			t.Fatalf("principal user = %s, created = %s", login.Principal.UserID, created.ID)
		}
		principals = append(principals, login.Principal)
	}
	minimum, _ := domain.RiskScoreFromFloat(2)
	maximum, _ := domain.RiskScoreFromFloat(8)
	rangeValue, _ := domain.NewRiskRange(minimum, maximum)
	opsCtx := requestmeta.WithPrincipal(ctx, principals[0])
	workspace, err := services.Catalog.CreateWorkspace(opsCtx, domain.Workspace{Code: "STUDY-1", Name: "Cold workspace", RiskScore: rangeValue, MaxExecution: 24 * time.Hour, ReviewDeadline: 4 * time.Hour, BusinessTimezone: "Asia/Shanghai"})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err = services.Catalog.ActivateWorkspace(opsCtx, workspace.ID)
	if err != nil {
		t.Fatal(err)
	}
	origin, err := services.Catalog.CreateTrustZone(opsCtx, domain.TrustZone{Code: "SITE-1", Name: "Origin", Timezone: "Asia/Shanghai", DailyLimit: 10, CutoffHour: 6})
	if err != nil {
		t.Fatal(err)
	}
	destination, err := services.Catalog.CreateTrustZone(opsCtx, domain.TrustZone{Code: "SITE-2", Name: "Destination", Timezone: "Asia/Shanghai", DailyLimit: 10, CutoffHour: 6})
	if err != nil {
		t.Fatal(err)
	}
	now = fixed.Now()
	execution_pool, err := services.Catalog.CreateExecutionPool(opsCtx, domain.ExecutionPool{PoolKey: "BOX-1", CapacityUnits: 1000, AttestationDueAt: now.Add(48 * time.Hour), LastReconciledAt: now})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := services.Catalog.RegisterToolRevision(opsCtx, domain.ToolRevision{WorkspaceID: workspace.ID, RequesterZoneID: origin.ID, VersionTag: "EXT-1", ProtocolFamily: "plasma", OperationCount: 2, RequestedUnits: 100, ExpiresAt: now.Add(48 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = services.Catalog.VerifyToolRevision(opsCtx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &serviceFixture{t: t, ctx: ctx, store: store, services: services, clock: fixed, agent_developer: principals[0], tool_operator: principals[1], security_reviewer: principals[2], workspace: workspace, origin: origin, destination: destination, execution_pool: execution_pool, batch: batch}
}

func (f *serviceFixture) as(principal domain.Principal) context.Context {
	return requestmeta.WithPrincipal(requestmeta.WithRequestID(f.ctx, "req-test"), principal)
}

func TestAuthRejectsWrongPasswordAndHonorsLogout(t *testing.T) {
	f := newServiceFixture(t)
	if _, err := f.services.Auth.Login(f.ctx, LoginInput{Email: f.agent_developer.Email, Password: "wrong-password"}); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("wrong password error = %v", err)
	}
	if err := f.services.Auth.Logout(f.as(f.agent_developer), f.agent_developer); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Auth.Authenticate(f.ctx, "missing-token"); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("missing token error = %v, want unauthenticated", err)
	}
}

func TestExecutionIsIdempotentAndReservesRelatedEntities(t *testing.T) {
	f := newServiceFixture(t)
	input := PlanExecutionRequestInput{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, ExecutionZoneID: f.destination.ID, ExecutionPoolID: f.execution_pool.ID, RequestKey: "SHIP-1", ToolRevisionIDs: []string{f.batch.ID}, ScheduledStartAt: f.clock.Now().Add(time.Hour), ExpectedFinishAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "plan-key"}
	ctx := f.as(f.agent_developer)
	first, err := f.services.Execution.PlanExecutionRequest(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.services.Execution.PlanExecutionRequest(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.RequestKey != "SHIP-1" {
		t.Fatalf("idempotent responses differ: %+v / %+v", first, second)
	}
	if err := f.store.Read(ctx, func(reader repository.Reader) error {
		batch, err := reader.GetToolRevision(ctx, f.batch.ID)
		if err != nil {
			return err
		}
		if batch.State != domain.ToolRevisionReserved || batch.ExecutionRequestID != first.ID {
			t.Fatalf("reserved batch = %+v", batch)
		}
		execution_pool, err := reader.GetExecutionPool(ctx, f.execution_pool.ID)
		if err != nil {
			return err
		}
		if execution_pool.State != domain.ExecutionPoolReserved || execution_pool.ReservedRequestID != first.ID {
			t.Fatalf("reserved execution_pool = %+v", execution_pool)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionRejectsDifferentIdempotencyPayload(t *testing.T) {
	f := newServiceFixture(t)
	input := PlanExecutionRequestInput{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, ExecutionZoneID: f.destination.ID, ExecutionPoolID: f.execution_pool.ID, RequestKey: "SHIP-1", ToolRevisionIDs: []string{f.batch.ID}, ScheduledStartAt: f.clock.Now().Add(time.Hour), ExpectedFinishAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "plan-key"}
	ctx := f.as(f.agent_developer)
	if _, err := f.services.Execution.PlanExecutionRequest(ctx, input); err != nil {
		t.Fatal(err)
	}
	input.RequestKey = "SHIP-OTHER"
	if _, err := f.services.Execution.PlanExecutionRequest(ctx, input); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("different payload error = %v", err)
	}
}

func TestExecutionLifecycleMovesToolRevisionsAndExecutionPool(t *testing.T) {
	f := newServiceFixture(t)
	ctx := f.as(f.agent_developer)
	run, err := f.services.Execution.PlanExecutionRequest(ctx, PlanExecutionRequestInput{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, ExecutionZoneID: f.destination.ID, ExecutionPoolID: f.execution_pool.ID, RequestKey: "SHIP-LIFE", ToolRevisionIDs: []string{f.batch.ID}, ScheduledStartAt: f.clock.Now().Add(time.Hour), ExpectedFinishAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "life-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.AuthorizeExecutionRequest(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.BeginExecutionRequest(f.as(f.tool_operator), run.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Read(ctx, func(reader repository.Reader) error {
		batch, err := reader.GetToolRevision(ctx, f.batch.ID)
		if err != nil {
			return err
		}
		if batch.State != domain.ToolRevisionExecuting {
			t.Fatalf("in execution batch = %+v", batch)
		}
		execution_pool, err := reader.GetExecutionPool(ctx, f.execution_pool.ID)
		if err != nil {
			return err
		}
		if execution_pool.State != domain.ExecutionPoolAllocated {
			t.Fatalf("in execution execution_pool = %+v", execution_pool)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.CompleteExecutionRequest(f.as(f.tool_operator), run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.ArchiveExecutionRequest(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalOnlyReceiverCanResolve(t *testing.T) {
	f := newServiceFixture(t)
	ctx := f.as(f.agent_developer)
	run, err := f.services.Execution.PlanExecutionRequest(ctx, PlanExecutionRequestInput{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, ExecutionZoneID: f.destination.ID, ExecutionPoolID: f.execution_pool.ID, RequestKey: "SHIP-HAND", ToolRevisionIDs: []string{f.batch.ID}, ScheduledStartAt: f.clock.Now().Add(time.Hour), ExpectedFinishAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "hand-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.AuthorizeExecutionRequest(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.BeginExecutionRequest(f.as(f.tool_operator), run.ID); err != nil {
		t.Fatal(err)
	}
	approval_task, err := f.services.Approval.CreateApprovalTask(f.as(f.tool_operator), CreateApprovalTaskInput{ExecutionRequestID: run.ID, RequesterID: f.agent_developer.UserID, ReviewerID: f.tool_operator.UserID, ReviewQueue: "Dock 2"})
	if err != nil {
		t.Fatal(err)
	}
	other := domain.Principal{UserID: "compliance_auditor-user", Role: domain.RoleComplianceAuditor}
	if _, err := f.services.Approval.ResolveApprovalTask(f.as(other), approval_task.ID, true, "wrong actor"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("wrong actor error = %v", err)
	}
	if _, err := f.services.Approval.ResolveApprovalTask(f.as(f.tool_operator), approval_task.ID, true, "seal intact"); err != nil {
		t.Fatal(err)
	}
}

func (f *serviceFixture) planAndStart(t *testing.T, ref string) domain.ExecutionRequest {
	t.Helper()
	run, err := f.services.Execution.PlanExecutionRequest(f.as(f.agent_developer), PlanExecutionRequestInput{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, ExecutionZoneID: f.destination.ID, ExecutionPoolID: f.execution_pool.ID, RequestKey: ref, ToolRevisionIDs: []string{f.batch.ID}, ScheduledStartAt: f.clock.Now().Add(time.Hour), ExpectedFinishAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: ref + "-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.AuthorizeExecutionRequest(f.as(f.agent_developer), run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Execution.BeginExecutionRequest(f.as(f.tool_operator), run.ID); err != nil {
		t.Fatal(err)
	}
	return run
}

func TestRiskScorePolicyIncidentQuarantinesAndReviewerClears(t *testing.T) {
	f := newServiceFixture(t)
	run := f.planAndStart(t, "RUN-DRIFT")
	receipt, policy_incident, err := f.services.Receipts.RecordReceipt(f.as(f.tool_operator), RecordReceiptInput{ExecutionRequestID: run.ID, SignalKey: "sensor-1", Sequence: 1, RiskScore: 12000, RecordedAt: f.clock.Now().Add(10 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ID == "" || policy_incident == nil || policy_incident.ReceiptCount != 1 {
		t.Fatalf("receipt=%+v policy_incident=%+v", receipt, policy_incident)
	}
	if _, err := f.services.Execution.CompleteExecutionRequest(f.as(f.tool_operator), run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Review.StartReview(f.as(f.security_reviewer), policy_incident.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Review.Decide(f.as(f.security_reviewer), DecideInput{PolicyIncidentID: policy_incident.ID, Decision: domain.PolicyIncidentCleared, Rationale: "verified logger trace"}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Read(f.ctx, func(reader repository.Reader) error {
		batch, err := reader.GetToolRevision(f.ctx, f.batch.ID)
		if err != nil {
			return err
		}
		if batch.State != domain.ToolRevisionApproved {
			t.Fatalf("batch after clear = %+v", batch)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInRangeReceiptDoesNotOpenPolicyIncident(t *testing.T) {
	f := newServiceFixture(t)
	run := f.planAndStart(t, "RUN-IN-RANGE")
	_, policy_incident, err := f.services.Receipts.RecordReceipt(f.as(f.tool_operator), RecordReceiptInput{ExecutionRequestID: run.ID, SignalKey: "sensor-1", Sequence: 1, RiskScore: 5000, RecordedAt: f.clock.Now()})
	if err != nil || policy_incident != nil {
		t.Fatalf("in range result policy_incident=%+v error=%v", policy_incident, err)
	}
}

func TestQueryReadinessReportsBlockers(t *testing.T) {
	f := newServiceFixture(t)
	run := f.planAndStart(t, "RUN-REPORT")
	report, err := f.services.Query.ReconcileExecutionRequest(f.as(f.agent_developer), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.ExpectedToolRevisionCount != 1 || report.Complete {
		t.Fatalf("report = %+v", report)
	}
}

func TestContextCancellationReachesTransaction(t *testing.T) {
	f := newServiceFixture(t)
	cancelled, cancel := context.WithCancel(f.as(f.agent_developer))
	cancel()
	_, err := f.services.Catalog.VerifyToolRevision(cancelled, f.batch.ID)
	if err == nil {
		t.Fatal("cancelled command succeeded")
	}
}

func TestExecutionPoolReconcilingAndRetirementLifecycle(t *testing.T) {
	f := newServiceFixture(t)
	ctx := f.as(f.agent_developer)
	reconciliation, err := f.services.ExecutionPools.StartReconciliation(ctx, f.execution_pool.ID)
	if err != nil || reconciliation.State != domain.ExecutionPoolReconciling {
		t.Fatalf("start reconciliation = %+v, error=%v", reconciliation, err)
	}
	f.clock.Advance(time.Hour)
	available, err := f.services.ExecutionPools.CompleteReconciliation(ctx, f.execution_pool.ID)
	if err != nil || available.State != domain.ExecutionPoolAvailable || !available.LastReconciledAt.Equal(f.clock.Now()) {
		t.Fatalf("complete reconciliation = %+v, error=%v", available, err)
	}
	retired, err := f.services.ExecutionPools.Retire(ctx, f.execution_pool.ID, "attestation program ended")
	if err != nil || retired.State != domain.ExecutionPoolRetired {
		t.Fatalf("retire = %+v, error=%v", retired, err)
	}
	if _, err := f.services.ExecutionPools.StartReconciliation(ctx, f.execution_pool.ID); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("clean retired error = %v", err)
	}
}

func TestBulkRegistrationReturnsPartialFailures(t *testing.T) {
	f := newServiceFixture(t)
	now := f.clock.Now()
	inputs := []domain.ToolRevision{
		{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, VersionTag: "BULK-OK", ProtocolFamily: "serum", OperationCount: 1, RequestedUnits: 20, ExpiresAt: now.Add(time.Hour)},
		{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, VersionTag: "", ProtocolFamily: "serum", OperationCount: 1, RequestedUnits: 20, ExpiresAt: now.Add(time.Hour)},
		{WorkspaceID: f.workspace.ID, RequesterZoneID: f.origin.ID, VersionTag: "BULK-OK", ProtocolFamily: "serum", OperationCount: 1, RequestedUnits: 20, ExpiresAt: now.Add(time.Hour)},
	}
	result, err := f.services.Catalog.BulkRegisterToolRevisions(f.as(f.agent_developer), inputs)
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Failed != 2 || len(result.Items) != 3 {
		t.Fatalf("bulk result = %+v", result)
	}
	if result.Items[0].Code != "created" || result.Items[1].Code != "invalid" || result.Items[2].Code != "conflict" {
		t.Fatalf("bulk item codes = %+v", result.Items)
	}
}

func TestPlatformSummaryRequiresReadPermissionAndCountsRows(t *testing.T) {
	f := newServiceFixture(t)
	if _, err := f.services.Query.PlatformSummary(f.as(f.security_reviewer)); err != nil {
		t.Fatalf("security_reviewer summary: %v", err)
	}
	summary, err := f.services.Query.PlatformSummary(f.as(f.agent_developer))
	if err != nil {
		t.Fatal(err)
	}
	if summary.WorkspacesActive != 1 || summary.ToolRevisionsValidated != 1 || summary.ExecutionPoolsAvailable != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := f.services.Query.PlatformSummary(f.as(domain.Principal{UserID: "tool_operator", Role: domain.RoleToolOperator})); err != nil {
		t.Fatalf("tool_operator read summary: %v", err)
	}
}
