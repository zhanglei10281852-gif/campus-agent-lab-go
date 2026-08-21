CREATE INDEX idx_tool_revisions_run_state ON tool_revisions(request_id, state);
CREATE INDEX idx_execution_requests_workspace_created ON execution_requests(workspace_id, created_at);
CREATE INDEX idx_approval_tasks_custodians ON approval_tasks(reviewer_id, status, created_at);
CREATE INDEX idx_jobs_aggregate ON outbox_jobs(kind, aggregate_id, created_at);
