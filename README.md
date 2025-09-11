# go-multicast

A Go library for efficient multicast UDP packet handling with support for multiple network interfaces.

## Overview

`go-multicast` provides a simple and efficient abstraction for receiving multicast UDP traffic across multiple network interfaces. It features:

- **Multi-interface support**: Listen on multiple network interfaces simultaneously
- **Independent consumers**: Each consumer manages its own multicast connections
- **Flexible API**: Use via listener for managed consumers or create consumers directly
- **Automatic cleanup**: Resources are automatically cleaned up when consumers are closed
- **Thread-safe**: All operations are safe for concurrent use
- **Multicast validation**: Automatic validation of multicast addresses

## Installation

```bash
go get github.com/holoplot/go-multicast
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "log"
    "net"

    "github.com/holoplot/go-multicast/pkg/multicast"
)

func main() {
    // Get multicast-capable interfaces
    ifis, err := net.Interfaces()
    if err != nil {
        log.Fatal(err)
    }

    var multicastIfis []*net.Interface
    for i := range ifis {
        if ifis[i].Flags&net.FlagMulticast != 0 && ifis[i].Flags&net.FlagUp != 0 {
            multicastIfis = append(multicastIfis, &ifis[i])
        }
    }

    // Create listener
    listener := multicast.NewListener(multicastIfis)
    defer listener.Close()

    // Parse multicast address
    addr, err := net.ResolveUDPAddr("udp", "224.1.1.1:12345")
    if err != nil {
        log.Fatal(err)
    }

    // Add consumer
    consumer, err := listener.AddConsumer(addr, func(ifi *net.Interface, src net.Addr, payload []byte) {
        fmt.Printf("Received on %s from %s: %s\n", ifi.Name, src.String(), string(payload))
    })
    if err != nil {
        log.Fatal(err)
    }
    defer consumer.Close()

    // Keep running...
    select {}
}
```

### Command Line Tool

A receiver command is provided for testing:

```bash
# Build the receiver
go build -o bin/receiver ./cmd/receiver

# Listen on multicast address
./bin/receiver 224.1.1.1:12345

# For verbose output
./bin/receiver -v 224.1.1.1:12345
```

### Testing Multicast

You can test multicast functionality using standard tools:

```bash
# Terminal 1: Start receiver
./bin/receiver 224.1.1.1:12345

# Terminal 2: Send test data (Linux/macOS)
echo "Hello multicast" | socat - UDP-DATAGRAM:224.1.1.1:12345,broadcast
```

## Requirements

- Go 1.19 or later

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
