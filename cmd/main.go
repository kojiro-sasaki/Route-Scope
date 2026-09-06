package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
	"github.com/kojiro-sasaki/Route-Scope.git/internal/trace"
)

func main() {
	target := "8.8.8.8"

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	prober, err := probe.NewICMPProber(
		2 * time.Second,
	)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := prober.Close(); err != nil {
			fmt.Printf(
				"close ICMP handle: %v\n",
				err,
			)
		}
	}()

	engine := trace.NewEngine(
		prober,
		30,
		1*time.Second,
	)

	go func() {
		err := engine.Run(
			ctx,
			target,
		)

		if err != nil &&
			ctx.Err() == nil {

			fmt.Printf(
				"engine error: %v\n",
				err,
			)

			cancel()
		}
	}()

	ticker := time.NewTicker(
		1 * time.Second,
	)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nStopping...")
			return

		case <-ticker.C:
			printHops(
				engine,
				target,
			)
		}
	}
}

func printHops(
	engine *trace.Engine,
	target string,
) {
	hops := engine.Hops()

	sort.Slice(
		hops,
		func(i, j int) bool {
			return hops[i].TTL < hops[j].TTL
		},
	)

	// Очистка терминала.
	fmt.Print("\033[H\033[2J")

	fmt.Printf(
		"Go Tool MTR → %s\n\n",
		target,
	)

	fmt.Printf(
		"%-4s %-18s %-8s %-6s %-9s %-9s %-9s %-9s\n",
		"Hop",
		"Host",
		"Loss",
		"Sent",
		"Last",
		"Avg",
		"Best",
		"Worst",
	)

	for _, hop := range hops {
		stats := hop.Stats

		host := "*"

		if hop.Addr != nil {
			host = hop.Addr.String()
		}

		last := "-"
		avg := "-"
		best := "-"
		worst := "-"

		if stats.Received > 0 {
			last = stats.Last.String()
			avg = stats.Average().String()
			best = stats.Min.String()
			worst = stats.Max.String()
		}

		fmt.Printf(
			"%-4d %-18s %6.1f%% %6d %-9s %-9s %-9s %-9s\n",
			hop.TTL,
			host,
			stats.Loss(),
			stats.Sent,
			last,
			avg,
			best,
			worst,
		)
	}
}
