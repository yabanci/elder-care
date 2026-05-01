// Package push delivers Web Push notifications to subscribed users when
// critical alerts fire. Uses VAPID for sender authentication; the public
// key is exposed to the frontend so it can subscribe with PushManager.
//
// Sending is best-effort: failures are logged, never propagated to the
// HTTP request that triggered the alert. Stale subscriptions (HTTP 404
// or 410 from the push service) are pruned automatically.
package push

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/httpx"
)

// Service holds the VAPID keypair + DB pool. One instance per process.
type Service struct {
	pool       *pgxpool.Pool
	publicKey  string
	privateKey string
	subject    string // "mailto:..." or app URL; required by webpush spec
}

// NewService initialises with VAPID keys. If publicKey is empty, push is
// effectively disabled — Send() and the subscribe endpoint short-circuit
// without persisting anything. This lets dev environments run without
// configuring VAPID.
func NewService(pool *pgxpool.Pool, publicKey, privateKey, subject string) *Service {
	if subject == "" {
		subject = "mailto:admin@eldercare.local"
	}
	return &Service{pool: pool, publicKey: publicKey, privateKey: privateKey, subject: subject}
}

// Enabled reports whether the service has a usable VAPID keypair.
func (s *Service) Enabled() bool {
	return s.publicKey != "" && s.privateKey != ""
}

// PublicKey handler exposes the VAPID public key so the frontend can
// pass it to PushManager.subscribe().
func (s *Service) PublicKey(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"public_key": s.publicKey,
		"enabled":    s.Enabled(),
	})
}

type subscribeReq struct {
	Endpoint string `json:"endpoint" binding:"required,url"`
	Keys     struct {
		P256dh string `json:"p256dh" binding:"required"`
		Auth   string `json:"auth" binding:"required"`
	} `json:"keys"`
}

// Subscribe persists a push subscription for the current user. Idempotent
// per (user_id, endpoint) — re-subscribing replaces the auth keys.
func (s *Service) Subscribe(c *gin.Context) {
	if !s.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "push notifications are not configured on this server"})
		return
	}
	var req subscribeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	userID := c.GetString(auth.CtxUserID)
	_, err := s.pool.Exec(c.Request.Context(), `
		INSERT INTO push_subscriptions(user_id, endpoint, p256dh, auth)
		VALUES($1, $2, $3, $4)
		ON CONFLICT (user_id, endpoint) DO UPDATE SET p256dh=EXCLUDED.p256dh, auth=EXCLUDED.auth
	`, userID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Unsubscribe removes a subscription (called when the browser revokes
// permission or the user explicitly opts out).
func (s *Service) Unsubscribe(c *gin.Context) {
	endpoint := c.Query("endpoint")
	if endpoint == "" {
		httpx.BadRequest(c, "endpoint query param required")
		return
	}
	userID := c.GetString(auth.CtxUserID)
	_, err := s.pool.Exec(c.Request.Context(),
		`DELETE FROM push_subscriptions WHERE user_id=$1 AND endpoint=$2`,
		userID, endpoint)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// AlertPayload is the JSON body delivered as the push notification. The
// service worker on the client decides how to render it.
type AlertPayload struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	Severity  string `json:"severity"`
	PatientID string `json:"patient_id"`
	AlertID   string `json:"alert_id"`
}

// SendToUser pushes the payload to every active subscription for `userID`.
// Best-effort: per-subscription errors are logged; HTTP 404/410 cause
// pruning. Safe to call from request handlers — never blocks longer than
// the underlying HTTP timeout.
func (s *Service) SendToUser(ctx context.Context, userID string, payload AlertPayload) {
	if !s.Enabled() {
		return
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id=$1`,
		userID)
	if err != nil {
		log.Printf("push: list subs for %s: %v", userID, err)
		return
	}
	defer rows.Close()

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("push: marshal payload: %v", err)
		return
	}

	type subRow struct {
		ID, Endpoint, P256dh, Auth string
	}
	var subs []subRow
	for rows.Next() {
		var r subRow
		if err := rows.Scan(&r.ID, &r.Endpoint, &r.P256dh, &r.Auth); err != nil {
			log.Printf("push: scan sub row: %v", err)
			continue
		}
		subs = append(subs, r)
	}

	for _, r := range subs {
		sub := &webpush.Subscription{
			Endpoint: r.Endpoint,
			Keys:     webpush.Keys{P256dh: r.P256dh, Auth: r.Auth},
		}
		opts := &webpush.Options{
			VAPIDPublicKey:  s.publicKey,
			VAPIDPrivateKey: s.privateKey,
			Subscriber:      s.subject,
			TTL:             60,
		}
		resp, err := webpush.SendNotification(body, sub, opts)
		if err != nil {
			log.Printf("push: send to sub %s: %v", r.ID, err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			if _, err := s.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE id=$1`, r.ID); err != nil {
				log.Printf("push: prune stale sub %s: %v", r.ID, err)
			}
			continue
		}
		if resp.StatusCode >= 400 {
			log.Printf("push: sub %s returned %d", r.ID, resp.StatusCode)
		}
	}
}

// Disabled is a sentinel error returned when push service is not configured.
var Disabled = errors.New("push notifications disabled")
