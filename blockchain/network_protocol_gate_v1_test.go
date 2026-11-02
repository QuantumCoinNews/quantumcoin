package blockchain

import (
	"strings"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestNetworkProtocolGateAcceptsLegacyV1(
	t *testing.T,
) {
	if err := validateNetworkBlockProtocolV1(
		qconsensus.ProtocolLegacyV1,
	); err != nil {
		t.Fatalf(
			"LegacyV1 network protocol rejected: %v",
			err,
		)
	}
}

func TestNetworkProtocolGateAcceptsAtomV3V1(
	t *testing.T,
) {
	if err := validateNetworkBlockProtocolV1(
		qconsensus.ProtocolAtomV3,
	); err != nil {
		t.Fatalf(
			"AtomV3 network protocol rejected: %v",
			err,
		)
	}
}

func TestNetworkProtocolGateRejectsContinuityV2V1(
	t *testing.T,
) {
	err := validateNetworkBlockProtocolV1(
		qconsensus.ProtocolContinuityV2,
	)
	if err == nil {
		t.Fatal(
			"ContinuityV2 network protocol unexpectedly accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"not implemented",
	) {
		t.Fatalf(
			"unexpected ContinuityV2 error: %v",
			err,
		)
	}
}

func TestNetworkProtocolGateRejectsUnknownV1(
	t *testing.T,
) {
	err := validateNetworkBlockProtocolV1(
		qconsensus.ProtocolVersion("UNKNOWN"),
	)
	if err == nil {
		t.Fatal(
			"unknown network protocol unexpectedly accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"unsupported protocol",
	) {
		t.Fatalf(
			"unexpected unknown protocol error: %v",
			err,
		)
	}
}
