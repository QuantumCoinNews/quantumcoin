import os, json, time, hmac, hashlib, urllib.parse, sys

token = (
    os.getenv("TELEGRAM_BOT_TOKEN")
    or os.getenv("QC_TELEGRAM_BOT_TOKEN")
    or os.getenv("TELEGRAM_GAME_BOT_TOKEN")
    or ""
).strip()

if not token:
    print("ERROR: telegram bot token not found in this shell environment", file=sys.stderr)
    sys.exit(2)

data = {
    "query_id": "test_query_item5",
    "user": json.dumps({
        "id": 123456,
        "first_name": "Test",
        "username": "test_user",
        "language_code": "tr"
    }, separators=(",", ":"), ensure_ascii=False),
    "auth_date": str(int(time.time())),
}

check_string = "\n".join(f"{k}={data[k]}" for k in sorted(data.keys()))

secret_key = hmac.new(
    b"WebAppData",
    token.encode("utf-8"),
    hashlib.sha256
).digest()

data["hash"] = hmac.new(
    secret_key,
    check_string.encode("utf-8"),
    hashlib.sha256
).hexdigest()

print(urllib.parse.urlencode(data))
