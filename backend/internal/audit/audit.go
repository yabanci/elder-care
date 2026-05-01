// Package audit records access to patient health information so that
// who-saw-what can be reconstructed after the fact. Required for any
// medical-data system claiming compliance hygiene.
//
// Recorded fields: actor (user_id + role), target patient, request
// (method/path/status), source (IP, user-agent), timestamp. Does NOT
// store request bodies or response payloads — those would itself be PII.
package audit

import (
	"context"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/auth"
)

// patientIDPattern matches `:patientID` segments and captures the UUID.
// We restrict to UUID syntax so e.g. "/api/me" doesn't get parsed as
// a patient_id.
var patientIDPattern = regexp.MustCompile(
	`^/api/patients/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(/|$)`,
)

// extractPatientID returns the UUID from /api/patients/:id/... routes,
// or empty string if the path does not target a specific patient.
func extractPatientID(path string) string {
	m := patientIDPattern.FindStringSubmatch(path)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// Middleware returns a gin middleware that logs every authenticated
// request to PHI-touching paths into audit_log. Logs after the handler
// completes so we capture the response status. Logging failures are
// best-effort: errors are swallowed so a transient DB issue cannot lock
// up patient access.
func Middleware(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Skip non-PHI paths. We log everything under /api/patients/ and
		// /api/me — these are the only routes that surface patient data.
		// Login/register stay out of audit.
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/patients/") && path != "/api/me" {
			return
		}

		actorID := c.GetString(auth.CtxUserID)
		actorRole := c.GetString(auth.CtxRole)
		if actorID == "" {
			return // unauthenticated; auth middleware should have already 401'd
		}
		patientID := extractPatientID(path)
		if patientID == "" && path == "/api/me" {
			// Self-access: actor IS the patient.
			patientID = actorID
		}

		ip := clientIP(c)
		ua := c.Request.UserAgent()
		method := c.Request.Method
		status := c.Writer.Status()

		// Run async so audit write does not add latency to user requests;
		// background context so the goroutine survives request return.
		go func() {
			_, err := pool.Exec(context.Background(), `
				INSERT INTO audit_log(actor_id, actor_role, patient_id, method, path, status, ip, user_agent)
				VALUES($1, $2, $3, $4, $5, $6, $7, $8)
			`, actorID, actorRole, nullableUUID(patientID), method, path, status, ip, ua)
			if err != nil {
				log.Printf("audit: insert failed: %v", err)
			}
		}()
	}
}

// nullableUUID returns nil for empty strings so the column stays NULL
// rather than being set to '' (which would fail UUID parsing in PG).
func nullableUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// clientIP extracts the connection-level remote IP. For a real reverse
// proxy you would check X-Forwarded-For after validating against an
// allowlist of trusted proxies — out of scope for thesis MVP.
func clientIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
