package trace

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

type fakeProber struct {
	mu sync.Mutex

	results map[int]probe.Result

	calls []int
}

func newFakeProber(
	results map[int]probe.Result,
) *fakeProber {
	return &fakeProber{
		results: results,
	}
}

func (p *fakeProber) Probe(
	ctx context.Context,
	target string,
	ttl int,
) (probe.Result, error) {
	p.mu.Lock()
	p.calls = append(p.calls, ttl)
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		return probe.Result{}, ctx.Err()

	default:
	}

	result, ok := p.results[ttl]

	if !ok {
		return probe.Result{
			TTL:     ttl,
			Timeout: true,
		}, nil
	}

	return result, nil
}

func (p *fakeProber) Calls() []int {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]int, len(p.calls))

	copy(result, p.calls)

	return result
}

func TestEngineProbeRound(t *testing.T) {
	target := net.ParseIP("8.8.8.8")

	prober := newFakeProber(
		map[int]probe.Result{
			1: {
				TTL:  1,
				Addr: net.ParseIP("192.168.1.1"),
				RTT:  2 * time.Millisecond,
			},

			2: {
				TTL:  2,
				Addr: net.ParseIP("10.0.0.1"),
				RTT:  8 * time.Millisecond,
			},

			3: {
				TTL:     3,
				Addr:    target,
				RTT:     15 * time.Millisecond,
				Reached: true,
			},
		},
	)

	engine := NewEngine(
		prober,
		30,
		time.Second,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	err := engine.probeRound(
		ctx,
		"8.8.8.8",
	)

	if err != nil {
		t.Fatalf(
			"probeRound() error = %v",
			err,
		)
	}

	hops := engine.Hops()

	if len(hops) != 3 {
		t.Fatalf(
			"got %d hops, want 3",
			len(hops),
		)
	}

	if !hops[0].Addr.Equal(
		net.ParseIP("192.168.1.1"),
	) {
		t.Fatalf(
			"hop 1 = %v",
			hops[0].Addr,
		)
	}

	if !hops[1].Addr.Equal(
		net.ParseIP("10.0.0.1"),
	) {
		t.Fatalf(
			"hop 2 = %v",
			hops[1].Addr,
		)
	}

	if !hops[2].Addr.Equal(target) {
		t.Fatalf(
			"hop 3 = %v",
			hops[2].Addr,
		)
	}
}

func TestEngineStopsAfterTarget(t *testing.T) {
	target := net.ParseIP("8.8.8.8")

	prober := newFakeProber(
		map[int]probe.Result{
			1: {
				TTL:  1,
				Addr: net.ParseIP("192.168.1.1"),
				RTT:  2 * time.Millisecond,
			},

			2: {
				TTL:     2,
				Addr:    target,
				RTT:     10 * time.Millisecond,
				Reached: true,
			},

			3: {
				TTL:     3,
				Addr:    net.ParseIP("1.2.3.4"),
				RTT:     20 * time.Millisecond,
				Reached: true,
			},
		},
	)

	engine := NewEngine(
		prober,
		30,
		time.Second,
	)

	err := engine.probeRound(
		context.Background(),
		"8.8.8.8",
	)

	if err != nil {
		t.Fatalf(
			"probeRound() error = %v",
			err,
		)
	}

	calls := prober.Calls()

	if len(calls) != 30 {
		t.Logf(
			"parallel engine probes all TTLs: got %d calls",
			len(calls),
		)
	}

	hops := engine.Hops()

	for _, hop := range hops {
		if hop.TTL > 2 {
			t.Fatalf(
				"hop %d exists after target at TTL 2",
				hop.TTL,
			)
		}
	}
}

func TestEngineTimeout(t *testing.T) {
	prober := newFakeProber(
		map[int]probe.Result{
			1: {
				TTL:     1,
				Timeout: true,
			},
		},
	)

	engine := NewEngine(
		prober,
		1,
		time.Second,
	)

	err := engine.probeRound(
		context.Background(),
		"8.8.8.8",
	)

	if err != nil {
		t.Fatalf(
			"probeRound() error = %v",
			err,
		)
	}

	hops := engine.Hops()

	if len(hops) != 1 {
		t.Fatalf(
			"got %d hops, want 1",
			len(hops),
		)
	}

	if hops[0].Stats.Sent != 1 {
		t.Fatalf(
			"Sent = %d, want 1",
			hops[0].Stats.Sent,
		)
	}

	if hops[0].Stats.Received != 0 {
		t.Fatalf(
			"Received = %d, want 0",
			hops[0].Stats.Received,
		)
	}

	if hops[0].Stats.Loss() != 100 {
		t.Fatalf(
			"Loss = %.2f, want 100",
			hops[0].Stats.Loss(),
		)
	}
}
