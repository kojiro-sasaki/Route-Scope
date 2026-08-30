package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kojiro-sasaki/Route-Scope.git/internal/probe"
)

func main() {
	ctx := context.Background()

	p, err := probe.NewICMPProber(2 * time.Second)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := p.Close(); err != nil {
			fmt.Printf("close ICMP handle: %v\n", err)
		}
	}()

	target := "8.8.8.8"

	fmt.Printf("Tracing route to %s\n\n", target)

	for ttl := 1; ttl <= 30; ttl++ {
		result, err := p.Probe(ctx, target, ttl)
		if err != nil {
			fmt.Printf(
				"%2d  ERROR  %v\n",
				ttl,
				err,
			)
			continue
		}

		if result.Timeout {
			fmt.Printf(
				"%2d  *\n",
				ttl,
			)

			continue
		}

		fmt.Printf(
			"%2d  %-15s  %6v",
			ttl,
			result.Addr,
			result.RTT,
		)

		if result.Reached {
			fmt.Print("  [target]")
		}

		fmt.Println()

		if result.Reached {
			break
		}
	}
}
