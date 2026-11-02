import os
import subprocess
import sys
import tempfile
from pathlib import Path

telegram_game_root = Path(__file__).resolve().parents[1]
child_code = r"""
import sqlite3
from telegram_game import dev_api

result = dev_api.session_complete(
    dev_api.SessionCompleteIn(
        playerId="tg_unverified_session_test",
        qcEarned=1000,
        xpEarned=1000,
        durationSec=9999,
    )
)

with sqlite3.connect(dev_api.APP_DB) as conn:
    row = conn.execute(
        "SELECT sessions, total_qc, total_xp FROM players WHERE player_id=?",
        ("tg_unverified_session_test",),
    ).fetchone()

if result.get("ok") or (row and (row[0] or row[1] or row[2])):
    print("EXPECTED_RED: unverified session changed player counters")
    raise SystemExit(42)

print("PASS: unverified session did not change player counters")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-session-test-") as temp_dir:
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
