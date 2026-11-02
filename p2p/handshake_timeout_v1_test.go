package p2p

import (
	"net"
	"testing"
	"time"
)

func TestHandshakeTimeoutV1ClosesSilentPeer(
	t *testing.T,
) {
	left, right := net.Pipe()

	addr := left.RemoteAddr().String()

	handshakeMu.Lock()
	delete(handshakeOK, addr)
	handshakeMu.Unlock()

	timer := startHandshakeTimeoutV1(
		left,
		30*time.Millisecond,
	)
	if timer == nil {
		t.Fatal(
			"handshake timer was not created",
		)
	}
	defer timer.Stop()

	_ = right.SetReadDeadline(
		time.Now().Add(
			500 * time.Millisecond,
		),
	)

	buf := make([]byte, 1)

	_, err := right.Read(buf)
	if err == nil {
		t.Fatal(
			"silent peer remained connected after handshake timeout",
		)
	}

	_ = left.Close()
	_ = right.Close()

	handshakeMu.Lock()
	delete(handshakeOK, addr)
	handshakeMu.Unlock()
}
