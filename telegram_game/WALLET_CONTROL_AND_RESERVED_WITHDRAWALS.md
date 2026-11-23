# Wallet control and reserved withdrawals

**Delivery decision — 2026-11-04:** The user deferred real BSC payment/receipt
verification and its activation to the final mainnet phase. Keep the existing
disabled confirmation gate and retain the outstanding integration in the audit
record. Current work stops at the tested reservation/cancel/reject boundaries;
this decision does not certify a live payout or close the deferred integration.

The canonical service remains `telegram_game.runtime:app`. Compatibility entry
points use this same application and database; the independent legacy Store is
retired. These changes do not deploy a BSC claim contract or send tokens.

## Node authentication and signing

Configure the server-side `QC_API_TOKEN` to match the node. Health, balance,
transaction status, signed submission and client PoW job/submission carry
`X-QC-API-KEY`. HTTP redirects are refused. Remote connections require HTTPS;
loopback HTTP remains available for local development. Neither API tokens nor
wallet private keys are sent to the WebApp. The game accepts public, already
signed transaction envelopes; the old key-forwarding helper is retired.

Same-origin browser requests need no CORS setting. Additional trusted origins
can be explicitly listed in `QC_GAME_CORS_ORIGINS`, separated by commas. A `*`
entry is ignored. `/api/dev/reset` remains operator-protected and returns 410;
it has no data-deletion implementation.

## Prove wallet ownership

Set `QC_WALLET_LINK_NETWORK_ID` and `QC_WALLET_LINK_GENESIS_HEX` to the intended
node's network identifier and actual genesis hash. Blank values disable proof
issuance; no network or genesis is invented by the game.

1. Sign in through the verified Telegram WebApp session.
2. POST `/api/wallet/link-challenge` with `playerId` and the Base58Check address.
3. Review the downloaded public challenge. Use a trusted local wallet or run
   `python scripts/sign_wallet_link.py path/to/challenge.json` with the project's
   Python environment containing `cryptography`. Its private-key prompt is local,
   hidden terminal input. The program performs no HTTP requests and saves no key.
   The root project directory is the working directory for this command.
4. Paste only the resulting public proof into the wallet panel, or POST it to
   `/api/wallet/link-qc-address`. Fields are `playerId`, `address`, `nonce`,
   `publicKey` and `signature`.

The message is `QC_GAME_WALLET_LINK_V1` followed by a NUL byte and ASCII canonical
JSON with sorted keys and compact separators. The signature is secp256k1 ECDSA
with SHA256, DER encoded, lowercase hex. The public key is `04||X||Y` hex. Address
ownership uses the existing HASH160(X||Y) rule. These messages are distinct from
transaction signatures and the monthly NFT proof domain. Challenges bind the
Telegram account, address, network, genesis and a random nonce; expire after five
minutes; and are consumed atomically once. Only a valid public signature changes
the linkage. Balance visibility is not ownership evidence.

A proved address already held by another verified Telegram account cannot be
claimed again. An old, unproved association cannot squat on the real holder's
address: successful proof archives that old association before clearing it. Past
reward records are retained. Old database entries are not silently presented as
verified ownership proofs.

## Browser recovery migration

New seed generation and automatic plaintext seed loading/saving are retired in
both old wallet script blocks. Create wallets and sign in the wallet application.
The former prototype's word generator was a demo, not a supported mnemonic
generator; old recovery text is preserved without claiming it represents a valid
wallet.

If `qc_wallet_seed_v2` already exists, the recovery panel warns about it. On an
explicit user action it encrypts that local text with AES-256-GCM and a separate
random salt/IV, using PBKDF2-SHA256 with 600,000 iterations and a passphrase of at
least 14 characters. Only the encrypted envelope is written to the new storage
key. An encrypted backup can be downloaded and imported on another device.

The old plaintext is removed only after a successful encryption/decryption check,
confirmation that the encrypted backup and passphrase have been saved, and a
second passphrase check. Failures preserve recovery data. It is never
automatically copied into game state or sent over HTTP. Recovery requires HTTPS
or localhost. Unlocked output is cleared when the page is hidden or closed.
An existing browser's plaintext risk remains until its owner completes migration.
Passphrase encryption does not protect an unlocked page against active XSS.

Recovery writes and deletion use an exclusive browser Web Lock shared across
tabs. Browsers without secure Web Locks retain existing data and refuse these
mutations. No fallback writes plaintext. Every asynchronous operation also
checks its session generation and current storage before publishing results or
removing legacy data. A changed backup or legacy value invalidates the operation.

After a refresh, use **Download encrypted backup** with the passphrase to verify
and save the current backup again. Merely unlocking it does not authorize deleting
the old plaintext copy. Import never overwrites an existing backup and does not
automatically display recovery text; use **Unlock encrypted recovery** explicitly.
Unlocked text expires after 60 seconds. **Hide recovery**, page hiding, BFCache
return and a relevant change in another tab clear visible recovery and invalidate
pending work, preventing a late decryption from showing it again.

## Canonical TGWT accounting

All values use integer micro-units: 1 TGWT = 1,000,000 units. The shared pool
remains 3,000,000 TGWT and each verified social task earns exactly 1 TGWT.

| Field | Meaning |
|---|---|
| `earned_u` | Lifetime awarded TGWT, not a second spendable balance |
| `pending_u` | Unverified legacy credits; no automatic promotion |
| `verified_u` | Available verified off-chain entitlement |
| `reserved_u` | Entitlement held for managed requests in admin review |
| `withdrawn_u` | Entitlement settled by a verified confirmed payout |

For accounts following this contract, `earned_u = pending_u + verified_u +
reserved_u + withdrawn_u`. Reads expose `accountingConsistent` and
`legacyUnreconciledRequests`. Inconsistent accounts or unmanaged outstanding
legacy requests block new reservations pending a historical evidence review.
Historical amounts and payouts are not guessed, erased or automatically repaired.

POST `/api/v1/withdraw/request` reserves verified units, writes the request and
ledger event in one SQLite transaction, and leaves `withdrawn_u` unchanged. Amounts
with more than six decimal places are rejected. `(player, nonce)` is unique;
repeating the identical request returns it, while changing its payload rejects it.

POST `/api/v1/withdraw/cancel` accepts an owned `requestId` and releases an open
reservation. POST `/api/admin/tgwt/withdraw/reject` is operator-only and performs
the same refund with a rejected state. Both are idempotent. Neither reduces the
shared pool's lifetime allocated total: the entitlement still belongs to the user.

The internal confirmed transition requires a trusted server verifier's
request-bound chain evidence, records each BSC transaction/log only once, and
moves reserved units to withdrawn atomically. Its verifier is supplied only by
server code, never by client JSON. This boundary is exercised with test fixtures;
it is **not a deployed BSC receipt verifier**. The HTTP confirmation endpoint
returns 503 even for an operator until that real adapter and the claim contract
are delivered. A typed hash, operator click or admin-review request cannot mark
a payout as complete. No live BSC payout was tested or activated.

Before enabling user-paid claims, the authorization handoff must lock a reviewed
reservation against cancellation, bind the on-chain claim to its request/account,
and verify the deployed contract, token units, canonical receipt and finality.
The user pays transfer gas in BNB. There is no backend hot wallet or gas subsidy.
[BNB Chain's finality documentation](https://docs.bnbchain.org/bnb-smart-chain/introduction/)
describes the finalized block query that the future adapter must check; a local
receipt or arbitrary confirmation count must not substitute for that integration.
