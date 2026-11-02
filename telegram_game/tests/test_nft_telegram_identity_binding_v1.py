import os
import subprocess
import sys
import tempfile
from pathlib import Path

telegram_game_root = Path(__file__).resolve().parents[1]
child_code = r"""
import hashlib
import hmac
import json
import time
import urllib.parse
from telegram_game import dev_api

helper = getattr(dev_api, "_nft_verified_player_id_v1", None)
if helper is None:
    print("EXPECTED_RED: NFT Telegram identity binding helper is missing")
    raise SystemExit(42)

token = "qma0066-identity-test-token"

def signed_init_data(user_id, auth_date):
    values = {
        "auth_date": str(auth_date),
        "user": json.dumps({"id": user_id}, separators=(",", ":")),
    }
    check = "\n".join(f"{key}={values[key]}" for key in sorted(values))
    secret = hmac.new(b"WebAppData", token.encode(), hashlib.sha256).digest()
    values["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
    return urllib.parse.urlencode(values)

valid = signed_init_data(123456, int(time.time()))
assert helper(valid, "tg_123456") == "tg_123456"
assert not helper(valid, "tg_999999"), "Signed user must not claim another playerId"
assert not helper("", "tg_123456"), "Missing signature must be rejected"
assert not helper(valid + "x", "tg_123456"), "Tampered signature must be rejected"

expired = signed_init_data(123456, int(time.time()) - 86500)
assert not helper(expired, "tg_123456"), "Expired signature must be rejected"
print("PASS: NFT Telegram identity is bound to signed, current initData")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-identity-test-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-identity-test-token"
    env["PYTHONPATH"] = (
        str(telegram_game_root) + os.pathsep + env.get("PYTHONPATH", "")
    )
    completed = subprocess.run(
        [sys.executable, "-c", child_code],
        cwd=temp_dir,
        env=env,
        text=True,
        capture_output=True,
    )
    print(completed.stdout, end="")
    print(completed.stderr, end="", file=sys.stderr)
    exit_code = completed.returncode

raise SystemExit(exit_code)
