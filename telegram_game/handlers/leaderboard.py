from telegram.ext import MessageHandler, CommandHandler, filters
from database.sqlite_store import get_conn

async def _lb(u, c):
    with get_conn() as conn:
        rows = conn.execute("SELECT username, qc FROM users ORDER BY qc DESC LIMIT 10").fetchall()
    if not rows:
        await u.message.reply_text("Leaderboard: no players yet.")
        return
    lines = [f"{i+1}. {(r['username'] or 'player')} — {r['qc']}" for i, r in enumerate(rows)]
    await u.message.reply_text("Leaderboard (by QC):\n" + "\n".join(lines))

def wire(app):
    app.add_handler(MessageHandler(filters.Regex(r"^Leaderboard$"), _lb), group=10)
    app.add_handler(CommandHandler("leaderboard", _lb), group=10)
