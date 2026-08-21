PRAGMA foreign_keys = ON;

CREATE TABLE cohorts (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL,
    grade TEXT NOT NULL,
    instructor TEXT NOT NULL,
    workspace_id TEXT REFERENCES workspaces(id),
    capacity INTEGER NOT NULL CHECK(capacity > 0 AND capacity <= 500),
    status TEXT NOT NULL CHECK(status IN ('draft', 'active', 'closed')),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_cohorts_status_grade ON cohorts(status, grade, created_at);
CREATE INDEX idx_cohorts_workspace ON cohorts(workspace_id, status);

CREATE TABLE trainees (
    id TEXT PRIMARY KEY,
    student_no TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL,
    gender TEXT NOT NULL DEFAULT '',
    birth_date TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    cohort_id TEXT NOT NULL REFERENCES cohorts(id),
    status TEXT NOT NULL CHECK(status IN ('active', 'suspended', 'completed')),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_trainees_cohort_status ON trainees(cohort_id, status, created_at);
CREATE INDEX idx_trainees_name ON trainees(name COLLATE NOCASE);

CREATE TABLE trainee_workspace_assignments (
    trainee_id TEXT NOT NULL REFERENCES trainees(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    assigned_by TEXT NOT NULL REFERENCES users(id),
    assigned_at TEXT NOT NULL,
    PRIMARY KEY(trainee_id, workspace_id)
);
CREATE INDEX idx_training_assignment_workspace ON trainee_workspace_assignments(workspace_id, assigned_at);
