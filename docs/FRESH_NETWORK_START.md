# Fresh testnet and mainnet

Decision recorded on 13 November 2026: both networks start with new chain
history. No historical balance, UTXO, block or mining result is migrated.
Existing chain files remain historical records and are not imported. Their
filenames do not establish network origin. Wallet custody remains separate:
an existing key or recovery phrase can still identify an address, but it does
not transfer that address's old-chain balance into a new network.

Clarified on 14 November 2026: development/test mining already performed is
not a head start for mainnet. Its first launch starts with only its finalized
mainnet genesis at height zero; its first mined block is height one. The final
mainnet genesis date/inputs/fingerprint must be fixed in the later mainnet
acceptance, not inherited from a development draft. Reuse of keys does not
copy balances. A normal restart after that launch preserves the legitimate
mainnet history; startup is never an automatic chain deletion/reset operation.

Genesis founder/reserve allocation, final emission policy, signed release and
mainnet launch acceptance remain their own work items. This reset decision
does not finalize those inputs or start either network.

## Separate runtime directories

Windows defaults are outside the source tree and signed package:

| Network | Directory | Chain file |
| --- | --- | --- |
| Testnet | `%LOCALAPPDATA%/QuantumCoin/networks/testnet-v4` | `chain_data_testnet.dat` |
| Mainnet, when ready | `%LOCALAPPDATA%/QuantumCoin/networks/mainnet-v4` | `chain_data.dat` |

Both use the `network-v4` genesis/signature profile. Their genesis markers,
P2P domains and transaction-signature scopes differ. Local-release data stays
in its original location for inspection; renaming it does not create a new
mainnet or testnet.

Canonical mainnet startup now rejects empty/V2/V3 genesis profiles, even before
the later release permits a mainnet launch. Public testnet has the same V4
requirement. This does not bypass mainnet economics, fee or release gates.

```powershell
python scripts/network_workspaces.py plan --network testnet
python scripts/network_workspaces.py plan --network mainnet
.\release\RUN_TESTNET_API_18082.ps1 -PlanOnly
.\release\RUN_TESTNET_NODE_18082_13001.ps1 -PlanOnly
```

Planning does not create a directory or start a process. The testnet launch
wrappers use the dedicated testnet directory by default. They retain the
verified-release and API-token requirements. Only after those gates pass does
the launcher prepare a new empty directory and invoke the node.

`python scripts/network_workspaces.py prepare --network testnet` prepares only
the directory marker, without creating a chain, wallet or node process. A
first preparation rejects any existing nonempty directory. There is no force,
reset, copy or migration switch. A marker for another network, a corrupt marker,
source/package paths, indirect paths and foreign chain filenames are rejected.
Prepared directories can be reused for the same network on restart. The
marker is an operational guard, never consensus authority: the node must still
validate the configured genesis, file header, signatures and complete history.
A copied legacy chain with the expected filename therefore still fails node
validation rather than being relabeled.

The mainnet example `configs/mainnet.fresh.example.json` is a draft directory/
network template only. Do not treat it as a finalized genesis allocation or a
launch approval. Mainnet is prepared/launched only in its later release stage.

## Historical files

`python scripts/network_workspaces.py inventory` hashes existing chain `.dat`
files in the checkout read-only. It does not decode wallet keys, balances or
untrusted Gob, and it never deletes, moves or imports the files. The inventory
and reset decision belong in local audit evidence; runtime data and keys stay
out of source/release packages.

For an old chain that must be inspected, retain its original configuration and
data directory. The legacy-local example is separate from the new default
testnet profile. Keep required encrypted recovery backups independently.
