# Policy evidence register

A document ID ending in V1 identifies a revision of a policy or runbook.
Evidence stage describes the strongest evidence for the scoped behavior below.

| Stage | Evidence required |
| --- | --- |
| Design | Written rule, plan, checklist or runbook. |
| Implementation | Source or configuration implements the scoped behavior. |
| Partial implementation | Source covers part of the policy and named work remains. |
| Runtime wired | A canonical launch path, route or runtime import reaches the behavior. |
| Tested | A named automated test exercises the behavior. A source fixture has its own limits. |
| Source verification | An audit, dependency graph or build verifies a source/layout requirement. |
| Release active | Matching release commit, artifact, configuration and observed behavior, supported by an independently trusted manifest. |

Release activity is **unverified for every row in this register**. Local source,
tests, a fixture signature and a deployment are separate evidence. When promoting
a row to release active, record the release commit, artifact digest, profile and
check result. Existing per-item audit reports preserve their dated findings.

| Document | Evidence stage | Source or check | Scope |
| --- | --- | --- | --- |
| [Active Source Standard](QC_ACTIVE_SOURCE_STANDARD.md) | Source verification | [tools/qc_active_source_standard_check.ps1](../tools/qc_active_source_standard_check.ps1) | Source classification only. |
| [Admin Panel Auth Pagination](QC_ADMIN_PANEL_AUTH_PAGINATION.md) | Runtime wired | [telegram_game/telegram_game/dev_api.py](../telegram_game/telegram_game/dev_api.py) | Admin route authentication and page bounds. |
| [Backup Location Policy](QC_BACKUP_LOCATION_POLICY.md) | Design | [docs/QC_BACKUP_RESTORE_RUNBOOK.md](../docs/QC_BACKUP_RESTORE_RUNBOOK.md) | Off-host placement and operational drill require operator evidence. |
| [Blockchain Package Standard](QC_BLOCKCHAIN_PACKAGE_STANDARD.md) | Design | [blockchain/](../blockchain/) | Source-layout rule. Use the Go import graph for current runtime ownership. |
| [Burn Sink Control](QC_BURN_SINK_CONTROL.md) | Tested | [blockchain/burn_consensus_v1_test.go](../blockchain/burn_consensus_v1_test.go) | Canonical burn output and spend restrictions. |
| [Chain File Separation](QC_CHAIN_FILE_SEPARATION.md) | Tested | [node/network_identity_startup_test.go](../node/network_identity_startup_test.go) | Runtime identity and legacy/mainnet separation. Historical origin remains a separate custody check. |
| [Db Persistence Plan](QC_DB_PERSISTENCE_PLAN.md) | Design | [telegram_game/telegram_game/dev_api.py](../telegram_game/telegram_game/dev_api.py) | Persistence plan visibility is separate from the active SQLite implementation. |
| [Dev Miner Endpoint Lock](QC_DEV_MINER_ENDPOINT_LOCK.md) | Tested | [dev_miner_endpoint_lock_v1_test.go](../dev_miner_endpoint_lock_v1_test.go) | Root dev/miner gate. The former standalone Go HTTP API is retired. |
| [Founder Vesting Policy](QC_FOUNDER_VESTING_POLICY.md) | Partial implementation | [foundervesting/](../foundervesting/) | Policy and runtime checks exist. Final address and launch evidence are separate. |
| [Go Build Targets](QC_GO_BUILD_TARGETS.md) | Source verification | [scripts/check_headless_build_matrix.ps1](../scripts/check_headless_build_matrix.ps1) | Headless build matrix. Native targets are a separate build boundary. |
| [Hardfork Policy](QC_HARDFORK_POLICY.md) | Tested | [docs/HARDFORK_ACTIVATION_VERIFICATION.md](../docs/HARDFORK_ACTIVATION_VERIFICATION.md) | Verified activation mechanics. Live activation requires separate evidence. |
| [Invalid Address Panic Hardening](QC_INVALID_ADDRESS_PANIC_HARDENING.md) | Tested | [address_api_contract_test.go](../address_api_contract_test.go) | Address handling at API boundaries. |
| [Legacy Noise Separation](QC_LEGACY_NOISE_SEPARATION.md) | Design | [docs/QC_ACTIVE_SOURCE_STANDARD.md](../docs/QC_ACTIVE_SOURCE_STANDARD.md) | Repository hygiene rule. The current inventory is tracked separately. |
| [Load Test Scripts](QC_LOAD_TEST_SCRIPTS.md) | Tested | [scripts/test_operator_tools.py](../scripts/test_operator_tools.py) | Plan and real loopback HTTP fixture, with safety limits. |
| [Mainnet Before Economy Control](QC_MAINNET_BEFORE_ECONOMY_CONTROL.md) | Partial implementation | [economycontrol/](../economycontrol/) | Preflight policy and gates. Final economy activation remains separate. |
| [Main Entrypoints](QC_MAIN_ENTRYPOINTS.md) | Source verification | [docs/DEPENDENCY_BOUNDARIES_AND_AUDIT.md](../docs/DEPENDENCY_BOUNDARIES_AND_AUDIT.md) | Entrypoint and target classification. |
| [Mining Node Transition](QC_MINING_NODE_TRANSITION.md) | Implementation | [miningtransition/](../miningtransition/) | Transition policy and tests. Current reward schedule is owned by the consensus runtime. |
| [Monitoring Event Log Alert Runbook](QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK.md) | Runtime wired | [docs/MONITORING_AND_LIVE_SECURITY.md](../docs/MONITORING_AND_LIVE_SECURITY.md) | Node event/alert lifecycle and published runbook are separate concerns. |
| [Monthly Nft Placeholder](QC_MONTHLY_NFT_PLACEHOLDER.md) | Implementation | [docs/NFT_CATALOG.md](../docs/NFT_CATALOG.md) | Catalog preparation. Final artwork and approval remain pending. |
| [Monthly Nft Stock Pre Mainnet](QC_MONTHLY_NFT_STOCK_PRE_MAINNET.md) | Implementation | [docs/NFT_FINAL_PREPARATION.md](../docs/NFT_FINAL_PREPARATION.md) | Draft stock and readiness checks. Final product assets remain pending. |
| [Network Fee Policy](QC_NETWORK_FEE_POLICY.md) | Tested | [docs/DENOMINATION_AND_FEE_RUNTIME.md](../docs/DENOMINATION_AND_FEE_RUNTIME.md) | Atom precision and fee enforcement. Use current runtime documentation for activation. |
| [P2P Message Size Security](QC_P2P_MESSAGE_SIZE_SECURITY.md) | Runtime wired | [p2p/](../p2p/) | Message limits on the live peer receive path. |
| [Peer Block Validation](QC_PEER_BLOCK_VALIDATION.md) | Runtime wired | [p2p/](../p2p/) | Peer handling delegates to the blockchain validation boundary. |
| [Peer Transaction Validation](QC_PEER_TRANSACTION_VALIDATION.md) | Runtime wired | [p2p/](../p2p/) | Peer transactions enter canonical validation. |
| [Post Checklist Unified Dashboard Backlog](QC_POST_CHECKLIST_UNIFIED_DASHBOARD_BACKLOG.md) | Design | [docs/QC_POST_CHECKLIST_UNIFIED_DASHBOARD_BACKLOG.md](../docs/QC_POST_CHECKLIST_UNIFIED_DASHBOARD_BACKLOG.md) | Backlog for a unified dashboard. |
| [Query Token Fallback Security](QC_QUERY_TOKEN_FALLBACK_SECURITY.md) | Tested | [actual_mux_integration_test.go](../actual_mux_integration_test.go) | Header-only auth in all modes, superseding the earlier narrower public-testnet note. |
| [Queue Worker Integration Plan](QC_QUEUE_WORKER_INTEGRATION_PLAN.md) | Design | [telegram_game/telegram_game/dev_api.py](../telegram_game/telegram_game/dev_api.py) | Plan endpoint describes queues. A plan response does not start a queue worker. |
| [Replacechain Security](QC_REPLACECHAIN_SECURITY.md) | Tested | [blockchain/chain_replacement_validation_v1_test.go](../blockchain/chain_replacement_validation_v1_test.go) | Trusted replacement validation and genesis boundary. |
| [Telegram Game Testnet Connection](QC_TELEGRAM_GAME_TESTNET_CONNECTION.md) | Implementation | [telegram_game/telegram_game/testnet_connection_v1.py](../telegram_game/telegram_game/testnet_connection_v1.py) | Connection policy and bridge code. Deployment evidence is separate. |
| [Testnet Config Skeleton](QC_TESTNET_CONFIG_SKELETON.md) | Implementation | [config/testnet_config_skeleton_v1.go](../config/testnet_config_skeleton_v1.go) | Profile skeleton and loader, with separate activation. |
| [Testnet Faucet Premine Security](QC_TESTNET_FAUCET_PREMINE_SECURITY.md) | Runtime wired | [config/testnet_faucet_premine_safety_v1.go](../config/testnet_faucet_premine_safety_v1.go) | Runtime profile guard. Faucet enablement is a separate decision. |
| [Testnet Label Ui Api](QC_TESTNET_LABEL_UI_API.md) | Tested | [webui/network-label.test.mjs](../webui/network-label.test.mjs) | Root API and embedded badge behavior. |
| [Tgwt Bsc Claim Mainnet Testnet Lock](QC_TGWT_BSC_CLAIM_MAINNET_TESTNET_LOCK.md) | Partial implementation | [telegram_game/telegram_game/dev_api.py](../telegram_game/telegram_game/dev_api.py) | Ledger and withdrawal gates. Live BSC payout verification remains pending. |
| [Watch Earn Testnet Final Lock](QC_WATCH_EARN_TESTNET_FINAL_LOCK.md) | Implementation | [telegram_game/telegram_game/dev_api.py](../telegram_game/telegram_game/dev_api.py) | Claim guards. Live provider verification requires separate evidence. |
| [Webui Embed Dist Standard](QC_WEBUI_EMBED_DIST_STANDARD.md) | Tested | [webui/handler_test.go](../webui/handler_test.go) | Embedded asset delivery and the separate WebWallet build contract. |
