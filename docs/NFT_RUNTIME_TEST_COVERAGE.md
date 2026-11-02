# Runtime and NFT verification coverage

The integrated source passed 1,088 Go tests across 24 tested packages and
23 isolated Python regression scripts. The Python set reports 98 unittest cases
plus assertion-based scripts; these are different counting units. The prior full
Go race run also passed. A later tool-file rename is checked with the catalog
suite and does not alter runtime Go/Python reward logic.

Covered areas include consensus validation and signed activation, exact atom fees,
canonical burn accounting, founder binding, token authorization, durable writes
and restart/idempotency, API auth/CORS contracts, peer chain replacement, NFT
ownership and trusted reload, retired miner side effects, authenticated player
activity, one-use wallet challenges, immutable pools/receipts, fixed-round BLS
evidence, independent witness first-write behavior, lost publication replies,
final-content hash/approval binding and missing-receipt reconciliation.

`telegram_game/tests/test_nft_verified_lifecycle_v1.py` includes an isolated flow
against a real running Go test node. `test_nft_external_services_v1.py` covers
the witness/network and final preparation boundaries. Test keys, catalogs and
databases are temporary test inputs and do not authorize a live award.

Run `go test ./... -count=1` and the catalog test script for the core source and
catalog checks. `scripts/nft_regression_v2.py` runs isolated regression scripts
using `--output` and a previously generated Go test contract via `--contract`.
Source manifests and per-script logs are written to the selected output directory.
