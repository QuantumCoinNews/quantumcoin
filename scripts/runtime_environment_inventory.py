"""Read source references only. Never read .env files or process credentials."""
import os
from pathlib import Path
import re

ROOT=Path(__file__).resolve().parents[1]
PREFIXES=('QC_','QUANTUMCOIN_','TELEGRAM_','OPENAI_','YOUTUBE_','X_','TWITTER_','INSTAGRAM_','TIKTOK_','CONTENT_')
SKIP={'.git','node_modules','__pycache__','dist','build','bin','_local_audit_reports','_wm_review'}
PATTERNS=[r'\b(?:os\.(?:Getenv|LookupEnv|getenv)|os\.environ\.get|env\.get)\(\s*[\"\x27]([A-Z][A-Z0-9_]*)',
          r'\b(?:os\.environ|env)\[\s*[\"\x27]([A-Z][A-Z0-9_]*)',
          r'\b(?:process\.)?env\.([A-Z][A-Z0-9_]*)',r'\$env:([A-Za-z][A-Za-z0-9_]*)',
          r'%((?:QC_|QUANTUMCOIN_|TELEGRAM_|OPENAI_|YOUTUBE_|X_|CONTENT_)[A-Z0-9_]+)%',
          r'(?i)\b\w*env\w*\(\s*[\"\x27]([A-Z][A-Z0-9_]*)']


def inventory():
    found={}
    for directory,dirs,files in os.walk(ROOT):
        dirs[:]=[d for d in dirs if d not in SKIP and not d.startswith('_legacy')]
        for filename in files:
            path=Path(directory)/filename;relative=path.relative_to(ROOT)
            if path.suffix not in {'.go','.py','.ts','.js','.mjs','.ps1','.cmd'}:continue
            if any(p.lower() in {'tests','test'} for p in relative.parts) or re.search(r'(_test\.(go|py)|\.test\.(mjs|js)|^test_)',filename):continue
            text=path.read_text(encoding='utf-8-sig',errors='replace')
            keys={m[1] for pattern in PATTERNS for m in re.finditer(pattern,text)}
            # Selector/key arrays feed name-based readers. Include their literal
            # references, but exclude protocol/domain constants ending in _Vn.
            keys.update(m[1] for m in re.finditer(r'[\"\x27]((?:QC_|QUANTUMCOIN_|TELEGRAM_|OPENAI_|YOUTUBE_|X_|TWITTER_|CONTENT_)[A-Z0-9_]+)[\"\x27]',text)
                        if not re.search(r'_V\d+$',m[1]))
            for key in keys:
                if not key.endswith('_') and (key.startswith(PREFIXES) or key in {'ADMIN_TOKEN','HTTP_PORT'}):
                    found.setdefault(key,set()).add(relative.as_posix())
    return {k:sorted(v) for k,v in sorted(found.items())}


def owners(paths):
    result=set()
    for path in paths:
        if path.startswith('autopublisher/'):result.add('publisher')
        elif path.startswith('telegram_game/'):result.add('game')
        elif path.startswith(('scripts/','tools/','release/')):result.add('tooling')
        else:result.add('node')
    return sorted(result)
