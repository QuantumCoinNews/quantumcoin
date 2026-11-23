"""Audit an isolated Windows source snapshot; findings never mean success."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import sys
import tempfile

from go_source_audit import run
from source_hygiene import working_source_inventory

ROOT = Path(__file__).resolve().parents[1]
GOVULNCHECK = "golang.org/x/vuln/cmd/govulncheck@v1.9.0"
PIP_AUDIT = "pip-audit==2.10.1"


def go_findings(output):
    """JSON mode exits zero even when vulnerable code is found."""
    decoder = json.JSONDecoder()
    messages, findings = [], []
    remaining = output.strip()
    while remaining:
        if remaining.startswith("go: downloading "):
            line, _, remaining = remaining.partition("\n")
            if not re.fullmatch(r"go: downloading \S+ \S+", line):
                raise ValueError("unrecognized Go download output")
            remaining = remaining.strip()
            continue
        value, end = decoder.raw_decode(remaining)
        if not isinstance(value, dict):
            raise ValueError("unexpected govulncheck message")
        if "error" in value:
            raise ValueError("govulncheck reported a scan error")
        messages.append(value)
        if "finding" in value:
            finding = value["finding"]
            if not isinstance(finding, dict) or not finding.get("osv"):
                raise ValueError("invalid govulncheck finding")
            findings.append(finding)
        remaining = remaining[end:].strip()
    if not any("config" in item for item in messages):
        raise ValueError("govulncheck did not provide a scan configuration")
    return findings


def npm_findings(output):
    value, _ = json.JSONDecoder().raw_decode(output.lstrip())
    counts = value.get("metadata", {}).get("vulnerabilities", {})
    required = ("info", "low", "moderate", "high", "critical", "total")
    if value.get("error") or any(type(counts.get(k)) is not int or counts[k] < 0 for k in required):
        raise ValueError("npm audit did not provide valid vulnerability counts")
    return counts


def python_findings(value):
    dependencies = value.get("dependencies")
    if not isinstance(dependencies, list) or not dependencies:
        raise ValueError("pip-audit did not scan dependencies")
    findings = []
    for dependency in dependencies:
        if dependency.get("skip_reason") or not isinstance(dependency.get("vulns"), list):
            raise ValueError("pip-audit skipped or could not inspect a dependency")
        findings.extend({"package": dependency["name"], **v} for v in dependency["vulns"])
    return findings


def audit(output):
    if os.name != "nt":
        raise ValueError("this acceptance profile currently targets Windows only")
    lock = json.loads((ROOT / "configs/windows-build-toolchain.json").read_text(encoding="utf-8"))
    environment = {
        k: v for k, v in os.environ.items()
        if not k.upper().startswith(("QC_", "TG_", "TELEGRAM_", "OPENAI_", "GO", "CGO_",
                                     "GIT_", "NPM_CONFIG_", "NODE_", "PIP_", "PYTHON"))
        and not any(word in k.upper() for word in ("TOKEN", "SECRET", "PASSWORD", "API_KEY"))
    }
    environment.update(GOENV="off", GOWORK="off", GOTOOLCHAIN="local", CGO_ENABLED="1",
                       GOFLAGS="-mod=readonly -buildvcs=false -p=2", PIP_CONFIG_FILE=os.devnull)
    receipt = {"scope": "Windows source dependency security", "checks": {}, "ok": False,
               "timestampUtc": datetime.now(timezone.utc).isoformat(),
               "scanners": {"go": GOVULNCHECK, "python": PIP_AUDIT}}

    def save():
        output.write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")

    def execute(name, command, directory, timeout=900):
        record, log = run(command, directory, environment, name, timeout)
        path = output.with_name(output.stem + "-" + name + ".log")
        path.write_text(log, encoding="utf-8")
        receipt["checks"][name] = {**record, "log": str(path)}
        save()
        return record, log

    with tempfile.TemporaryDirectory(prefix="qc-dependency-security-") as temporary:
        base = Path(temporary)
        source = base / "source"
        source.mkdir()
        fingerprint = hashlib.sha256()
        names = working_source_inventory(ROOT)["sourcePaths"]
        for name in names:
            data = (ROOT / name).read_bytes()
            target = source / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(data)
            fingerprint.update(name.encode() + b"\0" + hashlib.sha256(data).digest())
        receipt["source"] = {"files": len(names), "sha256": fingerprint.hexdigest()}
        for level in ("USER", "GLOBAL"):
            path = base / (level.lower() + ".npmrc")
            path.touch()
            environment["NPM_CONFIG_" + level + "CONFIG"] = str(path)
        environment["NPM_CONFIG_REGISTRY"] = "https://registry.npmjs.org"
        node = Path(shutil.which("node") or "")
        npm = node.parent / "node_modules/npm/bin/npm-cli.js"
        if not npm.is_file():
            raise ValueError("the Windows Node installation must include npm")
        compiler_paths = {}
        for key, default in (("CC", "gcc"), ("CXX", "g++"), ("AR", "ar")):
            binary = shutil.which(environment.get(key, default), path=environment.get("PATH", environment.get("Path", "")))
            if not binary:
                raise ValueError("missing MinGW tool: " + key)
            compiler_paths[key] = Path(binary).resolve()
        if len({p.parent for p in compiler_paths.values()}) != 1:
            raise ValueError("CC, CXX and AR must belong to the same MinGW directory")
        receipt["compilerFamily"] = {k: str(v) for k, v in compiler_paths.items()}
        versions = [
            ("go-version", ["go", "version"], "go version " + lock["go"]["version"] + " windows/amd64"),
            ("node-version", [str(node), "--version"], "v" + lock["node"]["version"]),
            ("npm-version", [str(node), str(npm), "--version"], lock["npm"]["version"]),
            ("python-version", [sys.executable, "--version"], "Python " + lock["python"]["version"]),
            ("ffmpeg-version", ["ffmpeg", "-version"], "ffmpeg version " + lock["ffmpeg"]["version"]),
            ("gcc-version", [str(compiler_paths["CC"]), "-dumpfullversion"], lock["mingw"]["version"]),
            ("gxx-version", [str(compiler_paths["CXX"]), "-dumpfullversion"], lock["mingw"]["version"]),
            ("gcc-target", [str(compiler_paths["CC"]), "-dumpmachine"], "x86_64-w64-mingw32"),
        ]
        for name, command, expected in versions:
            record, log = execute(name, command, source, timeout=60)
            first_line = log.strip().splitlines()[0] if log.strip() else ""
            matches = first_line == expected
            if name == "ffmpeg-version":
                matches = matches or first_line.startswith(expected + "-") or first_line.startswith(expected + " ")
            if record["exitCode"] != 0 or not matches:
                raise ValueError("toolchain version does not match the Windows lock: " + name)
        tools = base / "audit-tools"
        python = tools / "Scripts/python.exe"
        record, _ = execute("audit-venv", [sys.executable, "-m", "venv", str(tools)], source)
        if record["exitCode"] != 0:
            return False
        record, _ = execute("audit-install", [str(python), "-m", "pip", "--isolated", "install",
                                             "--index-url", "https://pypi.org/simple", PIP_AUDIT], source)
        if record["exitCode"] != 0:
            return False
        record, log = execute("go", ["go", "run", GOVULNCHECK, "-json", "./..."], source)
        findings = go_findings(log)
        receipt["checks"]["go"]["vulnerabilityIDs"] = sorted({v["osv"] for v in findings})
        receipt["checks"]["go"]["accepted"] = record["exitCode"] == 0 and not findings
        for name, directory in (("root", source), ("webwallet", source / "webwallet"),
                                ("publisher", source / "autopublisher")):
            record, log = execute("npm-" + name, [str(node), str(npm), "audit", "--package-lock-only", "--json"], directory)
            counts = npm_findings(log)
            receipt["checks"]["npm-" + name].update(vulnerabilities=counts,
                accepted=record["exitCode"] == 0 and not any(counts.values()))
        for name, requirements in (("python", "telegram_game/requirements.lock"),
                                   ("privacy-python", "scripts/privacy-tools.requirements.txt")):
            python_report = base / (name + "-audit.json")
            record, _ = execute(name, [str(python), "-m", "pip_audit", "-r", requirements,
                                       "--disable-pip", "--require-hashes", "-f", "json", "-o", str(python_report)], source)
            findings = python_findings(json.loads(python_report.read_text(encoding="utf-8")))
            receipt["checks"][name].update(findings=findings, accepted=record["exitCode"] == 0 and not findings)
        receipt["ok"] = all(item.get("accepted", item["exitCode"] == 0)
                            for item in receipt["checks"].values())
        save()
    return receipt["ok"]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    target = args.output.resolve()
    if target.is_relative_to(ROOT) or target.exists():
        parser.error("choose a new output file outside the checkout")
    try:
        return 0 if audit(target) else 1
    except (ValueError, OSError, KeyError) as exc:
        print("Dependency security acceptance failed: " + str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
