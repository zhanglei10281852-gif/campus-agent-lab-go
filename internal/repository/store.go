package repository

import (
	"context"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
)

type Store interface {
	WithTx(ctx context.Context, fn func(Tx) error) error
	Read(ctx context.Context, fn func(Reader) error) error
	Ping(ctx context.Context) error
	Close() error
}

type Reader interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUser(ctx context.Context, id string) (domain.User, error)
	ListUsers(ctx context.Context, filter UserFilter) (UserPage, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error)
	GetWorkspace(ctx context.Context, id string) (domain.Workspace, error)
	GetTrustZone(ctx context.Context, id string) (domain.TrustZone, error)
	GetToolRevision(ctx context.Context, id string) (domain.ToolRevision, error)
	GetExecutionPool(ctx context.Context, id string) (domain.ExecutionPool, error)
	GetExecutionRequest(ctx context.Context, id string) (domain.ExecutionRequest, error)
	ListExecutionRequestInputs(ctx context.Context, requestID string) ([]domain.ToolRevision, error)
	GetPendingApprovalTask(ctx context.Context, requestID string) (domain.ApprovalTask, error)
	GetApprovalTask(ctx context.Context, id string) (domain.ApprovalTask, error)
	GetActivePolicyIncident(ctx context.Context, requestID string) (domain.PolicyIncident, error)
	GetPolicyIncident(ctx context.Context, id string) (domain.PolicyIncident, error)
	GetRunReadiness(ctx context.Context, requestID string) (domain.RunReadiness, error)
	GetPlatformSummary(ctx context.Context) (PlatformSummary, error)
	ListExecutionRequests(ctx context.Context, filter ExecutionRequestFilter) (ExecutionRequestPage, error)
	ListToolRevisions(ctx context.Context, filter ToolRevisionFilter) (ToolRevisionPage, error)
	ListPolicyIncidents(ctx context.Context, filter PolicyIncidentFilter) (PolicyIncidentPage, error)
	ListAuditEvents(ctx context.Context, filter AuditFilter) (AuditPage, error)
	GetIdempotency(ctx context.Context, scope, key string) (IdempotencyRecord, error)
	CountTrustZoneExecutionRequestsForBusinessDay(ctx context.Context, trust_zoneID, businessDay string) (int, error)
}

type Tx interface {
	Reader
	InsertUser(ctx context.Context, user domain.User) error
	UpdateUser(ctx context.Context, user domain.User, expectedVersion int64) error
	DeleteUser(ctx context.Context, id string, expectedVersion int64) error
	InsertSession(ctx context.Context, session domain.Session) error
	RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error
	InsertWorkspace(ctx context.Context, workspace domain.Workspace) error
	UpdateWorkspace(ctx context.Context, workspace domain.Workspace, expectedVersion int64) error
	InsertTrustZone(ctx context.Context, trust_zone domain.TrustZone) error
	InsertToolRevision(ctx context.Context, batch domain.ToolRevision) error
	UpdateToolRevision(ctx context.Context, batch domain.ToolRevision, expectedVersion int64) error
	InsertExecutionPool(ctx context.Context, execution_pool domain.ExecutionPool) error
	UpdateExecutionPool(ctx context.Context, execution_pool domain.ExecutionPool, expectedVersion int64) error
	InsertExecutionRequest(ctx context.Context, run domain.ExecutionRequest) error
	UpdateExecutionRequest(ctx context.Context, run domain.ExecutionRequest, expectedVersion int64) error
	InsertExecutionRequestInput(ctx context.Context, item domain.ExecutionRequestInput) error
	InsertApprovalTask(ctx context.Context, approval_task domain.ApprovalTask) error
	UpdateApprovalTask(ctx context.Context, approval_task domain.ApprovalTask, expectedVersion int64) error
	InsertReceipt(ctx context.Context, receipt domain.ExecutionReceipt) error
	InsertPolicyIncident(ctx context.Context, policy_incident domain.PolicyIncident) error
	UpdatePolicyIncident(ctx context.Context, policy_incident domain.PolicyIncident, expectedVersion int64) error
	InsertPolicyDecision(ctx context.Context, decision domain.PolicyDecision) error
	InsertAuditEvent(ctx context.Context, event domain.AuditEvent) error
	PutIdempotency(ctx context.Context, record IdempotencyRecord) error
	InsertJob(ctx context.Context, job domain.OutboxJob) error
	ClaimJobs(ctx context.Context, now time.Time, limit int) ([]domain.OutboxJob, error)
	CompleteJob(ctx context.Context, id string, now time.Time) error
	RetryJob(ctx context.Context, id string, availableAt time.Time, lastError string, dead bool) error
	ExpireApprovalTasks(ctx context.Context, now time.Time, limit int) ([]domain.ApprovalTask, error)
}

type PageRequest struct {
	Limit  int
	Offset int
	Sort   string
	Desc   bool
}

func (p PageRequest) Normalize(max int) PageRequest {
	if p.Limit < 1 {
		p.Limit = 50
	}
	if p.Limit > max {
		p.Limit = max
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

type ExecutionRequestFilter struct {
	Page            PageRequest
	WorkspaceID     string
	RequesterZoneID string
	ExecutionZoneID string
	State           domain.ExecutionRequestState
	From            *time.Time
	To              *time.Time
}

type ExecutionRequestPage struct {
	Items []domain.ExecutionRequest
	Total int
}

type ToolRevisionFilter struct {
	Page               PageRequest
	WorkspaceID        string
	TrustZoneID        string
	ExecutionRequestID string
	State              domain.ToolRevisionState
	ExpiresBy          *time.Time
}

type ToolRevisionPage struct {
	Items []domain.ToolRevision
	Total int
}

type PolicyIncidentFilter struct {
	Page               PageRequest
	ExecutionRequestID string
	Status             domain.PolicyIncidentStatus
	DueBefore          *time.Time
}

type PolicyIncidentPage struct {
	Items []domain.PolicyIncident
	Total int
}

type AuditFilter struct {
	Page       PageRequest
	EntityType string
	EntityID   string
	Actor      string
	RequestID  string
}

type AuditPage struct {
	Items []domain.AuditEvent
	Total int
}

type UserFilter struct {
	Page   PageRequest
	Email  string
	Status domain.UserStatus
}

type UserPage struct {
	Items []domain.User
	Total int
}

type PlatformSummary struct {
	WorkspacesActive           int `json:"workspaces_active"`
	ToolRevisionsValidated     int `json:"tool_revisions_verified"`
	ToolRevisionsMaterializing int `json:"tool_revisions_executing"`
	ToolRevisionsQuarantined   int `json:"tool_revisions_blocked"`
	ExecutionPoolsAvailable    int `json:"execution_pools_available"`
	ExecutionRequestsActive    int `json:"execution_requests_active"`
	OpenPolicyIncidents        int `json:"open_policy_incidents"`
	PendingApprovalTasks       int `json:"pending_approval_tasks"`
	FailedJobs                 int `json:"failed_jobs"`
}

type IdempotencyRecord struct {
	Scope        string
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
