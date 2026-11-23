# Dependency ownership and Windows security acceptance

Review date: 2026-11-12. This acceptance profile is Windows AMD64 only. Linux,
macOS, other architectures, hosted CI execution and signed release provenance
are separate acceptance work. A local test or clean scan is not live deployment.

## Canonical owners

| Component | Dependency source | Runtime boundary |
| --- | --- | --- |
| Go node, miner and native wallet | Root go.mod / go.sum | Root node; cmd/miner; cmd/wallet; cmd/quantumcoin-gui. No npm dependency is needed for the headless node. |
| Telegram Game and bot | telegram_game/requirements.in -> requirements.lock | telegram_game.runtime:app and the documented bot. Install the complete hash-locked environment. |
| Autopublisher | autopublisher/package.json / package-lock.json | publisher.mjs and its canonical TypeScript pipeline. Tests use fixtures; successful mocks do not establish live platform delivery. |
| Separate browser balance viewer | webwallet/package.json / package-lock.json | Read-only React application; the node embeds its separately listed landing and operator miner assets. |
| Root social compatibility tools | Root package.json / package-lock.json | Compatibility dependency tree, separate from Go consensus. |

## Validated tool versions

The machine-readable source is configs/windows-build-toolchain.json:
Go 1.26.11, Node 22.23.5, Python 3.14.10, MinGW GCC 15.1.0 and FFmpeg/FFprobe 9.0.2.
The Go module requires Go 1.26.0 and selects toolchain go1.26.11. The previous
Go 1.21 declaration is no longer compatible with the updated Go dependencies.

VS Code workspace settings select the managed Go installation and gopls v0.23.0.
Use a new project terminal after changing tool paths. Other global installations
and already running services are separate from this workspace configuration.
C, C++ and ar must resolve to one MinGW installation; the Windows CI wrapper sets
all three explicitly. The command runner resolves bare tool names using its
supplied PATH rather than silently selecting the parent process's older Go.

Python 3.14.10 was selected because the latest Python 3.13 security source release
has no Windows installer. Windows installers and archives are versioned and
SHA-256 checked; the Python installer also passed Authenticode verification.
No runtime database/key migration is performed by changing these pins.

## Repeatable security gate

From the validated project environment, choose a new output file outside the checkout:

```powershell
python scripts/verify_dependency_security.py --output C:/QuantumCoinAudit/dependency-security.json
```

Create the destination directory first. The command copies only the canonical
source inventory to temporary storage, excluding actual keys, environment files,
chain data, databases and historical runtime artifacts. It does not publish,
make payments, start the node or open an application database.

Pinned scanners: govulncheck v1.9.0 and pip-audit 2.10.1. npm audit examines all
three lockfiles, including development dependencies. pip-audit examines the whole
resolved requirements.lock with dependency resolution disabled. Scanner tools are
installed separately from the application runtime environment.

The gate rejects any Go finding, nonzero npm vulnerability count, Python finding,
skipped dependency, malformed/missing result or unsuccessful tool invocation.
In particular, govulncheck JSON mode can exit zero with findings: JSON findings
are parsed separately. Advisory data changes over time; every receipt has a UTC
timestamp and source fingerprint. A clean result means the selected scanners
reported no known findings at that time, not that unknown vulnerabilities do not exist.

The Windows CI wrapper runs both verify_clean_stack.py and this security gate
before release reproduction. Parser and tool-path regression tests run with the
clean-stack tooling suite. Staticcheck is pinned to v0.8.2 for Go 1.26 support;
remaining Staticcheck/race work retains its own finding (0247).

## Scoped evidence and remaining work

The dependency candidate uses updated Go crypto/image/net/text/sys/goldmark,
updated Python runtime/bot/cryptography dependencies with newly generated hashes,
and the previously repaired npm locks. Keep historical advisory reports as dated
evidence rather than current package counts. The latest local receipt determines
whether this source snapshot passed; interrupted scans never count as clean.

Hosted CI (0251/0263), clean signed release/provenance (0192/0242), secret rotation
and custody (0170/0198/0239), and live social/BSC/NFT feature acceptance remain
separate. Package updates do not close those findings or activate product features.
