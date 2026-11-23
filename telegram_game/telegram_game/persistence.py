"""One database policy for every supported game entry point.

No database is selected from cwd, copied or merged automatically. Historical
source-local databases require an explicit operator selection before startup.
"""
import os
from pathlib import Path
import sqlite3
import sys


def database_path(environ=None, *, home=None, source_root=None, working_dir=None, profile=None):
    env=os.environ if environ is None else environ
    explicit=env.get("QC_GAME_DB","").strip()
    if explicit:
        path=Path(explicit).expanduser()
        if not path.is_absolute():
            raise ValueError("QC_GAME_DB must be an absolute SQLite file path")
        if path.is_dir():
            raise ValueError("QC_GAME_DB must name a file, not a directory")
        return str(path.resolve())
    if env.get("QC_TGWT_DB","").strip():
        raise ValueError("QC_TGWT_DB is retired; explicitly select the canonical database with an absolute QC_GAME_DB path")
    # A source-local old DB must never become an unnoticed empty replacement.
    source=Path(source_root) if source_root is not None else Path(__file__).resolve().parents[1]
    cwd=Path(working_dir) if working_dir is not None else Path.cwd()
    directories={source,source/"telegram_game",source.parent,cwd}
    legacy=[d/name for d in directories for name in ("dev_api_state.db","qc_tgwt.sqlite3","qc_tgwt.db") if (d/name).is_file()]
    if legacy:
        raise ValueError("Legacy game database detected. Back it up and set QC_GAME_DB to its absolute path; automatic migration is disabled")
    if profile is None:
        from .testnet_connection_v1 import telegram_game_chain_mode_v1
        profile=telegram_game_chain_mode_v1()
    if profile not in {"local","testnet","mainnet"}:
        raise ValueError("Invalid game database network profile")
    configured=env.get("QC_GAME_DATA_DIR","").strip()
    user_home=Path(home) if home is not None else Path.home()
    if configured:
        base=Path(configured).expanduser()
    elif sys.platform=="win32":
        base=Path(env.get("LOCALAPPDATA") or user_home/"AppData"/"Local")/"QuantumCoin"/"telegram-game"
    else:
        base=Path(env.get("XDG_DATA_HOME") or user_home/".local"/"share")/"quantumcoin"/"telegram-game"
    if not base.is_absolute():
        raise ValueError("Game data directory must be absolute")
    return str((base/profile/"state.sqlite3").resolve())


def connect_database(path):
    target=Path(path)
    if not target.is_absolute():
        raise ValueError("Canonical database path must be absolute")
    target.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
    conn=sqlite3.connect(str(target),timeout=5,check_same_thread=False)
    try:
        conn.execute("PRAGMA journal_mode=WAL")
        conn.execute("PRAGMA synchronous=NORMAL")
        conn.execute("PRAGMA busy_timeout=5000")
        return conn
    except BaseException:
        conn.close()
        raise
