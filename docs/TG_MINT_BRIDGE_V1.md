# QMA-0066 Telegram -> canonical node mint bridge V1

Date: 2026-10-19. Scope: public approved award transport, canonical receipt
verification, and atomic game-side confirmation. QMA-0066 remains OPEN pending
the remaining selection/catalog/readiness integration and Windows verification.
QMA-0067/0068/0069 retain their accepted, bounded closures.

The existing authenticated `/api/admin/nft/monthly/controlled-mint` route accepts
an optional `signedAward` object with the node's existing public request schema.
Neither private keys nor authority settings are accepted. The node still checks
its independently configured signers, quorum, activation and monthly limits.

## State and retries

1. Installation leaves activation and environment settings untouched. Both
   `QC_ENABLE_CHAIN_NFT_MINT=1` and `QC_ENABLE_TG_NFT_MINT_BRIDGE=1`, explicit
   mainnet game connection mode, a verified active node program and final server
   catalog are prerequisites. This package does not choose launch dates/keys.
2. The selected candidate must retain approved AI/admin states, a server-verified
   session in the award period and a unique linked Base58Check wallet. The
   signed record must match that wallet, program, period/serial and metadata.
   This binds the existing player record; it does not prove wallet possession,
   implement Sybil resistance, or replace QMA-0070's fair winner selection.
3. All 12 server-owned stock entries must match the program's 12 periods and
   contain final flags, unique final hashes and matching physical asset files.
   The current placeholder stock fails this check. No request flag can make it
   final. QMA-0071/0072 still require their full catalog/readiness work.
4. A `PENDING` SQLite intent is committed before HTTP. It is not an NFT award.
   One intent is permitted per network/month and per candidate. A different
   record/recipient/catalog binding cannot overwrite it.
5. Query the signed receipt first. Only an authenticated 404 permits one mint
   POST. A lost/malformed mint response causes receipt lookup, not another mint
   POST. If confirmation is unavailable, retain the pending intent and leave
   the candidate `NOT_MINTED`. An operator retry uses the same signed award.
6. Accept success only for the exact canonical token/record digest, recipient,
   metadata, genesis, program and period, with `persisted=true` and consistent
   mint/chain/file heights and commitments. The node checks both canonical RAM
   state and its configured persisted file. Redirects never forward credentials.
7. Commit the public receipt and `MINTED_CONFIRMED` candidate state in one SQLite
   transaction after rechecking the candidate binding. A DB failure after node
   success is recovered by receipt lookup on retry. Previously confirmed awards
   are never automatically minted again when a later lookup is unavailable.

These are point-in-time canonical confirmations, not finality guarantees. A
reorg or changed local candidate/catalog binding requires reconciliation; the
bridge does not infer a new recipient or silently remint. The node's ordinary
canonical block reward/fee path remains responsible for block economics; the
game bridge does not award additional QC/TGWT or claim that mining is balance-
neutral. No Telegram earned-NFT notification is added in this step.

## Verification

`TestNFTGameMintClientContractV1` emits the real node's request, persisted receipt,
verified program and known metadata bytes. It exports public test evidence only.
The Python tests consume those exact bytes, use temporary SQLite databases and
loopback HTTP servers, and check loss/retry, malformed receipt, binding changes,
draft/tampered catalogs, DB rollback, concurrent workers, admin authentication
and closed-by-default behavior. Existing session/identity/activation tests remain
part of the installer verification.

The installer backs up source files, checks their known hashes, installs the
bounded patch and runs the focused Go race contract tests plus Python scripts.
It restores source files on a failed verification when their hashes are still
recognized. Runtime configuration, chain data, keys and Git remotes are untouched.
Apply does not restart services, enable minting, run a real award or push GitHub.
