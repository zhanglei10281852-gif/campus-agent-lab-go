package httpapi

import (
	"net/http"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
)

type createWorkspaceRequest struct {
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	MinimumRiskScore  float64 `json:"minimum_risk_score"`
	MaximumRiskScore  float64 `json:"maximum_risk_score"`
	MaxExecutionHours int     `json:"max_execution_hours"`
	ReviewHours       int     `json:"review_hours"`
	BusinessTimezone  string  `json:"business_timezone"`
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var input createWorkspaceRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	minimum, err := domain.RiskScoreFromFloat(input.MinimumRiskScore)
	if err != nil {
		writeError(w, r, err)
		return
	}
	maximum, err := domain.RiskScoreFromFloat(input.MaximumRiskScore)
	if err != nil {
		writeError(w, r, err)
		return
	}
	rangeValue, err := domain.NewRiskRange(minimum, maximum)
	if err != nil {
		writeError(w, r, err)
		return
	}
	workspace, err := s.services.Catalog.CreateWorkspace(r.Context(), domain.Workspace{Code: input.Code, Name: input.Name, RiskScore: rangeValue, MaxExecution: time.Duration(input.MaxExecutionHours) * time.Hour, ReviewDeadline: time.Duration(input.ReviewHours) * time.Hour, BusinessTimezone: input.BusinessTimezone})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (s *Server) activateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspace, err := s.services.Catalog.ActivateWorkspace(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

type createTrustZoneRequest struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Timezone   string `json:"timezone"`
	DailyLimit int    `json:"daily_limit"`
	CutoffHour int    `json:"cutoff_hour"`
}

func (s *Server) createTrustZone(w http.ResponseWriter, r *http.Request) {
	var input createTrustZoneRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	trust_zone, err := s.services.Catalog.CreateTrustZone(r.Context(), domain.TrustZone{Code: input.Code, Name: input.Name, Timezone: input.Timezone, DailyLimit: input.DailyLimit, CutoffHour: input.CutoffHour})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, trust_zone)
}

type createExecutionPoolRequest struct {
	PoolKey          string    `json:"pool_key"`
	CapacityUnits    int       `json:"capacity_units"`
	AttestationDueAt time.Time `json:"attestation_due_at"`
	LastReconciledAt time.Time `json:"last_reconciled_at"`
}

func (s *Server) createExecutionPool(w http.ResponseWriter, r *http.Request) {
	var input createExecutionPoolRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	execution_pool, err := s.services.Catalog.CreateExecutionPool(r.Context(), domain.ExecutionPool{PoolKey: input.PoolKey, CapacityUnits: input.CapacityUnits, AttestationDueAt: input.AttestationDueAt, LastReconciledAt: input.LastReconciledAt})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, execution_pool)
}

type registerToolRevisionRequest struct {
	WorkspaceID     string    `json:"workspace_id"`
	RequesterZoneID string    `json:"requester_zone_id"`
	VersionTag      string    `json:"version_tag"`
	ProtocolFamily  string    `json:"protocol_family"`
	OperationCount  int       `json:"operation_count"`
	RequestedUnits  int       `json:"requested_units"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func (s *Server) registerToolRevision(w http.ResponseWriter, r *http.Request) {
	var input registerToolRevisionRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	batch, err := s.services.Catalog.RegisterToolRevision(r.Context(), domain.ToolRevision{WorkspaceID: input.WorkspaceID, RequesterZoneID: input.RequesterZoneID, VersionTag: input.VersionTag, ProtocolFamily: input.ProtocolFamily, OperationCount: input.OperationCount, RequestedUnits: input.RequestedUnits, ExpiresAt: input.ExpiresAt})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, batch)
}

func (s *Server) verifyToolRevision(w http.ResponseWriter, r *http.Request) {
	batch, err := s.services.Catalog.VerifyToolRevision(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}
