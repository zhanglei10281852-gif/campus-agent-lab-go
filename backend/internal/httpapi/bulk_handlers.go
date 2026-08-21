package httpapi

import (
	"net/http"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
)

type bulkRegisterRequest struct {
	ToolRevisions []registerToolRevisionRequest `json:"revisions"`
}

func (s *Server) bulkRegisterToolRevisions(w http.ResponseWriter, r *http.Request) {
	var input bulkRegisterRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	batches := make([]domain.ToolRevision, 0, len(input.ToolRevisions))
	for _, item := range input.ToolRevisions {
		batches = append(batches, domain.ToolRevision{WorkspaceID: item.WorkspaceID, RequesterZoneID: item.RequesterZoneID, VersionTag: item.VersionTag, ProtocolFamily: item.ProtocolFamily, OperationCount: item.OperationCount, RequestedUnits: item.RequestedUnits, ExpiresAt: item.ExpiresAt})
	}
	result, err := s.services.Catalog.BulkRegisterToolRevisions(r.Context(), batches)
	if err != nil {
		writeError(w, r, err)
		return
	}
	status := http.StatusCreated
	if result.Failed > 0 {
		status = http.StatusMultiStatus
	}
	writeJSON(w, status, result)
}

func (s *Server) startExecutionPoolReconciling(w http.ResponseWriter, r *http.Request) {
	execution_pool, err := s.services.ExecutionPools.StartReconciliation(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, execution_pool)
}

func (s *Server) completeExecutionPoolReconciling(w http.ResponseWriter, r *http.Request) {
	execution_pool, err := s.services.ExecutionPools.CompleteReconciliation(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, execution_pool)
}

func (s *Server) retireExecutionPool(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	execution_pool, err := s.services.ExecutionPools.Retire(r.Context(), parseID(r), input.Reason)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, execution_pool)
}
