// Copyright (c) 2025 Sorbonne Université
// SPDX-License-Identifier: MIT

// Package model provides idiomatic Go types mirroring the wire/v2 proto
// schema, so wire-layer artifacts (protoc-gen-go's Id/Ns field-name
// mangling, unit-less int64s, string addresses) don't leak into
// hand-written logic that threads these types deeply (e.g.
// retina-orchestrator's PD diff mechanism, ring buffer, and scheduler).
//
// Field names match retina-orchestrator's existing v1 API. TTLs are
// uint8, addresses are net.IP, timestamps are time.Time — matching what
// these values actually are, not the wider/looser types proto3 forces
// at the wire layer. FromProto rejects malformed wire values (TTL
// overflow, an unparseable IP, an invalid or absent timestamp) rather
// than silently truncating or normalizing them; ToProto is the
// symmetric check in the other direction, validating required fields
// before serializing. A nil argument passed directly to a FromProto
// function is caller misuse; a required nested field being absent
// inside an otherwise valid message (e.g. ForwardingInfoElement.Agent)
// is a distinct, legitimate validation failure, not misuse.
//
// Only the types actually threaded deeply are covered: ProbingDirective
// and ForwardingInfoElement, plus their dependencies (Agent, Info).
// NextHeader is left as the native wire.NextHeader type — its fields
// don't have any of the above problems.
package model

import (
	"fmt"
	"math"
	"net"
	"time"

	wire "github.com/dioptra-io/retina-commons/wire/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ipToWireString converts a net.IP to its wire string form. Returns ""
// for a nil or zero-length IP rather than relying on net.IP.String(),
// which returns the literal text "<nil>" for both — a zero-length,
// non-nil net.IP is a distinct possible value in Go, not just nil.
func ipToWireString(ip net.IP) string {
	if len(ip) == 0 {
		return ""
	}
	return ip.String()
}

// normalizeIP collapses an IPv4 address to its own freshly copied
// 4-byte form. net.ParseIP always returns IPv4 addresses in 16-byte
// IPv4-mapped form, which can silently break equality checks, map
// keys, or hashing against a 4-byte net.IPv4(...) value constructed
// elsewhere. Callers comparing IPs should still prefer net.IP.Equal
// over == or reflect.DeepEqual. The copy guards against To4()'s
// subslice aliasing the original's backing array.
func normalizeIP(ip net.IP) net.IP {
	if ip4 := ip.To4(); ip4 != nil {
		return append(net.IP(nil), ip4...)
	}
	return ip
}

// validateIP checks a model net.IP value before it's serialized —
// the ToProto-side symmetric check to parseRequiredIP/parseOptionalIP.
// Unlike those, there's no string to parse here: the model value may
// have been constructed directly rather than produced by FromProto, so
// its length/shape needs validating on its own terms.
func validateIP(ip net.IP, required bool, fieldName string) error {
	if len(ip) == 0 {
		if required {
			return fmt.Errorf("%s: required but empty", fieldName)
		}
		return nil
	}
	if ip.To4() == nil && ip.To16() == nil {
		return fmt.Errorf("%s: invalid IP", fieldName)
	}
	return nil
}

// parseRequiredIP parses a wire address string that must always be
// present (a directive or FIE always has one). An empty string is
// treated as invalid, same as an unparseable one.
func parseRequiredIP(s, fieldName string) (net.IP, error) {
	if s == "" {
		return nil, fmt.Errorf("%s: empty", fieldName)
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("%s: %q is not a valid IP address", fieldName, s)
	}
	return normalizeIP(ip), nil
}

// parseOptionalIP parses a wire address string that may legitimately be
// absent (e.g. Info.ReplyAddress on a probe with no reply). An empty
// string converts to a nil net.IP without error; a non-empty but
// unparseable string is still treated as invalid.
func parseOptionalIP(s, fieldName string) (net.IP, error) {
	if s == "" {
		return nil, nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("%s: %q is not a valid IP address", fieldName, s)
	}
	return normalizeIP(ip), nil
}

// timestampFromProto converts a google.protobuf.Timestamp to time.Time
// (always UTC — location isn't preserved on the wire, so compare with
// Time.Equal, not ==). AsTime() alone is best-effort and can silently
// normalize an invalid timestamp rather than rejecting it, so
// CheckValid is called first; nil is also rejected explicitly, since
// every timestamp field in this schema is always set when its
// containing message exists. Note: time.Time{} (year 1) is itself a
// valid protobuf timestamp, so a caller-constructed zero-value model
// field is indistinguishable from a real 0001-01-01 timestamp — accepted
// here since timestamps are required, not optional, in this schema.
func timestampFromProto(ts *timestamppb.Timestamp, fieldName string) (time.Time, error) {
	if ts == nil {
		return time.Time{}, fmt.Errorf("%s: absent", fieldName)
	}
	if err := ts.CheckValid(); err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", fieldName, err)
	}
	return ts.AsTime(), nil
}

// timestampToProto is the ToProto-side symmetric check: it rejects a
// time.Time that would produce an invalid google.protobuf.Timestamp
// (out of the representable range) rather than emitting a wire value
// FromProto would then reject on the way back in.
func timestampToProto(t time.Time, fieldName string) (*timestamppb.Timestamp, error) {
	ts := timestamppb.New(t)
	if err := ts.CheckValid(); err != nil {
		return nil, fmt.Errorf("%s: %w", fieldName, err)
	}
	return ts, nil
}

// Agent mirrors wire.Agent, with AgentId renamed to the idiomatic ID.
// Always required: a nil wire Agent is rejected rather than silently
// canonicalized to a zero-value Agent{}, which would otherwise
// re-serialize as a non-nil, empty Agent on the way back out.
type Agent struct {
	ID string
}

func AgentFromProto(a *wire.Agent) (Agent, error) {
	if a == nil {
		return Agent{}, fmt.Errorf("agent: absent")
	}
	id := a.GetAgentId()
	if id == "" {
		return Agent{}, fmt.Errorf("agent_id: empty")
	}
	return Agent{ID: id}, nil
}

func (a Agent) ToProto() *wire.Agent {
	return &wire.Agent{AgentId: a.ID}
}

// Info mirrors wire.Info, with google.protobuf.Timestamp converted to
// time.Time, ProbeTTL narrowed to uint8, and ReplyAddress parsed to
// net.IP (nil represents a legitimately absent reply — see package doc).
type Info struct {
	ProbeTTL          uint8
	ReplyAddress      net.IP
	SentTimestamp     time.Time
	ReceivedTimestamp time.Time
}

// InfoFromProto converts a wire.Info to a model Info. Returns an error
// if ProbeTtl exceeds what fits in a uint8, ReplyAddress is non-empty
// but not a valid IP, either timestamp is absent or invalid, or i is
// nil (see package doc).
func InfoFromProto(i *wire.Info) (Info, error) {
	if i == nil {
		return Info{}, fmt.Errorf("info: absent")
	}
	rawTTL := i.GetProbeTtl()
	if rawTTL > math.MaxUint8 {
		return Info{}, fmt.Errorf("probe_ttl: value %d exceeds uint8 range", rawTTL)
	}
	replyAddr, err := parseOptionalIP(i.GetReplyAddress(), "reply_address")
	if err != nil {
		return Info{}, err
	}
	sentAt, err := timestampFromProto(i.GetSentTimestamp(), "sent_timestamp")
	if err != nil {
		return Info{}, err
	}
	receivedAt, err := timestampFromProto(i.GetReceivedTimestamp(), "received_timestamp")
	if err != nil {
		return Info{}, err
	}
	return Info{
		ProbeTTL:          uint8(rawTTL),
		ReplyAddress:      replyAddr,
		SentTimestamp:     sentAt,
		ReceivedTimestamp: receivedAt,
	}, nil
}

func (i Info) ToProto() (*wire.Info, error) {
	if err := validateIP(i.ReplyAddress, false, "reply_address"); err != nil {
		return nil, err
	}
	sentTS, err := timestampToProto(i.SentTimestamp, "sent_timestamp")
	if err != nil {
		return nil, err
	}
	receivedTS, err := timestampToProto(i.ReceivedTimestamp, "received_timestamp")
	if err != nil {
		return nil, err
	}
	return &wire.Info{
		ProbeTtl:          uint32(i.ProbeTTL), // widening, always safe
		ReplyAddress:      ipToWireString(i.ReplyAddress),
		SentTimestamp:     sentTS,
		ReceivedTimestamp: receivedTS,
	}, nil
}

// ProbingDirective mirrors wire.ProbingDirective, with ProbingDirectiveId
// and AgentId renamed to their idiomatic ID forms, NearTTL narrowed to
// uint8, and DestinationAddress parsed to net.IP (always required — see
// package doc). NextHeader is left as the native proto type (see
// package doc).
type ProbingDirective struct {
	ProbingDirectiveID uint64
	IPVersion          wire.IPVersion
	Protocol           wire.Protocol
	AgentID            string
	DestinationAddress net.IP
	NearTTL            uint8
	NextHeader         *wire.NextHeader
}

// ProbingDirectiveFromProto converts a wire.ProbingDirective to a model
// ProbingDirective. Returns an error if NearTtl exceeds what fits in a
// uint8, DestinationAddress is missing or not a valid IP, or pd is nil
// (see package doc).
func ProbingDirectiveFromProto(pd *wire.ProbingDirective) (ProbingDirective, error) {
	if pd == nil {
		return ProbingDirective{}, fmt.Errorf("probing_directive: absent")
	}
	rawTTL := pd.GetNearTtl()
	if rawTTL > math.MaxUint8 {
		return ProbingDirective{}, fmt.Errorf("near_ttl: value %d exceeds uint8 range", rawTTL)
	}
	destAddr, err := parseRequiredIP(pd.GetDestinationAddress(), "destination_address")
	if err != nil {
		return ProbingDirective{}, err
	}
	return ProbingDirective{
		ProbingDirectiveID: pd.GetProbingDirectiveId(),
		IPVersion:          pd.GetIpVersion(),
		Protocol:           pd.GetProtocol(),
		AgentID:            pd.GetAgentId(),
		DestinationAddress: destAddr,
		NearTTL:            uint8(rawTTL),
		NextHeader:         pd.GetNextHeader(),
	}, nil
}

// ToProto validates that DestinationAddress is set and well-formed
// (required — see package doc) before serializing, so a malformed
// model value can't silently produce a wire message that FromProto
// would itself reject.
func (pd ProbingDirective) ToProto() (*wire.ProbingDirective, error) {
	if err := validateIP(pd.DestinationAddress, true, "destination_address"); err != nil {
		return nil, err
	}
	return &wire.ProbingDirective{
		ProbingDirectiveId: pd.ProbingDirectiveID,
		IpVersion:          pd.IPVersion,
		Protocol:           pd.Protocol,
		AgentId:            pd.AgentID,
		DestinationAddress: ipToWireString(pd.DestinationAddress),
		NearTtl:            uint32(pd.NearTTL), // widening, always safe
		NextHeader:         pd.NextHeader,
	}, nil
}

// ForwardingInfoElement mirrors wire.ForwardingInfoElement (see package
// doc). NearInfo/FarInfo stay nilable pointers, preserving proto3's
// optional-absent-means-timeout semantics. SourceAddress is likewise nil
// when unavailable — a probe that gets no reply at all (both near and
// far time out) has no source address to report, unlike
// DestinationAddress, which is always known and required.
type ForwardingInfoElement struct {
	Agent               Agent
	ProbingDirectiveID  uint64
	IPVersion           wire.IPVersion
	Protocol            wire.Protocol
	SourceAddress       net.IP // nil if unavailable (e.g. both probes timed out)
	DestinationAddress  net.IP
	NearInfo            *Info // nil represents a timeout, same as proto3 optional-absent
	FarInfo             *Info
	ProductionTimestamp time.Time
}

// ForwardingInfoElementFromProto converts a wire.ForwardingInfoElement to
// a model ForwardingInfoElement. Returns an error if Agent or
// DestinationAddress is missing/invalid, ProductionTimestamp is absent
// or invalid, either NearInfo or FarInfo fails to convert, or fie is nil
// (see package doc). SourceAddress is optional — an empty wire string
// converts to a nil net.IP without error.
func ForwardingInfoElementFromProto(fie *wire.ForwardingInfoElement) (ForwardingInfoElement, error) {
	if fie == nil {
		return ForwardingInfoElement{}, fmt.Errorf("forwarding_info_element: absent")
	}
	agent, err := AgentFromProto(fie.GetAgent())
	if err != nil {
		return ForwardingInfoElement{}, fmt.Errorf("agent: %w", err)
	}
	srcAddr, err := parseOptionalIP(fie.GetSourceAddress(), "source_address")
	if err != nil {
		return ForwardingInfoElement{}, err
	}
	destAddr, err := parseRequiredIP(fie.GetDestinationAddress(), "destination_address")
	if err != nil {
		return ForwardingInfoElement{}, err
	}
	producedAt, err := timestampFromProto(fie.GetProductionTimestamp(), "production_timestamp")
	if err != nil {
		return ForwardingInfoElement{}, err
	}
	out := ForwardingInfoElement{
		Agent:               agent,
		ProbingDirectiveID:  fie.GetProbingDirectiveId(),
		IPVersion:           fie.GetIpVersion(),
		Protocol:            fie.GetProtocol(),
		SourceAddress:       srcAddr,
		DestinationAddress:  destAddr,
		ProductionTimestamp: producedAt,
	}
	if info := fie.GetNearInfo(); info != nil {
		converted, err := InfoFromProto(info)
		if err != nil {
			return ForwardingInfoElement{}, fmt.Errorf("near_info: %w", err)
		}
		out.NearInfo = &converted
	}
	if info := fie.GetFarInfo(); info != nil {
		converted, err := InfoFromProto(info)
		if err != nil {
			return ForwardingInfoElement{}, fmt.Errorf("far_info: %w", err)
		}
		out.FarInfo = &converted
	}
	return out, nil
}

// ToProto validates DestinationAddress and ProductionTimestamp (required)
// and, if SourceAddress is present, that it's well-formed — SourceAddress
// itself is optional (a probe that gets no reply at all has no source
// address to report; see package doc) — before serializing, so a
// malformed model value can't silently produce a wire message that
// FromProto would itself reject.
func (fie ForwardingInfoElement) ToProto() (*wire.ForwardingInfoElement, error) {
	if err := validateIP(fie.SourceAddress, false, "source_address"); err != nil {
		return nil, err
	}
	if err := validateIP(fie.DestinationAddress, true, "destination_address"); err != nil {
		return nil, err
	}
	productionTS, err := timestampToProto(fie.ProductionTimestamp, "production_timestamp")
	if err != nil {
		return nil, err
	}
	out := &wire.ForwardingInfoElement{
		Agent:               fie.Agent.ToProto(),
		ProbingDirectiveId:  fie.ProbingDirectiveID,
		IpVersion:           fie.IPVersion,
		Protocol:            fie.Protocol,
		SourceAddress:       ipToWireString(fie.SourceAddress),
		DestinationAddress:  ipToWireString(fie.DestinationAddress),
		ProductionTimestamp: productionTS,
	}
	if fie.NearInfo != nil {
		near, err := fie.NearInfo.ToProto()
		if err != nil {
			return nil, fmt.Errorf("near_info: %w", err)
		}
		out.NearInfo = near
	}
	if fie.FarInfo != nil {
		far, err := fie.FarInfo.ToProto()
		if err != nil {
			return nil, fmt.Errorf("far_info: %w", err)
		}
		out.FarInfo = far
	}
	return out, nil
}
