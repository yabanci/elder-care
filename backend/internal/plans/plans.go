package plans

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arsenozhetov/elder-care/backend/internal/auth"
	"github.com/arsenozhetov/elder-care/backend/internal/httpx"
)

type Plan struct {
	ID        string    `json:"id"`
	PatientID string    `json:"patient_id"`
	DayOfWeek int       `json:"day_of_week"`
	Title     string    `json:"title"`
	PlanType  string    `json:"plan_type"`
	TimeOfDay *string   `json:"time_of_day,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type createReq struct {
	DayOfWeek int    `json:"day_of_week" binding:"min=0,max=6"`
	Title     string `json:"title" binding:"required"`
	PlanType  string `json:"plan_type" binding:"required,oneof=doctor_visit take_med rest other"`
	TimeOfDay string `json:"time_of_day"`
}

type updateReq struct {
	DayOfWeek *int    `json:"day_of_week"`
	Title     *string `json:"title"`
	PlanType  *string `json:"plan_type"`
	TimeOfDay *string `json:"time_of_day"`
}

func (s *Service) List(c *gin.Context) {
	patientID := c.GetString(auth.CtxUserID)
	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT id, patient_id, day_of_week, title, plan_type, time_of_day, created_at
		FROM plans
		WHERE patient_id=$1
		ORDER BY day_of_week, time_of_day NULLS LAST, created_at
	`, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	items := []Plan{}
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.ID, &p.PatientID, &p.DayOfWeek, &p.Title, &p.PlanType, &p.TimeOfDay, &p.CreatedAt); err != nil {
			httpx.Internal(c, err)
			return
		}
		items = append(items, p)
	}
	c.JSON(http.StatusOK, items)
}

func (s *Service) Create(c *gin.Context) {
	patientID := c.GetString(auth.CtxUserID)
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	var timeArg interface{}
	if req.TimeOfDay != "" {
		timeArg = req.TimeOfDay
	}
	var p Plan
	err := s.pool.QueryRow(c.Request.Context(), `
		INSERT INTO plans(patient_id, day_of_week, title, plan_type, time_of_day)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id, patient_id, day_of_week, title, plan_type, time_of_day, created_at
	`, patientID, req.DayOfWeek, req.Title, req.PlanType, timeArg).
		Scan(&p.ID, &p.PatientID, &p.DayOfWeek, &p.Title, &p.PlanType, &p.TimeOfDay, &p.CreatedAt)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (s *Service) Update(c *gin.Context) {
	patientID := c.GetString(auth.CtxUserID)
	id := c.Param("id")
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	ct, err := s.pool.Exec(c.Request.Context(), `
		UPDATE plans SET
			day_of_week = COALESCE($3, day_of_week),
			title       = COALESCE($4, title),
			plan_type   = COALESCE($5, plan_type),
			time_of_day = COALESCE($6, time_of_day)
		WHERE id=$1 AND patient_id=$2
	`, id, patientID, req.DayOfWeek, req.Title, req.PlanType, req.TimeOfDay)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	if ct.RowsAffected() == 0 {
		httpx.NotFound(c, "plan not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Service) Delete(c *gin.Context) {
	patientID := c.GetString(auth.CtxUserID)
	id := c.Param("id")
	ct, err := s.pool.Exec(c.Request.Context(), `
		DELETE FROM plans WHERE id=$1 AND patient_id=$2
	`, id, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	if ct.RowsAffected() == 0 {
		httpx.NotFound(c, "plan not found")
		return
	}
	c.Status(http.StatusNoContent)
}
