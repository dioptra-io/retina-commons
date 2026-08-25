// Copyright (c) 2025 Sorbonne Université
// SPDX-License-Identifier: MIT

package framing

// Two branches in framing.go are intentionally left uncovered by these
// tests, not overlooked:
//   - isNilMessage's `default` case in the reflect.Kind() switch is
//     unreachable via any real proto.Message, since every generated
//     message type is a pointer (Kind() is always reflect.Pointer).
//   - Send's proto.Marshal error branch can't be triggered by a
//     well-formed generated message; hitting it would require a
//     custom proto.Message implementation built solely to fail
//     Marshal, which would test the fake type, not this package.

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	wire "github.com/dioptra-io/retina-commons/wire/v2"
	"google.golang.org/protobuf/proto"
)

// -- Test doubles -----------------------------------------------------------

// shortWriteConn wraps a real net.Conn but caps each individual Write
// call to maxPerWrite bytes, forcing writeFull's looping logic to
// actually loop rather than succeed in a single call.
type shortWriteConn struct {
	net.Conn
	maxPerWrite int
}

func (c *shortWriteConn) Write(p []byte) (int, error) {
	if len(p) > c.maxPerWrite {
		p = p[:c.maxPerWrite]
	}
	return c.Conn.Write(p)
}

// zeroWriteConn always reports writing 0 bytes with no error — the
// pathological case writeFull must reject as io.ErrShortWrite rather
// than looping forever.
type zeroWriteConn struct {
	net.Conn
}

func (c *zeroWriteConn) Write(p []byte) (int, error) {
	return 0, nil
}

// -- Send/Receive round trip --------------------------------------------

func TestSendReceiveRoundTrip(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	want := &wire.ProbingDirective{
		ProbingDirectiveId: 42,
		AgentId:            "agent-1",
		DestinationAddress: "192.0.2.1",
		NearTtl:            10,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- Send(client, 0, want) }()

	var got wire.ProbingDirective
	if err := Receive(server, 0, &got); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got.GetProbingDirectiveId() != want.GetProbingDirectiveId() {
		t.Errorf("ProbingDirectiveId: got %d, want %d", got.GetProbingDirectiveId(), want.GetProbingDirectiveId())
	}
}

// TestReceiveReusesDestinationMessage confirms proto.Unmarshal resets
// msg before decoding, so the same value can be reused across a loop of
// Receive calls without stale fields leaking between messages.
func TestReceiveReusesDestinationMessage(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	first := &wire.ProbingDirective{ProbingDirectiveId: 1, DestinationAddress: "192.0.2.1"}
	second := &wire.ProbingDirective{ProbingDirectiveId: 2} // no DestinationAddress

	go func() {
		_ = Send(client, 0, first)
		_ = Send(client, 0, second)
	}()

	var msg wire.ProbingDirective
	if err := Receive(server, 0, &msg); err != nil {
		t.Fatalf("Receive (first): %v", err)
	}
	if msg.GetDestinationAddress() != "192.0.2.1" {
		t.Fatalf("first message: got destination %q, want 192.0.2.1", msg.GetDestinationAddress())
	}

	if err := Receive(server, 0, &msg); err != nil {
		t.Fatalf("Receive (second): %v", err)
	}
	if msg.GetDestinationAddress() != "" {
		t.Errorf("second message: got leftover destination %q from first message, want empty (Unmarshal should reset msg)", msg.GetDestinationAddress())
	}
	if msg.GetProbingDirectiveId() != 2 {
		t.Errorf("second message: got id %d, want 2", msg.GetProbingDirectiveId())
	}
}

// -- Oversized frames ---------------------------------------------------

func TestSendRejectsOversizedPayload(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// A ReplyAddress string alone bigger than maxFrameSize guarantees
	// the marshaled payload exceeds it too, regardless of field overhead.
	oversized := &wire.Info{ReplyAddress: strings.Repeat("A", maxFrameSize+1)}

	err := Send(client, 0, oversized)
	if err == nil {
		t.Fatal("expected error for oversized payload, got nil")
	}
	if errors.Is(err, ErrProtocolViolation) {
		t.Error("a local oversized-payload error should not be classified as ErrProtocolViolation — the stream was never touched")
	}
}

func TestReceiveRejectsOversizedDeclaredLength(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		header := []byte{0xFF, 0xFF, 0xFF, 0xFF} // declares ~4GB, way over maxFrameSize
		_, _ = client.Write(header)
	}()

	var msg wire.ProbingDirective
	err := Receive(server, time.Second, &msg)
	if err == nil {
		t.Fatal("expected error for oversized declared length, got nil")
	}
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation, got: %v", err)
	}
}

// -- Short writes ---------------------------------------------------------

func TestSendHandlesShortWrites(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	wrapped := &shortWriteConn{Conn: client, maxPerWrite: 1} // force many tiny writes
	want := &wire.ProbingDirective{ProbingDirectiveId: 7, DestinationAddress: "192.0.2.1"}

	errCh := make(chan error, 1)
	go func() { errCh <- Send(wrapped, 0, want) }()

	var got wire.ProbingDirective
	if err := Receive(server, 0, &got); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got.GetProbingDirectiveId() != 7 {
		t.Errorf("got id %d, want 7", got.GetProbingDirectiveId())
	}
}

func TestWriteFullRejectsZeroByteWrite(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()

	wrapped := &zeroWriteConn{Conn: client}
	err := writeFull(wrapped, []byte{1, 2, 3})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("expected io.ErrShortWrite, got: %v", err)
	}
}

// -- Truncated reads ------------------------------------------------------

func TestReceivePartialHeaderThenEOF(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		_, _ = client.Write([]byte{0x00, 0x00}) // 2 of 4 header bytes
		client.Close()                          // then hang up
	}()

	var msg wire.ProbingDirective
	err := Receive(server, time.Second, &msg)
	if err == nil {
		t.Fatal("expected error for partial header, got nil")
	}
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation (partial header desyncs the stream), got: %v", err)
	}
}

func TestReceiveCompleteHeaderThenTruncatedPayload(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		header := make([]byte, 4)
		header[3] = 100 // declares a 100-byte payload
		_, _ = client.Write(header)
		_, _ = client.Write([]byte{1, 2, 3}) // deliver only 3 of 100 bytes
		client.Close()
	}()

	var msg wire.ProbingDirective
	err := Receive(server, time.Second, &msg)
	if err == nil {
		t.Fatal("expected error for truncated payload, got nil")
	}
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation, got: %v", err)
	}
}

// -- Invalid protobuf -------------------------------------------------------

func TestReceiveRejectsInvalidProtobuf(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		garbage := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF} // not valid protobuf
		header := make([]byte, 4)
		header[3] = byte(len(garbage))
		_, _ = client.Write(header)
		_, _ = client.Write(garbage)
	}()

	var msg wire.ProbingDirective
	err := Receive(server, time.Second, &msg)
	if err == nil {
		t.Fatal("expected error for invalid protobuf payload, got nil")
	}
	if !errors.Is(err, ErrProtocolViolation) {
		t.Errorf("expected ErrProtocolViolation, got: %v", err)
	}
}

// -- Timeouts -------------------------------------------------------------

func TestReceiveTimesOutWithNoData(t *testing.T) {
	_, server := net.Pipe()
	defer server.Close()

	err := Receive(server, 50*time.Millisecond, &wire.ProbingDirective{})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Errorf("expected a net.Error with Timeout() true, got: %v", err)
	}
}

func TestSendTimesOutWhenUnread(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	defer client.Close()

	err := Send(client, 50*time.Millisecond, &wire.ProbingDirective{DestinationAddress: "192.0.2.1"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Errorf("expected a net.Error with Timeout() true, got: %v", err)
	}
}

// -- Nil / invalid arguments ------------------------------------------------

func TestSendRejectsNilConn(t *testing.T) {
	if err := Send(nil, 0, &wire.ProbingDirective{}); err == nil {
		t.Error("expected error for nil conn, got nil")
	}
}

func TestSendRejectsNilMessage(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()
	if err := Send(client, 0, nil); err == nil {
		t.Error("expected error for nil message, got nil")
	}
}

func TestSendRejectsTypedNilMessage(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()

	var pd *wire.ProbingDirective // nil pointer
	var msg proto.Message = pd    // non-nil interface wrapping a nil pointer

	if err := Send(client, 0, msg); err == nil {
		t.Error("expected error for typed-nil message, got nil")
	}
}

func TestReceiveRejectsNilConn(t *testing.T) {
	if err := Receive(nil, 0, &wire.ProbingDirective{}); err == nil {
		t.Error("expected error for nil conn, got nil")
	}
}

func TestReceiveRejectsNilMessage(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()
	if err := Receive(client, 0, nil); err == nil {
		t.Error("expected error for nil message, got nil")
	}
}

func TestSendRejectsNegativeTimeout(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()
	if err := Send(client, -time.Second, &wire.ProbingDirective{DestinationAddress: "192.0.2.1"}); err == nil {
		t.Error("expected error for negative timeout, got nil")
	}
}

func TestReceiveRejectsNegativeTimeout(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()
	if err := Receive(client, -time.Second, &wire.ProbingDirective{}); err == nil {
		t.Error("expected error for negative timeout, got nil")
	}
}

// -- Additional coverage: Send payload-write failure -----------------------

// failAfterHeaderConn lets the 4-byte header write through untouched
// (so Send gets past that point), then fails every subsequent write —
// exercising the "failed to write payload" branch, which the short-write
// test doesn't reach since it always succeeds eventually.
type failAfterHeaderConn struct {
	net.Conn
	headerWritten bool
}

func (c *failAfterHeaderConn) Write(p []byte) (int, error) {
	if !c.headerWritten && len(p) == 4 {
		c.headerWritten = true
		return c.Conn.Write(p)
	}
	return 0, errors.New("simulated write failure")
}

func TestSendFailsOnPayloadWriteError(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	wrapped := &failAfterHeaderConn{Conn: client}

	// The header write blocks until read on a real net.Pipe, so drain it
	// on the other end before Send attempts the payload write.
	go func() {
		header := make([]byte, 4)
		_, _ = io.ReadFull(server, header)
	}()

	err := Send(wrapped, time.Second, &wire.ProbingDirective{DestinationAddress: "192.0.2.1"})
	if err == nil {
		t.Fatal("expected error for failed payload write, got nil")
	}
}

// -- Additional coverage: Send deadline and header-write failures ----------

// failAllWritesConn fails every write immediately, including the header —
// unlike failAfterHeaderConn, which lets the header through first.
type failAllWritesConn struct {
	net.Conn
}

func (c *failAllWritesConn) Write(p []byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

func TestSendFailsOnHeaderWriteError(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()

	wrapped := &failAllWritesConn{Conn: client}
	err := Send(wrapped, 0, &wire.ProbingDirective{DestinationAddress: "192.0.2.1"})
	if err == nil {
		t.Fatal("expected error for failed header write, got nil")
	}
}

// failSetWriteDeadlineConn fails SetWriteDeadline outright — the one
// branch in setDeadline that a well-behaved net.Pipe never exercises.
type failSetWriteDeadlineConn struct {
	net.Conn
}

func (c *failSetWriteDeadlineConn) SetWriteDeadline(t time.Time) error {
	return errors.New("simulated deadline failure")
}

func TestSendFailsWhenSetWriteDeadlineErrors(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()

	wrapped := &failSetWriteDeadlineConn{Conn: client}
	err := Send(wrapped, 0, &wire.ProbingDirective{DestinationAddress: "192.0.2.1"})
	if err == nil {
		t.Fatal("expected error for failed SetWriteDeadline, got nil")
	}
}
