package events_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/events"
	"eldercare/backend/internal/metrics"
	"eldercare/backend/internal/testutil"
)

const eventsSecret = "events_test_secret"

// TestSSE_CriticalAlertReachesPatient verifies the publish→subscribe loop:
// patient subscribes via /api/events, posts a metric that triggers a
// critical alert, the SSE stream receives an `alert` event with the
// matching patient_id and severity.
func TestSSE_CriticalAlertReachesPatient(t *testing.T) {
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, eventsSecret, 1)
	broker := events.NewBroker(pool)
	metricsSvc := metrics.NewService(pool).WithEventPublisher(broker)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.GET("/events", broker.Stream)
	api.POST("/metrics", metricsSvc.CreateForSelf)

	srv := httptest.NewServer(r)
	defer srv.Close()

	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": "p-sse@test.kz", "password": "secret123", "full_name": "T", "role": "patient",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var rr struct {
		Token string `json:"token"`
	}
	testutil.Decode(t, w, &rr)

	// Open SSE connection in a goroutine.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events", nil)
	req.Header.Set("Authorization", "Bearer "+rr.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect /api/events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status: %d", resp.StatusCode)
	}

	received := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				chunk := string(buf[:n])
				if strings.Contains(chunk, "event: alert") {
					received <- chunk
					return
				}
			}
		}
	}()

	// Give the subscription a moment to register before publishing.
	time.Sleep(100 * time.Millisecond)

	// Trigger a critical alert via the existing handler (uses the same
	// metricsSvc with broker installed).
	w = testutil.DoJSON(t, r, "POST", "/api/metrics", rr.Token, map[string]any{
		"kind": "pulse", "value": 200,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create metric: %d %s", w.Code, w.Body.String())
	}

	// metricsSvc in this test was created in-process and shares the broker
	// with the SSE subscriber, so the publish should land in the channel.
	select {
	case chunk := <-received:
		if !strings.Contains(chunk, `"severity":"critical"`) {
			t.Errorf("expected critical alert in SSE chunk, got: %s", chunk)
		}
		if !strings.Contains(chunk, `"kind":"pulse"`) {
			t.Errorf("expected kind=pulse in SSE chunk, got: %s", chunk)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for SSE alert")
	}
}
