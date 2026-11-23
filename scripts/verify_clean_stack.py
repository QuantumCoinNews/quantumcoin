"""Restore locked dependencies and verify Go, WebWallet, publisher and Telegram Game."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
import os
from pathlib import Path
import re
import shutil
import sys
import tempfile
import threading

from go_source_audit import ROOT, STATICCHECK, git, run, snapshot


def version_matches(label, actual, lock):
    """A present but different toolchain must not produce an acceptance receipt."""
    if not actual:
        return False
    if label == "node":
        actual = actual.removeprefix("v")
    entry = {"ffprobe": "ffmpeg", "gcc": "mingw", "gxx": "mingw"}.get(label, label)
    return actual == lock[entry]["version"]


def gate_passed(versions, lock, groups, checks):
    required = {"go", "node", "npm", "python", "ffmpeg", "ffprobe", "gcc", "gxx"}
    return (set(versions) == required and all(version_matches(k, v, lock) for k, v in versions.items())
            and set(checks) == set(groups) and all(
                set(checks[group]) == {label for label, _ in commands}
                and all(row.get("exitCode") == 0 and not row.get("error")
                        and not row.get("failedTests") for row in checks[group].values())
                for group, (_, commands) in groups.items()))


def initialize_fixture_git(source, base, env):
    # The tests exercise Git hygiene. Give the disposable source snapshot its
    # own index and HEAD without reading or modifying the workspace's Git state.
    hooks = base / "empty-git-hooks"
    hooks.mkdir()
    git_env = {**env, "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
               "GIT_AUTHOR_NAME": "QuantumCoin test", "GIT_AUTHOR_EMAIL": "test@example.invalid",
               "GIT_COMMITTER_NAME": "QuantumCoin test", "GIT_COMMITTER_EMAIL": "test@example.invalid"}
    paths = sorted(path.relative_to(source).as_posix() for path in source.rglob("*") if path.is_file())
    command = ["git", "-c", f"core.hooksPath={hooks}", "-c", "commit.gpgsign=false"]
    record, _ = run(command + ["init", "--quiet"], source, git_env, "fixture-git/init", timeout=60)
    if record["exitCode"] != 0:
        return record
    # The hygiene tests inspect names. Use one placeholder blob for the index
    # and HEAD while the actual tests read the exact source bytes in the tree.
    record, result = run(command + ["hash-object", "-w", "--stdin"], source, git_env,
                         "fixture-git/blob", timeout=60, input_data=b"Disposable path fixture\n")
    object_id = result.strip()
    if record["exitCode"] != 0 or not re.fullmatch(r"[0-9a-f]{40,64}", object_id):
        return {**record, "exitCode": 1}
    entries = "".join(f"100644 {object_id}\t{path}\0" for path in paths).encode("utf-8")
    record, _ = run(command + ["update-index", "-z", "--index-info"], source, git_env,
                    "fixture-git/index", timeout=60, input_data=entries)
    if record["exitCode"] != 0:
        return record
    record, _ = run(command + ["commit", "--quiet", "-m", "Disposable source path fixture"],
                    source, git_env, "fixture-git/snapshot", timeout=60)
    return record


def verify(output, jobs=1):
    env = {name: value for name, value in os.environ.items()
           if name.upper() in {"PATH", "PATHEXT", "SYSTEMROOT", "COMSPEC", "WINDIR", "TEMP", "TMP",
                               "USERPROFILE", "HOME", "LOCALAPPDATA", "APPDATA", "PROGRAMFILES",
                               "PROGRAMFILES(X86)", "SYSTEMDRIVE", "PROGRAMDATA", "CC", "CXX", "AR"}}
    env.update(GOENV="off", GOWORK="off", GOTOOLCHAIN="local", CGO_ENABLED="1",
               GOFLAGS="-mod=readonly -buildvcs=false -p=3", PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD="1")
    npm = shutil.which("npm.cmd" if os.name == "nt" else "npm")
    if not npm:
        raise SystemExit("npm is required")
    lock = json.loads((ROOT / "configs/windows-build-toolchain.json").read_text(encoding="utf-8"))
    receipt = {"kind": "unsignedCleanStackVerification", "ok": False,
               "head": git("rev-parse", "HEAD").decode().strip(),
               "workingTreeSnapshot": bool(git("status", "--porcelain")),
               "concurrentSuites": jobs}
    receipt_lock = threading.Lock()

    def save():
        output.seek(0)
        output.truncate()
        output.write(json.dumps(receipt, indent=2) + "\n")
        output.flush()

    save()
    with tempfile.TemporaryDirectory(prefix="qc-clean-stack-") as scratch:
        base = Path(scratch).resolve()
        source = base / "source"
        source.mkdir()
        receipt["source"] = snapshot(source)
        receipt["fixtureGit"] = initialize_fixture_git(source, base, env)
        if receipt["fixtureGit"]["exitCode"] != 0:
            receipt["ok"] = False
            save()
            return 1
        for level in ("USER", "GLOBAL"):
            npm_config = base / f"{level.lower()}.npmrc"
            npm_config.touch()
            env[f"NPM_CONFIG_{level}CONFIG"] = str(npm_config)
        venv = base / "venv"
        vpython = venv / ("Scripts/python.exe" if os.name == "nt" else "bin/python")
        privacy_venv = base / "privacy-tools"
        privacy_python = privacy_venv / ("Scripts/python.exe" if os.name == "nt" else "bin/python")
        versions = {}
        for label, command in (("go", ["go", "version"]), ("node", ["node", "--version"]),
                               ("npm", [npm, "--version"]), ("python", [sys.executable, "--version"])):
            record, value = run(command, base, env, [label, "--version"], timeout=30)
            version = re.search(r"(?:go|v)?\d+\.\d+\.\d+", value)
            versions[label] = version[0] if record["exitCode"] == 0 and version else None
        for label in ("ffmpeg", "ffprobe"):
            if shutil.which(label, path=env.get("PATH")):
                record, value = run([label, "-version"], base, env, [label, "-version"], timeout=30)
                version = re.search(rf"^{label} version (\d+\.\d+(?:\.\d+)?)", value)
                versions[label] = version[1] if record["exitCode"] == 0 and version else None
            else:
                versions[label] = None
        for label, key, default in (("gcc", "CC", "gcc"), ("gxx", "CXX", "g++")):
            record, value = run([env.get(key, default), "-dumpfullversion"], base, env,
                                [label, "-dumpfullversion"], timeout=30)
            versions[label] = value.strip() if record["exitCode"] == 0 else None
        receipt["versions"] = versions
        receipt["toolchainMatches"] = {key: version_matches(key, value, lock)
                                        for key, value in versions.items()}
        if not all(receipt["toolchainMatches"].values()):
            save()
            return 1
        receipt["checks"] = {}
        save()

        def execute(group, directory, commands):
            results = {}
            for label, command in commands:
                command_env = env
                if group == "tooling" and label == "release-trust":
                    command_env = {**env, "QC_RELEASE_E2E_CC": env.get("CC") or shutil.which("gcc", path=env.get("PATH", "")) or ""}
                elif group == "embedded-webui":
                    command_env = {**env, "QC_WEB_MINING_VECTOR_FILE": str(base / "browser-pow-vectors.json")}
                elif group == "go":
                    # MinGW can ship another python.exe earlier in PATH.
                    # SQLite recovery fixtures must use the checked interpreter.
                    command_env = {**env, "QC_RECOVERY_TEST_PYTHON": sys.executable}
                try:
                    record, log = run(command, directory, command_env, f"{group}/{label}",
                                      timeout=5400 if group == "go" else 1800)
                except OSError as exc:
                    # Do not lose all completed evidence if a command cannot start.
                    record, log = {"exitCode": None, "error": type(exc).__name__}, ""
                log_path = Path(output.name).with_name(Path(output.name).stem + f"-{group}-{label}.log")
                log_path.write_text(log, encoding="utf-8")
                record["log"] = str(log_path)
                if group == "tooling" and label in {"release-trust", "release-signer", "privacy-test"} and re.search(r"OK \(skipped=", log):
                    record["error"] = "required-integration-test-skipped"
                    record["exitCode"] = 1
                record["failedTests"] = sorted(set(re.findall(
                    r"^(?:--- FAIL: (Test\w+)|(?:FAIL|ERROR): (test_\w+)|.*(?:not ok \d+ - )([\w -]+))",
                    log, re.M)))
                record["failedTests"] = [next(value for value in match if value).strip()
                                         for match in record["failedTests"]]
                if group == "go" and label == "test":
                    record["passedPackages"] = len(re.findall(r"^ok\s+quantumcoin(?:/\S+)?\s", log, re.M))
                    record["failedPackages"] = sorted(set(re.findall(r"^FAIL\s+(quantumcoin(?:/\S+)?)\s", log, re.M)))
                if record["failedTests"]:
                    print(f"{group}/{label} failed tests: {', '.join(record['failedTests'])}", flush=True)
                with receipt_lock:
                    results[label] = record
                    receipt["checks"][group] = results
                    save()
                if record["exitCode"] != 0:
                    break
            return results

        install = [npm, "ci", "--no-audit", "--no-fund"]
        groups = {
            "tooling": (source, [
                ("network-workspaces", [sys.executable, "-m", "unittest", "discover", "-s", "scripts",
                                        "-p", "test_network_workspaces.py", "-v"]),
                ("security-results", [sys.executable, "-m", "unittest", "discover", "-s", "scripts",
                                      "-p", "test_dependency_security.py", "-v"]),
                ("tool-path", [sys.executable, "-m", "unittest", "discover", "-s", "scripts",
                               "-p", "test_tool_path_resolution.py", "-v"]),
                ("gate-contract", [sys.executable, "-m", "unittest", "discover", "-s", "scripts",
                                   "-p", "test_clean_stack_gate.py", "-v"]),
                ("ci-wrapper", [sys.executable, "-m", "unittest", "discover", "-s", "scripts",
                                "-p", "test_ci_wrapper.py", "-v"]),
                ("source-boundaries", [sys.executable, "-m", "unittest", "discover", "-s", "scripts",
                                       "-p", "test_runtime_config_and_cleanup.py", "-v"]),
                ("source-runtime", [sys.executable, "-m", "unittest", "discover", "-s", "scripts",
                                    "-p", "test_source_runtime_boundary.py", "-v"]),
                ("privacy-venv", [sys.executable, "-m", "venv", str(privacy_venv)]),
                ("privacy-restore", [str(privacy_python), "-m", "pip", "--isolated", "install", "--require-hashes",
                                     "-r", "scripts/privacy-tools.requirements.txt"]),
                ("privacy-test", [str(privacy_python), "-m", "unittest", "discover", "-s", "scripts",
                                  "-p", "test_git_privacy.py", "-v"]),
                ("release-trust", [sys.executable, "-m", "unittest", "discover", "-s", "tools",
                                   "-p", "test_qc_release.py", "-v"]),
                ("release-signer", [sys.executable, "-m", "unittest", "discover", "-s", "tools",
                                    "-p", "test_release_signer.py", "-v"]),
            ]),
            "go": (source, [("restore", ["go", "mod", "download"]), ("verify", ["go", "mod", "verify"]),
                            ("test", ["go", "test", "./...", "-count=1", "-timeout=45m"]),
                            ("vet", ["go", "vet", "./..."]),
                            ("race", ["go", "test", "-race", "./...", "-count=1", "-timeout=45m"]),
                            ("staticcheck", ["go", "run", STATICCHECK, "-checks=all", "./..."])]),
            "webwallet": (source / "webwallet", [("restore", install), ("test", [npm, "test"]),
                                                   ("build", [npm, "run", "build"])]),
            "embedded-webui": (source, [
                ("go-vectors", ["go", "test", "./blockchain", "-run", "^TestExternalMiningBrowserVectors$",
                                "-count=1", "-timeout=5m"]),
                ("test", ["node", "--test", "webui/*.test.mjs"])]),
            "autopublisher": (source / "autopublisher", [
                ("restore", install),
                ("build", [npm, "run", "build"]),
                ("test", ["node", "--test", "--test-concurrency=1", "tests/*.test.mjs"]),
            ]),
            "telegram_game": (source, [

                ("wallet-crypto", [sys.executable, "scripts/build_wallet_crypto.py"]),
                ("venv", [sys.executable, "-m", "venv", str(venv)]),
                ("restore", [str(vpython), "-m", "pip", "--isolated", "install", "--require-hashes",
                             "-r", "telegram_game/requirements.lock"]),
                ("dependency-consistency", [str(vpython), "-m", "pip", "check"]),
                ("testclient-compatibility", [str(vpython), "-m", "unittest", "discover", "-s", "scripts",
                                               "-p", "test_starlette_client_dependency.py", "-v"]),
                ("shared-config", [str(vpython), "-m", "unittest", "discover", "-s", "scripts", "-p", "test_shared_runtime_environment.py", "-v"]),
                ("environment-inventory", [str(vpython), "scripts/export_runtime_environment.py", "--check"]),
                ("test", [str(vpython), "-m", "unittest", "discover", "-s", "telegram_game", "-v"]),
                ("endpoint-inventory", [str(vpython), "scripts/export_game_endpoints.py", "--check"]),
                ("recovery-ui", ["node", "--test", "telegram_game/tests/test_wallet_recovery_ui_v1.js"]),
            ]),
        }
        # Suites include subprocesses, native compilation and tight loopback
        # deadlines. Concurrent suites can invalidate those timing checks on a
        # small CI runner. Default to serial suites; allow explicit parallelism.
        # Check clean dependency restores before spending the long Go race budget.
        groups = {name: groups[name] for name in ("tooling", "webwallet", "embedded-webui",
                                                  "autopublisher", "telegram_game", "go")}
        receipt["requiredStages"] = {group: [name for name, _ in commands]
                                     for group, (_, commands) in groups.items()}
        save()
        with ThreadPoolExecutor(max_workers=jobs) as pool:
            futures = {group: pool.submit(execute, group, directory, commands)
                       for group, (directory, commands) in groups.items()}
            receipt["checks"] = {group: future.result() for group, future in futures.items()}
        receipt["ok"] = gate_passed(versions, lock, groups, receipt["checks"])
    save()
    return 0 if receipt["ok"] else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--jobs", type=int, choices=range(1, 5), default=1,
                        help="concurrent language suites (default: 1)")
    args = parser.parse_args()
    target = args.output.resolve()
    if target.is_relative_to(ROOT):
        parser.error("choose a receipt path outside the checkout")
    with target.open("x", encoding="utf-8") as stream:
        return verify(stream, args.jobs)


if __name__ == "__main__":
    raise SystemExit(main())
