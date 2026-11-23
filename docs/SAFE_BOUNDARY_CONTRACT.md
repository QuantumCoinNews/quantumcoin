# Safe-boundary declaration contract

Safe-boundary maps are serialization and validation metadata. They are not
capabilities, authorization tokens or proofs that money moved safely. Callers
may edit their own policy proposal; validators must reject any deviation from
the complete canonical key/value set, including unknown, missing and replaced
keys. Validation must not silently repair the caller's proposal.

Every policy factory must return independently owned maps. Editing one proposal
must not change another proposal, a subsequent default or a live runtime. The
node startup gate validates the five active monetary/control policy packages;
it does not retain caller maps as consensus authority. Actual signatures,
UTXO, supply, burn, vesting and fork activation remain in the canonical runtime.

The preparation persistence and queue plans, and mining-transition proposal,
have the same declaration discipline. They do not become activated features
because a boundary says `true`. Queue runtime mode has explicit false values
for its plan-only/start-worker declarations; exact validation checks those too.
The queue constructor owns a copy of its validated live policy.

Caller mutation concurrent with validation of the *same* Go map is unsupported:
callers must synchronize access or pass an owned snapshot. Independent proposals
may be validated and mutated concurrently. No claim is made that public Go
structs sandbox arbitrary in-process code.

Acceptance tests change/delete/replace every key for the five startup policies,
mining transition, persistence and both queue modes; validate independent maps
concurrently; call the real node gate for every corrupted runtime-policy key;
and check that validation leaves caller input unchanged. Behavioral authority is
also covered separately by the canonical runtime tests recorded in the policy,
queue, persistence and internal acceptance reports.
