# QuantumCoin Hardfork Policy

Status: QC_HARDFORK_POLICY_V1

Date: 20260619_210657

## Purpose

This step defines when a hardfork may be considered and what rules must be followed.

This step does not execute a hardfork. It does not change consensus, mint QC, change balances, replace chain data, or change max supply.

## Allowed hardfork reasons

Hardfork may only be considered for serious network-level reasons:

- Critical consensus bug
- Chain halt or finality failure
- Critical security vulnerability
- Cryptographic algorithm break or urgent rotation
- State corruption repair with audited snapshot
- Network-wide protocol upgrade with public notice

## Forbidden reasons

Hardfork must not be used for:

- Changing max supply
- Minting extra QC
- Resetting founder vesting
- Bypassing burn rules
- Stealing or reassigning balances
- Changing economy for convenience
- Admin shortcut
- AI-only automation
- Game/TGWT-only request

## Required process

A valid hardfork process requires:

- Founder approval
- Multi-step admin approval
- Public notice
- Testnet rehearsal
- Migration height
- Snapshot hash
- Rollback plan
- Node upgrade window
- Release binary hashes
- Post-fork audit

## Automation restrictions

The following cannot run a hardfork:

- AI
- Worker
- Telegram Game
- TGWT
- NFT system
- Admin panel alone

## Safe boundary

This step does not:

- execute hardfork
- change consensus now
- mint QC
- change max supply
- change balances
- bypass founder vesting
- bypass burn rules
- start or stop nodes
- replace chain
