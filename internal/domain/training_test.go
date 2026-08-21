package domain

import (
	"strings"
	"testing"
	"time"
)

func validCohort() Cohort {
	return Cohort{ID: "cohort_test", Code: "AI-2026-A", Name: "智能体实训一班", Grade: "2026", Instructor: "林老师",
		Capacity: 30, Status: CohortActive, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func validTrainee() Trainee {
	return Trainee{ID: "trainee_test", StudentNo: "S2026001", Name: "张三", Email: "zhangsan@example.test",
		CohortID: "cohort_test", Status: TraineeActive, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func TestCohortValidationRejectsCapacityOutsideRange(t *testing.T) {
	for _, capacity := range []int{0, -1, 501} {
		value := validCohort()
		value.Capacity = capacity
		if err := value.Validate(); err == nil {
			t.Fatalf("capacity %d should be rejected", capacity)
		}
	}
}

func TestCohortValidationKeepsEnrollmentBelowCapacity(t *testing.T) {
	value := validCohort()
	value.StudentCount = value.Capacity + 1
	if err := value.Validate(); err == nil || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("expected capacity error, got %v", err)
	}
}

func TestTraineeValidationRequiresInstitutionalEmail(t *testing.T) {
	value := validTrainee()
	value.Email = "invalid"
	if err := value.Validate(); err == nil {
		t.Fatal("invalid email accepted")
	}
}

func TestTraineeValidationRejectsUnknownStatus(t *testing.T) {
	value := validTrainee()
	value.Status = "pending"
	if err := value.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}
