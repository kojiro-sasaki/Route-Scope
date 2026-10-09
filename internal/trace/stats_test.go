package trace

import (
	"testing"
	"time"
)

func TestStatsAdd(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)

	if stats.Sent != 1 {
		t.Fatalf("Sent = %d, want 1", stats.Sent)
	}

	if stats.Received != 1 {
		t.Fatalf("Received = %d, want 1", stats.Received)
	}

	if stats.Last != 10*time.Millisecond {
		t.Fatalf("Last = %v, want 10ms", stats.Last)
	}

	if stats.Min != 10*time.Millisecond {
		t.Fatalf("Min = %v, want 10ms", stats.Min)
	}

	if stats.Max != 10*time.Millisecond {
		t.Fatalf("Max = %v, want 10ms", stats.Max)
	}

	if stats.Average() != 10*time.Millisecond {
		t.Fatalf(
			"Average = %v, want 10ms",
			stats.Average(),
		)
	}

	if stats.Jitter != 0 {
		t.Fatalf(
			"Jitter = %v, want 0",
			stats.Jitter,
		)
	}
}

func TestStatsAddMultiple(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Add(30 * time.Millisecond)

	if stats.Sent != 3 {
		t.Fatalf("Sent = %d, want 3", stats.Sent)
	}

	if stats.Received != 3 {
		t.Fatalf("Received = %d, want 3", stats.Received)
	}

	if stats.Min != 10*time.Millisecond {
		t.Fatalf("Min = %v, want 10ms", stats.Min)
	}

	if stats.Max != 30*time.Millisecond {
		t.Fatalf("Max = %v, want 30ms", stats.Max)
	}

	if stats.Last != 30*time.Millisecond {
		t.Fatalf("Last = %v, want 30ms", stats.Last)
	}

	if stats.Average() != 20*time.Millisecond {
		t.Fatalf(
			"Average = %v, want 20ms",
			stats.Average(),
		)
	}

	if stats.Jitter != 10*time.Millisecond {
		t.Fatalf(
			"Jitter = %v, want 10ms",
			stats.Jitter,
		)
	}
}

func TestStatsJitter(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Add(15 * time.Millisecond)
	stats.Add(25 * time.Millisecond)

	want := 25 * time.Millisecond / 3

	if stats.Jitter != want {
		t.Fatalf(
			"Jitter = %v, want %v",
			stats.Jitter,
			want,
		)
	}
}

func TestStatsJitterIgnoresTimeouts(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Timeout()
	stats.Timeout()
	stats.Add(20 * time.Millisecond)

	if stats.Sent != 4 {
		t.Fatalf("Sent = %d, want 4", stats.Sent)
	}

	if stats.Received != 2 {
		t.Fatalf("Received = %d, want 2", stats.Received)
	}

	if stats.Jitter != 10*time.Millisecond {
		t.Fatalf(
			"Jitter = %v, want 10ms",
			stats.Jitter,
		)
	}
}

func TestStatsJitterWithDecreasingRTT(t *testing.T) {
	var stats Stats

	stats.Add(30 * time.Millisecond)
	stats.Add(10 * time.Millisecond)

	if stats.Jitter != 20*time.Millisecond {
		t.Fatalf(
			"Jitter = %v, want 20ms",
			stats.Jitter,
		)
	}
}

func TestStatsTimeout(t *testing.T) {
	var stats Stats

	stats.Timeout()

	if stats.Sent != 1 {
		t.Fatalf("Sent = %d, want 1", stats.Sent)
	}

	if stats.Received != 0 {
		t.Fatalf("Received = %d, want 0", stats.Received)
	}

	if stats.Loss() != 100 {
		t.Fatalf(
			"Loss = %.1f, want 100",
			stats.Loss(),
		)
	}

	if stats.Jitter != 0 {
		t.Fatalf(
			"Jitter = %v, want 0",
			stats.Jitter,
		)
	}
}

func TestStatsAverage(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Add(30 * time.Millisecond)

	if got := stats.Average(); got != 20*time.Millisecond {
		t.Fatalf(
			"Average = %v, want 20ms",
			got,
		)
	}
}

func TestStatsAverageEmpty(t *testing.T) {
	var stats Stats

	if got := stats.Average(); got != 0 {
		t.Fatalf(
			"Average = %v, want 0",
			got,
		)
	}
}

func TestStatsVariance(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Add(30 * time.Millisecond)

	want := float64(100 * time.Millisecond * time.Millisecond)

	if got := stats.Variance(); got != want {
		t.Fatalf(
			"Variance = %v, want %v",
			got,
			want,
		)
	}
}

func TestStatsVarianceInsufficientSamples(t *testing.T) {
	var stats Stats

	if got := stats.Variance(); got != 0 {
		t.Fatalf(
			"Variance = %v, want 0",
			got,
		)
	}

	stats.Add(10 * time.Millisecond)

	if got := stats.Variance(); got != 0 {
		t.Fatalf(
			"Variance = %v, want 0",
			got,
		)
	}
}

func TestStatsStdDev(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Add(30 * time.Millisecond)

	want := 10 * time.Millisecond

	if got := stats.StdDev(); got != want {
		t.Fatalf(
			"StdDev = %v, want 10ms",
			got,
		)
	}
}

func TestStatsLoss(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Timeout()
	stats.Timeout()

	if got := stats.Loss(); got != 50 {
		t.Fatalf(
			"Loss = %.1f, want 50",
			got,
		)
	}
}

func TestStatsLossEmpty(t *testing.T) {
	var stats Stats

	if got := stats.Loss(); got != 0 {
		t.Fatalf(
			"Loss = %.1f, want 0",
			got,
		)
	}
}
