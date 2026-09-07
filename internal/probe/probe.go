package probe

import (
	"context"
	"net"
	"time"
)

type Status uint8

const (
	StatusUnknown Status = iota
	StatusSuccess
	StatusTTLExpired
	StatusTimeout
	StatusUnreachable
)

type Result struct {
	TTL int

	Addr    net.IP
	RTT     time.Duration
	Reached bool
	Timeout bool
	Status  Status
}

type Prober interface {
	Probe(ctx context.Context, target string, ttl int) (Result, error)
}