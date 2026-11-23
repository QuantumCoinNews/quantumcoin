# Mining startup options

`miner.MergeOptionsV1` is the shared option resolver for `miner.Start` and
`MinerTokenBridge.Options` / `StartMiningWithToken`.

| Field | Merge rule |
| --- | --- |
| OnBlock, OnError, OnTick, Broadcaster | Last non-nil function wins; nil preserves the earlier function |
| Interval | Last positive duration wins; zero/negative durations preserve the earlier value |
| Animate | Last packet wins, including false in an otherwise empty packet |

Without option packets the existing compatibility miner defaults apply:
1200 milliseconds and animation enabled. The bridge now uses these same
defaults; callers wanting animation disabled should pass `Animate: false`.

The bridge copies the resolved options and wraps OnBlock once to process its
token record, report its error, and deliver the caller's canonical block
notification. Its error callback, tick callback and broadcaster are retained.
Callbacks are selected, not concatenated: an overridden callback is not invoked.

These are startup rules. They do not make an already-running miner accept live
option changes or attach a token bridge. See the
[attach and exclusive startup contract](MINER_ATTACH_CONTRACT.md).
Interval scheduling inside the mining loop, consensus difficulty,
PoW, rewards and halving are unchanged. The token bridge remains disabled by
default; resolving options does not enable it.
