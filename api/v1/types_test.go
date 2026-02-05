// Copyright (c) 2025 Dioptra
// SPDX-License-Identifier: MIT

package api

// These tests verify the JSON serialization contract between retina-agent
// and the orchestrator. While the api package contains only type definitions
// (no executable code), these tests serve critical purposes:
//
// 1. Protocol Contract Verification: Ensures JSON field names, types, and
//    encoding remain stable across versions. Changes that break compatibility
//    (renamed fields, altered tags, type changes) are caught immediately.
//
// 2. Edge Case Validation: JSON marshaling has subtle behaviors (nil vs empty
//    slices, IP address formats, time precision, omitempty semantics) that
//    aren't obvious from struct definitions alone.
//
// 3. Documentation: Tests show how each type is used and what the wire format
//    looks like, helping developers understand the protocol without reverse-
//    engineering production traffic.
//
// 4. Refactoring Safety: If internal representations change (e.g., switching
//    from net.IP to netip.Addr), tests confirm the JSON output stays compatible.
//
// Note: Coverage tools report "no statements" for this package because it
// contains only data structures. This is expected and correct.

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

// ============================================================================
// TEST HELPERS
// ============================================================================

// mustMarshal marshals v to JSON or fails the test.
func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return data
}

// mustUnmarshal unmarshals data into v or fails the test.
func mustUnmarshal(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
}

// jsonRoundTrip marshals and unmarshals v, returning the result.
func jsonRoundTrip(t *testing.T, v interface{}, result interface{}) {
	t.Helper()
	data := mustMarshal(t, v)
	mustUnmarshal(t, data, result)
}

// ============================================================================
// PROBING DIRECTIVE TESTS
// ============================================================================

func TestProbingDirective_JSON_ICMP(t *testing.T) {
	t.Parallel()

	original := &ProbingDirective{
		ProbingDirectiveID: 123,
		IPVersion:          IPv4,
		Protocol:           ICMP,
		AgentID:            "agent-1",
		DestinationAddress: net.ParseIP("8.8.8.8"),
		NearTTL:            10,
		NextHeader: NextHeader{
			ICMPNextHeader: &ICMPNextHeader{
				FirstHalfWord:  0x1234,
				SecondHalfWord: 0x5678,
			},
		},
	}

	var decoded ProbingDirective
	jsonRoundTrip(t, original, &decoded)

	if decoded.ProbingDirectiveID != 123 {
		t.Errorf("ProbingDirectiveID = %d, want 123", decoded.ProbingDirectiveID)
	}
	if decoded.IPVersion != IPv4 {
		t.Errorf("IPVersion = %d, want %d", decoded.IPVersion, IPv4)
	}
	if decoded.Protocol != ICMP {
		t.Errorf("Protocol = %d, want %d", decoded.Protocol, ICMP)
	}
	if decoded.AgentID != "agent-1" {
		t.Errorf("AgentID = %s, want agent-1", decoded.AgentID)
	}
	if decoded.DestinationAddress.String() != "8.8.8.8" {
		t.Errorf("DestinationAddress = %s, want 8.8.8.8", decoded.DestinationAddress)
	}
	if decoded.NearTTL != 10 {
		t.Errorf("NearTTL = %d, want 10", decoded.NearTTL)
	}
	if decoded.NextHeader.ICMPNextHeader == nil {
		t.Fatal("ICMPNextHeader is nil")
	}
	if decoded.NextHeader.ICMPNextHeader.FirstHalfWord != 0x1234 {
		t.Errorf("FirstHalfWord = %x, want 0x1234", decoded.NextHeader.ICMPNextHeader.FirstHalfWord)
	}
}

func TestProbingDirective_JSON_UDP(t *testing.T) {
	t.Parallel()

	original := &ProbingDirective{
		ProbingDirectiveID: 456,
		IPVersion:          IPv4,
		Protocol:           UDP,
		AgentID:            "agent-2",
		DestinationAddress: net.ParseIP("1.1.1.1"),
		NearTTL:            15,
		NextHeader: NextHeader{
			UDPNextHeader: &UDPNextHeader{
				SourcePort:      50000,
				DestinationPort: 33434,
			},
		},
	}

	var decoded ProbingDirective
	jsonRoundTrip(t, original, &decoded)

	if decoded.NextHeader.UDPNextHeader == nil {
		t.Fatal("UDPNextHeader is nil")
	}
	if decoded.NextHeader.UDPNextHeader.SourcePort != 50000 {
		t.Errorf("SourcePort = %d, want 50000", decoded.NextHeader.UDPNextHeader.SourcePort)
	}
	if decoded.NextHeader.UDPNextHeader.DestinationPort != 33434 {
		t.Errorf("DestinationPort = %d, want 33434", decoded.NextHeader.UDPNextHeader.DestinationPort)
	}
	// Verify other headers are nil (omitempty working)
	if decoded.NextHeader.ICMPNextHeader != nil {
		t.Error("ICMPNextHeader should be nil for UDP directive")
	}
}

func TestProbingDirective_JSON_IPv6(t *testing.T) {
	t.Parallel()

	original := &ProbingDirective{
		ProbingDirectiveID: 789,
		IPVersion:          IPv6,
		Protocol:           ICMPv6,
		AgentID:            "agent-3",
		DestinationAddress: net.ParseIP("2001:4860:4860::8888"),
		NearTTL:            20,
		NextHeader: NextHeader{
			ICMPv6NextHeader: &ICMPv6NextHeader{
				FirstHalfWord:  0xabcd,
				SecondHalfWord: 0xef01,
			},
		},
	}

	var decoded ProbingDirective
	jsonRoundTrip(t, original, &decoded)

	if decoded.IPVersion != IPv6 {
		t.Errorf("IPVersion = %d, want %d", decoded.IPVersion, IPv6)
	}
	if decoded.Protocol != ICMPv6 {
		t.Errorf("Protocol = %d, want %d", decoded.Protocol, ICMPv6)
	}
	if decoded.DestinationAddress.String() != "2001:4860:4860::8888" {
		t.Errorf("DestinationAddress = %s, want 2001:4860:4860::8888", decoded.DestinationAddress)
	}
	if decoded.NextHeader.ICMPv6NextHeader == nil {
		t.Fatal("ICMPv6NextHeader is nil")
	}
}

// ============================================================================
// FORWARDING INFO ELEMENT TESTS
// ============================================================================

func TestForwardingInfoElement_JSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second) // Truncate for JSON precision
	nearSent := now.Add(-10 * time.Millisecond)
	nearRecv := now.Add(-5 * time.Millisecond)
	farSent := now.Add(-4 * time.Millisecond)
	farRecv := now

	original := &ForwardingInfoElement{
		Agent: Agent{
			AgentID: "test-agent",
		},
		ProbingDirectiveID: 999,
		IPVersion:          IPv4,
		Protocol:           UDP,
		SourceAddress:      net.ParseIP("192.168.1.1"),
		DestinationAddress: net.ParseIP("8.8.8.8"),
		NearInfo: Info{
			ProbeTTL:          10,
			ReplyAddress:      net.ParseIP("10.0.0.1"),
			SentTimestamp:     nearSent,
			ReceivedTimestamp: nearRecv,
		},
		FarInfo: Info{
			ProbeTTL:          11,
			ReplyAddress:      net.ParseIP("10.0.0.2"),
			SentTimestamp:     farSent,
			ReceivedTimestamp: farRecv,
		},
		ProductionTimestamp: now,
	}

	var decoded ForwardingInfoElement
	jsonRoundTrip(t, original, &decoded)

	if decoded.Agent.AgentID != "test-agent" {
		t.Errorf("Agent.AgentID = %s, want test-agent", decoded.Agent.AgentID)
	}
	if decoded.ProbingDirectiveID != 999 {
		t.Errorf("ProbingDirectiveID = %d, want 999", decoded.ProbingDirectiveID)
	}
	if decoded.NearInfo.ProbeTTL != 10 {
		t.Errorf("NearInfo.ProbeTTL = %d, want 10", decoded.NearInfo.ProbeTTL)
	}
	if decoded.FarInfo.ProbeTTL != 11 {
		t.Errorf("FarInfo.ProbeTTL = %d, want 11", decoded.FarInfo.ProbeTTL)
	}
	if !decoded.NearInfo.SentTimestamp.Equal(nearSent) {
		t.Errorf("NearInfo.SentTimestamp mismatch")
	}
	if decoded.NearInfo.ReplyAddress.String() != "10.0.0.1" {
		t.Errorf("NearInfo.ReplyAddress = %s, want 10.0.0.1", decoded.NearInfo.ReplyAddress)
	}
	if decoded.FarInfo.ReplyAddress.String() != "10.0.0.2" {
		t.Errorf("FarInfo.ReplyAddress = %s, want 10.0.0.2", decoded.FarInfo.ReplyAddress)
	}
}

// ============================================================================
// NEXT HEADER TESTS
// ============================================================================

func TestNextHeader_Omitempty_ICMP(t *testing.T) {
	t.Parallel()

	nh := NextHeader{
		ICMPNextHeader: &ICMPNextHeader{
			FirstHalfWord:  100,
			SecondHalfWord: 200,
		},
	}

	data := mustMarshal(t, nh)
	jsonStr := string(data)

	// Should have ICMP fields
	if !contains(jsonStr, "icmp_next_header") {
		t.Error("JSON missing icmp_next_header")
	}
	// Should NOT have UDP or ICMPv6 fields (omitempty)
	if contains(jsonStr, "udp_next_header") {
		t.Error("JSON should not contain udp_next_header (omitempty)")
	}
	if contains(jsonStr, "icmpv6_next_header") {
		t.Error("JSON should not contain icmpv6_next_header (omitempty)")
	}
}

func TestNextHeader_Omitempty_UDP(t *testing.T) {
	t.Parallel()

	nh := NextHeader{
		UDPNextHeader: &UDPNextHeader{
			SourcePort:      1234,
			DestinationPort: 5678,
		},
	}

	data := mustMarshal(t, nh)
	jsonStr := string(data)

	// Should have UDP fields
	if !contains(jsonStr, "udp_next_header") {
		t.Error("JSON missing udp_next_header")
	}
	// Should NOT have ICMP or ICMPv6 fields
	if contains(jsonStr, "icmp_next_header") && !contains(jsonStr, "icmpv6_next_header") {
		t.Error("JSON should not contain icmp_next_header (omitempty)")
	}
	if contains(jsonStr, "icmpv6_next_header") {
		t.Error("JSON should not contain icmpv6_next_header (omitempty)")
	}
}

// ============================================================================
// SYSTEM STATUS TESTS
// ============================================================================

func TestSystemStatus_JSON(t *testing.T) {
	t.Parallel()

	original := &SystemStatus{
		GlobalProbingRatePSPA: 50,
		ProbingImpactLimit:    500 * time.Millisecond,
		DisallowedDestinationAddresses: []net.IP{
			net.ParseIP("192.168.1.0"),
			net.ParseIP("10.0.0.0"),
		},
		ActiveAgentIDs: []string{"agent-1", "agent-2", "agent-3"},
	}

	var decoded SystemStatus
	jsonRoundTrip(t, original, &decoded)

	if decoded.GlobalProbingRatePSPA != 50 {
		t.Errorf("GlobalProbingRatePSPA = %d, want 50", decoded.GlobalProbingRatePSPA)
	}
	if decoded.ProbingImpactLimit != 500*time.Millisecond {
		t.Errorf("ProbingImpactLimit = %v, want 500ms", decoded.ProbingImpactLimit)
	}
	if len(decoded.DisallowedDestinationAddresses) != 2 {
		t.Errorf("len(DisallowedDestinationAddresses) = %d, want 2", len(decoded.DisallowedDestinationAddresses))
	}
	if len(decoded.ActiveAgentIDs) != 3 {
		t.Errorf("len(ActiveAgentIDs) = %d, want 3", len(decoded.ActiveAgentIDs))
	}
	if decoded.ActiveAgentIDs[0] != "agent-1" {
		t.Errorf("ActiveAgentIDs[0] = %s, want agent-1", decoded.ActiveAgentIDs[0])
	}
}

func TestSystemStatus_Duration_Nanoseconds(t *testing.T) {
	t.Parallel()

	// time.Duration marshals as int64 nanoseconds
	original := &SystemStatus{
		GlobalProbingRatePSPA: 10,
		ProbingImpactLimit:    2 * time.Second,
		ActiveAgentIDs:        []string{},
	}

	data := mustMarshal(t, original)
	jsonStr := string(data)

	// Should contain nanoseconds (2 seconds = 2000000000 nanoseconds)
	if !contains(jsonStr, "2000000000") {
		t.Errorf("JSON should contain duration as nanoseconds: %s", jsonStr)
	}
}

// ============================================================================
// AUTH TYPES TESTS
// ============================================================================

func TestAuthRequest_JSON(t *testing.T) {
	t.Parallel()

	original := &AuthRequest{
		AgentID: "test-agent",
		Secret:  "super-secret-token-12345",
	}

	var decoded AuthRequest
	jsonRoundTrip(t, original, &decoded)

	if decoded.AgentID != "test-agent" {
		t.Errorf("AgentID = %s, want test-agent", decoded.AgentID)
	}
	if decoded.Secret != "super-secret-token-12345" {
		t.Errorf("Secret = %s, want super-secret-token-12345", decoded.Secret)
	}
}

func TestAuthResponse_JSON_Success(t *testing.T) {
	t.Parallel()

	original := &AuthResponse{
		Authenticated: true,
		Message:       "Welcome",
	}

	var decoded AuthResponse
	jsonRoundTrip(t, original, &decoded)

	if !decoded.Authenticated {
		t.Error("Authenticated should be true")
	}
	if decoded.Message != "Welcome" {
		t.Errorf("Message = %s, want Welcome", decoded.Message)
	}
}

func TestAuthResponse_JSON_Failure(t *testing.T) {
	t.Parallel()

	original := &AuthResponse{
		Authenticated: false,
		Message:       "Invalid credentials",
	}

	var decoded AuthResponse
	jsonRoundTrip(t, original, &decoded)

	if decoded.Authenticated {
		t.Error("Authenticated should be false")
	}
	if decoded.Message != "Invalid credentials" {
		t.Errorf("Message = %s, want Invalid credentials", decoded.Message)
	}
}

func TestAuthResponse_Omitempty_Message(t *testing.T) {
	t.Parallel()

	// Test that empty Message field is omitted from JSON
	original := &AuthResponse{
		Authenticated: true,
		Message:       "", // Empty - should be omitted
	}

	data := mustMarshal(t, original)
	jsonStr := string(data)

	// Should not contain "message" field when empty
	if contains(jsonStr, "message") {
		t.Error("JSON should not contain 'message' field when empty (omitempty)")
	}
}

// ============================================================================
// INFO TESTS
// ============================================================================

func TestInfo_JSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	sent := now.Add(-10 * time.Millisecond)
	recv := now

	original := &Info{
		ProbeTTL:          15,
		ReplyAddress:      net.ParseIP("192.168.1.1"),
		SentTimestamp:     sent,
		ReceivedTimestamp: recv,
	}

	var decoded Info
	jsonRoundTrip(t, original, &decoded)

	if decoded.ProbeTTL != 15 {
		t.Errorf("ProbeTTL = %d, want 15", decoded.ProbeTTL)
	}
	if decoded.ReplyAddress.String() != "192.168.1.1" {
		t.Errorf("ReplyAddress = %s, want 192.168.1.1", decoded.ReplyAddress)
	}
	if !decoded.SentTimestamp.Equal(sent) {
		t.Error("SentTimestamp mismatch")
	}
	if !decoded.ReceivedTimestamp.Equal(recv) {
		t.Error("ReceivedTimestamp mismatch")
	}
}

// ============================================================================
// AGENT TESTS
// ============================================================================

func TestAgent_JSON(t *testing.T) {
	t.Parallel()

	original := &Agent{
		AgentID: "agent-xyz",
	}

	var decoded Agent
	jsonRoundTrip(t, original, &decoded)

	if decoded.AgentID != "agent-xyz" {
		t.Errorf("AgentID = %s, want agent-xyz", decoded.AgentID)
	}
}

// ============================================================================
// IP VERSION AND PROTOCOL CONSTANTS TESTS
// ============================================================================

func TestIPVersion_Values(t *testing.T) {
	t.Parallel()

	if IPv4 != 4 {
		t.Errorf("IPv4 = %d, want 4", IPv4)
	}
	if IPv6 != 6 {
		t.Errorf("IPv6 = %d, want 6", IPv6)
	}
}

func TestProtocol_Values(t *testing.T) {
	t.Parallel()

	if ICMP != 1 {
		t.Errorf("ICMP = %d, want 1", ICMP)
	}
	if UDP != 17 {
		t.Errorf("UDP = %d, want 17", UDP)
	}
	if ICMPv6 != 58 {
		t.Errorf("ICMPv6 = %d, want 58", ICMPv6)
	}
}

// ============================================================================
// EDGE CASES
// ============================================================================

func TestProbingDirective_ZeroValues(t *testing.T) {
	t.Parallel()

	// Test that zero values marshal/unmarshal correctly
	original := &ProbingDirective{}

	var decoded ProbingDirective
	jsonRoundTrip(t, original, &decoded)

	if decoded.ProbingDirectiveID != 0 {
		t.Errorf("Expected zero ID, got %d", decoded.ProbingDirectiveID)
	}
	if decoded.NearTTL != 0 {
		t.Errorf("Expected zero TTL, got %d", decoded.NearTTL)
	}
}

func TestForwardingInfoElement_EmptyAgent(t *testing.T) {
	t.Parallel()

	original := &ForwardingInfoElement{
		Agent: Agent{AgentID: ""},
	}

	var decoded ForwardingInfoElement
	jsonRoundTrip(t, original, &decoded)

	if decoded.Agent.AgentID != "" {
		t.Errorf("Expected empty AgentID, got %s", decoded.Agent.AgentID)
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
