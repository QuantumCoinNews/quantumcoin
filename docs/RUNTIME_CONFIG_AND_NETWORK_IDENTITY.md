# Runtime configuration and network identity

The root runtime loads `config.json` from its configured data directory, or the
explicit config path selected at startup. `.chain.json` was a historical
snapshot and has been retired from the active workspace. Reviewed examples live
under `configs/`. Keep runtime copies and machine inventory exports out of source
releases.

Reward percentages must each be in 0..100 and total exactly 100. Only the active
100% miner / 0% staking / 0% developer / 0% burn / 0% community contract is accepted.
An old 70/10/10/5 proposal is rejected with a configuration error; it does not
activate a new consensus split. Back up old configuration and deliberately replace
those proposal values with the implemented contract. No configuration is rewritten
automatically. The same validation runs before the canonical runtime is published.

`config.QC` is a deprecated alias for `consensus.CanonicalAtomsPerQCV1`: 1,000,000
atoms per QC. Decimal parsing and protocol boundaries remain authoritative. Old
signed whole-QC transactions must not be multiplied or re-signed by a config alias.

The mutable globals in `config/params.go` are deprecated source compatibility
symbols. They are not consulted by runtime mining, staking, block size or reward
validation. Staking and its yield default to disabled/zero. RuntimeCapabilitiesV1
and the health response describe implemented capabilities, using the canonical
consensus source. A future staking implementation requires separate design and
activation; changing a legacy boolean cannot enable it.

The canonical bootstrap pins a network identity `(version, network, genesis hash)`
before loading persisted state. Networks are local-release, testnet and mainnet;
local/public testnet are operational profiles of the same testnet domain. The
identity is serialized with the chain, restored from trusted local context on
reload, and retained by ordinary and NFT mining commits. An incoming file or peer
cannot choose the runtime's identity. Local and mainnet peers use distinct handshake
magic; the runtime's pinned identity takes precedence over mutable environment.

The legacy default filename and ports remain compatible. They do not establish
network authority. Mainnet rejects both a local-marked chain and an untagged legacy
file, even if named chain_data.dat. OS-locked destination validation also prevents
two networks that started before the first save from overwriting one another's
newly saved chain. Untagged writers cannot overwrite a tagged destination.

Legacy files are not deleted, renamed or rehashed. A validated untagged legacy file
can be opened outside mainnet; an in-memory identity is attached and written on the
next ordinary save. A read-only load does not migrate the file. Unknown legacy
origin is never silently promoted to mainnet. Use separate data directories when
running different networks. An operational identity tag is not a signature proving
historical origin; full genesis, chain, PoW, supply and activation validation remains
mandatory. Deliberate manual alteration of data files is outside that tag's trust
boundary and does not grant authority to change canonical rules.

For a fresh isolated local release, use `configs/local-release.isolated.example.json`
with a separate existing data directory. This opt-in profile uses
`chain_data_local.dat`, HTTP port 28082 and P2P port 23001. It commits the local
domain to the genesis transaction digest and uses distinct genesis transaction
IDs and P2P magic. Startup verifies that genesis from trusted local configuration.
Changing a disk tag cannot make an isolated local chain pass mainnet validation.
Legacy local chains keep their original genesis and are outside this new
profile. Moving an old chain to the new filename does not migrate it.

The live `/api/network/label` response uses the bound runtime identity and
genesis hash. A filename, port or inherited environment setting alone cannot
establish the displayed network. A runtime without a bound identity reports
UNVERIFIED.
