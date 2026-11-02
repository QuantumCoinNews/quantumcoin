import os
import subprocess
import sys
import tempfile
from pathlib import Path

telegram_game_root = Path(__file__).resolve().parents[1]
child_code = r"""
import datetime
import sqlite3
from telegram_game import dev_api

utc = datetime.timezone.utc
activation = int(datetime.datetime(2031, 12, 15, 10, 0, tzinfo=utc).timestamp())
periods = dev_api._monthly_nft_award_periods_from_activation_v1(activation)

assert len(periods) == 12, periods
assert len(set(periods)) == 12, periods
assert periods[0] == "2032-01", periods
assert periods[-1] == "2032-12", periods
assert "2031-12" not in periods, periods
assert "2033-01" not in periods, periods
assert dev_api._monthly_nft_award_periods_from_activation_v1(None) == ()

first_day = int(datetime.datetime(2031, 1, 1, tzinfo=utc).timestamp())
assert dev_api._monthly_nft_award_periods_from_activation_v1(first_day)[0] == "2031-02"

# Inject an activation value only inside this isolated test process.
dev_api._nft_verified_mainnet_activation_at_v1 = lambda: activation

cases = (
    ("tg_first_month", "2032-01", True),
    ("tg_last_month", "2032-12", True),
    ("tg_thirteenth_month", "2033-01", False),
)
with sqlite3.connect(dev_api.APP_DB) as conn:
    conn.execute(
        "CREATE TABLE monthly_nft_verified_game_sessions ("
        "session_id TEXT PRIMARY KEY, user_id TEXT NOT NULL, "
        "period TEXT NOT NULL, completed_at INTEGER NOT NULL, "
        "activity_verified INTEGER NOT NULL DEFAULT 0)"
    )
    for user_id, period, _ in cases:
        year, month = map(int, period.split("-"))
        completed_at = int(
            datetime.datetime(year, month, 10, tzinfo=utc).timestamp()
        )
        conn.execute(
            "INSERT INTO monthly_nft_verified_game_sessions "
            "(session_id, user_id, period, completed_at, activity_verified) "
            "VALUES(?,?,?,?,1)",
            ("session_" + user_id, user_id, period, completed_at),
        )

for user_id, period, should_accept in cases:
    # Test each month at its own time; future evidence must never count.
    year, month = map(int, period.split("-"))
    dev_api._now = lambda: int(datetime.datetime(year, month, 20, tzinfo=utc).timestamp())
    result = dev_api.nft_monthly_candidate_create(
        dev_api.MonthlyNFTCandidateIn(userId=user_id, period=period)
    )
    assert bool(result.get("ok")) == should_accept, (period, result)

print("PASS: first 12 full UTC months admitted; activation and month 13 excluded")
"""

with tempfile.TemporaryDirectory(prefix="qma0066-periods-") as temp_dir:
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
