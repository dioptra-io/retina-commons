// Copyright (c) 2025 Dioptra
// SPDX-License-Identifier: MIT
//
// Package api defines the shared types used by retina components for
// network measurement and probing orchestration.
package api

import (
	"net"
	"time"
)

// IPVersion identifies the IP version used for a probe/observation (IPv4 or
// IPv6). It is encoded as the conventional version number (4 or 6).
type IPVersion uint8

const (
	IPv4 IPVersion = 4 // IP version 4
	IPv6 IPVersion = 6 // IP version 6
)

// Protocol identifies the L4 protocol carried by the IP packet (with the
// exception of ICMP and ICMPv6). Values match the IANA IP Protocol Numbers
// (e.g., UDP=17, ICMP=1, ICMPv6=58).
type Protocol uint8

const (
	ICMP   Protocol = 1
	UDP    Protocol = 17
	ICMPv6 Protocol = 58
)

// ICMPNextHeader carries protocol-specific header fields for ICMP probes.
// These raw 16-bit words are interpreted by the prober implementation and may
// encode flow identifiers, checksums, or other metadata depending on the probing
// strategy.
type ICMPNextHeader struct {
	FirstHalfWord  uint16 `json:"first_half_word"`
	SecondHalfWord uint16 `json:"second_half_word"`
}

// UDPNextHeader carries protocol-specific header fields for UDP probes.
type UDPNextHeader struct {
	SourcePort      uint16 `json:"source_port"`
	DestinationPort uint16 `json:"destination_port"`
}

// ICMPv6NextHeader carries protocol-specific header fields for ICMPv6 probes.
// These raw 16-bit words are interpreted by the prober implementation and may
// encode flow identifiers, checksums, or other metadata depending on the probing
// strategy.
type ICMPv6NextHeader struct {
	FirstHalfWord  uint16 `json:"first_half_word"`
	SecondHalfWord uint16 `json:"second_half_word"`
}

// NextHeader contains protocol-specific header fields. Set the field matching
// your Protocol (e.g., UDPNextHeader when Protocol is UDP).
type NextHeader struct {
	ICMPNextHeader   *ICMPNextHeader   `json:"icmp_next_header,omitempty"`
	UDPNextHeader    *UDPNextHeader    `json:"udp_next_header,omitempty"`
	ICMPv6NextHeader *ICMPv6NextHeader `json:"icmpv6_next_header,omitempty"`
}

// ProbingDirective is an instruction for an agent to probe a specific destination
// at a given TTL. The agent sends two probes (near and far, typically NearTTL and
// NearTTL+1) to detect forwarding behavior between adjacent hops.
type ProbingDirective struct {
	// ProbingDirectiveID is assigned by the generator to track this directive.
	ProbingDirectiveID uint64 `json:"probing_directive_id"`

	IPVersion          IPVersion `json:"ip_version"`
	Protocol           Protocol  `json:"protocol"`
	AgentID            string    `json:"agent_id"`
	DestinationAddress net.IP    `json:"destination_address"`

	// NearTTL is the TTL for the first probe. The agent will also probe at NearTTL+1 (far).
	NearTTL uint8 `json:"near_ttl"`

	// NextHeader holds protocol-specific fields (UDP ports, ICMP identifiers, etc.).
	NextHeader NextHeader `json:"next_header"`
}

// Agent identifies the entity executing probes. May be extended with location
// or capability metadata.
type Agent struct {
	AgentID string `json:"agent_id"`
}

// Info captures a single probe/response observation at a given TTL hop,
// including the reply source and timing data for RTT computation.
type Info struct {
	ProbeTTL uint8 `json:"probe_ttl"`

	// ReplyAddress is the IP address in the reply (e.g., router interface address).
	ReplyAddress net.IP `json:"reply_address"`

	SentTimestamp     time.Time `json:"sent_timestamp"`
	ReceivedTimestamp time.Time `json:"received_timestamp"`
}

// ForwardingInfoElement captures probe results at two consecutive TTL values.
// Comparing the near and far observations reveals forwarding behavior such as
// load balancing or path changes.
//
// NearInfo and FarInfo will be nil when no response was received for that probe.
type ForwardingInfoElement struct {
	Agent          Agent  `json:"agent"`
	SequenceNumber uint64 `json:"sequence_number"`

	// ProbingDirectiveID links this result to its originating directive.
	ProbingDirectiveID uint64 `json:"probing_directive_id"`

	IPVersion          IPVersion `json:"ip_version"`
	Protocol           Protocol  `json:"protocol"`
	SourceAddress      net.IP    `json:"source_address,omitempty"`
	DestinationAddress net.IP    `json:"destination_address,omitempty"`

	NearInfo *Info `json:"near_info,omitempty"`
	FarInfo  *Info `json:"far_info,omitempty"`

	// ProductionTimestamp is when this record was created (not when probes were sent).
	ProductionTimestamp time.Time `json:"production_timestamp"`
}

// SystemStatus summarizes current orchestration state for coordinating
// distributed probing across agents.
type SystemStatus struct {
	// GlobalProbingRatePSPA is the target probing rate per agent in probes/second.
	// Note: Each ProbingDirective generates two probes (near + far TTL).
	GlobalProbingRatePSPA uint `json:"global_probing_rate_pspa"`

	// ProbingImpactLimit is the minimum time between probes that would elicit a
	// response from the same IP address (to limit load on individual routers/hosts).
	ProbingImpactLimit time.Duration `json:"probing_impact_limit"`

	// DisallowedDestinationAddresses lists IPs excluded from probing
	// (private addresses, reserved ranges, etc.).
	DisallowedDestinationAddresses []net.IP `json:"disallowed_destination_addresses"`

	// ActiveAgentIDs lists agents currently eligible to receive directives.
	ActiveAgentIDs []string `json:"active_agent_ids"`
}

// AuthRequest is sent by the agent immediately after connecting to authenticate.
type AuthRequest struct {
	AgentID string `json:"agent_id"`
	Secret  string `json:"secret"`
}

// AuthResponse is sent by the orchestrator in response to AuthRequest.
type AuthResponse struct {
	// Authenticated is true if the secret is valid.
	Authenticated bool `json:"authenticated"`

	// Message provides additional context (e.g., error details on failure).
	Message string `json:"message,omitempty"`
}
