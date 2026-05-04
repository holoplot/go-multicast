//go:build darwin

package multicast

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
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

	// Required on BSD/Darwin so multiple per-interface sockets can share the same multicast addr:port.
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1); err != nil {
		_ = syscall.Close(s)

		return nil, fmt.Errorf("failed to set SO_REUSEPORT: %w", err)
	}

	// Darwin equivalent of Linux's SO_BINDTODEVICE: restrict the socket to a single interface by index.
	if err := syscall.SetsockoptInt(s, syscall.IPPROTO_IP, unix.IP_BOUND_IF, ifi.Index); err != nil {
		_ = syscall.Close(s)

		return nil, fmt.Errorf("failed to set IP_BOUND_IF: %w", err)
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
