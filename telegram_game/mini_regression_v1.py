import json
import urllib.request
import urllib.error
import sqlite3
from pathlib import Path

BASE = "http://127.0.0.1:8080"
QC_BASE = "http://127.0.0.1:8082"
USER_ID = "dev_user"

def get_json(url, timeout=20):
    try:
        with urllib.request.urlopen(url, timeout=timeout) as r:
            raw = r.read().decode("utf-8", errors="replace")
            return True, json.loads(raw)
    except Exception as e:
        return False, {"error": str(e)}

def check(name, ok, detail=""):
    mark = "PASS" if ok else "FAIL"
    print(f"[{mark}] {name}" + (f" -> {detail}" if detail else ""))
    return ok

print("\n===== MINI REGRESSION V1 =====")

all_ok = True

ok, data = get_json(f"{QC_BASE}/api/health", timeout=10)
all_ok &= check("QC node health 8082", ok and data.get("ok") is True, data)

ok, data = get_json(f"{BASE}/api/wallet?playerId={USER_ID}", timeout=10)
all_ok &= check("Telegram backend wallet", ok and data.get("playerId") == USER_ID, {
    "real_qc_ok": data.get("real_qc_ok"),
    "qc_balance": data.get("qc_balance"),
    "tgwt_balance": data.get("tgwt_balance"),
})

ok, data = get_json(f"{BASE}/api/v1/tgwt/ledger?user_id={USER_ID}&limit=10", timeout=10)
all_ok &= check("TGWT ledger", ok and data.get("ok") is True, {
    "distributed": (data.get("pool") or {}).get("distributed"),
    "earned_u": (data.get("balances") or {}).get("earned_u"),
    "pending_u": (data.get("balances") or {}).get("pending_u"),
})

ok, data = get_json(f"{BASE}/api/social/links", timeout=10)
links = data.get("links") or {}
all_ok &= check("Social links", ok and bool(links.get("x")) and bool(links.get("youtube")), links)

ok, data = get_json(f"{BASE}/api/watch", timeout=10)
all_ok &= check("Watch config", ok and data.get("enabled") is True, data)

ok, data = get_json(f"{BASE}/api/v1/withdraw/list?user_id={USER_ID}&limit=10", timeout=10)
items = data.get("items") or []
has_admin_review = any((x.get("status") == "ADMIN_REVIEW") for x in items)
all_ok &= check("Withdraw list / ADMIN_REVIEW present", ok and data.get("ok") is True and has_admin_review, {
    "count": len(items),
    "has_admin_review": has_admin_review,
})

print("\n===== FILE MARKERS =====")
html = Path("telegram_game/dev_ui/space_miner_prototype.html").read_text(encoding="utf-8")
api = Path("telegram_game/dev_api.py").read_text(encoding="utf-8")

markers = [
    ("TGWT_API_PANEL_MINIMUM_V1", html),
    ("TGWT_HUD_SYNC_V1", html),
    ("SOCIAL_WATCH_CLAIM_UI_V1", html),
    ("WATCH_REWARDED_AD_POLICY_UI_V1", html),
    ("WATCH_START_SAFE_BIND_V1", html),
    ("MOBILE_X_CLOSE_XP_SLOW_V1", html),
    ("WATCH_CLOSE_INSIDE_CARD_V1", html),
    ("TGWT_WITHDRAW_ADMIN_REVIEW_V1", api),
    ("WATCH_REWARDED_AD_POLICY_V1", api),
    ("WATCH_DEV_TEST_REWARD", api),
]

for marker, src in markers:
    all_ok &= check(f"Marker {marker}", marker in src)

print("\n===== DATABASE CHECK =====")
try:
    conn = sqlite3.connect("dev_api_state.db")
    conn.row_factory = sqlite3.Row

    pool = conn.execute("SELECT total_u, distributed_u FROM tgwt_pool WHERE id=1").fetchone()
    all_ok &= check("TGWT pool row", pool is not None, dict(pool) if pool else "")

    events = conn.execute("SELECT kind, COUNT(*) c FROM tgwt_events GROUP BY kind ORDER BY kind").fetchall()
    print("TGWT event counts:")
    for e in events:
        print(f"  - {e['kind']}: {e['c']}")

    conn.close()
except Exception as e:
    all_ok &= check("SQLite DB readable", False, str(e))

print("\n===== RESULT =====")
if all_ok:
    print("REGRESSION PASS")
else:
    print("REGRESSION FAIL")
    raise SystemExit(1)
