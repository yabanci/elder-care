package links

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/httpx"
)

type LinkedPatient struct {
	PatientID string  `json:"patient_id"`
	FullName  string  `json:"full_name"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
	Relation  string  `json:"relation"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// MyPatients returns patients linked to the current doctor/family user.
func (s *Service) MyPatients(c *gin.Context) {
	userID := c.GetString(auth.CtxUserID)
	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT u.id, u.full_name, u.email, u.phone, pl.relation
		FROM patient_links pl
		JOIN users u ON u.id = pl.patient_id
		WHERE pl.linked_id = $1
		ORDER BY u.full_name
	`, userID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	out := []LinkedPatient{}
	for rows.Next() {
		var lp LinkedPatient
		if err := rows.Scan(&lp.PatientID, &lp.FullName, &lp.Email, &lp.Phone, &lp.Relation); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, lp)
	}
	c.JSON(http.StatusOK, out)
}

// MyCaregivers returns caregivers (doctor+family) linked to the current patient.
func (s *Service) MyCaregivers(c *gin.Context) {
	patientID := c.GetString(auth.CtxUserID)
	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT u.id, u.full_name, u.email, u.phone, pl.relation
		FROM patient_links pl
		JOIN users u ON u.id = pl.linked_id
		WHERE pl.patient_id = $1
		ORDER BY pl.relation, u.full_name
	`, patientID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	type caregiver struct {
		ID       string  `json:"id"`
		FullName string  `json:"full_name"`
		Email    string  `json:"email"`
		Phone    *string `json:"phone,omitempty"`
		Relation string  `json:"relation"`
	}
	out := []caregiver{}
	for rows.Next() {
		var cg caregiver
		if err := rows.Scan(&cg.ID, &cg.FullName, &cg.Email, &cg.Phone, &cg.Relation); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, cg)
	}
	c.JSON(http.StatusOK, out)
}

type linkReq struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

// Link is called by a doctor or family member to connect to a patient by the patient's invite code.
func (s *Service) Link(c *gin.Context) {
	var req linkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	userID := c.GetString(auth.CtxUserID)
	role := c.GetString(auth.CtxRole)
	if role != "doctor" && role != "family" {
		httpx.Forbidden(c, "only doctor or family can link to a patient")
		return
	}

	var patientID, patientRole string
	err := s.pool.QueryRow(c.Request.Context(),
		`SELECT id, role FROM users WHERE invite_code=$1`,
		strings.ToUpper(req.InviteCode)).Scan(&patientID, &patientRole)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.NotFound(c, "invite code not found")
			return
		}
		httpx.Internal(c, err)
		return
	}
	if patientRole != "patient" {
		httpx.BadRequest(c, "invite code does not belong to a patient")
		return
	}

	_, err = s.pool.Exec(c.Request.Context(), `
		INSERT INTO patient_links(patient_id, linked_id, relation)
		VALUES($1,$2,$3)
		ON CONFLICT (patient_id, linked_id) DO NOTHING
	`, patientID, userID, role)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "patient_id": patientID})
}
