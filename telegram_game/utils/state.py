import time
from datetime import date, datetime
from typing import Dict, Optional

from constants import USER_DAILY_SESSION_CAP, FLOOD_COOLDOWN_SEC
from database.sqlite_store import get_or_create_user, update_user_balances

def today_key() -> str:
    d = date.today()
    return f"{d.year:04d}{d.month:02d}{d.day:02d}"

class UserState:
    __slots__ = (
        "user_id", "username",
        "qc", "tgwt", "energy", "level", "ship", "ship_mul",
        "sector", "session_start", "session_active", "session_duration",
        "last_msg_ts", "verified", "verify_code",
        "sessions_day_key", "sessions_today"
    )
    def __init__(self, uid: int, username: Optional[str]):
        self.user_id = uid
        self.username = username or ""
        self.qc = 0.0
        self.tgwt = 0
        self.energy = 100
        self.level = 1
        self.ship = "Basic Starter Ship"
        self.ship_mul = 1.0
        self.sector = None
        self.session_start: Optional[datetime] = None
        self.session_active = False
        self.session_duration = 60
        self.last_msg_ts = 0.0
        self.verified = False
        self.verify_code = ""
        self.sessions_day_key = today_key()
        self.sessions_today = 0

USERS: Dict[int, UserState] = {}

def get_user(update) -> UserState:
    uid = update.effective_user.id
    uname = update.effective_user.username or ""
    row = get_or_create_user(uid, uname)

    if uid not in USERS:
        u = UserState(uid, uname)
        USERS[uid] = u
    else:
        u = USERS[uid]
        u.username = uname or u.username

    # sync DB -> memory
    u.qc = float(row["qc"])
    u.tgwt = int(row["tgwt"])
    u.energy = int(row["energy"])
    u.level = int(row["level"])
    u.ship = row["ship"]
    u.ship_mul = float(row["ship_mul"])
    u.verified = bool(row["verified"])

    tk = today_key()
    if u.sessions_day_key != tk:
        u.sessions_day_key = tk
        u.sessions_today = 0
    return u

def flood_guard(u: UserState) -> bool:
    now = time.time()
    if now - u.last_msg_ts < FLOOD_COOLDOWN_SEC:
        return False
    u.last_msg_ts = now
    return True

def persist_user(u: UserState) -> None:
    update_user_balances(
        u.user_id,
        qc=u.qc,
        tgwt=u.tgwt,
        energy=u.energy,
        level=u.level,
        verified=1 if u.verified else 0,
        ship=u.ship,
        ship_mul=u.ship_mul,
    )
