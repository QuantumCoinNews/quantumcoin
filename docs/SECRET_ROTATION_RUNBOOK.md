# Secret rotation and history cleanup

Encryption, local import, provider revocation and Git history cleanup are
separate operations. A successful `secrets-update` deliberately reports
`rotationPerformed: false`: it cannot revoke a credential at its provider.
Treat a credential previously shipped in a ZIP, committed or posted in chat as
exposed until its provider-side revocation is recorded. GitHub recommends
revoking/rotating secrets before repository history cleanup.
[GitHub procedure](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository).

## Account owner actions

| Service | Required action | Local handover |
| --- | --- | --- |
| OpenAI | Delete/revoke the exposed project key in the API key dashboard; review usage; issue a restricted replacement if the feature will be used | Update `OPENAI_API_KEY` through the existing encrypted-store command. An absent local key is not evidence that the old key was revoked. |
| X | Identify the publisher app in the Developer Console; revoke exposed credentials and authorize the intended account with the minimum required scopes | Prefer the canonical renewable OAuth 2.0 user flow with `tweet.write`, `users.read` and `offline.access`; legacy OAuth 1.0a remains a compatibility path during migration. Store the OAuth 2.0 client ID and consented token set in the Windows DPAPI store. Do not send tokens in chat. |
| Google / YouTube | Revoke the exposed OAuth grant and re-authorize the correct account/channel; review the OAuth client type and any exposed confidential client secret | Import fresh client/token JSON through the canonical embedded credential loader. Do not reuse an archived refresh token. |
| Telegram | Revoke the exposed bot token using BotFather and obtain its replacement | Update every consumer of that same bot, including game and publisher when applicable. Keep bot identity consistent. |
| GitHub or other provider | If any personal/app token was exposed, revoke it in its issuer's account settings and replace its authorized consumers | Store through that service's credential manager; never put a replacement into Git remote URLs, command history or tracked files. |
| Internal node/game/admin/monitor tokens | Inventory the configured values and their server/client consumers; replace exposed shared secrets together | Coordinate node, bridge, GUI/miner/admin callers and service restart; validate rejection of the retired token and acceptance of the new token. Do not silently replace only one side. |

OpenAI's official guidance describes rotation of leaked keys and usage review:
[API key safety](https://help.openai.com/en/articles/5112595-best-practices-for-api-key-safety).
X documents the credential types and key rotation:
[Developer apps](https://docs.x.com/fundamentals/developer-apps).
Telegram tokens can be revoked through BotFather:
[Bot token documentation](https://core.telegram.org/bots/tutorial#obtain-your-bot-token).

Google OAuth revocation can invalidate the grant's access/refresh tokens across
clients in the project; identify the affected consumers before cutover.
An installed/native OAuth app cannot keep a client secret confidential, so
replacing that string alone does not secure an exposed refresh token.
[Google token revocation](https://developers.google.com/identity/protocols/oauth2/web-server#tokenrevoke),
[native app credentials](https://developers.google.com/identity/protocols/oauth2/native-app).

## Existing Windows publisher handover

Use the canonical build and the existing `publisher.mjs secrets-update` flow.
Keep the replacement credential input outside the checkout. Paste keys only
into the local input file or approved secret-manager UI, never the chat or a
command argument. The example below contains paths only:

```powershell
$store = Join-Path $env:LOCALAPPDATA 'QuantumCoin/autopublisher/publisher-credentials.dpapi.json'
$changes = Join-Path $env:LOCALAPPDATA 'QuantumCoin/autopublisher/replacement-credentials.local'
node -- autopublisher/publisher.mjs secrets-update --env-file "$changes" --secret-file "$store"
node -- autopublisher/publisher.mjs auth-health --secret-file "$store" --settings-file autopublisher/publisher.settings
```

Run from the repository root. The example requires an already prepared input
file; it does not create credentials. For YouTube, that input can reference
fresh local JSON files through `YOUTUBE_CLIENT_SECRET_PATH` and
`YOUTUBE_TOKEN_PATH`. The loader embeds their contents in the encrypted store.

The update uses the runtime lease, preserves fields not supplied in the change
and round-trip verifies Windows CurrentUser DPAPI before atomic replacement.
Invalid replacement JSON must leave the original encrypted store intact.
Neither import nor update deletes the input file. After verification, retire
temporary plaintext through the approved local storage procedure; an ordinary
file deletion is not a guarantee of physical erasure from SSDs/backups.

DPAPI ties recovery to the Windows user/profile and machine key material. Do
not treat its encrypted file alone as a portable cross-machine recovery backup.
Keep wallet keys/seeds under the separate wallet custody tasks; provider API
rotation must not regenerate or discard a user's QC wallet.

## Evidence and closure

For each affected credential, record a non-secret account/app label, its
consumer services, the provider revocation event/time, replacement-install
time and the redacted authentication test receipt. A provider's revocation
record is distinct from a temporary 401/quota/network error. No key values,
refresh tokens, passwords or wallet material belong in the record.

Do not send an old exposed credential merely to see whether it still works.
Use the provider's revocation record; validate the replacement through the
explicit read-only health command. Authentication does not prove publication
permissions or successful delivery. Publication tests belong to the platform
acceptance task.

Closure also requires the agreed Git cleanup cutover: preserve outstanding
source changes, reconcile intended refs, apply reviewed rewritten history,
verify the actual remote and address old clones/caches/PR refs as applicable.
A sanitized mirror with no remote is preparation evidence, not completion.
See [Git privacy cleanup](GIT_PRIVACY_CLEANUP.md).

Retired encrypted archives are recovery/incident evidence, never a source of
credentials to re-enable. Keep their access and retention under custody review.
