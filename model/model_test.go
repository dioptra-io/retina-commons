// Copyright (c) 2025 Sorbonne Université
// SPDX-License-Identifier: MIT

package model

import (
	"net"
	"testing"
	"time"

	wire "github.com/dioptra-io/retina-commons/wire/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mustParseIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("test setup: %q is not a valid IP", s)
	}
	return ip
}

// -- Agent --------------------------------------------------------------

func TestAgentRoundTrip(t *testing.T) {
	want := Agent{ID: "agent-1"}
	proto := want.ToProto()
	got, err := AgentFromProto(proto)
	if err != nil {
		t.Fatalf("AgentFromProto: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAgentFromProtoRejectsNil(t *testing.T) {
	if _, err := AgentFromProto(nil); err == nil {
		t.Error("expected error for nil wire.Agent, got nil")
	}
}

func TestAgentFromProtoRejectsEmptyID(t *testing.T) {
	if _, err := AgentFromProto(&wire.Agent{AgentId: ""}); err == nil {
		t.Error("expected error for empty agent_id, got nil")
	}
}

// -- Info -----------------------------------------------------------------

func TestInfoRoundTripWithReply(t *testing.T) {
	want := Info{
		ProbeTTL:          64,
		ReplyAddress:      mustParseIP(t, "192.0.2.1"),
		SentTimestamp:     time.Now().UTC().Truncate(time.Second),
		ReceivedTimestamp: time.Now().UTC().Truncate(time.Second),
	}
	proto, err := want.ToProto()
	if err != nil {
		t.Fatalf("ToProto: %v", err)
	}
	got, err := InfoFromProto(proto)
	if err != nil {
		t.Fatalf("InfoFromProto: %v", err)
	}
	if got.ProbeTTL != want.ProbeTTL {
		t.Errorf("ProbeTTL: got %d, want %d", got.ProbeTTL, want.ProbeTTL)
	}
	if !got.ReplyAddress.Equal(want.ReplyAddress) {
		t.Errorf("ReplyAddress: got %v, want %v", got.ReplyAddress, want.ReplyAddress)
	}
	if !got.SentTimestamp.Equal(want.SentTimestamp) {
		t.Errorf("SentTimestamp: got %v, want %v", got.SentTimestamp, want.SentTimestamp)
	}
	if !got.ReceivedTimestamp.Equal(want.ReceivedTimestamp) {
		t.Errorf("ReceivedTimestamp: got %v, want %v", got.ReceivedTimestamp, want.ReceivedTimestamp)
	}
}

func TestInfoFromProtoAcceptsAbsentReplyAddress(t *testing.T) {
	proto := &wire.Info{
		ProbeTtl:          64,
		ReplyAddress:      "", // no reply — legitimate, not an error
		SentTimestamp:     timestamppb.New(time.Now()),
		ReceivedTimestamp: timestamppb.New(time.Now()),
	}
	got, err := InfoFromProto(proto)
	if err != nil {
		t.Fatalf("InfoFromProto: unexpected error: %v", err)
	}
	if got.ReplyAddress != nil {
		t.Errorf("ReplyAddress: got %v, want nil", got.ReplyAddress)
	}
}

func TestInfoFromProtoRejectsTTLOverflow(t *testing.T) {
	proto := &wire.Info{
		ProbeTtl:          300, // > uint8 max
		SentTimestamp:     timestamppb.New(time.Now()),
		ReceivedTimestamp: timestamppb.New(time.Now()),
	}
	if _, err := InfoFromProto(proto); err == nil {
		t.Error("expected error for probe_ttl > 255, got nil")
	}
}

func TestInfoFromProtoRejectsInvalidReplyAddress(t *testing.T) {
	proto := &wire.Info{
		ProbeTtl:          64,
		ReplyAddress:      "not-an-ip",
		SentTimestamp:     timestamppb.New(time.Now()),
		ReceivedTimestamp: timestamppb.New(time.Now()),
	}
	if _, err := InfoFromProto(proto); err == nil {
		t.Error("expected error for invalid reply_address, got nil")
	}
}

func TestInfoFromProtoRejectsNilTimestamp(t *testing.T) {
	proto := &wire.Info{
		ProbeTtl:          64,
		SentTimestamp:     nil, // absent — always required in this schema
		ReceivedTimestamp: timestamppb.New(time.Now()),
	}
	if _, err := InfoFromProto(proto); err == nil {
		t.Error("expected error for nil sent_timestamp, got nil")
	}
}

func TestInfoFromProtoRejectsInvalidTimestamp(t *testing.T) {
	proto := &wire.Info{
		ProbeTtl:          64,
		SentTimestamp:     &timestamppb.Timestamp{Seconds: 0, Nanos: -1}, // invalid: negative nanos
		ReceivedTimestamp: timestamppb.New(time.Now()),
	}
	if _, err := InfoFromProto(proto); err == nil {
		t.Error("expected error for invalid sent_timestamp, got nil")
	}
}

func TestInfoFromProtoRejectsNil(t *testing.T) {
	if _, err := InfoFromProto(nil); err == nil {
		t.Error("expected error for nil wire.Info, got nil")
	}
}

// -- ProbingDirective -------------------------------------------------------

func TestProbingDirectiveRoundTrip(t *testing.T) {
	want := ProbingDirective{
		ProbingDirectiveID: 42,
		IPVersion:          wire.IPVersion_IP_VERSION_IPV4,
		Protocol:           wire.Protocol_PROTOCOL_ICMP,
		AgentID:            "agent-1",
		DestinationAddress: mustParseIP(t, "192.0.2.1"),
		NearTTL:            10,
	}
	proto, err := want.ToProto()
	if err != nil {
		t.Fatalf("ToProto: %v", err)
	}
	got, err := ProbingDirectiveFromProto(proto)
	if err != nil {
		t.Fatalf("ProbingDirectiveFromProto: %v", err)
	}
	if got.ProbingDirectiveID != want.ProbingDirectiveID {
		t.Errorf("ProbingDirectiveID: got %d, want %d", got.ProbingDirectiveID, want.ProbingDirectiveID)
	}
	if got.NearTTL != want.NearTTL {
		t.Errorf("NearTTL: got %d, want %d", got.NearTTL, want.NearTTL)
	}
	if !got.DestinationAddress.Equal(want.DestinationAddress) {
		t.Errorf("DestinationAddress: got %v, want %v", got.DestinationAddress, want.DestinationAddress)
	}
}

func TestProbingDirectiveFromProtoRejectsMissingDestination(t *testing.T) {
	proto := &wire.ProbingDirective{
		ProbingDirectiveId: 1,
		AgentId:            "agent-1",
		DestinationAddress: "",
		NearTtl:            10,
	}
	if _, err := ProbingDirectiveFromProto(proto); err == nil {
		t.Error("expected error for missing destination_address, got nil")
	}
}

func TestProbingDirectiveFromProtoRejectsTTLOverflow(t *testing.T) {
	proto := &wire.ProbingDirective{
		ProbingDirectiveId: 1,
		AgentId:            "agent-1",
		DestinationAddress: "192.0.2.1",
		NearTtl:            300,
	}
	if _, err := ProbingDirectiveFromProto(proto); err == nil {
		t.Error("expected error for near_ttl > 255, got nil")
	}
}

func TestProbingDirectiveToProtoRejectsNilDestination(t *testing.T) {
	pd := ProbingDirective{ProbingDirectiveID: 1, AgentID: "agent-1", NearTTL: 10}
	if _, err := pd.ToProto(); err == nil {
		t.Error("expected error for nil DestinationAddress, got nil")
	}
}

func TestProbingDirectiveFromProtoRejectsNil(t *testing.T) {
	if _, err := ProbingDirectiveFromProto(nil); err == nil {
		t.Error("expected error for nil wire.ProbingDirective, got nil")
	}
}

// -- ForwardingInfoElement --------------------------------------------------

func TestForwardingInfoElementPreservesInfoPresence(t *testing.T) {
	want := ForwardingInfoElement{
		Agent:               Agent{ID: "agent-1"},
		ProbingDirectiveID:  1,
		SourceAddress:       mustParseIP(t, "192.0.2.1"),
		DestinationAddress:  mustParseIP(t, "192.0.2.2"),
		ProductionTimestamp: time.Now().UTC().Truncate(time.Second),
		NearInfo: &Info{
			ProbeTTL:          10,
			SentTimestamp:     time.Now().UTC().Truncate(time.Second),
			ReceivedTimestamp: time.Now().UTC().Truncate(time.Second),
		},
		FarInfo: nil, // simulates a timeout on the far probe
	}
	proto, err := want.ToProto()
	if err != nil {
		t.Fatalf("ToProto: %v", err)
	}
	got, err := ForwardingInfoElementFromProto(proto)
	if err != nil {
		t.Fatalf("ForwardingInfoElementFromProto: %v", err)
	}
	if got.NearInfo == nil {
		t.Error("NearInfo: got nil, want present")
	}
	if got.FarInfo != nil {
		t.Errorf("FarInfo: got %+v, want nil (timeout)", got.FarInfo)
	}
}

func TestForwardingInfoElementFromProtoRejectsMissingAgent(t *testing.T) {
	proto := &wire.ForwardingInfoElement{
		Agent:               nil,
		SourceAddress:       "192.0.2.1",
		DestinationAddress:  "192.0.2.2",
		ProductionTimestamp: timestamppb.New(time.Now()),
	}
	if _, err := ForwardingInfoElementFromProto(proto); err == nil {
		t.Error("expected error for missing agent, got nil")
	}
}

func TestForwardingInfoElementFromProtoRejectsMissingDestination(t *testing.T) {
	missingDest := &wire.ForwardingInfoElement{
		Agent:               &wire.Agent{AgentId: "agent-1"},
		SourceAddress:       "192.0.2.1",
		DestinationAddress:  "",
		ProductionTimestamp: timestamppb.New(time.Now()),
	}
	if _, err := ForwardingInfoElementFromProto(missingDest); err == nil {
		t.Error("expected error for missing destination_address, got nil")
	}
}

// TestForwardingInfoElementFromProtoAcceptsMissingSource confirms
// SourceAddress is optional, unlike DestinationAddress — a probe that
// gets no reply at all (both near and far time out) has no source
// address to report; see ForwardingInfoElement's doc comment.
func TestForwardingInfoElementFromProtoAcceptsMissingSource(t *testing.T) {
	proto := &wire.ForwardingInfoElement{
		Agent:               &wire.Agent{AgentId: "agent-1"},
		SourceAddress:       "",
		DestinationAddress:  "192.0.2.2",
		ProductionTimestamp: timestamppb.New(time.Now()),
	}
	got, err := ForwardingInfoElementFromProto(proto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SourceAddress != nil {
		t.Errorf("SourceAddress: got %v, want nil", got.SourceAddress)
	}
}

func TestForwardingInfoElementFromProtoRejectsNil(t *testing.T) {
	if _, err := ForwardingInfoElementFromProto(nil); err == nil {
		t.Error("expected error for nil wire.ForwardingInfoElement, got nil")
	}
}

// TestForwardingInfoElementToProtoRejectsNilDestination confirms
// DestinationAddress is still required — unlike SourceAddress, which is
// nil in this fixture too but no longer causes an error on its own (see
// TestForwardingInfoElementToProtoRejectsInvalidDestination for the
// source-valid/destination-nil case, which isolates this more precisely).
func TestForwardingInfoElementToProtoRejectsNilDestination(t *testing.T) {
	fie := ForwardingInfoElement{
		Agent:               Agent{ID: "agent-1"},
		ProductionTimestamp: time.Now(),
	}
	if _, err := fie.ToProto(); err == nil {
		t.Error("expected error for nil DestinationAddress, got nil")
	}
}

// -- IP round trips (v4/v6, including normalization) -----------------------

func TestIPRoundTripIPv4(t *testing.T) {
	ip, err := parseRequiredIP("192.0.2.1", "test")
	if err != nil {
		t.Fatalf("parseRequiredIP: %v", err)
	}
	if got := ipToWireString(ip); got != "192.0.2.1" {
		t.Errorf("got %q, want %q", got, "192.0.2.1")
	}
}

func TestIPRoundTripIPv6(t *testing.T) {
	const addr = "2001:db8::1"
	ip, err := parseRequiredIP(addr, "test")
	if err != nil {
		t.Fatalf("parseRequiredIP: %v", err)
	}
	if got := ipToWireString(ip); got != addr {
		t.Errorf("got %q, want %q", got, addr)
	}
}

func TestZeroLengthIPSerializesAsEmpty(t *testing.T) {
	if got := ipToWireString(net.IP{}); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestValidateIPRejectsGarbageLength(t *testing.T) {
	garbage := net.IP{1, 2, 3} // neither 4 nor 16 bytes
	if err := validateIP(garbage, true, "test"); err == nil {
		t.Error("expected error for malformed-length IP, got nil")
	}
}

// -- Additional coverage: Info error branches --------------------------

func TestInfoFromProtoRejectsInvalidReceivedTimestamp(t *testing.T) {
	proto := &wire.Info{
		ProbeTtl:          64,
		SentTimestamp:     timestamppb.New(time.Now()),
		ReceivedTimestamp: nil, // absent, mirrors the SentTimestamp case already tested
	}
	if _, err := InfoFromProto(proto); err == nil {
		t.Error("expected error for nil received_timestamp, got nil")
	}
}

func TestInfoToProtoRejectsInvalidReplyAddress(t *testing.T) {
	info := Info{
		ReplyAddress:      net.IP{1, 2, 3}, // neither 4 nor 16 bytes
		SentTimestamp:     time.Now(),
		ReceivedTimestamp: time.Now(),
	}
	if _, err := info.ToProto(); err == nil {
		t.Error("expected error for malformed-length ReplyAddress, got nil")
	}
}

func farFutureTime(t *testing.T) time.Time {
	t.Helper()
	// Year 300000 is far outside google.protobuf.Timestamp's representable
	// range (0001-01-01 through 9999-12-31), so CheckValid must reject it.
	return time.Date(300000, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestInfoToProtoRejectsOutOfRangeTimestamp(t *testing.T) {
	info := Info{
		SentTimestamp:     farFutureTime(t),
		ReceivedTimestamp: time.Now(),
	}
	if _, err := info.ToProto(); err == nil {
		t.Error("expected error for out-of-range SentTimestamp, got nil")
	}
}

// -- Additional coverage: ProbingDirective error branches ----------------

func TestProbingDirectiveFromProtoRejectsMalformedDestination(t *testing.T) {
	proto := &wire.ProbingDirective{
		ProbingDirectiveId: 1,
		AgentId:            "agent-1",
		DestinationAddress: "not-an-ip", // non-empty but unparseable
		NearTtl:            10,
	}
	if _, err := ProbingDirectiveFromProto(proto); err == nil {
		t.Error("expected error for malformed destination_address, got nil")
	}
}

// -- Additional coverage: ForwardingInfoElement error branches -----------

func validFIEProto(t *testing.T) *wire.ForwardingInfoElement {
	t.Helper()
	return &wire.ForwardingInfoElement{
		Agent:               &wire.Agent{AgentId: "agent-1"},
		SourceAddress:       "192.0.2.1",
		DestinationAddress:  "192.0.2.2",
		ProductionTimestamp: timestamppb.New(time.Now()),
	}
}

func TestForwardingInfoElementFromProtoRejectsMissingProductionTimestamp(t *testing.T) {
	proto := validFIEProto(t)
	proto.ProductionTimestamp = nil
	if _, err := ForwardingInfoElementFromProto(proto); err == nil {
		t.Error("expected error for missing production_timestamp, got nil")
	}
}

func TestForwardingInfoElementFromProtoRejectsInvalidNearInfo(t *testing.T) {
	proto := validFIEProto(t)
	proto.NearInfo = &wire.Info{ProbeTtl: 300} // present, but ProbeTtl overflows uint8
	if _, err := ForwardingInfoElementFromProto(proto); err == nil {
		t.Error("expected error for invalid near_info, got nil")
	}
}

func TestForwardingInfoElementFromProtoRejectsInvalidFarInfo(t *testing.T) {
	proto := validFIEProto(t)
	proto.FarInfo = &wire.Info{ProbeTtl: 300}
	if _, err := ForwardingInfoElementFromProto(proto); err == nil {
		t.Error("expected error for invalid far_info, got nil")
	}
}

func validFIE(t *testing.T) ForwardingInfoElement {
	t.Helper()
	return ForwardingInfoElement{
		Agent:               Agent{ID: "agent-1"},
		SourceAddress:       mustParseIP(t, "192.0.2.1"),
		DestinationAddress:  mustParseIP(t, "192.0.2.2"),
		ProductionTimestamp: time.Now(),
	}
}

func TestForwardingInfoElementToProtoRejectsInvalidDestination(t *testing.T) {
	fie := validFIE(t)
	fie.DestinationAddress = nil // SourceAddress stays valid, so this exercises the second validateIP check
	if _, err := fie.ToProto(); err == nil {
		t.Error("expected error for nil DestinationAddress, got nil")
	}
}

func TestForwardingInfoElementToProtoRejectsOutOfRangeProductionTimestamp(t *testing.T) {
	fie := validFIE(t)
	fie.ProductionTimestamp = farFutureTime(t)
	if _, err := fie.ToProto(); err == nil {
		t.Error("expected error for out-of-range ProductionTimestamp, got nil")
	}
}

func TestForwardingInfoElementToProtoRejectsInvalidNearInfo(t *testing.T) {
	fie := validFIE(t)
	fie.NearInfo = &Info{ReplyAddress: net.IP{1, 2, 3}} // present, but malformed
	if _, err := fie.ToProto(); err == nil {
		t.Error("expected error for invalid NearInfo, got nil")
	}
}

func TestForwardingInfoElementToProtoRejectsInvalidFarInfo(t *testing.T) {
	fie := validFIE(t)
	fie.FarInfo = &Info{ReplyAddress: net.IP{1, 2, 3}}
	if _, err := fie.ToProto(); err == nil {
		t.Error("expected error for invalid FarInfo, got nil")
	}
}

// TestForwardingInfoElementRoundTripWithBothInfoPresent covers the
// FarInfo success branch in ToProto — the earlier presence test only
// ever set NearInfo, leaving FarInfo's "if fie.FarInfo != nil" branch
// in ToProto unexercised.
func TestForwardingInfoElementRoundTripWithBothInfoPresent(t *testing.T) {
	fie := validFIE(t)
	fie.NearInfo = &Info{ProbeTTL: 10, SentTimestamp: time.Now(), ReceivedTimestamp: time.Now()}
	fie.FarInfo = &Info{ProbeTTL: 11, SentTimestamp: time.Now(), ReceivedTimestamp: time.Now()}

	proto, err := fie.ToProto()
	if err != nil {
		t.Fatalf("ToProto: %v", err)
	}
	got, err := ForwardingInfoElementFromProto(proto)
	if err != nil {
		t.Fatalf("ForwardingInfoElementFromProto: %v", err)
	}
	if got.NearInfo == nil || got.FarInfo == nil {
		t.Errorf("expected both NearInfo and FarInfo present, got NearInfo=%v FarInfo=%v", got.NearInfo, got.FarInfo)
	}
}

func TestInfoToProtoRejectsOutOfRangeReceivedTimestamp(t *testing.T) {
	info := Info{
		SentTimestamp:     time.Now(),
		ReceivedTimestamp: farFutureTime(t),
	}
	if _, err := info.ToProto(); err == nil {
		t.Error("expected error for out-of-range ReceivedTimestamp, got nil")
	}
}

// TestForwardingInfoElementToProtoAcceptsNilSource is the ToProto-side
// mirror of TestForwardingInfoElementFromProtoAcceptsMissingSource.
func TestForwardingInfoElementToProtoAcceptsNilSource(t *testing.T) {
	fie := ForwardingInfoElement{
		Agent:               Agent{ID: "agent-1"},
		SourceAddress:       nil,
		DestinationAddress:  mustParseIP(t, "192.0.2.2"),
		ProductionTimestamp: time.Now(),
	}
	proto, err := fie.ToProto()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proto.SourceAddress != "" {
		t.Errorf("SourceAddress: got %q, want empty string", proto.SourceAddress)
	}
}
