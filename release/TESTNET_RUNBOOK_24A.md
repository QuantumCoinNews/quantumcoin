# Local testnet runbook

Use the single runtime profile in config.testnet.json. The node loads it through
the strict Go config loader. The API script selects observer mode and the node
script selects writable run mode. Both delegate to START_TESTNET_LOCAL.ps1.

Run either script with -PlanOnly to see its config and storage paths.
For a real run, supply -ReleaseDirectory, -ReleaseCommit and -TrustedSigners
for a verified package, -DataDir for a separate existing testnet directory,
and QC_API_TOKEN through the process environment. The local P2P listener is
127.0.0.1:13001. The HTTP port is 18082. The chain file is
chain_data_testnet.dat. Stop the foreground process with Ctrl+C in its terminal.

The testnet profile does not start a miner. Public testnet activation is a
separate procedure. Follow the authenticated offline backup runbook for
snapshot and recovery. [Backup and recovery](../docs/QC_BACKUP_RESTORE_RUNBOOK.md)
