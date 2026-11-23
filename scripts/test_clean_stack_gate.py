"""Acceptance must fail closed for incomplete or mismatched CI evidence."""
import copy
import json
from pathlib import Path
import unittest

from verify_clean_stack import gate_passed, version_matches


class CleanStackGateTests(unittest.TestCase):
    def setUp(self):
        self.lock = json.loads((Path(__file__).resolve().parents[1] /
                                "configs/windows-build-toolchain.json").read_text(encoding="utf-8"))
        self.versions = {name: self.lock[name]["version"] for name in ("go", "node", "python", "ffmpeg")}
        self.versions.update(npm=self.lock["npm"]["version"], ffprobe=self.versions["ffmpeg"],
                             gcc=self.lock["mingw"]["version"], gxx=self.lock["mingw"]["version"])
        self.groups = {"go": (None, [("test", []), ("race", [])]),
                       "game": (None, [("restore", []), ("test", [])])}
        self.checks = {group: {name: {"exitCode": 0} for name, _ in commands}
                       for group, (_, commands) in self.groups.items()}

    def accepted(self, checks=None):
        return gate_passed(self.versions, self.lock, self.groups, self.checks if checks is None else checks)

    def test_complete_evidence_passes(self):
        self.assertTrue(self.accepted())
        self.versions["node"] = "v" + self.lock["node"]["version"]
        self.assertTrue(self.accepted())

    def test_every_locked_tool_must_match(self):
        for name in self.versions:
            with self.subTest(tool=name):
                self.assertFalse(version_matches(name, None, self.lock))
                self.assertFalse(version_matches(name, "wrong-version", self.lock))
        self.versions["go"] = "go1.21.0"
        self.assertFalse(self.accepted())

    def test_missing_tool_fails(self):
        del self.versions["ffprobe"]
        self.assertFalse(self.accepted())

    def test_other_numeric_npm_version_cannot_pass(self):
        self.versions["npm"] = "10.9.3"
        self.assertFalse(self.accepted())

    def test_missing_or_renamed_stage_fails(self):
        del self.checks["go"]["race"]
        self.assertFalse(self.accepted())
        self.checks["go"]["something-else"] = {"exitCode": 0}
        self.assertFalse(self.accepted())

    def test_missing_group_fails(self):
        del self.checks["game"]
        self.assertFalse(self.accepted())

    def test_failure_timeout_or_failed_test_cannot_pass(self):
        for result in ({"exitCode": 1}, {"exitCode": None, "error": "timeout"},
                       {"exitCode": 0, "error": "timeout"},
                       {"exitCode": 0, "failedTests": ["TestAuthorization"]}):
            with self.subTest(result=result):
                checks = copy.deepcopy(self.checks)
                checks["go"]["race"] = result
                self.assertFalse(self.accepted(checks))


if __name__ == "__main__":
    unittest.main()
