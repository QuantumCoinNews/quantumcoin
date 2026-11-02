import os
import subprocess
import sys
import tempfile
from pathlib import Path

telegram_game_root = Path(__file__).resolve().parents[1]
coordinator = r'''
import hashlib
import hmac
import json
import os
import sqlite3
import subprocess
import sys
import time
import urllib.parse
from telegram_game import dev_api

t0 = int(time.time())
token = "qma0066-multiprocess-test-token"
values = {
    "auth_date": str(t0),
    "user": json.dumps({"id": 123456}, separators=(",", ":")),
}
check = "\n".join(f"{key}={values[key]}" for key in sorted(values))
secret = hmac.new(b"WebAppData", token.encode(), hashlib.sha256).digest()
values["hash"] = hmac.new(secret, check.encode(), hashlib.sha256).hexdigest()
init_data = urllib.parse.urlencode(values)

with sqlite3.connect(dev_api.APP_DB) as conn:
    dev_api._touch_player(conn, "tg_123456", None)
    dev_api._ensure_qc_address_column(conn)
    conn.execute(
        "UPDATE players SET qc_address=? WHERE player_id=?",
        ("QC_NFT_MULTIPROCESS_TEST", "tg_123456"),
    )

dev_api._now = lambda: t0
started = dev_api.nft_monthly_session_start(
    dev_api.MonthlyNFTSessionStartIn(
        playerId="tg_123456", initData=init_data
    )
)
assert started.get("ok"), started
session_id = started["sessionId"]

dev_api._now = lambda: t0 + 1
game = dev_api.mine_start(
    dev_api.MineStartIn(
        playerId="tg_123456",
        siteId=1,
        initData=init_data,
        nftSessionId=session_id,
    )
)
assert game.get("ok"), game

worker = r"""
import json
import os
from telegram_game import dev_api

ts = int(os.environ["QMA_NFT_TEST_T0"]) + 61
dev_api._now = lambda: ts
result = dev_api.nft_monthly_session_complete(
    dev_api.MonthlyNFTSessionCompleteIn(
        playerId="tg_123456",
        initData=os.environ["QMA_NFT_TEST_INIT_DATA"],
        sessionId=os.environ["QMA_NFT_TEST_SESSION_ID"],
    )
)
print(json.dumps(result))
"""
env = os.environ.copy()
env["QMA_NFT_TEST_T0"] = str(t0)
env["QMA_NFT_TEST_INIT_DATA"] = init_data
env["QMA_NFT_TEST_SESSION_ID"] = session_id

processes = [
    subprocess.Popen(
        [sys.executable, "-c", worker],
        cwd=os.getcwd(),
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    for _ in range(8)
]

results = []
for process in processes:
    stdout, stderr = process.communicate(timeout=30)
    assert process.returncode == 0, (process.returncode, stderr)
    results.append(json.loads(stdout.strip().splitlines()[-1]))

successes = sum(result.get("ok") is True for result in results)
with sqlite3.connect(dev_api.APP_DB) as conn:
    row = conn.execute(
        "SELECT status FROM monthly_nft_game_sessions WHERE session_id=?",
        (session_id,),
    ).fetchone()
    evidence = conn.execute(
        "SELECT COUNT(*), MIN(activity_verified), MAX(activity_verified) "
        "FROM monthly_nft_verified_game_sessions WHERE session_id=?",
        (session_id,),
    ).fetchone()

assert successes == 1, results
assert row == ("COMPLETED",), row
assert evidence == (1, 1, 1), evidence
print("PASS: 8 processes completed one signed game session exactly once")
'''

with tempfile.TemporaryDirectory(prefix="qma0066-multiprocess-") as temp_dir:
    env = os.environ.copy()
    env["TELEGRAM_BOT_TOKEN"] = "qma0066-multiprocess-test-token"
    env["PYTHONPATH"] = (
        str(telegram_game_root) + os.pathsep + env.get("PYTHONPATH", "")
    )
    completed = subprocess.run(
        [sys.executable, "-c", coordinator],
        cwd=temp_dir,
        env=env,
        text=True,
        capture_output=True,
        timeout=60,
    )
    print(completed.stdout, end="")
    print(completed.stderr, end="", file=sys.stderr)
    exit_code = completed.returncode

raise SystemExit(exit_code)
