import os
from pathlib import Path
import tempfile
import unittest

from go_source_audit import run


class ToolPathResolutionTest(unittest.TestCase):
    def test_child_uses_supplied_tool_directory_instead_of_parent_path(self):
        with tempfile.TemporaryDirectory(prefix="qc-selected-tool-") as directory:
            root = Path(directory)
            tool = root / ("qc_fixture_tool.cmd" if os.name == "nt" else "qc_fixture_tool")
            tool.write_text("@echo off\necho selected-tool-directory\n" if os.name == "nt" else
                            "#!/bin/sh\necho selected-tool-directory\n", encoding="ascii")
            if os.name != "nt":
                tool.chmod(0o700)
            environment = {**os.environ, "PATH": str(root)}
            record, output = run([tool.name], root, environment, "selected-tool-fixture", timeout=30)
            self.assertEqual(record["exitCode"], 0)
            self.assertEqual(output.strip(), "selected-tool-directory")
            self.assertEqual(Path(record["executable"]).resolve(), tool.resolve())


if __name__ == "__main__":
    unittest.main()
