package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

type ReviewService struct{ dependencies }

func (s *ReviewService) StartReview(ctx context.Context, policy_incidentID string) (domain.PolicyIncident, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleSecurityReviewer); err != nil {
		return domain.PolicyIncident{}, err
	}
	var result domain.PolicyIncident
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		policy_incident, err := tx.GetPolicyIncident(ctx, policy_incidentID)
		if err != nil {
			return err
		}
		before := policy_incident.Version
		if err := policy_incident.StartReview(s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdatePolicyIncident(ctx, policy_incident, before); err != nil {
			return err
		}
		result = policy_incident
		return s.audit.Record(ctx, tx, "policy_incident_review_started", "policy_incident", policy_incident.ID, "success", nil)
	})
	return result, err
}

type DecideInput struct {
	PolicyIncidentID string
	Decision         domain.PolicyIncidentStatus
	Rationale        string
}

func (s *ReviewService) Decide(ctx context.Context, input DecideInput) (domain.PolicyIncident, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleSecurityReviewer); err != nil {
		return domain.PolicyIncident{}, err
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return domain.PolicyIncident{}, domain.FieldError{Field: "rationale", Message: "is required"}
	}
	var result domain.PolicyIncident
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		policy_incident, err := tx.GetPolicyIncident(ctx, input.PolicyIncidentID)
		if err != nil {
			return err
		}
		before := policy_incident.Version
		if err := policy_incident.Decide(input.Decision, s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdatePolicyIncident(ctx, policy_incident, before); err != nil {
			return err
		}
		run, err := tx.GetExecutionRequest(ctx, policy_incident.ExecutionRequestID)
		if err != nil {
			return err
		}
		items, err := tx.ListExecutionRequestInputs(ctx, run.ID)
		if err != nil {
			return err
		}
		now := s.clock.Now()
		for _, batch := range items {
			switch input.Decision {
			case domain.PolicyIncidentCleared:
				if batch.State != domain.ToolRevisionBlocked {
					continue
				}
				batch.State = domain.ToolRevisionApproved
				batch.QuarantineNote = ""
			case domain.PolicyIncidentRejected:
				if batch.State != domain.ToolRevisionBlocked {
					continue
				}
				batch.State = domain.ToolRevisionRejected
				batch.QuarantineNote = strings.TrimSpace(input.Rationale)
			default:
				return fmt.Errorf("unsupported review decision: %w", domain.ErrValidation)
			}
			batch.UpdatedAt = now
			if err := tx.UpdateToolRevision(ctx, batch, batch.Version); err != nil {
				return err
			}
		}
		decision := domain.PolicyDecision{ID: identity.New("decision"), PolicyIncidentID: policy_incident.ID, Reviewer: principal.UserID, Decision: input.Decision, Rationale: strings.TrimSpace(input.Rationale), CreatedAt: now}
		if err := tx.InsertPolicyDecision(ctx, decision); err != nil {
			return err
		}
		result = policy_incident
		return s.audit.Record(ctx, tx, "policy_incident_decided", "policy_incident", policy_incident.ID, "success", map[string]string{"decision": string(input.Decision)})
	})
	return result, err
}

func (s *ReviewService) EnsureReviewable(ctx context.Context, policy_incidentID string) error {
	return s.store.Read(ctx, func(reader repository.Reader) error {
		policy_incident, err := reader.GetPolicyIncident(ctx, policy_incidentID)
		if err != nil {
			return err
		}
		if policy_incident.Status != domain.PolicyIncidentOpen && policy_incident.Status != domain.PolicyIncidentReviewing {
			return errors.New("policy_incident is already decided")
		}
		return nil
	})
}
