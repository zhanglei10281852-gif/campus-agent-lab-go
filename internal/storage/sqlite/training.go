package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/identity"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
)

const cohortColumns = `c.id, c.code, c.name, c.grade, c.instructor, COALESCE(c.workspace_id, ''),
	c.capacity, (SELECT COUNT(*) FROM trainees t WHERE t.cohort_id = c.id), c.status,
	c.version, c.created_at, c.updated_at`

func scanCohort(row scanner) (domain.Cohort, error) {
	var value domain.Cohort
	var status, createdAt, updatedAt string
	if err := row.Scan(&value.ID, &value.Code, &value.Name, &value.Grade, &value.Instructor,
		&value.WorkspaceID, &value.Capacity, &value.StudentCount, &status, &value.Version,
		&createdAt, &updatedAt); err != nil {
		return domain.Cohort{}, err
	}
	value.Status = domain.CohortStatus(status)
	var err error
	if value.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Cohort{}, err
	}
	if value.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Cohort{}, err
	}
	return value, nil
}

func (s *Store) ListCohorts(ctx context.Context, page repository.PageRequest, search string, status domain.CohortStatus) ([]domain.Cohort, int, error) {
	page = page.Normalize(200)
	where := []string{"1=1"}
	args := make([]any, 0, 6)
	if search != "" {
		where = append(where, "(c.code LIKE ? OR c.name LIKE ? OR c.instructor LIKE ?)")
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern, pattern)
	}
	if status != "" {
		where = append(where, "c.status = ?")
		args = append(args, status)
	}
	clause := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cohorts c WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count cohorts: %w", err)
	}
	order := "c.created_at"
	switch page.Sort {
	case "name":
		order = "c.name COLLATE NOCASE"
	case "grade":
		order = "c.grade COLLATE NOCASE"
	case "student_count":
		order = "8"
	}
	if page.Desc {
		order += " DESC"
	} else {
		order += " ASC"
	}
	queryArgs := append(append([]any(nil), args...), page.Limit, page.Offset)
	rows, err := s.db.QueryContext(ctx, "SELECT "+cohortColumns+" FROM cohorts c WHERE "+clause+" ORDER BY "+order+" LIMIT ? OFFSET ?", queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list cohorts: %w", err)
	}
	defer rows.Close()
	result := make([]domain.Cohort, 0)
	for rows.Next() {
		value, scanErr := scanCohort(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan cohort: %w", scanErr)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate cohorts: %w", err)
	}
	return result, total, nil
}

func (s *Store) ListAllCohorts(ctx context.Context) ([]domain.Cohort, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+cohortColumns+" FROM cohorts c WHERE c.status <> 'closed' ORDER BY c.grade DESC, c.name")
	if err != nil {
		return nil, fmt.Errorf("list all cohorts: %w", err)
	}
	defer rows.Close()
	result := make([]domain.Cohort, 0)
	for rows.Next() {
		value, scanErr := scanCohort(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan cohort: %w", scanErr)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *Store) GetCohort(ctx context.Context, id string) (domain.Cohort, error) {
	value, err := scanCohort(s.db.QueryRowContext(ctx, "SELECT "+cohortColumns+" FROM cohorts c WHERE c.id = ?", id))
	return value, translateError("get cohort", err)
}

func (s *Store) CreateCohort(ctx context.Context, value domain.Cohort, actor, requestID string) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.trainingTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO cohorts(id, code, name, grade, instructor, workspace_id, capacity,
			status, version, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			value.ID, value.Code, value.Name, value.Grade, value.Instructor, nullableString(value.WorkspaceID),
			value.Capacity, value.Status, value.Version, formatTime(value.CreatedAt), formatTime(value.UpdatedAt))
		if err != nil {
			return translateError("insert cohort", err)
		}
		return insertTrainingAudit(ctx, tx, actor, requestID, "cohort.create", "cohort", value.ID, "success", map[string]string{"code": value.Code})
	})
}

func (s *Store) UpdateCohort(ctx context.Context, value domain.Cohort, expected int64, actor, requestID string) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.trainingTx(ctx, func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM trainees WHERE cohort_id = ?", value.ID).Scan(&count); err != nil {
			return translateError("count cohort trainees", err)
		}
		if count > value.Capacity {
			return fmt.Errorf("cohort capacity is below current enrollment: %w", domain.ErrCapacityExceeded)
		}
		result, err := tx.ExecContext(ctx, `UPDATE cohorts SET code=?, name=?, grade=?, instructor=?, workspace_id=?,
			capacity=?, status=?, version=version+1, updated_at=? WHERE id=? AND version=?`,
			value.Code, value.Name, value.Grade, value.Instructor, nullableString(value.WorkspaceID), value.Capacity,
			value.Status, formatTime(value.UpdatedAt), value.ID, expected)
		if err != nil {
			return translateError("update cohort", err)
		}
		if err := expectVersion(result, "update cohort"); err != nil {
			return err
		}
		return insertTrainingAudit(ctx, tx, actor, requestID, "cohort.update", "cohort", value.ID, "success", map[string]string{"version": fmt.Sprint(expected + 1)})
	})
}

func (s *Store) DeleteCohort(ctx context.Context, id string, expected int64, actor, requestID string) error {
	return s.trainingTx(ctx, func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM trainees WHERE cohort_id = ?", id).Scan(&count); err != nil {
			return translateError("count cohort trainees", err)
		}
		if count > 0 {
			return fmt.Errorf("cohort still has trainees: %w", domain.ErrConflict)
		}
		result, err := tx.ExecContext(ctx, "DELETE FROM cohorts WHERE id=? AND version=?", id, expected)
		if err != nil {
			return translateError("delete cohort", err)
		}
		if err := expectVersion(result, "delete cohort"); err != nil {
			return err
		}
		return insertTrainingAudit(ctx, tx, actor, requestID, "cohort.delete", "cohort", id, "success", nil)
	})
}

const traineeColumns = `t.id, t.student_no, t.name, t.gender, t.birth_date, t.phone, t.email,
	t.cohort_id, c.name, t.status, t.version, t.created_at, t.updated_at`

func scanTrainee(row scanner) (domain.Trainee, error) {
	var value domain.Trainee
	var status, createdAt, updatedAt string
	if err := row.Scan(&value.ID, &value.StudentNo, &value.Name, &value.Gender, &value.BirthDate,
		&value.Phone, &value.Email, &value.CohortID, &value.Cohort, &status, &value.Version,
		&createdAt, &updatedAt); err != nil {
		return domain.Trainee{}, err
	}
	value.Status = domain.TraineeStatus(status)
	var err error
	if value.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Trainee{}, err
	}
	if value.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Trainee{}, err
	}
	return value, nil
}

func (s *Store) ListTrainees(ctx context.Context, page repository.PageRequest, name, studentNo, cohortID string, status domain.TraineeStatus) ([]domain.Trainee, int, error) {
	page = page.Normalize(200)
	where := []string{"1=1"}
	args := make([]any, 0, 6)
	if name != "" {
		where = append(where, "t.name LIKE ?")
		args = append(args, "%"+name+"%")
	}
	if studentNo != "" {
		where = append(where, "t.student_no LIKE ?")
		args = append(args, "%"+studentNo+"%")
	}
	if cohortID != "" {
		where = append(where, "t.cohort_id = ?")
		args = append(args, cohortID)
	}
	if status != "" {
		where = append(where, "t.status = ?")
		args = append(args, status)
	}
	clause := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM trainees t WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count trainees: %w", err)
	}
	order := "t.created_at"
	switch page.Sort {
	case "name":
		order = "t.name COLLATE NOCASE"
	case "student_no":
		order = "t.student_no COLLATE NOCASE"
	case "status":
		order = "t.status"
	}
	if page.Desc {
		order += " DESC"
	} else {
		order += " ASC"
	}
	queryArgs := append(append([]any(nil), args...), page.Limit, page.Offset)
	rows, err := s.db.QueryContext(ctx, "SELECT "+traineeColumns+" FROM trainees t JOIN cohorts c ON c.id=t.cohort_id WHERE "+clause+" ORDER BY "+order+" LIMIT ? OFFSET ?", queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list trainees: %w", err)
	}
	defer rows.Close()
	result := make([]domain.Trainee, 0)
	for rows.Next() {
		value, scanErr := scanTrainee(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan trainee: %w", scanErr)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate trainees: %w", err)
	}
	return result, total, nil
}

func (s *Store) GetTrainee(ctx context.Context, id string) (domain.Trainee, error) {
	value, err := scanTrainee(s.db.QueryRowContext(ctx, "SELECT "+traineeColumns+" FROM trainees t JOIN cohorts c ON c.id=t.cohort_id WHERE t.id=?", id))
	return value, translateError("get trainee", err)
}

func (s *Store) CreateTrainee(ctx context.Context, value domain.Trainee, actor, requestID string) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.trainingTx(ctx, func(tx *sql.Tx) error {
		if err := ensureCohortCapacity(ctx, tx, value.CohortID, ""); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO trainees(id,student_no,name,gender,birth_date,phone,email,cohort_id,status,version,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, value.ID, value.StudentNo, value.Name, value.Gender, value.BirthDate, value.Phone,
			value.Email, value.CohortID, value.Status, value.Version, formatTime(value.CreatedAt), formatTime(value.UpdatedAt))
		if err != nil {
			return translateError("insert trainee", err)
		}
		return insertTrainingAudit(ctx, tx, actor, requestID, "trainee.create", "trainee", value.ID, "success", map[string]string{"cohort_id": value.CohortID})
	})
}

func (s *Store) UpdateTrainee(ctx context.Context, value domain.Trainee, expected int64, actor, requestID string) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.trainingTx(ctx, func(tx *sql.Tx) error {
		var oldCohort string
		if err := tx.QueryRowContext(ctx, "SELECT cohort_id FROM trainees WHERE id=?", value.ID).Scan(&oldCohort); err != nil {
			return translateError("get trainee cohort", err)
		}
		if oldCohort != value.CohortID {
			if err := ensureCohortCapacity(ctx, tx, value.CohortID, value.ID); err != nil {
				return err
			}
		}
		result, err := tx.ExecContext(ctx, `UPDATE trainees SET student_no=?,name=?,gender=?,birth_date=?,phone=?,email=?,
			cohort_id=?,status=?,version=version+1,updated_at=? WHERE id=? AND version=?`,
			value.StudentNo, value.Name, value.Gender, value.BirthDate, value.Phone, value.Email, value.CohortID, value.Status,
			formatTime(value.UpdatedAt), value.ID, expected)
		if err != nil {
			return translateError("update trainee", err)
		}
		if err := expectVersion(result, "update trainee"); err != nil {
			return err
		}
		return insertTrainingAudit(ctx, tx, actor, requestID, "trainee.update", "trainee", value.ID, "success", map[string]string{"cohort_id": value.CohortID})
	})
}

func (s *Store) DeleteTrainee(ctx context.Context, id string, expected int64, actor, requestID string) error {
	return s.trainingTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, "DELETE FROM trainees WHERE id=? AND version=?", id, expected)
		if err != nil {
			return translateError("delete trainee", err)
		}
		if err := expectVersion(result, "delete trainee"); err != nil {
			return err
		}
		return insertTrainingAudit(ctx, tx, actor, requestID, "trainee.delete", "trainee", id, "success", nil)
	})
}

func ensureCohortCapacity(ctx context.Context, tx *sql.Tx, cohortID, excludeTrainee string) error {
	var capacity, count int
	var status string
	err := tx.QueryRowContext(ctx, `SELECT c.capacity,c.status,(SELECT COUNT(*) FROM trainees t WHERE t.cohort_id=c.id AND t.id<>?)
		FROM cohorts c WHERE c.id=?`, excludeTrainee, cohortID).Scan(&capacity, &status, &count)
	if err != nil {
		return translateError("check cohort capacity", err)
	}
	if status != "active" {
		return fmt.Errorf("cohort is not active: %w", domain.ErrConflict)
	}
	if count >= capacity {
		return fmt.Errorf("cohort capacity reached: %w", domain.ErrCapacityExceeded)
	}
	return nil
}

func (s *Store) TrainingSummary(ctx context.Context) (domain.TrainingSummary, error) {
	var result domain.TrainingSummary
	err := s.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM cohorts),
		(SELECT COUNT(*) FROM cohorts WHERE status='active'),
		(SELECT COUNT(*) FROM trainees),
		(SELECT COUNT(*) FROM trainees WHERE status='active')`).Scan(
		&result.Cohorts, &result.ActiveCohorts, &result.Trainees, &result.ActiveTrainees)
	if err != nil {
		return domain.TrainingSummary{}, fmt.Errorf("training summary: %w", err)
	}
	return result, nil
}

func (s *Store) trainingTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin training transaction: %w", err)
	}
	if err = fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback training transaction: %w", rollbackErr))
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit training transaction: %w", err)
	}
	return nil
}

func insertTrainingAudit(ctx context.Context, tx *sql.Tx, actor, requestID, action, entityType, entityID, outcome string, metadata map[string]string) error {
	if metadata == nil {
		metadata = map[string]string{}
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode training audit: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(id,request_id,actor,action,entity_type,entity_id,outcome,metadata_json,created_at)
		VALUES(?,?,?,?,?,?,?,?,?)`, identity.New("audit"), requestID, actor, action, entityType, entityID, outcome, string(payload), formatTime(time.Now().UTC()))
	return translateError("insert training audit", err)
}
