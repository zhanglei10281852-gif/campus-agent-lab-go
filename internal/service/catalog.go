package service

import (
	"context"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/requestmeta"
)

type CatalogService struct{ dependencies }

func (s *CatalogService) CreateWorkspace(ctx context.Context, workspace domain.Workspace) (domain.Workspace, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.Workspace{}, err
	}
	now := s.clock.Now()
	workspace.ID = identity.New("workspace")
	workspace.Code = domain.NormalizeCode(workspace.Code)
	if err := domain.ValidateBusinessCode("code", workspace.Code); err != nil {
		return domain.Workspace{}, err
	}
	workspace.Status = domain.WorkspaceDraft
	workspace.Version = 1
	workspace.CreatedAt, workspace.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertWorkspace(ctx, workspace); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "workspace_created", "workspace", workspace.ID, "success", nil)
	})
	return workspace, err
}

func (s *CatalogService) ActivateWorkspace(ctx context.Context, workspaceID string) (domain.Workspace, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.Workspace{}, err
	}
	var result domain.Workspace
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		workspace, err := tx.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return err
		}
		if workspace.Status != domain.WorkspaceDraft {
			return domain.TransitionError{Entity: "workspace", From: string(workspace.Status), To: string(domain.WorkspaceActive)}
		}
		before := workspace.Version
		workspace.Status = domain.WorkspaceActive
		workspace.UpdatedAt = s.clock.Now()
		if err := tx.UpdateWorkspace(ctx, workspace, before); err != nil {
			return err
		}
		result = workspace
		return s.audit.Record(ctx, tx, "workspace_activated", "workspace", workspace.ID, "success", nil)
	})
	return result, err
}

func (s *CatalogService) CreateTrustZone(ctx context.Context, trust_zone domain.TrustZone) (domain.TrustZone, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.TrustZone{}, err
	}
	now := s.clock.Now()
	trust_zone.ID = identity.New("trust_zone")
	trust_zone.Code = domain.NormalizeCode(trust_zone.Code)
	if err := domain.ValidateBusinessCode("code", trust_zone.Code); err != nil {
		return domain.TrustZone{}, err
	}
	trust_zone.Status = domain.TrustZoneActive
	trust_zone.Version = 1
	trust_zone.CreatedAt, trust_zone.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertTrustZone(ctx, trust_zone); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "trust_zone_created", "trust_zone", trust_zone.ID, "success", nil)
	})
	return trust_zone, err
}

func (s *CatalogService) CreateExecutionPool(ctx context.Context, execution_pool domain.ExecutionPool) (domain.ExecutionPool, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.ExecutionPool{}, err
	}
	now := s.clock.Now()
	execution_pool.ID = identity.New("pool")
	execution_pool.PoolKey = domain.NormalizeCode(execution_pool.PoolKey)
	if err := domain.ValidateBusinessCode("pool_key", execution_pool.PoolKey); err != nil {
		return domain.ExecutionPool{}, err
	}
	execution_pool.State = domain.ExecutionPoolAvailable
	execution_pool.Version = 1
	execution_pool.CreatedAt, execution_pool.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertExecutionPool(ctx, execution_pool); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "execution_pool_created", "execution_pool", execution_pool.ID, "success", nil)
	})
	return execution_pool, err
}

func (s *CatalogService) RegisterToolRevision(ctx context.Context, batch domain.ToolRevision) (domain.ToolRevision, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.ToolRevision{}, err
	}
	now := s.clock.Now()
	batch.ID = identity.New("revision")
	batch.State = domain.ToolRevisionRegistered
	batch.Version = 1
	batch.CreatedAt, batch.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertToolRevision(ctx, batch); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "revision_registered", "tool_revision", batch.ID, "success", nil)
	})
	return batch, err
}

func (s *CatalogService) VerifyToolRevision(ctx context.Context, revisionID string) (domain.ToolRevision, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return domain.ToolRevision{}, err
	}
	var result domain.ToolRevision
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		batch, err := tx.GetToolRevision(ctx, revisionID)
		if err != nil {
			return err
		}
		if err := batch.Transition(domain.ToolRevisionVerified, s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdateToolRevision(ctx, batch, batch.Version); err != nil {
			return err
		}
		result = batch
		return s.audit.Record(ctx, tx, "revision_verified", "tool_revision", batch.ID, "success", nil)
	})
	return result, err
}

func principalOrEmpty(ctx context.Context) (domain.Principal, bool) {
	principal, ok := requestmeta.Principal(ctx)
	return principal, ok
}
