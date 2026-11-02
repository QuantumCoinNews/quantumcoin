# QuantumCoin TGWT BSC Claim Mainnet/Testnet Lock

Status: TGWT_BSC_CLAIM_MAINNET_TESTNET_LOCK_V1

Date: 20260618_211116

## Decision

TGWT is a BSC-side token claim model. Telegram Game can award off-chain TGWT ledger balances through Watch & Earn and social verification, but BSC claim/transfer remains separate from the QuantumCoin native chain.

## Claim model

- Claim model: USER_PAID_BSC_CLAIM
- User pays BSC gas/fee in BNB.
- Backend does not pay user gas.
- Backend does not create BSC transactions.
- Backend does not store or use private keys.
- Distribution wallet address is public config only.
- TGWT token contract address is public config only.
- Claim contract address is public config only.

## Public env fields

- TGWT_BSC_CLAIM_ENABLED
- TGWT_BSC_NETWORK
- TGWT_BSC_CHAIN_ID
- TGWT_BSC_DISTRIBUTION_WALLET_ADDRESS
- TGWT_BSC_TOKEN_CONTRACT_ADDRESS
- TGWT_BSC_CLAIM_CONTRACT_ADDRESS
- TGWT_BSC_MAINNET_CLAIM_APPROVAL

## Network lock

- BSC testnet: chain id 97
- BSC mainnet: chain id 56
- Mainnet requires explicit approval: I_APPROVE_TGWT_BSC_MAINNET_CLAIM

## Forbidden

Never put these in code, repo, UI, public config, or env examples:

- private key
- seed phrase
- mnemonic
- hot wallet secret
- treasury private key
- distribution private key

## Scope

This step does not deploy token, create claim contract, connect MetaMask, send BSC transaction, pay BNB fee, or activate mainnet claim.
