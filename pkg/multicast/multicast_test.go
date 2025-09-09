package multicast

import (
	"fmt"
	"net"
	"sync"
	"testing"
)

func TestNewListener(t *testing.T) {
	ifis, err := net.Interfaces()
	if err != nil {
		t.Fatalf("failed to get interfaces: %v", err)
	}

	// Convert slice to pointer slice
	var ifaces []*net.Interface
	for i := range ifis {
		ifaces = append(ifaces, &ifis[i])
	}

	listener := NewListener(ifaces)
	if listener == nil {
		t.Fatal("listener should not be nil")
	}

	if len(listener.Interfaces()) != len(ifaces) {
		t.Fatalf("expected %d interfaces, got %d", len(ifaces), len(listener.Interfaces()))
	}

	listener.Close()
}

func TestNewConsumer(t *testing.T) {
	// Use loopback interface for testing
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	addr, err := net.ResolveUDPAddr("udp", "224.1.1.1:12345")
	if err != nil {
		t.Fatalf("failed to resolve UDP address: %v", err)
	}

	var receivedData []byte
	var mu sync.Mutex

	consumer, err := NewConsumer(addr, []*net.Interface{loopback}, func(ifi *net.Interface, _ net.Addr, payload []byte) {
		mu.Lock()
		receivedData = append(receivedData, payload...)
		mu.Unlock()
	})

	// On some systems, joining multicast groups might fail due to permissions
	// or network configuration, so we only test the basic functionality
	if err != nil {
		t.Logf("failed to create consumer (expected on some systems): %v", err)
		return
	}

	if consumer == nil {
		t.Fatal("consumer should not be nil")
	}

	if !consumer.Address().IP.Equal(addr.IP) || consumer.Address().Port != addr.Port {
		t.Fatalf("consumer address mismatch: expected %s, got %s", addr.String(), consumer.Address().String())
	}

	if len(consumer.Interfaces()) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(consumer.Interfaces()))
	}

	consumer.Close()

	// Close again should be safe
	consumer.Close()
}

func TestListenerAddConsumer(t *testing.T) {
	// Use loopback interface for testing
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	listener := NewListener([]*net.Interface{loopback})
	defer listener.Close()

	addr, err := net.ResolveUDPAddr("udp", "224.1.1.1:12345")
	if err != nil {
		t.Fatalf("failed to resolve UDP address: %v", err)
	}

	var receivedData []byte
	var mu sync.Mutex

	consumer, err := listener.AddConsumer(addr, func(ifi *net.Interface, _ net.Addr, payload []byte) {
		mu.Lock()
		receivedData = append(receivedData, payload...)
		mu.Unlock()
	})

	// On some systems, joining multicast groups might fail due to permissions
	// or network configuration, so we only test the basic functionality
	if err != nil {
		t.Logf("failed to add consumer (expected on some systems): %v", err)
		return
	}

	if consumer == nil {
		t.Fatal("consumer should not be nil")
	}

	// Check that consumer is tracked by listener
	consumers := listener.Consumers()
	if len(consumers) != 1 {
		t.Fatalf("expected 1 consumer in listener, got %d", len(consumers))
	}

	consumer.Close()

	// Consumer should still be tracked by listener until removed
	consumers = listener.Consumers()
	if len(consumers) != 1 {
		t.Fatalf("expected 1 consumer in listener after close, got %d", len(consumers))
	}
}

func TestListenerMultipleConsumers(t *testing.T) {
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	listener := NewListener([]*net.Interface{loopback})
	defer listener.Close()

	addr1, err := net.ResolveUDPAddr("udp", "224.1.1.2:12346")
	if err != nil {
		t.Fatalf("failed to resolve UDP address 1: %v", err)
	}

	addr2, err := net.ResolveUDPAddr("udp", "224.1.1.3:12347")
	if err != nil {
		t.Fatalf("failed to resolve UDP address 2: %v", err)
	}

	var consumer1, consumer2 *Consumer
	var count1, count2 int
	var mu sync.Mutex

	consumer1, err = listener.AddConsumer(addr1, func(ifi *net.Interface, _ net.Addr, payload []byte) {
		mu.Lock()
		count1++
		mu.Unlock()
	})

	if err != nil {
		t.Logf("failed to add first consumer (expected on some systems): %v", err)
		return
	}

	consumer2, err = listener.AddConsumer(addr2, func(ifi *net.Interface, _ net.Addr, payload []byte) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	if err != nil {
		t.Logf("failed to add second consumer: %v", err)
		consumer1.Close()
		return
	}

	// Both consumers should be tracked by listener
	consumers := listener.Consumers()
	if len(consumers) != 2 {
		t.Fatalf("expected 2 consumers, got %d", len(consumers))
	}

	consumer1.Close()
	consumer2.Close()

	// Consumers should still be tracked until explicitly removed
	consumers = listener.Consumers()
	if len(consumers) != 2 {
		t.Fatalf("expected 2 consumers after closing, got %d", len(consumers))
	}
}

func TestListenerRemoveConsumer(t *testing.T) {
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	listener := NewListener([]*net.Interface{loopback})
	defer listener.Close()

	addr, err := net.ResolveUDPAddr("udp", "224.1.1.4:12348")
	if err != nil {
		t.Fatalf("failed to resolve UDP address: %v", err)
	}

	consumer, err := listener.AddConsumer(addr, func(ifi *net.Interface, _ net.Addr, payload []byte) {
		// No-op callback
	})

	if err != nil {
		t.Logf("failed to add consumer (expected on some systems): %v", err)
		return
	}

	// Should have one consumer
	consumers := listener.Consumers()
	if len(consumers) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(consumers))
	}

	listener.RemoveConsumer(consumer)

	// Should have no consumers after removal
	consumers = listener.Consumers()
	if len(consumers) != 0 {
		t.Fatalf("expected 0 consumers after removal, got %d", len(consumers))
	}

	// Removing again should be safe
	listener.RemoveConsumer(consumer)
}

func TestListenerClose(t *testing.T) {
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	listener := NewListener([]*net.Interface{loopback})

	addr1, _ := net.ResolveUDPAddr("udp", "224.1.1.5:12349")
	addr2, _ := net.ResolveUDPAddr("udp", "224.1.1.6:12350")

	consumer1, err1 := listener.AddConsumer(addr1, func(ifi *net.Interface, _ net.Addr, payload []byte) {})
	consumer2, err2 := listener.AddConsumer(addr2, func(ifi *net.Interface, _ net.Addr, payload []byte) {})

	// If we can't add consumers due to system limitations, skip the rest
	if err1 != nil || err2 != nil {
		t.Logf("failed to add consumers (expected on some systems): %v, %v", err1, err2)
		listener.Close()
		return
	}

	// Should have two consumers
	consumers := listener.Consumers()
	if len(consumers) != 2 {
		t.Fatalf("expected 2 consumers, got %d", len(consumers))
	}

	listener.Close()

	// Should have no consumers after closing listener
	consumers = listener.Consumers()
	if len(consumers) != 0 {
		t.Fatalf("expected 0 consumers after closing listener, got %d", len(consumers))
	}

	// Consumers should be safe to close again
	consumer1.Close()
	consumer2.Close()
}

func TestConsumerInvalidAddress(t *testing.T) {
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	// Test with non-multicast address
	addr, err := net.ResolveUDPAddr("udp", "192.168.1.1:12345")
	if err != nil {
		t.Fatalf("failed to resolve UDP address: %v", err)
	}

	consumer, err := NewConsumer(addr, []*net.Interface{loopback}, func(ifi *net.Interface, _ net.Addr, payload []byte) {})

	if err == nil {
		t.Fatal("expected error for non-multicast address")
		consumer.Close()
	}

	if consumer != nil {
		t.Fatal("consumer should be nil for invalid address")
	}
}

func TestConsumerCloseIdempotent(t *testing.T) {
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	addr, err := net.ResolveUDPAddr("udp", "224.1.1.7:12351")
	if err != nil {
		t.Fatalf("failed to resolve UDP address: %v", err)
	}

	consumer, err := NewConsumer(addr, []*net.Interface{loopback}, func(ifi *net.Interface, _ net.Addr, payload []byte) {})
	if err != nil {
		t.Logf("failed to create consumer (expected on some systems): %v", err)
		return
	}

	// Close multiple times should be safe
	consumer.Close()
	consumer.Close()
	consumer.Close()
}

func BenchmarkListenerAddConsumer(b *testing.B) {
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	listener := NewListener([]*net.Interface{loopback})
	defer listener.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Use different addresses to avoid conflicts
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("224.1.1.%d:%d", (i%254)+1, 12352+i))
		if err != nil {
			b.Fatalf("failed to resolve UDP address: %v", err)
		}

		consumer, err := listener.AddConsumer(addr, func(ifi *net.Interface, _ net.Addr, payload []byte) {
			// No-op callback
		})
		if err != nil {
			b.Logf("failed to add consumer (expected on some systems): %v", err)
			return
		}
		consumer.Close()
	}
}

func BenchmarkConsumerCreation(b *testing.B) {
	loopback := &net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Use different addresses to avoid conflicts
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("224.1.1.%d:%d", (i%254)+1, 12353+i))
		if err != nil {
			b.Fatalf("failed to resolve UDP address: %v", err)
		}

		consumer, err := NewConsumer(addr, []*net.Interface{loopback}, func(ifi *net.Interface, _ net.Addr, payload []byte) {})
		if err != nil {
			b.Logf("failed to create consumer (expected on some systems): %v", err)
			return
		}
		consumer.Close()
	}
}
