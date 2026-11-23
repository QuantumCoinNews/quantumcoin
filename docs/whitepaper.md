# QuantumCoin technical overview

This document describes the current source and names release or activation
requirements separately. The [README](../README.md) describes the product
scope. Autopublisher uses its own [editorial status and source registry](../autopublisher/src/runtime/sources.ts).

| Area | Current scope |
| --- | --- |
| Supply and mining | The canonical cap is 25,500,000 QC. Founder allocation is 5% when configured. The miner receives the block subsidy and validated fees. The whole-QC subsidy starts at 50 and the first scheduled halving is at height 2,102,401. At the 30-second target rate, the current cap can be reached before that halving. |
| Consensus and wallet | The node validates PoW, transaction signatures, UTXO ownership and supply. The wallet signs locally. Atom-denominated consensus applies to explicitly activated protocols. |
| Burn | A signed burn uses its applicable protocol and policy gates. There is no automatic percentage burn of block rewards or replacement mint. |
| AI and monitoring | AI produces advisory observations. Transaction admission and consensus run independently of AI. Monitoring records incidents and supports operator review. |
| NFTs | The monthly lifecycle is implemented in source. Final artwork, production approvals and activation are separate milestones. |
| Token Factory and staking | Token Factory and project-token mining are planned. Staking is inactive in consensus. |
| Telegram and publishing | Telegram Game has its own ledger and identity boundary. The publisher implements YouTube, Telegram and X paths. Live acceptance is provider-specific. TikTok is manual. |
| Network and release | Testnet uses a separate profile. Mainnet activation and signed production packaging are separate release decisions. |

See the [mining reward contract](MINING_REWARD_CONTRACT.md), the [network and
emission notes](NETWORK_GENESIS_AND_EMISSION.md), and the [testnet
instructions](../testnet/README.md) for operational detail.
