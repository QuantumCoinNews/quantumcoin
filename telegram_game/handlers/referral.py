from telegram.ext import MessageHandler, CommandHandler, filters
from utils.state import get_user, flood_guard

async def _ref(u, c):
    me = await c.bot.get_me()
    usr = get_user(u)
    if not flood_guard(usr):
        return
    link = f"https://t.me/{me.username}?start=ref_{usr.user_id}"
    await u.message.reply_text("Invite friends with your link:\n" + link)

def wire(app):
    app.add_handler(MessageHandler(filters.Regex(r"^Referral$"), _ref), group=10)
    app.add_handler(CommandHandler("referral", _ref), group=10)
