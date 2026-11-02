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

complete = getattr(dev_api, "nft_monthly_session_complete", None)
model = getattr(dev_api, "MonthlyNFTSessionCompleteIn", None)
if complete is None or model is None:
    print("EXPECTED_RED: verified NFT game-session completion is missing")
    raise SystemExit(42)

token = "qma0066-session-complete-test-token"
t0 = int(time.time())

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

started = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId="tg_123456", initData=alice)
)
assert started.get("ok") and started.get("sessionId"), started
session_id = started["sessionId"]

dev_api._now = lambda: t0 + 30
early = complete(model(playerId="tg_123456", initData=alice, sessionId=session_id))
assert not early.get("ok"), "Early completion must be rejected"

with sqlite3.connect(dev_api.APP_DB) as conn:
    dev_api._touch_player(conn, "tg_123456", None)
    dev_api._ensure_qc_address_column(conn)
    conn.execute(
        "UPDATE players SET qc_address=? WHERE player_id=?",
        ("QC_NFT_COMPLETE_TEST", "tg_123456"),
    )

dev_api._now = lambda: t0 + 31
game_start = dev_api.mine_start(
    dev_api.MineStartIn(
        playerId="tg_123456",
        siteId=1,
        initData=alice,
        nftSessionId=session_id,
    )
)
assert game_start.get("ok"), game_start

dev_api._now = lambda: t0 + 61
spoof = complete(model(playerId="tg_999999", initData=bob, sessionId=session_id))
assert not spoof.get("ok"), "Another signed user must not complete this session"

done = complete(model(playerId="tg_123456", initData=alice, sessionId=session_id))
assert done.get("ok"), done

replay = complete(model(playerId="tg_123456", initData=alice, sessionId=session_id))
assert not replay.get("ok"), "Completed session must not be used twice"

with sqlite3.connect(dev_api.APP_DB) as conn:
    rows = conn.execute(
        "SELECT user_id, period, completed_at "
        "FROM monthly_nft_verified_game_sessions WHERE session_id=?",
        (session_id,),
    ).fetchall()
assert rows == [
    ("tg_123456", dev_api._nft_month_key(t0 + 61), t0 + 61)
], rows

print("PASS: signed, timed, single-use completion produced one monthly record")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-session-complete-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-session-complete-test-token"
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
