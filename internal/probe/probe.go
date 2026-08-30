package probe

import (
	"context"
	"net"
	"time"
)

type Result struct {
	Addr    net.IP
	RTT     time.Duration
	Reached bool
	Timeout bool
}

type Prober interface {
	Probe(ctx context.Context, target string, ttl int) (Result, error)
}
