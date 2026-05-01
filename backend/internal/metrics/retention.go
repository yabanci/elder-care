package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AlgorithmRunsRetentionDays bounds how long we keep per-invocation
// telemetry. Anything older is dropped on the daily sweep. 90 days gives
// the offline replay corpus 3 months of recent history without unbounded
// growth (10 readings/patient/day × 365 days × N patients gets large).
const AlgorithmRunsRetentionDays = 90

// AuditLogRetentionDays bounds how long PHI-access records are kept.
// 180 days covers the typical clinical-incident-investigation window;
// adjust upward if a compliance regime requires it.
const AuditLogRetentionDays = 180

// StartRetentionSweep runs an initial cleanup immediately, then loops
// once every 24 hours until ctx is cancelled. Designed to be launched
// from cmd/server in a goroutine; survives transient DB errors by
// logging and retrying on the next tick.
func StartRetentionSweep(ctx context.Context, pool *pgxpool.Pool) {
	go func() {
		// First sweep right away so a long-running deployment without
		// process restarts still bounds the table size.
		runSweep(ctx, pool)
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runSweep(ctx, pool)
			}
		}
	}()
}

func runSweep(ctx context.Context, pool *pgxpool.Pool) {
	sweepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pruneTable(sweepCtx, pool, "algorithm_runs", AlgorithmRunsRetentionDays)
	pruneTable(sweepCtx, pool, "audit_log", AuditLogRetentionDays)
}

// pruneTable runs `DELETE FROM <table> WHERE created_at < now() - interval 'N days'`.
// The table name and N are code constants, never user input — interpolated
// directly to sidestep pgx interval-encoding quirks; not an injection risk.
func pruneTable(ctx context.Context, pool *pgxpool.Pool, table string, days int) {
	tag, err := pool.Exec(ctx, fmt.Sprintf(
		`DELETE FROM %s WHERE created_at < now() - interval '%d days'`,
		table, days,
	))
	if err != nil {
		log.Printf("retention: %s sweep failed: %v", table, err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("retention: pruned %d %s rows older than %d days", n, table, days)
	}
}
