package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

func (q *queries) InsertUser(ctx context.Context, user domain.User) error {
	if err := user.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO users(id, email, display_name, password_hash, role, status, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, user.ID, user.Email, user.DisplayName, user.PasswordHash, user.Role,
		user.Status, user.Version, formatTime(user.CreatedAt), formatTime(user.UpdatedAt))
	return translateError("insert user", err)
}

func (q *queries) UpdateUser(ctx context.Context, user domain.User, expectedVersion int64) error {
	if err := user.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE users SET email=?, display_name=?, password_hash=?, role=?, status=?,
		version=version+1, updated_at=? WHERE id=? AND version=?`,
		user.Email, user.DisplayName, user.PasswordHash, user.Role, user.Status,
		formatTime(user.UpdatedAt), user.ID, expectedVersion)
	if err != nil {
		return translateError("update user", err)
	}
	return expectVersion(result, "update user")
}

func (q *queries) DeleteUser(ctx context.Context, id string, expectedVersion int64) error {
	result, err := q.q.ExecContext(ctx, "DELETE FROM users WHERE id=? AND version=?", id, expectedVersion)
	if err != nil {
		return translateError("delete user", err)
	}
	return expectVersion(result, "delete user")
}

func (q *queries) InsertSession(ctx context.Context, session domain.Session) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO sessions(id, user_id, token_hash, expires_at, created_at, revoked_at)
        VALUES(?, ?, ?, ?, ?, NULL)`, session.ID, session.UserID, session.TokenHash, formatTime(session.ExpiresAt), formatTime(session.CreatedAt))
	return translateError("insert session", err)
}

func (q *queries) RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error {
	result, err := q.q.ExecContext(ctx, `UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, formatTime(revokedAt), sessionID)
	if err != nil {
		return translateError("revoke session", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke session rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("revoke session: %w", domain.ErrNotFound)
	}
	return nil
}

func (q *queries) InsertWorkspace(ctx context.Context, workspace domain.Workspace) error {
	if err := workspace.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO workspaces(id, code, name, status, minimum_risk_score_millis, maximum_risk_score_millis,
        max_execution_seconds, review_deadline_seconds, business_timezone, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, workspace.ID, workspace.Code, workspace.Name, workspace.Status,
		workspace.RiskScore.Minimum, workspace.RiskScore.Maximum, int64(workspace.MaxExecution/time.Second),
		int64(workspace.ReviewDeadline/time.Second), workspace.BusinessTimezone, workspace.Version,
		formatTime(workspace.CreatedAt), formatTime(workspace.UpdatedAt))
	return translateError("insert workspace", err)
}

func (q *queries) UpdateWorkspace(ctx context.Context, workspace domain.Workspace, expectedVersion int64) error {
	if err := workspace.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE workspaces SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, workspace.Status, formatTime(workspace.UpdatedAt), workspace.ID, expectedVersion)
	if err != nil {
		return translateError("update workspace", err)
	}
	return expectVersion(result, "update workspace")
}

func (q *queries) InsertTrustZone(ctx context.Context, trust_zone domain.TrustZone) error {
	if err := trust_zone.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO trust_zones(id, code, name, timezone, status, daily_limit, cutoff_hour, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, trust_zone.ID, trust_zone.Code, trust_zone.Name, trust_zone.Timezone, trust_zone.Status,
		trust_zone.DailyLimit, trust_zone.CutoffHour, trust_zone.Version, formatTime(trust_zone.CreatedAt), formatTime(trust_zone.UpdatedAt))
	return translateError("insert trust_zone", err)
}

func (q *queries) InsertToolRevision(ctx context.Context, batch domain.ToolRevision) error {
	if err := batch.Validate(); err != nil {
		return err
	}
	requestID := nullableString(batch.ExecutionRequestID)
	_, err := q.q.ExecContext(ctx, `INSERT INTO tool_revisions(id, workspace_id, requester_zone_id, version_tag, protocol_family,
        operation_count, requested_units, state, expires_at, request_id, quarantine_note, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, batch.ID, batch.WorkspaceID, batch.RequesterZoneID,
		batch.VersionTag, batch.ProtocolFamily, batch.OperationCount, batch.RequestedUnits, batch.State,
		formatTime(batch.ExpiresAt), requestID, batch.QuarantineNote, batch.Version,
		formatTime(batch.CreatedAt), formatTime(batch.UpdatedAt))
	return translateError("insert revision batch", err)
}

func (q *queries) UpdateToolRevision(ctx context.Context, batch domain.ToolRevision, expectedVersion int64) error {
	if err := batch.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE tool_revisions SET state = ?, request_id = ?, quarantine_note = ?,
        expires_at = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, batch.State,
		nullableString(batch.ExecutionRequestID), batch.QuarantineNote, formatTime(batch.ExpiresAt), formatTime(batch.UpdatedAt),
		batch.ID, expectedVersion)
	if err != nil {
		return translateError("update revision batch", err)
	}
	return expectVersion(result, "update revision batch")
}

func (q *queries) InsertExecutionPool(ctx context.Context, execution_pool domain.ExecutionPool) error {
	if err := execution_pool.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO execution_pools(id, pool_key, state, capacity_units, attestation_due_at,
        last_reconciled_at, reserved_request_id, version, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		execution_pool.ID, execution_pool.PoolKey, execution_pool.State, execution_pool.CapacityUnits,
		formatTime(execution_pool.AttestationDueAt), formatTime(execution_pool.LastReconciledAt), nullableString(execution_pool.ReservedRequestID),
		execution_pool.Version, formatTime(execution_pool.CreatedAt), formatTime(execution_pool.UpdatedAt))
	return translateError("insert execution_pool", err)
}

func (q *queries) UpdateExecutionPool(ctx context.Context, execution_pool domain.ExecutionPool, expectedVersion int64) error {
	if err := execution_pool.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE execution_pools SET state = ?, capacity_units = ?, attestation_due_at = ?,
        last_reconciled_at = ?, reserved_request_id = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`,
		execution_pool.State, execution_pool.CapacityUnits, formatTime(execution_pool.AttestationDueAt), formatTime(execution_pool.LastReconciledAt),
		nullableString(execution_pool.ReservedRequestID), formatTime(execution_pool.UpdatedAt), execution_pool.ID, expectedVersion)
	if err != nil {
		return translateError("update execution_pool", err)
	}
	return expectVersion(result, "update execution_pool")
}

func (q *queries) InsertExecutionRequest(ctx context.Context, run domain.ExecutionRequest) error {
	if err := run.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO execution_requests(id, workspace_id, requester_zone_id, execution_zone_id, execution_pool_id,
        request_key, state, scheduled_start_at, expected_finish_at, started_at, completed_at, archived_at,
        total_requested_units, version, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.WorkspaceID, run.RequesterZoneID, run.ExecutionZoneID, run.ExecutionPoolID,
		run.RequestKey, run.State, formatTime(run.ScheduledStartAt), formatTime(run.ExpectedFinishAt),
		nullableTime(run.StartedAt), nullableTime(run.CompletedAt), nullableTime(run.ArchivedAt),
		run.TotalRequestedUnits, run.Version, formatTime(run.CreatedAt), formatTime(run.UpdatedAt))
	return translateError("insert run", err)
}

func (q *queries) UpdateExecutionRequest(ctx context.Context, run domain.ExecutionRequest, expectedVersion int64) error {
	if err := run.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE execution_requests SET state = ?, started_at = ?, completed_at = ?, archived_at = ?,
        version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, run.State,
		nullableTime(run.StartedAt), nullableTime(run.CompletedAt), nullableTime(run.ArchivedAt),
		formatTime(run.UpdatedAt), run.ID, expectedVersion)
	if err != nil {
		return translateError("update run", err)
	}
	return expectVersion(result, "update run")
}

func (q *queries) InsertExecutionRequestInput(ctx context.Context, item domain.ExecutionRequestInput) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO execution_request_tools(request_id, tool_revision_id, added_at) VALUES(?, ?, ?)`,
		item.ExecutionRequestID, item.ToolRevisionID, formatTime(item.AddedAt))
	return translateError("insert run item", err)
}

func (q *queries) InsertApprovalTask(ctx context.Context, approval_task domain.ApprovalTask) error {
	if err := approval_task.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO approval_tasks(id, request_id, requester_id, reviewer_id,
        review_queue, status, expires_at, resolved_at, resolution_note, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, approval_task.ID, approval_task.ExecutionRequestID, approval_task.RequesterID,
		approval_task.ReviewerID, approval_task.ReviewQueue, approval_task.Status, formatTime(approval_task.ExpiresAt), nullableTime(approval_task.ResolvedAt),
		approval_task.ResolutionNote, approval_task.Version, formatTime(approval_task.CreatedAt), formatTime(approval_task.UpdatedAt))
	return translateError("insert approval_task", err)
}

func (q *queries) UpdateApprovalTask(ctx context.Context, approval_task domain.ApprovalTask, expectedVersion int64) error {
	if err := approval_task.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE approval_tasks SET status = ?, resolved_at = ?, resolution_note = ?,
        version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, approval_task.Status,
		nullableTime(approval_task.ResolvedAt), approval_task.ResolutionNote, formatTime(approval_task.UpdatedAt), approval_task.ID, expectedVersion)
	if err != nil {
		return translateError("update approval_task", err)
	}
	return expectVersion(result, "update approval_task")
}

func (q *queries) InsertReceipt(ctx context.Context, receipt domain.ExecutionReceipt) error {
	if err := receipt.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO execution_receipts(id, request_id, signal_key, sequence,
        risk_score_millis, recorded_at, received_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, receipt.ID,
		receipt.ExecutionRequestID, receipt.SignalKey, receipt.Sequence, receipt.RiskScore,
		formatTime(receipt.RecordedAt), formatTime(receipt.ReceivedAt))
	return translateError("insert risk_score receipt", err)
}

func (q *queries) InsertPolicyIncident(ctx context.Context, policy_incident domain.PolicyIncident) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO policy_incidents(id, request_id, status, first_receipt_at, last_receipt_at,
        minimum_risk_score_millis, maximum_risk_score_millis, receipt_count, review_due_at, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, policy_incident.ID, policy_incident.ExecutionRequestID, policy_incident.Status,
		formatTime(policy_incident.FirstReceiptAt), formatTime(policy_incident.LastReceiptAt), policy_incident.Minimum, policy_incident.Maximum,
		policy_incident.ReceiptCount, formatTime(policy_incident.ReviewDueAt), policy_incident.Version,
		formatTime(policy_incident.CreatedAt), formatTime(policy_incident.UpdatedAt))
	return translateError("insert policy_incident", err)
}

func (q *queries) UpdatePolicyIncident(ctx context.Context, policy_incident domain.PolicyIncident, expectedVersion int64) error {
	result, err := q.q.ExecContext(ctx, `UPDATE policy_incidents SET status = ?, first_receipt_at = ?, last_receipt_at = ?,
        minimum_risk_score_millis = ?, maximum_risk_score_millis = ?, receipt_count = ?, review_due_at = ?,
        version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, policy_incident.Status,
		formatTime(policy_incident.FirstReceiptAt), formatTime(policy_incident.LastReceiptAt), policy_incident.Minimum, policy_incident.Maximum,
		policy_incident.ReceiptCount, formatTime(policy_incident.ReviewDueAt), formatTime(policy_incident.UpdatedAt), policy_incident.ID, expectedVersion)
	if err != nil {
		return translateError("update policy_incident", err)
	}
	return expectVersion(result, "update policy_incident")
}

func (q *queries) InsertPolicyDecision(ctx context.Context, decision domain.PolicyDecision) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO review_decisions(id, policy_incident_id, security_reviewer, decision, rationale, created_at)
        VALUES(?, ?, ?, ?, ?, ?)`, decision.ID, decision.PolicyIncidentID, decision.Reviewer, decision.Decision,
		decision.Rationale, formatTime(decision.CreatedAt))
	return translateError("insert review decision", err)
}

func (q *queries) InsertAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	_, err = q.q.ExecContext(ctx, `INSERT INTO audit_events(id, request_id, actor, action, entity_type, entity_id,
        outcome, metadata_json, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, event.ID, event.RequestID,
		event.Actor, event.Action, event.EntityType, event.EntityID, event.Outcome, string(metadata), formatTime(event.CreatedAt))
	return translateError("insert audit event", err)
}

func (q *queries) PutIdempotency(ctx context.Context, record repository.IdempotencyRecord) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO idempotency_records(scope, idempotency_key, request_hash,
        response_code, response_body, expires_at, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, record.Scope,
		record.Key, record.RequestHash, record.ResponseCode, append([]byte(nil), record.ResponseBody...),
		formatTime(record.ExpiresAt), formatTime(record.CreatedAt))
	return translateError("put idempotency record", err)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}
