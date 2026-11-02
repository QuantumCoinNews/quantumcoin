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

token = "qma0066-action-completion-test-token"
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

no_action_start = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId="tg_123456", initData=alice)
)
assert no_action_start.get("ok"), no_action_start
no_action_id = no_action_start["sessionId"]

dev_api._now = lambda: t0 + 61
no_action_done = dev_api.nft_monthly_session_complete(
    dev_api.MonthlyNFTSessionCompleteIn(
        playerId="tg_123456", initData=alice, sessionId=no_action_id
    )
)
with sqlite3.connect(dev_api.APP_DB) as conn:
    table = conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' "
        "AND name='monthly_nft_verified_game_sessions'"
    ).fetchone()
    no_action_rows = (
        conn.execute(
            "SELECT COUNT(*) FROM monthly_nft_verified_game_sessions "
            "WHERE session_id=?",
            (no_action_id,),
        ).fetchone()[0]
        if table else 0
    )

if no_action_done.get("ok") or no_action_rows:
    print("EXPECTED_RED: elapsed session completed without verified game action")
    raise SystemExit(42)

dev_api._now = lambda: t0
with sqlite3.connect(dev_api.APP_DB) as conn:
    dev_api._touch_player(conn, "tg_999999", None)
    dev_api._ensure_qc_address_column(conn)
    conn.execute(
        "UPDATE players SET qc_address=? WHERE player_id=?",
        ("QC_NFT_ACTION_COMPLETION_TEST", "tg_999999"),
    )

action_start = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(playerId="tg_999999", initData=bob)
)
assert action_start.get("ok"), action_start
action_id = action_start["sessionId"]

dev_api._now = lambda: t0 + 1
game_start = dev_api.mine_start(
    dev_api.MineStartIn(
        playerId="tg_999999",
        siteId=1,
        initData=bob,
        nftSessionId=action_id,
    )
)
assert game_start.get("ok"), game_start

dev_api._now = lambda: t0 + 61
action_done = dev_api.nft_monthly_session_complete(
    dev_api.MonthlyNFTSessionCompleteIn(
        playerId="tg_999999", initData=bob, sessionId=action_id
    )
)
assert action_done.get("ok") and action_done.get("activityVerified"), action_done

with sqlite3.connect(dev_api.APP_DB) as conn:
    evidence = conn.execute(
        "SELECT activity_verified FROM monthly_nft_verified_game_sessions "
        "WHERE session_id=?",
        (action_id,),
    ).fetchall()
assert evidence == [(1,)], evidence

candidate = dev_api.nft_monthly_candidate_create(
    dev_api.MonthlyNFTCandidateIn(
        userId="tg_999999", period=dev_api._nft_month_key(t0 + 61)
    )
)
assert not candidate.get("ok"), "Pre-activation candidate gate must stay closed"
print("PASS: only signed game action completed; pre-activation candidacy stayed closed")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-action-complete-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-action-completion-test-token"
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
