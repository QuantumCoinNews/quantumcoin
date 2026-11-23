# Stack verification and cleanup

Run `python scripts/runtime_config_preflight.py` in the intended service
environment. It checks the shared API and publisher monitoring origins,
non-loopback API authorization, and absolute paths for game and publisher
state. The node separately rejects unknown or conflicting network aliases at
startup and validates the selected network against its chain file and ports.
Service-specific environment settings continue to be validated by their owners.

Run `python scripts/verify_clean_stack.py --output <absolute-path-outside-repo>`
from the checkout. The command restores dependencies and runs Go tests,
WebWallet tests and production build, the Autopublisher build and tests, and
Telegram Game unittest discovery. The receipt records the source fingerprint,
toolchain versions and command exit codes. The source snapshot can include
uncommitted edits and the receipt identifies that case. The manual Windows CI
workflow applies the same gate with locked toolchains after requiring a clean,
source-only checkout. It then compares two release builds.

Run `python scripts/source_hygiene.py --cleanup-plan` for a read-only
classification of index and HEAD paths. The output records whether each
candidate is still present in the working directory. Before removing a path,
confirm source ownership with the dependency graph, verify build and regression
results against the replacement, and apply the recovery or custody policy for
the candidate class. Treat untracked operator files and old external delivery
archives as separate inventories.
