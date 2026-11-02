import os
import subprocess
import sys
import tempfile
from pathlib import Path

telegram_game_root = Path(__file__).resolve().parents[1]
child_code = r"""
from concurrent.futures import ThreadPoolExecutor
import hashlib
import hmac
import json
import sqlite3
import time
import urllib.parse
from telegram_game import dev_api

t0 = int(time.time())
values = {
    "auth_date": str(t0),
    "user": json.dumps({"id": 123456}, separators=(",", ":")),
}
check = "\n".join(f"{key}={values[key]}" for key in sorted(values))
secret = hmac.new(
    b"WebAppData",
    b"qma0066-concurrency-test-token",
    hashlib.sha256,
).digest()
values["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
init_data = urllib.parse.urlencode(values)

dev_api._now = lambda: t0
started = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(
        playerId="tg_123456",
        initData=init_data,
    )
)
assert started.get("ok"), started
session_id = started["sessionId"]
with sqlite3.connect(dev_api.APP_DB) as conn:
    dev_api._touch_player(conn, "tg_123456", None)
    dev_api._ensure_qc_address_column(conn)
    conn.execute(
        "UPDATE players SET qc_address=? WHERE player_id=?",
        ("QC_NFT_RACE_TEST", "tg_123456"),
    )

dev_api._now = lambda: t0 + 1
game_start = dev_api.mine_start(
    dev_api.MineStartIn(
        playerId="tg_123456",
        siteId=1,
        initData=init_data,
        nftSessionId=session_id,
    )
)
assert game_start.get("ok"), game_start

dev_api._now = lambda: t0 + 61

def finish(_):
    return dev_api.nft_monthly_session_complete(
        dev_api.MonthlyNFTSessionCompleteIn(
            playerId="tg_123456",
            initData=init_data,
            sessionId=session_id,
        )
    )

with ThreadPoolExecutor(max_workers=12) as pool:
    results = list(pool.map(finish, range(12)))

successes = sum(result.get("ok") is True for result in results)
with sqlite3.connect(dev_api.APP_DB) as conn:
    row = conn.execute(
        "SELECT status, completed_at FROM monthly_nft_game_sessions "
        "WHERE session_id=?",
        (session_id,),
    ).fetchone()
    evidence = conn.execute(
        "SELECT COUNT(*), MIN(activity_verified), MAX(activity_verified) "
        "FROM monthly_nft_verified_game_sessions WHERE session_id=?",
        (session_id,),
    ).fetchone()

assert successes == 1, results
assert row == ("COMPLETED", t0 + 61), row
assert evidence == (1, 1, 1), evidence
print("PASS: 12 concurrent completions produced exactly one active evidence row")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-session-race-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-concurrency-test-token"
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
