package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type scanner interface {
	Scan(dest ...any) error
}

func (q *queries) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row := q.q.QueryRowContext(ctx, userSelect+` WHERE email = ? COLLATE NOCASE`, strings.TrimSpace(email))
	user, err := scanUser(row)
	return user, translateError("get user by email", err)
}

func (q *queries) GetUser(ctx context.Context, id string) (domain.User, error) {
	user, err := scanUser(q.q.QueryRowContext(ctx, userSelect+` WHERE id = ?`, id))
	return user, translateError("get user", err)
}

func (q *queries) ListUsers(ctx context.Context, filter repository.UserFilter) (repository.UserPage, error) {
	page := filter.Page.Normalize(200)
	clauses := []string{"1=1"}
	args := make([]any, 0, 2)
	if strings.TrimSpace(filter.Email) != "" {
		clauses = append(clauses, "email LIKE ?")
		args = append(args, "%"+strings.TrimSpace(filter.Email)+"%")
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	where := " WHERE " + strings.Join(clauses, " AND ")
	var total int
	if err := q.q.QueryRowContext(ctx, "SELECT COUNT(*) FROM users"+where, args...).Scan(&total); err != nil {
		return repository.UserPage{}, translateError("count users", err)
	}
	rows, err := q.q.QueryContext(ctx, "SELECT id,email,display_name,password_hash,role,status,version,created_at,updated_at FROM users"+where+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?", append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.UserPage{}, translateError("list users", err)
	}
	defer rows.Close()
	items := make([]domain.User, 0)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return repository.UserPage{}, translateError("scan user", scanErr)
		}
		user.PasswordHash = ""
		items = append(items, user)
	}
	if err := rows.Err(); err != nil {
		return repository.UserPage{}, translateError("iterate users", err)
	}
	return repository.UserPage{Items: items, Total: total}, nil
}

const userSelect = `SELECT id, email, display_name, password_hash, role, status, version, created_at, updated_at FROM users`

func scanUser(row scanner) (domain.User, error) {
	var user domain.User
	var role, status, createdAt, updatedAt string
	if err := row.Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &role, &status, &user.Version, &createdAt, &updatedAt); err != nil {
		return domain.User{}, err
	}
	var err error
	if user.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.User{}, err
	}
	if user.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.User{}, err
	}
	user.Role = domain.Role(role)
	user.Status = domain.UserStatus(status)
	return user, nil
}

func (q *queries) GetSessionByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, user_id, token_hash, expires_at, created_at, revoked_at FROM sessions WHERE token_hash = ?`, tokenHash)
	var session domain.Session
	var expiresAt, createdAt string
	var revokedAt sql.NullString
	if err := row.Scan(&session.ID, &session.UserID, &session.TokenHash, &expiresAt, &createdAt, &revokedAt); err != nil {
		return domain.Session{}, translateError("get session", err)
	}
	var err error
	if session.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return domain.Session{}, err
	}
	if session.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Session{}, err
	}
	if session.RevokedAt, err = parseNullableTime(revokedAt); err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (q *queries) GetWorkspace(ctx context.Context, id string) (domain.Workspace, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, code, name, status, minimum_risk_score_millis, maximum_risk_score_millis,
        max_execution_seconds, review_deadline_seconds, business_timezone, version, created_at, updated_at
        FROM workspaces WHERE id = ?`, id)
	var workspace domain.Workspace
	var status, createdAt, updatedAt string
	var maxExecutionSeconds, reviewDeadlineSeconds int64
	if err := row.Scan(&workspace.ID, &workspace.Code, &workspace.Name, &status, &workspace.RiskScore.Minimum,
		&workspace.RiskScore.Maximum, &maxExecutionSeconds, &reviewDeadlineSeconds, &workspace.BusinessTimezone,
		&workspace.Version, &createdAt, &updatedAt); err != nil {
		return domain.Workspace{}, translateError("get workspace", err)
	}
	workspace.Status = domain.WorkspaceStatus(status)
	workspace.MaxExecution = durationSeconds(maxExecutionSeconds)
	workspace.ReviewDeadline = durationSeconds(reviewDeadlineSeconds)
	var err error
	if workspace.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Workspace{}, err
	}
	if workspace.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Workspace{}, err
	}
	return workspace, nil
}

func (q *queries) GetTrustZone(ctx context.Context, id string) (domain.TrustZone, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, code, name, timezone, status, daily_limit, cutoff_hour, version, created_at, updated_at FROM trust_zones WHERE id = ?`, id)
	var trust_zone domain.TrustZone
	var status, createdAt, updatedAt string
	if err := row.Scan(&trust_zone.ID, &trust_zone.Code, &trust_zone.Name, &trust_zone.Timezone, &status, &trust_zone.DailyLimit, &trust_zone.CutoffHour, &trust_zone.Version, &createdAt, &updatedAt); err != nil {
		return domain.TrustZone{}, translateError("get trust_zone", err)
	}
	trust_zone.Status = domain.TrustZoneStatus(status)
	var err error
	if trust_zone.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.TrustZone{}, err
	}
	if trust_zone.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.TrustZone{}, err
	}
	return trust_zone, nil
}

func (q *queries) GetToolRevision(ctx context.Context, id string) (domain.ToolRevision, error) {
	batch, err := scanToolRevision(q.q.QueryRowContext(ctx, revisionSelect+` WHERE id = ?`, id))
	return batch, translateError("get revision batch", err)
}

const revisionSelect = `SELECT id, workspace_id, requester_zone_id, version_tag, protocol_family, operation_count, requested_units,
    state, expires_at, COALESCE(request_id, ''), quarantine_note, version, created_at, updated_at FROM tool_revisions`

func scanToolRevision(row scanner) (domain.ToolRevision, error) {
	var batch domain.ToolRevision
	var state, expiresAt, createdAt, updatedAt string
	if err := row.Scan(&batch.ID, &batch.WorkspaceID, &batch.RequesterZoneID, &batch.VersionTag, &batch.ProtocolFamily,
		&batch.OperationCount, &batch.RequestedUnits, &state, &expiresAt, &batch.ExecutionRequestID, &batch.QuarantineNote,
		&batch.Version, &createdAt, &updatedAt); err != nil {
		return domain.ToolRevision{}, err
	}
	batch.State = domain.ToolRevisionState(state)
	var err error
	if batch.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return domain.ToolRevision{}, err
	}
	if batch.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.ToolRevision{}, err
	}
	if batch.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.ToolRevision{}, err
	}
	return batch, nil
}

func (q *queries) GetExecutionPool(ctx context.Context, id string) (domain.ExecutionPool, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, pool_key, state, capacity_units, attestation_due_at, last_reconciled_at,
        COALESCE(reserved_request_id, ''), version, created_at, updated_at FROM execution_pools WHERE id = ?`, id)
	execution_pool, err := scanExecutionPool(row)
	return execution_pool, translateError("get execution_pool", err)
}

func scanExecutionPool(row scanner) (domain.ExecutionPool, error) {
	var execution_pool domain.ExecutionPool
	var state, attestationDueAt, lastReconciledAt, createdAt, updatedAt string
	if err := row.Scan(&execution_pool.ID, &execution_pool.PoolKey, &state, &execution_pool.CapacityUnits,
		&attestationDueAt, &lastReconciledAt, &execution_pool.ReservedRequestID, &execution_pool.Version, &createdAt, &updatedAt); err != nil {
		return domain.ExecutionPool{}, err
	}
	execution_pool.State = domain.ExecutionPoolState(state)
	var err error
	if execution_pool.AttestationDueAt, err = parseTime(attestationDueAt); err != nil {
		return domain.ExecutionPool{}, err
	}
	if execution_pool.LastReconciledAt, err = parseTime(lastReconciledAt); err != nil {
		return domain.ExecutionPool{}, err
	}
	if execution_pool.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.ExecutionPool{}, err
	}
	if execution_pool.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.ExecutionPool{}, err
	}
	return execution_pool, nil
}

func (q *queries) GetExecutionRequest(ctx context.Context, id string) (domain.ExecutionRequest, error) {
	run, err := scanExecutionRequest(q.q.QueryRowContext(ctx, runSelect+` WHERE id = ?`, id))
	return run, translateError("get run", err)
}

const runSelect = `SELECT id, workspace_id, requester_zone_id, execution_zone_id, execution_pool_id, request_key, state,
    scheduled_start_at, expected_finish_at, started_at, completed_at, archived_at, total_requested_units, version, created_at, updated_at FROM execution_requests`

func scanExecutionRequest(row scanner) (domain.ExecutionRequest, error) {
	var run domain.ExecutionRequest
	var state, scheduledStartAt, expectedFinishAt, createdAt, updatedAt string
	var startedAt, completedAt, archivedAt sql.NullString
	if err := row.Scan(&run.ID, &run.WorkspaceID, &run.RequesterZoneID, &run.ExecutionZoneID,
		&run.ExecutionPoolID, &run.RequestKey, &state, &scheduledStartAt, &expectedFinishAt,
		&startedAt, &completedAt, &archivedAt, &run.TotalRequestedUnits, &run.Version,
		&createdAt, &updatedAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	run.State = domain.ExecutionRequestState(state)
	var err error
	if run.ScheduledStartAt, err = parseTime(scheduledStartAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	if run.ExpectedFinishAt, err = parseTime(expectedFinishAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	if run.StartedAt, err = parseNullableTime(startedAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	if run.CompletedAt, err = parseNullableTime(completedAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	if run.ArchivedAt, err = parseNullableTime(archivedAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	if run.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	if run.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.ExecutionRequest{}, err
	}
	return run, nil
}

func (q *queries) ListExecutionRequestInputs(ctx context.Context, requestID string) ([]domain.ToolRevision, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT tool_revisions.id, tool_revisions.workspace_id, tool_revisions.requester_zone_id,
        tool_revisions.version_tag, tool_revisions.protocol_family, tool_revisions.operation_count, tool_revisions.requested_units,
        tool_revisions.state, tool_revisions.expires_at, COALESCE(tool_revisions.request_id, ''), tool_revisions.quarantine_note,
        tool_revisions.version, tool_revisions.created_at, tool_revisions.updated_at
		FROM tool_revisions JOIN execution_request_tools ri ON ri.tool_revision_id = tool_revisions.id
		WHERE ri.request_id = ? ORDER BY ri.added_at, tool_revisions.id`, requestID)
	if err != nil {
		return nil, translateError("list run items", err)
	}
	defer rows.Close()
	items := make([]domain.ToolRevision, 0)
	for rows.Next() {
		batch, err := scanToolRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run item: %w", err)
		}
		items = append(items, batch.Clone())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run items: %w", err)
	}
	return items, nil
}

func decodeMetadata(raw string) (map[string]string, error) {
	metadata := make(map[string]string)
	if raw == "" || raw == "{}" {
		return metadata, nil
	}
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, fmt.Errorf("decode audit metadata: %w", err)
	}
	return metadata, nil
}
