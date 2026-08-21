package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

func (q *queries) GetPendingApprovalTask(ctx context.Context, requestID string) (domain.ApprovalTask, error) {
	approval_task, err := scanApprovalTask(q.q.QueryRowContext(ctx, approval_taskSelect+` WHERE request_id = ? AND status = 'pending'`, requestID))
	return approval_task, translateError("get pending approval_task", err)
}

func (q *queries) GetApprovalTask(ctx context.Context, id string) (domain.ApprovalTask, error) {
	approval_task, err := scanApprovalTask(q.q.QueryRowContext(ctx, approval_taskSelect+` WHERE id = ?`, id))
	return approval_task, translateError("get approval_task", err)
}

const approval_taskSelect = `SELECT id, request_id, requester_id, reviewer_id, review_queue, status, expires_at,
    resolved_at, resolution_note, version, created_at, updated_at FROM approval_tasks`

func scanApprovalTask(row scanner) (domain.ApprovalTask, error) {
	var approval_task domain.ApprovalTask
	var status, expiresAt, createdAt, updatedAt string
	var resolvedAt sql.NullString
	if err := row.Scan(&approval_task.ID, &approval_task.ExecutionRequestID, &approval_task.RequesterID, &approval_task.ReviewerID,
		&approval_task.ReviewQueue, &status, &expiresAt, &resolvedAt, &approval_task.ResolutionNote,
		&approval_task.Version, &createdAt, &updatedAt); err != nil {
		return domain.ApprovalTask{}, err
	}
	approval_task.Status = domain.ApprovalTaskStatus(status)
	var err error
	if approval_task.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return domain.ApprovalTask{}, err
	}
	if approval_task.ResolvedAt, err = parseNullableTime(resolvedAt); err != nil {
		return domain.ApprovalTask{}, err
	}
	if approval_task.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.ApprovalTask{}, err
	}
	if approval_task.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.ApprovalTask{}, err
	}
	return approval_task, nil
}

func (q *queries) GetActivePolicyIncident(ctx context.Context, requestID string) (domain.PolicyIncident, error) {
	policy_incident, err := scanPolicyIncident(q.q.QueryRowContext(ctx, policy_incidentSelect+` WHERE request_id = ? AND status IN ('open', 'reviewing')`, requestID))
	return policy_incident, translateError("get active policy_incident", err)
}

func (q *queries) GetPolicyIncident(ctx context.Context, id string) (domain.PolicyIncident, error) {
	policy_incident, err := scanPolicyIncident(q.q.QueryRowContext(ctx, policy_incidentSelect+` WHERE id = ?`, id))
	return policy_incident, translateError("get policy_incident", err)
}

const policy_incidentSelect = `SELECT id, request_id, status, first_receipt_at, last_receipt_at,
    minimum_risk_score_millis, maximum_risk_score_millis, receipt_count, review_due_at, version, created_at, updated_at FROM policy_incidents`

func scanPolicyIncident(row scanner) (domain.PolicyIncident, error) {
	var policy_incident domain.PolicyIncident
	var status, firstReceiptAt, lastReceiptAt, reviewDueAt, createdAt, updatedAt string
	if err := row.Scan(&policy_incident.ID, &policy_incident.ExecutionRequestID, &status, &firstReceiptAt, &lastReceiptAt,
		&policy_incident.Minimum, &policy_incident.Maximum, &policy_incident.ReceiptCount, &reviewDueAt,
		&policy_incident.Version, &createdAt, &updatedAt); err != nil {
		return domain.PolicyIncident{}, err
	}
	policy_incident.Status = domain.PolicyIncidentStatus(status)
	var err error
	if policy_incident.FirstReceiptAt, err = parseTime(firstReceiptAt); err != nil {
		return domain.PolicyIncident{}, err
	}
	if policy_incident.LastReceiptAt, err = parseTime(lastReceiptAt); err != nil {
		return domain.PolicyIncident{}, err
	}
	if policy_incident.ReviewDueAt, err = parseTime(reviewDueAt); err != nil {
		return domain.PolicyIncident{}, err
	}
	if policy_incident.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.PolicyIncident{}, err
	}
	if policy_incident.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.PolicyIncident{}, err
	}
	return policy_incident, nil
}

func (q *queries) GetIdempotency(ctx context.Context, scope, key string) (repository.IdempotencyRecord, error) {
	row := q.q.QueryRowContext(ctx, `SELECT scope, idempotency_key, request_hash, response_code, response_body, expires_at, created_at
        FROM idempotency_records WHERE scope = ? AND idempotency_key = ?`, scope, key)
	var record repository.IdempotencyRecord
	var expiresAt, createdAt string
	if err := row.Scan(&record.Scope, &record.Key, &record.RequestHash, &record.ResponseCode, &record.ResponseBody, &expiresAt, &createdAt); err != nil {
		return repository.IdempotencyRecord{}, translateError("get idempotency record", err)
	}
	var err error
	if record.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return repository.IdempotencyRecord{}, err
	}
	if record.CreatedAt, err = parseTime(createdAt); err != nil {
		return repository.IdempotencyRecord{}, err
	}
	record.ResponseBody = append([]byte(nil), record.ResponseBody...)
	return record, nil
}

func (q *queries) CountTrustZoneExecutionRequestsForBusinessDay(ctx context.Context, trust_zoneID, businessDay string) (int, error) {
	var count int
	err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_requests
        WHERE requester_zone_id = ? AND substr(scheduled_start_at, 1, 10) = ? AND state != 'cancelled'`, trust_zoneID, businessDay).Scan(&count)
	if err != nil {
		return 0, translateError("count trust_zone execution_requests", err)
	}
	return count, nil
}

func (q *queries) ListExecutionRequests(ctx context.Context, filter repository.ExecutionRequestFilter) (repository.ExecutionRequestPage, error) {
	page := filter.Page.Normalize(200)
	where, args := buildExecutionRequestWhere(filter)
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_requests`+where, args...).Scan(&total); err != nil {
		return repository.ExecutionRequestPage{}, translateError("count execution_requests", err)
	}
	sortColumn := runSortColumn(page.Sort)
	direction := " ASC"
	if page.Desc {
		direction = " DESC"
	}
	query := runSelect + where + ` ORDER BY ` + sortColumn + direction + `, id ASC LIMIT ? OFFSET ?`
	rows, err := q.q.QueryContext(ctx, query, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.ExecutionRequestPage{}, translateError("list execution_requests", err)
	}
	defer rows.Close()
	items := make([]domain.ExecutionRequest, 0, page.Limit)
	for rows.Next() {
		run, err := scanExecutionRequest(rows)
		if err != nil {
			return repository.ExecutionRequestPage{}, fmt.Errorf("scan run: %w", err)
		}
		items = append(items, run)
	}
	if err := rows.Err(); err != nil {
		return repository.ExecutionRequestPage{}, fmt.Errorf("iterate execution_requests: %w", err)
	}
	return repository.ExecutionRequestPage{Items: items, Total: total}, nil
}

func buildExecutionRequestWhere(filter repository.ExecutionRequestFilter) (string, []any) {
	clauses := make([]string, 0, 6)
	args := make([]any, 0, 6)
	appendStringFilter := func(column, value string) {
		if value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	appendStringFilter("workspace_id", filter.WorkspaceID)
	appendStringFilter("requester_zone_id", filter.RequesterZoneID)
	appendStringFilter("execution_zone_id", filter.ExecutionZoneID)
	appendStringFilter("state", string(filter.State))
	if filter.From != nil {
		clauses = append(clauses, "scheduled_start_at >= ?")
		args = append(args, formatTime(*filter.From))
	}
	if filter.To != nil {
		clauses = append(clauses, "scheduled_start_at < ?")
		args = append(args, formatTime(*filter.To))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func runSortColumn(value string) string {
	switch value {
	case "expected_finish_at":
		return "expected_finish_at"
	case "updated_at":
		return "updated_at"
	case "request_key":
		return "request_key"
	default:
		return "scheduled_start_at"
	}
}

func (q *queries) ListToolRevisions(ctx context.Context, filter repository.ToolRevisionFilter) (repository.ToolRevisionPage, error) {
	page := filter.Page.Normalize(200)
	where, args := buildToolRevisionWhere(filter)
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM tool_revisions`+where, args...).Scan(&total); err != nil {
		return repository.ToolRevisionPage{}, translateError("count tool_revisions", err)
	}
	query := revisionSelect + where + ` ORDER BY expires_at ASC, id ASC LIMIT ? OFFSET ?`
	rows, err := q.q.QueryContext(ctx, query, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.ToolRevisionPage{}, translateError("list tool_revisions", err)
	}
	defer rows.Close()
	items := make([]domain.ToolRevision, 0, page.Limit)
	for rows.Next() {
		batch, err := scanToolRevision(rows)
		if err != nil {
			return repository.ToolRevisionPage{}, fmt.Errorf("scan revision: %w", err)
		}
		items = append(items, batch.Clone())
	}
	if err := rows.Err(); err != nil {
		return repository.ToolRevisionPage{}, fmt.Errorf("iterate tool_revisions: %w", err)
	}
	return repository.ToolRevisionPage{Items: items, Total: total}, nil
}

func buildToolRevisionWhere(filter repository.ToolRevisionFilter) (string, []any) {
	clauses := make([]string, 0, 5)
	args := make([]any, 0, 5)
	values := []struct{ column, value string }{
		{"workspace_id", filter.WorkspaceID}, {"requester_zone_id", filter.TrustZoneID}, {"request_id", filter.ExecutionRequestID}, {"state", string(filter.State)},
	}
	for _, item := range values {
		if item.value != "" {
			clauses = append(clauses, item.column+" = ?")
			args = append(args, item.value)
		}
	}
	if filter.ExpiresBy != nil {
		clauses = append(clauses, "expires_at <= ?")
		args = append(args, formatTime(*filter.ExpiresBy))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (q *queries) ListPolicyIncidents(ctx context.Context, filter repository.PolicyIncidentFilter) (repository.PolicyIncidentPage, error) {
	page := filter.Page.Normalize(200)
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if filter.ExecutionRequestID != "" {
		clauses = append(clauses, "request_id = ?")
		args = append(args, filter.ExecutionRequestID)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.DueBefore != nil {
		clauses = append(clauses, "review_due_at <= ?")
		args = append(args, formatTime(*filter.DueBefore))
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM policy_incidents`+where, args...).Scan(&total); err != nil {
		return repository.PolicyIncidentPage{}, translateError("count policy_incidents", err)
	}
	rows, err := q.q.QueryContext(ctx, policy_incidentSelect+where+` ORDER BY review_due_at ASC, id ASC LIMIT ? OFFSET ?`, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.PolicyIncidentPage{}, translateError("list policy_incidents", err)
	}
	defer rows.Close()
	items := make([]domain.PolicyIncident, 0, page.Limit)
	for rows.Next() {
		policy_incident, err := scanPolicyIncident(rows)
		if err != nil {
			return repository.PolicyIncidentPage{}, fmt.Errorf("scan policy_incident: %w", err)
		}
		items = append(items, policy_incident)
	}
	if err := rows.Err(); err != nil {
		return repository.PolicyIncidentPage{}, fmt.Errorf("iterate policy_incidents: %w", err)
	}
	return repository.PolicyIncidentPage{Items: items, Total: total}, nil
}

func (q *queries) ListAuditEvents(ctx context.Context, filter repository.AuditFilter) (repository.AuditPage, error) {
	page := filter.Page.Normalize(500)
	clauses := make([]string, 0, 4)
	args := make([]any, 0, 4)
	values := []struct{ column, value string }{
		{"entity_type", filter.EntityType}, {"entity_id", filter.EntityID}, {"actor", filter.Actor}, {"request_id", filter.RequestID},
	}
	for _, item := range values {
		if item.value != "" {
			clauses = append(clauses, item.column+" = ?")
			args = append(args, item.value)
		}
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events`+where, args...).Scan(&total); err != nil {
		return repository.AuditPage{}, translateError("count audit events", err)
	}
	rows, err := q.q.QueryContext(ctx, `SELECT id, request_id, actor, action, entity_type, entity_id, outcome, metadata_json, created_at
        FROM audit_events`+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.AuditPage{}, translateError("list audit events", err)
	}
	defer rows.Close()
	items := make([]domain.AuditEvent, 0, page.Limit)
	for rows.Next() {
		var event domain.AuditEvent
		var metadataJSON, createdAt string
		if err := rows.Scan(&event.ID, &event.RequestID, &event.Actor, &event.Action, &event.EntityType,
			&event.EntityID, &event.Outcome, &metadataJSON, &createdAt); err != nil {
			return repository.AuditPage{}, fmt.Errorf("scan audit event: %w", err)
		}
		metadata, err := decodeMetadata(metadataJSON)
		if err != nil {
			return repository.AuditPage{}, err
		}
		event.Metadata = metadata
		if event.CreatedAt, err = parseTime(createdAt); err != nil {
			return repository.AuditPage{}, err
		}
		items = append(items, event.Clone())
	}
	if err := rows.Err(); err != nil {
		return repository.AuditPage{}, fmt.Errorf("iterate audit events: %w", err)
	}
	return repository.AuditPage{Items: items, Total: total}, nil
}

func beginningOfUTCDate(day string) (time.Time, error) {
	return time.Parse("2006-01-02", day)
}
