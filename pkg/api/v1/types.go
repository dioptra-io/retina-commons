// Copyright (c) 2025 Dioptra
// SPDX-License-Identifier: MIT
package api

import (
	"net"
	"time"
)

// IPVersion identifies the IP version used for a probe/observation (IPv4 or IPv6). It is encoded as the conventional
// version number (4 or 6).
type IPVersion uint8

const (
	// TypeIPv4 indicates IPv4 (version number 4).
	TypeIPv4 IPVersion = 4
	// TypeIPv6 indicates IPv6 (version number 6).
	TypeIPv6 IPVersion = 6
)

// Protocol identifies the L4 protocol carried by the IP packet (with the exception of ICMP and ICMPv6). Values match
// the IANA IP Protocol Numbers (e.g., UDP=17, ICMP=1, ICMPv6=58).
type Protocol uint8

const (
	// ICMP is the IPv4 ICMP protocol number (1).
	ICMP Protocol = 1
	// UDP is the UDP protocol number (17).
	UDP Protocol = 17
	// ICMPv6 is the ICMPv6 protocol number (58).
	ICMPv6 Protocol = 58
)

// ICMPNextHeader carries protocol-specific header fields for ICMP probes/replies. The fields are stored as two 16-bit
// words to preserve raw header bits across different ICMP message types.
type ICMPNextHeader struct {
	FirstHalfWord  uint16 `json:"first_half_word"`
	SecondHalfWord uint16 `json:"second_half_word"`
}

// UDPNextHeader carries protocol-specific header fields for UDP probes/replies.
type UDPNextHeader struct {
	// SourcePort is the UDP source port used in the probe packet.
	SourcePort uint16 `json:"source_port"`

	// DestinationPort is the UDP destination port used in the probe packet.
	DestinationPort uint16 `json:"destination_port"`
}

// ICMPv6NextHeader carries protocol-specific header fields for ICMPv6 probes/replies. The fields are stored as two
// 16-bit words to preserve raw header bits across different ICMPv6 message types.
type ICMPv6NextHeader struct {
	FirstHalfWord  uint16 `json:"first_half_word"`
	SecondHalfWord uint16 `json:"second_half_word"`
}

// NextHeader is a tagged container for protocol-specific next-header metadata. Exactly one field should be non-nil,
// matching the Protocol in the surrounding object. The omitempty tags keep JSON payloads compact.
type NextHeader struct {
	// ICMPNextHeader is set when Protocol == ICMP.
	ICMPNextHeader *ICMPNextHeader `json:"icmp_next_header,omitempty"`

	// ICMPNextHeader is set when Protocol == UDP.
	UDPNextHeader *UDPNextHeader `json:"udp_next_header,omitempty"`

	// ICMPNextHeader is set when Protocol == ICMPv6.
	ICMPv6NextHeader *ICMPv6NextHeader `json:"icmpv6_next_header,omitempty"`
}

// ProbingDirective describes what an agent should probe and how. It specifies the IP version and transport protocol,
// the target address, the TTL to probe near the destination, and any protocol-specific header parameters required to
// craft the probe packet.
type ProbingDirective struct {
	// IPVersion selects IPv4 vs IPv6 for the probe packet.
	IPVersion IPVersion `json:"ip_version"`

	// Protocol selects the transport protocol used for probing (ICMP/UDP/ICMPv6).
	Protocol Protocol `json:"protocol"`

	// AgentID identifies the agent that should execute this directive.
	AgentID string `json:"agent_id"`

	// DestinationAddress is the target IP address to probe.
	DestinationAddress net.IP `json:"destination_address"`

	// NearTTL is the TTL/hop-limit value used for the "near" probe (typically close to the destination).
	NearTTL uint8 `json:"near_ttl"`

	// NextHeader contains protocol-specific fields needed to craft the packet
	// (e.g., UDP ports, ICMP words). Should match Protocol.
	NextHeader NextHeader `json:"next_header"`
}

// Agent identifies the vantage point (probe sender / measurement agent).
type Agent struct {
	// AgentID is the unique identifier of the agent.
	AgentID string `json:"agent_id"`
}

// Info describes a single probe/response observation at a given TTL hop. It records addressing, payload size, and
// send/receive timestamps for RTT computation and timing analysis.
type Info struct {
	// ProbeTTL is the ProbeTTL/hop-limit used for the probe that produced this observation.
	ProbeTTL uint8 `json:"probe_ttl"`

	// ReplyAddress is the IP address observed in the received reply (e.g., router interface address).
	ReplyAddress net.IP `json:"reply_address"`

	// ProbePayloadSize is the probe payload size (bytes). Keep in mind uint8 caps at 255.
	ProbePayloadSize uint8 `json:"probe_payload_size"`

	// SentTimestamp is when the probe packet was sent.
	SentTimestamp time.Time `json:"sent_timestamp"`

	// ReceivedTimestamp is when the corresponding reply packet was received.
	ReceivedTimestamp time.Time `json:"received_timestamp"`
}

// ForwardingInfoElement is a measurement record produced by an agent that captures forwarding behavior between a "near"
// TTL and a "far" TTL for the same (source, destination, protocol, IP version) tuple.
//
// NearInfo/FarInfo typically represent observations at adjacent or related TTLs, enabling inference about forwarding
// changes, load balancing, or hop behavior.
type ForwardingInfoElement struct {
	// Agent identifies the vantage point that produced this record.
	Agent Agent `json:"agent"`

	// IPVersion indicates whether the observation is IPv4 or IPv6.
	IPVersion IPVersion `json:"ip_version"`

	// Protocol indicates the probing protocol used (ICMP/UDP/ICMPv6).
	Protocol Protocol `json:"protocol"`

	// SourceAddress is the probe source IP address as seen/used by the agent.
	SourceAddress net.IP `json:"source_address"`

	// DestinationAddress is the probe destination IP address.
	DestinationAddress net.IP `json:"destination_address"`

	// NearInfo is the observation corresponding to the "near" TTL/hop-limit.
	NearInfo Info `json:"near_info"`

	// FarInfo is the observation corresponding to the "far" TTL/hop-limit.
	FarInfo Info `json:"far_info"`

	// ProductionTimestamp is when this record was produced/serialized by the agent/system.
	ProductionTimestamp time.Time `json:"production_timestamp"`
}

// SystemStatus summarizes current orchestration state relevant to probing load.
type SystemStatus struct {
	// GlobalProbingRatePSPA is the target probing rate in probes per second per agent (PSPA). Example: 50 means each
	// active agent should probe at ~50 probes/second and generator should generate 25 PDs/second per each agent since
	// one directive causes two probes.
	GlobalProbingRatePSPA uint `json:"global_probing_rate_pspa"`

	// ActiveAgentIDs is the list of agent identifiers currently considered active and eligible
	// to receive probing directives.
	ActiveAgentIDs []string `json:"active_agent_ids"`
}
