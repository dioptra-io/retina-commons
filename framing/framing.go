// Copyright (c) 2025 Sorbonne Université
// SPDX-License-Identifier: MIT

// Package framing implements length-prefixed message framing for streaming
// wire/v2 protobuf messages over a persistent TCP connection. Protobuf's
// binary encoding may contain arbitrary bytes, so messages can't be
// newline-delimited the way the legacy JSONL protocol was — each message
// is prefixed with its own length instead.
//
// Send/Receive take a plain proto.Message rather than a self-describing
// envelope type, since the orchestrator<->agent protocol is staged and
// directional — each side always already knows the concrete type of the
// next frame. This mirrors v1's json.Encoder.Encode(v)/Decoder.Decode(&v)
// pattern, just swapping JSON for length-prefixed protobuf.
//
// timeout == 0 means no deadline: both functions explicitly clear any
// previously set deadline in that case, since net.Conn deadlines persist
// across calls otherwise. timeout < 0 is rejected as a caller error
// rather than also silently meaning "no deadline".
package framing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"

	"google.golang.org/protobuf/proto"
)

// maxFrameSize bounds the declared payload length so a corrupted or
// malicious length prefix can't trigger unbounded allocation. FIEs and
// PDs serialize to well under 1KB in practice, so 32MB leaves large
// headroom for schema growth while still rejecting a bogus length early.
// Enforced on both Send and Receive, so a peer can never emit a frame
// its own Receive (or the other side's) is guaranteed to reject.
const maxFrameSize = 32 * 1024 * 1024 // 32MB

// ErrProtocolViolation wraps errors indicating the stream itself is
// desynchronized or the peer sent a malformed message, as opposed to a
// transient network error. Callers should treat errors.Is(err,
// ErrProtocolViolation) as fatal to the connection — close the socket
// and force reconnection — rather than continuing on the same one. A
// local frame that's merely too large before anything is written is not
// wrapped in this, since the stream itself was never touched.
var ErrProtocolViolation = errors.New("protocol violation")

// isNilMessage reports whether msg is nil, including a typed-nil
// pointer stored in a non-nil interface (e.g. var pd *wire.Info; var
// msg proto.Message = pd — msg != nil here, but pd itself is nil).
// A plain msg == nil check misses this case, which proto.Marshal or
// proto.Unmarshal may then handle unexpectedly.
func isNilMessage(msg proto.Message) bool {
	if msg == nil {
		return true
	}
	v := reflect.ValueOf(msg)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// setDeadline applies timeout, or explicitly clears any existing
// deadline when timeout is 0 (see package doc). Rejects timeout < 0.
func setDeadline(set func(time.Time) error, timeout time.Duration) error {
	if timeout < 0 {
		return errors.New("timeout must be non-negative")
	}
	if timeout > 0 {
		return set(time.Now().Add(timeout))
	}
	return set(time.Time{})
}

// writeFull writes all of p, looping over short writes rather than
// assuming a single Write call consumes the whole buffer.
func writeFull(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if n > 0 {
			p = p[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

// Send marshals msg and writes it to conn, prefixed with its length.
func Send(conn net.Conn, timeout time.Duration, msg proto.Message) error {
	if conn == nil {
		return errors.New("connection is nil")
	}
	if isNilMessage(msg) {
		return errors.New("message is nil")
	}

	payload, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}
	if len(payload) > maxFrameSize {
		return fmt.Errorf("payload size %d exceeds max allowed %d", len(payload), maxFrameSize)
	}

	if err := setDeadline(conn.SetWriteDeadline, timeout); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))

	if err := writeFull(conn, header[:]); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if err := writeFull(conn, payload); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}
	return nil
}

// Receive reads the length-prefixed frame from conn and unmarshals it
// into msg, which must be a non-nil pointer to a concrete proto message
// (e.g. &wire.ProbingDirective{}) — mirrors json.Decoder.Decode(&v).
// proto.Unmarshal resets msg before decoding, so the same msg value can
// be safely reused across repeated Receive calls in a loop.
func Receive(conn net.Conn, timeout time.Duration, msg proto.Message) error {
	if conn == nil {
		return errors.New("connection is nil")
	}
	if isNilMessage(msg) {
		return errors.New("message is nil")
	}

	if err := setDeadline(conn.SetReadDeadline, timeout); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	header := make([]byte, 4)
	if n, err := io.ReadFull(conn, header); err != nil {
		// No frame bytes were consumed, so the framing state remains
		// aligned — the caller still decides whether the underlying
		// connection error itself is retryable. n > 0 means part of a
		// frame was read and can never be un-read, so the stream is
		// desynchronized regardless of why the read stopped.
		if n == 0 {
			return err
		}
		return fmt.Errorf("%w: incomplete frame header (%d/%d bytes): %v", ErrProtocolViolation, n, len(header), err)
	}

	length := binary.BigEndian.Uint32(header)
	if length > maxFrameSize {
		return fmt.Errorf("%w: declared frame size %d exceeds max allowed %d", ErrProtocolViolation, length, maxFrameSize)
	}

	// Unlike the header read above, there's no clean-boundary case here:
	// the header was already fully consumed, committing the connection
	// to length more bytes. Any failure at this point — including a
	// zero-byte read — means the peer declared a frame it didn't
	// deliver, so this is always a protocol violation.
	payload := make([]byte, length)
	if n, err := io.ReadFull(conn, payload); err != nil {
		return fmt.Errorf("%w: incomplete payload (%d/%d bytes): %v", ErrProtocolViolation, n, length, err)
	}

	if err := proto.Unmarshal(payload, msg); err != nil {
		return fmt.Errorf("%w: failed to unmarshal protobuf: %w", ErrProtocolViolation, err)
	}
	return nil
}
