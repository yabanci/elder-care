package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"eldercare/backend/internal/audit"
	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/config"
	"eldercare/backend/internal/db"
	"eldercare/backend/internal/events"
	"eldercare/backend/internal/httpx"
	"eldercare/backend/internal/links"
	"eldercare/backend/internal/medications"
	"eldercare/backend/internal/messages"
	"eldercare/backend/internal/metrics"
	"eldercare/backend/internal/notes"
	"eldercare/backend/internal/plans"
	"eldercare/backend/internal/push"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run migrations and exit")
	flag.Parse()

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(context.Background(), pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if *migrateOnly {
		log.Println("migrations applied; exiting")
		return
	}

	// Long-lived background context for daily housekeeping; cancelled on
	// shutdown so retention sweep loops exit cleanly.
	bgCtx, cancelBg := context.WithCancel(context.Background())
	defer cancelBg()

	authSvc := auth.NewService(pool, cfg.JWTSecret, cfg.JWTTTLHours).WithSecureCookies(cfg.SecureCookies)
	metrics.StartRetentionSweep(bgCtx, pool)
	pushSvc := push.NewService(pool, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
	eventBroker := events.NewBroker(pool)
	metricsSvc := metrics.NewService(pool).
		WithNotifier(pushAdapter{pushSvc}).
		WithEventPublisher(eventBroker)
	medSvc := medications.NewService(pool)
	linksSvc := links.NewService(pool)
	msgSvc := messages.NewService(pool)
	plansSvc := plans.NewService(pool)
	notesSvc := notes.NewService(pool)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	corsCfg := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
		MaxAge:       12 * time.Hour,
	}
	// Mobile clients use Bearer tokens (no cookies needed). Allow "*" so
	// quick browser-based testing works; allow-credentials must be off
	// when origin is wildcard per the CORS spec. For a specific origin we
	// re-enable credentials so the optional web cookie path keeps working.
	if cfg.CORSOrigin == "*" {
		corsCfg.AllowAllOrigins = true
	} else {
		corsCfg.AllowOrigins = []string{cfg.CORSOrigin}
		corsCfg.AllowCredentials = true
	}
	r.Use(cors.New(corsCfg))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// public — login/register sit behind a per-IP rate-limit. 5 requests
	// per minute per IP discourages credential-stuffing without bothering
	// real users (5 typos in 60s is well above normal). For multi-instance
	// deployments swap for a Redis-backed limiter.
	loginLimiter := httpx.NewTokenBucket(5, 12*time.Second)
	r.POST("/api/auth/register", httpx.RateLimitMiddleware(loginLimiter), authSvc.Register)
	r.POST("/api/auth/login", httpx.RateLimitMiddleware(loginLimiter), authSvc.Login)
	r.POST("/api/auth/logout", authSvc.Logout)

	api := r.Group("/api")
	api.Use(authSvc.Middleware(), audit.Middleware(pool))

	api.GET("/me", authSvc.Me)
	api.PATCH("/me", authSvc.UpdateMe)

	// patient self-endpoints
	api.POST("/metrics", metricsSvc.CreateForSelf)
	api.GET("/metrics", metricsSvc.List)
	api.GET("/metrics/summary", metricsSvc.Summary)
	api.GET("/alerts", metricsSvc.ListAlerts)
	api.POST("/alerts/:id/acknowledge", metricsSvc.AcknowledgeAlert)

	api.POST("/medications", medSvc.Create)
	api.GET("/medications", medSvc.List)
	api.GET("/medications/today", medSvc.Today)
	api.DELETE("/medications/:id", medSvc.Deactivate)
	api.POST("/medications/:id/log", medSvc.LogDose)

	api.GET("/caregivers", linksSvc.MyCaregivers)

	// doctor/family viewing a specific patient
	api.GET("/patients", linksSvc.MyPatients)
	api.POST("/patients/link", linksSvc.Link)
	api.GET("/patients/:patientID/metrics", metricsSvc.List)
	api.GET("/patients/:patientID/metrics/summary", metricsSvc.Summary)
	api.GET("/patients/:patientID/alerts", metricsSvc.ListAlerts)
	api.GET("/patients/:patientID/medications", medSvc.List)
	api.GET("/patients/:patientID/medications/today", medSvc.Today)
	api.POST("/patients/:patientID/medications", medSvc.Create) // doctor prescribes
	api.POST("/patients/:patientID/metrics", metricsSvc.CreateForPatient)
	api.POST("/patients/:patientID/notes", notesSvc.Create)
	api.GET("/patients/:patientID/notes", notesSvc.List)

	// plans (weekly schedule) — patients only
	plansGroup := api.Group("/plans", auth.RequireRole("patient"))
	plansGroup.GET("", plansSvc.List)
	plansGroup.POST("", plansSvc.Create)
	plansGroup.PATCH("/:id", plansSvc.Update)
	plansGroup.DELETE("/:id", plansSvc.Delete)

	// messaging
	api.POST("/messages", msgSvc.Send)
	api.GET("/messages/:otherID", msgSvc.Thread)

	// live alert stream (SSE) — long-lived; auth via cookie/Bearer like
	// every other /api endpoint.
	api.GET("/events", eventBroker.Stream)

	// web push
	r.GET("/api/push/public-key", pushSvc.PublicKey) // public — needed to subscribe
	api.POST("/api/push/subscribe", pushSvc.Subscribe)
	api.DELETE("/api/push/subscribe", pushSvc.Unsubscribe)

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown: SIGTERM (k8s, docker stop) and SIGINT (Ctrl-C)
	// trigger ListenAndServe to return; we then give in-flight requests
	// up to 15 seconds to finish before closing the DB pool.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server listening on %s", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	sig := <-stop
	log.Printf("received %s, shutting down...", sig)

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	// Wait for in-flight push deliveries to finish (best-effort, capped
	// by the same shutdown deadline). Without this, the process can exit
	// while a goroutine is mid-HTTP-call to FCM/APNs/Mozilla.
	pushSvc.Drain(shutdownCtx)
	log.Println("server stopped")
}

// pushAdapter bridges metrics.Notifier (a pure interface, no push-pkg
// dep) to push.Service. Lives here so internal/metrics doesn't import
// internal/push (avoids a backend-wide import cycle if push ever needs
// to query metrics).
type pushAdapter struct {
	svc *push.Service
}

func (a pushAdapter) SendToUser(ctx context.Context, userID string, p metrics.PushPayload) {
	a.svc.SendToUser(ctx, userID, push.AlertPayload{
		Title:     p.Title,
		Body:      p.Body,
		URL:       p.URL,
		Severity:  p.Severity,
		PatientID: p.PatientID,
		AlertID:   p.AlertID,
	})
}
