# Network genesis and emission

The existing height-based subsidy uses whole QC with a one-QC floor. At a
30-second target interval and a 25.5 million QC cap, a genesis issuing
1,275,050 QC leaves 24,224,950 QC for mining. Subsidy ends at height 484,499,
about 168.23 target days after genesis. The first halving would be at height
2,102,401. The runtime policy snapshot exposes the projected last subsidy
height and target elapsed time for its configured genesis. Actual elapsed time
depends on block production.

The active policy sends the subsidy and verified fees to the miner. Historical
500,000 QC bonus / 100,000 QC NFT reserve proposals are not funded pools in the
current canonical consensus. There is no date-based subsidy expiry. A future fee-only mining
era needs its own consensus and mining-template rules.

## Fresh network profiles

For a fresh testnet or mainnet, use a separate, empty data directory with the
matching network settings and add `"genesis_profile": "network-v4"` to its
`config.json`. Agree on all founder allocation inputs before generating the
first block. The profile commits the network to the genesis transaction tree
with a zero-value marker. The founder outpoints and P2P handshake magic are
also separated between testnet and mainnet. Local and public testnet modes
share the testnet chain when their genesis inputs match.

Changing the setting on an existing data directory causes startup to reject
the old genesis. An operator can retain an existing chain with its original
profile and directory. The previously issued 50 QC at the legacy genesis
remains part of that chain's history.

The fresh `network-v4` profile activates genesis-bound transaction signatures
from the first spend. Its digest is SHA-256 of the domain
`QC_TX_SIG_NETWORK_V2\0`, the trusted genesis hash and the deterministic input
digest. Fresh V4 transaction content uses fixed-format QC_TX_CONTENT_V2 and
QC_TX_INPUT_SIG_V2 encoding, never process-local Gob type IDs. It inherits V3 median-time/difficulty rules. Testnet and mainnet have
distinct V4 genesis markers; local and public modes of the same testnet share
the same genesis and signing scope.

The genesis profile version is separate from Atom V3 / NFT V4 transaction and
reward protocols. Selecting this signing profile does not activate those
features or change the whole-QC subsidy series, fees, allocations or addresses.

CLI send, native-wallet local signing, HTTP signed submission, peer admission,
mining, replacement validation and trusted disk replay all use the chain-aware
signature check. A V4 verifier never retries a failed signature with the legacy
digest. Transaction fields and 32-byte ID size are unchanged. Fresh V4 uses the
deterministic V2 codec; older profiles retain historical Gob hashing. Gob is
still a storage/wire format but is outside fresh-network consensus hashing.
Chain context selects one codec; there is no verification fallback. A
transaction cannot override its signing domain. Previous pre-launch V4
candidate chains must not be reused as this finalized testnet genesis.

Public testnet and new mainnet startup require this V4 profile. Older empty/V2/V3 profiles keep
their historical signature verification only for explicit legacy/local use.
Do not change the profile in an existing directory to migrate history: startup
rejects the genesis mismatch and preserves the file. Keep the original config
and directory for legacy inspection and use a separate fresh V4 directory.
Wallet addresses, keys and recovery words are unchanged; spendable balances
belong to the chosen chain. Mainnet rollout and historical-chain migration
remain separate release/transition decisions.

The legacy `Transaction.Sign`, `Transaction.Verify` and `SigningHashes` APIs
remain for historical byte compatibility. New integrations must use the local
trusted chain's `SignTransaction`, `VerifyTransaction` and
`TransactionSigningHashes`. Supplying an old signature through a legacy helper
does not bypass canonical V4 admission. The native GUI's trusted local signer
loads the same genesis configuration; the game bridge forwards public signed
transactions unchanged and receives no wallet keys.

## Median time profile

Fresh official networks use `network-v4`, which inherits the median-time
rules introduced by `network-v3`. The older V3 profile remains only for
historical local inspection. A candidate timestamp must exceed the
median of up to eleven preceding blocks. Retarget uses two eleven-block
medians spaced one hundred blocks apart and starts when both windows are
available. The required difficulty is checked in peer admission, disk replay,
and local or external mining. Existing genesis profiles retain their
historical time rules. Runtime startup allows fixed difficulty only with the
isolated local chain profile.

For fresh V4, the first complete pair of eleven-block median windows exists
after height 110. The candidate at height 111 uses the first retarget result.
This is a rolling 100-block lookback evaluated for each next block, not a
change restricted to every 100th height. Against the 30-second target,
an elapsed median span below 80% raises difficulty by one bit; above 120%
lowers it by one bit. Equality at either threshold keeps the current value.
The floor is 8 bits for testnet and 16 for mainnet; the ceiling is 255.
These are existing rules, not new emission or reward decisions. Actual block
pace depends on hashrate and will be measured during testnet; a target is not
a promise that blocks always arrive exactly 30 seconds apart.

Fresh-network acceptance exercises valid-PoW blocks with invalid difficulty,
MTP equality/backdating and future clock drift through peer, replacement and
disk replay. Two independent testnet chain instances agree across retarget
and restart. A mainnet startup test rejects a mined V4 testnet file even when
renamed, preserves that file and starts from genesis in a separate empty
location. No legacy-chain rule migration is required for the approved fresh
launches; historical profiles retain their original validation rules.

## Amount precision

The signed Atom V3 and NFT V4 protocol paths use six decimal places and
represent fees of 0.000001 QC through 0.000010 QC exactly. Historical V1 and
continuity V2 amounts use whole QC. The eight-decimal 5.79982325 QC subsidy
cannot be represented at the current atom scale. The active subsidy series
and miner coinbase allocation remain authoritative.

## Chain-file format

The node's next successful save wraps the existing Gob payload in a
versioned disk header with network, genesis, chain ID, payload size and
SHA-256 checksum. The runtime validates the framing, then replays the chain
against its trusted genesis and consensus configuration. Legacy files still
use trusted replay and can be read before the first upgraded save. Retain an
earlier verified backup for rollback to binaries that read bare Gob files.
Peer chain messages keep their separate wire encoding.
