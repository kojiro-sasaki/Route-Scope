package trace

import (
	"net"
	"testing"
	"time"
)

func TestHopUpdate(t *testing.T) {
	hop := NewHop(3)

	addr := net.ParseIP("192.168.1.1")

	hop.Update(
		addr,
		10*time.Millisecond,
		true,
	)

	snapshot := hop.Snapshot()

	if snapshot.TTL != 3 {
		t.Fatalf(
			"TTL = %d, want 3",
			snapshot.TTL,
		)
	}

	if !snapshot.Addr.Equal(addr) {
		t.Fatalf(
			"Addr = %v, want %v",
			snapshot.Addr,
			addr,
		)
	}

	if snapshot.Stats.Received != 1 {
		t.Fatalf(
			"Received = %d, want 1",
			snapshot.Stats.Received,
		)
	}

	if snapshot.Stats.Last != 10*time.Millisecond {
		t.Fatalf(
			"Last = %v, want 10ms",
			snapshot.Stats.Last,
		)
	}
}

func TestHopTimeout(t *testing.T) {
	hop := NewHop(2)

	hop.Update(
		nil,
		0,
		false,
	)

	snapshot := hop.Snapshot()

	if snapshot.Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			snapshot.Stats.Sent,
		)
	}

	if snapshot.Stats.Received != 0 {
		t.Fatalf(
			"Received = %d, want 0",
			snapshot.Stats.Received,
		)
	}

	if snapshot.Stats.Loss() != 100 {
		t.Fatalf(
			"Loss = %.2f, want 100",
			snapshot.Stats.Loss(),
		)
	}
}

func TestHopSnapshotCopiesIP(t *testing.T) {
	hop := NewHop(1)

	addr := net.ParseIP("10.0.0.1")

	hop.Update(
		addr,
		time.Millisecond,
		true,
	)

	snapshot := hop.Snapshot()

	// Меняем исходный IP.
	addr[0] = 255

	if snapshot.Addr.String() != "10.0.0.1" {
		t.Fatalf(
			"Snapshot IP was modified: %v",
			snapshot.Addr,
		)
	}
}
