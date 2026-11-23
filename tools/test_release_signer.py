"""Fail-closed preparation boundaries; never create a real signing identity."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


@unittest.skipUnless(os.name == 'nt', 'Windows key custody')
class ReleaseSignerPreparationTest(unittest.TestCase):
    def invoke(self, target):
        return subprocess.run(['pwsh', '-NoProfile', '-NonInteractive', '-File',
                               str(ROOT/'tools/prepare_release_signer.ps1'), '-Directory', str(target)],
                              stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=30)

    def test_source_directory_cannot_receive_a_release_key(self):
        target = ROOT/'must-not-create-release-key'
        self.assertFalse(target.exists())
        result = self.invoke(target)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('outside the source', result.stderr)
        self.assertFalse(target.exists())

    def test_existing_custody_material_is_preserved(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory)
            marker = target/'marker'
            marker.write_bytes(b'preserve fixture')
            result = self.invoke(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(marker.read_bytes(), b'preserve fixture')
            self.assertEqual([p.name for p in target.iterdir()], ['marker'])

    def test_redirected_input_cannot_generate_an_unattended_empty_passphrase_key(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory)/'new-key'
            result = self.invoke(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Run interactively', result.stderr)
            self.assertFalse(target.exists())


if __name__ == '__main__':
    unittest.main()
