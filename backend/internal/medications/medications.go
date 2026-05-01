package medications

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/httpx"
)

type Medication struct {
	ID         string    `json:"id"`
	PatientID  string    `json:"patient_id"`
	Name       string    `json:"name"`
	Dosage     *string   `json:"dosage,omitempty"`
	TimesOfDay []string  `json:"times_of_day"`
	StartDate  time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	Active     bool      `json:"active"`
	Notes      *string   `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Log struct {
	ID            string    `json:"id"`
	MedicationID  string    `json:"medication_id"`
	PatientID     string    `json:"patient_id"`
	ScheduledAt   time.Time `json:"scheduled_at"`
	Status        string    `json:"status"`
	LoggedAt      time.Time `json:"logged_at"`
	MedicationName string   `json:"medication_name,omitempty"`
	Dosage        *string   `json:"dosage,omitempty"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type createReq struct {
	Name       string   `json:"name" binding:"required"`
	Dosage     string   `json:"dosage"`
	TimesOfDay []string `json:"times_of_day"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	Notes      string   `json:"notes"`
}

func (s *Service) Create(c *gin.Context) {
	patientID := resolvePatientID(c)
	if !s.hasAccess(c.Request.Context(), c.GetString(auth.CtxUserID), patientID) {
		httpx.Forbidden(c, "")
		return
	}
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	// Default start_date is today in the patient's timezone, matching how
	// Today() resolves the schedule. Falls back to UTC if the user has no
	// tz set (the column default after migration 0008).
	loc, _, err := s.patientLocation(c.Request.Context(), patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	start := time.Now().In(loc)
	if req.StartDate != "" {
		t, perr := time.Parse("2006-01-02", req.StartDate)
		if perr != nil {
			httpx.BadRequest(c, "invalid start_date")
			return
		}
		start = t
	}
	var end interface{}
	if req.EndDate != "" {
		t, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			httpx.BadRequest(c, "invalid end_date")
			return
		}
		end = t
	}
	var dosage, notes interface{}
	if req.Dosage != "" {
		dosage = req.Dosage
	}
	if req.Notes != "" {
		notes = req.Notes
	}

	var m Medication
	err = s.pool.QueryRow(c.Request.Context(), `
		INSERT INTO medications(patient_id, name, dosage, times_of_day, start_date, end_date, notes)
		VALUES($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, patient_id, name, dosage, times_of_day, start_date, end_date, active, notes, created_at
	`, patientID, req.Name, dosage, req.TimesOfDay, start, end, notes).
		Scan(&m.ID, &m.PatientID, &m.Name, &m.Dosage, &m.TimesOfDay, &m.StartDate, &m.EndDate, &m.Active, &m.Notes, &m.CreatedAt)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (s *Service) List(c *gin.Context) {
	patientID := resolvePatientID(c)
	if !s.hasAccess(c.Request.Context(), c.GetString(auth.CtxUserID), patientID) {
		httpx.Forbidden(c, "")
		return
	}
	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT id, patient_id, name, dosage, times_of_day, start_date, end_date, active, notes, created_at
		FROM medications WHERE patient_id=$1 AND active=TRUE
		ORDER BY created_at DESC
	`, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	out := []Medication{}
	for rows.Next() {
		var m Medication
		if err := rows.Scan(&m.ID, &m.PatientID, &m.Name, &m.Dosage, &m.TimesOfDay, &m.StartDate, &m.EndDate, &m.Active, &m.Notes, &m.CreatedAt); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) Deactivate(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(auth.CtxUserID)
	var patientID string
	if err := s.pool.QueryRow(c.Request.Context(),
		`SELECT patient_id FROM medications WHERE id=$1`, id).Scan(&patientID); err != nil {
		httpx.HandleDBError(c, err)
		return
	}
	if !s.hasAccess(c.Request.Context(), userID, patientID) {
		httpx.Forbidden(c, "")
		return
	}
	if _, err := s.pool.Exec(c.Request.Context(),
		`UPDATE medications SET active=FALSE WHERE id=$1`, id); err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type logReq struct {
	ScheduledAt string `json:"scheduled_at" binding:"required"`
	Status      string `json:"status" binding:"required,oneof=taken missed skipped"`
}

func (s *Service) LogDose(c *gin.Context) {
	medID := c.Param("id")
	userID := c.GetString(auth.CtxUserID)
	var patientID string
	if err := s.pool.QueryRow(c.Request.Context(),
		`SELECT patient_id FROM medications WHERE id=$1`, medID).Scan(&patientID); err != nil {
		httpx.HandleDBError(c, err)
		return
	}
	if !s.hasAccess(c.Request.Context(), userID, patientID) {
		httpx.Forbidden(c, "")
		return
	}
	var req logReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	t, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		httpx.BadRequest(c, "invalid scheduled_at")
		return
	}
	_, err = s.pool.Exec(c.Request.Context(), `
		INSERT INTO medication_logs(medication_id, patient_id, scheduled_at, status)
		VALUES($1,$2,$3,$4)
		ON CONFLICT (medication_id, scheduled_at) DO UPDATE SET status=EXCLUDED.status, logged_at=now()
	`, medID, patientID, t, req.Status)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Today returns today's medication schedule with status (taken/missed/pending).
//
// "Today" is resolved in the patient's timezone (users.tz, default UTC).
// Both Go and Postgres compute the date in that TZ so the boundary
// matches: a 22:00-Almaty-time medication is "today" for an Almaty
// patient even when the server clock has rolled past midnight UTC.
func (s *Service) Today(c *gin.Context) {
	patientID := resolvePatientID(c)
	if !s.hasAccess(c.Request.Context(), c.GetString(auth.CtxUserID), patientID) {
		httpx.Forbidden(c, "")
		return
	}

	loc, tzName, err := s.patientLocation(c.Request.Context(), patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT id, name, dosage, times_of_day
		FROM medications
		WHERE patient_id=$1 AND active=TRUE
		  AND start_date <= (now() AT TIME ZONE $2)::date
		  AND (end_date IS NULL OR end_date >= (now() AT TIME ZONE $2)::date)
	`, patientID, tzName)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()

	type scheduleItem struct {
		MedicationID string    `json:"medication_id"`
		Name         string    `json:"name"`
		Dosage       *string   `json:"dosage,omitempty"`
		ScheduledAt  time.Time `json:"scheduled_at"`
		Status       string    `json:"status"` // pending|taken|missed|skipped
	}

	today := time.Now().In(loc).Format("2006-01-02")
	items := []scheduleItem{}
	for rows.Next() {
		var id, name string
		var dosage *string
		var times []string
		if err := rows.Scan(&id, &name, &dosage, &times); err != nil {
			httpx.Internal(c, err)
			return
		}
		for _, hhmm := range times {
			t, err := time.ParseInLocation("2006-01-02 15:04", today+" "+hhmm, loc)
			if err != nil {
				continue
			}
			items = append(items, scheduleItem{
				MedicationID: id, Name: name, Dosage: dosage, ScheduledAt: t.UTC(), Status: "pending",
			})
		}
	}

	logRows, err := s.pool.Query(c.Request.Context(), `
		SELECT medication_id, scheduled_at, status
		FROM medication_logs
		WHERE patient_id=$1
		  AND (scheduled_at AT TIME ZONE $2)::date = (now() AT TIME ZONE $2)::date
	`, patientID, tzName)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer logRows.Close()
	logs := map[string]string{}
	for logRows.Next() {
		var medID string
		var sched time.Time
		var status string
		if err := logRows.Scan(&medID, &sched, &status); err != nil {
			httpx.Internal(c, err)
			return
		}
		logs[medID+"|"+sched.UTC().Format(time.RFC3339)] = status
	}
	now := time.Now().UTC()
	for i := range items {
		key := items[i].MedicationID + "|" + items[i].ScheduledAt.Format(time.RFC3339)
		if s, ok := logs[key]; ok {
			items[i].Status = s
		} else if items[i].ScheduledAt.Before(now.Add(-30 * time.Minute)) {
			items[i].Status = "missed"
		}
	}
	c.JSON(http.StatusOK, items)
}

// patientLocation looks up the patient's IANA timezone, defaulting to UTC
// if missing or unparseable. Returns both the *time.Location (for Go
// date math) and the textual TZ name (for Postgres `AT TIME ZONE` casts).
func (s *Service) patientLocation(ctx context.Context, patientID string) (*time.Location, string, error) {
	var tz string
	err := s.pool.QueryRow(ctx, `SELECT tz FROM users WHERE id=$1`, patientID).Scan(&tz)
	if err != nil {
		return time.UTC, "UTC", err
	}
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// Fall back rather than 500: stale tz data shouldn't break the
		// schedule. Log so it's visible.
		return time.UTC, "UTC", nil
	}
	return loc, tz, nil
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
