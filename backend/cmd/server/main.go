package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/arsenozhetov/elder-care/backend/internal/auth"
	"github.com/arsenozhetov/elder-care/backend/internal/config"
	"github.com/arsenozhetov/elder-care/backend/internal/db"
	"github.com/arsenozhetov/elder-care/backend/internal/links"
	"github.com/arsenozhetov/elder-care/backend/internal/medications"
	"github.com/arsenozhetov/elder-care/backend/internal/messages"
	"github.com/arsenozhetov/elder-care/backend/internal/metrics"
	"github.com/arsenozhetov/elder-care/backend/internal/plans"
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

	authSvc := auth.NewService(pool, cfg.JWTSecret, cfg.JWTTTLHours)
	metricsSvc := metrics.NewService(pool)
	medSvc := medications.NewService(pool)
	linksSvc := links.NewService(pool)
	msgSvc := messages.NewService(pool)
	plansSvc := plans.NewService(pool)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// public
	r.POST("/api/auth/register", authSvc.Register)
	r.POST("/api/auth/login", authSvc.Login)

	api := r.Group("/api")
	api.Use(authSvc.Middleware())

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
	api.POST("/patients/:patientID/metrics", metricsSvc.CreateForPatient)

	// plans (weekly schedule)
	api.GET("/plans", plansSvc.List)
	api.POST("/plans", plansSvc.Create)
	api.PATCH("/plans/:id", plansSvc.Update)
	api.DELETE("/plans/:id", plansSvc.Delete)

	// messaging
	api.POST("/messages", msgSvc.Send)
	api.GET("/messages/:otherID", msgSvc.Thread)

	log.Printf("server listening on %s", cfg.ServerAddr)
	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
