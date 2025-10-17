# web/app.py — Game UI (Canvas) + API (FastAPI)
from fastapi import FastAPI, Request, HTTPException
from fastapi.responses import HTMLResponse, JSONResponse, PlainTextResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel
import os, sys
from datetime import datetime, date

# make project root importable
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if BASE_DIR not in sys.path:
    sys.path.insert(0, BASE_DIR)

# storage / constants
from database.sqlite_store import (
    init_db, get_or_create_user, update_user_balances,
    session_start, session_finish, get_conn,
    get_or_init_daily_pool, dec_daily_pool, insert_tgwt_claim
)
from constants import SECTORS, SOCIAL_TASKS, TGWT_DAILY_POOL as _DAILY_POOL

init_db()  # ensure tables exist at import

app = FastAPI(title="Quantum Coin Web")

# static + templates
THIS_DIR = os.path.dirname(__file__)
STATIC_DIR = os.path.join(THIS_DIR, "static")
TPL_DIR = os.path.join(THIS_DIR, "templates")
os.makedirs(STATIC_DIR, exist_ok=True)
os.makedirs(TPL_DIR, exist_ok=True)

app.mount("/static", StaticFiles(directory=STATIC_DIR), name="static")
templates = Jinja2Templates(directory=TPL_DIR)

# ---- helpers ----
def _now_iso() -> str:
    return datetime.utcnow().isoformat()

def _today_key() -> str:
    return date.today().isoformat()

def _ensure_demo_user() -> dict:
    # for local web demo, use fixed user id=1
    u = get_or_create_user(1, "local_web")
    return u

# ---- models ----
class StartMiningIn(BaseModel):
    sector: str

class FinishMiningIn(BaseModel):
    pass

class BuyShipIn(BaseModel):
    ship_id: int

class ClaimSocialIn(BaseModel):
    task: str

class LinkWalletIn(BaseModel):
    bnb_address: str

class TransferQCIn(BaseModel):
    to_user_id: int
    amount: float

# ---- UI page ----
@app.get("/", response_class=HTMLResponse)
def game_page(request: Request):
    # one template that loads /static/app.js and /static/styles.css
    return templates.TemplateResponse("game.html", {"request": request})

# ---- boot info ----
@app.get("/api/boot")
def api_boot():
    u = _ensure_demo_user()
    # demo ship catalog (5 ships, price and multipliers)
    ships = [
        {"id": 1, "name": "Scout I",     "mul": 1.00, "price_qc": 100, "img": "/static/img/ship1.svg"},
        {"id": 2, "name": "Miner Mk-II", "mul": 1.10, "price_qc": 200, "img": "/static/img/ship2.svg"},
        {"id": 3, "name": "Miner Mk-III","mul": 1.20, "price_qc": 300, "img": "/static/img/ship3.svg"},
        {"id": 4, "name": "Prospector",  "mul": 1.35, "price_qc": 400, "img": "/static/img/ship4.svg"},
        {"id": 5, "name": "Quantum Rig", "mul": 1.50, "price_qc": 500, "img": "/static/img/ship5.svg"},
    ]
    # sectors list
    sectors = [{"name": k, **v} for k, v in SECTORS.items()]
    return {
        "user": {
            "id": u["id"], "username": u["username"], "qc": u["qc"], "tgwt": u["tgwt"],
            "energy": u["energy"], "level": u["level"], "ship": u["ship"], "ship_mul": u["ship_mul"],
        },
        "sectors": sectors,
        "ships": ships,
        "social_tasks": [{"task": k, "amount": v} for k, v in SOCIAL_TASKS.items()],
        "bnb": {"chainId": "0x38", "name": "BNB Smart Chain"},  # mainnet (devde sadece connect)
    }

# ---- mining ----
@app.post("/api/start_mining")
def api_start_mining(payload: StartMiningIn):
    u = _ensure_demo_user()
    sec = payload.sector
    if sec not in SECTORS:
        raise HTTPException(400, "invalid sector")
    cfg = SECTORS[sec]
    if u["energy"] < cfg["energy"]:
        raise HTTPException(400, "not enough energy")
    # consume energy and start session
    new_energy = max(0, int(u["energy"]) - int(cfg["energy"]))
    update_user_balances(u["id"], energy=new_energy)
    session_start(u["id"], sec, _now_iso())
    return {"ok": True, "duration": int(cfg["duration"]), "energy": new_energy}

@app.post("/api/finish_mining")
def api_finish_mining(payload: FinishMiningIn):
    u = _ensure_demo_user()
    # compute reward based on user's current ship multiplier and sector base
    with get_conn() as c:
        r = c.execute("SELECT sector, start_ts FROM sessions WHERE user_id=? AND end_ts IS NULL ORDER BY id DESC LIMIT 1", (u["id"],)).fetchone()
    if not r:
        raise HTTPException(400, "no active session")
    sec = r["sector"]
    cfg = SECTORS.get(sec)
    if not cfg:
        raise HTTPException(400, "unknown sector")
    reward = round(cfg["base_qc"] * cfg["sector_mul"] * float(u["ship_mul"]), 3)
    new_qc = round(float(u["qc"]) + reward, 3)
    update_user_balances(u["id"], qc=new_qc)
    session_finish(u["id"], sec, _now_iso(), reward)
    # refresh user
    u2 = get_or_create_user(u["id"], u["username"])
    return {"ok": True, "reward_qc": reward, "qc": u2["qc"], "energy": u2["energy"]}

# ---- shop ----
@app.post("/api/buy_ship")
def api_buy_ship(payload: BuyShipIn):
    u = _ensure_demo_user()
    catalog = {
        1: ("Scout I",     1.00, 100),
        2: ("Miner Mk-II", 1.10, 200),
        3: ("Miner Mk-III",1.20, 300),
        4: ("Prospector",  1.35, 400),
        5: ("Quantum Rig", 1.50, 500),
    }
    if payload.ship_id not in catalog:
        raise HTTPException(400, "unknown ship id")
    name, mul, price = catalog[payload.ship_id]
    if float(u["qc"]) < price:
        raise HTTPException(400, "not enough QC")
    new_qc = round(float(u["qc"]) - price, 3)
    update_user_balances(u["id"], qc=new_qc, ship=name, ship_mul=mul)
    return {"ok": True, "qc": new_qc, "ship": name, "ship_mul": mul}

# ---- social / TGWT ----
@app.post("/api/claim_social")
def api_claim_social(payload: ClaimSocialIn):
    u = _ensure_demo_user()
    task = payload.task
    if task not in SOCIAL_TASKS:
        raise HTTPException(400, "unknown task")
    amount = int(SOCIAL_TASKS[task])
    day = _today_key()
    pool_left = get_or_init_daily_pool(day, int(_DAILY_POOL))
    if pool_left < amount:
        raise HTTPException(400, "daily pool depleted")
    # unique per user-task-day
    try:
        insert_tgwt_claim(u["id"], task, amount, day)
    except Exception:
        raise HTTPException(400, "already claimed today")
    dec_daily_pool(day, amount)
    new_tgwt = int(u["tgwt"]) + amount
    update_user_balances(u["id"], tgwt=new_tgwt)
    return {"ok": True, "tgwt": new_tgwt}

# ---- wallet ----
@app.get("/api/wallet")
def api_wallet():
    u = _ensure_demo_user()
    # ensure internal QC wallet (simple deterministic for demo)
    with get_conn() as c:
        c.execute("ALTER TABLE users ADD COLUMN qc_addr TEXT",) if "qc_addr" not in [x["name"] for x in c.execute("PRAGMA table_info(users)")] else None
        c.execute("ALTER TABLE users ADD COLUMN bnb_addr TEXT",) if "bnb_addr" not in [x["name"] for x in c.execute("PRAGMA table_info(users)")] else None
        r = c.execute("SELECT qc_addr, bnb_addr FROM users WHERE id=?", (u["id"],)).fetchone()
    qc_addr = r["qc_addr"] if r and r["qc_addr"] else f"QC-{u['id']:06d}"
    # store qc_addr if empty
    with get_conn() as c:
        c.execute("UPDATE users SET qc_addr=?, updated_at=? WHERE id=?", (qc_addr, _now_iso(), u["id"]))
    return {"qc_addr": qc_addr, "bnb_addr": r["bnb_addr"] if r else None}

@app.post("/api/link_wallet")
def api_link_wallet(payload: LinkWalletIn):
    u = _ensure_demo_user()
    addr = payload.bnb_address.strip()
    if not addr.startswith("0x") or len(addr) != 42:
        raise HTTPException(400, "invalid evm address")
    with get_conn() as c:
        c.execute("UPDATE users SET bnb_addr=?, updated_at=? WHERE id=?", (addr, _now_iso(), u["id"]))
    return {"ok": True, "bnb_addr": addr}

@app.post("/api/transfer_qc")
def api_transfer_qc(payload: TransferQCIn):
    u = _ensure_demo_user()
    amt = float(payload.amount)
    if amt <= 0:
        raise HTTPException(400, "amount must be >0")
    if float(u["qc"]) < amt:
        raise HTTPException(400, "insufficient qc")
    # receiver must exist
    _ = get_or_create_user(payload.to_user_id, f"player_{payload.to_user_id}")
    with get_conn() as c:
        c.execute("UPDATE users SET qc=qc-?, updated_at=? WHERE id=?", (amt, _now_iso(), u["id"]))
        c.execute("UPDATE users SET qc=qc+?, updated_at=? WHERE id=?", (amt, _now_iso(), payload.to_user_id))
        c.execute("INSERT INTO tx_log(user_id, kind, amount, meta, created_at) VALUES(?,?,?,?,?)",
                  (u["id"], "qc_send", -amt, str(payload.to_user_id), _now_iso()))
        c.execute("INSERT INTO tx_log(user_id, kind, amount, meta, created_at) VALUES(?,?,?,?,?)",
                  (payload.to_user_id, "qc_recv", amt, str(u["id"]), _now_iso()))
    u2 = get_or_create_user(u["id"], u["username"])
    return {"ok": True, "qc": u2["qc"]}

# ---- debug/plain (for troubleshooting) ----
@app.get("/plain", response_class=PlainTextResponse)
def plain():
    return "OK: web server is responding"

@app.get("/health")
def health():
    return {"ok": True}

@app.get("/debug")
def debug():
    try:
        files = sorted(os.listdir(TPL_DIR))
    except Exception as e:
        files = [f"err: {e!r}"]
    return {"tpl_dir": TPL_DIR, "files": files}

# --- direct run ---
if __name__ == "__main__":
    import uvicorn
    uvicorn.run("web.app:app", host="0.0.0.0", port=8080, reload=True)
