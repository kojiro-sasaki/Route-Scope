package main

import (
	"context"
	"flag"
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
	if len(os.Args) < 2 {
		fmt.Fprintf(
			os.Stderr,
			"usage: %s <target> [options]\n",
			os.Args[0],
		)
		os.Exit(2)
	}

	target := os.Args[1]

	fs := flag.NewFlagSet(
		os.Args[0],
		flag.ExitOnError,
	)

	interval := fs.Duration(
		"interval",
		time.Second,
		"",
	)

	timeout := fs.Duration(
		"timeout",
		2*time.Second,
		"",
	)

	maxTTL := fs.Int(
		"max-ttl",
		30,
		"",
	)

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"parse options: %v\n",
			err,
		)
		os.Exit(2)
	}

	if *interval <= 0 {
		fmt.Fprintln(
			os.Stderr,
			"interval must be greater than zero",
		)
		os.Exit(2)
	}

	if *timeout <= 0 {
		fmt.Fprintln(
			os.Stderr,
			"timeout must be greater than zero",
		)
		os.Exit(2)
	}

	if *maxTTL < 1 || *maxTTL > 255 {
		fmt.Fprintln(
			os.Stderr,
			"max-ttl must be between 1 and 255",
		)
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	prober, err := probe.NewICMPProber(
		*timeout,
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"create ICMP prober: %v\n",
			err,
		)
		os.Exit(1)
	}

	defer func() {
		if err := prober.Close(); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"close ICMP prober: %v\n",
				err,
			)
		}
	}()

	engine := trace.NewEngine(
		prober,
		*maxTTL,
		*interval,
	)

	go func() {
		if err := engine.Run(
			ctx,
			target,
		); err != nil &&
			ctx.Err() == nil {
			fmt.Fprintf(
				os.Stderr,
				"trace error: %v\n",
				err,
			)
			cancel()
		}
	}()

	ticker := time.NewTicker(
		*interval,
	)
	defer ticker.Stop()

	printHops(
		engine,
		target,
	)

	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			fmt.Println("Stopping...")
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

	fmt.Print(
		"\033[H\033[2J",
	)

	fmt.Printf(
		"Route-Scope → %s\n\n",
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
