# Policy packages and runtime enforcement

The five policy packages are declarations checked by
`node.ValidateRuntimePolicyGateV1`. The production bootstrap path calls the same
private validation boundary after loading verified hardfork activation and
before configuring advisory storage, opening a chain, or constructing Runtime.
Root CLI starts HTTP/mining/P2P only after canonical bootstrap succeeds.

| Package | Startup binding | Behavioral authority |
| --- | --- | --- |
| burncontrol | ValidatePolicyV1; canonical cap/sink alignment | Blockchain signed burn admission, unspendable sink, burn limits and circulation accounting; no replacement mint |
| economycontrol | ValidatePolicyV1; consensus/config reward, supply, maturity and allocation checks | Consensus/blockchain block reward and supply validation |
| foundervesting | ValidatePolicyV1; canonical allocation, schedule and config checks | Genesis-bound founder outputs and canonical spend/vesting checks |
| hardforkpolicy | ValidatePolicyV1; activation validation | Verified canonical release, trusted signer quorum, signed network/genesis/height binding and blockchain activation |
| networkfee | ValidatePolicyV1; canonical fee accounting probe | Signed transaction UTXO conservation, atom fee accounting and verified fees in block reward |

Policy validation does not itself mint, spend, activate a fork or replace
consensus checks. A startup gate passing is not proof of mainnet launch readiness.
The explicit-mainnet final fee/activation guard remains in place; final release
evidence, genesis/economy decisions and other outstanding acceptances stay in
their own tracker items. No fee or allocation policy is changed by this review.

Acceptance includes full tests of the five policy packages and consensus,
root bootstrap ordering, actual node invalid/default policy gates, signed
hardfork startup/reload rejection, existing-chain preservation on config drift,
and targeted real blockchain burn/fee/vesting behavior. Existing projection-only
tests establish their stated boundary, not a production deployment. Test fixtures
and temporary paths never migrate or overwrite an operator's chain.
