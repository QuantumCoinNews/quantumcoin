"""Exact game accounting. Legacy REAL columns are retained only for audit.

QC uses the canonical Go denomination: 1 QC = 1,000,000 atoms. Signed
transaction wire integers retain their protocol-specific unit at the node.
No historical client-reported float is promoted to a financial entitlement.
"""
import re
from decimal import Decimal, InvalidOperation

ATOMS_PER_QC = 1_000_000
MAX_INTEGER = (1 << 63) - 1


def integer(value):
    if type(value) is not int or not 0 <= value <= MAX_INTEGER:
        raise ValueError("nonnegative_int64_required")
    return value


def parse_qc(value):
    if type(value) is not str or not re.fullmatch(r"[0-9]{1,19}(?:\.[0-9]{1,6})?", value):
        raise ValueError("qc_decimal_string_required")
    whole, _, fraction = value.partition(".")
    return integer(int(whole) * ATOMS_PER_QC + int(fraction.ljust(6, "0") or "0"))


def format_qc(atoms):
    whole, fraction = divmod(integer(atoms), ATOMS_PER_QC)
    return str(whole) if not fraction else f"{whole}.{fraction:06d}".rstrip("0")


def decimal_atoms(value):
    """Decimal tuple conversion never rounds in the ambient Decimal context."""
    if type(value) not in (str, int, Decimal):
        raise ValueError("decimal_amount_required")
    try:
        number = Decimal(value)
    except InvalidOperation:
        raise ValueError("invalid_decimal_amount") from None
    if not number.is_finite():
        raise ValueError("finite_amount_required")
    sign, digits, exponent = number.as_tuple()
    if sign or exponent < -6 or exponent > 18 or len(digits) > 25:
        raise ValueError("invalid_micro_unit_amount")
    coefficient = int("".join(map(str, digits)))
    return integer(coefficient * 10 ** (exponent + 6))


def ensure_schema(conn):
    for table, columns in {
        "players": ("qc_atoms", "xp_units"),
        "reward_records": ("amount_atoms", "xp_units", "block_reward_atoms"),
    }.items():
        existing = {row[1] for row in conn.execute(f"PRAGMA table_info({table})")}
        for column in columns:
            if column not in existing:
                conn.execute(f"ALTER TABLE {table} ADD COLUMN {column} INTEGER NOT NULL DEFAULT 0 "
                             f"CHECK(typeof({column})='integer' AND {column}>=0)")
        if "numeric_version" not in existing:
            # Existing rows stay version 0; new rows explicitly select version 1.
            conn.execute(f"ALTER TABLE {table} ADD COLUMN numeric_version INTEGER NOT NULL DEFAULT 0")
    conn.execute("CREATE INDEX IF NOT EXISTS idx_players_qc_atoms ON players(qc_atoms)")


def add_xp(conn, player_id, xp):
    xp = integer(xp)
    row = conn.execute("SELECT xp_units FROM players WHERE player_id=?", (player_id,)).fetchone()
    if row is None:
        raise ValueError("player_required")
    integer(integer(row[0]) + xp)  # Prevent SQLite promoting int64 overflow to REAL.
    conn.execute("UPDATE players SET xp_units=xp_units+?,sessions=sessions+1 WHERE player_id=?", (xp, player_id))
