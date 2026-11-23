# secp256k1 analyzer compatibility review

Go 1.26 deprecates the exported ECDSA scalar and coordinates. Its replacement
`PrivateKey.Bytes`, `ParseRawPrivateKey` and `PublicKey.Bytes` support NIST curves,
not QuantumCoin's secp256k1 curve. Switching QC keys to another curve would break
existing addresses, signatures and recovery. Reflection or unchecked type tricks
are not substitutes for a supported encoding API.

The existing pinned `decred/secp256k1/v4` backend now owns key generation, scalar
import and public-point parsing/serialization. The public ECDSA facade remains
for source and stored-key compatibility. New helpers in
`utils/secp_ecdsa_compat.go` validate the curve, scalar range and private/public
pair before exporting. Encodings remain a 32-byte scalar and uncompressed 65-byte
point; wallet addresses, transaction digests, low-S signatures and disk schemas
are not changed. Existing API consumers no longer read or write raw ECDSA fields.

Three narrow, reviewed SA1019 directives remain at the facade boundary:

- Read secp256k1 X/Y for validation and native backend serialization; Go has no
  supported secp256k1 PublicKey.Bytes path.
- Read D for bounded validated scalar export; Go has no supported secp256k1
  PrivateKey.Bytes path.
- Clear the discarded, exclusively owned facade's logical scalar after last use.
  This is best-effort cleanup, not a guarantee of forensic zeroization of all
  Go heap copies. Never run cleanup concurrently with signing or reuse the key.

Each directive is attached to one statement, not a package-wide analyzer exclusion.
The compatibility test deliberately fails for review if Go starts supporting the
curve through Bytes. Known generator point, scalar roundtrip, detached exports,
invalid/mismatched/cleared keys, old wallet stores, mnemonic recovery, signatures
and runtime admission are tested. A future native-only public-key type migration
is not required to change stored wallet identities and is not claimed here.

Two other narrow directives document intentional negative tests: SA1012 verifies
nil-context shutdown rejection; SA9005 verifies JSON cannot manufacture an opaque
verified NFT program. Those behaviors remain tested. No cryptographic warning or
production nil-context call is globally disabled.

Analyzer closure means old call sites were migrated and these necessary boundary
operations/negative tests were reviewed with evidence. It does not mean deprecated
Go fields ceased to exist. Historical findings and the complete pre-change register
remain archived; the active register only removes records supported by final tests.
