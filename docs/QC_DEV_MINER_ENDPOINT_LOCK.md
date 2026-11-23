# QuantumCoin Dev / Miner Endpoint Lock

Document ID: `QC_DEV_MINER_ENDPOINT_LOCK_V1`\
Evidence stage: **Tested**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

The canonical node registers the dev fast-mine and miner-control endpoints.
They require explicit development mode, a numeric loopback listener, a direct
loopback caller and header authentication. The request guard checks local Host
and browser origin metadata and rejects forwarding headers. Public-testnet
switches disable access even with valid credentials.

The standalone Go API executable has been retired. See
[API token and listener policy](QC_PUBLIC_TESTNET_API_TOKEN_SECURITY.md).
