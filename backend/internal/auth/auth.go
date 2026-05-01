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

	"eldercare/backend/internal/httpx"
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
	TZ                string     `json:"tz"`
}

type Service struct {
	pool          *pgxpool.Pool
	jwtSecret     []byte
	jwtTTL        time.Duration
	secureCookies bool
}

// NewService constructs an auth service. secureCookies should be true in
// any environment that has TLS terminated *anywhere* in the request path
// (production, staging behind a real proxy). Set false only for local
// dev on plain http.
func NewService(pool *pgxpool.Pool, secret string, ttlHours int) *Service {
	return &Service{
		pool:          pool,
		jwtSecret:     []byte(secret),
		jwtTTL:        time.Duration(ttlHours) * time.Hour,
		secureCookies: true,
	}
}

// WithSecureCookies returns the same service with the Secure-cookie flag
// overridden. Chain it from cmd/server when initialising:
//
//	auth.NewService(...).WithSecureCookies(cfg.SecureCookies)
func (s *Service) WithSecureCookies(secure bool) *Service {
	s.secureCookies = secure
	return s
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

// CookieName is the HttpOnly cookie that holds the JWT for browser
// clients. Bearer tokens in the Authorization header continue to work for
// programmatic clients (integration tests, CLI scripts).
const CookieName = "eldercare_token"

// setAuthCookie writes a fresh JWT cookie on the response. SameSite=Lax
// is sufficient because we only attach the cookie to top-level navigations
// + same-origin XHR; the API and UI share an origin in production.
//
// Secure is controlled by the operator via SECURE_COOKIES env (default
// true). We deliberately do NOT autodetect from X-Forwarded-Proto — that
// header is attacker-controllable on a misconfigured proxy and would
// flip Secure to false silently.
func (s *Service) setAuthCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, token, int(ttl.Seconds()), "/", "", s.secureCookies, true /*httpOnly*/)
}

func (s *Service) clearAuthCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, "", -1, "/", "", s.secureCookies, true)
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

	onboarded := req.Role != "patient"

	// Invite codes are 8 hex chars (~4B combos). Collisions are unlikely but
	// possible; retry up to 5 times before giving up so a register isn't
	// rejected for a transient unique-violation. Email collisions are
	// surfaced as 400 on the first attempt.
	var u User
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		inviteCode, err := generateInviteCode()
		if err != nil {
			httpx.Internal(c, fmt.Errorf("generate invite code: %w", err))
			return
		}
		err = s.pool.QueryRow(c.Request.Context(), `
			INSERT INTO users(email, password_hash, full_name, role, phone, birth_date, invite_code, onboarded)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id, email, full_name, role, phone, birth_date, invite_code,
			          height_cm, chronic_conditions, bp_norm, prescribed_meds, onboarded, lang, tz
		`, strings.ToLower(req.Email), string(hash), req.FullName, req.Role, phone, birth, inviteCode, onboarded).
			Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.Phone, &u.BirthDate, &u.InviteCode,
				&u.HeightCm, &u.ChronicConditions, &u.BPNorm, &u.PrescribedMeds, &u.Onboarded, &u.Lang, &u.TZ)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		if strings.Contains(err.Error(), "users_email_key") {
			httpx.BadRequest(c, "email already registered")
			return
		}
		if !strings.Contains(err.Error(), "users_invite_code_key") {
			httpx.Internal(c, err)
			return
		}
		// Invite-code collision: retry with a fresh code.
	}
	if lastErr != nil {
		httpx.Internal(c, fmt.Errorf("invite code retries exhausted: %w", lastErr))
		return
	}

	token, err := s.issueToken(u.ID, u.Role)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	s.setAuthCookie(c, token, s.jwtTTL)
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
		       height_cm, chronic_conditions, bp_norm, prescribed_meds, onboarded, lang, tz, password_hash
		FROM users WHERE email=$1
	`, strings.ToLower(req.Email)).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.Phone, &u.BirthDate, &u.InviteCode,
			&u.HeightCm, &u.ChronicConditions, &u.BPNorm, &u.PrescribedMeds, &u.Onboarded, &u.Lang, &u.TZ, &hash)
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
	s.setAuthCookie(c, token, s.jwtTTL)
	c.JSON(http.StatusOK, authResp{Token: token, User: u})
}

// Logout clears the auth cookie. JWTs are stateless so we cannot revoke
// the token itself — but clearing the cookie removes the browser's
// ability to authenticate. Programmatic Bearer holders are unaffected.
func (s *Service) Logout(c *gin.Context) {
	s.clearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
		       height_cm, chronic_conditions, bp_norm, prescribed_meds, onboarded, lang, tz
		FROM users WHERE id=$1
	`, id).Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.Phone, &u.BirthDate, &u.InviteCode,
		&u.HeightCm, &u.ChronicConditions, &u.BPNorm, &u.PrescribedMeds, &u.Onboarded, &u.Lang, &u.TZ)
	return u, err
}

type updateProfileReq struct {
	HeightCm          *int    `json:"height_cm"`
	ChronicConditions *string `json:"chronic_conditions"`
	BPNorm            *string `json:"bp_norm"`
	PrescribedMeds    *string `json:"prescribed_meds"`
	Onboarded         *bool   `json:"onboarded"`
	Lang              *string `json:"lang"`
	TZ                *string `json:"tz"`
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
	if req.TZ != nil {
		// Validate via Go's tzdata: time.LoadLocation parses the IANA name
		// and errors on garbage. Bound the string length to prevent abuse.
		if len(*req.TZ) > 64 {
			httpx.BadRequest(c, "tz too long")
			return
		}
		if _, err := time.LoadLocation(*req.TZ); err != nil {
			httpx.BadRequest(c, "invalid tz, expected IANA name like 'Asia/Almaty'")
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
			tz                 = COALESCE($8, tz),
			updated_at         = now()
		WHERE id=$1
	`, userID, req.HeightCm, req.ChronicConditions, req.BPNorm, req.PrescribedMeds, req.Onboarded, req.Lang, req.TZ)
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
		raw := s.extractToken(c)
		if raw == "" {
			httpx.Unauthorized(c, "missing token")
			return
		}
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

// extractToken returns the JWT from the request, preferring the
// Authorization header (programmatic clients) and falling back to the
// HttpOnly cookie (browser sessions). Empty string means no credential
// was presented.
func (s *Service) extractToken(c *gin.Context) string {
	if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	if cookie, err := c.Cookie(CookieName); err == nil && cookie != "" {
		return cookie
	}
	return ""
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

// generateInviteCode returns an 8-char uppercase hex code from 4 bytes of
// crypto/rand. Errors from the random source are surfaced — silently
// returning a zero-byte code would degrade to a single deterministic
// invite that all newcomers would collide on.
func generateInviteCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand read: %w", err)
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}
