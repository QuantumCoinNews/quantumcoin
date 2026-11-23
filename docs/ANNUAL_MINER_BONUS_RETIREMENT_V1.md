# QMA-0047 / QMA-0047-S1 annual mining bonus retirement

Decision date: 2026-10-04

The user cancelled the legacy annual 100 QC CMD mining bonus.
No bonus reserve, transfer, settlement engine or halving rule will be added for it.
Telegram monthly rare NFTs are a separate program.

Implementation scope:
- Retire the compatibility amount at constant zero.
- Ignore QC_ANNUAL_BONUS_QC; it cannot enable a payment.
- Remove the miner comment promising future annual settlement.
- Keep the compatibility hook inert and preserve its restart regression tests.
- Preserve historical records and existing generic record-only APIs.
- Preserve reward metadata serialization fields for compatibility; runtime annual bonus remains zero.

QMA-0047: closure pending implementation and verification.
QMA-0047-S1: feature cancelled by user; no annual mining bonus settlement is required.
QMA-0068: this change does not close the remaining NFT scope.
