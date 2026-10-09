package trace

import (
	"net"
	"testing"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

func TestNewHop(t *testing.T) {
	hop := NewHop(5)

	if hop.TTL != 5 {
		t.Fatalf(
			"TTL = %d, want 5",
			hop.TTL,
		)
	}

	if hop.Addr != nil {
		t.Fatalf(
			"Addr = %v, want nil",
			hop.Addr,
		)
	}

	if hop.Status != probe.StatusUnknown {
		t.Fatalf(
			"Status = %d, want %d",
			hop.Status,
			probe.StatusUnknown,
		)
	}

	if hop.Stats.Sent != 0 {
		t.Fatalf(
			"Sent = %d, want 0",
			hop.Stats.Sent,
		)
	}
}

func TestHopUpdateSuccess(t *testing.T) {
	hop := NewHop(1)

	addr := net.ParseIP(
		"192.168.1.1",
	)

	rtt := 5 * time.Millisecond

	hop.Update(
		addr,
		rtt,
		probe.StatusSuccess,
	)

	if hop.Status != probe.StatusSuccess {
		t.Fatalf(
			"Status = %d, want %d",
			hop.Status,
			probe.StatusSuccess,
		)
	}

	if !hop.Addr.Equal(addr) {
		t.Fatalf(
			"Addr = %s, want %s",
			hop.Addr,
			addr,
		)
	}

	if hop.Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			hop.Stats.Sent,
		)
	}

	if hop.Stats.Received != 1 {
		t.Fatalf(
			"Received = %d, want 1",
			hop.Stats.Received,
		)
	}

	if hop.Stats.Last != rtt {
		t.Fatalf(
			"Last = %v, want %v",
			hop.Stats.Last,
			rtt,
		)
	}
}

func TestHopUpdateTTLExpired(t *testing.T) {
	hop := NewHop(3)

	addr := net.ParseIP(
		"10.0.0.1",
	)

	rtt := 8 * time.Millisecond

	hop.Update(
		addr,
		rtt,
		probe.StatusTTLExpired,
	)

	if hop.Status != probe.StatusTTLExpired {
		t.Fatalf(
			"Status = %d, want %d",
			hop.Status,
			probe.StatusTTLExpired,
		)
	}

	if !hop.Addr.Equal(addr) {
		t.Fatalf(
			"Addr = %s, want %s",
			hop.Addr,
			addr,
		)
	}

	if hop.Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			hop.Stats.Sent,
		)
	}

	if hop.Stats.Received != 1 {
		t.Fatalf(
			"Received = %d, want 1",
			hop.Stats.Received,
		)
	}

	if hop.Stats.Last != rtt {
		t.Fatalf(
			"Last = %v, want %v",
			hop.Stats.Last,
			rtt,
		)
	}
}

func TestHopUpdateTimeout(t *testing.T) {
	hop := NewHop(4)

	hop.Update(
		nil,
		0,
		probe.StatusTimeout,
	)

	if hop.Status != probe.StatusTimeout {
		t.Fatalf(
			"Status = %d, want %d",
			hop.Status,
			probe.StatusTimeout,
		)
	}

	if hop.Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			hop.Stats.Sent,
		)
	}

	if hop.Stats.Received != 0 {
		t.Fatalf(
			"Received = %d, want 0",
			hop.Stats.Received,
		)
	}

	if hop.Stats.Loss() != 100 {
		t.Fatalf(
			"Loss = %.1f, want 100",
			hop.Stats.Loss(),
		)
	}
}

func TestHopUpdateUnreachable(t *testing.T) {
	hop := NewHop(5)

	addr := net.ParseIP(
		"192.168.100.1",
	)

	hop.Update(
		addr,
		0,
		probe.StatusUnreachable,
	)

	if hop.Status != probe.StatusUnreachable {
		t.Fatalf(
			"Status = %d, want %d",
			hop.Status,
			probe.StatusUnreachable,
		)
	}

	if !hop.Addr.Equal(addr) {
		t.Fatalf(
			"Addr = %s, want %s",
			hop.Addr,
			addr,
		)
	}

	if hop.Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			hop.Stats.Sent,
		)
	}

	if hop.Stats.Received != 0 {
		t.Fatalf(
			"Received = %d, want 0",
			hop.Stats.Received,
		)
	}
}

func TestHopUpdateUnknown(t *testing.T) {
	hop := NewHop(6)

	hop.Update(
		nil,
		0,
		probe.StatusUnknown,
	)

	if hop.Status != probe.StatusUnknown {
		t.Fatalf(
			"Status = %d, want %d",
			hop.Status,
			probe.StatusUnknown,
		)
	}

	if hop.Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			hop.Stats.Sent,
		)
	}

	if hop.Stats.Received != 0 {
		t.Fatalf(
			"Received = %d, want 0",
			hop.Stats.Received,
		)
	}
}

func TestHopSnapshotCopiesIP(t *testing.T) {
	hop := NewHop(1)

	addr := net.ParseIP(
		"10.0.0.1",
	)

	hop.Update(
		addr,
		3*time.Millisecond,
		probe.StatusSuccess,
	)

	snapshot := hop.Snapshot()

	if snapshot.Addr == nil {
		t.Fatal(
			"snapshot Addr is nil",
		)
	}

	original := append(
		net.IP(nil),
		snapshot.Addr...,
	)

	hop.Update(
		net.ParseIP("10.0.0.2"),
		4*time.Millisecond,
		probe.StatusSuccess,
	)

	if !snapshot.Addr.Equal(original) {
		t.Fatalf(
			"snapshot Addr changed from %s to %s",
			original,
			snapshot.Addr,
		)
	}

	if !snapshot.Addr.Equal(
		net.ParseIP("10.0.0.1"),
	) {
		t.Fatalf(
			"snapshot Addr = %s, want 10.0.0.1",
			snapshot.Addr,
		)
	}
}

func TestHopSnapshot(t *testing.T) {
	hop := NewHop(2)

	addr := net.ParseIP(
		"172.16.0.1",
	)

	hop.Update(
		addr,
		7*time.Millisecond,
		probe.StatusTTLExpired,
	)

	snapshot := hop.Snapshot()

	if snapshot.TTL != 2 {
		t.Fatalf(
			"TTL = %d, want 2",
			snapshot.TTL,
		)
	}

	if !snapshot.Addr.Equal(addr) {
		t.Fatalf(
			"Addr = %s, want %s",
			snapshot.Addr,
			addr,
		)
	}

	if snapshot.Status != probe.StatusTTLExpired {
		t.Fatalf(
			"Status = %d, want %d",
			snapshot.Status,
			probe.StatusTTLExpired,
		)
	}

	if snapshot.Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			snapshot.Stats.Sent,
		)
	}

	if snapshot.Stats.Received != 1 {
		t.Fatalf(
			"Received = %d, want 1",
			snapshot.Stats.Received,
		)
	}
}
