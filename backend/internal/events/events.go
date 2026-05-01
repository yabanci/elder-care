// Package events implements an in-process Server-Sent Events broadcaster
// for live alert streaming. Browsers open a long-lived `GET /api/events`
// connection; the server pushes a JSON event whenever a critical alert
// fires for the patient or any patient linked to the connecting user.
//
// Single-process only: subscriptions live in a map guarded by a Mutex.
// For a multi-instance deployment swap the in-memory broadcaster for
// PostgreSQL LISTEN/NOTIFY or Redis pub/sub — the public HTTP shape
// (text/event-stream) does not change.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/auth"
)

// Event is the JSON payload pushed to clients. Generic envelope so we
// can extend (alerts today, message ping tomorrow, etc.) without
// versioning the wire protocol.
type Event struct {
	Type      string `json:"type"`       // "alert"
	PatientID string `json:"patient_id"` // event subject
	AlertID   string `json:"alert_id,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

// Broker dispatches events to subscribers. Subscribers register with
// (userID, channel) — the broker computes "is this user a recipient for
// this patient's event?" by looking at patient_links + identity match.
type Broker struct {
	pool *pgxpool.Pool
	mu   sync.RWMutex
	subs map[string][]subscriber // keyed by user_id; multiple tabs allowed
}

type subscriber struct {
	ch chan Event
}

func NewBroker(pool *pgxpool.Pool) *Broker {
	return &Broker{pool: pool, subs: make(map[string][]subscriber)}
}

// subscribe registers a per-user channel and returns it + an unregister
// callback. Buffered to 16 so a slow client doesn't backpressure the
// publisher; if the buffer fills we drop the event (clients can refresh
// the alerts page to recover state).
func (b *Broker) subscribe(userID string) (chan Event, func()) {
	ch := make(chan Event, 16)
	sub := subscriber{ch: ch}
	b.mu.Lock()
	b.subs[userID] = append(b.subs[userID], sub)
	b.mu.Unlock()
	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.subs[userID]
		for i, s := range list {
			if s.ch == ch {
				b.subs[userID] = append(list[:i], list[i+1:]...)
				break
			}
		}
		if len(b.subs[userID]) == 0 {
			delete(b.subs, userID)
		}
		close(ch)
	}
	return ch, unsub
}

// PublishAlert sends an alert event to the patient and to every linked
// caregiver (doctor + family). Lookup happens once per publish — DB
// query is small but this is still a cost; for very high alert volume
// consider caching link lists with TTL.
func (b *Broker) PublishAlert(ctx context.Context, patientID, alertID, severity, kind string) {
	recipients := b.recipients(ctx, patientID)
	ev := Event{
		Type:      "alert",
		PatientID: patientID,
		AlertID:   alertID,
		Severity:  severity,
		Kind:      kind,
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, uid := range recipients {
		for _, s := range b.subs[uid] {
			// Non-blocking send: drop if subscriber is slow rather than
			// stalling the publisher.
			select {
			case s.ch <- ev:
			default:
			}
		}
	}
}

func (b *Broker) recipients(ctx context.Context, patientID string) []string {
	out := []string{patientID}
	rows, err := b.pool.Query(ctx, `
		SELECT linked_id FROM patient_links
		WHERE patient_id=$1 AND relation IN ('doctor', 'family')
	`, patientID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// Stream is the gin handler for GET /api/events. Holds the connection
// open, writes one `event: alert\ndata: {...json...}\n\n` block per
// published event, and closes when the client disconnects.
func (b *Broker) Stream(c *gin.Context) {
	userID := c.GetString(auth.CtxUserID)
	if userID == "" {
		c.AbortWithStatus(401)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // hint to nginx

	ch, unsub := b.subscribe(userID)
	defer unsub()

	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		c.AbortWithStatus(500)
		return
	}

	// Initial comment: confirms the stream is live (browsers don't fire
	// EventSource.onopen until they see at least one byte).
	_, _ = io.WriteString(c.Writer, ": connected\n\n")
	flusher.Flush()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, open := <-ch:
			if !open {
				return
			}
			payload, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Type, payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
