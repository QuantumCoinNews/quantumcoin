package token

import (
	"errors"
	"testing"
)

func TestAttachToRunningMinerV1FailsClosed(
	t *testing.T,
) {
	bridge := &MinerTokenBridge{}

	err := AttachToRunningMiner(bridge)

	if !errors.Is(
		err,
		ErrAttachToRunningMinerUnsupportedV1,
	) {
		t.Fatalf(
			"AttachToRunningMiner error = %v, want %v",
			err,
			ErrAttachToRunningMinerUnsupportedV1,
		)
	}
}

func TestAttachToRunningMinerV1NilBridgeStillFailsClosed(
	t *testing.T,
) {
	err := AttachToRunningMiner(nil)

	if !errors.Is(
		err,
		ErrAttachToRunningMinerUnsupportedV1,
	) {
		t.Fatalf(
			"AttachToRunningMiner(nil) error = %v, want %v",
			err,
			ErrAttachToRunningMinerUnsupportedV1,
		)
	}
}
