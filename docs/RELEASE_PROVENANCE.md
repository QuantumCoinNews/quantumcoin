# Windows release provenance

The distributable package has one node executable, one native wallet, and one
browser launcher. Their source packages are `.`, `./cmd/wallet`, and
`./cmd/quantumcoin-gui`, respectively. The old `release/` working directory
contains operator runtime state and maintenance scripts. Historical executable
copies belong in private archive storage. The 27A checkpoint records a
historical comparison. The signed package manifest controls release approval.

## Build from an authorized commit

Run the builder on Windows with Python 3.11+, Git, Go, and MinGW for amd64.
Provide the full reviewed Git commit and the versions of the provisioned
compilers. The builder checks out that commit twice in isolated directories,
with separate Go build caches. It verifies module data, builds with
`-mod=readonly`, `-trimpath` and VCS metadata, and compares binary bytes.
Every executable must identify the same commit and a clean source checkout.

The Windows reproducibility workflow uses the reviewed commit and
`configs/windows-build-toolchain.json` for exact versions. Go and the WinLibs
MinGW-w64 UCRT archives are verified by SHA-256 before extraction.
The compiler, binutils, linker and native headers come from that same archive.
The workflow checks source hygiene before provisioning and uploads the unsigned
manifest, SBOM, provenance and toolchain lock as CI evidence. A hosted run and
release signature are separate evidence. Module/toolchain directives and exact
Go acceptance versions are read from the reviewed source; old diagnostic Go
versions are not release approval.
[Go downloads](https://go.dev/dl/) · [WinLibs release](https://github.com/brechtsanders/winlibs_mingw/releases/tag/15.1.0posix-13.0.0-ucrt-r2)

```powershell
python tools/qc_release.py build --commit $Commit --version $Version `
  --go-version $GoVersion --cc $MinGWCompilerPath --cc-version $MinGWVersion `
  --output $NewPackageDirectory
```

Choose a new directory outside the checkout. The resulting candidate contains
`quantumcoin.exe`, `wallet-gui.exe`, `quantumcoin-launcher.exe`,
`run_miner.cmd`, `_qc_api_mempool_miner.ps1`, `sbom.cdx.json`,
`provenance.json`, and `release-manifest.json`. The two miner client scripts
come from the same clean commit and their bytes are included in both the
manifest and provenance.

Distribute only the verified package directory. The legacy `release/` directory
is local operator runtime. Keep wallet and chain files, balance and recipient
caches, logs, checkpoints and backups in operator storage outside the package.
An extra file or folder causes package verification to fail.

The CycloneDX SBOM lists Go modules linked into each executable and the Go
standard library version. The in-toto statement binds the commit, tree, build
options, compiler identities and artifact hashes. The recorded reproducibility
check covers two builds on one host. Reproduction on another host requires
the same Go distribution, MinGW compiler, linker, libraries and SDK.
[CycloneDX 1.6 JSON specification](https://cyclonedx.org/docs/1.6/json/)

## Sign and verify

Review the candidate before signing. The signing authority uses the release
key in independent custody. Distribute the public allowed-signers file and
expected commit through a trusted channel separate from the package.

```powershell
$Manifest = Join-Path $PackageDirectory 'release-manifest.json'
ssh-keygen -Y sign -f $SigningKey -n quantumcoin-release-v1 $Manifest
python tools/qc_release.py verify --directory $PackageDirectory `
  --commit $Commit --trusted-signers $AllowedSigners
```

The allowed-signers entry uses principal `quantumcoin-release` and the
approved public key. A package is valid only if its signature, externally
expected commit, exact file set, SHA-256 hashes, binary build metadata, SBOM
and provenance all agree. Keep the verified directory under trusted write
control when distributing or using it.

Compiler version and executable SHA-256 identities are mandatory provenance
fields. The builder reads the toolchain lock from the selected source commit
and binds its byte hash into provenance. Verification uses the independently
trusted lock in this checkout, or explicit `--toolchain-lock` outside the
package. A package cannot select its own compiler policy, omit identity fields
or choose an older Go version through its signed metadata. Observed executable
hashes are attested builder inputs; this verifier alone does not establish that
an entire SDK was extracted from a particular archive. The CI bootstrap verifies
archives and the two-build check compares output bytes.

## Initial testnet authority

Initial testnet hardfork activation deliberately remains disabled. Real
founder/admin identities and quorums must be reviewed before the first hardfork;
test authorities are never production trust. A release signature does not grant
a hardfork vote or replace quorum validation.

The initial release custodian is the QuantumCoinNews operator, using a separate
passphrase-protected Ed25519 OpenSSH key. Wallet keys and login/API credentials
are not release keys. Prepare the key interactively in PowerShell 7.3+:

```powershell
pwsh -NoProfile -File tools/prepare_release_signer.ps1 -Directory $NewExternalCustodyDirectory
```

The parent directory must already exist outside Git. The helper restricts NTFS
access to the operator and SYSTEM, invokes OpenSSH's own hidden passphrase prompt
and refuses headless input or an unprotected key. It does not overwrite existing
custody or sign/publish a release. Review the displayed PUBLIC fingerprint;
keep the encrypted private key, its independent backup and passphrase in custody.
Only the public allowed-signers entry goes to verifiers through a trusted channel.
Actual signer creation/fingerprint adoption and real release signing remain
operator acceptance steps; fixture signatures do not complete them.
[OpenSSH signature commands](https://man.openbsd.org/ssh-keygen.1)

## Audit mode

Each of the seven PowerShell audits accepts `-SourceOnly` for source checks.
For release checks supply `-ReleaseDirectory`, `-ReleaseCommit`, and
`-TrustedSigners` together. Directory and trust paths must be absolute.
Run release-mode audits in a clean checkout of the authorized commit.
Source-only reports say which mode was used. Signed release checks reject an
unsigned package, modified file, extra executable, or mismatched commit.
For a shareable snapshot of build and analysis results, use the
[Go source audit receipt](GO_SOURCE_AUDIT.md).

```powershell
pwsh -NoProfile -File tools/qc_repo_safety_check.ps1 -SourceOnly
pwsh -NoProfile -File tools/qc_repo_safety_check.ps1 `
  -ReleaseDirectory $PackageDirectory -ReleaseCommit $Commit -TrustedSigners $AllowedSigners
```

The local regression suite runs with `python tools/test_qc_release.py`.
Set `QC_RELEASE_E2E_CC` to an explicit MinGW compiler path to include the
temporary Go/CGo build, signature and PowerShell gate test.
