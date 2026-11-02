package p2p

import "testing"

func TestValidateP2PMessageSizeStrictV1AcceptsSmallHello(t *testing.T) {
	msg := Message{Type: MsgHello, Data: make([]byte, 128)}
	if err := ValidateP2PMessageSizeStrictV1(msg); err != nil {
		t.Fatalf("expected small hello accepted, got: %v", err)
	}
}

func TestValidateP2PMessageSizeStrictV1RejectsOversizeHello(t *testing.T) {
	msg := Message{Type: MsgHello, Data: make([]byte, MaxP2PHelloDataBytesV1+1)}
	if err := ValidateP2PMessageSizeStrictV1(msg); err == nil {
		t.Fatalf("expected oversize hello rejected")
	}
}

func TestValidateP2PMessageSizeStrictV1RejectsOversizeTx(t *testing.T) {
	msg := Message{Type: MsgTx, Data: make([]byte, MaxP2PTxDataBytesV1+1)}
	if err := ValidateP2PMessageSizeStrictV1(msg); err == nil {
		t.Fatalf("expected oversize tx rejected")
	}
}

func TestValidateP2PMessageSizeStrictV1RejectsGlobalOversizeChain(t *testing.T) {
	msg := Message{Type: MsgChain, Data: make([]byte, MaxMessageSize+1)}
	if err := ValidateP2PMessageSizeStrictV1(msg); err == nil {
		t.Fatalf("expected globally oversize chain rejected")
	}
}
