package domain

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestRiskRangeBoundaries(t *testing.T) {
	min := MilliRiskScore(2000)
	max := MilliRiskScore(8000)
	rangeValue, err := NewRiskRange(min, max)
	if err != nil {
		t.Fatalf("create range: %v", err)
	}
	tests := []struct {
		name  string
		value MilliRiskScore
		want  bool
	}{
		{name: "minimum included", value: min, want: true},
		{name: "middle included", value: 5000, want: true},
		{name: "maximum included", value: max, want: true},
		{name: "below minimum", value: 1999, want: false},
		{name: "above maximum", value: 8001, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rangeValue.Contains(test.value); got != test.want {
				t.Fatalf("Contains(%d) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func TestRiskScoreParsingRejectsInvalidValues(t *testing.T) {
	for _, value := range []float64{-197, 101, math.NaN()} {
		_, err := RiskScoreFromFloat(value)
		if err == nil {
			t.Fatalf("RiskScoreFromFloat(%v) succeeded", value)
		}
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("error %v does not wrap validation", err)
		}
	}
}

func TestToolRevisionTransitionTable(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	base := ToolRevision{State: ToolRevisionRegistered, ExpiresAt: now.Add(24 * time.Hour)}
	cases := []struct {
		name string
		from ToolRevisionState
		to   ToolRevisionState
		want bool
	}{
		{"registered to verified", ToolRevisionRegistered, ToolRevisionVerified, true},
		{"verified to reserved", ToolRevisionVerified, ToolRevisionReserved, true},
		{"reserved to executing", ToolRevisionReserved, ToolRevisionExecuting, true},
		{"executing to executed", ToolRevisionExecuting, ToolRevisionExecuted, true},
		{"executed to approved", ToolRevisionExecuted, ToolRevisionApproved, true},
		{"executed to quarantine", ToolRevisionExecuted, ToolRevisionBlocked, true},
		{"quarantine to rejected", ToolRevisionBlocked, ToolRevisionRejected, true},
		{"registered to approved", ToolRevisionRegistered, ToolRevisionApproved, false},
		{"approved to verified", ToolRevisionApproved, ToolRevisionVerified, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			batch := base
			batch.State = test.from
			err := batch.Transition(test.to, now)
			if (err == nil) != test.want {
				t.Fatalf("transition %s -> %s error = %v, want allowed=%v", test.from, test.to, err, test.want)
			}
			if test.want && batch.State != test.to {
				t.Fatalf("state = %s, want %s", batch.State, test.to)
			}
		})
	}
}

func TestExpiredToolRevisionCanOnlyBeDestroyedOrQuarantined(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	batch := ToolRevision{State: ToolRevisionExecuted, ExpiresAt: now.Add(-time.Minute)}
	if err := batch.Transition(ToolRevisionApproved, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("expired release error = %v, want conflict", err)
	}
	batch.State = ToolRevisionExecuted
	if err := batch.Transition(ToolRevisionBlocked, now); err != nil {
		t.Fatalf("expired quarantine failed: %v", err)
	}
}

func TestExecutionRequestTransitionSetsTimestamps(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	run := ExecutionRequest{State: ExecutionRequestSubmitted, ScheduledStartAt: now, ExpectedFinishAt: now.Add(2 * time.Hour)}
	if err := run.Transition(ExecutionRequestAuthorized, now); err != nil {
		t.Fatalf("pack: %v", err)
	}
	if err := run.Transition(ExecutionRequestExecuting, now.Add(time.Minute)); err != nil {
		t.Fatalf("start: %v", err)
	}
	if run.StartedAt == nil || !run.StartedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("started_at = %v", run.StartedAt)
	}
	if err := run.Transition(ExecutionRequestCompleted, now.Add(time.Hour)); err != nil {
		t.Fatalf("arrive: %v", err)
	}
	if err := run.Transition(ExecutionRequestArchived, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("close: %v", err)
	}
	if run.ArchivedAt == nil {
		t.Fatal("archived_at is nil")
	}
}

func TestExecutionRequestRejectsSkippedState(t *testing.T) {
	run := ExecutionRequest{State: ExecutionRequestSubmitted}
	err := run.Transition(ExecutionRequestCompleted, time.Now())
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("error = %v, want invalid transition", err)
	}
}

func TestApprovalTaskResolutionAndExpiry(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	approval_task := ApprovalTask{Status: ApprovalTaskPending, ExpiresAt: now.Add(time.Hour)}
	if err := approval_task.Resolve(ApprovalTaskAccepted, "seal intact", now); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if approval_task.Status != ApprovalTaskAccepted || approval_task.ResolvedAt == nil {
		t.Fatalf("approval_task after accept = %+v", approval_task)
	}
	approval_task = ApprovalTask{Status: ApprovalTaskPending, ExpiresAt: now.Add(-time.Minute)}
	if err := approval_task.Resolve(ApprovalTaskAccepted, "", now); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired accept error = %v", err)
	}
	if err := approval_task.Resolve(ApprovalTaskExpired, "expired", now); err != nil {
		t.Fatalf("expire: %v", err)
	}
}

func TestPolicyIncidentAggregatesReceipts(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	policy_incident := PolicyIncident{Status: PolicyIncidentOpen}
	receipts := []ExecutionReceipt{
		{RiskScore: 9200, RecordedAt: now.Add(10 * time.Minute)},
		{RiskScore: 8500, RecordedAt: now.Add(5 * time.Minute)},
		{RiskScore: 11000, RecordedAt: now.Add(20 * time.Minute)},
	}
	for _, receipt := range receipts {
		policy_incident.Include(receipt, now)
	}
	if policy_incident.ReceiptCount != 3 || policy_incident.Minimum != 8500 || policy_incident.Maximum != 11000 {
		t.Fatalf("aggregate = %+v", policy_incident)
	}
	if !policy_incident.FirstReceiptAt.Equal(now.Add(5*time.Minute)) || !policy_incident.LastReceiptAt.Equal(now.Add(20*time.Minute)) {
		t.Fatalf("receipt window = %v..%v", policy_incident.FirstReceiptAt, policy_incident.LastReceiptAt)
	}
}

func TestPolicyIncidentDecisionTable(t *testing.T) {
	now := time.Now().UTC()
	for _, decision := range []PolicyIncidentStatus{PolicyIncidentCleared, PolicyIncidentRejected} {
		policy_incident := PolicyIncident{Status: PolicyIncidentReviewing}
		if err := policy_incident.Decide(decision, now); err != nil {
			t.Fatalf("decision %s: %v", decision, err)
		}
		if policy_incident.Status != decision {
			t.Fatalf("status = %s, want %s", policy_incident.Status, decision)
		}
	}
	policy_incident := PolicyIncident{Status: PolicyIncidentCleared}
	if err := policy_incident.Decide(PolicyIncidentRejected, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("decide closed policy_incident = %v", err)
	}
}

func TestTrustZoneBusinessDayUsesCutoffAndTimezone(t *testing.T) {
	trust_zone := TrustZone{Timezone: "Asia/Shanghai", CutoffHour: 6}
	before, err := trust_zone.BusinessDay(time.Date(2026, 8, 18, 21, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if before != "2026-08-18" {
		t.Fatalf("business day = %s", before)
	}
	after, err := trust_zone.BusinessDay(time.Date(2026, 8, 18, 22, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if after != "2026-08-19" {
		t.Fatalf("business day after cutoff = %s", after)
	}
}

func TestExecutionPoolEligibility(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	base := ExecutionPool{State: ExecutionPoolAvailable, CapacityUnits: 1000, AttestationDueAt: now.Add(time.Hour)}
	if err := base.EligibleFor(now, 1000); err != nil {
		t.Fatalf("capacity boundary: %v", err)
	}
	if err := base.EligibleFor(now, 1001); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("capacity overflow = %v", err)
	}
	base.State = ExecutionPoolReserved
	if err := base.EligibleFor(now, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("reserved execution_pool = %v", err)
	}
}

func TestReadinessEvaluate(t *testing.T) {
	report := RunReadiness{ExecutionRequestState: ExecutionRequestCompleted, ExpectedToolRevisionCount: 2, MaterializedToolRevisionCount: 2, PendingApprovalTask: true}
	report.Evaluate()
	if report.Complete || len(report.Blockers) != 1 || report.Blockers[0] != "pending approval task" {
		t.Fatalf("report = %+v", report)
	}
	report.PendingApprovalTask = false
	report.Evaluate()
	if !report.Complete {
		t.Fatalf("resolved report = %+v", report)
	}
}

func TestAuditAndJobCloneIsolation(t *testing.T) {
	event := AuditEvent{Metadata: map[string]string{"one": "1"}}
	clone := event.Clone()
	clone.Metadata["one"] = "2"
	if event.Metadata["one"] != "1" {
		t.Fatal("audit metadata was shared")
	}
	job := OutboxJob{Payload: []byte("payload")}
	jobClone := job.Clone()
	jobClone.Payload[0] = 'P'
	if string(job.Payload) != "payload" {
		t.Fatal("job payload was shared")
	}
}

func TestExecutionWindowChecksWorkspaceLimitAndExpiry(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	workspace := Workspace{MaxExecution: 2 * time.Hour}
	batch := ToolRevision{ExpiresAt: now.Add(4 * time.Hour)}
	valid := ExecutionWindow{StartAt: now.Add(time.Hour), FinishAt: now.Add(2 * time.Hour)}
	if err := valid.Validate(workspace, []ToolRevision{batch}, now); err != nil {
		t.Fatalf("valid window: %v", err)
	}
	tooLong := ExecutionWindow{StartAt: now.Add(time.Hour), FinishAt: now.Add(4 * time.Hour)}
	if err := tooLong.Validate(workspace, []ToolRevision{batch}, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("long window = %v", err)
	}
	batch.ExpiresAt = now.Add(90 * time.Minute)
	if err := valid.Validate(workspace, []ToolRevision{batch}, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("expired batch window = %v", err)
	}
}

func TestPrincipalActionMatrix(t *testing.T) {
	cases := []struct {
		role   Role
		action Action
		want   bool
	}{
		{RoleAgentDeveloper, ActionSubmitExecution, true},
		{RoleAgentDeveloper, ActionReviewPolicyIncident, false},
		{RoleToolOperator, ActionRecordReceipt, true},
		{RoleToolOperator, ActionCatalogWrite, false},
		{RoleSecurityReviewer, ActionReviewPolicyIncident, true},
		{RoleComplianceAuditor, ActionReadAudit, true},
		{RoleComplianceAuditor, ActionManageExecution, false},
	}
	for _, test := range cases {
		principal := Principal{Role: test.role}
		if got := principal.CanAction(test.action); got != test.want {
			t.Fatalf("%s %s = %v, want %v", test.role, test.action, got, test.want)
		}
	}
}

func TestIdentifierNormalizationAndValidation(t *testing.T) {
	if got := NormalizeCode("  data-zone-sh-01 "); got != "DATA-ZONE-SH-01" {
		t.Fatalf("normalized code = %q", got)
	}
	for _, value := range []string{"A", "with spaces", "ümlaut", "", strings.Repeat("X", 65)} {
		if err := ValidateBusinessCode("code", value); err == nil {
			t.Fatalf("invalid code %q passed", value)
		}
	}
	for _, value := range []string{"valid-key", "request-1234", strings.Repeat("x", 128)} {
		if err := ValidateIdempotencyKey(value); err != nil {
			t.Fatalf("valid idempotency key %q: %v", value, err)
		}
	}
	for _, value := range []string{"short", "line\nbreak", strings.Repeat("x", 129)} {
		if err := ValidateIdempotencyKey(value); err == nil {
			t.Fatalf("invalid idempotency key %q passed", value)
		}
	}
}

func TestTerminalStateHelpers(t *testing.T) {
	if !ExecutionRequestArchived.IsTerminal() || !ExecutionRequestCancelled.IsTerminal() || ExecutionRequestCompleted.IsTerminal() {
		t.Fatal("run terminal states are incorrect")
	}
	if !ToolRevisionApproved.IsTerminal() || !ToolRevisionRejected.IsTerminal() || ToolRevisionBlocked.IsTerminal() {
		t.Fatal("revision terminal states are incorrect")
	}
	if !PolicyIncidentCleared.IsResolved() || !PolicyIncidentRejected.IsResolved() || PolicyIncidentOpen.IsResolved() {
		t.Fatal("policy_incident resolved states are incorrect")
	}
}
