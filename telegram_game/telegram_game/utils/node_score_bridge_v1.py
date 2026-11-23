"""Server-to-server advisory scores. Never expose the signing key to WebApps.

The caller is the authoritative game service and supplies its validated absolute
score and persisted monotonic player sequence, not a client-provided score delta.
Missing authority configuration fails closed; this module creates no keys.
"""
import json
import os
import re
from pathlib import Path

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

VERSION = "QC_GAME_SCORE_EVENT_V1"
_NAME = re.compile(r"^[A-Za-z0-9_-]{1,64}$")

def signing_bytes(event):
    if event.get("version") != VERSION or not _NAME.fullmatch(event.get("player", "")) or not _NAME.fullmatch(event.get("network", "")):
        raise ValueError("invalid score identity")
    genesis = event.get("genesis", "")
    if len(genesis) != 64 or len(bytes.fromhex(genesis)) != 32:
        raise ValueError("invalid score scope")
    if type(event.get("score")) is not int or not 0 <= event["score"] <= 1000000000 or type(event.get("sequence")) is not int or not 1 <= event["sequence"] <= 9007199254740991 or type(event.get("expires")) is not int:
        raise ValueError("invalid score bounds")
    return json.dumps([VERSION,event["network"],genesis,event["player"],event["score"],event["sequence"],event["expires"]],separators=(",", ":"),ensure_ascii=True).encode("ascii")

def sign_authoritative_score(event, key_path=None):
    path = key_path or os.getenv("QC_GAME_SCORE_SIGNING_KEY_FILE", "")
    if not path:
        raise ValueError("game score signing authority is not configured")
    # A dedicated raw Ed25519 seed file, supplied by the operator.
    with Path(path).open("rb") as source:
        seed = source.read(33)
    if len(seed) != 32:
        raise ValueError("invalid score authority key file")
    result = dict(event)
    result["signature"] = Ed25519PrivateKey.from_private_bytes(seed).sign(signing_bytes(event)).hex()
    return result

async def publish_authoritative_score(base, event, token, timeout=10.0):
    """Deprecated: the root score API is retired; never transmit credentials."""
    raise ValueError("root_game_retired: use canonical Telegram session/evidence/ledger flows")
