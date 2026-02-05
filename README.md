# retina-commons

Shared Go types and API definitions for the Retina network measurement system.

## Overview

This package defines the core data structures used by Retina components for distributed network probing and topology measurement. These types form the communication protocol between components in the Retina system.

**Retina Architecture:**
- **Generator** creates `ProbingDirective` messages based on measurement strategy
- **Orchestrator** distributes directives to agents via TCP and collects `ForwardingInfoElement` results  
- **Agents** execute network probes and return FIE measurements

All communication uses JSON over newline-delimited TCP streams.

```
┌───────────┐
│ Generator │
└─────┬─────┘
      │ ProbingDirective
      ▼
┌─────────────┐         ProbingDirective         ┌───────┐
│Orchestrator │────────────────────────────────▶ │ Agent │
└─────────────┘                                  └───┬───┘
      ▲           ForwardingInfoElement              │
      └──────────────────────────────────────────────┘
```

### Core Types

- **ProbingDirective**: Instructions sent from orchestrator to agents, specifying which destinations to probe at which TTL values
- **ForwardingInfoElement**: Measurement results returned by agents, containing near/far probe observations
- **SystemStatus**: Orchestration state and configuration (probing rates, disallowed destinations, active agents)
- **Auth types**: Agent authentication flow (AuthRequest, AuthResponse)
- **Protocol types**: Shared enums and headers (IPVersion, Protocol, NextHeader variants)

All types are designed for JSON serialization over TCP connections between distributed components.

## Installation

```bash
go get github.com/dioptra-io/retina-commons@latest
```

## Usage

### Creating a Probing Directive

```go
import api "github.com/dioptra-io/retina-commons/api/v1"

// Create a UDP probing directive
pd := &api.ProbingDirective{
    ProbingDirectiveID: 1,
    AgentID:            "agent-1",
    IPVersion:          api.IPv4,
    Protocol:           api.UDP,
    DestinationAddress: net.ParseIP("8.8.8.8"),
    NearTTL:            10,  // Agent will probe at TTL 10 and 11
    NextHeader: api.NextHeader{
        UDPNextHeader: &api.UDPNextHeader{
            SourcePort:      50000,
            DestinationPort: 33434,
        },
    },
}

// Serialize for transmission
data, err := json.Marshal(pd)
```

### Handling Measurement Results

```go
// Deserialize incoming result
var fie api.ForwardingInfoElement
if err := json.Unmarshal(data, &fie); err != nil {
    log.Fatal(err)
}

// Extract RTT measurements
nearRTT := fie.NearInfo.ReceivedTimestamp.Sub(fie.NearInfo.SentTimestamp)
farRTT := fie.FarInfo.ReceivedTimestamp.Sub(fie.FarInfo.SentTimestamp)

fmt.Printf("Near hop (%s): %v RTT\n", fie.NearInfo.ReplyAddress, nearRTT)
fmt.Printf("Far hop (%s): %v RTT\n", fie.FarInfo.ReplyAddress, farRTT)
```

### Protocol Support

The package supports three probing protocols:

```go
// ICMP (IPv4)
api.Protocol = api.ICMP
NextHeader: api.NextHeader{
    ICMPNextHeader: &api.ICMPNextHeader{
        FirstHalfWord:  0x0001,  // ICMP echo ID
        SecondHalfWord: 0x0000,  // Sequence number
    },
}

// UDP (IPv4)
api.Protocol = api.UDP
NextHeader: api.NextHeader{
    UDPNextHeader: &api.UDPNextHeader{
        SourcePort:      50000,
        DestinationPort: 33434,
    },
}

// ICMPv6 (IPv6)
api.Protocol = api.ICMPv6
NextHeader: api.NextHeader{
    ICMPv6NextHeader: &api.ICMPv6NextHeader{
        FirstHalfWord:  0x0001,
        SecondHalfWord: 0x0000,
    },
}
```

## Development

### Testing

The package includes comprehensive tests for JSON serialization:

```bash
go test ./...
```

**Note on Coverage**: The test suite reports `coverage: [no statements]` because this package contains only type definitions and constants (no executable functions). This is expected and correct. The tests verify the JSON serialization contract, edge cases (nil vs empty, omitempty behavior), and type compatibility across the wire protocol.

### Running Tests

```bash
# Run all tests
go test -v ./...

# Check JSON encoding behavior
go test -v -run TestNextHeader_Omitempty

# Verify protocol constants
go test -v -run TestProtocol_Values
```

## Wire Format

All types use JSON encoding with snake_case field names:

```json
{
  "probing_directive_id": 123,
  "agent_id": "agent-1",
  "ip_version": 4,
  "protocol": 17,
  "destination_address": "8.8.8.8",
  "near_ttl": 10,
  "next_header": {
    "udp_next_header": {
      "source_port": 50000,
      "destination_port": 33434
    }
  }
}
```

Time values are encoded as RFC3339 strings, durations as int64 nanoseconds.

## Related Projects

**Part of the Retina distributed measurement system:**

- [retina-generator](https://github.com/dioptra-io/retina-generator) - Creates probing directives based on measurement strategy
- [retina-orchestrator](https://github.com/dioptra-io/retina-orchestrator) - Distributes directives to agents and collects forwarding information elements
- [retina-agent](https://github.com/dioptra-io/retina-agent) - Executes network probes and returns measurement results

## Contributing

This package defines the communication protocol between Retina components. Changes to types or JSON tags constitute breaking changes and require coordination across agent and orchestrator deployments.

When modifying types:
1. Update tests to verify JSON compatibility
2. Document wire format changes
3. Consider backward compatibility requirements
4. Test with both agent and orchestrator implementations

## License

MIT License - see [LICENSE](LICENSE) for details

## Authors

- Ufuk Bombar <ufuk.bombar@lip6.fr>
- Elena Nardi <elena.nardi@lip6.fr>
- Saied Kazemi <saied.kazemi@lip6.fr>