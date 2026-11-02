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

start = getattr(dev_api, "nft_monthly_session_start", None)
model = getattr(dev_api, "MonthlyNFTSessionStartIn", None)
if start is None or model is None:
    print("EXPECTED_RED: verified NFT game-session start is missing")
    raise SystemExit(42)

token = "qma0066-session-start-test-token"
values = {
    "auth_date": str(int(time.time())),
    "user": json.dumps({"id": 123456}, separators=(",", ":")),
}
check = "\n".join(f"{key}={values[key]}" for key in sorted(values))
secret = hmac.new(b"WebAppData", token.encode(), hashlib.sha256).digest()
values["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
init_data = urllib.parse.urlencode(values)

spoof = start(model(playerId="tg_999999", initData=init_data))
assert not spoof.get("ok"), "Signed user must not start another user's session"

valid = start(model(playerId="tg_123456", initData=init_data))
assert valid.get("ok") and valid.get("sessionId"), valid
session_id = valid["sessionId"]

with sqlite3.connect(dev_api.APP_DB) as conn:
    rows = conn.execute(
        "SELECT user_id, completed_at FROM monthly_nft_game_sessions "
        "WHERE session_id=?",
        (session_id,),
    ).fetchall()
assert rows == [("tg_123456", None)], rows

print("PASS: signed user starts only their own server-recorded session")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-session-start-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-session-start-test-token"
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
