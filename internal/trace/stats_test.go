package trace

import (
	"math"
	"testing"
	"time"
)

func TestStatsAdd(t *testing.T) {
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

	if stats.Last != 30*time.Millisecond {
		t.Fatalf(
			"Last = %v, want 30ms",
			stats.Last,
		)
	}

	if stats.Min != 10*time.Millisecond {
		t.Fatalf(
			"Min = %v, want 10ms",
			stats.Min,
		)
	}

	if stats.Max != 30*time.Millisecond {
		t.Fatalf(
			"Max = %v, want 30ms",
			stats.Max,
		)
	}

	wantAverage := 20 * time.Millisecond

	if stats.Average() != wantAverage {
		t.Fatalf(
			"Average = %v, want %v",
			stats.Average(),
			wantAverage,
		)
	}
}

func TestStatsLoss(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Timeout()
	stats.Timeout()

	got := stats.Loss()
	want := 50.0

	if got != want {
		t.Fatalf(
			"Loss = %.2f, want %.2f",
			got,
			want,
		)
	}
}

func TestStatsNoPackets(t *testing.T) {
	var stats Stats

	if stats.Loss() != 0 {
		t.Fatalf(
			"Loss = %.2f, want 0",
			stats.Loss(),
		)
	}

	if stats.Average() != 0 {
		t.Fatalf(
			"Average = %v, want 0",
			stats.Average(),
		)
	}

	if stats.StdDev() != 0 {
		t.Fatalf(
			"StdDev = %v, want 0",
			stats.StdDev(),
		)
	}
}

func TestStatsVariance(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Add(30 * time.Millisecond)

	// Variance для [10, 20, 30] = 100 ms².
	want := 100.0 *
		float64(time.Millisecond) *
		float64(time.Millisecond)

	got := stats.Variance()

	if math.Abs(got-want) > 1 {
		t.Fatalf(
			"Variance = %f, want approximately %f",
			got,
			want,
		)
	}
}

func TestStatsStdDev(t *testing.T) {
	var stats Stats

	stats.Add(10 * time.Millisecond)
	stats.Add(20 * time.Millisecond)
	stats.Add(30 * time.Millisecond)

	// StdDev = 10ms.
	want := 10 * time.Millisecond

	got := stats.StdDev()

	if got != want {
		t.Fatalf(
			"StdDev = %v, want %v",
			got,
			want,
		)
	}
}

func TestStatsSinglePacket(t *testing.T) {
	var stats Stats

	stats.Add(25 * time.Millisecond)

	if stats.Min != 25*time.Millisecond {
		t.Fatalf(
			"Min = %v, want 25ms",
			stats.Min,
		)
	}

	if stats.Max != 25*time.Millisecond {
		t.Fatalf(
			"Max = %v, want 25ms",
			stats.Max,
		)
	}

	if stats.Average() != 25*time.Millisecond {
		t.Fatalf(
			"Average = %v, want 25ms",
			stats.Average(),
		)
	}

	if stats.StdDev() != 0 {
		t.Fatalf(
			"StdDev = %v, want 0",
			stats.StdDev(),
		)
	}
}
