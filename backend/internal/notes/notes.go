// Package notes implements doctor/family care notes — short text annotations
// attached to a patient (e.g. "switched to morning dose", "complains of
// dizziness"). Patients can read notes about themselves but cannot author
// them; doctors and family can both author. Notes are append-only —
// editing/deleting requires re-authoring with corrected text, on purpose
// (clinical record hygiene).
package notes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/httpx"
)

type Note struct {
	ID         string    `json:"id"`
	PatientID  string    `json:"patient_id"`
	AuthorID   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	AuthorRole string    `json:"author_role"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type createReq struct {
	Body string `json:"body" binding:"required,min=1,max=4000"`
}

// Create handles POST /api/patients/:patientID/notes. Author must be
// linked to the patient AND have role doctor or family. Patients can't
// add notes about themselves through this route.
func (s *Service) Create(c *gin.Context) {
	patientID := c.Param("patientID")
	authorID := c.GetString(auth.CtxUserID)
	role := c.GetString(auth.CtxRole)

	if role != "doctor" && role != "family" {
		httpx.Forbidden(c, "only doctor or family may author notes")
		return
	}
	if !s.linked(c, authorID, patientID) {
		httpx.Forbidden(c, "not linked to this patient")
		return
	}

	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	var n Note
	err := s.pool.QueryRow(c.Request.Context(), `
		WITH inserted AS (
			INSERT INTO care_notes(patient_id, author_id, body)
			VALUES($1, $2, $3)
			RETURNING id, patient_id, author_id, body, created_at
		)
		SELECT i.id, i.patient_id, i.author_id, u.full_name, u.role, i.body, i.created_at
		FROM inserted i
		JOIN users u ON u.id = i.author_id
	`, patientID, authorID, req.Body).
		Scan(&n.ID, &n.PatientID, &n.AuthorID, &n.AuthorName, &n.AuthorRole, &n.Body, &n.CreatedAt)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, n)
}

// List handles GET /api/patients/:patientID/notes. Returns notes
// chronologically newest-first. Both the patient themselves and any
// linked caregiver can read.
func (s *Service) List(c *gin.Context) {
	patientID := c.Param("patientID")
	userID := c.GetString(auth.CtxUserID)

	// Self-access is allowed (patient reading their own notes).
	if userID != patientID && !s.linked(c, userID, patientID) {
		httpx.Forbidden(c, "")
		return
	}

	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT n.id, n.patient_id, n.author_id, u.full_name, u.role, n.body, n.created_at
		FROM care_notes n
		JOIN users u ON u.id = n.author_id
		WHERE n.patient_id = $1
		ORDER BY n.created_at DESC
		LIMIT 100
	`, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.PatientID, &n.AuthorID, &n.AuthorName, &n.AuthorRole, &n.Body, &n.CreatedAt); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, n)
	}
	c.JSON(http.StatusOK, out)
}

// linked returns true iff a patient_links row exists between userID and
// patientID. We re-implement the check here (rather than depending on
// internal/links) to keep this package's import surface minimal.
func (s *Service) linked(c *gin.Context, userID, patientID string) bool {
	var exists bool
	err := s.pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS(SELECT 1 FROM patient_links WHERE patient_id=$1 AND linked_id=$2)
	`, patientID, userID).Scan(&exists)
	return err == nil && exists
}
