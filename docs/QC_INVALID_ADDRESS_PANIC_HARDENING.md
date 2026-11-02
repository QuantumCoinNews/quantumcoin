# QuantumCoin Invalid Address Panic Hardening

Status: QC_INVALID_ADDRESS_NO_PANIC_V1

Date: 20260617_221758

Invalid external addresses must not crash the node.

## Change

- wallet.Base58DecodeAddress no longer panics on invalid address.
- wallet.Base58DecodeAddressSafe remains the error-returning validator.
- Invalid address decode returns nil pubKeyHash through the legacy wrapper.

## Verified by tests

- invalid Base58 decode does not panic
- safe decoder returns error
- ValidateAddress returns false
- GetBalance / GetSpendableBalance do not panic
- NewTransaction does not panic
- Transaction.Verify does not panic

## Safety boundary

This does not mint, transfer, change balances, change rewards, change consensus, or change total supply.
