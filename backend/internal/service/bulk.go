package service

import (
	"context"
	"errors"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
)

type BulkToolRevisionItem struct {
	Index        int                  `json:"index"`
	ToolRevision *domain.ToolRevision `json:"batch,omitempty"`
	Error        string               `json:"error,omitempty"`
	Code         string               `json:"code"`
}

type BulkToolRevisionResult struct {
	Items     []BulkToolRevisionItem `json:"items"`
	Succeeded int                    `json:"succeeded"`
	Failed    int                    `json:"failed"`
}

func (s *CatalogService) BulkRegisterToolRevisions(ctx context.Context, batches []domain.ToolRevision) (BulkToolRevisionResult, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleAgentDeveloper); err != nil {
		return BulkToolRevisionResult{}, err
	}
	if len(batches) == 0 {
		return BulkToolRevisionResult{}, domain.FieldError{Field: "batches", Message: "at least one batch is required"}
	}
	if len(batches) > 100 {
		return BulkToolRevisionResult{}, domain.FieldError{Field: "batches", Message: "cannot contain more than 100 items"}
	}
	result := BulkToolRevisionResult{Items: make([]BulkToolRevisionItem, 0, len(batches))}
	for index, input := range batches {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		created, err := s.RegisterToolRevision(ctx, input.Clone())
		if err != nil {
			result.Failed++
			result.Items = append(result.Items, BulkToolRevisionItem{Index: index, Error: err.Error(), Code: classifyBulkError(err)})
			continue
		}
		result.Succeeded++
		createdCopy := created.Clone()
		result.Items = append(result.Items, BulkToolRevisionItem{Index: index, ToolRevision: &createdCopy, Code: "created"})
	}
	return result, nil
}

func classifyBulkError(err error) string {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return "invalid"
	case errors.Is(err, domain.ErrConflict):
		return "conflict"
	default:
		return "failed"
	}
}
