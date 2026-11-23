# QuantumCoin Query/Form Token Fallback Security

Document ID: `QC_QUERY_FORM_TOKEN_FALLBACK_CLOSED_FOR_PUBLIC_TESTNET_V1`\
Evidence stage: **Tested**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

The canonical node accepts API credentials in X-QC-API-KEY or
Authorization: Bearer in every network profile. The query string and request
body have no credential fallback. See [API token and listener policy](QC_PUBLIC_TESTNET_API_TOKEN_SECURITY.md).
