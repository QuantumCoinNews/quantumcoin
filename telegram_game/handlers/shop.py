from telegram.ext import MessageHandler, CommandHandler, filters

async def _shop(u, c):
    await u.message.reply_text("Shop: coming soon.")

def wire(app):
    app.add_handler(MessageHandler(filters.Regex(r"^Shop$"), _shop), group=10)
    app.add_handler(CommandHandler("shop", _shop), group=10)
