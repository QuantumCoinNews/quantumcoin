# QMA-0068 NFT award evidence in block PoW

Status: PASS - proof block capability tests, NFT/Monthly race, annual bonus regressions, build and source preservation/scope. QMA-0068 remains OPEN for runtime admission and ownership.

## Verified preceding checkpoint

Local trust binding PASS on 2026-10-05:
QMA-0068-local-trust-result-20261005-193213-3595591.zip
SHA256 11E2C673995FACCDE9816F7D3D2463D43DD4DB18ED734692E5EFF229ABD9B79A
Eight local trust tests, 51 NFT/Monthly race tests, bonus retirement regressions and build passed. All recorded exit codes were zero. Blockchain changes were restricted to one private field and all 17 protected files were unchanged. Test counts are not NFT quantities.
QMA-0069 and QMA-0047/S1 remain CLOSED / PASS (annual 100 QC feature cancelled).

## This wave

Adds a public proof field to Block: decision and approvals only, never trusted authority configuration. The ordered record/proof commitment binds the canonical decision digest and canonical signer/signature bytes with explicit lengths and a separate domain.
V4 PoW uses this combined commitment when proofs are present. Proofless blocks preserve their old record-only digest; no activation policy or deployed historical protocol is changed.
A detached V4 construction helper places evidence into the block before mining and deep-copies inputs. It does not append a block, validate the local signer set, activate V4, issue an NFT or update ownership.
Block clone/snapshot paths deep-copy proof slices, signer keys and signatures. Gob carries public evidence but cannot carry private local trust.
Authorization and file-load guards also reject proof-only/orphan data, so a newly added field cannot bypass an old empty-record shortcut. All nonempty NFT data still fails closed at live admission, including structurally valid signed proofs.
Five existing Go files are patched with exact anchors and hash guards; three Go files are added. Test helpers use ephemeral authorities, never real production private keys.
These are additive capability and regression tests, not a fabricated claim that this new schema was exploitable in the old code.

## Product rule and next integration

Telegram Game awards one rare NFT per month to one random active verified user for 12 months, total 12. AI review and admin approval remain part of the intended flow. No fixed mainnet/calendar launch is set. A signature or PoW commitment does not prove user eligibility or the randomness of selection.

Still open: trusted program/bootstrap configuration, activation/reload policy, actual approved record admission across mining/peer/replacement/load, month/campaign/uniqueness/12-award enforcement, persisted and rebuilt ownership, runtime notification after canonical settlement, catalog and selection scopes.
QMA-0068 stays OPEN. QMA-0066 and QMA-0070-0073 are not closed by this wave.