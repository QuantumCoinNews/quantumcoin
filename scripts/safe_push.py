"""Review the Git index and optionally commit exactly the staged changes."""

import argparse
from pathlib import Path
import re
import subprocess


ROOT = Path(__file__).resolve().parents[1]
SECRET = re.compile(
    rb"sk-(?:proj-)?[A-Za-z0-9_-]{32,}|ghp_[A-Za-z0-9]{36}|"
    rb"github_pat_[A-Za-z0-9_]{36,}|AIza[0-9A-Za-z_-]{35}|AKIA[0-9A-Z]{16}|"
    rb"xox[baprs]-[A-Za-z0-9-]{10,}|[0-9]{7,12}:[A-Za-z0-9_-]{35}|"
    rb"https://discord\.com/api/webhooks/[0-9]+/[A-Za-z0-9_-]+|"
    rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"
)


def git(*args):
    result = subprocess.run(["git", *args], cwd=ROOT, capture_output=True, check=False)
    if result.returncode:
        raise ValueError(f"git {args[0]} failed with exit code {result.returncode}")
    return result.stdout


def paths(raw):
    return [entry for entry in raw.split(b"\0") if entry]


def display(raw):
    return repr(raw.decode("utf-8", errors="backslashreplace"))


def scan_index(head):
    ignored = paths(git("ls-files", "--cached", "--ignored", "--exclude-standard", "-z"))
    if ignored:
        names = ", ".join(display(name) for name in ignored[:10])
        raise ValueError(f"{len(ignored)} tracked path(s) match Git ignore rules. Review explicitly: {names}")

    staged = set(paths(git("diff", "--cached", "--name-only", "--diff-filter=ACMRTUXB", "--no-renames", "-z", head)))
    remaining = staged.copy()
    for entry in paths(git("ls-files", "--stage", "-z")):
        fields, name = entry.split(b"\t", 1)
        if name not in staged:
            continue
        mode, oid, stage = fields.split()
        if stage != b"0" or mode == b"160000":
            raise ValueError(f"cannot verify staged entry {display(name)}")
        oid = oid.decode("ascii")
        if int(git("cat-file", "-s", oid)) > 10 * 1024 * 1024:
            raise ValueError(f"staged blob exceeds scan limit: {display(name)}")
        blob = git("cat-file", "blob", oid)
        if SECRET.search(blob) or re.fullmatch(rb"[\s\xef\xbb\xbf]*[0-9a-fA-F]{64}\s*", blob):
            raise ValueError(f"possible credential in staged blob: {display(name)}")
        remaining.discard(name)
    if remaining:
        raise ValueError("Git index contains staged changes that could not be inspected")


def diff_index(head):
    return git("diff", "--cached", "--raw", "--no-abbrev", "--no-renames", "-z", head)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--message", default="chore: safe update")
    parser.add_argument("--remote", default="origin")
    parser.add_argument("--branch")
    parser.add_argument("--commit", action="store_true")
    parser.add_argument("--push", action="store_true")
    args = parser.parse_args()
    if args.push and not args.commit:
        parser.error("--push requires --commit")
    try:
        head = git("rev-parse", "--verify", "HEAD").decode().strip()
        planned = diff_index(head)
        scan_index(head)
        if git("rev-parse", "--verify", "HEAD").decode().strip() != head or diff_index(head) != planned:
            raise ValueError("HEAD or index changed during inspection")
        if not args.commit:
            print(f"SAFE_PUSH_CHECK_PASS staged_changes={bool(planned)}")
            return
        if not planned:
            raise ValueError("stage the reviewed files before requesting a commit")
        if args.push:
            branch = git("symbolic-ref", "--short", "HEAD").decode().strip()
            if args.branch and args.branch != branch:
                raise ValueError("requested branch differs from the checked-out branch")
            if args.remote.startswith("-") or args.remote not in git("remote").decode().splitlines():
                raise ValueError("requested Git remote does not exist")

        git("commit", "-m", args.message)
        created = git("rev-list", "--parents", "-n", "1", "HEAD").decode().split()
        if len(created) != 2 or created[1] != head or git("diff", "--raw", "--no-abbrev", "--no-renames", "-z", head, created[0]) != planned:
            raise ValueError("commit differs from reviewed index. Inspect the local commit before publishing")
        if args.push:
            # Use the reviewed commit ID rather than a potentially changed HEAD.
            git("push", "--", args.remote, f"{created[0]}:refs/heads/{branch}")
            print(f"SAFE_PUSH_PUBLISHED {created[0]}")
        else:
            print(f"SAFE_PUSH_COMMITTED {created[0]}")
    except (OSError, ValueError) as error:
        parser.exit(1, f"SAFE_PUSH_REJECTED {error}\n")


if __name__ == "__main__":
    main()
