package probe

import (
	"context"
	"fmt"
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
	TTL    int
	Addr   net.IP
	RTT    time.Duration
	Status Status
}

type Prober interface {
	Probe(ctx context.Context, target net.IP, ttl int) (Result, error)
}

func ResolveIPv4(ctx context.Context, target string) (net.IP, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if target == "" {
		return nil, fmt.Errorf("target must not be empty")
	}

	if ip := net.ParseIP(target); ip != nil {
		ipv4 := ip.To4()
		if ipv4 == nil {
			return nil, fmt.Errorf("target %q is not IPv4", target)
		}
		return append(net.IP(nil), ipv4...), nil
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", target)
	if err != nil {
		return nil, fmt.Errorf("resolve target %q: %w", target, err)
	}

	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return append(net.IP(nil), ipv4...), nil
		}
	}

	return nil, fmt.Errorf("no IPv4 address found for %q", target)
}
