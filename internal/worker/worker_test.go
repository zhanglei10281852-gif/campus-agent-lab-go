package worker

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/clock"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/storage/sqlite"
)

func workerFixture(t *testing.T) (*Worker, *sqlite.Store, context.Context, time.Time) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	fixed := clock.NewFixed(now)
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	return New(store, fixed, time.Second, 20, logger), store, ctx, now
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestRunOnceExpiresApprovalTasksAndCompletesJobs(t *testing.T) {
	worker, store, ctx, now := workerFixture(t)
	minimum, _ := domain.RiskScoreFromFloat(2)
	maximum, _ := domain.RiskScoreFromFloat(8)
	rangeValue, _ := domain.NewRiskRange(minimum, maximum)
	workspace := domain.Workspace{ID: "workspace_worker", Code: "WORKER", Name: "Worker", Status: domain.WorkspaceActive, RiskScore: rangeValue, MaxExecution: time.Hour, ReviewDeadline: time.Hour, BusinessTimezone: "UTC", Version: 1, CreatedAt: now, UpdatedAt: now}
	origin := domain.TrustZone{ID: "origin_worker", Code: "ORIGIN", Name: "Origin", Timezone: "UTC", Status: domain.TrustZoneActive, DailyLimit: 10, CutoffHour: 0, Version: 1, CreatedAt: now, UpdatedAt: now}
	destination := domain.TrustZone{ID: "dest_worker", Code: "DEST", Name: "Destination", Timezone: "UTC", Status: domain.TrustZoneActive, DailyLimit: 10, CutoffHour: 0, Version: 1, CreatedAt: now, UpdatedAt: now}
	execution_pool := domain.ExecutionPool{ID: "box_worker", PoolKey: "BOX-W", State: domain.ExecutionPoolAllocated, CapacityUnits: 1000, AttestationDueAt: now.Add(time.Hour), LastReconciledAt: now, ReservedRequestID: "ship_worker", Version: 1, CreatedAt: now, UpdatedAt: now}
	run := domain.ExecutionRequest{ID: "ship_worker", WorkspaceID: workspace.ID, RequesterZoneID: origin.ID, ExecutionZoneID: destination.ID, ExecutionPoolID: execution_pool.ID, RequestKey: "SHIP-W", State: domain.ExecutionRequestExecuting, ScheduledStartAt: now, ExpectedFinishAt: now.Add(time.Hour), TotalRequestedUnits: 1, Version: 1, CreatedAt: now, UpdatedAt: now}
	approval_task := domain.ApprovalTask{ID: "approval_task_worker", ExecutionRequestID: run.ID, RequesterID: "from", ReviewerID: "to", ReviewQueue: "dock", Status: domain.ApprovalTaskPending, ExpiresAt: now.Add(-time.Minute), Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	job := domain.OutboxJob{ID: "job_worker", Kind: "execution_request_submitted", AggregateID: run.ID, Payload: []byte(`{"id":"ship_worker"}`), Status: domain.JobPending, MaxAttempts: 3, AvailableAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		for _, entity := range []any{workspace, origin, destination, execution_pool, run, approval_task, job} {
			switch value := entity.(type) {
			case domain.Workspace:
				if err := tx.InsertWorkspace(ctx, value); err != nil {
					return err
				}
			case domain.TrustZone:
				if err := tx.InsertTrustZone(ctx, value); err != nil {
					return err
				}
			case domain.ExecutionPool:
				if err := tx.InsertExecutionPool(ctx, value); err != nil {
					return err
				}
			case domain.ExecutionRequest:
				if err := tx.InsertExecutionRequest(ctx, value); err != nil {
					return err
				}
			case domain.ApprovalTask:
				if err := tx.InsertApprovalTask(ctx, value); err != nil {
					return err
				}
			case domain.OutboxJob:
				if err := tx.InsertJob(ctx, value); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Read(ctx, func(reader repository.Reader) error {
		approval_task, err := reader.GetApprovalTask(ctx, approval_task.ID)
		if err != nil {
			return err
		}
		if approval_task.Status != domain.ApprovalTaskExpired {
			t.Fatalf("approval_task = %+v", approval_task)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRunOnceHonorsCancellation(t *testing.T) {
	worker, _, _, _ := workerFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := worker.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v", err)
	}
}
