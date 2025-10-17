from telegram.ext import MessageHandler, CommandHandler, filters
from utils.state import get_user, flood_guard

async def _profile(u, c):
    usr = get_user(u)
    if not flood_guard(usr):
        return
    await u.message.reply_text(
        f"Profile\nQC: {usr.qc}\nTGWT: {usr.tgwt}\nEnergy: {usr.energy}/100\nLevel: {usr.level}"
    )

def wire(app):
    app.add_handler(MessageHandler(filters.Regex(r"^Profile$"), _profile), group=10)
    app.add_handler(CommandHandler("profile", _profile), group=10)
