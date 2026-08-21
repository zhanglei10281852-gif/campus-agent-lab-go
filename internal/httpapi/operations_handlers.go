package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/service"
)

type createApprovalTaskRequest struct {
	RequesterID string `json:"requester_id"`
	ReviewerID  string `json:"reviewer_id"`
	ReviewQueue string `json:"review_queue"`
}

func (s *Server) createApprovalTask(w http.ResponseWriter, r *http.Request) {
	var input createApprovalTaskRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	approval_task, err := s.services.Approval.CreateApprovalTask(r.Context(), service.CreateApprovalTaskInput{ExecutionRequestID: parseID(r), RequesterID: input.RequesterID, ReviewerID: input.ReviewerID, ReviewQueue: input.ReviewQueue})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, approval_task)
}

type resolveApprovalTaskRequest struct {
	Accepted bool   `json:"accepted"`
	Note     string `json:"note"`
}

func (s *Server) resolveApprovalTask(w http.ResponseWriter, r *http.Request) {
	var input resolveApprovalTaskRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	approval_task, err := s.services.Approval.ResolveApprovalTask(r.Context(), parseID(r), input.Accepted, input.Note)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, approval_task)
}

type receiptRequest struct {
	SignalKey      string    `json:"signal_key"`
	Sequence       int64     `json:"sequence"`
	RiskScoreValue float64   `json:"risk_score"`
	RecordedAt     time.Time `json:"recorded_at"`
}

func (s *Server) recordReceipt(w http.ResponseWriter, r *http.Request) {
	var input receiptRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	risk_score, err := domain.RiskScoreFromFloat(input.RiskScoreValue)
	if err != nil {
		writeError(w, r, err)
		return
	}
	receipt, policy_incident, err := s.services.Receipts.RecordReceipt(r.Context(), service.RecordReceiptInput{ExecutionRequestID: parseID(r), SignalKey: input.SignalKey, Sequence: input.Sequence, RiskScore: risk_score, RecordedAt: input.RecordedAt})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"receipt": receipt, "policy_incident": policy_incident})
}

func (s *Server) listPolicyIncidents(w http.ResponseWriter, r *http.Request) {
	dueBefore, err := parseTimeQuery(r.URL.Query().Get("due_before"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	page, err := s.services.Query.PolicyIncidents(r.Context(), repository.PolicyIncidentFilter{Page: parsePage(r), ExecutionRequestID: r.URL.Query().Get("request_id"), Status: domain.PolicyIncidentStatus(r.URL.Query().Get("status")), DueBefore: dueBefore})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) startReview(w http.ResponseWriter, r *http.Request) {
	policy_incident, err := s.services.Review.StartReview(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, policy_incident)
}

type decisionRequest struct {
	Decision  domain.PolicyIncidentStatus `json:"decision"`
	Rationale string                      `json:"rationale"`
}

func (s *Server) decidePolicyIncident(w http.ResponseWriter, r *http.Request) {
	var input decisionRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	policy_incident, err := s.services.Review.Decide(r.Context(), service.DecideInput{PolicyIncidentID: parseID(r), Decision: input.Decision, Rationale: input.Rationale})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, policy_incident)
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	page, err := s.services.Query.Audit(r.Context(), repository.AuditFilter{Page: parsePage(r), EntityType: r.URL.Query().Get("entity_type"), EntityID: r.URL.Query().Get("entity_id"), Actor: r.URL.Query().Get("actor"), RequestID: r.URL.Query().Get("request_id")})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.services.Query.PlatformSummary(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func queryInt(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}
