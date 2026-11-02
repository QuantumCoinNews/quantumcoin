# QuantumCoin Local Testnet Runbook - 24A

Status: QC_PRE_TESTNET_STEP_24A

Scope: LOCAL DEV TESTNET only.
Public testnet is disabled until later approval.

Fixed values:
- Network: testnet
- Network magic: QC_TESTNET_V1
- API port: 18082
- P2P port: 13001
- Chain file: chain_data_testnet.dat
- Bonus file: bonus_store_testnet.json
- Wallet file: wallet_data_testnet.json
- Total supply: 25,500,000 QC
- Faucet: disabled
- Premine: disabled

Required files:
- config.testnet.json
- RUN_TESTNET_API_18082.ps1
- STOP_TESTNET_API_18082.ps1
- RUN_TESTNET_NODE_18082_13001.ps1
- STOP_TESTNET_NODE_18082_13001.ps1
- TESTNET_DEV_MODE_BOUNDARY_23B.md
- CHECK_TESTNET_BOUNDARY_23B.ps1
- RUN_PUBLIC_TESTNET_NODE_DISABLED_UNTIL_APPROVAL.ps1

Start API-only local testnet:
Run from release folder: .\RUN_TESTNET_API_18082.ps1
Expected: TESTNET label, ok health, chain_data_testnet.dat, port 18082 listening.

Start full local testnet node:
Run from release folder: .\RUN_TESTNET_NODE_18082_13001.ps1
Expected: port 18082 API and port 13001 P2P listening.

Stop local testnet:
API-only stop: .\STOP_TESTNET_API_18082.ps1
Full node stop: .\STOP_TESTNET_NODE_18082_13001.ps1

Public testnet rule:
Do not run public testnet yet.
RUN_PUBLIC_TESTNET_NODE_DISABLED_UNTIL_APPROVAL.ps1 must refuse to start.

Safety rules:
Never touch mainnet chain file during testnet steps.
Mainnet file: chain_data.dat
Testnet file: chain_data_testnet.dat

Before public testnet:
- backup rollback plan
- endpoint allowlist
- token CORS policy
- faucet premine decision
- first local dry-run

Next checklist:
25 backup rollback plan
26 first local testnet dry-run
