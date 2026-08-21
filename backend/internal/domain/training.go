package domain

import (
	"net/mail"
	"strings"
	"time"
)

type CohortStatus string

const (
	CohortDraft  CohortStatus = "draft"
	CohortActive CohortStatus = "active"
	CohortClosed CohortStatus = "closed"
)

type Cohort struct {
	ID           string       `json:"id"`
	Code         string       `json:"code"`
	Name         string       `json:"name"`
	Grade        string       `json:"grade"`
	Instructor   string       `json:"instructor"`
	WorkspaceID  string       `json:"workspace_id,omitempty"`
	Capacity     int          `json:"capacity"`
	StudentCount int          `json:"student_count"`
	Status       CohortStatus `json:"status"`
	Version      int64        `json:"version"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func (c Cohort) Validate() error {
	if strings.TrimSpace(c.Code) == "" || len(c.Code) > 32 {
		return FieldError{Field: "code", Message: "is required and must not exceed 32 characters"}
	}
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 80 {
		return FieldError{Field: "name", Message: "is required and must not exceed 80 characters"}
	}
	if strings.TrimSpace(c.Grade) == "" {
		return FieldError{Field: "grade", Message: "is required"}
	}
	if strings.TrimSpace(c.Instructor) == "" {
		return FieldError{Field: "instructor", Message: "is required"}
	}
	if c.Capacity < 1 || c.Capacity > 500 {
		return FieldError{Field: "capacity", Message: "must be between 1 and 500"}
	}
	if c.StudentCount < 0 || c.StudentCount > c.Capacity {
		return FieldError{Field: "student_count", Message: "must stay within cohort capacity"}
	}
	switch c.Status {
	case CohortDraft, CohortActive, CohortClosed:
	default:
		return FieldError{Field: "status", Message: "is invalid"}
	}
	return nil
}

type TraineeStatus string

const (
	TraineeActive    TraineeStatus = "active"
	TraineeSuspended TraineeStatus = "suspended"
	TraineeCompleted TraineeStatus = "completed"
)

type Trainee struct {
	ID        string        `json:"id"`
	StudentNo string        `json:"student_no"`
	Name      string        `json:"name"`
	Gender    string        `json:"gender,omitempty"`
	BirthDate string        `json:"birth_date,omitempty"`
	Phone     string        `json:"phone,omitempty"`
	Email     string        `json:"email"`
	CohortID  string        `json:"cohort_id"`
	Cohort    string        `json:"cohort_name,omitempty"`
	Status    TraineeStatus `json:"status"`
	Version   int64         `json:"version"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func (t Trainee) Validate() error {
	if strings.TrimSpace(t.StudentNo) == "" || len(t.StudentNo) > 32 {
		return FieldError{Field: "student_no", Message: "is required and must not exceed 32 characters"}
	}
	if strings.TrimSpace(t.Name) == "" || len(t.Name) > 80 {
		return FieldError{Field: "name", Message: "is required and must not exceed 80 characters"}
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(t.Email)); err != nil {
		return FieldError{Field: "email", Message: "is invalid"}
	}
	if strings.TrimSpace(t.CohortID) == "" {
		return FieldError{Field: "cohort_id", Message: "is required"}
	}
	switch t.Status {
	case TraineeActive, TraineeSuspended, TraineeCompleted:
	default:
		return FieldError{Field: "status", Message: "is invalid"}
	}
	return nil
}

type TrainingSummary struct {
	Cohorts        int `json:"cohorts"`
	ActiveCohorts  int `json:"active_cohorts"`
	Trainees       int `json:"trainees"`
	ActiveTrainees int `json:"active_trainees"`
}
