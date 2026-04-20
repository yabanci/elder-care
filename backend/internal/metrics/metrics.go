package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arsenozhetov/elder-care/backend/internal/auth"
	"github.com/arsenozhetov/elder-care/backend/internal/httpx"
)

type Metric struct {
	ID         string    `json:"id"`
	PatientID  string    `json:"patient_id"`
	Kind       string    `json:"kind"`
	Value      float64   `json:"value"`
	Value2     *float64  `json:"value_2,omitempty"`
	Note       *string   `json:"note,omitempty"`
	MeasuredAt time.Time `json:"measured_at"`
}

type Alert struct {
	ID           string    `json:"id"`
	PatientID    string    `json:"patient_id"`
	MetricID     *string   `json:"metric_id,omitempty"`
	Severity     string    `json:"severity"`
	Reason       string    `json:"reason"`
	Kind         string    `json:"kind"`
	Value        *float64  `json:"value,omitempty"`
	BaselineMean *float64  `json:"baseline_mean,omitempty"`
	BaselineStd  *float64  `json:"baseline_std,omitempty"`
	Acknowledged bool      `json:"acknowledged"`
	CreatedAt    time.Time `json:"created_at"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type createReq struct {
	Kind       string   `json:"kind" binding:"required,oneof=pulse bp_sys bp_dia glucose temperature weight spo2"`
	Value      float64  `json:"value" binding:"required"`
	Value2     *float64 `json:"value_2"`
	Note       string   `json:"note"`
	MeasuredAt string   `json:"measured_at"`
}

// CreateForSelf is used by a patient to log their own metric. Also runs the baseline alert engine.
func (s *Service) CreateForSelf(c *gin.Context) {
	patientID := c.GetString(auth.CtxUserID)
	s.create(c, patientID)
}

// CreateForPatient allows a doctor/family to log a metric for a linked patient (e.g. home visit).
func (s *Service) CreateForPatient(c *gin.Context) {
	patientID := c.Param("patientID")
	if !s.hasAccess(c.Request.Context(), c.GetString(auth.CtxUserID), patientID) {
		httpx.Forbidden(c, "no link to this patient")
		return
	}
	s.create(c, patientID)
}

func (s *Service) create(c *gin.Context, patientID string) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	measuredAt := time.Now()
	if req.MeasuredAt != "" {
		t, err := time.Parse(time.RFC3339, req.MeasuredAt)
		if err != nil {
			httpx.BadRequest(c, "invalid measured_at (use RFC3339)")
			return
		}
		measuredAt = t
	}
	var note interface{}
	if req.Note != "" {
		note = req.Note
	}

	var m Metric
	err := s.pool.QueryRow(c.Request.Context(), `
		INSERT INTO health_metrics(patient_id, kind, value, value_2, note, measured_at)
		VALUES($1,$2,$3,$4,$5,$6)
		RETURNING id, patient_id, kind, value, value_2, note, measured_at
	`, patientID, req.Kind, req.Value, req.Value2, note, measuredAt).
		Scan(&m.ID, &m.PatientID, &m.Kind, &m.Value, &m.Value2, &m.Note, &m.MeasuredAt)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	alert, err := s.runBaseline(c.Request.Context(), m)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"metric": m, "alert": alert})
}

func (s *Service) runBaseline(ctx context.Context, m Metric) (*Alert, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT value FROM health_metrics
		WHERE patient_id=$1 AND kind=$2 AND id<>$3
		ORDER BY measured_at DESC
		LIMIT 30
	`, m.PatientID, m.Kind, m.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		history = append(history, v)
	}

	res := Analyze(m.Kind, m.Value, history)
	if res.Severity == "normal" {
		return nil, nil
	}

	var a Alert
	var meanVal, stdVal interface{}
	if res.UsedHistory {
		meanVal, stdVal = res.Mean, res.Std
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO alerts(patient_id, metric_id, severity, reason, kind, value, baseline_mean, baseline_std)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, patient_id, metric_id, severity, reason, kind, value, baseline_mean, baseline_std, acknowledged, created_at
	`, m.PatientID, m.ID, res.Severity, res.Reason, m.Kind, m.Value, meanVal, stdVal).
		Scan(&a.ID, &a.PatientID, &a.MetricID, &a.Severity, &a.Reason, &a.Kind, &a.Value, &a.BaselineMean, &a.BaselineStd, &a.Acknowledged, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) List(c *gin.Context) {
	patientID := resolvePatientID(c)
	if patientID == "" {
		httpx.BadRequest(c, "patient id required")
		return
	}
	if !s.hasAccess(c.Request.Context(), c.GetString(auth.CtxUserID), patientID) {
		httpx.Forbidden(c, "")
		return
	}

	kind := c.Query("kind")
	limit := 100

	args := []interface{}{patientID}
	query := `SELECT id, patient_id, kind, value, value_2, note, measured_at
	           FROM health_metrics WHERE patient_id=$1`
	if kind != "" {
		query += ` AND kind=$2`
		args = append(args, kind)
	}
	query += ` ORDER BY measured_at DESC LIMIT ` + itoa(limit)

	rows, err := s.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()

	out := []Metric{}
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.ID, &m.PatientID, &m.Kind, &m.Value, &m.Value2, &m.Note, &m.MeasuredAt); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) ListAlerts(c *gin.Context) {
	patientID := resolvePatientID(c)
	if patientID == "" {
		httpx.BadRequest(c, "patient id required")
		return
	}
	if !s.hasAccess(c.Request.Context(), c.GetString(auth.CtxUserID), patientID) {
		httpx.Forbidden(c, "")
		return
	}
	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT id, patient_id, metric_id, severity, reason, kind, value, baseline_mean, baseline_std, acknowledged, created_at
		FROM alerts WHERE patient_id=$1
		ORDER BY created_at DESC LIMIT 50
	`, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	out := []Alert{}
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.PatientID, &a.MetricID, &a.Severity, &a.Reason, &a.Kind, &a.Value, &a.BaselineMean, &a.BaselineStd, &a.Acknowledged, &a.CreatedAt); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, a)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	userID := c.GetString(auth.CtxUserID)

	var patientID string
	err := s.pool.QueryRow(c.Request.Context(),
		`SELECT patient_id FROM alerts WHERE id=$1`, alertID).Scan(&patientID)
	if err != nil {
		httpx.HandleDBError(c, err)
		return
	}
	if !s.hasAccess(c.Request.Context(), userID, patientID) {
		httpx.Forbidden(c, "")
		return
	}
	_, err = s.pool.Exec(c.Request.Context(),
		`UPDATE alerts SET acknowledged=TRUE WHERE id=$1`, alertID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Summary returns the latest value per kind for dashboard rendering.
func (s *Service) Summary(c *gin.Context) {
	patientID := resolvePatientID(c)
	if patientID == "" {
		httpx.BadRequest(c, "patient id required")
		return
	}
	if !s.hasAccess(c.Request.Context(), c.GetString(auth.CtxUserID), patientID) {
		httpx.Forbidden(c, "")
		return
	}
	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT DISTINCT ON (kind) id, patient_id, kind, value, value_2, note, measured_at
		FROM health_metrics
		WHERE patient_id=$1
		ORDER BY kind, measured_at DESC
	`, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	out := []Metric{}
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.ID, &m.PatientID, &m.Kind, &m.Value, &m.Value2, &m.Note, &m.MeasuredAt); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) hasAccess(ctx context.Context, userID, patientID string) bool {
	if userID == patientID {
		return true
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM patient_links WHERE patient_id=$1 AND linked_id=$2)
	`, patientID, userID).Scan(&exists)
	return err == nil && exists
}

func resolvePatientID(c *gin.Context) string {
	if id := c.Param("patientID"); id != "" {
		return id
	}
	return c.GetString(auth.CtxUserID)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
