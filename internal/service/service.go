package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/audit"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/clock"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type Services struct {
	Auth           *AuthService
	Training       *TrainingService
	Scenario       *ScenarioService
	Catalog        *CatalogService
	ExecutionPools *ExecutionPoolService
	Execution      *ExecutionService
	Approval       *ApprovalService
	Receipts       *ReceiptService
	Review         *ReviewService
	Query          *QueryService
}

type dependencies struct {
	store repository.Store
	clock clock.Clock
	audit audit.Recorder
}

func New(store repository.Store, c clock.Clock, sessionTTL, approval_taskTTL time.Duration) *Services {
	deps := dependencies{store: store, clock: c, audit: audit.NewRecorder(c)}
	return &Services{
		Auth:           &AuthService{dependencies: deps, sessionTTL: sessionTTL},
		Training:       newTrainingService(store),
		Scenario:       &ScenarioService{dependencies: deps},
		Catalog:        &CatalogService{dependencies: deps},
		ExecutionPools: &ExecutionPoolService{dependencies: deps},
		Execution:      &ExecutionService{dependencies: deps},
		Approval:       &ApprovalService{dependencies: deps, approval_taskTTL: approval_taskTTL},
		Receipts:       &ReceiptService{dependencies: deps},
		Review:         &ReviewService{dependencies: deps},
		Query:          &QueryService{dependencies: deps},
	}
}

func requireRole(principal domain.Principal, roles ...domain.Role) error {
	if principal.UserID == "" {
		return fmt.Errorf("authentication required: %w", domain.ErrValidation)
	}
	if !principal.Can(roles...) {
		return fmt.Errorf("role %s is not permitted: %w", principal.Role, domain.ErrConflict)
	}
	return nil
}

func requireAction(principal domain.Principal, action domain.Action) error {
	if principal.UserID == "" {
		return fmt.Errorf("authentication required: %w", domain.ErrValidation)
	}
	if !principal.CanAction(action) {
		return fmt.Errorf("role %s cannot perform %s: %w", principal.Role, action, domain.ErrConflict)
	}
	return nil
}

func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }
