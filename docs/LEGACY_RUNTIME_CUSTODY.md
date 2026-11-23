# Legacy runtime custody

Treat old chain files, database snapshots and wallet material as restricted
runtime history. Keep them outside the source checkout, deployment packages and
operating data directories. The empty old-working-release delivery contains no
payload to rebuild.

Previously delivered legacy and bin chain snapshots were reported as 1,267 and
1,971 blocks, respectively. They shared the filename chain_data.dat and had
different genesis values. The current checkout has empty local placeholders
for those snapshots. The historical block counts are delivery findings.

Review any recovered copy against a trusted network and genesis before
selecting a runtime data directory. Canonical admission checks the pinned
identity and replays the chain before publishing a runtime. Mainnet rejects
untagged legacy files. Two same-named files with different genesis fail
substitution even if their network labels match. See
[chain startup validation](CHAIN_STARTUP_VALIDATION.md).

A wallet key delivered in plaintext is considered exposed. Its custody case
remains open until the operator determines whether the address held funds and
completes any necessary transfer to a freshly generated wallet with encrypted
recovery material. Record the custody decision without secret values. See
[backup and wallet recovery](QC_BACKUP_RESTORE_RUNBOOK.md).
