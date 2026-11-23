# QMA-0072: Monthly NFT draft catalogue consistency

The authoritative identity source is
`nft/catalog/monthly_rare_nft_stock_2026.json`. The Go placeholder builder embeds
this stock instead of maintaining a second list of names and concepts. The
existing stock/JSON assignment is preserved: Quantum Lunar Pioneer is June.
Invalid stock identity, ordering, uniqueness or paths cannot produce a valid
Go placeholder plan. All builder mint/preparation flags remain false.

`scripts/qma_monthly_nft_catalog_v1.py` derives the checked-in JSON plan and all
twelve draft metadata documents from this stock. Existing JSON field order and
formatting are immaterial to checks. The existing June metadata bytes are kept
when its contents already match. The single-item `rare_nft_catalog.json` is a
preview subset, not an alternative twelve-month award schedule; its identity
and metadata reference are checked against the stock.

Run from the repository root:

```powershell
python scripts/qma_monthly_nft_catalog_v1.py --check
python scripts/test_qma_monthly_nft_catalog_v1.py
go test ./nft -race -count=1
```

The default operation is read-only and fails on missing metadata or drift.
After intentionally editing the canonical draft stock, explicitly regenerate:

```powershell
python scripts/qma_monthly_nft_catalog_v1.py --write
```

The generator validates all existing outputs before writing any. It refuses
stock with mint/activation state, final hashes or completed approval stages,
and refuses to overwrite final/unknown metadata or a non-placeholder plan.
Draft writes use closed temporary files and atomic replacement. It neither
creates images nor produces final metadata hashes, signatures or approvals.

For explicit artwork file verification:

```powershell
python scripts/qma_monthly_nft_catalog_v1.py --check --require-images
```

This stricter check fails if any of the twelve declared image files is absent.
The normal draft check reports that image verification was not requested;
successful draft validation must never be interpreted as artwork readiness.
Stock `finalReady` remains false, hashes remain empty, AI/admin review remains
pending, and mainnet/testnet mint activation remains disabled. Final artwork,
approval and mint-readiness completion remain QMA-0071 work. This change does
not close QMA-0066, QMA-0070 or the broader QMA-0073 test scope.

The year-2026 stock is a draft inventory. It does not set the game activation
date or substitute for the verified node programme's twelve full UTC months.
Projecting the Go placeholder builder to another year only changes draft IDs
and paths; it does not create or activate an award programme. The rule remains
one NFT per month for twelve months, twelve NFTs total.

Regression evidence includes Go/JSON/stock equality, all twelve metadata
identities and disabled boundaries, preview consistency, malformed templates,
read-only missing-file detection, deliberate drift, generation idempotence,
typed JSON booleans, refusing final documents, and required missing artwork.
The original hard-coded Go builder must fail the new catalogue consistency
test, while the corrected builder passes. Windows installation is accepted
only after the guarded Apply report confirms tests and source hashes.
