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
