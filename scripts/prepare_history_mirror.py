"""Prepare and audit rewritten committed history in a new external mirror only."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

from git_privacy import ROOT, audit_history, detect, git, git_environment


def uninspected_sensitive_paths(repo, rows):
    """Drop uninspectable binary/large blobs when raw-byte secret patterns match."""
    paths = set()
    process = subprocess.Popen(["git", "cat-file", "--batch"], cwd=repo,
                               env=git_environment(), stdin=subprocess.PIPE,
                               stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    try:
        for row in rows:
            process.stdin.write((row["oid"] + "\n").encode("ascii"))
            process.stdin.flush()
            header = process.stdout.readline().split()
            if len(header) != 3 or header[1] != b"blob":
                raise ValueError("Unexpected object in uninspected blob scan")
            size = int(header[2])
            data = process.stdout.read(size)
            if len(data) != size or process.stdout.read(1) != b"\n":
                raise ValueError("Truncated uninspected blob scan")
            if detect(data):
                paths.update(row["paths"])
    finally:
        process.stdin.close()
        process.stdout.close()
    if process.wait(timeout=30):
        raise ValueError("Uninspected blob reader failed")
    return paths


def ref_inventory(repo):
    result = {}
    raw = git(repo, "for-each-ref", "--format=%(refname)%00%(objectname)%00%(symref)%00%(objecttype)")
    for line in raw.splitlines():
        name, oid, symbolic, kind = line.decode("utf-8").split("\0")
        result[name] = {"oid": oid, "symbolic": symbolic, "type": kind}
    return result


def prepare_mirror(repo, output, tool_python):
    repo, output, tool_python = repo.resolve(), output.resolve(), tool_python.resolve()
    if output.exists() or output.is_relative_to(repo) or output.is_relative_to(ROOT):
        raise ValueError("Choose a new mirror directory outside source")
    version = subprocess.run([str(tool_python), "-c", "import importlib.metadata; print(importlib.metadata.version('git-filter-repo'))"],
                             capture_output=True, text=True, timeout=30)
    if version.returncode or version.stdout.strip() != "2.47.0":
        raise ValueError("Use the hash-locked git-filter-repo 2.47.0 tool environment")
    output.mkdir(parents=True, mode=0o700)
    policies = [Path(__file__).resolve(), Path(__file__).with_name("git_privacy.py").resolve(),
                Path(__file__).with_name("source_hygiene.py").resolve()]
    policy_hashes = {str(path): hashlib.sha256(path.read_bytes()).hexdigest() for path in policies}
    mirror = output/"sanitized.git"
    original_refs = git(repo, "show-ref")
    source_refs = ref_inventory(repo)
    if any(name.startswith("refs/privacy-rewrite-input/") for name in source_refs):
        raise ValueError("Reserved privacy rewrite namespace already exists")
    git(repo, "clone", "--mirror", "--no-local", "--", str(repo), str(mirror))
    before = audit_history(mirror)
    (output/"before.json").write_text(json.dumps(before,indent=2)+"\n",encoding="utf-8")
    sensitive_binary_paths = uninspected_sensitive_paths(mirror, before["uninspectedRetainedBlobs"])
    paths = sorted(set(before["excludedPaths"]) | sensitive_binary_paths)
    if any("\n" in path or "\r" in path or "==>" in path for path in paths):
        raise ValueError("Unsupported historical filename; manual review required")
    pathfile = output/"remove-paths.txt"
    pathfile.write_bytes("".join("literal:"+path+"\n" for path in paths).encode("utf-8"))
    cloned_refs = ref_inventory(mirror)
    if ({name: value["oid"] for name, value in cloned_refs.items()} !=
            {name: value["oid"] for name, value in source_refs.items()}):
        raise ValueError("Mirror did not preserve every original ref")
    # Removing a remote can delete its tracking refs. Remove configuration only
    # and hide origin-tracking refs from filter-repo's automatic migration.
    git(mirror, "config", "--remove-section", "remote.origin")
    operational_refs = {name: value for name, value in source_refs.items() if name.startswith("refs/codex/")}
    unsupported = [row for row in before["nonCommitRefs"] if row["ref"] not in operational_refs]
    if unsupported:
        raise ValueError("Non-commit user ref requires manual review; original repository preserved")
    for name, value in operational_refs.items():
        git(mirror, "update-ref", "--no-deref", "-d", name, value["oid"])
    intended_refs = {name: value for name, value in source_refs.items() if name not in operational_refs}
    saved_refs = {}
    for name, value in intended_refs.items():
        if name.startswith("refs/remotes/origin/"):
            temporary = "refs/privacy-rewrite-input/" + hashlib.sha256(name.encode()).hexdigest()
            git(mirror, "update-ref", temporary, value["oid"])
            git(mirror, "update-ref", "--no-deref", "-d", name, value["oid"])
            saved_refs[name] = temporary
    callback = output/"sanitize-blob.py"
    callback.write_text("import sys\nsys.path.insert(0,"+repr(str(Path(__file__).resolve().parent))+
                        ")\nfrom git_privacy import sanitize_blob\nblob.data=sanitize_blob(blob.data)\n",encoding="utf-8")
    message = output/"sanitize-message.py"
    message.write_text("import sys\nsys.path.insert(0,"+repr(str(Path(__file__).resolve().parent))+
                       ")\nfrom git_privacy import sanitize\nreturn sanitize(message)\n",encoding="utf-8")
    command = [str(tool_python), "-m", "git_filter_repo", "--force", "--sensitive-data-removal", "--no-fetch",
               "--paths-from-file", str(pathfile), "--invert-paths", "--blob-callback", str(callback),
               "--message-callback", str(message)]
    with (output/"rewrite.log").open("wb") as log:
        result = subprocess.run(command,cwd=mirror,env=git_environment(),stdout=log,stderr=subprocess.STDOUT,timeout=1200)
    if result.returncode:
        raise ValueError("Mirror rewrite failed; inspect private rewrite.log")
    filtered_refs = ref_inventory(mirror)
    dropped_refs = []
    for name, temporary in saved_refs.items():
        if temporary not in filtered_refs:
            dropped_refs.append(name)
            continue
        git(mirror, "update-ref", name, filtered_refs[temporary]["oid"])
        git(mirror, "update-ref", "-d", temporary)
    current_refs = ref_inventory(mirror)
    for name, value in intended_refs.items():
        if value["symbolic"] and name in current_refs:
            if value["symbolic"] not in current_refs:
                raise ValueError("Rewritten symbolic ref target disappeared; manual review required")
            git(mirror, "symbolic-ref", name, value["symbolic"])
    final_refs = ref_inventory(mirror)
    missing_refs = sorted(set(intended_refs) - set(final_refs))
    if set(final_refs) - set(intended_refs):
        raise ValueError("Rewrite introduced unexpected ref names")
    if set(missing_refs) != set(dropped_refs):
        raise ValueError("Rewrite lost an original ref; manual review required")
    # Bind every surviving ref, including annotated tags, to filter-repo's map.
    map_rows = (mirror/"filter-repo/ref-map").read_text(encoding="utf-8").splitlines()[1:]
    mapped = {row.split()[2]: row.split()[:2] for row in map_rows if row.strip()}
    for name, value in final_refs.items():
        exported = saved_refs.get(name, name)
        if exported not in mapped or mapped[exported] != [source_refs[name]["oid"], value["oid"]]:
            raise ValueError("Rewritten ref does not match the audited ref map")
    after = audit_history(mirror)
    after_sensitive_binary_paths = uninspected_sensitive_paths(mirror, after["uninspectedRetainedBlobs"])
    after["rawBinaryPatternScan"] = {
        "blobsScanned": len(after["uninspectedRetainedBlobs"]),
        "sensitivePathCount": len(after_sensitive_binary_paths),
        "valuesIncluded": False,
    }
    after["pathAndPatternClean"] = after["pathAndPatternClean"] and not after_sensitive_binary_paths
    if any(hashlib.sha256(path.read_bytes()).hexdigest() != policy_hashes[str(path)] for path in policies):
        raise ValueError("Privacy policy changed during rewrite; do not publish the mirror")
    unchanged = git(repo, "show-ref") == original_refs
    receipt = {"scope":"external-mirror-only", "mirror":str(mirror), "toolVersion":"2.47.0",
               "originalRefsUnchanged":unchanged,"pushPerformed":False,"providerRotationPerformed":False,
               "policyHashes":policy_hashes,"refsBefore":source_refs,"refsAfter":final_refs,
               "excludedOperationalRefs":operational_refs,
               "droppedEmptyRefs":dropped_refs,
               "rawBinaryPatternScan":{"beforeBlobsScanned":len(before["uninspectedRetainedBlobs"]),
                   "beforeSensitivePathCount":len(sensitive_binary_paths),
                   "afterBlobsScanned":after["rawBinaryPatternScan"]["blobsScanned"],
                   "afterSensitivePathCount":len(after_sensitive_binary_paths),
                   "valuesIncluded":False},
               "before":before,"after":after}
    (output/"receipt.json").write_text(json.dumps(receipt,indent=2)+"\n",encoding="utf-8")
    if not unchanged:
        raise ValueError("Original refs changed concurrently; do not publish the mirror")
    return receipt


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo",type=Path,default=ROOT)
    parser.add_argument("--output",type=Path,required=True)
    parser.add_argument("--tool-python",type=Path,required=True)
    args=parser.parse_args()
    result=prepare_mirror(args.repo,args.output,args.tool_python)
    print(json.dumps({"mirror":result['mirror'],"pathAndPatternClean":result['after']['pathAndPatternClean'],
                      "uninspectedRetainedBlobs":len(result['after']['uninspectedRetainedBlobs']),
                      "originalRefsUnchanged":result['originalRefsUnchanged'],"pushPerformed":False}))


if __name__=='__main__':main()
