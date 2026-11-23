"""Build a deterministic source-only publisher archive; never copy runtime data."""
import argparse
import hashlib
import json
import re
from pathlib import Path
import zipfile

ROOT = Path(__file__).resolve().parents[1]
BASE = ROOT / 'autopublisher'
FILES = {
    'publisher.mjs', 'scripts/build.ps1', 'scripts/hold-build-lock.ps1', 'package.json', 'package-lock.json', 'tsconfig.json', 'README.md',
    '.gitignore', '.env.example', 'publisher.settings', 'run-daily.ps1',
    'run-daily.cmd', 'autopublish-simple.ps1', 'autopublish-3platform.ps1',
    'publish-all.ps1', 'scripts/run-slot.ps1', 'scripts/hold-runtime-lock.ps1',
    'scripts/secret-storage.ps1', 'scripts/install-schedule.ps1',
    'src/content/pool/topics_pool.json',
}

def sources():
    chosen = {BASE / name for name in FILES}
    chosen.update((BASE / 'src').rglob('*.ts'))
    chosen.update((BASE / 'tests').glob('*.mjs'))
    for file in sorted(chosen):
        if file.is_symlink() or not file.is_file() or file.stat().st_size > 4 * 1024 * 1024:
            raise ValueError('invalid required source file')
        yield file.relative_to(ROOT).as_posix(), file.read_bytes()

def build(output):
    if not output.is_absolute() or output.is_relative_to(ROOT):
        raise ValueError('source archive must be outside the checkout')
    entries = dict(sources())
    for name, data in entries.items():
        if re.search(rb'sk-(?:proj-)?[A-Za-z0-9_-]{36,}|-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----', data):
            raise ValueError('potential credential material in source: ' + name)
    proof = json.loads((BASE / 'dist/build-manifest.json').read_text(encoding='utf-8-sig'))
    if proof.get('version') != 'QC_BUILD_V1':
        raise ValueError('successful verified build required')
    for item in proof['source']:
        content = entries.get('autopublisher/' + item['path'])
        if content is None or hashlib.sha256(content).hexdigest() != item['sha256']:
            raise ValueError('source changed since verified build')
    manifest = {name: hashlib.sha256(data).hexdigest() for name, data in entries.items()}
    entries['publisher-source-manifest.json'] = (json.dumps({
        'version': 'QC_PUBLISHER_SOURCE_V1', 'files': manifest,
        'mediaIncluded': False, 'credentialsIncluded': False,
        'runtimeStateIncluded': False,
    }, sort_keys=True, indent=2) + '\n').encode()
    output.parent.mkdir(parents=True, exist_ok=True)
    with output.open('xb') as stream:
        with zipfile.ZipFile(stream, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
            for name, data in sorted(entries.items()):
                info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
                info.compress_type = zipfile.ZIP_DEFLATED
                info.external_attr = 0o100644 << 16
                archive.writestr(info, data)
    return {'files': len(entries), 'bytes': output.stat().st_size,
            'sha256': hashlib.sha256(output.read_bytes()).hexdigest()}

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    print(json.dumps(build(args.output)))
