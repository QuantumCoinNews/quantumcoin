"""Synthetic privacy fixtures; never rewrite or commit the user's repository."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

from git_privacy import apply_index, audit_history, detect, entries, git, logical_text, plan_index, sanitize_blob


class GitPrivacyTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="qc-git-privacy-")
        self.addCleanup(temporary.cleanup)
        self.base = Path(temporary.name).resolve()
        self.repo = self.base / "source"
        self.repo.mkdir()
        git(self.repo, "init", "--quiet")
        git(self.repo, "config", "core.autocrlf", "false")
        self.secret = b"sk-" + b"S" * 40
        self.write("main.go", b"package main\nfunc main() {}\n")
        self.write(".env", b"synthetic environment custody\n")
        self.write(".wallet", b"synthetic wallet custody; not a real key\n")
        self.write("notes.md", b"historical inline test token " + self.secret + b"\n")
        self.write("alias.md", self.secret)
        self.write("alias-copy.md", self.secret)
        self.write("legacy.txt", b"public old file\n")
        self.commit()
        git(self.repo, "mv", ".env", "old.env")
        self.commit()

    def write(self, name, data):
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)

    def commit(self):
        git(self.repo, "add", "--force", ".")
        git(self.repo, "-c", "user.name=Privacy fixture", "-c", "user.email=fixture@example.invalid",
            "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + str(self.base/"no-hooks"),
            "commit", "--quiet", "-m", "synthetic privacy fixture")

    def test_plan_is_read_only_and_contains_no_secret_values(self):
        before = entries(self.repo)
        plan = plan_index(self.repo)
        self.assertEqual({row["path"] for row in plan["remove"]}, {".wallet", "old.env"})
        self.assertEqual(before, entries(self.repo))
        self.assertNotIn(self.secret.decode(), json.dumps(plan))

    def test_apply_preserves_worktree_staged_source_and_prior_deletions(self):
        self.write("main.go", b"package main\n// staged source\nfunc main() {}\n")
        git(self.repo, "add", "main.go")
        self.write("main.go", b"package main\n// newer unstaged edit\nfunc main() {}\n")
        git(self.repo, "rm", "--quiet", "legacy.txt")
        before = entries(self.repo)
        wallet = (self.repo/".wallet").read_bytes()
        worktree = (self.repo/"main.go").read_bytes()
        head = git(self.repo, "rev-parse", "HEAD")
        result = apply_index(self.repo, plan_index(self.repo), self.base/"evidence")
        self.assertTrue(result["ok"])
        self.assertEqual(entries(self.repo)["main.go"], before["main.go"])
        self.assertEqual((self.repo/"main.go").read_bytes(), worktree)
        self.assertEqual((self.repo/".wallet").read_bytes(), wallet)
        self.assertEqual(git(self.repo, "rev-parse", "HEAD"), head)
        self.assertNotIn("legacy.txt", entries(self.repo))
        self.assertNotIn(".wallet", entries(self.repo))
        self.assertTrue((self.base/"evidence/before.index").is_file())

    def test_stale_or_modified_plan_cannot_remove_source(self):
        plan = plan_index(self.repo)
        altered = copy.deepcopy(plan)
        altered["remove"].append({"path": "main.go", "class": "secret"})
        with self.assertRaisesRegex(ValueError, "Stale or modified"):
            apply_index(self.repo, altered, self.base/"modified")
        self.write("new.go", b"package main\n")
        git(self.repo, "add", "new.go")
        before = entries(self.repo)
        with self.assertRaisesRegex(ValueError, "Stale or modified"):
            apply_index(self.repo, plan, self.base/"stale")
        self.assertEqual(entries(self.repo), before)

    def test_git_environment_cannot_redirect_cleanup_to_another_repository(self):
        other=self.base/'other';other.mkdir()
        git(other,'init','--quiet')
        (other/'safe.txt').write_bytes(b'unrelated staged source\n')
        git(other,'add','safe.txt')
        unchanged=(other/'.git/index').read_bytes()
        with patch.dict(os.environ,{'GIT_DIR':str(other/'.git'),'GIT_WORK_TREE':str(other),
                                    'GIT_INDEX_FILE':str(other/'.git/index'),'GIT_NAMESPACE':'unrelated'}):
            result=apply_index(self.repo,plan_index(self.repo),self.base/'isolated-evidence')
        self.assertTrue(result['ok'])
        self.assertEqual((other/'.git/index').read_bytes(),unchanged)
        self.assertNotIn('.wallet',entries(self.repo))

    def test_existing_git_lock_is_not_removed(self):
        lock = self.repo/".git/index.lock"
        lock.write_bytes(b"another writer owns this lock")
        before = entries(self.repo)
        with self.assertRaises(FileExistsError):
            apply_index(self.repo, plan_index(self.repo), self.base/"locked")
        self.assertEqual(lock.read_bytes(), b"another writer owns this lock")
        self.assertEqual(entries(self.repo), before)

    def test_history_sees_renames_all_aliases_and_redacts_output(self):
        result = audit_history(self.repo)
        self.assertEqual(result["commits"], 2)
        self.assertIn(".env", result["excludedPaths"])
        self.assertIn("old.env", result["excludedPaths"])
        aliases = next(row for row in result["retainedTextFindings"] if "alias.md" in row["paths"])
        self.assertEqual(aliases["paths"], ["alias-copy.md", "alias.md"])
        self.assertNotIn(self.secret.decode(), json.dumps(result))
        self.assertFalse(result["pathAndPatternClean"])

    def test_full_index_audit_includes_unchanged_retained_blobs(self):
        self.assertEqual(git(self.repo, "diff", "--cached"), b"")
        result = audit_history(self.repo, index_only=True)
        self.assertEqual(result["scope"], "current-index")
        self.assertTrue(any("notes.md" in row["paths"] for row in result["retainedTextFindings"]))
        self.assertNotIn(self.secret.decode(), json.dumps(result))

    def test_utf16_credentials_and_paths_are_sanitized_without_changing_encoding(self):
        data = b"\xff\xfe" + (self.secret.decode() + " C:\\Users\\USER\\file").encode("utf-16-le")
        cleaned = sanitize_blob(data)
        self.assertTrue(cleaned.startswith(b"\xff\xfe"))
        logical, _ = logical_text(cleaned)
        self.assertEqual(detect(logical), [])
        self.assertNotIn(b"private-user", logical)
        self.assertEqual(sanitize_blob(cleaned), cleaned)
        self.assertEqual(sanitize_blob(b"\0binary" + self.secret), b"\0binary" + self.secret)

    def test_redaction_preserves_quoted_assignments_and_is_idempotent(self):
        token=b"synthetic-"+b"long-token-for-test"
        value=b"synthetic-"+b"long-secret-for-test"
        oauth_client_secret=b"synthetic-"+b"oauth-client-secret-fixture"
        source=(b'{"refresh_token":"'+token+b'"}\n'+b"$env:X_APP_SECRET = '"+value+b"' # keep comment\n"+
                b'X_OAUTH2_CLIENT_SECRET="'+oauth_client_secret+b'"\n'+
                b'X_OAUTH2_TOKEN_JSON={"access_token":"'+token+b'"}\n')
        cleaned=sanitize_blob(source)
        self.assertIn(b"'[REDACTED_SECRET]' # keep comment",cleaned)
        self.assertEqual(detect(cleaned),[])
        self.assertNotIn(oauth_client_secret,cleaned)
        self.assertNotIn(token,cleaned)
        self.assertEqual(sanitize_blob(cleaned),cleaned)

    def test_rtf_terminator_is_preserved_but_does_not_hide_credentials(self):
        raw=b'{\\rtf1 '+self.secret+b'}\0'
        cleaned=sanitize_blob(raw)
        self.assertTrue(cleaned.endswith(b'\0'))
        self.assertEqual(detect(cleaned.rstrip(b'\0')),[])

    def test_non_commit_refs_are_explicit_uncovered_history(self):
        tree=git(self.repo,'rev-parse','HEAD^{tree}').decode().strip()
        git(self.repo,'update-ref','refs/codex/fixture/base',tree)
        result=audit_history(self.repo)
        self.assertEqual(result['nonCommitRefs'][0]['ref'],'refs/codex/fixture/base')
        self.assertFalse(result['pathAndPatternClean'])

    @unittest.skipUnless(importlib.util.find_spec("git_filter_repo"), "requires isolated hash-locked privacy tool")
    def test_independent_mirror_rewrite_keeps_original_refs_and_source(self):
        from prepare_history_mirror import prepare_mirror
        head = git(self.repo, "rev-parse", "HEAD")
        tree = git(self.repo, "rev-parse", "HEAD^{tree}").decode().strip()
        independent = git(self.repo, "-c", "user.name=Privacy fixture", "-c", "user.email=fixture@example.invalid",
                          "commit-tree", tree, "-p", head.decode().strip(), data=b"independent remote-only tip\n").decode().strip()
        git(self.repo, "update-ref", "refs/remotes/origin/not-merged", independent)
        git(self.repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/not-merged")
        git(self.repo, "-c", "user.name=Privacy fixture", "-c", "user.email=fixture@example.invalid",
            "tag", "-a", "privacy-fixture", "-m", "synthetic tag")
        git(self.repo, "update-ref", "refs/codex/fixture/base", tree)
        source = (self.repo/"main.go").read_bytes()
        result = prepare_mirror(self.repo, self.base/"history", Path(sys.executable))
        self.assertTrue(result["after"]["pathAndPatternClean"])
        self.assertEqual(git(self.repo, "rev-parse", "HEAD"), head)
        self.assertEqual(git(Path(result["mirror"]), "show", "HEAD:main.go"), source)
        self.assertTrue((self.repo/".wallet").exists())
        self.assertEqual(set(result["refsBefore"])-set(result["excludedOperationalRefs"]),set(result["refsAfter"]))
        self.assertEqual(result["refsAfter"]["refs/remotes/origin/HEAD"]["symbolic"],"refs/remotes/origin/not-merged")
        self.assertEqual(result["droppedEmptyRefs"],[])
        self.assertEqual(result['after']['nonCommitRefs'],[])


if __name__ == "__main__":
    unittest.main()
