package multicast

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"

	"golang.org/x/net/ipv4"
)

const (
	maxMTU = 1500
)

type ConsumerPacketCallback func(ifi *net.Interface, src net.Addr, payload []byte)

type Consumer struct {
	addr            *net.UDPAddr
	cb              ConsumerPacketCallback
	ifis            []*net.Interface
	ipv4PacketConns map[int]*ipv4.PacketConn
	mutex           sync.Mutex
	closed          bool
}

func NewConsumer(addr *net.UDPAddr, ifis []*net.Interface, cb ConsumerPacketCallback) (*Consumer, error) {
	if !addr.IP.IsMulticast() {
		return nil, fmt.Errorf("address %s is not a multicast address", addr.String())
	}

	c := &Consumer{
		addr:            addr,
		cb:              cb,
		ifis:            ifis,
		ipv4PacketConns: make(map[int]*ipv4.PacketConn),
	}

	if err := c.start(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Consumer) start() error {
	for _, ifi := range c.ifis {
		if ifi.Flags&net.FlagMulticast == 0 {
			continue
		}

		pc, err := c.openPacketConn(ifi)
		if err != nil {
			c.cleanup()
			return fmt.Errorf("failed to open multicast socket on interface %s: %w", ifi.Name, err)
		}

		if err := pc.SetControlMessage(ipv4.FlagDst, true); err != nil {
			c.cleanup()
			return fmt.Errorf("failed to set control message on interface %s: %w", ifi.Name, err)
		}

		if err := pc.JoinGroup(ifi, c.addr); err != nil {
			c.cleanup()
			return fmt.Errorf("failed to join group %s on interface %s: %w", c.addr.String(), ifi.Name, err)
		}

		c.ipv4PacketConns[ifi.Index] = pc

		go c.readLoop(pc, ifi)
	}

	return nil
}

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

func (c *Consumer) readLoop(pc *ipv4.PacketConn, ifi *net.Interface) {
	buf := make([]byte, maxMTU)

	for {
		c.mutex.Lock()
		if c.closed {
			c.mutex.Unlock()
			return
		}
		c.mutex.Unlock()

		n, cm, src, err := pc.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// Log error but continue
			continue
		}

		// Check if the destination matches our multicast address
		if cm != nil && cm.Dst.Equal(c.addr.IP) {
			// Create a copy of the payload for the callback
			payload := make([]byte, n)
			copy(payload, buf[:n])

			c.cb(ifi, src, payload)
		}
	}
}

func (c *Consumer) cleanup() {
	for _, pc := range c.ipv4PacketConns {
		_ = pc.Close()
	}

	c.ipv4PacketConns = make(map[int]*ipv4.PacketConn)
}

func (c *Consumer) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return
	}

	c.closed = true

	for _, pc := range c.ipv4PacketConns {
		_ = pc.Close()
	}

	c.ipv4PacketConns = make(map[int]*ipv4.PacketConn)
}

func (c *Consumer) Address() *net.UDPAddr {
	return c.addr
}

func (c *Consumer) Interfaces() []*net.Interface {
	return c.ifis
}
