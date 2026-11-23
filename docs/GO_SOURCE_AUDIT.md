# Go source audit receipt

Run the source audit on Windows with Python 3.11+, Go and an amd64 MinGW
compiler available on PATH. Choose a new output file outside the checkout.

```powershell
python scripts/go_source_audit.py --output C:\QuantumCoinAudit\go-source-audit.json
```

The audit copies source inputs into a temporary workspace. The source filter
excludes known runtime, secret, generated and historical paths. The root local
config is also excluded. The receipt records the base Git commit, whether the
working tree was dirty and a fingerprint over the copied paths and bytes.
Uncommitted source inputs are represented by the fingerprint.

The receipt records the actual Go version and exit status for build, tests,
vet, race tests and pinned Staticcheck. Individual node, miner, Fyne wallet
and launcher builds are checked by Go package identity and SHA-256. A nonzero
check remains visible in the receipt. The Staticcheck tool version is pinned
for Go 1.24 support. [Staticcheck 2025.1 release notes](https://staticcheck.dev/changes/2025.1/)

Use `--core-only` to verify the listed node and consensus packages plus the
CGO-disabled targets when the native wallet build is handled separately. The
receipt records the scope and exact package arguments.

The checks run sequentially in the isolated workspace. On timeout the audit
stops the command tree and records the timeout and scratch cleanup status.

Share the JSON receipt. Raw environment dumps, filesystem trees, command
output, absolute paths and binary build-info dumps are kept out of the JSON.
Historical diagnostic logs remain restricted. A Go package list or a
PowerShell wrapper message alone does not establish a successful build.

The receipt describes an unsigned source audit. To distribute executables,
use the clean-commit package builder and independently trusted manifest
described in [Windows release provenance](RELEASE_PROVENANCE.md).
