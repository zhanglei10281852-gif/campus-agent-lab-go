package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/requestmeta"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/service"
)

type cohortRequest struct {
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Grade       string              `json:"grade"`
	Instructor  string              `json:"instructor"`
	WorkspaceID string              `json:"workspace_id"`
	Capacity    int                 `json:"capacity"`
	Status      domain.CohortStatus `json:"status"`
	Version     int64               `json:"version"`
}

func (s *Server) listCohorts(w http.ResponseWriter, r *http.Request) {
	page, current, size := trainingPage(r)
	items, total, err := s.services.Training.ListCohorts(r.Context(), page, r.URL.Query().Get("search"), domain.CohortStatus(r.URL.Query().Get("status")))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": items, "total": total, "current": current, "size": size})
}

func (s *Server) listAllCohorts(w http.ResponseWriter, r *http.Request) {
	items, err := s.services.Training.ListAllCohorts(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createCohort(w http.ResponseWriter, r *http.Request) {
	var input cohortRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	now := time.Now().UTC()
	value := domain.Cohort{ID: identity.New("cohort"), Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name),
		Grade: strings.TrimSpace(input.Grade), Instructor: strings.TrimSpace(input.Instructor), WorkspaceID: strings.TrimSpace(input.WorkspaceID),
		Capacity: input.Capacity, Status: input.Status, Version: 1, CreatedAt: now, UpdatedAt: now}
	if value.Status == "" {
		value.Status = domain.CohortActive
	}
	result, err := s.services.Training.CreateCohort(r.Context(), principal, value)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) updateCohort(w http.ResponseWriter, r *http.Request) {
	var input cohortRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	current, err := s.services.Training.GetCohort(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	current.Code = strings.TrimSpace(input.Code)
	current.Name = strings.TrimSpace(input.Name)
	current.Grade = strings.TrimSpace(input.Grade)
	current.Instructor = strings.TrimSpace(input.Instructor)
	current.WorkspaceID = strings.TrimSpace(input.WorkspaceID)
	current.Capacity = input.Capacity
	current.Status = input.Status
	current.UpdatedAt = time.Now().UTC()
	result, err := s.services.Training.UpdateCohort(r.Context(), principal, current, input.Version)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) deleteCohort(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, r, domain.FieldError{Field: "version", Message: "must be an integer"})
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	if err := s.services.Training.DeleteCohort(r.Context(), principal, parseID(r), version); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type traineeRequest struct {
	StudentNo string               `json:"student_no"`
	Name      string               `json:"name"`
	Gender    string               `json:"gender"`
	BirthDate string               `json:"birth_date"`
	Phone     string               `json:"phone"`
	Email     string               `json:"email"`
	CohortID  string               `json:"cohort_id"`
	Status    domain.TraineeStatus `json:"status"`
	Version   int64                `json:"version"`
}

func (s *Server) listTrainees(w http.ResponseWriter, r *http.Request) {
	page, current, size := trainingPage(r)
	items, total, err := s.services.Training.ListTrainees(r.Context(), page, r.URL.Query().Get("name"), r.URL.Query().Get("student_no"),
		r.URL.Query().Get("cohort_id"), domain.TraineeStatus(r.URL.Query().Get("status")))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": items, "total": total, "current": current, "size": size})
}

func (s *Server) getTrainee(w http.ResponseWriter, r *http.Request) {
	value, err := s.services.Training.GetTrainee(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) createTrainee(w http.ResponseWriter, r *http.Request) {
	var input traineeRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	now := time.Now().UTC()
	value := domain.Trainee{ID: identity.New("trainee"), StudentNo: strings.TrimSpace(input.StudentNo), Name: strings.TrimSpace(input.Name),
		Gender: strings.TrimSpace(input.Gender), BirthDate: strings.TrimSpace(input.BirthDate), Phone: strings.TrimSpace(input.Phone),
		Email: strings.TrimSpace(strings.ToLower(input.Email)), CohortID: strings.TrimSpace(input.CohortID), Status: input.Status,
		Version: 1, CreatedAt: now, UpdatedAt: now}
	if value.Status == "" {
		value.Status = domain.TraineeActive
	}
	result, err := s.services.Training.CreateTrainee(r.Context(), principal, value)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) updateTrainee(w http.ResponseWriter, r *http.Request) {
	var input traineeRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	current, err := s.services.Training.GetTrainee(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	current.StudentNo = strings.TrimSpace(input.StudentNo)
	current.Name = strings.TrimSpace(input.Name)
	current.Gender = strings.TrimSpace(input.Gender)
	current.BirthDate = strings.TrimSpace(input.BirthDate)
	current.Phone = strings.TrimSpace(input.Phone)
	current.Email = strings.TrimSpace(strings.ToLower(input.Email))
	current.CohortID = strings.TrimSpace(input.CohortID)
	current.Status = input.Status
	current.UpdatedAt = time.Now().UTC()
	result, err := s.services.Training.UpdateTrainee(r.Context(), principal, current, input.Version)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) deleteTrainee(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, r, domain.FieldError{Field: "version", Message: "must be an integer"})
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	if err := s.services.Training.DeleteTrainee(r.Context(), principal, parseID(r), version); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) trainingSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.services.Training.Summary(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

type createExecutionScenarioRequest struct {
	Name            string `json:"name"`
	ProtocolFamily  string `json:"protocol_family"`
	OperationCount  int    `json:"operation_count"`
	RequestedUnits  int    `json:"requested_units"`
	DurationMinutes int    `json:"duration_minutes"`
}

func (s *Server) createExecutionScenario(w http.ResponseWriter, r *http.Request) {
	var input createExecutionScenarioRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.services.Scenario.Create(r.Context(), service.CreateScenarioInput{
		Name: input.Name, ProtocolFamily: input.ProtocolFamily, OperationCount: input.OperationCount,
		RequestedUnits: input.RequestedUnits, DurationMinutes: input.DurationMinutes,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) {
	principal, _ := requestmeta.Principal(r.Context())
	writeJSON(w, http.StatusOK, principal)
}

func trainingPage(r *http.Request) (repository.PageRequest, int, int) {
	current, _ := strconv.Atoi(r.URL.Query().Get("current"))
	if current < 1 {
		current = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return repository.PageRequest{Limit: size, Offset: (current - 1) * size, Sort: r.URL.Query().Get("sort"), Desc: r.URL.Query().Get("desc") == "true"}, current, size
}
