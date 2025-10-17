# utils/tgwt_pool.py — ASCII-only
from constants import TGWT_DAILY_POOL, USER_DAILY_TGWT_CAP, SOCIAL_TASKS
from utils.state import today_key
from database.sqlite_store import (
    get_or_init_daily_pool, dec_daily_pool, insert_tgwt_claim
)

# in-memory per-day, per-user per-task caps (safety net; DB has unique too)
_CLAIMED = {}  # key: (uid, task, day_key) -> True
_USER_TODAY = {}  # key: (uid, day_key) -> int

def try_claim_tgwt(uid: int, task_name: str) -> tuple[bool, int, str]:
    day = today_key()
    key = (uid, task_name, day)
    if task_name not in SOCIAL_TASKS:
        return False, 0, "Unknown task."

    # in-memory duplicate guard
    if _CLAIMED.get(key):
        return False, 0, "Already claimed today for this task."

    # per-user daily total cap
    used = _USER_TODAY.get((uid, day), 0)
    if used >= USER_DAILY_TGWT_CAP:
        return False, 0, "Daily TGWT cap reached."

    # ensure pool exists
    pool = get_or_init_daily_pool(day, TGWT_DAILY_POOL)
    award = int(SOCIAL_TASKS[task_name])
    award = min(award, USER_DAILY_TGWT_CAP - used)
    if pool <= 0 or award <= 0:
        return False, 0, "Daily pool exhausted."

    # commit in DB
    dec_daily_pool(day, award)
    insert_tgwt_claim(uid, task_name, award, day)

    # mark memory stats
    _CLAIMED[key] = True
    _USER_TODAY[(uid, day)] = used + award

    return True, award, f"+{award} TGWT"
