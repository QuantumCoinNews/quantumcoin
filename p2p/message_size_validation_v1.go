package p2p

import "fmt"

// P2P_MESSAGE_SIZE_STRICT_V1
//
// This file makes P2P message size validation explicit and always-on.
// It does not change chain, balances, mining, rewards, or consensus.
// It only rejects unsafe peer messages before message-specific decode/handling.

const (
	MaxP2PHelloDataBytesV1 = 16 << 10  // 16 KiB
	MaxP2PTxDataBytesV1    = 256 << 10 // 256 KiB
	MaxP2PBlockDataBytesV1 = 512 << 10 // 512 KiB
	MaxP2PChainDataBytesV1 = MaxMessageSize
	MaxP2PSmallDataBytesV1 = 16 << 10 // request/ping/pong/error/peerlist
)

// ValidateP2PMessageSizeStrictV1 validates the payload size by message type.
// It must run before any message-specific gob decode.
func ValidateP2PMessageSizeStrictV1(msg Message) error {
	size := len(msg.Data)
	if size < 0 {
		return fmt.Errorf("p2p message invalid negative size")
	}

	max := MaxMessageSize

	switch msg.Type {
	case MsgHello:
		max = MaxP2PHelloDataBytesV1
	case MsgTx:
		max = MaxP2PTxDataBytesV1
	case MsgBlock:
		max = MaxP2PBlockDataBytesV1
	case MsgChain:
		max = MaxP2PChainDataBytesV1
	case MsgRequest, MsgPing, MsgPong, MsgPeerList, MsgError:
		max = MaxP2PSmallDataBytesV1
	default:
		max = MaxP2PSmallDataBytesV1
	}

	if size > MaxMessageSize {
		return fmt.Errorf("%w: type=%s size=%d max=%d", ErrMsgTooLarge, msg.Type, size, MaxMessageSize)
	}

	if size > max {
		return fmt.Errorf("%w: type=%s size=%d typeMax=%d", ErrMsgTooLarge, msg.Type, size, max)
	}

	return nil
}
