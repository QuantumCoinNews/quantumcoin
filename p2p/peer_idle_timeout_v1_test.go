package p2p

import (
	"encoding/gob"
	"net"
	"testing"
	"time"
)

type cappedReadDeadlineConnV1 struct {
	net.Conn
	maxDelay time.Duration
}

func (c *cappedReadDeadlineConnV1) SetReadDeadline(
	_ time.Time,
) error {
	return c.Conn.SetReadDeadline(
		time.Now().Add(c.maxDelay),
	)
}

func TestHandleConnectionPeerIdleTimeoutV1CleansState(
	t *testing.T,
) {
	left, right := net.Pipe()
	conn := &cappedReadDeadlineConnV1{
		Conn:     left,
		maxDelay: 30 * time.Millisecond,
	}
	addr := conn.RemoteAddr().String()

	registerPeer(conn)
	setHandshakeOK(addr)

	done := make(chan struct{})
	go func() {
		HandleConnection(conn, nil)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = right.Close()
		t.Fatal("HandleConnection did not exit after peer idle timeout")
	}

	peersMu.Lock()
	_, peerExists := peers[addr]
	peersMu.Unlock()
	if peerExists {
		_ = right.Close()
		t.Fatal("idle peer remained registered after timeout")
	}

	if isHandshakeOK(addr) {
		_ = right.Close()
		t.Fatal("idle peer handshake state remained after timeout")
	}

	_ = right.Close()
}

func TestRunPeerHeartbeatV1SendsPing(
	t *testing.T,
) {
	left, right := net.Pipe()
	addr := left.RemoteAddr().String()

	registerPeer(left)
	setHandshakeOK(addr)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		runPeerHeartbeatV1(
			left,
			stop,
			10*time.Millisecond,
		)
		close(done)
	}()

	if err := right.SetReadDeadline(
		time.Now().Add(500 * time.Millisecond),
	); err != nil {
		close(stop)
		_ = right.Close()
		t.Fatalf("set heartbeat test deadline: %v", err)
	}

	var msg Message
	if err := gob.NewDecoder(right).Decode(&msg); err != nil {
		close(stop)
		_ = right.Close()
		t.Fatalf("decode heartbeat: %v", err)
	}
	if msg.Type != MsgPing {
		close(stop)
		_ = right.Close()
		t.Fatalf(
			"heartbeat type = %q, want %q",
			msg.Type,
			MsgPing,
		)
	}

	close(stop)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		_ = right.Close()
		t.Fatal("heartbeat goroutine did not stop")
	}

	unregisterPeer(left)
	_ = right.Close()
}
