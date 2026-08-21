package domain

import (
	"fmt"
	"math"
)

// MilliRiskScore stores policy risk scores without floating-point persistence error.
type MilliRiskScore int64

func RiskScoreFromFloat(value float64) (MilliRiskScore, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, FieldError{Field: "risk_score", Message: "must be finite"}
	}
	if value < -196 || value > 100 {
		return 0, FieldError{Field: "risk_score", Message: "outside supported range"}
	}
	return MilliRiskScore(math.Round(value * 1000)), nil
}

func (t MilliRiskScore) Float64() float64 { return float64(t) / 1000 }

func (t MilliRiskScore) String() string { return fmt.Sprintf("%.3f", t.Float64()) }

type RiskRange struct {
	Minimum MilliRiskScore `json:"minimum"`
	Maximum MilliRiskScore `json:"maximum"`
}

func NewRiskRange(minimum, maximum MilliRiskScore) (RiskRange, error) {
	if minimum >= maximum {
		return RiskRange{}, FieldError{Field: "risk_score_range", Message: "minimum must be lower than maximum"}
	}
	return RiskRange{Minimum: minimum, Maximum: maximum}, nil
}

func (r RiskRange) Contains(value MilliRiskScore) bool {
	return value >= r.Minimum && value <= r.Maximum
}

func (r RiskRange) Validate() error {
	if r.Minimum >= r.Maximum {
		return FieldError{Field: "risk_score_range", Message: "minimum must be lower than maximum"}
	}
	return nil
}
