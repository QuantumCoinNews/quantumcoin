"""Run Go checks from an isolated source snapshot and emit a shareable receipt."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time

from source_hygiene import working_source_inventory


ROOT = Path(__file__).resolve().parents[1]
STATICCHECK = "honnef.co/go/tools/cmd/staticcheck@v0.8.2"
TARGETS = {
    "node": (".", "0", "quantumcoin"),
    "miner": ("./cmd/miner", "0", "quantumcoin/cmd/miner"),
    "wallet": ("./cmd/wallet", "1", "quantumcoin/cmd/wallet"),
    "launcher": ("./cmd/quantumcoin-gui", "0", "quantumcoin/cmd/quantumcoin-gui"),
}
CORE_PACKAGES = [".", "./blockchain", "./config", "./node", "./p2p", "./walletsigner", "./webui"]
GO_COMMAND_TIMEOUT = 90 * 60
GO_PACKAGE_TEST_TIMEOUT = "45m"


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT)


def snapshot(destination):
    candidates = working_source_inventory(ROOT)["sourcePaths"]
    fingerprint = hashlib.sha256()
    count = go_bom = 0
    for name in candidates:
        source = ROOT / name
        if not source.exists():
            continue
        if not source.is_file() or source.resolve() != source:
            raise ValueError("source snapshot contains an indirect or non-file entry")
        data = source.read_bytes()
        output = destination / name
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_bytes(data)
        fingerprint.update(name.encode("utf-8") + b"\0" + hashlib.sha256(data).digest())
        count += 1
        if output.suffix == ".go" and data.startswith(b"\xef\xbb\xbf"):
            go_bom += 1
    return {"files": count, "sha256": fingerprint.hexdigest(), "goFilesWithBOM": go_bom}


def run(command, work, env, display=None, timeout=GO_COMMAND_TIMEOUT, input_data=None):
    started = time.monotonic()
    actual_command = list(command)
    if not os.path.dirname(os.fspath(actual_command[0])):
        executable = shutil.which(os.fspath(actual_command[0]), path=env.get("PATH", env.get("Path", "")))
        if not executable:
            raise FileNotFoundError("Command not found in supplied PATH: " + os.fspath(actual_command[0]))
        actual_command[0] = executable
    with subprocess.Popen(actual_command, cwd=work, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          stdin=subprocess.PIPE if input_data is not None else None,
                          start_new_session=os.name != "nt") as process:
        try:
            stdout, stderr = process.communicate(input=input_data, timeout=timeout)
            output = (stdout + b"\n" + stderr).decode("utf-8", "replace")
            record = {"exitCode": process.returncode}
        except subprocess.TimeoutExpired as expired:
            if os.name == "nt":
                try:
                    subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"], capture_output=True,
                                   timeout=15, check=False)
                except (OSError, subprocess.TimeoutExpired):
                    pass
            else:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
            if process.poll() is None:
                process.kill()
            process.wait()
            record = {"exitCode": None, "error": "timeout"}
            output = ((expired.output or b"") + b"\n" + (expired.stderr or b"")).decode("utf-8", "replace")
    record.update(command=display or command, executable=os.fspath(actual_command[0]),
                  seconds=round(time.monotonic() - started, 2))
    print(f'{record["command"]}: {record["exitCode"]}', flush=True)
    return record, output


def check(name, command, work, env):
    record, output = run(command, work, env)
    record["testHarnessTimeout"] = "panic: test timed out after " in output
    record["failedPackages"] = sorted(set(re.findall(r"^FAIL\s+(quantumcoin(?:/[\w./-]+)?)\s", output, re.M)))
    record["failedTests"] = sorted(set(re.findall(r"^--- FAIL: (Test[A-Za-z0-9_]+)(?:\s|/)", output, re.M)))
    if name == "staticcheck":
        record["diagnosticCodes"] = {}
        for line in output.splitlines():
            if not line.startswith("{"):
                continue
            try:
                code = json.loads(line).get("code", "")
            except ValueError:
                continue
            if re.fullmatch(r"[A-Z]{1,4}[0-9]+", code):
                counts = record["diagnosticCodes"]
                counts[code] = counts.get(code, 0) + 1
    return record


def build_targets(work, binaries, env, targets):
    records = {}
    for name, (package, cgo, expected_package) in targets.items():
        binary = binaries / (name + ".exe")
        command = ["go", "build", "-trimpath", "-o", str(binary), package]
        record, _ = run(command, work, {**env, "CGO_ENABLED": cgo},
                        ["go", "build", "-trimpath", "-o", binary.name, package])
        records[name] = record
        if record["exitCode"] == 0:
            evidence, info = run(["go", "version", "-m", str(binary)], work, env,
                                 ["go", "version", "-m", binary.name])
            record["binaryInspection"] = evidence
            match = re.search(r"^\s*path\s+([\w./-]+)$", info, re.M)
            package_id = match[1] if match else None
            record.update(package=package_id, sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                          bytes=binary.stat().st_size, fyne="\tdep\tfyne.io/fyne/v2\t" in info,
                          wails="github.com/wailsapp/" in info)
            record["packageMatch"] = evidence["exitCode"] == 0 and package_id == expected_package
    return records


def audit(output, core_only):
    head = git("rev-parse", "HEAD").decode("ascii").strip()
    dirty = bool(git("status", "--porcelain"))
    packages = CORE_PACKAGES if core_only else ["./..."]
    targets = {name: target for name, target in TARGETS.items() if not core_only or target[1] == "0"}
    commands = {
        "build": ["go", "build", "-trimpath", *packages],
        "test": ["go", "test", f"-timeout={GO_PACKAGE_TEST_TIMEOUT}", "-count=1", *packages],
        "vet": ["go", "vet", *packages],
        "race": ["go", "test", "-race", f"-timeout={GO_PACKAGE_TEST_TIMEOUT}", "-count=1", *packages],
        "staticcheck": ["go", "run", STATICCHECK, "-f", "json", *packages],
    }
    with tempfile.TemporaryDirectory(prefix="qc-go-audit-", ignore_cleanup_errors=True) as temporary:
        base = Path(temporary).resolve()
        work, binaries = base / "source", base / "artifacts"
        work.mkdir()
        binaries.mkdir()
        source = snapshot(work)
        if git("rev-parse", "HEAD").decode("ascii").strip() != head:
            raise ValueError("Git HEAD changed during snapshot")
        env = {k: v for k, v in os.environ.items()
               if not k.upper().startswith(("QC_", "GO", "GIT_", "CGO_", "TG_", "TELEGRAM_"))}
        env.update(GOENV="off", GOWORK="off", GOTOOLCHAIN="local", CGO_ENABLED="1",
                   GOFLAGS="-mod=readonly -buildvcs=false -p=4")
        version_record, version = run(["go", "version"], work, env)
        version = re.search(r"go version (go[0-9.]+ [a-z0-9]+/[a-z0-9]+)", version)
        # Keep test and race runs separate so fixture writes and resource use
        # cannot interfere with another check in the same snapshot.
        binary_records = build_targets(work, binaries, env, targets)
        receipt = {
            "kind": "unsignedGoSourceAudit", "head": head, "dirtyAtSnapshot": dirty,
            "scope": "core" if core_only else "all",
            "timestampUtc": datetime.now(timezone.utc).isoformat(),
            "sourceSnapshot": source, "goVersion": version[1] if version else None,
            "goVersionCheck": version_record, "goFlags": env["GOFLAGS"],
            "checks": {name: check(name, cmd, work, env) for name, cmd in commands.items()},
            "binaries": binary_records,
        }
    receipt["scratchCleanupComplete"] = not base.exists()
    receipt["ok"] = (receipt["scratchCleanupComplete"] and receipt["goVersion"] is not None and
                     all(row["exitCode"] == 0 for row in receipt["checks"].values()) and
                     all(row.get("packageMatch") for row in receipt["binaries"].values()))
    output.write(json.dumps(receipt, indent=2) + "\n")
    return 0 if receipt["ok"] else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--core-only", action="store_true", help="check core packages and build CGO-disabled targets")
    args = parser.parse_args()
    target = args.output.resolve()
    if target.is_relative_to(ROOT):
        parser.error("choose an output file outside the checkout")
    with target.open("x", encoding="utf-8") as stream:
        return audit(stream, args.core_only)


if __name__ == "__main__":
    raise SystemExit(main())
