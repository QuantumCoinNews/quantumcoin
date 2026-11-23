# Canonical economy and configuration

Consensus authority lives in `consensus/canonical_economy_policy_v1.go`,
`canonical_runtime_parameters_v1.go`, `canonical_denomination_v1.go` and the
versioned transaction/subsidy validators in the same package. The definition is
grouped by subject inside one authoritative package; it is not an editable JSON
policy. Existing whole-QC subsidy remains 50, 25, 12, 6, 3, 1 with a 1-QC floor,
remaining-supply cap, and a 2,102,400-block halving interval. Legacy wire values
are not converted or reinterpreted. Atom protocols use 1 QC = 1,000,000 atoms.

`config.Default`, mainnet immutable checks, config reward shares, and blockchain
economy adapters reference those constants. `blockchain/economy_policy.go` is
an adapter; parameterized reward helpers remain explicit local/test utilities.
`config/params.go` is a retired compatibility shim: mutating its globals grants
no consensus authority. Its size/mining placeholders are not active rules.
Network profile overrides are validated by the existing profile boundaries.
The legacy ten-year config horizon does not stop mining or distribute remaining
supply. Fee parameter finalization and explicit mainnet activation remain gated.

Generate seven economy/recovery documentation examples using:

```powershell
go run ./cmd/policy-export
go run ./cmd/policy-export --check
go test ./policyexamples ./consensus ./config
```

The examples now serialize their complete Go policy types directly. Old
handwritten wrappers such as `burn`, `economy` and `rules` are removed; consumers
must use the versioned fields in the generated document. These are documentation
fixtures, not node configs or activation approvals. The testnet skeleton contains
explicit testnet-only operational defaults; it also takes monetary/genesis
values from consensus. Other configs (queues, load tests, persistence, hardfork
release evidence) describe different contracts and do not redefine QC issuance.

The live Telegram economy panel obtains policy/context from the authenticated Go
node. Its offline Python compatibility models do not govern consensus. The
Python recovery runbook reads the checked-in generated Go declaration, with no
second artifact catalogue. Deploy that fixture alongside the Python package.

The safe load plan targets QC `/api/health`; Telegram's own `/health` is a
different service contract and remains unchanged.
