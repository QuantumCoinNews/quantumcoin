# Permanent burn and supply accounting

`QC_BURN_SINK` is a consensus label, not a normal wallet address. Burn outputs
use the dedicated marker returned by `NewCanonicalBurnOutputV1`. The marker is
not a wallet key hash, output-key matching refuses it, and canonical UTXO
validation explicitly rejects spending it. Security does not depend on assuming
that nobody knows a private key for a normal address. Returned marker bytes are
detached from the canonical marker.

A burn must still spend the owner's signed, valid, mature inputs and obey burn
limits. It is not an administrative balance deletion. Fees remain transferred
value and must not be counted as new mint or as the burned amount.

Issued supply and the hard cap do not decrease or reopen when coins are burned.
Circulating supply in canonical atoms is issued QC converted to atoms minus
confirmed canonical burned atoms. Legacy and atom-protocol outputs use their
containing block's denomination. Overflow, invalid units or burned supply above
issued supply are errors. There is no replacement subsidy or recycling of the
unspendable sink. Recycling proposals remain separately disabled.

Accounting derives from the current validated canonical chain, not a mutable
burn total or a client claim. Pending burns do not reduce circulation. An orphan
burn leaves accounting when a valid higher-work branch replaces it; trusted
restart replays the adopted chain. An orphan may only count again if legitimately
confirmed on the new canonical history.

Acceptance includes signed-owner/wrong-owner admission, explicit burn-output
spend rejection, protocol units and limit/overflow tests, actual burn-plus-fee
peer/reload tests, root wallet HTTP signing contracts, and a real mined fork
replacement with post-reorg disk replay. Accounting methods use existing
canonical runtime synchronization; they are not a second monetary state store.
