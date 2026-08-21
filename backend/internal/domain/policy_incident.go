package domain

import (
	"strings"
	"time"
)

type PolicyIncidentStatus string

const (
	PolicyIncidentOpen      PolicyIncidentStatus = "open"
	PolicyIncidentReviewing PolicyIncidentStatus = "reviewing"
	PolicyIncidentCleared   PolicyIncidentStatus = "cleared"
	PolicyIncidentRejected  PolicyIncidentStatus = "rejected"
)

type ExecutionReceipt struct {
	ID                 string         `json:"id"`
	ExecutionRequestID string         `json:"request_id"`
	SignalKey          string         `json:"signal_key"`
	Sequence           int64          `json:"sequence"`
	RiskScore          MilliRiskScore `json:"risk_score_millis"`
	RecordedAt         time.Time      `json:"recorded_at"`
	ReceivedAt         time.Time      `json:"received_at"`
}

type PolicyIncident struct {
	ID                 string               `json:"id"`
	ExecutionRequestID string               `json:"request_id"`
	Status             PolicyIncidentStatus `json:"status"`
	FirstReceiptAt     time.Time            `json:"first_receipt_at"`
	LastReceiptAt      time.Time            `json:"last_receipt_at"`
	Minimum            MilliRiskScore       `json:"minimum_risk_score_millis"`
	Maximum            MilliRiskScore       `json:"maximum_risk_score_millis"`
	ReceiptCount       int                  `json:"receipt_count"`
	ReviewDueAt        time.Time            `json:"review_due_at"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	Version            int64                `json:"version"`
}

type PolicyDecision struct {
	ID               string
	PolicyIncidentID string
	Reviewer         string
	Decision         PolicyIncidentStatus
	Rationale        string
	CreatedAt        time.Time
}

func (s PolicyIncidentStatus) IsResolved() bool {
	return s == PolicyIncidentCleared || s == PolicyIncidentRejected
}

func (s PolicyIncidentStatus) IsOpen() bool {
	return s == PolicyIncidentOpen || s == PolicyIncidentReviewing
}

func (r ExecutionReceipt) Validate() error {
	if strings.TrimSpace(r.ExecutionRequestID) == "" || strings.TrimSpace(r.SignalKey) == "" {
		return FieldError{Field: "receipt", Message: "run and metric key are required"}
	}
	if r.Sequence < 1 || r.RecordedAt.IsZero() {
		return FieldError{Field: "receipt", Message: "sequence and recorded_at are required"}
	}
	return nil
}

func (e *PolicyIncident) Include(receipt ExecutionReceipt, now time.Time) {
	if e.ReceiptCount == 0 || receipt.RecordedAt.Before(e.FirstReceiptAt) {
		e.FirstReceiptAt = receipt.RecordedAt
	}
	if e.ReceiptCount == 0 || receipt.RecordedAt.After(e.LastReceiptAt) {
		e.LastReceiptAt = receipt.RecordedAt
	}
	if e.ReceiptCount == 0 || receipt.RiskScore < e.Minimum {
		e.Minimum = receipt.RiskScore
	}
	if e.ReceiptCount == 0 || receipt.RiskScore > e.Maximum {
		e.Maximum = receipt.RiskScore
	}
	e.ReceiptCount++
	e.UpdatedAt = now.UTC()
}

func (e *PolicyIncident) StartReview(now time.Time) error {
	if e.Status != PolicyIncidentOpen {
		return TransitionError{Entity: "policy_incident", From: string(e.Status), To: string(PolicyIncidentReviewing)}
	}
	e.Status = PolicyIncidentReviewing
	e.UpdatedAt = now.UTC()
	return nil
}

func (e *PolicyIncident) Decide(decision PolicyIncidentStatus, now time.Time) error {
	if e.Status != PolicyIncidentOpen && e.Status != PolicyIncidentReviewing {
		return TransitionError{Entity: "policy_incident", From: string(e.Status), To: string(decision)}
	}
	if decision != PolicyIncidentCleared && decision != PolicyIncidentRejected {
		return FieldError{Field: "decision", Message: "must be cleared or rejected"}
	}
	e.Status = decision
	e.UpdatedAt = now.UTC()
	return nil
}
