package p2p

import (
	"net"
	"testing"
	"time"
)

func TestCloseAllPeersV1ClosesRegisteredPeers(
	t *testing.T,
) {
	left, right := net.Pipe()

	registerPeer(left)

	if GetPeerCount() != 1 {
		_ = left.Close()
		_ = right.Close()

		t.Fatalf(
			"peer count = %d, want 1",
			GetPeerCount(),
		)
	}

	closed := CloseAllPeersV1()

	if closed != 1 {
		_ = right.Close()

		t.Fatalf(
			"closed peer count = %d, want 1",
			closed,
		)
	}

	_ = right.SetWriteDeadline(
		time.Now().Add(
			100 * time.Millisecond,
		),
	)

	_, err := right.Write(
		[]byte("probe"),
	)
	if err == nil {
		_ = right.Close()

		t.Fatal(
			"peer connection remained writable after CloseAllPeersV1",
		)
	}

	unregisterPeer(left)
	_ = right.Close()
}
