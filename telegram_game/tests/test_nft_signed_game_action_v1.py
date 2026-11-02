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

token = "qma0066-game-action-test-token"
now = int(time.time())

def signed(user_id):
    values = {
        "auth_date": str(now),
        "user": json.dumps({"id": user_id}, separators=(",", ":")),
    }
    check = "\n".join(f"{key}={values[key]}" for key in sorted(values))
    secret = hmac.new(b"WebAppData", token.encode(), hashlib.sha256).digest()
    values["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
    return urllib.parse.urlencode(values)

alice = signed(123456)
bob = signed(999999)
player_id = "tg_123456"

with sqlite3.connect(dev_api.APP_DB) as conn:
    dev_api._touch_player(conn, player_id, None)
    dev_api._ensure_qc_address_column(conn)
    conn.execute(
        "UPDATE players SET qc_address=? WHERE player_id=?",
        ("QC_NFT_ACTION_TEST", player_id),
    )

started = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId=player_id, initData=alice)
)
assert started.get("ok") and started.get("sessionId"), started
session_id = started["sessionId"]

spoof = dev_api.mine_start(
    dev_api.MineStartIn(
        playerId=player_id,
        siteId=1,
        initData=bob,
        nftSessionId=session_id,
    )
)
assert spoof.get("ok"), spoof

with sqlite3.connect(dev_api.APP_DB) as conn:
    exists = conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' "
        "AND name='monthly_nft_game_actions'"
    ).fetchone()
    spoof_count = (
        conn.execute(
            "SELECT COUNT(*) FROM monthly_nft_game_actions WHERE session_id=?",
            (session_id,),
        ).fetchone()[0]
        if exists else 0
    )
assert spoof_count == 0, "Wrong signed identity wrote an NFT game action"

valid = dev_api.mine_start(
    dev_api.MineStartIn(
        playerId=player_id,
        siteId=1,
        initData=alice,
        nftSessionId=session_id,
    )
)
assert valid.get("ok"), valid

with sqlite3.connect(dev_api.APP_DB) as conn:
    exists = conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' "
        "AND name='monthly_nft_game_actions'"
    ).fetchone()
    rows = (
        conn.execute(
            "SELECT user_id, action_type FROM monthly_nft_game_actions "
            "WHERE session_id=?",
            (session_id,),
        ).fetchall()
        if exists else []
    )

if rows != [(player_id, "MINE_START")]:
    print("EXPECTED_RED: signed game start did not record NFT action evidence")
    raise SystemExit(42)

print("PASS: only matching signed game start recorded one NFT action")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-game-action-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-game-action-test-token"
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
