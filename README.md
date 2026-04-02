# retina-commons

Shared Go types and API definitions for the Retina network measurement system.

## Overview

This package defines the core data structures used by Retina components for distributed network probing and topology measurement. These types form the communication protocol between components in the Retina system.

**Retina Architecture:**
- **Generator**: Creates probing directives
- **Orchestrator**: Distributes directives to agents, collects FIEs
- **Agents**: Execute network probes and return measurements

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

## Installation

```bash
go get github.com/dioptra-io/retina-commons@latest
```

## Usage

```go
import api "github.com/dioptra-io/retina-commons/api/v1"

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

data, err := json.Marshal(pd)
```

## Wire Format

All types use JSON encoding with snake_case field names. Time values are encoded as RFC3339 strings, durations as int64 nanoseconds.

## Testing

```bash
go test ./...
```

**Note on coverage**: The test suite reports `coverage: [no statements]` because this package contains only type definitions and constants. This is expected — the tests verify the JSON serialization contract and omitempty behavior across the wire protocol.

## License

MIT License - see [LICENSE](LICENSE) for details