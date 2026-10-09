package trace

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

type failingProber struct {
	err error
}

func (p failingProber) Probe(
	ctx context.Context,
	target net.IP,
	ttl int,
) (probe.Result, error) {
	return probe.Result{TTL: ttl}, p.err
}

func TestEngineRecordsProbeErrorAndContinues(t *testing.T) {
	probeErr := errors.New("temporary probe failure")

	engine := NewEngine(
		failingProber{err: probeErr},
		1,
		time.Second,
	)

	err := engine.probeRound(
		context.Background(),
		net.ParseIP("8.8.8.8"),
	)
	if err != nil {
		t.Fatalf("probeRound() error = %v, want nil", err)
	}

	hops := engine.Hops()
	if len(hops) != 1 {
		t.Fatalf("len(Hops()) = %d, want 1", len(hops))
	}

	if hops[0].Status != probe.StatusTimeout {
		t.Fatalf(
			"hop status = %v, want %v",
			hops[0].Status,
			probe.StatusTimeout,
		)
	}

	if hops[0].Stats.Sent != 1 {
		t.Fatalf("sent = %d, want 1", hops[0].Stats.Sent)
	}

	if hops[0].Stats.Received != 0 {
		t.Fatalf("received = %d, want 0", hops[0].Stats.Received)
	}

	probeErrors := engine.Errors()
	if len(probeErrors) != 1 {
		t.Fatalf("len(Errors()) = %d, want 1", len(probeErrors))
	}

	if probeErrors[0].TTL != 1 {
		t.Fatalf("error TTL = %d, want 1", probeErrors[0].TTL)
	}

	if !errors.Is(probeErrors[0].Err, probeErr) {
		t.Fatalf("stored error = %v, want %v", probeErrors[0].Err, probeErr)
	}
}
