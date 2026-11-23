# Root runtime policy package decisions

Classification follows the current root Go dependency graph. Runtime policy
validation, diagnostics, and activation carry separate evidence. The named
commands and sources identify the owner for each package.

| Package | Decision | Evidence and boundary |
| --- | --- | --- |
| backuprestore | Integrate as an offline operator tool | cmd/recovery uses backuprestore/offline. It creates and verifies backups without attaching recovery operations to the running node. |
| burncontrol | Integrate as policy and preflight | node/runtime_policy_gate_v1.go checks the declaration. Consensus burn acceptance and accounting stay in blockchain. Recycling remains a separate disabled policy. |
| economycontrol | Integrate as startup preflight | node/runtime_policy_gate_v1.go checks the launch contract. Consensus owns reward and supply state. |
| foundervesting | Integrate as startup policy | node/runtime_policy_gate_v1.go imports policy. Actual block and spend validation remains in blockchain. |
| hardforkpolicy | Integrate as activation policy | node startup and consensus/hardfork_activation_v1.go use it. Verified activation still requires signed release evidence. |
| i18n | Archive as unwired support | The root graph has no import. Current desktop translations are owned by their client. Retain only for historical comparison until a client migration is designed. |
| miningtransition | Redesign before integration | policyexamples exports a proposal. The canonical blockchain reward schedule owns runtime transitions. |
| monitoring | Integrate in node runtime | node.Runtime owns the live engine, HTTP observation, persistence and lifecycle. The loopback operator dashboard reads it. |
| networkfee | Integrate as policy and startup guard | node/runtime_policy_gate_v1.go checks policy. Canonical atom accounting and activation run through blockchain and consensus. |
| nft | Archive legacy placeholder, keep catalog utilities separate | The root graph has no import. Consensus and blockchain own accepted NFT records and ownership. Catalog and review helpers do not authorize minting. |
| persistence | Archive standalone abstractions pending an explicit migration | Node chain persistence uses blockchain storage. Telegram uses its Python SQLite policy. These Go utilities do not own either live store. |
| queueworker | Retain as standalone preparation; preserve service ownership | No root/node/recovery worker starts. Telegram owns its review/ledger operations, Autopublisher its scheduler/jobs, node its monitoring, recovery its offline commands. See [queue ownership](QUEUE_RUNTIME_OWNERSHIP.md); future central integration requires an explicit migration contract. |
| token | Redesign before activation | QC20/QC721 and Token Factory preparation have no root runtime activation. Future issuance and mining require consensus and deployment approval. |

Verify the graph with go list -deps . from the repository root. The separate
release, recovery and policy export tools have their own dependency graphs.

The token review and approval-dependent completion sequence are documented in
[Token Factory status and completion plan](TOKEN_FACTORY_STATUS_AND_COMPLETION_PLAN.md).
