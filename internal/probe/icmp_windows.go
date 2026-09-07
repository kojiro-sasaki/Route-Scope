//go:build windows

package probe

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ipSuccess             = 0
	ipReqTimedOut         = 11010
	ipTTLExpiredTransit   = 11013
	ipDestNetUnreachable  = 11002
	ipDestHostUnreachable = 11003
	ipDestProtUnreachable = 11004
	ipDestPortUnreachable = 11005
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
	ID      uint16

	mu     sync.RWMutex
	handle windows.Handle
	closed bool
}

func NewICMPProber(timeout time.Duration) (*ICMPProber, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf(
			"timeout must be greater than zero",
		)
	}

	r1, _, err := procIcmpCreateFile.Call()

	handle := windows.Handle(r1)

	if handle == windows.InvalidHandle {
		if err != nil {
			return nil, fmt.Errorf(
				"IcmpCreateFile: %w",
				err,
			)
		}

		return nil, fmt.Errorf(
			"IcmpCreateFile failed",
		)
	}

	return &ICMPProber{
		Timeout: timeout,
		ID:      uint16(time.Now().UnixNano()),
		handle:  handle,
	}, nil
}

func (p *ICMPProber) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	r1, _, err := procIcmpCloseHandle.Call(
		uintptr(p.handle),
	)

	if r1 == 0 {
		if err != nil {
			return fmt.Errorf(
				"IcmpCloseHandle: %w",
				err,
			)
		}

		return fmt.Errorf(
			"IcmpCloseHandle failed",
		)
	}

	p.closed = true
	p.handle = windows.InvalidHandle

	return nil
}

func (p *ICMPProber) Probe(
	ctx context.Context,
	target string,
	ttl int,
) (Result, error) {
	var result Result

	result.TTL = ttl
	result.Status = StatusUnknown

	if ttl < 1 || ttl > 255 {
		return result, fmt.Errorf(
			"invalid TTL: %d",
			ttl,
		)
	}

	select {
	case <-ctx.Done():
		return result, ctx.Err()

	default:
	}

	ip, err := resolveIPv4(target)
	if err != nil {
		return result, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return result, fmt.Errorf(
			"ICMP prober is closed",
		)
	}

	destination := ipv4ToUint32(ip)

	payload := make([]byte, 16)

	binary.BigEndian.PutUint16(
		payload[0:2],
		p.ID,
	)

	binary.BigEndian.PutUint16(
		payload[2:4],
		uint16(ttl),
	)

	binary.BigEndian.PutUint64(
		payload[4:12],
		uint64(time.Now().UnixNano()),
	)

	options := ipOptionInformation{
		TTL: byte(ttl),
	}

	replyBuffer := make([]byte, 2048)

	timeoutMS := uint32(
		p.Timeout / time.Millisecond,
	)

	if timeoutMS < 1 {
		timeoutMS = 1
	}

	ret, _, _ := procIcmpSendEcho.Call(
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
		result.Timeout = true
		result.Status = StatusTimeout

		return result, nil
	}

	reply := (*icmpEchoReply)(
		unsafe.Pointer(&replyBuffer[0]),
	)

	result.Addr = uint32ToIP(
		reply.Address,
	)

	result.RTT =
		time.Duration(reply.RoundTripTime) *
			time.Millisecond

	switch reply.Status {
	case ipSuccess:
		result.Reached = true
		result.Status = StatusSuccess

	case ipTTLExpiredTransit:
		result.Status = StatusTTLExpired

	case ipReqTimedOut:
		result.Timeout = true
		result.Status = StatusTimeout

	case ipDestNetUnreachable,
		ipDestHostUnreachable,
		ipDestProtUnreachable,
		ipDestPortUnreachable:
		result.Timeout = true
		result.Status = StatusUnreachable

	default:
		result.Timeout = true
		result.Status = StatusUnknown
	}

	return result, nil
}

func resolveIPv4(target string) (net.IP, error) {
	ip := net.ParseIP(target)

	if ip != nil {
		ip = ip.To4()

		if ip == nil {
			return nil, fmt.Errorf(
				"target %q is not IPv4",
				target,
			)
		}

		return ip, nil
	}

	ips, err := net.LookupIP(target)

	if err != nil {
		return nil, fmt.Errorf(
			"resolve %q: %w",
			target,
			err,
		)
	}

	for _, candidate := range ips {
		if ipv4 := candidate.To4(); ipv4 != nil {
			return ipv4, nil
		}
	}

	return nil, fmt.Errorf(
		"no IPv4 address found for %q",
		target,
	)
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
