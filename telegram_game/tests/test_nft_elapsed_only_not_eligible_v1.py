import os
import subprocess
import sys
import tempfile
from pathlib import Path

telegram_game_root = Path(__file__).resolve().parents[1]
child_code = r"""
import sqlite3
from telegram_game import dev_api

ts = dev_api._now()
period = dev_api._nft_month_key(ts)
user_id = "tg_elapsed_only_test"

with sqlite3.connect(dev_api.APP_DB) as conn:
    conn.execute(
        "CREATE TABLE monthly_nft_verified_game_sessions ("
        "session_id TEXT PRIMARY KEY, user_id TEXT NOT NULL, "
        "period TEXT NOT NULL, completed_at INTEGER NOT NULL, "
        "activity_verified INTEGER NOT NULL DEFAULT 0)"
    )
    conn.execute(
        "INSERT INTO monthly_nft_verified_game_sessions "
        "(session_id, user_id, period, completed_at, activity_verified) "
        "VALUES(?,?,?,?,0)",
        ("elapsed_only_session", user_id, period, ts),
    )

result = dev_api.nft_monthly_candidate_create(
    dev_api.MonthlyNFTCandidateIn(
        userId=user_id,
        period=period,
        activity={"miningSessions": 99},
    )
)

with sqlite3.connect(dev_api.APP_DB) as conn:
    table = conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' "
        "AND name='monthly_nft_candidates'"
    ).fetchone()
    count = (
        conn.execute(
            "SELECT COUNT(*) FROM monthly_nft_candidates WHERE user_id=?",
            (user_id,),
        ).fetchone()[0]
        if table else 0
    )

if result.get("ok") or count:
    print("EXPECTED_RED: elapsed-only session opened NFT candidacy")
    raise SystemExit(42)

print("PASS: elapsed-only session did not open NFT candidacy")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-elapsed-test-") as temp_dir:
    env = os.environ.copy()
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
