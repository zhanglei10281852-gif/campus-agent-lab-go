package service

import (
	"context"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

func (s *QueryService) ReconcileExecutionRequest(ctx context.Context, requestID string) (domain.RunReadiness, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper, domain.RoleToolOperator, domain.RoleSecurityReviewer, domain.RoleComplianceAuditor); err != nil {
		return domain.RunReadiness{}, err
	}
	var report domain.RunReadiness
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		report, err = reader.GetRunReadiness(ctx, requestID)
		return err
	})
	return report, err
}
