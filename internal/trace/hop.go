package trace

import (
	"net"
	"sync"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

type Hop struct {
	TTL int

	mu sync.RWMutex

	Addr   net.IP
	Status probe.Status
	Stats  Stats
}

func NewHop(ttl int) *Hop {
	return &Hop{
		TTL: ttl,
	}
}

func (h *Hop) Update(
	addr net.IP,
	rtt time.Duration,
	status probe.Status,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Status = status

	if addr != nil {
		h.Addr = append(
			net.IP(nil),
			addr...,
		)
	}

	switch status {
	case probe.StatusSuccess,
		probe.StatusTTLExpired:
		h.Stats.Add(rtt)

	default:
		h.Stats.Timeout()
	}
}

func (h *Hop) Snapshot() HopSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var addr net.IP

	if h.Addr != nil {
		addr = append(
			net.IP(nil),
			h.Addr...,
		)
	}

	return HopSnapshot{
		TTL:    h.TTL,
		Addr:   addr,
		Status: h.Status,
		Stats:  h.Stats,
	}
}

type HopSnapshot struct {
	TTL    int
	Addr   net.IP
	Status probe.Status
	Stats  Stats
}
