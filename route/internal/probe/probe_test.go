package probe

import (
	"context"
	"net"
	"testing"
)

func TestResolveIPv4Literal(t *testing.T) {
	got, err := ResolveIPv4(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("ResolveIPv4() error = %v", err)
	}

	want := net.ParseIP("8.8.8.8").To4()
	if !got.Equal(want) {
		t.Fatalf("ResolveIPv4() = %v, want %v", got, want)
	}
}

func TestResolveIPv4RejectsIPv6(t *testing.T) {
	if _, err := ResolveIPv4(context.Background(), "2001:4860:4860::8888"); err == nil {
		t.Fatal("ResolveIPv4() error = nil, want IPv6 rejection")
	}
}

func TestResolveIPv4RejectsEmptyTarget(t *testing.T) {
	if _, err := ResolveIPv4(context.Background(), ""); err == nil {
		t.Fatal("ResolveIPv4() error = nil, want empty target error")
	}
}

func TestResolveIPv4HonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := ResolveIPv4(ctx, "example.com"); err != context.Canceled {
		t.Fatalf("ResolveIPv4() error = %v, want context.Canceled", err)
	}
}
