package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/requestmeta"
)

type TrainingBackend interface {
	ListCohorts(context.Context, repository.PageRequest, string, domain.CohortStatus) ([]domain.Cohort, int, error)
	ListAllCohorts(context.Context) ([]domain.Cohort, error)
	GetCohort(context.Context, string) (domain.Cohort, error)
	CreateCohort(context.Context, domain.Cohort, string, string) error
	UpdateCohort(context.Context, domain.Cohort, int64, string, string) error
	DeleteCohort(context.Context, string, int64, string, string) error
	ListTrainees(context.Context, repository.PageRequest, string, string, string, domain.TraineeStatus) ([]domain.Trainee, int, error)
	GetTrainee(context.Context, string) (domain.Trainee, error)
	CreateTrainee(context.Context, domain.Trainee, string, string) error
	UpdateTrainee(context.Context, domain.Trainee, int64, string, string) error
	DeleteTrainee(context.Context, string, int64, string, string) error
	TrainingSummary(context.Context) (domain.TrainingSummary, error)
}

type TrainingService struct{ backend TrainingBackend }

func requestID(ctx context.Context) string { return requestmeta.RequestID(ctx) }

func newTrainingService(store repository.Store) *TrainingService {
	backend, ok := store.(TrainingBackend)
	if !ok {
		return nil
	}
	return &TrainingService{backend: backend}
}

func (s *TrainingService) ensure() error {
	if s == nil || s.backend == nil {
		return fmt.Errorf("training backend unavailable")
	}
	return nil
}

func (s *TrainingService) ListCohorts(ctx context.Context, page repository.PageRequest, search string, status domain.CohortStatus) ([]domain.Cohort, int, error) {
	if err := s.ensure(); err != nil {
		return nil, 0, err
	}
	return s.backend.ListCohorts(ctx, page.Normalize(200), strings.TrimSpace(search), status)
}

func (s *TrainingService) ListAllCohorts(ctx context.Context) ([]domain.Cohort, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	return s.backend.ListAllCohorts(ctx)
}

func (s *TrainingService) GetCohort(ctx context.Context, id string) (domain.Cohort, error) {
	if err := s.ensure(); err != nil {
		return domain.Cohort{}, err
	}
	return s.backend.GetCohort(ctx, strings.TrimSpace(id))
}

func (s *TrainingService) CreateCohort(ctx context.Context, principal domain.Principal, cohort domain.Cohort) (domain.Cohort, error) {
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator); err != nil {
		return domain.Cohort{}, err
	}
	if err := cohort.Validate(); err != nil {
		return domain.Cohort{}, err
	}
	if err := s.ensure(); err != nil {
		return domain.Cohort{}, err
	}
	if err := s.backend.CreateCohort(ctx, cohort, principal.UserID, requestID(ctx)); err != nil {
		return domain.Cohort{}, err
	}
	return cohort, nil
}

func (s *TrainingService) UpdateCohort(ctx context.Context, principal domain.Principal, cohort domain.Cohort, expected int64) (domain.Cohort, error) {
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator); err != nil {
		return domain.Cohort{}, err
	}
	if err := cohort.Validate(); err != nil {
		return domain.Cohort{}, err
	}
	if expected < 1 {
		return domain.Cohort{}, domain.FieldError{Field: "version", Message: "must be positive"}
	}
	if err := s.ensure(); err != nil {
		return domain.Cohort{}, err
	}
	if err := s.backend.UpdateCohort(ctx, cohort, expected, principal.UserID, requestID(ctx)); err != nil {
		return domain.Cohort{}, err
	}
	cohort.Version = expected + 1
	return cohort, nil
}

func (s *TrainingService) DeleteCohort(ctx context.Context, principal domain.Principal, id string, expected int64) error {
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" || expected < 1 {
		return domain.FieldError{Field: "id", Message: "id and version are required"}
	}
	if err := s.ensure(); err != nil {
		return err
	}
	return s.backend.DeleteCohort(ctx, id, expected, principal.UserID, requestID(ctx))
}

func (s *TrainingService) ListTrainees(ctx context.Context, page repository.PageRequest, name, studentNo, cohortID string, status domain.TraineeStatus) ([]domain.Trainee, int, error) {
	if err := s.ensure(); err != nil {
		return nil, 0, err
	}
	return s.backend.ListTrainees(ctx, page.Normalize(200), strings.TrimSpace(name), strings.TrimSpace(studentNo), strings.TrimSpace(cohortID), status)
}

func (s *TrainingService) GetTrainee(ctx context.Context, id string) (domain.Trainee, error) {
	if err := s.ensure(); err != nil {
		return domain.Trainee{}, err
	}
	return s.backend.GetTrainee(ctx, strings.TrimSpace(id))
}

func (s *TrainingService) CreateTrainee(ctx context.Context, principal domain.Principal, trainee domain.Trainee) (domain.Trainee, error) {
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator); err != nil {
		return domain.Trainee{}, err
	}
	if err := trainee.Validate(); err != nil {
		return domain.Trainee{}, err
	}
	if err := s.ensure(); err != nil {
		return domain.Trainee{}, err
	}
	if err := s.backend.CreateTrainee(ctx, trainee, principal.UserID, requestID(ctx)); err != nil {
		return domain.Trainee{}, err
	}
	return trainee, nil
}

func (s *TrainingService) UpdateTrainee(ctx context.Context, principal domain.Principal, trainee domain.Trainee, expected int64) (domain.Trainee, error) {
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator); err != nil {
		return domain.Trainee{}, err
	}
	if err := trainee.Validate(); err != nil {
		return domain.Trainee{}, err
	}
	if expected < 1 {
		return domain.Trainee{}, domain.FieldError{Field: "version", Message: "must be positive"}
	}
	if err := s.ensure(); err != nil {
		return domain.Trainee{}, err
	}
	if err := s.backend.UpdateTrainee(ctx, trainee, expected, principal.UserID, requestID(ctx)); err != nil {
		return domain.Trainee{}, err
	}
	trainee.Version = expected + 1
	return trainee, nil
}

func (s *TrainingService) DeleteTrainee(ctx context.Context, principal domain.Principal, id string, expected int64) error {
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" || expected < 1 {
		return domain.FieldError{Field: "id", Message: "id and version are required"}
	}
	if err := s.ensure(); err != nil {
		return err
	}
	return s.backend.DeleteTrainee(ctx, id, expected, principal.UserID, requestID(ctx))
}

func (s *TrainingService) Summary(ctx context.Context) (domain.TrainingSummary, error) {
	if err := s.ensure(); err != nil {
		return domain.TrainingSummary{}, err
	}
	return s.backend.TrainingSummary(ctx)
}
