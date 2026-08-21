package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/service"
)

type planExecutionRequestRequest struct {
	WorkspaceID      string    `json:"workspace_id"`
	RequesterZoneID  string    `json:"requester_zone_id"`
	ExecutionZoneID  string    `json:"execution_zone_id"`
	ExecutionPoolID  string    `json:"execution_pool_id"`
	RequestKey       string    `json:"request_key"`
	ToolRevisionIDs  []string  `json:"tool_revision_ids"`
	ScheduledStartAt time.Time `json:"scheduled_start_at"`
	ExpectedFinishAt time.Time `json:"expected_finish_at"`
}

func (s *Server) planExecutionRequest(w http.ResponseWriter, r *http.Request) {
	var input planExecutionRequestRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	run, err := s.services.Execution.PlanExecutionRequest(r.Context(), service.PlanExecutionRequestInput{WorkspaceID: input.WorkspaceID, RequesterZoneID: input.RequesterZoneID, ExecutionZoneID: input.ExecutionZoneID, ExecutionPoolID: input.ExecutionPoolID, RequestKey: input.RequestKey, ToolRevisionIDs: append([]string(nil), input.ToolRevisionIDs...), ScheduledStartAt: input.ScheduledStartAt, ExpectedFinishAt: input.ExpectedFinishAt, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) getExecutionRequest(w http.ResponseWriter, r *http.Request) {
	run, items, err := s.services.Query.ExecutionRequest(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"request": run, "tool_revisions": items})
}

func (s *Server) reconcileExecutionRequest(w http.ResponseWriter, r *http.Request) {
	report, err := s.services.Query.ReconcileExecutionRequest(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) listExecutionRequests(w http.ResponseWriter, r *http.Request) {
	from, err := parseTimeQuery(r.URL.Query().Get("from"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r.URL.Query().Get("to"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	page, err := s.services.Query.ExecutionRequests(r.Context(), repository.ExecutionRequestFilter{Page: parsePage(r), WorkspaceID: r.URL.Query().Get("workspace_id"), RequesterZoneID: r.URL.Query().Get("requester_zone_id"), ExecutionZoneID: r.URL.Query().Get("execution_zone_id"), State: domain.ExecutionRequestState(r.URL.Query().Get("state")), From: from, To: to})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) authorizeExecutionRequest(w http.ResponseWriter, r *http.Request) {
	s.writeExecutionRequestTransition(w, r, s.services.Execution.AuthorizeExecutionRequest)
}
func (s *Server) beginExecutionRequest(w http.ResponseWriter, r *http.Request) {
	s.writeExecutionRequestTransition(w, r, s.services.Execution.BeginExecutionRequest)
}
func (s *Server) completeExecutionRequest(w http.ResponseWriter, r *http.Request) {
	s.writeExecutionRequestTransition(w, r, s.services.Execution.CompleteExecutionRequest)
}
func (s *Server) archiveExecutionRequest(w http.ResponseWriter, r *http.Request) {
	s.writeExecutionRequestTransition(w, r, s.services.Execution.ArchiveExecutionRequest)
}

func (s *Server) writeExecutionRequestTransition(w http.ResponseWriter, r *http.Request, transition func(context.Context, string) (domain.ExecutionRequest, error)) {
	run, err := transition(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) cancelExecutionRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	run, err := s.services.Execution.CancelExecutionRequest(r.Context(), parseID(r), input.Note)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}
