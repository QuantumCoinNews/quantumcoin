import os
import subprocess
import sys
import tempfile
from pathlib import Path

telegram_game_root = Path(__file__).resolve().parents[1]
child_code = r"""
import sqlite3
from telegram_game import dev_api

result = dev_api.nft_monthly_candidate_create(
    dev_api.MonthlyNFTCandidateIn(
        userId="tg_unverified_test_user",
        source="client",
        activity={"miningSessions": 99, "completedGameSessions": 99},
    )
)

with sqlite3.connect(dev_api.APP_DB) as conn:
    table_exists = conn.execute(
        "SELECT 1 FROM sqlite_master "
        "WHERE type='table' AND name='monthly_nft_candidates'"
    ).fetchone()
    count = (
        conn.execute(
            "SELECT COUNT(*) FROM monthly_nft_candidates WHERE user_id=?",
            ("tg_unverified_test_user",),
        ).fetchone()[0]
        if table_exists else 0
    )

if result.get("ok") or count:
    print("EXPECTED_RED: unverified client activity created an NFT candidate")
    raise SystemExit(42)

print("PASS: unverified client activity did not create an NFT candidate")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-nft-test-") as temp_dir:
    env = os.environ.copy()
    env["PYTHONPATH"] = (
        str(telegram_game_root)
        + os.pathsep
        + env.get("PYTHONPATH", "")
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

# The child has exited, so Windows can release its SQLite file handles.
raise SystemExit(exit_code)
