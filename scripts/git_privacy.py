"""Redacted Git privacy inventory and guarded index-only cleanup.

History rewriting is performed in a separate mirror by the operator. Neither
planning nor index cleanup deletes worktree files, commits, pushes or rotates keys.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

from source_hygiene import source_exclusion, cleanup_class

ROOT = Path(__file__).resolve().parents[1]
SANITIZERS = (
    ("openai-key", rb"sk-(?:proj-)?[A-Za-z0-9_-]{32,}", b"[REDACTED_API_KEY]"),
    ("github-key", rb"(?:ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{36,})", b"[REDACTED_API_KEY]"),
    ("google-key", rb"AIza[0-9A-Za-z_-]{35}", b"[REDACTED_API_KEY]"),
    ("aws-key-id", rb"AKIA[0-9A-Z]{16}", b"[REDACTED_API_KEY]"),
    ("slack-token", rb"xox[baprs]-[A-Za-z0-9-]{10,}", b"[REDACTED_API_KEY]"),
    ("telegram-token", rb"[0-9]{7,12}:[A-Za-z0-9_-]{35}", b"[REDACTED_BOT_TOKEN]"),
    ("private-key-pem", rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----", b"[REDACTED_PRIVATE_KEY]"),
    ("credential-field", rb'''(?i)(["'](?:refresh_token|access_token|client_secret|api_key|api_secret|consumer_key|consumer_secret|private_key|privatekey|priv_hex|privhex|mnemonic|seed_phrase|password)["']\s*:\s*["'])(?!\[REDACTED_SECRET\]["'])[^"'\r\n]{16,}(["'])''', rb"\1[REDACTED_SECRET]\2"),
    ("env-secret", rb'''(?im)^([ \t]*(?:\$env:)?(?:OPENAI_API_KEY|TW_CONSUMER_SECRET|TW_ACCESS_SECRET|TELEGRAM_BOT_TOKEN|X_APP_SECRET|X_OAUTH2_CLIENT_SECRET|X_OAUTH2_TOKEN_JSON|QC_API_TOKEN)[ \t]*=[ \t]*)(["']?)[A-Za-z0-9_./+:=-]{16,}\2([ \t]*(?:#.*)?)$''', rb"\1\2[REDACTED_SECRET]\2\3"),
    ("windows-personal-path", rb'''([A-Za-z]:[\\/]+Users[\\/]+)(?!USER(?:[\\/]|$))[^\\/\s"'<>`]+''', rb"\1USER"),
    ("unix-personal-path", rb'''(/(?:home|Users)/)(?!USER(?:/|$))[^/\s"'<>`]+''', rb"\1USER"),
)
PATTERNS = [(name, re.compile(pattern), replacement) for name, pattern, replacement in SANITIZERS]


def git_environment(explicit=None):
    source = os.environ if explicit is None else explicit
    redirects = {"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY",
                 "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_INDEX_FILE", "GIT_NAMESPACE",
                 "GIT_SHALLOW_FILE", "GIT_REPLACE_REF_BASE"}
    environment = {key: value for key, value in source.items() if key.upper() not in redirects}
    if explicit is not None and "GIT_INDEX_FILE" in explicit:
        environment["GIT_INDEX_FILE"] = explicit["GIT_INDEX_FILE"]
    environment["GIT_NO_REPLACE_OBJECTS"] = "1"
    return environment


def git(repo, *args, env=None, data=None):
    result = subprocess.run(["git", *args], cwd=repo, env=git_environment(env), input=data,
                            capture_output=True, timeout=300)
    if result.returncode:
        # stderr can contain credentials from user-configured helpers/URLs.
        raise ValueError("Git operation failed: " + args[0])
    return result.stdout


def index_path(repo):
    path = Path(os.fsdecode(git(repo, "rev-parse", "--git-path", "index").strip()))
    return path.resolve() if path.is_absolute() else (repo / path).resolve()


def entries(repo, env=None):
    result = {}
    for record in git(repo, "ls-files", "--stage", "-z", env=env).split(b"\0"):
        if not record:
            continue
        fields, name = record.split(b"\t", 1)
        mode, oid, stage = fields.split()
        if stage != b"0":
            raise ValueError("Resolve index conflicts before privacy cleanup")
        result[name.decode("utf-8")] = fields.decode("ascii")
    return result


def plan_index(repo):
    repo = repo.resolve()
    selected = entries(repo)
    return {"schema": "QC_INDEX_PRIVACY_V1", "repo": str(repo),
            "head": git(repo, "rev-parse", "HEAD").decode().strip(),
            "indexSha256": hashlib.sha256(index_path(repo).read_bytes()).hexdigest(),
            "entries": selected,
            "remove": [{"path": name, "class": cleanup_class(name)}
                       for name in sorted(selected) if source_exclusion(name)],
            "worktreeFilesDeleted": False, "commitPerformed": False}


def apply_index(repo, plan, evidence):
    repo, evidence = repo.resolve(), evidence.resolve()
    if evidence.is_relative_to(repo) or evidence.exists():
        raise ValueError("Choose a new evidence directory outside the repository")
    current = plan_index(repo)
    if plan != current:
        raise ValueError("Stale or modified index plan; inspect a fresh plan")
    evidence.mkdir(parents=True, mode=0o700)
    path = index_path(repo)
    original = path.read_bytes()
    (evidence / "before.index").write_bytes(original)
    candidate = evidence / "candidate.index"
    candidate.write_bytes(original)
    environment = {**os.environ, "GIT_INDEX_FILE": str(candidate)}
    removed = [row["path"] for row in plan["remove"]]
    if removed:
        git(repo, "update-index", "-z", "--force-remove", "--stdin", env=environment,
            data=("\0".join(removed) + "\0").encode("utf-8"))
    expected = {key: value for key, value in plan["entries"].items() if key not in set(removed)}
    if entries(repo, environment) != expected:
        raise ValueError("Candidate index changed protected source entries")
    lock = Path(str(path) + ".lock")
    fd = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    try:
        with os.fdopen(fd, "wb") as stream:
            if plan_index(repo) != plan:
                raise ValueError("Index or HEAD changed; original index preserved")
            stream.write(candidate.read_bytes())
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(lock, path)
    finally:
        # Only our exclusively created lock may be removed.
        lock.unlink(missing_ok=True)
    if entries(repo) != expected or git(repo, "rev-parse", "HEAD").decode().strip() != plan["head"]:
        raise ValueError("Concurrent repository change after publication; inspect index, do not auto-rollback")
    receipt = {"schema": "QC_INDEX_PRIVACY_RECEIPT_V1", "ok": True, "removed": removed,
               "beforeSha256": plan["indexSha256"],
               "afterSha256": hashlib.sha256(path.read_bytes()).hexdigest(),
               "protectedEntriesUnchanged": True, "worktreeFilesDeleted": False,
               "commitPerformed": False, "historyRewritten": False, "pushPerformed": False}
    (evidence / "receipt.json").write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
    return receipt


def sanitize(data):
    for _, pattern, replacement in PATTERNS:
        data = pattern.sub(replacement, data)
    return data


def logical_text(data):
    for marker, codec in ((b"\xff\xfe\0\0", "utf-32-le"), (b"\0\0\xfe\xff", "utf-32-be"),
                          (b"\xff\xfe", "utf-16-le"), (b"\xfe\xff", "utf-16-be")):
        if data.startswith(marker):
            return data[len(marker):].decode(codec).encode("utf-8"), (marker, codec)
    return data, None


def sanitize_blob(data):
    if len(data) > 2 * 1024 * 1024:
        return data
    try:
        logical, encoding = logical_text(data)
    except UnicodeError:
        return data
    suffix = b""
    if logical.startswith(b"{\\rtf"):
        trimmed = logical.rstrip(b"\0")
        suffix = logical[len(trimmed):]
        logical = trimmed
    if b"\0" in logical[:8192]:
        return data
    cleaned = sanitize(logical) + suffix
    if encoding:
        marker, codec = encoding
        return marker + cleaned.decode("utf-8").encode(codec)
    return cleaned


def detect(data):
    result = [name for name, pattern, _ in PATTERNS if pattern.search(data)]
    if re.fullmatch(rb"\s*[a-fA-F0-9]{64}\s*", data):
        result.append("possible-raw-wallet-key-or-hash")
    return result


def audit_history(repo, max_bytes=2 * 1024 * 1024, *, index_only=False):
    """Every reachable commit/tree, all path aliases; output no blob/message values."""
    non_commit_refs = []
    commits = []
    if not index_only:
        roots = set()
        for row in git(repo, "for-each-ref", "--format=%(refname) %(objectname) %(objecttype)").splitlines():
            name, oid, kind = row.decode().split()
            peeled = subprocess.run(["git", "rev-parse", "--verify", "--quiet", name + "^{commit}"],
                                    cwd=repo, env=git_environment(), capture_output=True, timeout=30)
            if peeled.returncode:
                non_commit_refs.append({"ref": name, "oid": oid, "type": kind})
            else:
                roots.add(peeled.stdout.strip())
        if roots:
            commits = git(repo, "rev-list", "--stdin", data=b"\n".join(sorted(roots)) + b"\n").decode().splitlines()
    aliases, excluded, message_hits = {}, set(), []
    if index_only:
        for path, fields in entries(repo).items():
            _, oid, _ = fields.split()
            aliases.setdefault(oid, set()).add(path)
            if source_exclusion(path):
                excluded.add(path)
    for commit in commits:
        for row in git(repo, "ls-tree", "-r", "-z", "--full-tree", commit).split(b"\0"):
            if not row:
                continue
            fields, rawpath = row.split(b"\t", 1)
            _, kind, oid = fields.split()
            if kind != b"blob":
                continue
            path = rawpath.decode("utf-8")
            aliases.setdefault(oid.decode(), set()).add(path)
            if source_exclusion(path):
                excluded.add(path)
        kinds = detect(git(repo, "show", "-s", "--format=%B", commit))
        if kinds:
            message_hits.append({"commit": commit, "kinds": kinds})
    findings, skipped, checked = [], [], 0
    selected = {oid: sorted(name for name in names if not source_exclusion(name))
                for oid, names in aliases.items()}
    selected = {oid: names for oid, names in selected.items() if names}
    sizes = {}
    if selected:
        rows = git(repo, "cat-file", "--batch-check",
                   data="".join(oid + "\n" for oid in selected).encode("ascii"))
        for row in rows.splitlines():
            oid, kind, size = row.split()
            if kind != b"blob":
                raise ValueError("Unexpected object in history audit")
            sizes[oid.decode()] = int(size)
    with subprocess.Popen(["git", "cat-file", "--batch"], cwd=repo, stdin=subprocess.PIPE,
                          stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, env=git_environment()) as reader:
        try:
            for oid, retained in sorted(selected.items()):
                size = sizes[oid]
                if size > max_bytes:
                    skipped.append({"oid": oid, "paths": retained, "reason": "size-limit", "bytes": size})
                    continue
                reader.stdin.write((oid + "\n").encode("ascii")); reader.stdin.flush()
                header = reader.stdout.readline().split()
                if len(header) != 3 or header[1] != b"blob" or int(header[2]) != size:
                    raise ValueError("Invalid history blob framing")
                blob = reader.stdout.read(size)
                if len(blob) != size or reader.stdout.read(1) != b"\n":
                    raise ValueError("Truncated history blob")
                try:
                    blob, _ = logical_text(blob)
                except UnicodeError:
                    skipped.append({"oid": oid, "paths": retained, "reason": "invalid-encoding"})
                    continue
                if blob.startswith(b"{\\rtf"):
                    blob = blob.rstrip(b"\0")
                if b"\0" in blob[:8192]:
                    skipped.append({"oid": oid, "paths": retained, "reason": "binary"})
                    continue
                checked += 1
                kinds = detect(blob)
                if kinds:
                    findings.append({"oid": oid, "paths": retained, "kinds": kinds})
        finally:
            reader.stdin.close()
            reader.stdout.close()
        if reader.wait(timeout=30):
            raise ValueError("History blob reader failed")
    return {"schema": "QC_GIT_PRIVACY_AUDIT_V1", "scope": "current-index" if index_only else "all-local-reachable-refs",
            "valuesIncluded": False, "commits": len(commits), "blobs": len(aliases),
            "textBlobsChecked": checked, "excludedPaths": sorted(excluded),
            "retainedTextFindings": findings, "messageFindings": message_hits,
            "uninspectedRetainedBlobs": skipped,
            "nonCommitRefs": non_commit_refs,
            "pathAndPatternClean": not (excluded or findings or message_hits or non_commit_refs),
            "limitations": ["Heuristic patterns cannot prove absence of every secret or personal datum.",
                            "Binary/oversize retained assets require separate review.",
                            "Remote refs, caches, forks, reflogs, unreachable objects and provider revocation are outside this scan."]}


def replacement_rules():
    return b"".join(b"regex:" + pattern + b"==>" + replacement + b"\n"
                    for _, pattern, replacement in SANITIZERS)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["plan-index", "apply-index", "audit-history", "audit-index"])
    parser.add_argument("--repo", type=Path, default=ROOT)
    parser.add_argument("--plan", type=Path)
    parser.add_argument("--evidence", type=Path)
    args = parser.parse_args()
    try:
        if args.action == "plan-index":
            result = plan_index(args.repo)
        elif args.action in {"audit-history", "audit-index"}:
            result = audit_history(args.repo, index_only=args.action == "audit-index")
        else:
            if args.plan is None or args.evidence is None:
                parser.error("apply-index requires --plan and --evidence")
            result = apply_index(args.repo, json.loads(args.plan.read_text(encoding="utf-8")), args.evidence)
        print(json.dumps(result, indent=2, ensure_ascii=True))
    except (ValueError, OSError) as exc:
        parser.exit(1, str(exc) + "\n")


if __name__ == "__main__":
    main()
