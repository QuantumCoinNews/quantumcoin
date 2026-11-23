# QMA-0068 NFT award approval foundation V1

Status: implementation prepared; Windows verification pending.

## Previous verified checkpoint

QMA-0069: CLOSED / PASS on 2026-10-05. The automatic mining NFT entry points are retired. Missing blockchain and disabled mint cannot create DB reward records or Telegram earned-NFT notifications. The second legacy stub no longer claims an award.

Evidence: QMA-0069-miner-nft-result-20261005-003629-1009602.zip
SHA256: 3C66B2FB3A4F642CFF9EE25E0EE31522F10F7267505BC6DC71876D7617307B3E
Both RED markers were present before patching; GREEN and the five-package NFT/Monthly race suite passed (39 top-level tests), bonus retirement regressions passed, build passed, protected source hashes unchanged. The uploaded ZIP hash matched the console output.

QMA-0047 and QMA-0047-S1 remain closed; the annual 100 QC mining bonus is cancelled.

## New additive foundation

Four new Go files implement and test a standalone NFT award approval verifier.
No existing Go source is rewritten.

- A signature commits to its NFT-specific domain, NetworkID, legacy genesis hash and canonical NFT record digest.
- The blockchain wrapper recomputes the commitment from the actual record: recipient, metadata, campaign, period and serial are covered.
- An independent local NFT authority set and quorum are required. They are not inherited from hardfork governance and must not be obtained from an award or peer payload.
- Existing canonical public-key/quorum/signature routines are reused. Duplicate signers, unknown authorities, malformed signatures, missing quorum, changed records and replay across networks/genesis are rejected.
- Tests use temporary keys only. Test quorum values are not a real-world authority-policy decision.

This new functionality is additive, so this wave uses positive/negative capability tests, not an invented existing-code RED. The existing QMA-0069 RED/GREEN evidence is preserved above.

## Boundary and remaining work

This is not live NFT minting. The old MintNFT gate and NFT block authorization rejection remain unchanged. Existing NFT blocks, persistence, activation, QC balances, reward rules and Telegram logic are not changed.

Remaining QMA-0068 integration: bind canonical NFT trust to locally verified program configuration, carry approved awards in consensus-protected blocks, validate authorization at all entry points, validate campaign/period/uniqueness limits, persist/rebuild ownership and expose verified ownership. Standalone signatures do not prove Telegram eligibility or execute minting.

QMA-0066 remains open for the canonical monthly runtime flow. QMA-0070-0073 remain open for their separate selection, readiness, catalog and test scopes.