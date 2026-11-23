# Local testnet boundary

config.testnet.json supplies the storage and network ports to the node config
loader. The shared launcher clears inherited QC runtime overrides in the child
environment, retains the API credential, and sets the local testnet identity.
The foreground observer has a read-only API. The writable local node binds its
P2P listener to loopback. See the active [testnet runbook](TESTNET_RUNBOOK_24A.md).
