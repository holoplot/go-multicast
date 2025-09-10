//go:build linux

package multicast

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/net/ipv4"
)

func (c *Consumer) openPacketConn(ifi *net.Interface) (*ipv4.PacketConn, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		_ = syscall.Close(s)

		return nil, fmt.Errorf("failed to set SO_REUSEADDR: %w", err)
	}

	if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifi.Name); err != nil {
		_ = syscall.Close(s)

		return nil, fmt.Errorf("failed to set SO_BINDTODEVICE: %w", err)
	}

	lsa := syscall.SockaddrInet4{Port: c.addr.Port}
	copy(lsa.Addr[:], c.addr.IP.To4())

	if err := syscall.Bind(s, &lsa); err != nil {
		_ = syscall.Close(s)

		return nil, fmt.Errorf("failed to bind socket: %w", err)
	}

	f := os.NewFile(uintptr(s), "")
	conn, err := net.FilePacketConn(f)
	_ = f.Close()

	if err != nil {
		_ = syscall.Close(s)

		return nil, fmt.Errorf("failed to create packet conn from file: %w", err)
	}

	return ipv4.NewPacketConn(conn), nil
}
