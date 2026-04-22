package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/arsenozhetov/elder-care/backend/internal/httpx"
)

type User struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	FullName          string     `json:"full_name"`
	Role              string     `json:"role"`
	Phone             *string    `json:"phone,omitempty"`
	BirthDate         *time.Time `json:"birth_date,omitempty"`
	InviteCode        *string    `json:"invite_code,omitempty"`
	HeightCm          *int       `json:"height_cm,omitempty"`
	ChronicConditions *string    `json:"chronic_conditions,omitempty"`
	BPNorm            *string    `json:"bp_norm,omitempty"`
	PrescribedMeds    *string    `json:"prescribed_meds,omitempty"`
	Onboarded         bool       `json:"onboarded"`
	Lang              *string    `json:"lang,omitempty"`
}

type Service struct {
	pool      *pgxpool.Pool
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewService(pool *pgxpool.Pool, secret string, ttlHours int) *Service {
	return &Service{
		pool:      pool,
		jwtSecret: []byte(secret),
		jwtTTL:    time.Duration(ttlHours) * time.Hour,
	}
}

type registerReq struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FullName  string `json:"full_name" binding:"required"`
	Role      string `json:"role" binding:"required,oneof=patient doctor family"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type authResp struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func (s *Service) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	inviteCode := generateInviteCode()

	var birth interface{}
	if req.BirthDate != "" {
		t, err := time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			httpx.BadRequest(c, "invalid birth_date, expected YYYY-MM-DD")
			return
		}
		birth = t
	}

	var phone interface{}
	if req.Phone != "" {
		phone = req.Phone
	}

	var u User
	onboarded := req.Role != "patient"
	err = s.pool.QueryRow(c.Request.Context(), `
		INSERT INTO users(email, password_hash, full_name, role, phone, birth_date, invite_code, onboarded)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, email, full_name, role, phone, birth_date, invite_code,
		          height_cm, chronic_conditions, bp_norm, prescribed_meds, onboarded, lang
	`, strings.ToLower(req.Email), string(hash), req.FullName, req.Role, phone, birth, inviteCode, onboarded).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.Phone, &u.BirthDate, &u.InviteCode,
			&u.HeightCm, &u.ChronicConditions, &u.BPNorm, &u.PrescribedMeds, &u.Onboarded, &u.Lang)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			httpx.BadRequest(c, "email already registered")
			return
		}
		httpx.Internal(c, err)
		return
	}

	token, err := s.issueToken(u.ID, u.Role)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, authResp{Token: token, User: u})
}

func (s *Service) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	var (
		u    User
		hash string
	)
	err := s.pool.QueryRow(c.Request.Context(), `
		SELECT id, email, full_name, role, phone, birth_date, invite_code,
		       height_cm, chronic_conditions, bp_norm, prescribed_meds, onboarded, lang, password_hash
		FROM users WHERE email=$1
	`, strings.ToLower(req.Email)).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.Phone, &u.BirthDate, &u.InviteCode,
			&u.HeightCm, &u.ChronicConditions, &u.BPNorm, &u.PrescribedMeds, &u.Onboarded, &u.Lang, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Unauthorized(c, "invalid credentials")
			return
		}
		httpx.Internal(c, err)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		httpx.Unauthorized(c, "invalid credentials")
		return
	}

	token, err := s.issueToken(u.ID, u.Role)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, authResp{Token: token, User: u})
}

func (s *Service) Me(c *gin.Context) {
	userID := c.GetString(CtxUserID)
	u, err := s.GetUser(c.Request.Context(), userID)
	if err != nil {
		httpx.HandleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (s *Service) GetUser(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, full_name, role, phone, birth_date, invite_code,
		       height_cm, chronic_conditions, bp_norm, prescribed_meds, onboarded, lang
		FROM users WHERE id=$1
	`, id).Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.Phone, &u.BirthDate, &u.InviteCode,
		&u.HeightCm, &u.ChronicConditions, &u.BPNorm, &u.PrescribedMeds, &u.Onboarded, &u.Lang)
	return u, err
}

type updateProfileReq struct {
	HeightCm          *int    `json:"height_cm"`
	ChronicConditions *string `json:"chronic_conditions"`
	BPNorm            *string `json:"bp_norm"`
	PrescribedMeds    *string `json:"prescribed_meds"`
	Onboarded         *bool   `json:"onboarded"`
	Lang              *string `json:"lang"`
}

func (s *Service) UpdateMe(c *gin.Context) {
	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	if req.Lang != nil {
		switch *req.Lang {
		case "ru", "kk", "en":
		default:
			httpx.BadRequest(c, "invalid lang, expected one of: ru, kk, en")
			return
		}
	}
	userID := c.GetString(CtxUserID)
	_, err := s.pool.Exec(c.Request.Context(), `
		UPDATE users SET
			height_cm          = COALESCE($2, height_cm),
			chronic_conditions = COALESCE($3, chronic_conditions),
			bp_norm            = COALESCE($4, bp_norm),
			prescribed_meds    = COALESCE($5, prescribed_meds),
			onboarded          = COALESCE($6, onboarded),
			lang               = COALESCE($7, lang),
			updated_at         = now()
		WHERE id=$1
	`, userID, req.HeightCm, req.ChronicConditions, req.BPNorm, req.PrescribedMeds, req.Onboarded, req.Lang)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	u, err := s.GetUser(c.Request.Context(), userID)
	if err != nil {
		httpx.HandleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (s *Service) issueToken(userID, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(s.jwtTTL).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

const (
	CtxUserID = "userID"
	CtxRole   = "role"
)

func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			httpx.Unauthorized(c, "missing bearer token")
			return
		}
		raw := strings.TrimPrefix(header, "Bearer ")
		token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return s.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			httpx.Unauthorized(c, "invalid token")
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			httpx.Unauthorized(c, "invalid claims")
			return
		}
		sub, _ := claims["sub"].(string)
		role, _ := claims["role"].(string)
		c.Set(CtxUserID, sub)
		c.Set(CtxRole, role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString(CtxRole)
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		httpx.Forbidden(c, "insufficient role")
	}
}

func generateInviteCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}
