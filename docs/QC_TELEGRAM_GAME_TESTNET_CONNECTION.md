# QuantumCoin Telegram Game Testnet Connection

Status: QC_TG_TESTNET_CONNECTION_V1

Date: 20260618_203750

Telegram Game must explicitly separate local/main QuantumCoin API connection from testnet connection.

## Rules

- Local/default Telegram Game uses http://127.0.0.1:8082.
- Testnet Telegram Game uses http://127.0.0.1:18082.
- Testnet mode requires explicit env:
  - TELEGRAM_GAME_CHAIN_MODE=testnet
  - QC_TESTNET=1
  - QC_NETWORK=testnet
  - QC_CHAIN_MODE=testnet
  - QC_NET_MAGIC=QC_TESTNET_V1
- Testnet mode rejects local/main QC API ports 8081/8082.
- Local/main mode rejects testnet QC API port 18082.
- Testnet runner uses TELEGRAM_GAME_MINING_MODE=TESTNET_SIMULATED.
- Telegram Game reads QuantumCoin network label via /api/network/label later where runtime checks need it.

## Safety boundary

This does not mine, transfer, mint, settle, select NFT winners, modify TGWT balances, change QC balances, change consensus, change rewards, or change total supply.

## Files

- telegram_game/telegram_game/testnet_connection_v1.py
- telegram_game/telegram_game/testnet_connection_v1_test.py
- telegram_game/telegram_game/config.py
- telegram_game/telegram_game/dev_api.py
- telegram_game/RUN_TELEGRAM_GAME_SERVER_8080_TESTNET.ps1
- telegram_game/RUN_TELEGRAM_GAME_SERVER_8080_TESTNET_SIM.ps1
- telegram_game/RUN_TELEGRAM_GAME_SERVER_8080_TESTNET_SIM_NORELOAD.ps1
- telegram_game/.env.example
