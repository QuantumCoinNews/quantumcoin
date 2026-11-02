# QuantumCoin Control Center Core Layout - 28D

Status: QC_DASHBOARD_STEP_28D_SECURITY_MONITORING_LAYOUT

Decision:
The dashboard will be organized around 6 main centers plus future modules.

Main centers:
1. AI
2. Blockchain
3. Security
4. Monitoring / Ops
5. Telegram Game
6. Wallet + Miner

Support:
7. Future Modules

Why Security is separate:
AI interprets risk and gives recommendations. Security owns guardrails, alerts, admin reports, suspicious action visibility, protected-file warnings and yellow warning cleanup.

Monitoring folder purpose:
Monitoring is not the dashboard itself. It is the operational observation layer: telemetry, event logs, alert runbooks, health timeline, backup/rollback status and next-action queue. It feeds Security and AI.

AI center:
- Risk interpretation
- Recommendations
- Anomaly explanation
- Approved automation suggestions
- No consensus/supply/balance mutation

Blockchain center:
- Mainnet API
- Local testnet API
- P2P
- Telemetry
- Block height
- Tx status
- Supply/economy/burn visibility

Security center:
- Admin alerts
- Admin reports
- Guardrails
- Suspicious actions
- Protected file status
- Public exposure warnings
- Yellow warnings

Monitoring / Ops center:
- Event logs
- Alert runbooks
- Backup / rollback
- Checkpoints
- Health timeline
- Next-action queue

Telegram Game center:
- Space Miner
- TGWT
- Player wallet mapping
- Withdraw/precheck
- Admin review
- Monthly NFT candidate status

Wallet + Miner center:
- Protected release GUI
- Balance
- Spendable/locked/maturing
- Miner status
- Release binary hash reference
- No private key display

Future modules:
- Token Factory later
- NFT Factory later
- Autopublisher later
- Public testnet readiness
- Mainnet readiness

Dashboard v1 action policy:
- Read-only first
- No GitHub
- No real transfer
- No mining button
- No public testnet
- No private key display
- No release exe modification
- No admin mutation without explicit approval

Next:
28E should create the actual Control Center shell using this 6+1 layout.
Then Token Factory foundation.
Then yellow warning cleanup.
