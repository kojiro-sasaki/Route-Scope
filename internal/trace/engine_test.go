package trace

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

type fakeProber struct {
	mu sync.Mutex

	results map[int]probe.Result
	calls   []int
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
	select {
	case <-ctx.Done():
		return probe.Result{
			TTL: ttl,
		}, ctx.Err()

	default:
	}

	p.mu.Lock()
	p.calls = append(
		p.calls,
		ttl,
	)
	p.mu.Unlock()

	result, exists := p.results[ttl]

	if !exists {
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

	result := make(
		[]int,
		len(p.calls),
	)

	copy(
		result,
		p.calls,
	)

	return result
}

func TestEngineProbeRound(t *testing.T) {
	target := net.ParseIP(
		"8.8.8.8",
	)

	prober := newFakeProber(
		map[int]probe.Result{
			1: {
				TTL: 1,
				Addr: net.ParseIP(
					"192.168.1.1",
				),
				RTT: 2 * time.Millisecond,
			},
			2: {
				TTL: 2,
				Addr: net.ParseIP(
					"10.0.0.1",
				),
				RTT: 8 * time.Millisecond,
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

	ctx := context.Background()

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
			"len(hops) = %d, want 3",
			len(hops),
		)
	}

	if hops[0].TTL != 1 {
		t.Fatalf(
			"hops[0].TTL = %d, want 1",
			hops[0].TTL,
		)
	}

	if hops[1].TTL != 2 {
		t.Fatalf(
			"hops[1].TTL = %d, want 2",
			hops[1].TTL,
		)
	}

	if hops[2].TTL != 3 {
		t.Fatalf(
			"hops[2].TTL = %d, want 3",
			hops[2].TTL,
		)
	}

	if hops[0].Stats.Sent != 1 {
		t.Fatalf(
			"hop 1 sent = %d, want 1",
			hops[0].Stats.Sent,
		)
	}

	if hops[1].Stats.Sent != 1 {
		t.Fatalf(
			"hop 2 sent = %d, want 1",
			hops[1].Stats.Sent,
		)
	}

	if hops[2].Stats.Sent != 1 {
		t.Fatalf(
			"hop 3 sent = %d, want 1",
			hops[2].Stats.Sent,
		)
	}

	if hops[2].Stats.Received != 1 {
		t.Fatalf(
			"hop 3 received = %d, want 1",
			hops[2].Stats.Received,
		)
	}

	if hops[2].Stats.Last != 15*time.Millisecond {
		t.Fatalf(
			"hop 3 last = %v, want 15ms",
			hops[2].Stats.Last,
		)
	}
}

func TestEngineStopsAfterTarget(
	t *testing.T,
) {
	prober := newFakeProber(
		map[int]probe.Result{
			1: {
				TTL: 1,
				Addr: net.ParseIP(
					"192.168.1.1",
				),
				RTT: 2 * time.Millisecond,
			},
			2: {
				TTL: 2,
				Addr: net.ParseIP(
					"8.8.8.8",
				),
				RTT:     10 * time.Millisecond,
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

	hops := engine.Hops()

	if len(hops) != 2 {
		t.Fatalf(
			"len(hops) = %d, want 2",
			len(hops),
		)
	}

	if hops[0].TTL != 1 {
		t.Fatalf(
			"hops[0].TTL = %d, want 1",
			hops[0].TTL,
		)
	}

	if hops[1].TTL != 2 {
		t.Fatalf(
			"hops[1].TTL = %d, want 2",
			hops[1].TTL,
		)
	}

	if hops[1].Addr == nil {
		t.Fatal(
			"target address is nil",
		)
	}

	if !hops[1].Addr.Equal(
		net.ParseIP("8.8.8.8"),
	) {
		t.Fatalf(
			"target address = %s, want 8.8.8.8",
			hops[1].Addr,
		)
	}

	calls := prober.Calls()

	if len(calls) != 30 {
		t.Fatalf(
			"probe calls = %d, want 30",
			len(calls),
		)
	}
}

func TestEngineContextCancellation(
	t *testing.T,
) {
	prober := newFakeProber(
		map[int]probe.Result{},
	)

	engine := NewEngine(
		prober,
		30,
		time.Second,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	cancel()

	err := engine.Run(
		ctx,
		"8.8.8.8",
	)

	if err == nil {
		t.Fatal(
			"Run() error = nil, want context cancellation",
		)
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Run() error = %v, want context.Canceled",
			err,
		)
	}
}

func TestEngineValidation(
	t *testing.T,
) {
	t.Run(
		"nil prober",
		func(t *testing.T) {
			engine := NewEngine(
				nil,
				30,
				time.Second,
			)

			err := engine.Run(
				context.Background(),
				"8.8.8.8",
			)

			if err == nil {
				t.Fatal(
					"Run() error = nil, want error",
				)
			}
		},
	)

	t.Run(
		"invalid max ttl",
		func(t *testing.T) {
			prober := newFakeProber(nil)

			engine := NewEngine(
				prober,
				0,
				time.Second,
			)

			err := engine.Run(
				context.Background(),
				"8.8.8.8",
			)

			if err == nil {
				t.Fatal(
					"Run() error = nil, want error",
				)
			}
		},
	)

	t.Run(
		"invalid interval",
		func(t *testing.T) {
			prober := newFakeProber(nil)

			engine := NewEngine(
				prober,
				30,
				0,
			)

			err := engine.Run(
				context.Background(),
				"8.8.8.8",
			)

			if err == nil {
				t.Fatal(
					"Run() error = nil, want error",
				)
			}
		},
	)
}
