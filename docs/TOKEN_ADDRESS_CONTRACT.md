# Token address contract

The token library accepts canonical Base58Check addresses with a 21-byte payload
and a four-byte checksum. Its existing address version is `0x00`. This is an
address-format/version rule, not a cryptographic testnet/mainnet signing domain.
Changing the version or token consensus authority requires a separate decision.

- `NormalizeTokenAddressV1` checks the alphabet, decoded size, version, checksum
  and round-trip representation. Surrounding whitespace can be trimmed for an
  explicit presentation/input normalization step.
- State-changing and signing paths require the original input to equal the
  canonical representation. Padded strings are rejected; they do not create
  alternate balance or allowance keys.
- Encoded input is limited to 35 characters after trimming, before Base58 big
  integer decoding. All 25-byte Base58Check values fit within that bound.
- Untrusted construction uses `NewValidatedTokenV1`, `NewValidatedQC20TokenV1`
  or `NewValidatedQC721TokenV1`; invalid owners return an error and no object.
- Legacy constructors keep their read/metadata compatibility. An invalid owner
  cannot receive initial QC20 supply; the result retains its requested cap and
  owner metadata, but has zero supply and no balances. Controlled Token and QC721
  constructors do not issue anything; their mutation paths check addresses.
- Transfers, burns, delegated operations, mint recipients, authorization claims
  and persisted address maps use the same canonical predicate. Read-only lookups
  may return zero/not-found for an unknown string; they do not validate ownership.
- Valid existing addresses and keys are unchanged. No wallet migration or key
  replacement is performed by these checks.

Format validation does not prove possession of a private key. The separately
tracked legacy mint authorization and token network/instance-domain gaps remain;
this contract does not activate the Token Factory or a miner-token bridge.
