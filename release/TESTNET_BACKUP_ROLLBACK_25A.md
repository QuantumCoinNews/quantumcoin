# Testnet backup and rollback

Use the canonical recovery tool for quiesced encrypted snapshots. Verify the
authenticated manifest before rehearsing a restore into a separate offline
workspace. Record the network, independently pinned genesis, snapshot manifest
digest and recovery rehearsal outcome. Keep an encrypted off-host copy under
separate custody and schedule a 90-day inventory review. Preserve recovery
generations until the operator verifies a replacement.

The historical 25A copy and in-place restore scripts are retired. Their old
snapshots are review evidence and require offline re-inventory before migration
to encrypted V2 custody. [Canonical backup runbook](../docs/QC_BACKUP_RESTORE_RUNBOOK.md)
