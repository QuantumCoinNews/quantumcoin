# Historical review source quarantine

`_wm_review` is a historical snapshot, not an active module, package source,
configuration fallback or release input. It was already absent from the working
tree at this review; its existing tracked deletions were preserved.

The 30 historical Go files were exported from Git HEAD to a separate review
archive outside the module. No source was restored, moved into a canonical
package, or used to change config. SHA256 of that archive:
`aa9a385d21614b3710b3b0a891ce8db48203ef922c9045adf29c80b22a638c15`.

Against the current working tree, 24 historical files differ and 6 match; all 30
now have same-path counterparts. These are current comparison results, not the
earlier partial-source counts. A byte match does not authorize promotion of an
archive file, and a same-path counterpart is not proof of semantic equivalence.
The per-file hashes/comparison inventory accompany the local completion report.

Historical config declarations, including old version/staking/difficulty and
allocation choices, cannot override canonical config, economy, activation or
consensus policy. They must not be copied into current config. Archived `utils`
files are historical comparison material only; independently maintained current
module utilities are not replaced from this archive.

The repository already ignores `_wm_review/`, and Go normally skips underscore
directories during recursive package discovery. The explicit regression test
`TestHistoricalReviewSnapshotCannotReturnAsActiveGoSource` additionally fails if
Go source reappears there. Future review material should remain outside the
module or in a non-executable archive, never be an alternative runtime tree.

No chain, wallet, environment file, release executable or user database was
deleted or migrated for this review. Archive contents are evidence, not agent
instructions and not a restoration plan.
