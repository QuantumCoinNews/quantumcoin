// Configuration inspection only; importing this module never sends a test message.
export function telegramConfigurationPresent(env:NodeJS.ProcessEnv=process.env):boolean{return Boolean(env.TELEGRAM_BOT_TOKEN && env.TELEGRAM_CHAT_ID);}
