package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/baseline"
	"eldercare/backend/internal/httpx"
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
	ID               string    `json:"id"`
	PatientID        string    `json:"patient_id"`
	MetricID         *string   `json:"metric_id,omitempty"`
	Severity         string    `json:"severity"`
	Reason           string    `json:"reason"`
	ReasonCode       string    `json:"reason_code"`
	AlgorithmVersion string    `json:"algorithm_version"`
	Kind             string    `json:"kind"`
	Value            *float64  `json:"value,omitempty"`
	BaselineMean     *float64  `json:"baseline_mean,omitempty"`
	BaselineStd      *float64  `json:"baseline_std,omitempty"`
	Acknowledged     bool      `json:"acknowledged"`
	CreatedAt        time.Time `json:"created_at"`
}

// Notifier abstracts the push subsystem so metrics doesn't import it
// directly (and so tests can inject a no-op). Implemented by *push.Service.
type Notifier interface {
	SendToUser(ctx context.Context, userID string, payload PushPayload)
}

// PushPayload mirrors push.AlertPayload but lives here so metrics doesn't
// need to depend on the push package. The server wires the two with a
// small adapter function.
type PushPayload struct {
	Title     string
	Body      string
	URL       string
	Severity  string
	PatientID string
	AlertID   string
}

// nopNotifier is the default when the server hasn't configured push.
type nopNotifier struct{}

func (nopNotifier) SendToUser(context.Context, string, PushPayload) {}

// EventPublisher streams alert events to live SSE clients.
// Implemented by *events.Broker; metrics depends on this minimal
// interface to avoid an internal/metrics ↔ internal/events import cycle.
type EventPublisher interface {
	PublishAlert(ctx context.Context, patientID, alertID, severity, kind string)
}

type nopPublisher struct{}

func (nopPublisher) PublishAlert(context.Context, string, string, string, string) {}

type Service struct {
	pool      *pgxpool.Pool
	notifier  Notifier
	publisher EventPublisher
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, notifier: nopNotifier{}, publisher: nopPublisher{}}
}

// WithNotifier returns the same service with the given notifier installed.
// Use chained at server startup: metrics.NewService(pool).WithNotifier(pushSvc)
func (s *Service) WithNotifier(n Notifier) *Service {
	if n == nil {
		s.notifier = nopNotifier{}
		return s
	}
	s.notifier = n
	return s
}

// WithEventPublisher returns the same service with a live-event broker
// installed. nil falls back to no-op.
func (s *Service) WithEventPublisher(p EventPublisher) *Service {
	if p == nil {
		s.publisher = nopPublisher{}
		return s
	}
	s.publisher = p
	return s
}

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

// runBaseline drives the v2 baseline pipeline: loads the patient's chronic
// conditions, fetches the recent same-kind reading history, runs
// baseline.Analyze, persists an algorithm_runs row (always) and an alerts
// row (only when severity != normal). Returns the new alert if one was
// created, or nil otherwise.
func (s *Service) runBaseline(ctx context.Context, m Metric) (*Alert, error) {
	chronic, err := s.fetchChronicConditions(ctx, m.PatientID)
	if err != nil {
		return nil, err
	}
	profile := baseline.ParseProfile(chronic)

	history, err := s.fetchHistory(ctx, m.PatientID, m.Kind, m.ID)
	if err != nil {
		return nil, err
	}

	res := baseline.Analyze(baseline.Input{
		Kind:    m.Kind,
		Value:   m.Value,
		History: history,
		Profile: profile,
		Now:     m.MeasuredAt,
	})

	if err := s.persistRun(ctx, m, res); err != nil {
		return nil, err
	}

	if res.Severity == baseline.SeverityNormal {
		return nil, nil
	}

	var meanVal, stdVal any
	if res.UsedHistory {
		meanVal, stdVal = res.Mean, res.Std
	}
	var a Alert
	err = s.pool.QueryRow(ctx, `
		INSERT INTO alerts(
			patient_id, metric_id, severity, reason, reason_code,
			algorithm_version, kind, value, baseline_mean, baseline_std
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, patient_id, metric_id, severity, reason, reason_code,
		          algorithm_version, kind, value, baseline_mean, baseline_std,
		          acknowledged, created_at
	`, m.PatientID, m.ID, res.Severity, res.ReasonCode, res.ReasonCode,
		res.AlgorithmVersion, m.Kind, m.Value, meanVal, stdVal).
		Scan(&a.ID, &a.PatientID, &a.MetricID, &a.Severity, &a.Reason, &a.ReasonCode,
			&a.AlgorithmVersion, &a.Kind, &a.Value, &a.BaselineMean, &a.BaselineStd,
			&a.Acknowledged, &a.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Push critical alerts to the patient and all linked caregivers.
	// SendToUser is fire-and-forget at the implementation level (each
	// delivery runs in its own goroutine tracked by push.Service's
	// WaitGroup), so we just enqueue here and return — graceful shutdown
	// will drain via push.Service.Drain.
	if a.Severity == baseline.SeverityCritical {
		recipients := s.recipientsForPush(ctx, a.PatientID)
		payload := PushPayload{
			Title:     "ElderCare alert",
			Body:      "Критический показатель — проверьте панель",
			URL:       "/patient/alerts",
			Severity:  a.Severity,
			PatientID: a.PatientID,
			AlertID:   a.ID,
		}
		for _, rid := range recipients {
			s.notifier.SendToUser(ctx, rid, payload)
		}
		// Stream event to live SSE clients (browser tabs that have the
		// dashboard open). Same recipients as push; broker handles them
		// independently. Best-effort, non-blocking.
		s.publisher.PublishAlert(ctx, a.PatientID, a.ID, a.Severity, a.Kind)
	}

	return &a, nil
}

// recipientsForPush returns the patient + every doctor/family caregiver
// linked to them. Filters by patient_links.relation explicitly so a
// future relation type (e.g. 'admin', 'researcher') does not start
// receiving PHI by default. Errors are swallowed — push is best-effort
// and must never block the alert insert path.
func (s *Service) recipientsForPush(ctx context.Context, patientID string) []string {
	out := []string{patientID}
	rows, err := s.pool.Query(ctx, `
		SELECT linked_id FROM patient_links
		WHERE patient_id=$1 AND relation IN ('doctor', 'family')
	`, patientID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// fetchChronicConditions returns the patient's chronic_conditions text, or
// "" if unset. Used to derive the condition profile.
func (s *Service) fetchChronicConditions(ctx context.Context, patientID string) (string, error) {
	var text *string
	err := s.pool.QueryRow(ctx,
		`SELECT chronic_conditions FROM users WHERE id=$1`, patientID).Scan(&text)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if text == nil {
		return "", nil
	}
	return *text, nil
}

// fetchHistory returns the patient's recent same-kind readings for
// baseline estimation. We fetch a generous window (last 90 days, capped
// 200 rows) and let baseline.WindowFilter trim down inside Analyze.
func (s *Service) fetchHistory(ctx context.Context, patientID, kind, excludeID string) ([]baseline.Reading, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT value, measured_at FROM health_metrics
		WHERE patient_id=$1 AND kind=$2 AND id<>$3
		  AND measured_at >= now() - interval '90 days'
		ORDER BY measured_at DESC
		LIMIT 200
	`, patientID, kind, excludeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []baseline.Reading
	for rows.Next() {
		var r baseline.Reading
		if err := rows.Scan(&r.Value, &r.MeasuredAt); err != nil {
			return nil, err
		}
		history = append(history, r)
	}
	return history, nil
}

// persistRun records every algorithm invocation, alert or not, into
// algorithm_runs. Useful for offline replay, defense telemetry, and as a
// safety net so we always know what the algorithm decided for any given
// reading.
func (s *Service) persistRun(ctx context.Context, m Metric, res baseline.Result) error {
	var meanVal, stdVal, zVal any
	if res.UsedHistory {
		meanVal, stdVal, zVal = res.Mean, res.Std, res.ZScore
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO algorithm_runs(
			patient_id, metric_id, kind, value,
			estimator, mean_used, std_used, z_score,
			severity, reason_code, used_history, history_size, algorithm_version
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, m.PatientID, m.ID, m.Kind, m.Value,
		string(res.Estimator), meanVal, stdVal, zVal,
		res.Severity, res.ReasonCode, res.UsedHistory, res.HistorySize, res.AlgorithmVersion,
	)
	return err
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
		SELECT id, patient_id, metric_id, severity, reason, reason_code,
		       algorithm_version, kind, value, baseline_mean, baseline_std,
		       acknowledged, created_at
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
		if err := rows.Scan(&a.ID, &a.PatientID, &a.MetricID, &a.Severity, &a.Reason,
			&a.ReasonCode, &a.AlgorithmVersion, &a.Kind, &a.Value,
			&a.BaselineMean, &a.BaselineStd, &a.Acknowledged, &a.CreatedAt); err != nil {
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
