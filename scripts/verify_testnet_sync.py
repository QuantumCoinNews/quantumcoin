"""Verify bounded P2P catch-up from an isolated working-tree source snapshot."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import tempfile

from go_source_audit import ROOT, git, run, snapshot


def verify(output, focused=False):
    base = Path(tempfile.mkdtemp(prefix="qc-testnet-sync-"))
    source = base / "source"
    source.mkdir()
    receipt = {"timestampUtc": datetime.now(timezone.utc).isoformat(),
               "head": git("rev-parse", "HEAD").decode().strip(),
               "branch": git("branch", "--show-current").decode().strip(),
               "source": snapshot(source), "scratch": str(base), "checks": {},
               "scope": "focused" if focused else "sync-core-vet-race", "ok": False}
    env = {k: v for k, v in os.environ.items()
           if not k.upper().startswith(("QC_", "TG_", "TELEGRAM_", "GO", "CGO_", "GIT_"))}
    env.update(GOENV="off", GOWORK="off", GOTOOLCHAIN="local", CGO_ENABLED="1",
               GOFLAGS="-mod=readonly -buildvcs=false -p=2")
    pattern = "TestTwoProcessSync|TestSyncRejects|TestSyncNodeProcess"
    commands = [("sync", ["go", "test", "-count=1", "-timeout=180s", "-v", "./p2p", "-run", pattern])]
    if not focused:
        commands += [("core", ["go", "test", "-count=1", "-timeout=300s", "./blockchain", "./p2p", "./node"]),
                     ("vet", ["go", "vet", "./blockchain", "./p2p", "./node"]),
                     ("race", ["go", "test", "-race", "-count=1", "-timeout=240s", "-v", "./p2p", "./blockchain",
                               "-run", pattern + "|TestReplacementReplays|TestSerializeBlockchainConcurrent|TestReplacementChooses"])]
    for name, command in commands:
        record, log = run(command, source, env, name, timeout=600)
        path = output.with_name(output.stem + "-" + name + ".log")
        path.write_text(log, encoding="utf-8")
        record.update(log=str(path), failedTests=re.findall(r"^--- FAIL: (\S+)", log, re.M),
                      expectedHelperSkips=re.findall(r"^--- SKIP: (TestSyncNodeProcess)\b", log, re.M))
        receipt["checks"][name] = record
        output.write_text(json.dumps(receipt, indent=2), encoding="utf-8")
        if record["exitCode"] != 0:
            print(log[-12000:], flush=True)
            return 1
    # Bind evidence to actual bytes of all changed implementation/test inputs.
    paths = ["blockchain/peer_sync.go", "p2p/chain_sync.go", "p2p/chain_sync_test.go",
             "p2p/network.go", "p2p/node.go", "p2p/message.go", "p2p/message_size_validation_v1.go"]
    receipt["inputs"] = {path: hashlib.sha256((source/path).read_bytes()).hexdigest() for path in paths}
    receipt["ok"] = True
    output.write_text(json.dumps(receipt, indent=2), encoding="utf-8")
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--focused", action="store_true")
    args = parser.parse_args()
    output = args.output.resolve()
    if output.is_relative_to(ROOT) or output.exists():
        parser.error("choose a new receipt outside the checkout")
    return verify(output, args.focused)


if __name__ == "__main__":
    raise SystemExit(main())
