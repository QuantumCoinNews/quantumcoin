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
import sqlite3
import time
import urllib.parse
from telegram_game import dev_api

t0 = int(time.time())
token = "qma0066-resume-test-token"

def signed(user_id):
    values = {
        "auth_date": str(t0),
        "user": json.dumps({"id": user_id}, separators=(",", ":")),
    }
    check = "\n".join(f"{key}={values[key]}" for key in sorted(values))
    secret = hmac.new(b"WebAppData", token.encode(), hashlib.sha256).digest()
    values["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
    return urllib.parse.urlencode(values)

alice = signed(123456)
bob = signed(999999)
dev_api._now = lambda: t0

first = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId="tg_123456", initData=alice)
)
assert first.get("ok") and first.get("sessionId"), first

spoof = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId="tg_123456", initData=bob)
)
assert not spoof.get("ok"), spoof

second = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId="tg_123456", initData=alice)
)
if not second.get("ok") or second.get("sessionId") != first["sessionId"]:
    print("EXPECTED_RED: active signed NFT session cannot be resumed")
    raise SystemExit(42)

with sqlite3.connect(dev_api.APP_DB) as conn:
    count = conn.execute(
        "SELECT COUNT(*) FROM monthly_nft_game_sessions WHERE user_id=?",
        ("tg_123456",),
    ).fetchone()[0]
assert count == 1, count

dev_api._now = lambda: t0 + 3601
third = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId="tg_123456", initData=alice)
)
assert third.get("ok") and third["sessionId"] != first["sessionId"], third
print("PASS: signed user resumes one active session; expired session rotates")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-session-resume-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-resume-test-token"
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
