package service

import (
	"context"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type QueryService struct{ dependencies }

func (s *QueryService) PlatformSummary(ctx context.Context) (repository.PlatformSummary, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireAction(principal, domain.ActionReadGovernance); err != nil {
		return repository.PlatformSummary{}, err
	}
	var summary repository.PlatformSummary
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		summary, err = reader.GetPlatformSummary(ctx)
		return err
	})
	return summary, err
}

func (s *QueryService) ExecutionRequest(ctx context.Context, id string) (domain.ExecutionRequest, []domain.ToolRevision, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator, domain.RoleSecurityReviewer, domain.RoleComplianceAuditor); err != nil {
		return domain.ExecutionRequest{}, nil, err
	}
	var run domain.ExecutionRequest
	var items []domain.ToolRevision
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		run, err = reader.GetExecutionRequest(ctx, id)
		if err != nil {
			return err
		}
		items, err = reader.ListExecutionRequestInputs(ctx, id)
		return err
	})
	return run, items, err
}

func (s *QueryService) ExecutionRequests(ctx context.Context, filter repository.ExecutionRequestFilter) (repository.ExecutionRequestPage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator, domain.RoleSecurityReviewer, domain.RoleComplianceAuditor); err != nil {
		return repository.ExecutionRequestPage{}, err
	}
	var page repository.ExecutionRequestPage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListExecutionRequests(ctx, filter)
		return err
	})
	return page, err
}

func (s *QueryService) ToolRevisions(ctx context.Context, filter repository.ToolRevisionFilter) (repository.ToolRevisionPage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator, domain.RoleSecurityReviewer, domain.RoleComplianceAuditor); err != nil {
		return repository.ToolRevisionPage{}, err
	}
	var page repository.ToolRevisionPage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListToolRevisions(ctx, filter)
		return err
	})
	return page, err
}

func (s *QueryService) PolicyIncidents(ctx context.Context, filter repository.PolicyIncidentFilter) (repository.PolicyIncidentPage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleSecurityReviewer, domain.RoleComplianceAuditor); err != nil {
		return repository.PolicyIncidentPage{}, err
	}
	var page repository.PolicyIncidentPage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListPolicyIncidents(ctx, filter)
		return err
	})
	return page, err
}

func (s *QueryService) Audit(ctx context.Context, filter repository.AuditFilter) (repository.AuditPage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleComplianceAuditor, domain.RoleAgentDeveloper); err != nil {
		return repository.AuditPage{}, err
	}
	var page repository.AuditPage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListAuditEvents(ctx, filter)
		return err
	})
	return page, err
}
