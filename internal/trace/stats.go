package trace

import (
	"math"
	"time"
)

type Stats struct {
	Sent     uint64
	Received uint64

	Last time.Duration
	Min  time.Duration
	Max  time.Duration

	Mean   float64
	Jitter time.Duration

	m2 float64

	previousRTT time.Duration
	jitterTotal time.Duration
	jitterCount uint64
}

func (s *Stats) Add(rtt time.Duration) {
	s.Sent++
	s.Received++

	s.Last = rtt

	if s.Received == 1 {
		s.Min = rtt
		s.Max = rtt
		s.Mean = float64(rtt)
		s.m2 = 0
		s.previousRTT = rtt
		return
	}

	if rtt < s.Min {
		s.Min = rtt
	}

	if rtt > s.Max {
		s.Max = rtt
	}

	delta := float64(rtt) - s.Mean
	s.Mean += delta / float64(s.Received)

	delta2 := float64(rtt) - s.Mean
	s.m2 += delta * delta2

	jitter := rtt - s.previousRTT

	if jitter < 0 {
		jitter = -jitter
	}

	s.jitterTotal += jitter
	s.jitterCount++
	s.Jitter = time.Duration(
		int64(s.jitterTotal) / int64(s.jitterCount),
	)

	s.previousRTT = rtt
}

func (s *Stats) Timeout() {
	s.Sent++
}

func (s Stats) Average() time.Duration {
	if s.Received == 0 {
		return 0
	}

	return time.Duration(s.Mean)
}

func (s Stats) Variance() float64 {
	if s.Received < 2 {
		return 0
	}

	return s.m2 / float64(s.Received-1)
}

func (s Stats) StdDev() time.Duration {
	return time.Duration(
		math.Sqrt(s.Variance()),
	)
}

func (s Stats) Loss() float64 {
	if s.Sent == 0 {
		return 0
	}

	return float64(
		s.Sent-s.Received,
	) * 100 / float64(s.Sent)
}
