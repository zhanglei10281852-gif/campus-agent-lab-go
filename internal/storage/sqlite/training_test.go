package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

func trainingStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), "file:training-test-"+identity.New("db")+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedWorkspaceForTraining(t *testing.T, store *Store) domain.Workspace {
	t.Helper()
	now := time.Now().UTC()
	workspace := domain.Workspace{ID: identity.New("ws"), Code: "TRAIN-" + identity.New("code")[:8], Name: "实训工作区",
		Status: domain.WorkspaceActive, RiskScore: domain.RiskRange{Minimum: 100, Maximum: 900}, MaxExecution: time.Hour,
		ReviewDeadline: time.Hour, BusinessTimezone: "Asia/Shanghai", Version: 1, CreatedAt: now, UpdatedAt: now}
	err := store.WithTx(context.Background(), func(tx repository.Tx) error { return tx.InsertWorkspace(context.Background(), workspace) })
	if err != nil {
		t.Fatal(err)
	}
	return workspace
}

func createActiveCohort(t *testing.T, store *Store, capacity int) domain.Cohort {
	t.Helper()
	now := time.Now().UTC()
	value := domain.Cohort{ID: identity.New("cohort"), Code: "C-" + identity.New("code")[:8], Name: "实训班", Grade: "2026",
		Instructor: "指导老师", Capacity: capacity, Status: domain.CohortActive, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateCohort(context.Background(), value, "user_test", "req_test"); err != nil {
		t.Fatal(err)
	}
	return value
}

func trainee(value, cohortID string) domain.Trainee {
	now := time.Now().UTC()
	return domain.Trainee{ID: identity.New("trainee"), StudentNo: value, Name: "学员" + value, Email: value + "@example.test",
		CohortID: cohortID, Status: domain.TraineeActive, Version: 1, CreatedAt: now, UpdatedAt: now}
}

func TestTrainingMigrationCreatesRelatedTables(t *testing.T) {
	store := trainingStore(t)
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('cohorts','trainees','trainee_workspace_assignments')").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected three training tables, got %d", count)
	}
}

func TestCohortCapacityAndDeleteGuard(t *testing.T) {
	store := trainingStore(t)
	cohort := createActiveCohort(t, store, 1)
	if err := store.CreateTrainee(context.Background(), trainee("S1", cohort.ID), "user_test", "req1"); err != nil {
		t.Fatal(err)
	}
	err := store.CreateTrainee(context.Background(), trainee("S2", cohort.ID), "user_test", "req2")
	if !errors.Is(err, domain.ErrCapacityExceeded) {
		t.Fatalf("expected capacity error, got %v", err)
	}
	if err := store.DeleteCohort(context.Background(), cohort.ID, 1, "user_test", "req3"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected delete conflict, got %v", err)
	}
}

func TestTraineeUpdateUsesOptimisticVersion(t *testing.T) {
	store := trainingStore(t)
	cohort := createActiveCohort(t, store, 4)
	value := trainee("S3", cohort.ID)
	if err := store.CreateTrainee(context.Background(), value, "user_test", "req4"); err != nil {
		t.Fatal(err)
	}
	value.Name = "新名字"
	value.UpdatedAt = time.Now().UTC()
	if err := store.UpdateTrainee(context.Background(), value, 99, "user_test", "req5"); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestTrainingSummaryReflectsEnrollment(t *testing.T) {
	store := trainingStore(t)
	cohort := createActiveCohort(t, store, 4)
	if err := store.CreateTrainee(context.Background(), trainee("S4", cohort.ID), "user_test", "req6"); err != nil {
		t.Fatal(err)
	}
	summary, err := store.TrainingSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Cohorts != 1 || summary.ActiveCohorts != 1 || summary.Trainees != 1 || summary.ActiveTrainees != 1 {
		t.Fatalf("unexpected summary %#v", summary)
	}
}

func TestListTraineesSupportsDescendingCreatedAtSort(t *testing.T) {
	store := trainingStore(t)
	cohort := createActiveCohort(t, store, 4)
	if err := store.CreateTrainee(context.Background(), trainee("S5", cohort.ID), "user_test", "req7"); err != nil {
		t.Fatal(err)
	}
	items, total, err := store.ListTrainees(context.Background(), repository.PageRequest{
		Limit: 6, Offset: 0, Sort: "created_at", Desc: true,
	}, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected trainee page: total=%d items=%d", total, len(items))
	}
}
