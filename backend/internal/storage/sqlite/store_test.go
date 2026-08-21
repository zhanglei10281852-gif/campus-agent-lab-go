package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

func testStore(t *testing.T) (*Store, context.Context, time.Time) {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "campuslab.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	return store, ctx, now
}

func seedCatalog(t *testing.T, store *Store, ctx context.Context, now time.Time) (domain.Workspace, domain.TrustZone, domain.TrustZone, domain.ExecutionPool, domain.ToolRevision) {
	t.Helper()
	minimum, _ := domain.RiskScoreFromFloat(0.8)
	maximum, _ := domain.RiskScoreFromFloat(0.99)
	rangeValue, _ := domain.NewRiskRange(minimum, maximum)
	workspace := domain.Workspace{ID: "workspace_1", Code: "MESH-1", Name: "Ranking workspace", Status: domain.WorkspaceActive, RiskScore: rangeValue, MaxExecution: 24 * time.Hour, ReviewDeadline: 4 * time.Hour, BusinessTimezone: "Asia/Shanghai", Version: 1, CreatedAt: now, UpdatedAt: now}
	origin := domain.TrustZone{ID: "trust_zone_1", Code: "ZONE-1", Name: "Feature source", Timezone: "Asia/Shanghai", Status: domain.TrustZoneActive, DailyLimit: 10, CutoffHour: 6, Version: 1, CreatedAt: now, UpdatedAt: now}
	destination := domain.TrustZone{ID: "trust_zone_2", Code: "ZONE-2", Name: "Execution target", Timezone: "Asia/Shanghai", Status: domain.TrustZoneActive, DailyLimit: 10, CutoffHour: 6, Version: 1, CreatedAt: now, UpdatedAt: now}
	execution_pool := domain.ExecutionPool{ID: "pool_1", PoolKey: "GPU-POOL-1", State: domain.ExecutionPoolAvailable, CapacityUnits: 1000, AttestationDueAt: now.Add(48 * time.Hour), LastReconciledAt: now, Version: 1, CreatedAt: now, UpdatedAt: now}
	batch := domain.ToolRevision{ID: "revision_1", WorkspaceID: workspace.ID, RequesterZoneID: origin.ID, VersionTag: "REV-1", ProtocolFamily: "ranking-features-v2", OperationCount: 2, RequestedUnits: 100, State: domain.ToolRevisionVerified, ExpiresAt: now.Add(48 * time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now}
	err := store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertWorkspace(ctx, workspace); err != nil {
			return err
		}
		if err := tx.InsertTrustZone(ctx, origin); err != nil {
			return err
		}
		if err := tx.InsertTrustZone(ctx, destination); err != nil {
			return err
		}
		if err := tx.InsertExecutionPool(ctx, execution_pool); err != nil {
			return err
		}
		return tx.InsertToolRevision(ctx, batch)
	})
	if err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	return workspace, origin, destination, execution_pool, batch
}

func TestOpenAppliesMigrationsAndEnablesForeignKeys(t *testing.T) {
	store, ctx, _ := testStore(t)
	if err := store.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	var tableCount int
	if err := store.Read(ctx, func(reader repository.Reader) error {
		_, err := reader.GetWorkspace(ctx, "missing")
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount < 15 {
		t.Fatalf("table count = %d, want at least 15", tableCount)
	}
	var foreignKeys int
	if err := store.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d", foreignKeys)
	}
}

func TestTransactionRollsBackAllEntities(t *testing.T) {
	store, ctx, now := testStore(t)
	minimum, _ := domain.RiskScoreFromFloat(2)
	maximum, _ := domain.RiskScoreFromFloat(8)
	rangeValue, _ := domain.NewRiskRange(minimum, maximum)
	workspace := domain.Workspace{ID: "workspace_roll", Code: "ROLL", Name: "Rollback", Status: domain.WorkspaceActive, RiskScore: rangeValue, MaxExecution: time.Hour, ReviewDeadline: time.Hour, BusinessTimezone: "UTC", Version: 1, CreatedAt: now, UpdatedAt: now}
	err := store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertWorkspace(ctx, workspace); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("rollback transaction returned nil")
	}
	if err := store.Read(ctx, func(reader repository.Reader) error {
		_, err := reader.GetWorkspace(ctx, workspace.ID)
		return err
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("workspace after rollback error = %v", err)
	}
}

func TestRepositoryReadsAndDeepCopiesToolRevision(t *testing.T) {
	store, ctx, now := testStore(t)
	_, origin, _, _, batch := seedCatalog(t, store, ctx, now)
	got, err := storeReadToolRevision(store, ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RequesterZoneID != origin.ID || got.State != domain.ToolRevisionVerified {
		t.Fatalf("revision = %+v", got)
	}
	got.QuarantineNote = "local mutation"
	again, err := storeReadToolRevision(store, ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.QuarantineNote != "" {
		t.Fatalf("stored revision was mutated: %+v", again)
	}
}

func storeReadToolRevision(store *Store, ctx context.Context, id string) (domain.ToolRevision, error) {
	var result domain.ToolRevision
	err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		result, err = reader.GetToolRevision(ctx, id)
		return err
	})
	return result, err
}

func TestOptimisticVersionRejectsStaleUpdate(t *testing.T) {
	store, ctx, now := testStore(t)
	_, _, _, _, batch := seedCatalog(t, store, ctx, now)
	first, err := storeReadToolRevision(store, ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	second := first
	first.State = domain.ToolRevisionReserved
	first.UpdatedAt = now.Add(time.Minute)
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.UpdateToolRevision(ctx, first, first.Version) }); err != nil {
		t.Fatal(err)
	}
	second.State = domain.ToolRevisionReserved
	second.UpdatedAt = now.Add(2 * time.Minute)
	err = store.WithTx(ctx, func(tx repository.Tx) error { return tx.UpdateToolRevision(ctx, second, second.Version) })
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale update error = %v", err)
	}
}

func TestExecutionRequestFilterPaginationAndOrdering(t *testing.T) {
	store, ctx, now := testStore(t)
	workspace, origin, destination, execution_pool, batch := seedCatalog(t, store, ctx, now)
	secondToolRevision := batch
	secondToolRevision.ID = "revision_2"
	secondToolRevision.VersionTag = "EXT-2"
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertToolRevision(ctx, secondToolRevision) }); err != nil {
		t.Fatal(err)
	}
	execution_requests := []domain.ExecutionRequest{
		{ID: "ship_1", WorkspaceID: workspace.ID, RequesterZoneID: origin.ID, ExecutionZoneID: destination.ID, ExecutionPoolID: execution_pool.ID, RequestKey: "REF-1", State: domain.ExecutionRequestSubmitted, ScheduledStartAt: now.Add(time.Hour), ExpectedFinishAt: now.Add(2 * time.Hour), TotalRequestedUnits: 100, Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "ship_2", WorkspaceID: workspace.ID, RequesterZoneID: origin.ID, ExecutionZoneID: destination.ID, ExecutionPoolID: execution_pool.ID, RequestKey: "REF-2", State: domain.ExecutionRequestAuthorized, ScheduledStartAt: now.Add(2 * time.Hour), ExpectedFinishAt: now.Add(3 * time.Hour), TotalRequestedUnits: 100, Version: 1, CreatedAt: now, UpdatedAt: now},
	}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		for _, run := range execution_requests {
			if err := tx.InsertExecutionRequest(ctx, run); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var page repository.ExecutionRequestPage
	err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListExecutionRequests(ctx, repository.ExecutionRequestFilter{Page: repository.PageRequest{Limit: 1, Sort: "scheduled_start_at"}, WorkspaceID: workspace.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != "ship_1" {
		t.Fatalf("page = %+v", page)
	}
}

func TestIdempotencyRecordCopiesResponse(t *testing.T) {
	store, ctx, now := testStore(t)
	record := repository.IdempotencyRecord{Scope: "scope", Key: "key", RequestHash: "hash", ResponseCode: 201, ResponseBody: []byte("body"), ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.PutIdempotency(ctx, record) }); err != nil {
		t.Fatal(err)
	}
	var got repository.IdempotencyRecord
	if err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		got, err = reader.GetIdempotency(ctx, record.Scope, record.Key)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	got.ResponseBody[0] = 'B'
	var again repository.IdempotencyRecord
	if err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		again, err = reader.GetIdempotency(ctx, record.Scope, record.Key)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if string(again.ResponseBody) != "body" {
		t.Fatalf("response body = %q", again.ResponseBody)
	}
}

func TestOutboxClaimRetryAndCompletion(t *testing.T) {
	store, ctx, now := testStore(t)
	job := domain.OutboxJob{ID: "job_1", Kind: "execution_request_submitted", AggregateID: "ship_1", Payload: []byte("{}"), Status: domain.JobPending, MaxAttempts: 2, AvailableAt: now, CreatedAt: now, UpdatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertJob(ctx, job) }); err != nil {
		t.Fatal(err)
	}
	var claimed []domain.OutboxJob
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		var err error
		claimed, err = tx.ClaimJobs(ctx, now, 10)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].Attempts != 1 || claimed[0].Status != domain.JobRunning {
		t.Fatalf("claimed = %+v", claimed)
	}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		return tx.RetryJob(ctx, job.ID, now.Add(time.Minute), "temporary", false)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		jobs, err := tx.ClaimJobs(ctx, now.Add(2*time.Minute), 10)
		if err != nil || len(jobs) != 1 {
			return errors.New("job was not re-claimed")
		}
		return tx.CompleteJob(ctx, jobs[0].ID, now.Add(2*time.Minute))
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRestartRecoversPersistedRows(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "restart.db")
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, batch := seedCatalog(t, store, ctx, now)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := storeReadToolRevision(reopened, ctx, batch.ID)
	if err != nil || got.ID != batch.ID {
		t.Fatalf("recovered revision = %+v, error = %v", got, err)
	}
}

func TestForeignKeyRejectsUnknownWorkspace(t *testing.T) {
	store, ctx, now := testStore(t)
	batch := domain.ToolRevision{ID: "orphan", WorkspaceID: "missing", RequesterZoneID: "missing", VersionTag: "EXT", ProtocolFamily: "plasma", OperationCount: 1, RequestedUnits: 1, State: domain.ToolRevisionRegistered, ExpiresAt: now.Add(time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now}
	err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertToolRevision(ctx, batch) })
	if err == nil {
		t.Fatal("orphan insert succeeded")
	}
}

func TestReadinessCountsRelatedRows(t *testing.T) {
	store, ctx, now := testStore(t)
	workspace, origin, destination, execution_pool, batch := seedCatalog(t, store, ctx, now)
	run := domain.ExecutionRequest{ID: "ship_report", WorkspaceID: workspace.ID, RequesterZoneID: origin.ID, ExecutionZoneID: destination.ID, ExecutionPoolID: execution_pool.ID, RequestKey: "REPORT-1", State: domain.ExecutionRequestCompleted, ScheduledStartAt: now, ExpectedFinishAt: now.Add(time.Hour), TotalRequestedUnits: batch.RequestedUnits, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertExecutionRequest(ctx, run); err != nil {
			return err
		}
		batch.State = domain.ToolRevisionExecuted
		batch.ExecutionRequestID = run.ID
		if err := tx.UpdateToolRevision(ctx, batch, batch.Version); err != nil {
			return err
		}
		return tx.InsertExecutionRequestInput(ctx, domain.ExecutionRequestInput{ExecutionRequestID: run.ID, ToolRevisionID: batch.ID, AddedAt: now})
	}); err != nil {
		t.Fatal(err)
	}
	var report domain.RunReadiness
	if err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		report, err = reader.GetRunReadiness(ctx, run.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if report.ExpectedToolRevisionCount != 1 || report.MaterializedToolRevisionCount != 1 || !report.Complete {
		t.Fatalf("report = %+v", report)
	}
}
