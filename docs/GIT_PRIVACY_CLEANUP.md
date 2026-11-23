# Git index and history privacy cleanup

Index cleanup, committed-history rewriting, remote publication and credential
rotation are four separate acceptance steps. Ignoring or untracking a file does
not revoke its key or erase earlier commits, reflogs, caches or third-party clones.

## Guarded index cleanup

`scripts/git_privacy.py plan-index` records the selected HEAD, exact index hash,
entry modes/blob IDs and paths excluded by the canonical source policy. Keep the
plan outside the checkout. `apply-index --plan PLAN --evidence NEW_DIRECTORY`
requires that exact plan, preserves a private index backup, prepares a candidate,
verifies protected entries, and publishes using Git's exclusive `index.lock`.
It rejects stale/tampered plans and existing locks. Files in the working tree,
already staged source contents and existing staged deletions are preserved.

The operation creates no commit and pushes nothing. The old index backup is
restricted operational evidence; restoring it would reintroduce the old tracking
state and must be a deliberate, locked Git operation.

`audit-index` inspects **all** retained index blobs, including unchanged blobs.
This complements `safe_push.py`, which checks explicitly staged source changes.

## Independent history rehearsal

Install the optional tool into a separate environment using
`pip --isolated install --require-hashes -r scripts/privacy-tools.requirements.txt`.
This is not a Telegram Game runtime dependency. Clean-stack CI restores the tool
in its own venv and runs synthetic mirror/index regressions without skip.

```text
python scripts/prepare_history_mirror.py --repo SOURCE --output NEW_EXTERNAL_DIRECTORY --tool-python ISOLATED_PYTHON
```

The helper creates a full mirror with `--no-local`, removes its origin remote,
and runs the pinned git-filter-repo with no fetch/push. Every historical path in
every locally reachable commit is classified, so renamed copies are included.
Retained text and commit messages are scanned without printing values. UTF-16/32
BOM text is handled while keeping its encoding. Strong credential patterns and
machine-specific home paths are redacted; unrelated source is preserved.

The original refs/worktree are not rewritten. The mirror contains committed
history only; current uncommitted fixes must not be lost when eventually adopting
the rewritten base. It is a **history-cleanup candidate, not a release package**.
Origin-tracking refs are temporarily placed in a protected namespace and restored
against filter-repo's ref map, preserving remote-only tips and symbolic aliases.
Codex operational capture refs are excluded only from the candidate; they can
point to tree snapshots rather than commits. Other non-commit user refs require
manual review. Original operational refs remain untouched.

Binary/oversize retained assets are listed as uninspected. Pattern scanning cannot
prove the absence of every secret. Raw 64-hex text needs manual interpretation;
it can be either a wallet key or a public hash. Never automatically delete it
solely because of its shape.

## Final acceptance

After reviewing the candidate, preserve outstanding source work, reconcile all
intended branches/tags, and coordinate any destructive remote ref replacement.
Verify the actual remote refs, old sensitive-object exposure and host caches/PR
references afterward. A clean local mirror does not mean GitHub was cleaned.
Rotate/revoke exposed provider keys independently; wallet custody and any real
fund movement are separate authorized operations. Preserve authorship and
copyright metadata rather than anonymizing legitimate project credit.

Primary procedure references:
[GitHub sensitive-data removal](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository),
[git-filter-repo](https://github.com/newren/git-filter-repo).
