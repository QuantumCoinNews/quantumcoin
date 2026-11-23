# Network label API and UI

Document ID: `QC_TESTNET_LABEL_UI_API_V1`\
Evidence stage: **Tested**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

The public `GET /api/network/label` returns the node's bound network identity
with `version`, `network`, `label`, `is_testnet` and `genesis_hash` when known.
It sends `Cache-Control: no-store` and the network, label and testnet headers.

The active embedded landing loads `/network-label.mjs`. It starts at
`UNVERIFIED` and renders a visible `TESTNET` badge when the label, network and
genesis fields agree. The separate React balance viewer reads the same endpoint.
The API obtains identity from the live runtime. See
[runtime identity](RUNTIME_CONFIG_AND_NETWORK_IDENTITY.md).

The embedded module test checks testnet, mainnet, missing genesis, inconsistent
labels and API failure. The root Go test verifies that a late environment
change leaves the running testnet label intact.
