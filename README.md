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
```

Receiving side:

```go
var in wire.ProbingDirective
if err := proto.Unmarshal(data, &in); err != nil {
    // handle error
}
pd, err := model.ProbingDirectiveFromProto(&in)
```

## Model layer

`model` mirrors `wire/v2` with idiomatic Go types: `uint8` TTLs, `net.IP`
addresses, `time.Time` timestamps, and `ID` instead of `Id` in field names. It
covers `ProbingDirective`, `ForwardingInfoElement`, and their dependencies
(`Agent`, `Info`). `NextHeader` stays the native `wire.NextHeader`.

Conversion is validating in both directions:

- `FromProto` rejects TTL overflow, unparseable or missing required IPs, and
  absent or invalid timestamps, rather than truncating or normalizing.
- `ToProto` checks required fields before serializing, so a malformed model
  value can't produce a wire message that `FromProto` would reject.
- Passing `nil` to a `FromProto` function is an error. A required nested field
  missing inside an otherwise valid message (e.g. `ForwardingInfoElement.Agent`)
  is a validation failure.
- `NearInfo`/`FarInfo` are nilable: `nil` means the probe timed out. When
  present, an `Info` always carries both timestamps. `SourceAddress` is nil when
  unavailable (e.g. both probes timed out).

Compare IPs with `net.IP.Equal` and times with `time.Time.Equal`, not `==`.
IPv4 addresses are normalized to 4-byte form, and times come back in UTC.

## Wire format

Messages are defined in [`wire/v2/wire.proto`](wire/v2/wire.proto). Conventions:

- IP addresses are canonical textual strings, without port or zone identifier.
- Timestamps are `google.protobuf.Timestamp`, always UTC.
- `IPVersion` and `Protocol` enum values match IP version numbers and IANA
  protocol numbers.
- `AuthRequest`/`AuthResponse` must only be exchanged over mutually
  authenticated TLS. Never log `secret`.
- PD files are JSONL: one
  [protojson](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson)-encoded
  `ProbingDirective` per line.

### Regenerating

Generated code is checked in, so consumers need no protobuf tooling. To
regenerate after editing `wire.proto`, install
[`buf`](https://buf.build/docs/installation) and `protoc-gen-go` v1.36.12:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
make generate
```

Commit `wire.proto` and `wire.pb.go` together. Never edit `wire.pb.go` by hand.

## Development

Run `make help` for available targets. Run tests with `make test`.

## License

MIT License - see [LICENSE](LICENSE) for details