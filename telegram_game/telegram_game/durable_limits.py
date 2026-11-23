"""Atomic fixed-window limits shared across workers, bounded persisted state."""
import hashlib
from contextlib import closing

MAX_ROWS = 8192


def ensure_schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS request_limits_v1 (
        scope TEXT NOT NULL, identity_hash TEXT NOT NULL, bucket INTEGER NOT NULL,
        count INTEGER NOT NULL, expires_at INTEGER NOT NULL,
        PRIMARY KEY(scope,identity_hash,bucket))""")
    conn.execute("CREATE INDEX IF NOT EXISTS request_limit_expiry_v1 ON request_limits_v1(expires_at)")


def admit(db, dimensions, now):
    """Each dimension is (scope, trusted identity, limit, window seconds)."""
    with closing(db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        ensure_schema(conn)
        conn.execute("DELETE FROM request_limits_v1 WHERE expires_at<=?", (now,))
        size = conn.execute("SELECT COUNT(*) FROM request_limits_v1").fetchone()[0]
        pending = []
        for scope, identity, limit, window in dimensions:
            digest = hashlib.sha256(str(identity).encode()).hexdigest()
            bucket = now // window
            row = conn.execute("SELECT count FROM request_limits_v1 WHERE scope=? AND identity_hash=? AND bucket=?", (scope,digest,bucket)).fetchone()
            if row and row[0] >= limit:
                return False
            if not row:
                size += 1
                if size > MAX_ROWS:
                    return False
            pending.append((scope,digest,bucket,(row[0] if row else 0)+1,(bucket+1)*window))
        conn.executemany("INSERT OR REPLACE INTO request_limits_v1 VALUES (?,?,?,?,?)", pending)
        return True
