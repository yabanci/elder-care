package audit_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/audit"
	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/links"
	"eldercare/backend/internal/metrics"
	"eldercare/backend/internal/testutil"
)

const auditSecret = "audit_test_secret"

func setupAudit(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, auditSecret, 1)
	metricsSvc := metrics.NewService(pool)
	linksSvc := links.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware(), audit.Middleware(pool))
	api.GET("/me", authSvc.Me)
	api.POST("/metrics", metricsSvc.CreateForSelf)
	api.GET("/patients", linksSvc.MyPatients)
	api.GET("/patients/:patientID/metrics", metricsSvc.List)
	return r, pool
}

// TestAudit_RecordsMeAccess checks that hitting /api/me appears in
// audit_log with the actor's user_id and the same id as patient_id
// (self-access pattern).
func TestAudit_RecordsMeAccess(t *testing.T) {
	r, pool := setupAudit(t)

	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": "p-audit@test.kz", "password": "secret123", "full_name": "T", "role": "patient",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	testutil.Decode(t, w, &resp)

	w = testutil.DoJSON(t, r, "GET", "/api/me", resp.Token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("/me: %d", w.Code)
	}

	// Audit insert is async (goroutine); poll briefly.
	deadline := time.Now().Add(2 * time.Second)
	var actor, patient string
	var status int
	for time.Now().Before(deadline) {
		err := pool.QueryRow(context.Background(), `
			SELECT actor_id::text, patient_id::text, status FROM audit_log
			WHERE path='/api/me' AND actor_id=$1
			ORDER BY created_at DESC LIMIT 1
		`, resp.User.ID).Scan(&actor, &patient, &status)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if actor != resp.User.ID {
		t.Errorf("audit actor: got %q want %q", actor, resp.User.ID)
	}
	if patient != resp.User.ID {
		t.Errorf("audit patient_id should equal actor for /api/me self-access; got %q want %q", patient, resp.User.ID)
	}
	if status != http.StatusOK {
		t.Errorf("audit status: got %d want 200", status)
	}
}
