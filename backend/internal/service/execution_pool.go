package service

import (
	"context"
	"strings"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type ExecutionPoolService struct{ dependencies }

func (s *ExecutionPoolService) StartReconciliation(ctx context.Context, execution_poolID string) (domain.ExecutionPool, error) {
	return s.change(ctx, execution_poolID, "execution_pool_reconciliation_started", func(execution_pool *domain.ExecutionPool) error {
		return execution_pool.StartReconciliation(s.clock.Now())
	})
}

func (s *ExecutionPoolService) CompleteReconciliation(ctx context.Context, execution_poolID string) (domain.ExecutionPool, error) {
	return s.change(ctx, execution_poolID, "execution_pool_reconciliation_completed", func(execution_pool *domain.ExecutionPool) error {
		return execution_pool.CompleteReconciliation(s.clock.Now())
	})
}

func (s *ExecutionPoolService) Retire(ctx context.Context, execution_poolID, reason string) (domain.ExecutionPool, error) {
	if strings.TrimSpace(reason) == "" {
		return domain.ExecutionPool{}, domain.FieldError{Field: "reason", Message: "is required"}
	}
	return s.changeWithMetadata(ctx, execution_poolID, "execution_pool_retired", map[string]string{"reason": strings.TrimSpace(reason)}, func(execution_pool *domain.ExecutionPool) error {
		return execution_pool.Retire(s.clock.Now())
	})
}

func (s *ExecutionPoolService) change(ctx context.Context, execution_poolID, action string, mutate func(*domain.ExecutionPool) error) (domain.ExecutionPool, error) {
	return s.changeWithMetadata(ctx, execution_poolID, action, nil, mutate)
}

func (s *ExecutionPoolService) changeWithMetadata(ctx context.Context, execution_poolID, action string, metadata map[string]string, mutate func(*domain.ExecutionPool) error) (domain.ExecutionPool, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.ExecutionPool{}, err
	}
	var result domain.ExecutionPool
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		execution_pool, err := tx.GetExecutionPool(ctx, execution_poolID)
		if err != nil {
			return err
		}
		before := execution_pool.Version
		if err := mutate(&execution_pool); err != nil {
			return err
		}
		if err := tx.UpdateExecutionPool(ctx, execution_pool, before); err != nil {
			return err
		}
		result = execution_pool
		return s.audit.Record(ctx, tx, action, "execution_pool", execution_pool.ID, "success", metadata)
	})
	return result, err
}
