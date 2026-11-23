"""Shared server-only credential parsing for TGWT provider verification."""
from __future__ import annotations

import os
from pathlib import Path
from urllib.parse import urlparse

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey


_REPOSITORY = Path(__file__).resolve().parents[2]


def reward_signing_key():
    """Load the campaign receipt signer without exposing key material."""
    raw = (os.getenv("QC_TGWT_REWARD_SIGNING_SEED_FILE") or
           os.getenv("QC_X_REWARD_SIGNING_SEED_FILE") or "").strip()
    if not raw:
        return None
    try:
        path = Path(raw).expanduser()
        if not path.is_absolute():
            return None
        path = path.resolve(strict=True)
        if path.is_relative_to(_REPOSITORY) or path.stat().st_size != 32:
            return None
        return Ed25519PrivateKey.from_private_bytes(path.read_bytes())
    except (OSError, ValueError):
        return None


def state_key(name: str):
    try:
        key = bytes.fromhex(os.getenv(name, ""))
    except ValueError:
        return None
    return key if len(key) == 32 else None


def https_callback(raw: str, path: str):
    """Return a valid absolute HTTPS callback or None."""
    try:
        parsed = urlparse(str(raw or "").strip())
        if (parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password
                or parsed.port or parsed.path != path or parsed.query or parsed.fragment):
            return None
        return parsed.geturl()
    except ValueError:
        return None
