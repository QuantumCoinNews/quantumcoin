"""Compatibility entry point. The canonical backend is telegram_game.runtime."""
from .runtime import app

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("telegram_game.runtime:app", host="127.0.0.1", port=8080,
                proxy_headers=False, reload=False)
