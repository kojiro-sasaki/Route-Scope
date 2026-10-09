package trace

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

type Engine struct {
	Prober probe.Prober

	MaxTTL   int
	Interval time.Duration

	mu   sync.RWMutex
	hops map[int]*Hop
}

func NewEngine(
	prober probe.Prober,
	maxTTL int,
	interval time.Duration,
) *Engine {
	return &Engine{
		Prober:   prober,
		MaxTTL:   maxTTL,
		Interval: interval,
		hops:     make(map[int]*Hop),
	}
}

func (e *Engine) Run(
	ctx context.Context,
	target string,
) error {
	if e.Prober == nil {
		return fmt.Errorf(
			"prober is nil",
		)
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

	ticker := time.NewTicker(
		e.Interval,
	)
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
	type probeResult struct {
		result probe.Result
		err    error
	}

	results := make(
		chan probeResult,
		e.MaxTTL,
	)

	var wg sync.WaitGroup

	wg.Add(e.MaxTTL)

	for ttl := 1; ttl <= e.MaxTTL; ttl++ {
		ttl := ttl

		go func() {
			defer wg.Done()

			result, err := e.Prober.Probe(
				ctx,
				target,
				ttl,
			)

			results <- probeResult{
				result: result,
				err:    err,
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var firstError error

	roundResults := make(
		[]probe.Result,
		0,
		e.MaxTTL,
	)

	for item := range results {
		if item.err != nil {
			if firstError == nil {
				firstError = fmt.Errorf(
					"probe ttl=%d: %w",
					item.result.TTL,
					item.err,
				)
			}

			continue
		}

		roundResults = append(
			roundResults,
			item.result,
		)
	}

	if firstError != nil {
		return firstError
	}

	sort.Slice(
		roundResults,
		func(i, j int) bool {
			return roundResults[i].TTL <
				roundResults[j].TTL
		},
	)

	lastTTL := e.MaxTTL

	for _, result := range roundResults {
		if result.Reached ||
			result.Status == probe.StatusSuccess {
			lastTTL = result.TTL
			break
		}
	}

	for _, result := range roundResults {
		if result.TTL > lastTTL {
			continue
		}

		hop := e.getHop(
			result.TTL,
		)

		hop.Update(
			result.Addr,
			result.RTT,
			result.Status,
		)
	}

	return nil
}

func (e *Engine) getHop(ttl int) *Hop {
	e.mu.Lock()
	defer e.mu.Unlock()

	hop, exists := e.hops[ttl]

	if !exists {
		hop = NewHop(
			ttl,
		)

		e.hops[ttl] = hop
	}

	return hop
}

func (e *Engine) Hops() []HopSnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

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

	sort.Slice(
		result,
		func(i, j int) bool {
			return result[i].TTL <
				result[j].TTL
		},
	)

	return result
}
