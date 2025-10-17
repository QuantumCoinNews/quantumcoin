from telegram.ext import MessageHandler, CommandHandler, filters
from utils.state import get_user, flood_guard

async def _wallet(u, c):
    usr = get_user(u)
    if not flood_guard(usr):
        return
    await u.message.reply_text("Wallet: not linked yet.")

def wire(app):
    app.add_handler(MessageHandler(filters.Regex(r"^Wallet$"), _wallet), group=10)
    app.add_handler(CommandHandler("wallet", _wallet), group=10)
