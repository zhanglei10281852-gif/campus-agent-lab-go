PRAGMA foreign_keys = ON;

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_sessions_user_expiry ON sessions(user_id, expires_at);

CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    minimum_risk_score_millis INTEGER NOT NULL,
    maximum_risk_score_millis INTEGER NOT NULL,
    max_execution_seconds INTEGER NOT NULL,
    review_deadline_seconds INTEGER NOT NULL,
    business_timezone TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE trust_zones (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    status TEXT NOT NULL,
    daily_limit INTEGER NOT NULL,
    cutoff_hour INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE tool_revisions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    requester_zone_id TEXT NOT NULL REFERENCES trust_zones(id),
    version_tag TEXT NOT NULL,
    protocol_family TEXT NOT NULL,
    operation_count INTEGER NOT NULL,
    requested_units INTEGER NOT NULL,
    state TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    request_id TEXT,
    quarantine_note TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(workspace_id, version_tag)
);
CREATE INDEX idx_tool_revisions_state ON tool_revisions(state, expires_at);
CREATE INDEX idx_tool_revisions_trust_zone ON tool_revisions(requester_zone_id, created_at);
CREATE TABLE execution_pools (
    id TEXT PRIMARY KEY,
    pool_key TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL,
    capacity_units INTEGER NOT NULL,
    attestation_due_at TEXT NOT NULL,
    last_reconciled_at TEXT NOT NULL,
    reserved_request_id TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_execution_pools_state_attestation ON execution_pools(state, attestation_due_at);
CREATE TABLE execution_requests (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    requester_zone_id TEXT NOT NULL REFERENCES trust_zones(id),
    execution_zone_id TEXT NOT NULL REFERENCES trust_zones(id),
    execution_pool_id TEXT NOT NULL REFERENCES execution_pools(id),
    request_key TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL,
    scheduled_start_at TEXT NOT NULL,
    expected_finish_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT,
    archived_at TEXT,
    total_requested_units INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_execution_requests_route_window ON execution_requests(requester_zone_id, execution_zone_id, scheduled_start_at);
CREATE INDEX idx_execution_requests_state ON execution_requests(state, expected_finish_at);
CREATE TABLE execution_request_tools (
    request_id TEXT NOT NULL REFERENCES execution_requests(id) ON DELETE CASCADE,
    tool_revision_id TEXT NOT NULL UNIQUE REFERENCES tool_revisions(id),
    added_at TEXT NOT NULL,
    PRIMARY KEY(request_id, tool_revision_id)
);
CREATE TABLE approval_tasks (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES execution_requests(id),
    requester_id TEXT NOT NULL,
    reviewer_id TEXT NOT NULL,
    review_queue TEXT NOT NULL,
    status TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    resolved_at TEXT,
    resolution_note TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_approval_task_pending_run ON approval_tasks(request_id) WHERE status = 'pending';
CREATE INDEX idx_approval_task_expiry ON approval_tasks(status, expires_at);
CREATE TABLE execution_receipts (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES execution_requests(id),
    signal_key TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    risk_score_millis INTEGER NOT NULL,
    recorded_at TEXT NOT NULL,
    received_at TEXT NOT NULL,
    UNIQUE(request_id, signal_key, sequence)
);
CREATE INDEX idx_receipts_run_time ON execution_receipts(request_id, recorded_at);
CREATE TABLE policy_incidents (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES execution_requests(id),
    status TEXT NOT NULL,
    first_receipt_at TEXT NOT NULL,
    last_receipt_at TEXT NOT NULL,
    minimum_risk_score_millis INTEGER NOT NULL,
    maximum_risk_score_millis INTEGER NOT NULL,
    receipt_count INTEGER NOT NULL,
    review_due_at TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_policy_incident_active_run ON policy_incidents(request_id) WHERE status IN ('open', 'reviewing');
CREATE INDEX idx_policy_incident_review_due ON policy_incidents(status, review_due_at);
CREATE TABLE review_decisions (
    id TEXT PRIMARY KEY,
    policy_incident_id TEXT NOT NULL REFERENCES policy_incidents(id),
    security_reviewer TEXT NOT NULL,
    decision TEXT NOT NULL,
    rationale TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    metadata_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_audit_entity ON audit_events(entity_type, entity_id, created_at);
CREATE INDEX idx_audit_request ON audit_events(request_id);
CREATE TABLE idempotency_records (
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_code INTEGER NOT NULL,
    response_body BLOB NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY(scope, idempotency_key)
);
CREATE TABLE outbox_jobs (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload BLOB NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL,
    available_at TEXT NOT NULL,
    locked_at TEXT,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_outbox_claim ON outbox_jobs(status, available_at, created_at);
