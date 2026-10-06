# retina-commons

Shared Go types and wire definitions for the Retina network measurement system.

## Overview

This package defines the core data structures used by Retina components for
distributed network probing and topology measurement, and converts them to and
from the Protobuf messages in `wire/v2`. Those messages form the communication
protocol between components in the Retina system.

**Retina Architecture:**
- **PD source**: Produces probing directives (PDs) as JSONL files: the
  [Generator](https://github.com/dioptra-io/retina-generator) or the
  [retina-tools](https://github.com/dioptra-io/retina-tools) pipeline for
  Iris-derived directives
- **[Orchestrator](https://github.com/dioptra-io/retina-orchestrator)**: Loads PD files,
  distributes directives to agents, collects forwarding info elements (FIEs)
- **[Agents](https://github.com/dioptra-io/retina-agent)**: Execute network probes and
  return measurements

Orchestrator-agent communication uses Protobuf messages over length-prefixed
TCP streams (see the `framing` package). PD files are JSONL, one
protojson-encoded directive per line.

```
┌───────────┐
│ PD source │
└─────┬─────┘
      │ PD files (JSONL)
      ▼
┌─────────────┐         ProbingDirective         ┌───────┐
│Orchestrator │────────────────────────────────▶ │ Agent │
└─────────────┘                                  └───┬───┘
      ▲           ForwardingInfoElement              │
      └──────────────────────────────────────────────┘
```

## Layout

| Path | Contents |
|---|---|
| `model/` | Core Go types for probing directives and FIEs, with validating conversion to and from the `wire/v2` messages |
| `wire/v2/` | Protobuf schema (`wire.proto`) and generated Go code (`wire.pb.go`, package `wire`) |
| `framing/` | Length-prefixed message framing over TCP streams |
| `api/v1/` | Legacy JSON types, superseded by `model` and `wire/v2`. Kept because research code still depends on it. Don't use it in new components. |

## Installation

```bash
go get github.com/dioptra-io/retina-commons/v2@latest
```

Requires Go 1.24 or later.

## Usage

Components work with the `model` types and convert at the wire boundary.

```go
import (
    "net"

    "google.golang.org/protobuf/proto"

    "github.com/dioptra-io/retina-commons/v2/model"
    wire "github.com/dioptra-io/retina-commons/v2/wire/v2"
)

pd := model.ProbingDirective{
    ProbingDirectiveID: 1,
    IPVersion:          wire.IPVersion_IP_VERSION_IPV4,
    Protocol:           wire.Protocol_PROTOCOL_UDP,
    AgentID:            "agent-1",
    DestinationAddress: net.ParseIP("8.8.8.8"),
    NearTTL:            10, // Agent will probe at TTL 10 and 11
    NextHeader: &wire.NextHeader{
        Header: &wire.NextHeader_UdpNextHeader{
            UdpNextHeader: &wire.UDPNextHeader{
                SourcePort:      50000,
                DestinationPort: 33434,
            },
        },
    },
}

msg, err := pd.ToProto()
if err != nil {
    // handle error
}
data, err := proto.Marshal(msg)

## License

MIT License - see [LICENSE](LICENSE) for details
