package blockchain

import (
"bytes"
"testing"
"time"

"quantumcoin/config"
"quantumcoin/wallet"
)

func TestFounderVestingUsesGenesisBoundAddressV1(t *testing.T) {
original := config.Current()
t.Cleanup(func() {
config.Set(original)
})

// Genesis founder addresses.
genesisLiquidWallet := wallet.NewWallet()
genesisLockedWallet := wallet.NewWallet()

cfg := config.Default()
cfg.PremineLiquidAddress = genesisLiquidWallet.GetAddress()
cfg.PremineLockedAddress = genesisLockedWallet.GetAddress()

if err := config.SetValidated(cfg); err != nil {
t.Fatalf("set genesis founder config: %v", err)
}

chain := NewBlockchain(
cfg.InitialReward,
cfg.TotalSupply,
)
if chain == nil || len(chain.Blocks) != 1 || chain.Blocks[0] == nil {
t.Fatalf("expected canonical genesis chain")
}

genesisHash := append([]byte(nil), chain.Blocks[0].Hash...)

lockedHash, err := wallet.Base58DecodeAddressSafe(
cfg.PremineLockedAddress,
)
if err != nil {
t.Fatalf("decode genesis locked address: %v", err)
}

liquidHash, err := wallet.Base58DecodeAddressSafe(
cfg.PremineLiquidAddress,
)
if err != nil {
t.Fatalf("decode genesis liquid address: %v", err)
}

var lockedTxID []byte
lockedOutIndex := -1

for _, tx := range chain.Blocks[0].Transactions {
if tx == nil {
continue
}

for outputIndex, output := range tx.Outputs {
if bytes.Equal(output.PubKeyHash, lockedHash) {
lockedTxID = append([]byte(nil), tx.ID...)
lockedOutIndex = outputIndex
break
}
}

if lockedOutIndex >= 0 {
break
}
}

if len(lockedTxID) == 0 || lockedOutIndex < 0 {
t.Fatalf("genesis locked founder outpoint not found")
}

makeSpend := func() *Transaction {
return &Transaction{
Inputs: []TransactionInput{
{
TxID:     append([]byte(nil), lockedTxID...),
OutIndex: lockedOutIndex,
},
},
Outputs: []TransactionOutput{
{
Amount: FounderLockedAllocationV1 - 1,
PubKeyHash: append(
[]byte(nil),
lockedHash...,
),
},
{
Amount: 1,
PubKeyHash: append(
[]byte(nil),
liquidHash...,
),
},
},
Timestamp: time.Unix(
CanonicalGenesisUnixV1,
0,
).UTC(),
}
}

preUnlockUnix := CanonicalGenesisTimeV1().
AddDate(1, 0, 0).
Unix()

// Baseline: with the original runtime config, the genesis founder
// output is recognized and the pre-cliff spend must be rejected.
if err := chain.ValidateFounderVestingSpendV1(
makeSpend(),
preUnlockUnix,
); err == nil {
t.Fatalf("baseline founder vesting protection did not reject pre-unlock spend")
}

// Change only the runtime founder addresses after genesis already exists.
// The chain/genesis remains unchanged.
otherLiquidWallet := wallet.NewWallet()
otherLockedWallet := wallet.NewWallet()

runtimeCfg := config.Default()
runtimeCfg.PremineLiquidAddress = otherLiquidWallet.GetAddress()
runtimeCfg.PremineLockedAddress = otherLockedWallet.GetAddress()

if err := config.SetValidated(runtimeCfg); err != nil {
t.Fatalf("set changed runtime founder config: %v", err)
}

if !bytes.Equal(chain.Blocks[0].Hash, genesisHash) {
t.Fatalf("genesis hash changed after runtime config mutation")
}

// Consensus must continue enforcing the rule encoded by the existing
// genesis founder allocation. Current implementation instead re-reads
// runtime config and can return nil because the old locked output no
// longer matches the newly configured locked address.
if err := chain.ValidateFounderVestingSpendV1(
makeSpend(),
preUnlockUnix,
); err == nil {
t.Fatalf(
"founder vesting accepted pre-unlock spend after runtime founder address changed",
)
}
}
