package trace

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

type ProbeError struct {
	TTL int
	Err error
}

type Engine struct {
	Prober   probe.Prober
	MaxTTL   int
	Interval time.Duration

	mu          sync.RWMutex
	hops        map[int]*Hop
	probeErrors map[int]error
}

func NewEngine(
	prober probe.Prober,
	maxTTL int,
	interval time.Duration,
) *Engine {
	return &Engine{
		Prober:      prober,
		MaxTTL:      maxTTL,
		Interval:    interval,
		hops:        make(map[int]*Hop),
		probeErrors: make(map[int]error),
	}
}

func (e *Engine) Run(ctx context.Context, target net.IP) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if target.To4() == nil {
		return fmt.Errorf("target must be an IPv4 address")
	}

	if e.Prober == nil {
		return fmt.Errorf("prober is nil")
	}

	if e.MaxTTL < 1 || e.MaxTTL > 255 {
		return fmt.Errorf("invalid max TTL: %d", e.MaxTTL)
	}

	if e.Interval <= 0 {
		return fmt.Errorf("interval must be greater than zero")
	}

	if err := e.probeRound(ctx, target); err != nil {
		return err
	}

	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := e.probeRound(ctx, target); err != nil {
				return err
			}
		}
	}
}

func (e *Engine) probeRound(ctx context.Context, target net.IP) error {
	type probeResult struct {
		ttl    int
		result probe.Result
		err    error
	}

	results := make(chan probeResult, e.MaxTTL)
	var wg sync.WaitGroup

	for ttl := 1; ttl <= e.MaxTTL; ttl++ {
		if err := ctx.Err(); err != nil {
			break
		}

		wg.Add(1)

		go func(ttl int) {
			defer wg.Done()

			result, err := e.Prober.Probe(ctx, target, ttl)
			results <- probeResult{
				ttl:    ttl,
				result: result,
				err:    err,
			}
		}(ttl)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	roundResults := make([]probe.Result, 0, e.MaxTTL)

	for item := range results {
		if err := ctx.Err(); err != nil {
			return err
		}

		item.result.TTL = item.ttl

		if item.err != nil {
			if errors.Is(item.err, context.Canceled) ||
				errors.Is(item.err, context.DeadlineExceeded) {
				if err := ctx.Err(); err != nil {
					return err
				}
			}

			e.recordProbeError(item.ttl, item.err)
			item.result.Addr = nil
			item.result.RTT = 0
			item.result.Status = probe.StatusTimeout
			roundResults = append(roundResults, item.result)
			continue
		}

		e.clearProbeError(item.ttl)
		roundResults = append(roundResults, item.result)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	sort.Slice(roundResults, func(i, j int) bool {
		return roundResults[i].TTL < roundResults[j].TTL
	})

	lastTTL := e.MaxTTL

	for _, result := range roundResults {
		if result.Status == probe.StatusSuccess {
			lastTTL = result.TTL
			break
		}
	}

	for _, result := range roundResults {
		if result.TTL > lastTTL {
			continue
		}

		hop := e.getHop(result.TTL)
		hop.Update(result.Addr, result.RTT, result.Status)
	}

	return nil
}

func (e *Engine) recordProbeError(ttl int, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.probeErrors[ttl] = err
}

func (e *Engine) clearProbeError(ttl int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.probeErrors, ttl)
}

func (e *Engine) Errors() []ProbeError {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]ProbeError, 0, len(e.probeErrors))

	for ttl, err := range e.probeErrors {
		result = append(result, ProbeError{
			TTL: ttl,
			Err: err,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TTL < result[j].TTL
	})

	return result
}

func (e *Engine) getHop(ttl int) *Hop {
	e.mu.Lock()
	defer e.mu.Unlock()

	hop, exists := e.hops[ttl]
	if !exists {
		hop = NewHop(ttl)
		e.hops[ttl] = hop
	}

	return hop
}

func (e *Engine) Hops() []HopSnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]HopSnapshot, 0, len(e.hops))

	for _, hop := range e.hops {
		result = append(result, hop.Snapshot())
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TTL < result[j].TTL
	})

	return result
}
