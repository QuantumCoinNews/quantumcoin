import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile

spec = importlib.util.spec_from_file_location('publisher_source', Path(__file__).with_name('build_publisher_source.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class SourceBoundary(unittest.TestCase):
    def test_reproducible_archive_excludes_runtime_and_credentials(self):
        with tempfile.TemporaryDirectory(prefix='qc-source-test-') as temporary:
            a, b = Path(temporary)/'a.zip', Path(temporary)/'b.zip'
            self.assertEqual(module.build(a)['sha256'], module.build(b)['sha256'])
            with zipfile.ZipFile(a) as archive:
                names = archive.namelist()
                for name in names:
                    self.assertNotRegex(name, r'node_modules|/dist/|/logs/|/tmp/|/assets/|\.log$|\.mp[34]$|\.wav$|\.lock$|pool_state|state_slot|last\.json|recent-topics')
                    self.assertFalse(name.endswith('/.env'))
                self.assertIn('autopublisher/package-lock.json', names)
                self.assertIn('autopublisher/src/content/pool/topics_pool.json', names)
                self.assertIn('publisher-source-manifest.json', names)

    def test_secret_pattern_blocks_export_without_exposing_the_value(self):
        with tempfile.TemporaryDirectory(prefix='qc-secret-export-') as temporary:
            key = b'sk-proj-' + b'A' * 48
            with patch.object(module, 'sources', return_value=iter([('autopublisher/bad.ts', key)])):
                with self.assertRaisesRegex(ValueError, 'credential') as error:
                    module.build(Path(temporary)/'blocked.zip')
            self.assertNotIn(key.decode(), str(error.exception))
            self.assertFalse((Path(temporary)/'blocked.zip').exists())

if __name__ == '__main__':
    unittest.main()
