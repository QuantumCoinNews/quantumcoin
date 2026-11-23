# Fresh V4 transaction commitments

This encoding activates only through a trusted `network-v4` genesis. It does
not migrate historical chains. Gob remains a wire/storage format; its type
allocation IDs must never enter fresh V4 transaction hashes or signatures.

`QC_TX_CONTENT_V2` serializes the following sequence with no padding:

| Field | Encoding |
| --- | --- |
| Domain | ASCII `QC_TX_CONTENT_V2` followed by a zero byte |
| Input count | Unsigned 32-bit big endian |
| Each input | Length-prefixed TxID, then signed outpoint index represented as 64-bit two's complement big endian |
| Output count | Unsigned 32-bit big endian |
| Each output | Signed amount as 64-bit two's complement big endian, then length-prefixed public-key hash |
| Timestamp | Unix seconds as 64-bit two's complement big endian, then nanoseconds as unsigned 32-bit big endian |
| Sender | Length-prefixed UTF-8 bytes |
| Convenience amount | IEEE-754 binary64 bits, unsigned 64-bit big endian |

Each byte-string length is unsigned 32-bit big endian. Transaction ID, input
signatures and public keys are excluded. Time zones and monotonic clock data
are excluded. The existing amount-validation rules still apply; this encoding
does not redefine denomination. Non-finite convenience amounts are rejected.
Limits are 5,000 inputs/outputs, 1 MiB per byte string and 16 MiB total content.

The transaction ID is SHA-256 of these content bytes. An input digest is
SHA-256 of `QC_TX_INPUT_SIG_V2\0`, these content bytes and the input index as
unsigned 32-bit big endian. The final signature digest is SHA-256 of
`QC_TX_SIG_NETWORK_V2\0`, the trusted 32-byte genesis hash and that input
digest. Signatures retain the existing secp256k1 verification rules.

Structural parsing can recognize either supported commitment. Chain-aware
admission and block validation require exactly the codec selected by the
trusted genesis, including coinbase. They never retry a rejected V4 signature
or commitment with historical Gob encoding.

Independent cold/warmed executable processes verify identical genesis,
transaction IDs and signing digests, and verify a transaction signed in another
process. A fixed content vector independently constructed with Python
`struct.pack` is checked in `blockchain/transaction_encoding_v2_test.go`.
The official zero-allocation testnet fingerprint is `testnet/genesis.json`.
The mainnet fingerprint and allocation remain unfinalized.
