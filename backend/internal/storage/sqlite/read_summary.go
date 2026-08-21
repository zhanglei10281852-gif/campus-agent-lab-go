package sqlite

import (
	"context"
	"fmt"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

func (q *queries) GetPlatformSummary(ctx context.Context) (repository.PlatformSummary, error) {
	var summary repository.PlatformSummary
	queries := []struct {
		name   string
		target *int
		sql    string
	}{
		{"active workspaces", &summary.WorkspacesActive, `SELECT COUNT(*) FROM workspaces WHERE status = 'active'`},
		{"verified tool revisions", &summary.ToolRevisionsValidated, `SELECT COUNT(*) FROM tool_revisions WHERE state = 'verified'`},
		{"executing tool revisions", &summary.ToolRevisionsMaterializing, `SELECT COUNT(*) FROM tool_revisions WHERE state = 'executing'`},
		{"blocked tool_revisions", &summary.ToolRevisionsQuarantined, `SELECT COUNT(*) FROM tool_revisions WHERE state = 'blocked'`},
		{"available execution_pools", &summary.ExecutionPoolsAvailable, `SELECT COUNT(*) FROM execution_pools WHERE state = 'available'`},
		{"active execution requests", &summary.ExecutionRequestsActive, `SELECT COUNT(*) FROM execution_requests WHERE state IN ('submitted', 'authorized', 'executing', 'completed')`},
		{"open policy_incidents", &summary.OpenPolicyIncidents, `SELECT COUNT(*) FROM policy_incidents WHERE status IN ('open', 'reviewing')`},
		{"pending approval_tasks", &summary.PendingApprovalTasks, `SELECT COUNT(*) FROM approval_tasks WHERE status = 'pending'`},
		{"failed jobs", &summary.FailedJobs, `SELECT COUNT(*) FROM outbox_jobs WHERE status IN ('failed', 'dead')`},
	}
	for _, item := range queries {
		if err := q.q.QueryRowContext(ctx, item.sql).Scan(item.target); err != nil {
			return repository.PlatformSummary{}, fmt.Errorf("count %s: %w", item.name, err)
		}
	}
	return summary, nil
}
