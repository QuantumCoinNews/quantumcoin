"""Inventory the actual Telegram ASGI application with an isolated database."""

import argparse
import ast
import inspect
import json
import os
from pathlib import Path
import sys
import tempfile
import textwrap


ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "docs" / "telegram-endpoints.json"
GATED = {
    "/api/admin/nft/monthly/controlled-mint",
    "/api/admin/nft/monthly/final-readiness",
    "/api/admin/nft/monthly/publish-draw-commitment",
    "/api/admin/nft/monthly/draw-from-beacon",
    "/api/social/x/start", "/api/social/x/complete", "/oauth/x/callback",
    "/api/mine/work", "/api/mine/work/submit", "/api/mine/work/status",
}
COMPATIBILITY_READS = {
    "/api/withdraw/status", "/api/settlement/queue", "/api/settlement/pending",
    "/api/settlement/confirmed", "/api/settlement/summary",
}
DISABLED = {
    "/api/session/complete": "retired",
    "/api/admin/nft/monthly/select-winner": "retired",
    "/api/admin/tgwt/withdraw/confirm": "unavailable",
}


def response_evidence(endpoint):
    function = ast.parse(textwrap.dedent(inspect.getsource(endpoint))).body[-1]
    keys, statuses = set(), set()
    returns = [node for node in ast.walk(function) if isinstance(node, ast.Return)]
    retired = bool(returns)
    for node in returns:
        value = node.value
        status = None
        if isinstance(value, ast.Call):
            kwargs = {kw.arg: kw.value for kw in value.keywords}
            code = kwargs.get("status_code")
            if isinstance(code, ast.Constant) and isinstance(code.value, int):
                status = code.value
                statuses.add(status)
            value = kwargs.get("content")
        if status not in (410, 501):
            retired = False
        if isinstance(value, ast.Dict):
            keys.update(key.value for key in value.keys
                        if isinstance(key, ast.Constant) and isinstance(key.value, str))
    return sorted(keys), sorted(statuses), retired


def catalog(app, player_auth):
    from starlette.routing import Mount

    spec = app.openapi()
    inventory = []
    for route in app.routes:
        if isinstance(route, Mount):
            inventory.append({"owner": "telegram_game", "method": "GET/HEAD", "path": route.path + "/{file}",
                              "auth": "public", "lifecycle": "active", "request": {"path": "file"},
                              "response": {"content": "static asset"}, "handler": "StaticFiles"})
            continue
        if not getattr(route, "methods", None):
            continue
        path = route.path
        keys, statuses, retired = response_evidence(route.endpoint)
        for method in sorted(route.methods):
            operation = spec.get("paths", {}).get(path, {}).get(method.lower(), {})
            auth = "public"
            if (method, path) == ("POST", "/api/telegram/identity/verify"):
                auth = "telegram-initData"
            elif path == "/oauth/x/callback":
                auth = "oauth-state-and-code"
            elif player_auth.operator_path(path):
                auth = "operator-token"
            elif path.startswith("/api/") and (method, path) not in player_auth.PUBLIC:
                auth = "telegram-session-bearer"

            lifecycle = "active"
            if retired:
                lifecycle = "retired" if 410 in statuses else "unavailable"
            elif path in DISABLED:
                lifecycle = DISABLED[path]
            elif path in GATED:
                lifecycle = "configuration-gated"
            elif path in COMPATIBILITY_READS:
                lifecycle = "compatibility-read"

            parameters = operation.get("parameters", [])
            if path == "/oauth/x/callback":
                parameters = [{"name": name, "in": "query", "schema": {"type": "string"}}
                              for name in ("state", "code", "error")]
            responses = operation.get("responses", {})
            success = responses.get("200", {}).get("content", {}).get("application/json", {}).get("schema")
            inventory.append({
                "owner": "telegram_game", "method": method, "path": path,
                "auth": auth, "lifecycle": lifecycle,
                "handler": f"{route.endpoint.__module__}.{route.endpoint.__name__}",
                "request": {"parameters": parameters, "body": operation.get("requestBody")},
                "security": operation.get("security", []),
                "response": {"openapi": responses, "literalKeys": keys, "explicitStatusCodes": statuses,
                             "typedSuccessSchema": bool(success),
                             "schemaKind": ("node-owned-json-document" if route.endpoint.__name__ in
                                            {"qc_node_health", "qc_address_balance"} else "named-wire-contract")},
            })
    identities = [(row["path"], row["method"]) for row in inventory]
    if len(identities) != len(set(identities)):
        raise ValueError("duplicate Telegram endpoint registrations")
    return {
        "schemaVersion": 2,
        "entrypoint": "telegram_game.runtime:app",
        "components": spec.get("components", {}),
        "endpoints": sorted(inventory, key=lambda row: (row["path"], row["method"])),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="qc-endpoint-catalog-") as temp:
        os.environ["QC_GAME_DB"] = str(Path(temp) / "catalog.sqlite3")
        os.environ["QC_GAME_LOAD_DOTENV"] = "0"
        sys.path.insert(0, str(ROOT / "telegram_game"))
        from telegram_game.runtime import app
        from telegram_game import player_auth

        result = catalog(app, player_auth)
    rendered = json.dumps(result, indent=2, sort_keys=True, ensure_ascii=True) + "\n"
    if args.check:
        if OUTPUT.read_text(encoding="utf-8") != rendered:
            parser.exit(1, "Telegram endpoint inventory differs from the canonical app.\n")
    else:
        OUTPUT.write_text(rendered, encoding="utf-8", newline="\n")
    print(f"Verified {len(result['endpoints'])} Telegram endpoint entries")


if __name__ == "__main__":
    main()
