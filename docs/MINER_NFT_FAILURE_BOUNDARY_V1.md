# QMA-0069 miner NFT failure boundary

Verification: PASS on 2026-10-05. QMA-0069 CLOSED / PASS; RED/GREEN, NFT/Monthly race, bonus retirement regressions, build and source preservation verified.

QMA-0069: mining NFT compatibility calls must never create a DB award record or Telegram earned-NFT notification when no NFT is minted. The legacy automatic mining program is retired; ENV flags cannot restore it.

Two preserved public entry points, GrantNFTBonus and GrantNFTReward, report that mining awards are disabled. The hook declarations and NFTMiningReward type remain for source compatibility, but these entry points never invoke hooks, mint, change chain metadata or claim an award.

Tests cover both legacy ENV flags, nil blockchain, disabled chain mint and the old success-claiming stub.

This does not cancel the Telegram monthly rare NFT program (one verified active winner per month, twelve NFTs across the program). It does not enable a live mint or introduce QC rewards.

QMA-0068 remains open: authorized consensus-protected mint and ownership remain to be implemented.
QMA-0066 remains open: the canonical monthly runtime architecture is not yet integrated.
QMA-0073 receives evidence for NFT failure behavior; its entire test scope does not close here.