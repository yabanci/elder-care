package messages

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arsenozhetov/elder-care/backend/internal/auth"
	"github.com/arsenozhetov/elder-care/backend/internal/httpx"
)

type Message struct {
	ID          string     `json:"id"`
	SenderID    string     `json:"sender_id"`
	RecipientID string     `json:"recipient_id"`
	Body        string     `json:"body"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	SenderName  string     `json:"sender_name,omitempty"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type sendReq struct {
	RecipientID string `json:"recipient_id" binding:"required,uuid"`
	Body        string `json:"body" binding:"required,min=1,max=2000"`
}

func (s *Service) Send(c *gin.Context) {
	var req sendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	userID := c.GetString(auth.CtxUserID)
	if !s.canContact(c, userID, req.RecipientID) {
		httpx.Forbidden(c, "recipient is not linked to you")
		return
	}
	var m Message
	err := s.pool.QueryRow(c.Request.Context(), `
		INSERT INTO messages(sender_id, recipient_id, body)
		VALUES($1,$2,$3)
		RETURNING id, sender_id, recipient_id, body, read_at, created_at
	`, userID, req.RecipientID, req.Body).
		Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.Body, &m.ReadAt, &m.CreatedAt)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

// Thread returns messages between the current user and :otherID.
func (s *Service) Thread(c *gin.Context) {
	userID := c.GetString(auth.CtxUserID)
	otherID := c.Param("otherID")
	if !s.canContact(c, userID, otherID) {
		httpx.Forbidden(c, "")
		return
	}
	rows, err := s.pool.Query(c.Request.Context(), `
		SELECT m.id, m.sender_id, m.recipient_id, m.body, m.read_at, m.created_at, u.full_name
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE (m.sender_id=$1 AND m.recipient_id=$2)
		   OR (m.sender_id=$2 AND m.recipient_id=$1)
		ORDER BY m.created_at ASC
		LIMIT 500
	`, userID, otherID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.Body, &m.ReadAt, &m.CreatedAt, &m.SenderName); err != nil {
			httpx.Internal(c, err)
			return
		}
		out = append(out, m)
	}

	// mark incoming messages as read
	_, _ = s.pool.Exec(c.Request.Context(), `
		UPDATE messages SET read_at=now()
		WHERE recipient_id=$1 AND sender_id=$2 AND read_at IS NULL
	`, userID, otherID)

	c.JSON(http.StatusOK, out)
}

func (s *Service) canContact(c *gin.Context, userID, otherID string) bool {
	if userID == otherID {
		return false
	}
	var exists bool
	err := s.pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM patient_links
			WHERE (patient_id=$1 AND linked_id=$2)
			   OR (patient_id=$2 AND linked_id=$1)
		)
	`, userID, otherID).Scan(&exists)
	return err == nil && exists
}
