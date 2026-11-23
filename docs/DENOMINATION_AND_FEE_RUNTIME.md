# QC denomination and micro-fee runtime contract

The single QC scale is `consensus.CanonicalAtomsPerQCV1 = 1,000,000` (six decimals).
`config.QC` is a deprecated alias to the same constant. Use exact decimal strings
at user-facing boundaries and integers for transaction outputs, UTXO accounting,
fees and reward composition. The legacy float Amount field is compatibility/display
information; authoritative value comes from validated integer inputs and outputs.

Protocol matters: LegacyV1 / ContinuityV2 signed wire integers use whole QC;
AtomV3 / NFTV4 signed wire integers use atoms. The verified activation policy
selects the protocol at each height. No old signed payload, history, or wallet
balance is automatically rescaled after changing a config constant. Unsupported
fractional amounts fail at legacy boundaries. Atom protocols require a supported
64-bit runtime and exact amount conversion.

The micro-fee policy targets 0.000001 QC = 1 atom and a congestion ceiling of
0.00001 QC = 10 atoms. End-to-end tests exercise owner signatures, canonical
mempool admission, actual mining, miner fee receipt, disk persistence and trusted
restart for both amounts. Fees are existing value and are not new issuance.
Explicit mainnet startup remains blocked pending final fee parameters and
activation, even though atom representation is implemented. These remain governed
by their existing policy; this change does not silently activate a new mandatory fee floor.

The user confirmed on 5 November 2026 that the existing whole-QC subsidy series
is retained: 50, 25, 12, 6, 3, 1 QC, then the 1-QC floor until remaining supply
is exhausted. The interval is 2,102,400 blocks; year labels are illustrative.
Atom-denominated protocol coinbases convert those whole-QC policy rewards exactly
into atoms and add verified atom fees. No fractional subsidy schedule is enabled,
no old block is reinterpreted, and no new activation is introduced. Go policy
visibility is derived from the canonical subsidy function. Historical fractional
schedule examples are superseded by this explicit decision.

TGWT is a separate BSC token, not QC atoms. A numeric TGWT ledger entry cannot be
cast into QC without an explicit exchange/settlement contract. No automatic
TGWT-to-QC conversion or token-factory fee settlement is enabled by this change.
Future monetary integrations must use the canonical QC scale and exact rate,
rounding, authorization and settlement rules. Existing 3,000,000 TGWT pool and
1 TGWT per verified separate task are unchanged; BSC receipt/payout integration
remains deferred to the final mainnet phase.
