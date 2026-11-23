# Windows CI and release acceptance

The supported acceptance target is Windows amd64. The toolchain versions and
download checksums come from `configs/windows-build-toolchain.json`.
Node is installed from the checksum-verified archive in that lock; its bundled
npm version is independently pinned and must match in both stack and dependency
security gates. An arbitrary numeric npm version cannot pass. Python is selected
by the pinned setup-python action and exact version; the workflow does not claim
to verify the standalone installer hash for that action-managed installation.

## Source verification

`windows-reproducibility.yml` runs on pull requests, pushes to main/master, and
manual dispatch. It uses read-only repository permissions, no publishing secrets,
pinned actions, and cancels obsolete runs for the same ref. A failed stage fails
the job; uploading its diagnostic artifacts does not change that result.

`scripts/build_windows_release_ci.ps1` requires the exact selected commit and a
clean checkout. It checks source hygiene, downloads and verifies locked native
tools (Go, MinGW, FFmpeg and Node), then runs:

1. `verify_clean_stack.py`: isolated source snapshot, empty npm configuration,
   `npm ci`, a fresh Python venv with hash-locked dependencies, exact toolchain
   version checks, Go tests/vet/race/Staticcheck, both browser frontends, publisher,
   game integration and schema inventories, and release-trust regression tests.
2. `verify_dependency_security.py`: Go, npm and Python vulnerability scans.
   Findings, scanner errors, skipped dependencies and malformed results fail.
3. `tools/qc_release.py build`: two clean checkouts with distinct build caches,
   matching artifact bytes, binary build metadata, SBOM and provenance.

Go test and race stages use a 45-minute **per-package** test timeout and a
90-minute command deadline. This includes the full blockchain suite, without
`-short`, reduced PoW difficulty, or filtering out slow tests. The job deadline is
240 minutes. A timeout is a failure, never a successful test result.

The clean-stack snapshot's disposable Git HEAD contains path fixtures for hygiene
tests. Its source fingerprint identifies the tested bytes; it is **not** release
provenance. Release builds instead use the real clean source commit.

## Distribution acceptance

Passing CI produces an **unsigned candidate**, not an approved signed release.
The CI receipt explicitly records `signedReleaseAccepted: false`. Production
signer credentials are never supplied to pull-request jobs.

Before distribution, an authorized operator signs the candidate manifest and
verifies it with `tools/qc_release.py verify`, supplying the expected source commit
and an independently trusted signer file outside the source and package folders.
That gate checks signature, exact payload, artifact hashes, source identity,
binary metadata, SBOM and provenance. The release-trust tests use temporary
fixture keys only; they do not approve a production signer or production binary.

The repository administrator must separately make the successful Windows job a
required branch-protection check. A workflow file in an uncommitted workspace
does not establish a hosted CI pass or branch protection. Record the actual run
URL, commit, conclusion and retained evidence before closing final acceptance.
