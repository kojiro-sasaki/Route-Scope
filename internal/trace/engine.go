package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

type Engine struct {
	Prober probe.Prober

	MaxTTL   int
	Interval time.Duration

	hops map[int]*Hop
}

func NewEngine(
	prober probe.Prober,
	maxTTL int,
	interval time.Duration,
) *Engine {
	return &Engine{
		Prober:  prober,
		MaxTTL:  maxTTL,
		Interval: interval,

		hops: make(map[int]*Hop),
	}
}

func (e *Engine) Run(
	ctx context.Context,
	target string,
) error {
	if e.Prober == nil {
		return fmt.Errorf("prober is nil")
	}

	if e.MaxTTL < 1 || e.MaxTTL > 255 {
		return fmt.Errorf(
			"invalid max TTL: %d",
			e.MaxTTL,
		)
	}

	if e.Interval <= 0 {
		return fmt.Errorf(
			"interval must be greater than zero",
		)
	}

	if err := e.probeRound(
		ctx,
		target,
	); err != nil {
		return err
	}

	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := e.probeRound(
				ctx,
				target,
			); err != nil {
				return err
			}
		}
	}
}

func (e *Engine) probeRound(
	ctx context.Context,
	target string,
) error {
	for ttl := 1; ttl <= e.MaxTTL; ttl++ {

		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
		}

		result, err := e.Prober.Probe(
			ctx,
			target,
			ttl,
		)

		if err != nil {
			return fmt.Errorf(
				"probe ttl=%d: %w",
				ttl,
				err,
			)
		}

		hop := e.getHop(ttl)

		if result.Timeout {
			hop.Update(
				nil,
				0,
				false,
			)
		} else {
			hop.Update(
				result.Addr,
				result.RTT,
				true,
			)
		}

		if result.Reached {
			break
		}
	}

	return nil
}

func (e *Engine) getHop(ttl int) *Hop {
	hop, exists := e.hops[ttl]

	if !exists {
		hop = NewHop(ttl)
		e.hops[ttl] = hop
	}

	return hop
}

func (e *Engine) Hops() []HopSnapshot {
	result := make(
		[]HopSnapshot,
		0,
		len(e.hops),
	)

	for _, hop := range e.hops {
		result = append(
			result,
			hop.Snapshot(),
		)
	}

	return result
}