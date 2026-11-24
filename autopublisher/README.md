# QuantumCoin publisher

The canonical publisher is `publisher.mjs` → verified `dist/runtime/cli.js` → `src/runtime/engine.ts`. YouTube,
X and Telegram use real provider APIs. Instagram and TikTok explicitly report
`NOT_IMPLEMENTED`; they cannot return a simulated successful publication.

## Install and verify

The validated Windows profile uses Node.js 22.23.5, PowerShell 7 (`pwsh.exe`) and FFmpeg/FFprobe 9.0.2. Versions are recorded in ../configs/windows-build-toolchain.json.

```powershell
$store = Join-Path $env:LOCALAPPDATA 'QuantumCoin/autopublisher/publisher-credentials.dpapi.json'
npm ci --ignore-scripts
npm run check
npm test
npm run build
node -- publisher.mjs doctor --secret-file "$store" --settings-file publisher.settings
node -- publisher.mjs run --slot 1
```

The last command is an offline dry-run. Publication credentials use an explicit encrypted `--secret-file`. An explicit `--env-file` is limited to import/offline diagnostics; `.env` is never loaded automatically.
Keep credentials local. Never paste API keys into chats or commit them.
`publisher.settings` contains public settings only and accepts an allowlist
of configuration names. It cannot override credentials.

## First publication and migration

Inspect existing provider history before choosing the next pool index. Old
slot cursors and logs are preserved as historical evidence, not silently
imported as proof that a post succeeded.

```powershell
node -- publisher.mjs init --start-date 2026-11-06 --next-index 28 --secret-file "$store" --settings-file publisher.settings
node -- publisher.mjs run --publish --slot 1 --secret-file "$store" --settings-file publisher.settings
```

`init` refuses to overwrite an existing production ledger. The default state
directory is `%LOCALAPPDATA%\QuantumCoin\autopublisher`, outside the repository.
An absolute `--state-dir` can override it. Dry-run, render and publish have
separate state namespaces. Dry-runs never consume the production cursor.

One OS-held exclusive file lease protects every production slot. Offline
dry-runs use a separate namespace and lease, so retired task triggers cannot
prevent a live scheduled publication. A crash releases the
lease through the OS; timestamps and stale-file deletion do not grant access.
Each job persists its selected topic, media hashes, attempt count and real
provider receipts. A confirmed destination is skipped on a rerun. A network
timeout or crash after submission stops the job as `needs-review` because a
second upload could duplicate an already delivered post. Later slots remain
blocked until the current job is completed or reviewed.

For an explicit API rejection, rerun the same date and slot; there is a limit
of three attempts per destination. For an uncertain result, inspect the
provider account first. Do not edit a receipt into existence. If the job must
be skipped after review, `abandon --job YYYY-MM-DD.slot1` preserves all known
receipts and uncertain outcomes while advancing the cursor.

YouTube uploads use explicit privacy, English language metadata and a real
video stream. X supports a text post but is currently paused after a live HTTP 402
payment-required rejection. Resolve billing before adding X back to the public
platform list. Telegram uploads the video.
Authenticated account access alone does not prove upload permission, quota,
processing success or public visibility; confirm those from actual receipts
and the provider after the first live run.

## English content and AI

The topic pool contains 60 English seeds. The original 30 have prepared audio;
the added 30 cover wallet custody, signatures, mining, atomic amounts,
receipts, backups, social proof, NFT fairness and honest project status.
Prepared-audio mode cycles only indices with existing audio. AI mode can use
all 60 seeds; missing audio is never replaced by an empty placeholder.

AI is disabled in the shipped public settings until a valid local credential
is available. To enable it, add a newly created `OPENAI_API_KEY` to local `.env`
and set `QC_PUBLISHER_AI_ENABLED=1` in the public settings. Finish or review the
current production job before changing settings. `OPENAI_MODEL` defaults to
`gpt-4o-mini`; `OPENAI_TTS_MODEL` defaults to `gpt-4o-mini-tts`.

The AI path uses the Responses API with a strict English JSON schema and
`store:false`, then the Speech API. It checks length and unsupported product
claims, discloses synthetic narration, persists the generated asset once per
job and caps generation attempts. It does not invent current news, market
prices, mainnet launch status or completed payouts. Unit tests use mock clients
and never create paid content or publish to a real account.

## Scheduling

Use one scheduling authority. The canonical scheduler uses
`QC_PUBLISHER_CRON=0 18,21 * * *` and `Europe/Istanbul`:

```powershell
node -- publisher.mjs schedule --publish --secret-file "$store" --settings-file publisher.settings
```

Alternatively register a Windows task with two daily triggers that calls
`node -- publisher.mjs run --publish ...`. Disable the old project publishing
tasks after recording their definitions. Do not run both scheduling methods.
Publication requires explicit `--publish`; the scheduler otherwise remains a
dry-run. The OS lease and receipts also protect against overlapping triggers.

The PowerShell and CMD entry points are thin launchers to this runtime.
Legacy direct-upload MJS files and `publish-all.ps1` exit with code 64 and
instructions to use the canonical path. Auth setup and asset preparation
utilities are manual tools, not alternative scheduled publishing pipelines.

On this host, `scripts/install-schedule.ps1` installs the Windows task and
exports old task definitions before disabling them. Protected legacy tasks
that cannot be disabled must be proven to invoke only the offline default
launcher. The installer reports them separately; it never claims they were
disabled. The task runs while this Windows user is logged on, with catch-up
when available; it does not run while the computer is powered off.

Keep an encrypted operational backup of the publisher state directory,
including ledgers, jobs and receipts, using the project backup/restore
procedure. It is outside the source checkout. Stop publishing or hold the
production lease while snapshotting it. Before restoring an older snapshot,
reconcile provider history; an old receipt ledger cannot prove that a newer
external publication never happened. Do not delete production state to retry.


## Credential storage and account recovery

Production on this host loads `--secret-file` from a Windows CurrentUser DPAPI
store, outside the checkout. Do not pass both `--env-file` and `--secret-file`.
Plaintext `.env` loading remains an explicit migration compatibility path;
`--publish --env-file` is rejected. Production uses the encrypted store or
credentials injected into the process by an external secret manager.
The local publisher `.env`, its three old backups, the client/token files and
the token copy previously labeled revoked have been retired after encrypted archive verification.
Do not recreate an active token from a revoked archive.
The filename is not proof of provider revocation. Follow the
[secret rotation runbook](../docs/SECRET_ROTATION_RUNBOOK.md) and record the
provider-side event separately from local encryption/import.

For a fresh setup, obtain/re-authorize credentials through each provider's
console and put them in a temporary local env file. Then run:

```powershell
node -- publisher.mjs secrets-import --env-file new-credentials.local --secret-file "$store"
node -- publisher.mjs auth-health --secret-file "$store" --settings-file publisher.settings
```

Import refuses an existing destination. To add a newly issued key to an
existing store without losing other platform credentials:

```powershell
node -- publisher.mjs secrets-update --env-file changed-credentials.local --secret-file "$store" --settings-file publisher.settings
```

Updates hold the production lease, merge only supplied fields and verify
DPAPI round trips before atomic replacement. Securely retire the temporary
plaintext input after verification; these commands do not revoke provider keys
or delete input files. Never put a key in command arguments or chat messages.

YouTube's canonical credential service preserves refresh tokens omitted by
refresh responses and saves new tokens into the encrypted store. Read-only
health probes classify reauthorization, quota, billing and rate-limit failures
without printing provider response bodies. Authentication does not prove write
permission, quota or successful publication.

X supports the legacy OAuth 1.0a credential quartet and renewable OAuth 2.0
user authorization. For the renewable path, configure an X Developer App with
OAuth 2.0 and the `tweet.write`, `users.read` and `offline.access` scopes, then
obtain owner consent locally. X access tokens normally last two hours; with
`offline.access`, a refresh token is issued. The publisher refreshes at
`expires_at - min(5 minutes, max(30 seconds, 10% of expires_in))`. A two-hour
token therefore refreshes at 1 hour 55 minutes. Each rotated token is saved to
the Windows DPAPI store before its access token is used. If persistence fails,
the publisher stops and requires credential recovery. The actual X app ID,
owner consent and live posting acceptance remain account-owner setup steps;
the local source tree cannot create them. For local consent, register the exact
loopback callback `http://127.0.0.1:8765/callback` in the X app settings, then
run this command from `autopublisher` after supplying the public Client ID:

```powershell
node -- publisher.mjs x-oauth2-authorize --client-id "$clientId" --redirect-uri "http://127.0.0.1:8765/callback" --secret-file "$store" --settings-file publisher.settings
```

The command opens a loopback-only callback listener, verifies OAuth `state`,
uses PKCE S256, requires all three scopes and a refresh token, exchanges the
one-time code, and stores the token set in DPAPI without printing it. It does
not publish. The account owner must open the printed authorization URL and
approve the account access. The post-authentication X API write/billing check
is a separate live platform acceptance step.

DPAPI recovery requires the relevant Windows user/profile and machine key
material. It is not a portable cross-machine backup format by itself. Preserve
an approved encrypted recovery/export procedure before changing machines.
The encrypted original-file recovery archive is outside the repository in
`%LOCALAPPDATA%\QuantumCoin\autopublisher\retired-plaintext-recovery`.

## Editorial and API boundaries

AI output is evergreen educational content grounded in curated technical
references in `src/runtime/sources.ts`. Source IDs are validated and URLs are
attached from that registry. The registry also records the development status
constraints supplied for QuantumCoin. It does not fetch live news or certify
that every generated paraphrase is accurate. News, price updates and unreviewed
project launch claims are rejected; a live-news product is not enabled.
Legacy topic/script/image/voice placeholder modules fail explicitly.

Delivery backoff and circuit state persist under `delivery-health/` in the
state directory. Explicit rejections use bounded exponential cooldown plus
jitter; auth/billing failures open a one-hour circuit. Confirmed delivery
resets it. Rerunning before cooldown does not call the API. The per-job attempt
budget still applies. Ambiguous outcomes require review and never become
safe retries merely because a timer expired. Resume the same job/date/slot
after resolving its failure; do not abandon a job just to bypass its breaker.

A heuristic secret scan is available as `python scripts/audit_publisher_secrets.py`.
It scans the worktree and all local Git refs without printing secret values.
It cannot prove provider rotation or removal from remote forks, caches,
release archives or chats. Git history cleanup and provider rotation remain
separate operational acceptance steps; no force push is performed by the scan.


## Source release boundary and operations storage

Build the publisher before exporting source. From the repository root:

```powershell
python scripts/test_publisher_source.py
python scripts/build_publisher_source.py --output C:/outside-checkout/publisher-source.zip
```

The exporter uses an allowlist, checks the successful build's source hashes,
blocks known credential patterns and writes a deterministic ZIP with a SHA-256
file manifest. It includes source, tests, lockfile, public configuration and the
curated text catalog. It excludes dependencies, compiled output, credentials,
media pools, logs, temp files, locks, cursors and delivery records. Historical
Git blobs are not rewritten by export. A clean working tree/source release is
not proof that old repository history has been sanitized.

Install the exported `autopublisher` folder with `npm ci --ignore-scripts`,
then `npm run build`. Do not distribute a developer's node_modules snapshot.
PowerShell 7, Node 22+ and FFmpeg/FFprobe remain runtime requirements. The
source-only package intentionally does not include ready audio/backgrounds.

The default prepared-media directories are now:

- `%LOCALAPPDATA%/QuantumCoin/autopublisher/media/audio`
- `%LOCALAPPDATA%/QuantumCoin/autopublisher/media/bgs`

Place operator-supplied media there and run `publisher.mjs media-register`.
The versioned media manifest binds an audio filename/hash to the topic ID,
title and narration hash. Reordering topics follows their identity; changing
narration requires a reviewed replacement baseline. Registration does not
perform speech transcription or certify the original recording's words.
Fully generated AI visuals/audio do not require a legacy media snapshot.

The migration helper archives old logs, output JSON, temp media, slot files
and locks outside the checkout, verifies hashes and preserves current media
as canonical copies. It does not delete dependency installations or unrelated
project data. Legacy media/auth/upload entry points explicitly retire with an
error; they are not alternate operating modes. `src/index.ts` is also retired;
use the static `publisher.mjs` bootstrap.

## Verified build startup

The bootstrap holds a shared build lease and verifies source/compiled-file
fingerprints before importing the CLI. The build takes an exclusive lease,
invalidates the old completion marker and only publishes a new marker after a
successful compile. A build cannot overwrite files beneath a running publisher;
wait for it to finish or stop the cron service before rebuilding. A failed or
stale build cannot report a successful publication through the official entry.
The build manifest establishes consistency, not a signed supply-chain trust
root. Use `npm ci` before upgrades while the publisher is stopped.

## Scheduler authority and recovery

`QC_PUBLISHER_SCHEDULER=windows-task` is the current deployment authority. The
Windows installer passes `--scheduler windows-task`, an explicit state path,
and the static bootstrap. Do not invoke `schedule` with Windows settings.
For cron, stop the Windows publisher first and explicitly select `cron` in
settings. A scheduler authority lease also rejects a competing scheduler;
manual one-off runs still share the production job lease and receipt ledger.

The two logical slots are 18:00 and 21:00 Istanbul time. Before 18:00 the last
due identity is the previous day's slot2. Late runs retain that identity across
midnight; a reserved incomplete job is resumed before a new slot. Closed-job
cursor recovery does not create a second publication. Missed unreserved slots
are coalesced to the latest due slot, not bulk-backfilled. There is no separate
YouTube publishAt queue. `--date`/`--slot` identify explicit operator jobs.
Windows retries a failed invocation three times at five-minute intervals;
provider retry budgets/cooldowns and uncertain-outcome review still apply.

The job record is authoritative for frozen content and delivery receipts. The
ledger only reserves the active job and next catalog position. Dry-run/render
and publish namespaces are separate; copying a dry-run job into production is
rejected. Do not recreate old `last.json`/slot cursors to recover current state.

## Retention and access policy

Use `publisher.mjs maintenance --state-dir <absolute-state-path>` to plan cleanup.
`--apply` removes only recognized generated artifacts older than 30 days for
closed jobs, after writing an operation manifest. Failed/incomplete/uncertain
work, job records, receipts, ledger, input media and credentials are preserved.
Keep job/receipt history for idempotency until a separately reviewed archival
migration; do not delete it as ordinary log cleanup. Legacy migration archives
are recovery evidence: review after 90 days and securely retire only after
backup/reconciliation approval. They are not automatically pruned.

Operational data belongs in the user profile with private access, not source
releases or issue attachments. `scripts/protect_publisher_state.ps1` audits
broad-read Windows ACLs; `-Repair` can re-materialize legacy archived files in
already-private parent directories, verifying identical hashes before removing
the previous copy. It preserves Windows service/capability permissions rather
than escalating privileges or altering other applications' ACLs. For custom
storage, the operator must supply equally private permissions and backups.
# Node monitoring

Set `QC_PUBLISHER_MONITOR_ORIGIN` to the direct numeric loopback HTTP origin of
the writable node. Set the same dedicated `QC_PUBLISHER_MONITOR_TOKEN` in the
node's process environment and the publisher's encrypted credential store or
process environment. Leave both settings empty on the publisher to disable
reporting. The origin can also be set in `publisher.settings`.

After a publish run, the publisher reports the durable job ID, editorial
decision, and confirmed, failed, or uncertain delivery states. A confirmation
represents a provider receipt. Operators can review incidents at
`/api/monitoring`. Reporting has a two-second timeout. A subsequent run for
the same slot can report its saved state again without resubmitting confirmed
deliveries. Dry-run and render-only jobs do not report.
