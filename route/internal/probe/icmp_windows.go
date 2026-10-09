//go:build windows

package probe

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ipSuccess             = 0
	ipDestNetUnreachable  = 11002
	ipDestHostUnreachable = 11003
	ipDestProtUnreachable = 11004
	ipDestPortUnreachable = 11005
	ipReqTimedOut         = 11010
	ipTTLExpiredTransit   = 11013
)

type ipOptionInformation struct {
	TTL         byte
	TOS         byte
	Flags       byte
	OptionsSize byte
	OptionsData uintptr
}

type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       ipOptionInformation
}

var (
	iphlpapi = windows.NewLazySystemDLL("iphlpapi.dll")

	procIcmpCreateFile  = iphlpapi.NewProc("IcmpCreateFile")
	procIcmpCloseHandle = iphlpapi.NewProc("IcmpCloseHandle")
	procIcmpSendEcho    = iphlpapi.NewProc("IcmpSendEcho")
)

type ICMPProber struct {
	Timeout time.Duration

	mu     sync.RWMutex
	handle windows.Handle
	closed bool
}

func NewICMPProber(timeout time.Duration) (*ICMPProber, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}

	r1, _, err := procIcmpCreateFile.Call()
	handle := windows.Handle(r1)

	if handle == windows.InvalidHandle {
		if err != nil && err != syscall.Errno(0) {
			return nil, fmt.Errorf("IcmpCreateFile: %w", err)
		}
		return nil, fmt.Errorf("IcmpCreateFile failed")
	}

	return &ICMPProber{
		Timeout: timeout,
		handle:  handle,
	}, nil
}

func (p *ICMPProber) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	r1, _, err := procIcmpCloseHandle.Call(uintptr(p.handle))
	if r1 == 0 {
		if err != nil && err != syscall.Errno(0) {
			return fmt.Errorf("IcmpCloseHandle: %w", err)
		}
		return fmt.Errorf("IcmpCloseHandle failed")
	}

	p.closed = true
	p.handle = windows.InvalidHandle
	return nil
}

func (p *ICMPProber) Probe(
	ctx context.Context,
	target net.IP,
	ttl int,
) (Result, error) {
	result := Result{
		TTL:    ttl,
		Status: StatusUnknown,
	}

	if ttl < 1 || ttl > 255 {
		return result, fmt.Errorf("invalid TTL: %d", ttl)
	}

	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
	}

	ip := target.To4()
	if ip == nil {
		return result, fmt.Errorf("target must be an IPv4 address")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return result, fmt.Errorf("ICMP prober is closed")
	}

	destination := ipv4ToUint32(ip)
	payload := []byte("RouteScopeProbe")
	options := ipOptionInformation{TTL: byte(ttl)}
	replyBuffer := make([]byte, 2048)

	timeoutMS := uint32(p.Timeout / time.Millisecond)
	if timeoutMS < 1 {
		timeoutMS = 1
	}

	ret, _, callErr := procIcmpSendEcho.Call(
		uintptr(p.handle),
		uintptr(destination),
		uintptr(unsafe.Pointer(&payload[0])),
		uintptr(len(payload)),
		uintptr(unsafe.Pointer(&options)),
		uintptr(unsafe.Pointer(&replyBuffer[0])),
		uintptr(len(replyBuffer)),
		uintptr(timeoutMS),
	)

	if ret == 0 {
		if callErr == nil || callErr == syscall.Errno(0) {
			return result, fmt.Errorf("IcmpSendEcho failed without a Windows error code")
		}

		errno, ok := callErr.(syscall.Errno)
		if !ok {
			return result, fmt.Errorf("IcmpSendEcho: %w", callErr)
		}

		switch uint32(errno) {
		case ipReqTimedOut:
			result.Status = StatusTimeout
			return result, nil
		case ipDestNetUnreachable,
			ipDestHostUnreachable,
			ipDestProtUnreachable,
			ipDestPortUnreachable:
			result.Status = StatusUnreachable
			return result, nil
		default:
			return result, fmt.Errorf("IcmpSendEcho: %w", callErr)
		}
	}

	reply := (*icmpEchoReply)(unsafe.Pointer(&replyBuffer[0]))
	result.Addr = uint32ToIP(reply.Address)
	result.RTT = time.Duration(reply.RoundTripTime) * time.Millisecond

	switch reply.Status {
	case ipSuccess:
		result.Status = StatusSuccess
	case ipTTLExpiredTransit:
		result.Status = StatusTTLExpired
	case ipReqTimedOut:
		result.Status = StatusTimeout
	case ipDestNetUnreachable,
		ipDestHostUnreachable,
		ipDestProtUnreachable,
		ipDestPortUnreachable:
		result.Status = StatusUnreachable
	default:
		return result, fmt.Errorf("IcmpSendEcho returned unexpected ICMP status %d", reply.Status)
	}

	return result, nil
}

func ipv4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0]) |
		uint32(ip[1])<<8 |
		uint32(ip[2])<<16 |
		uint32(ip[3])<<24
}

func uint32ToIP(addr uint32) net.IP {
	return net.IPv4(
		byte(addr),
		byte(addr>>8),
		byte(addr>>16),
		byte(addr>>24),
	)
}
