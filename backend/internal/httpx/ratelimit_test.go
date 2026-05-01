package httpx

import (
	"testing"
	"time"
)

func TestTokenBucket_AllowsUpToCapacity(t *testing.T) {
	b := NewTokenBucket(3, time.Second)
	for i := 0; i < 3; i++ {
		if !b.Allow("k") {
			t.Fatalf("attempt %d: expected allowed", i)
		}
	}
	if b.Allow("k") {
		t.Errorf("4th attempt should be denied")
	}
}

func TestTokenBucket_PerKeyIsolation(t *testing.T) {
	b := NewTokenBucket(1, time.Second)
	if !b.Allow("alice") {
		t.Fatalf("alice first should pass")
	}
	if b.Allow("alice") {
		t.Errorf("alice second should fail (bucket size 1)")
	}
	if !b.Allow("bob") {
		t.Errorf("bob's first should pass; per-key bucket")
	}
}

func TestTokenBucket_Refills(t *testing.T) {
	b := NewTokenBucket(1, 50*time.Millisecond)
	if !b.Allow("k") {
		t.Fatal("first should pass")
	}
	if b.Allow("k") {
		t.Fatal("second should fail")
	}
	time.Sleep(60 * time.Millisecond)
	if !b.Allow("k") {
		t.Errorf("after refill window, should pass again")
	}
}

func TestTokenBucket_PrunesStaleEntries(t *testing.T) {
	// Use a 10ms staleness window so the test runs fast.
	b := NewTokenBucket(1, time.Second)
	b.staleAfter = 10 * time.Millisecond

	// Fill the map with N distinct keys.
	for i := 0; i < pruneInterval+10; i++ {
		b.Allow("ip-" + string(rune(i)))
	}
	// All entries should be present (no pruning yet — they're all fresh).
	if got := b.Size(); got < pruneInterval {
		t.Fatalf("expected at least %d entries before pruning, got %d", pruneInterval, got)
	}

	// Wait past staleness, then trigger prune via the sampling counter.
	time.Sleep(20 * time.Millisecond)
	for i := 0; i < pruneInterval; i++ {
		b.Allow("trigger") // sweeps when calls % pruneInterval == 0
	}

	// All the original "ip-N" entries are stale; "trigger" stays fresh.
	if got := b.Size(); got > 5 {
		t.Errorf("after prune expected ≤5 entries, got %d", got)
	}
}
