"""Independent operator entry: uvicorn ...:create_app --factory.

No production defaults, key generation, game DB access or node credentials.
TLS must be terminated by the independent operator's HTTPS reverse proxy.
"""
import json
import os
from pathlib import Path
import sqlite3
import time


def create_app():
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    from .utils.nft_publication_witness_v1 import create_witness_app_v1
    from .utils.nft_draw_evidence_v1 import hex_bytes, require
    from .utils.nft_mint_bridge_v1 import _unique_object
    key_path = Path(os.environ["QMA_WITNESS_KEY_FILE"])
    db_path = Path(os.environ["QMA_WITNESS_DB_FILE"])
    scope_path = Path(os.environ["QMA_WITNESS_SCOPE_FILE"])
    require(all(p.is_absolute() for p in (key_path, db_path, scope_path)), "witness_absolute_paths_required")
    require(key_path.is_file() and key_path.stat().st_size <= 256 and scope_path.is_file()
            and scope_path.stat().st_size <= 4096 and db_path.parent.is_dir(), "witness_operator_files_required")
    key = Ed25519PrivateKey.from_private_bytes(hex_bytes(key_path.read_text(encoding="ascii").strip(), 32))
    scope = json.loads(scope_path.read_text(encoding="utf-8-sig"), object_pairs_hook=_unique_object)
    return create_witness_app_v1(lambda: sqlite3.connect(db_path, timeout=20), key,
        os.environ["QMA_WITNESS_PUBLIC_ORIGIN"], os.environ["QMA_WITNESS_SUBMIT_TOKEN"],
        scope, lambda: int(time.time()))
