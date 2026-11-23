# Advisory transaction analysis

Transaction admission belongs to the canonical blockchain validator. HTTP signed
submission, wallet CLI, peer admission and the internal compatibility adapter all
delegate to that validator. AI signals cannot reject a valid transaction, bypass
signature/UTXO/duplicate checks, change fees, mint rewards or move funds.

`GET /api/ai/analysis` observes pending transactions from every ingress together
with recent confirmed history under one chain read lock. It scans at most 200
blocks and 2,000 transaction entries with a 4 MiB transaction cost budget; pending
entries take priority. Response headers identify the bounded scope and truncation.
Coinbase entries are excluded and repeated transaction IDs count once. Amounts are
converted to QC using each transaction's origin protocol height. Analysis does
not mutate or drain the pending pool and keeps no separate transaction cache.

Frequency signals describe transaction timestamps in this sample, not trusted
server arrival times. They are advisory observations, not an ingress rate limiter
or a complete historical fraud detector. Entries outside the sample can be
missed. Recent confirmed history survives normal chain persistence; unconfirmed
history has the same lifetime as the canonical pending pool.

Environment is loaded once on first access. `QC_AI_ENABLED=0`, `false` or `off`
(case insensitive) disables analysis; the default is enabled. `QC_AI_LEVEL`
defaults to `normal`; unknown levels also use `normal`.

| Level | Large transaction signal | Frequency signal in 20 seconds |
| --- | --- | --- |
| off | disabled | disabled |
| low | at least 1,000 QC | at least 10 distinct transactions |
| normal | at least 500 QC | at least 5 distinct transactions |
| aggressive | at least 250 QC | at least 3 distinct transactions |

`SetEnabled` is a process-local, synchronized override of the enable flag.
It initializes environment first and cannot override level `off`. Restart is
required to change environment/level. Telemetry state files do not override these
settings. These levels tune monitoring sensitivity only, never consensus or
transfer acceptance. Frequency analysis uses sorting and a linear sliding window
after sorting; the caller's transaction slice is not reordered.
