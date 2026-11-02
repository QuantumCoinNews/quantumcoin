# QuantumCoin Control Center Core Layout - 28C

Status: QC_DASHBOARD_STEP_28C_CORE_LAYOUT

Decision:
The dashboard will be organized around 4 main centers:

1. AI
2. Blockchain
3. Telegram Game
4. Wallet + Miner

Support area:
5. System / Ops / Future Modules

Reason:
The project already has many dashboards, panels and admin tools. The new Control Center should not become a random list of every file. It should show the whole ecosystem through major operational centers.

AI center:
- Security brain
- Risk score
- Alerts
- Admin recommendations
- Approved automation
- No consensus, supply, reward or balance mutation

Blockchain center:
- Mainnet API
- Local testnet API
- P2P
- Telemetry
- Block height
- Tx status
- Economy and burn policy visibility

Telegram Game center:
- Space Miner
- TGWT
- Player wallet mapping
- Withdraw/precheck
- Admin review
- Monthly NFT candidate status

Wallet + Miner center:
- Protected release GUI
- Wallet balance
- Spendable/locked/maturing
- Miner status
- Release binary hash reference
- No release GUI/miner modification without approval

System / Ops / Future Modules:
- Testnet runbook
- Backup / rollback
- Checkpoints
- Yellow warning cleanup
- Token Factory later
- NFT Factory later
- Autopublisher later

Token Factory decision:
Token Factory should not be a main dashboard center yet. It stays as a future module card until the foundation is created.

Dashboard v1 action policy:
- Read-only first
- No GitHub
- No real transfer
- No mining button
- No public testnet
- No private key display
- No release exe modification

Next step:
28D should create the actual Control Center shell using this 4+1 layout.
After dashboard shell: Token Factory foundation.
After that: yellow warning cleanup.
