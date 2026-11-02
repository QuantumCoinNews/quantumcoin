# Monthly NFT draft catalog

`nft/catalog/monthly_rare_nft_stock_2026.json` is the source of draft catalog
identity. The Go builder embeds this stock; the JSON plan and twelve draft
metadata files are checked against the same source. Draft readiness/mint flags
remain false. Stock/calendar labels do not determine a real program activation.

```powershell
python scripts/monthly_nft_catalog_v1.py --check
python scripts/test_monthly_nft_catalog_v1.py
```

Explicit regeneration uses `--write`; final image presence can be checked with
`--check --require-images`. This latter check is expected to fail while actual
artwork is absent. It is not a reason to insert test images as production assets.
The activation-derived `catalog-draft` admin route returns a draft without changing
the checked-in stock. See [final preparation](NFT_FINAL_PREPARATION.md).
