# Reviewed Git changes

Run `./safe-push.ps1` for a read-only preflight. It uses Git's ignore rules and
reports tracked paths that match them. Review those paths and any staged
findings individually before committing.

Stage the intended paths explicitly and review the staged diff. Use
`./safe-push.ps1 -Commit -Message "Describe the change"` to commit that selection.
Add `-Push` only when publishing the verified commit to the selected remote and
branch is intended. The script leaves unstaged files and tracked ignored paths
in place.
