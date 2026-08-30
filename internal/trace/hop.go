package trace

import (
	"net"
	"sync"
	"time"
)

type Hop struct {
	TTL int

	mu sync.RWMutex

	Addr net.IP

	Stats Stats
}

func NewHop(ttl int) *Hop {
	return &Hop{
		TTL: ttl,
	}
}

func (h *Hop) Update(
	addr net.IP,
	rtt time.Duration,
	received bool,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if addr != nil {
		h.Addr = append(
			net.IP(nil),
			addr...,
		)
	}

	if received {
		h.Stats.Add(rtt)
	} else {
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
		TTL:   h.TTL,
		Addr:  addr,
		Stats: h.Stats,
	}
}

type HopSnapshot struct {
	TTL   int
	Addr  net.IP
	Stats Stats
}