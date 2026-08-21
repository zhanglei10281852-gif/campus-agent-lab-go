package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
)

func (q *queries) GetRunReadiness(ctx context.Context, requestID string) (domain.RunReadiness, error) {
	run, err := q.GetExecutionRequest(ctx, requestID)
	if err != nil {
		return domain.RunReadiness{}, err
	}
	var report domain.RunReadiness
	report.ExecutionRequestID = run.ID
	report.ExecutionRequestState = run.State
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_tools WHERE request_id = ?`, requestID).Scan(&report.ExpectedToolRevisionCount); err != nil {
		return domain.RunReadiness{}, translateError("count run revisions", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_tools ri JOIN tool_revisions s ON s.id = ri.tool_revision_id
        WHERE ri.request_id = ? AND s.state IN ('executed', 'approved', 'rejected', 'blocked')`, requestID).Scan(&report.MaterializedToolRevisionCount); err != nil {
		return domain.RunReadiness{}, translateError("count executed revisions", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_tools ri JOIN tool_revisions s ON s.id = ri.tool_revision_id
		WHERE ri.request_id = ? AND s.state = 'approved'`, requestID).Scan(&report.ApprovedToolRevisionCount); err != nil {
		return domain.RunReadiness{}, translateError("count approved revisions", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_tools ri JOIN tool_revisions s ON s.id = ri.tool_revision_id
		WHERE ri.request_id = ? AND s.state = 'rejected'`, requestID).Scan(&report.RejectedToolRevisionCount); err != nil {
		return domain.RunReadiness{}, translateError("count rejected revisions", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_tools ri JOIN tool_revisions s ON s.id = ri.tool_revision_id
		WHERE ri.request_id = ? AND s.state = 'blocked'`, requestID).Scan(&report.QuarantinedCount); err != nil {
		return domain.RunReadiness{}, translateError("count blocked revisions", err)
	}
	var pending int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM approval_tasks WHERE request_id = ? AND status = 'pending'`, requestID).Scan(&pending); err != nil {
		return domain.RunReadiness{}, translateError("count pending approval_tasks", err)
	}
	report.PendingApprovalTask = pending > 0
	var open int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM policy_incidents WHERE request_id = ? AND status IN ('open', 'reviewing')`, requestID).Scan(&open); err != nil {
		return domain.RunReadiness{}, translateError("count open policy_incidents", err)
	}
	report.OpenPolicyIncident = open > 0
	var lastReceipt sql.NullString
	if err := q.q.QueryRowContext(ctx, `SELECT MAX(recorded_at) FROM execution_receipts WHERE request_id = ?`, requestID).Scan(&lastReceipt); err != nil {
		return domain.RunReadiness{}, translateError("get last receipt", err)
	}
	if lastReceipt.Valid {
		parsed, err := parseTime(lastReceipt.String)
		if err != nil {
			return domain.RunReadiness{}, err
		}
		report.LastReceiptAt = &parsed
	}
	report.Evaluate()
	return report.Clone(), nil
}

func (q *queries) latestReceiptAt(ctx context.Context, requestID string) (time.Time, error) {
	var raw string
	if err := q.q.QueryRowContext(ctx, `SELECT recorded_at FROM execution_receipts WHERE request_id = ? ORDER BY recorded_at DESC LIMIT 1`, requestID).Scan(&raw); err != nil {
		return time.Time{}, translateError("get latest receipt", err)
	}
	parsed, err := parseTime(raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse latest receipt: %w", err)
	}
	return parsed, nil
}
