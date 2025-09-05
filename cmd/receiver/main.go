package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/holoplot/go-multicast/pkg/multicast"
)

const (
	defaultLogLevel = slog.LevelInfo
)

func main() {
	var (
		verbose = flag.Bool("v", false, "Enable verbose logging")
		help    = flag.Bool("h", false, "Show help")
	)
	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	// Setup logging
	logLevel := defaultLogLevel
	if *verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Check arguments
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: multicast address is required\n\n")
		printUsage()
		os.Exit(1)
	}

	// Parse multicast address
	addr, err := net.ResolveUDPAddr("udp", args[0])
	if err != nil {
		slog.Error("failed to parse multicast address", "addr", args[0], "error", err)
		os.Exit(1)
	}

	// Validate it's a multicast address
	if !addr.IP.IsMulticast() {
		slog.Error("address is not a multicast address", "addr", addr.String())
		os.Exit(1)
	}

	// Get all network interfaces
	ifis, err := net.Interfaces()
	if err != nil {
		slog.Error("failed to get network interfaces", "error", err)
		os.Exit(1)
	}

	// Filter to multicast-capable interfaces
	var multicastIfis []*net.Interface
	for i := range ifis {
		if ifis[i].Flags&net.FlagMulticast != 0 && ifis[i].Flags&net.FlagUp != 0 {
			multicastIfis = append(multicastIfis, &ifis[i])
		}
	}

	if len(multicastIfis) == 0 {
		slog.Error("no multicast-capable interfaces found")
		os.Exit(1)
	}

	slog.Info("starting multicast receiver",
		"addr", addr.String(),
		"interfaces", len(multicastIfis))

	for _, ifi := range multicastIfis {
		slog.Debug("using interface", "name", ifi.Name, "index", ifi.Index)
	}

	// Create listener
	listener := multicast.NewListener(multicastIfis)
	defer listener.Close()

	// Create consumer with callback
	consumer, err := listener.AddConsumer(addr, func(ifi *net.Interface, payload []byte) {
		slog.Info("packet received", "interface", ifi.Name, "length", len(payload))
		fmt.Printf("%s", hex.Dump(payload))
	})
	if err != nil {
		slog.Error("failed to add consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	slog.Info("receiver started, listening for packets...")

	select {}
}

func printUsage() {
	fmt.Printf(`Usage: %s [options] <multicast_address>

Receive multicast UDP packets from the specified address.

Arguments:
  multicast_address    Multicast address in format IP:PORT (e.g., 224.1.1.1:12345)

Options:
  -v    Enable verbose logging (debug level)
  -h    Show this help message

Examples:
  %s 224.1.1.1:12345
  %s -v 239.255.255.250:1900

`, os.Args[0], os.Args[0], os.Args[0])
}
