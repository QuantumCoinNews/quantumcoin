# QuantumCoin Source Clean Standard

This project separates source code, runtime state, release artifacts, secrets, and audit outputs.

## Source may contain

- Go source files
- Python source files
- TypeScript/JavaScript source files
- Config examples
- Documentation
- Test files
- Safe scripts

## Source must not contain

- Wallet private keys
- Recovery words
- Real .env files
- OAuth tokens
- Bot tokens
- Chain runtime data
- Local DB files
- Cache files
- Logs
- Old executable backups
- Generated release packages
- Testnet backup state
- Media render outputs

## Release package rule

A user release package must be assembled separately from source and must contain only approved runtime files.

Expected minimal release candidate:

- quantumcoin.exe
- wallet-gui.exe
- run_miner.cmd
- _qc_api_mempool_miner.ps1
- config.json or approved runtime config
- assets/
- README.md

No private key, wallet file, recovery file, chain data, DB, token, secret, backup exe, or log file is allowed in release.
