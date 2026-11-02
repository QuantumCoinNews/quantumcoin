# QuantumCoin WebUI Embed / Dist Standard

Status: QC_WEBUI_EMBED_DIST_STANDARD_V1

This document defines how WebUI files are checked before testnet preparation.

## Purpose

The WebUI must not be ambiguous before testnet.

We need to know:

- where the WebUI source lives
- whether Go uses //go:embed
- whether embedded files exist
- whether root index.html is active or legacy/static
- whether go build . works without touching release outputs

## Expected safe rule

- webui/ can be active source if it is imported by root main.go or a Go package.
- webwallet/ can be active source if it is maintained separately.
- root index.html must be classified later if it is not used by embed/runtime.
- dist/build/generated folders must not be confused with source.
- audit builds must go only to _local_audit_reports.
- release/wallet-gui.exe and release/quantumcoin.exe must not be overwritten.

## Testnet rule

Before public testnet:

- WebUI static serving must be explicit.
- Missing embed/dist files must fail the check.
- WebUI must clearly show TESTNET later in item 21.
