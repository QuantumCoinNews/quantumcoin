"""Build and verify a QuantumCoin Windows release without writing to release/."""

import argparse
import contextlib
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import tempfile


REPO = Path(__file__).resolve().parents[1]
MANIFEST = "release-manifest.json"
SIGNATURE = MANIFEST + ".sig"
SBOM = "sbom.cdx.json"
PROVENANCE = "provenance.json"
SIGNER = "quantumcoin-release"
NAMESPACE = "quantumcoin-release-v1"
TOOLCHAIN_LOCK = "configs/windows-build-toolchain.json"
STATEMENT = "https://in-toto.io/Statement/v1"
PREDICATE = "https://github.com/QuantumCoinNews/quantumcoin/release-provenance/v1"
REPRODUCTION = "two_clean_checkouts_distinct_build_caches_same_host"
TARGETS = {
    "quantumcoin.exe": ("quantumcoin", ".", "0"),
    "wallet-gui.exe": ("quantumcoin/cmd/wallet", "./cmd/wallet", "1"),
    "quantumcoin-launcher.exe": ("quantumcoin/cmd/quantumcoin-gui", "./cmd/quantumcoin-gui", "0"),
}
BUILD_FLAGS = ["-mod=readonly", "-buildvcs=true", "-trimpath", "-ldflags=-buildid="]
CGO_LDFLAGS = "-O2 -g -Wl,--no-insert-timestamp"
MINER_SCRIPTS = {
    "run_miner.cmd": "release/run_miner.cmd",
    "_qc_api_mempool_miner.ps1": "release/_qc_api_mempool_miner.ps1",
}
BUILD_OUTPUTS = set(TARGETS) | set(MINER_SCRIPTS)
PAYLOAD = BUILD_OUTPUTS | {SBOM, PROVENANCE}


def run(*args, cwd=None, env=None, data=None):
    result = subprocess.run(
        args, cwd=cwd, env=env, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode:
        detail = result.stderr.decode("utf-8", errors="replace").strip()
        raise ValueError(f"{Path(args[0]).name} failed: {detail or result.returncode}")
    return result.stdout


def regular(path):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or getattr(info, "st_file_attributes", 0) & 0x400:
        raise ValueError(f"expected a regular file: {path.name}")
    return info


def sha256(path):
    regular(path)
    with path.open("rb") as file:
        return hashlib.file_digest(file, "sha256").hexdigest()


def json_bytes(obj):
    return (json.dumps(obj, sort_keys=True, indent=2, ensure_ascii=True) + "\n").encode("utf-8")


def read_json(path):
    if regular(path).st_size > 2 * 1024 * 1024:
        raise ValueError(f"metadata too large: {path.name}")
    raw = path.read_bytes()
    obj = json.loads(raw)
    if not isinstance(obj, dict) or json_bytes(obj) != raw:
        raise ValueError(f"noncanonical metadata: {path.name}")
    return obj, raw


def revision(value):
    if not re.fullmatch(r"[0-9a-f]{40}", value):
        raise ValueError("supply the full authorized Git commit")
    return value


def toolchain_policy(raw):
    if len(raw) > 2 * 1024 * 1024:
        raise ValueError("toolchain policy too large")
    policy = json.loads(raw)
    go_version = policy["go"]["version"]
    cc_version = policy["mingw"]["version"]
    if (not isinstance(go_version, str) or not re.fullmatch(r"go\d+\.\d+\.\d+", go_version)
            or not isinstance(cc_version, str) or not re.fullmatch(r"\d+\.\d+\.\d+", cc_version)):
        raise ValueError("invalid trusted toolchain versions")
    return {"goVersion": go_version, "cCompilerVersion": cc_version,
            "toolchainLockSha256": hashlib.sha256(raw).hexdigest()}


def inspect_binary(go, binary, name, commit, go_version):
    lines = run(go, "version", "-m", str(binary)).decode("utf-8").splitlines()
    if not lines or lines[0].rsplit(None, 1)[-1] != go_version:
        raise ValueError(f"unexpected Go toolchain in {name}")
    settings = {}
    modules = {}
    package = None
    for line in lines[1:]:
        fields = line.strip().split("\t")
        if fields[0] == "path":
            package = fields[1]
        elif fields[0] == "build" and "=" in fields[1]:
            key, value = fields[1].split("=", 1)
            settings[key] = value
        elif fields[0] == "dep":
            modules[fields[1]] = fields[2]
        elif fields[0] == "=>":
            raise ValueError(f"module replacement in {name}")
    expected_path, _, cgo = TARGETS[name]
    required = {
        "vcs": "git", "vcs.revision": commit, "vcs.modified": "false",
        "GOOS": "windows", "GOARCH": "amd64", "GOAMD64": "v1", "CGO_ENABLED": cgo, "-trimpath": "true",
    }
    if package != expected_path or any(settings.get(k) != v for k, v in required.items()):
        raise ValueError(f"wrong source commit, package or build settings in {name}")
    return modules


def sbom_for(version, go_version, dependencies):
    stdlib = ("go-stdlib", go_version)
    libraries = {stdlib}
    for modules in dependencies.values():
        libraries.update(modules.items())
    components = [
        {"type": "library", "name": name, "version": ver, "bom-ref": f"{name}@{ver}"}
        for name, ver in sorted(libraries)
    ]
    components.extend(
        {"type": "application", "name": name, "version": version, "bom-ref": name}
        for name in sorted(TARGETS)
    )
    return {
        "$schema": "http://cyclonedx.org/schema/bom-1.6.schema.json",
        "bomFormat": "CycloneDX", "specVersion": "1.6", "version": 1,
        "metadata": {"component": {"type": "application", "name": "QuantumCoin", "version": version, "bom-ref": "release"}},
        "components": components,
        "dependencies": [{"ref": "release", "dependsOn": sorted(TARGETS)}] + [
            {"ref": name, "dependsOn": sorted([f"{module}@{ver}" for module, ver in modules.items()] + ["@".join(stdlib)])}
            for name, modules in sorted(dependencies.items())
        ],
    }


def artifacts(directory, names):
    return {
        name: {"bytes": regular(directory / name).st_size, "sha256": sha256(directory / name)}
        for name in sorted(names)
    }


def git_environment():
    env = {key: value for key, value in os.environ.items() if not key.upper().startswith("GIT_")}
    env.update(GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull, GIT_ATTR_NOSYSTEM="1")
    return env


def build_environment(git_env, cache, epoch):
    env = {
        key: value for key, value in git_env.items()
        if not key.upper().startswith(("GO", "CGO_"))
        and key.upper() not in {"CC", "CXX", "CFLAGS", "CXXFLAGS", "CPPFLAGS", "LDFLAGS", "PKG_CONFIG_PATH", "PKG_CONFIG_LIBDIR", "CPATH", "LIBRARY_PATH", "COMPILER_PATH", "GCC_EXEC_PREFIX"}
    }
    env.update(
        GOOS="windows", GOARCH="amd64", GOAMD64="v1", GOTOOLCHAIN="local", GOENV="off",
        GOWORK="off", GOCACHE=str(cache), SOURCE_DATE_EPOCH=epoch, TZ="UTC",
    )
    return env


def build_once(repo, checkout, output, args, go, cc, commit, git_env, epoch):
    run("git", "clone", "--quiet", "--no-local", "--no-checkout", "--", str(repo), str(checkout), env=git_env)
    run("git", "-c", "core.autocrlf=false", "-c", f"core.hooksPath={checkout / '.git' / 'disabled-hooks'}",
        "checkout", "--quiet", "--detach", commit, cwd=checkout, env=git_env)
    entries = run("git", "ls-files", "--stage", "-z", cwd=checkout, env=git_env).split(b"\0")
    if any(entry.startswith((b"120000 ", b"160000 ")) for entry in entries):
        raise ValueError("release commit contains symlinks or submodules")
    env = build_environment(git_env, checkout.parent / (checkout.name + "-cache"), epoch)
    run(go, "mod", "verify", cwd=checkout, env=env)
    output.mkdir()
    dependencies = {}
    for name, (_, package, cgo) in TARGETS.items():
        print(f"Building {name} in {checkout.name}", flush=True)
        build_env = {**env, "CGO_ENABLED": cgo}
        if cgo == "1":
            build_env.update(CC=f'"{Path(cc).as_posix()}"', CGO_LDFLAGS=CGO_LDFLAGS)
        run(go, "build", *BUILD_FLAGS, "-o", str(output / name), package, cwd=checkout, env=build_env)
        dependencies[name] = inspect_binary(go, output / name, name, commit, args.go_version)
    for name, relative in MINER_SCRIPTS.items():
        source = checkout / relative
        regular(source)
        shutil.copyfile(source, output / name)
    if run("git", "status", "--porcelain", cwd=checkout, env=git_env).strip():
        raise ValueError("build modified its source checkout")
    return dependencies


def build(args):
    commit = revision(args.commit)
    if not re.fullmatch(r"[0-9A-Za-z][0-9A-Za-z._-]{0,63}", args.version):
        raise ValueError("invalid release version")
    repo = args.repo.resolve(strict=True)
    output = args.output.resolve()
    if output.is_relative_to(repo) or output.is_relative_to(REPO) or output.exists():
        raise ValueError("output must be a new directory outside the worktree")
    output.parent.mkdir(parents=True, exist_ok=True)
    git_env = git_environment()
    if run("git", "cat-file", "-t", commit, cwd=repo, env=git_env).strip() != b"commit":
        raise ValueError("source object is not a Git commit")
    tree = run("git", "rev-parse", commit + "^{tree}", cwd=repo, env=git_env).decode().strip()
    policy = toolchain_policy(run("git", "show", commit + ":" + TOOLCHAIN_LOCK, cwd=repo, env=git_env))
    if args.go_version != policy["goVersion"] or args.cc_version != policy["cCompilerVersion"]:
        raise ValueError("build toolchain differs from the authorized source policy")
    epoch = run("git", "show", "-s", "--format=%ct", commit, cwd=repo, env=git_env).decode().strip()
    go = shutil.which("go")
    if not go:
        raise ValueError("Go toolchain not found")
    cc = str(args.cc.resolve(strict=True))
    env = build_environment(git_env, output.parent / "unused-go-cache", epoch)
    installed_go = run(go, "version", env=env).decode().split()
    if installed_go[2] != args.go_version:
        raise ValueError("installed Go version differs from --go-version")
    cc_version = run(cc, "-dumpfullversion", env=env).decode().strip()
    if cc_version != args.cc_version:
        raise ValueError("installed compiler version differs from --cc-version")
    if run(cc, "-dumpmachine", env=env).decode().strip() != "x86_64-w64-mingw32":
        raise ValueError("compiler must target Windows amd64")
    work = getattr(args, "work_directory", None)
    if work is not None:
        work = work.resolve()
        if (work.exists() or work.is_relative_to(repo) or work.is_relative_to(REPO)
                or work == output or work.is_relative_to(output) or output.is_relative_to(work)):
            raise ValueError("build work directory must be new, external and separate from the package")
        work.mkdir(parents=True)
    workspace = contextlib.nullcontext(str(work)) if work is not None else tempfile.TemporaryDirectory(prefix=".qc-build-", dir=output.parent)
    with workspace as temporary:
        root = Path(temporary)
        first, second = root / "package", root / "comparison"
        dep_a = build_once(repo, root / "source-a", first, args, go, cc, commit, git_env, epoch)
        dep_b = build_once(repo, root / "source-b", second, args, go, cc, commit, git_env, epoch)
        if artifacts(first, BUILD_OUTPUTS) != artifacts(second, BUILD_OUTPUTS) or dep_a != dep_b:
            raise ValueError("independent checkouts produced different release bytes")
        (first / SBOM).write_bytes(json_bytes(sbom_for(args.version, args.go_version, dep_a)))
        provenance = {
            "_type": STATEMENT,
            "predicateType": PREDICATE,
            "subject": [
                {"name": name, "digest": {"sha256": entry["sha256"]}}
                for name, entry in artifacts(first, BUILD_OUTPUTS | {SBOM}).items()
            ],
            "predicate": {
                "sourceCommit": commit, "sourceTree": tree, "target": "windows/amd64",
                "goVersion": args.go_version, "goCommandSha256": sha256(Path(go)),
                "cCompilerVersion": cc_version, "cCompilerSha256": sha256(Path(cc)),
                "toolchainLockSha256": policy["toolchainLockSha256"],
                "buildFlags": BUILD_FLAGS, "walletCgoLdflags": CGO_LDFLAGS,
                "reproduction": REPRODUCTION,
            },
        }
        (first / PROVENANCE).write_bytes(json_bytes(provenance))
        manifest = {
            "schema": "QC_RELEASE_V1", "version": args.version, "target": "windows/amd64",
            "sourceCommit": commit, "sourceTree": tree, "goVersion": args.go_version,
            "files": artifacts(first, PAYLOAD),
        }
        (first / MANIFEST).write_bytes(json_bytes(manifest))
        if output.exists():
            raise ValueError("output was created during the build")
        first.rename(output)
    print(f"UNSIGNED_CANDIDATE {commit} {output}")


def verify(args):
    commit = revision(args.commit)
    directory = args.directory.resolve(strict=True)
    policy_path = (getattr(args, "toolchain_lock", None) or REPO / TOOLCHAIN_LOCK).resolve(strict=True)
    if policy_path.is_relative_to(directory):
        raise ValueError("trusted toolchain policy must be outside the package")
    if regular(policy_path).st_size > 2 * 1024 * 1024:
        raise ValueError("toolchain policy too large")
    policy = toolchain_policy(policy_path.read_bytes())
    trusted = args.trusted_signers.resolve(strict=True)
    if trusted.is_relative_to(directory) or trusted.is_relative_to(REPO):
        raise ValueError("trusted signers must be supplied outside the package and source tree")
    regular(trusted)
    actual = {path.name for path in directory.iterdir()}
    if actual != PAYLOAD | {MANIFEST, SIGNATURE}:
        raise ValueError("unexpected or missing release entries (including signature)")
    manifest, raw = read_json(directory / MANIFEST)
    sig = directory / SIGNATURE
    if regular(sig).st_size > 64 * 1024:
        raise ValueError("signature too large")
    run("ssh-keygen", "-Y", "verify", "-f", str(trusted), "-I", SIGNER, "-n", NAMESPACE, "-s", str(sig), data=raw)
    if (
        manifest["schema"] != "QC_RELEASE_V1" or manifest["sourceCommit"] != commit
        or manifest["target"] != "windows/amd64" or set(manifest["files"]) != PAYLOAD
    ):
        raise ValueError("signed release does not match the authorized commit or file set")
    if manifest["goVersion"] != policy["goVersion"]:
        raise ValueError("signed release toolchain differs from independent trust policy")
    tree = manifest.get("sourceTree")
    if not isinstance(tree, str) or not re.fullmatch(r"[0-9a-f]{40}", tree):
        raise ValueError("invalid source tree identity")
    if artifacts(directory, PAYLOAD) != manifest["files"]:
        raise ValueError("release file size or SHA-256 mismatch")
    provenance, _ = read_json(directory / PROVENANCE)
    expected_subjects = [
        {"name": name, "digest": {"sha256": manifest["files"][name]["sha256"]}}
        for name in sorted(BUILD_OUTPUTS | {SBOM})
    ]
    predicate = provenance["predicate"]
    if (predicate.get("cCompilerVersion") != policy["cCompilerVersion"]
            or predicate.get("toolchainLockSha256") != policy["toolchainLockSha256"]
            or any(not isinstance(predicate.get(key), str) or not re.fullmatch(r"[0-9a-f]{64}", predicate[key])
                   for key in ("goCommandSha256", "cCompilerSha256", "toolchainLockSha256"))):
        raise ValueError("provenance toolchain identity is missing, malformed or unauthorized")
    if (
        provenance["_type"] != STATEMENT or provenance["predicateType"] != PREDICATE
        or provenance["subject"] != expected_subjects or predicate["sourceCommit"] != commit
        or predicate["sourceTree"] != manifest["sourceTree"] or predicate["goVersion"] != manifest["goVersion"]
        or predicate["target"] != manifest["target"] or predicate["buildFlags"] != BUILD_FLAGS
        or predicate["walletCgoLdflags"] != CGO_LDFLAGS
        or predicate["reproduction"] != REPRODUCTION
    ):
        raise ValueError("release provenance differs from the manifest")
    dependencies = {
        name: inspect_binary("go", directory / name, name, commit, manifest["goVersion"])
        for name in TARGETS
    }
    sbom, _ = read_json(directory / SBOM)
    if sbom != sbom_for(manifest["version"], manifest["goVersion"], dependencies):
        raise ValueError("SBOM differs from binary build information")
    print(f"VERIFIED_RELEASE {commit} {hashlib.sha256(raw).hexdigest()}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    builder = commands.add_parser("build", help="build an unsigned candidate from a committed source tree")
    builder.add_argument("--repo", type=Path, default=REPO)
    builder.add_argument("--commit", required=True)
    builder.add_argument("--version", required=True)
    builder.add_argument("--go-version", required=True)
    builder.add_argument("--cc", type=Path, required=True)
    builder.add_argument("--cc-version", required=True)
    builder.add_argument("--output", type=Path, required=True)
    builder.add_argument("--work-directory", type=Path, help="optional new external workspace retained for build comparison/diagnosis; never a release payload")
    verifier = commands.add_parser("verify", help="verify an approved signed release against external trust")
    verifier.add_argument("--directory", type=Path, required=True)
    verifier.add_argument("--commit", required=True)
    verifier.add_argument("--trusted-signers", type=Path, required=True)
    verifier.add_argument("--toolchain-lock", type=Path, help="independently trusted source toolchain policy outside the package; defaults to this checkout's lock")
    args = parser.parse_args()
    try:
        (build if args.command == "build" else verify)(args)
    except (OSError, ValueError, KeyError, TypeError, IndexError) as error:
        parser.exit(1, f"RELEASE_REJECTED {error}\n")


if __name__ == "__main__":
    main()
