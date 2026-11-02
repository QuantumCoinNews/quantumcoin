# QuantumCoin P2P Message Size Security

Status: QC_P2P_MESSAGE_SIZE_STRICT_V1

Date: 20260617_193535

P2P messages must pass strict payload-size validation before message-specific decode or handling.

## Active rule

All inbound P2P messages must pass:

- p2p.ValidateP2PMessageSizeStrictV1

## Limits

- Global payload max: MaxMessageSize
- Hello max: 16 KiB
- Tx max: 256 KiB
- Block max: 512 KiB
- Chain max: MaxMessageSize
- Small messages: 16 KiB

## Enforcement points

- HandleConnection checks size immediately after gob message decode and before goroutine dispatch.
- handleMessage checks size again before per-type decode.
- PeerSecurity.AllowMessage still performs IP/rate/size enforcement.

## Safety boundary

This layer does not change chain, balances, rewards, total supply, mining, or consensus.

It only rejects unsafe or oversized peer messages before processing.
