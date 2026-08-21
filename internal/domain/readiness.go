package domain

import "time"

type RunReadiness struct {
	ExecutionRequestID            string                `json:"request_id"`
	ExecutionRequestState         ExecutionRequestState `json:"run_state"`
	ExpectedToolRevisionCount     int                   `json:"expected_revision_count"`
	MaterializedToolRevisionCount int                   `json:"executed_revision_count"`
	ApprovedToolRevisionCount     int                   `json:"approved_revision_count"`
	RejectedToolRevisionCount     int                   `json:"rejected_revision_count"`
	QuarantinedCount              int                   `json:"blocked_count"`
	PendingApprovalTask           bool                  `json:"pending_approval_task"`
	OpenPolicyIncident            bool                  `json:"open_policy_incident"`
	LastReceiptAt                 *time.Time            `json:"last_receipt_at,omitempty"`
	Complete                      bool                  `json:"complete"`
	Blockers                      []string              `json:"blockers"`
}

func (r RunReadiness) Clone() RunReadiness {
	clone := r
	clone.Blockers = append([]string(nil), r.Blockers...)
	if r.LastReceiptAt != nil {
		value := *r.LastReceiptAt
		clone.LastReceiptAt = &value
	}
	return clone
}

func (r *RunReadiness) Evaluate() {
	r.Blockers = r.Blockers[:0]
	if r.PendingApprovalTask {
		r.Blockers = append(r.Blockers, "pending approval task")
	}
	if r.OpenPolicyIncident {
		r.Blockers = append(r.Blockers, "open quality policy incident")
	}
	if r.QuarantinedCount > 0 {
		r.Blockers = append(r.Blockers, "blocked revisions require review")
	}
	if r.ExpectedToolRevisionCount == 0 {
		r.Blockers = append(r.Blockers, "run has no tool revisions")
	}
	if r.MaterializedToolRevisionCount < r.ExpectedToolRevisionCount && r.ExecutionRequestState == ExecutionRequestCompleted {
		r.Blockers = append(r.Blockers, "not all revisions are executed")
	}
	r.Complete = len(r.Blockers) == 0 && (r.ExecutionRequestState == ExecutionRequestArchived || r.ExecutionRequestState == ExecutionRequestCompleted)
}
