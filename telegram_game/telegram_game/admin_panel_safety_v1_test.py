# QC_ADMIN_PANEL_AUTH_PAGINATION_V1_TEST

import os
import sys
from pathlib import Path
from contextlib import contextmanager

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.admin_panel_safety_v1 import (
    QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
    QC_ADMIN_HEADER_PRIMARY,
    qc_admin_auth_review_v1,
    qc_admin_endpoint_registry_v1,
    qc_admin_lock_summary_v1,
    qc_admin_pagination_v1,
)

@contextmanager
def clean_env():
    keys = [
        "TELEGRAM_GAME_ADMIN_TOKEN",
        "QC_TG_ADMIN_TOKEN",
        "QC_ADMIN_TOKEN",
        "ADMIN_TOKEN",
        "TELEGRAM_GAME_ADMIN_AUTH_REQUIRED",
        "QC_ADMIN_AUTH_REQUIRED",
        "TELEGRAM_GAME_CHAIN_MODE",
        "TELEGRAM_GAME_NETWORK",
        "QC_CHAIN_MODE",
        "QC_NETWORK",
        "QC_TESTNET",
    ]
    old = {k: os.environ.get(k) for k in keys}
    try:
        for k in keys:
            os.environ.pop(k, None)
        yield
    finally:
        for k, v in old.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v

def test_admin_auth_local_without_token_allows_dev_fallback():
    with clean_env():
        r = qc_admin_auth_review_v1({}, "/api/security/rate-limit/status")
        assert r["ok"] is True
        assert r["devFallbackAllowed"] is True

def test_admin_path_requires_configured_token():
    with clean_env():
        r = qc_admin_auth_review_v1({}, "/api/admin/nft/monthly/candidates")
        assert r["ok"] is False
        assert r["error"] == "admin_token_not_configured"

def test_admin_token_header_accepts():
    with clean_env():
        os.environ["TELEGRAM_GAME_ADMIN_TOKEN"] = "secret"
        r = qc_admin_auth_review_v1({QC_ADMIN_HEADER_PRIMARY: "secret"}, "/api/admin/nft/monthly/candidates")
        assert r["ok"] is True
        assert r["adminAuthRequired"] is True

def test_admin_token_bearer_accepts():
    with clean_env():
        os.environ["QC_ADMIN_TOKEN"] = "secret2"
        r = qc_admin_auth_review_v1({"Authorization": "Bearer secret2"}, "/api/admin/nft/monthly/candidates")
        assert r["ok"] is True

def test_admin_bad_token_rejects():
    with clean_env():
        os.environ["QC_ADMIN_TOKEN"] = "secret3"
        r = qc_admin_auth_review_v1({"X-Admin-Token": "bad"}, "/api/admin/nft/monthly/candidates")
        assert r["ok"] is False
        assert r["error"] == "admin_auth_failed"

def test_testnet_requires_token_config():
    with clean_env():
        os.environ["QC_TESTNET"] = "1"
        r = qc_admin_auth_review_v1({}, "/api/security/rate-limit/status")
        assert r["ok"] is False
        assert r["error"] == "admin_token_not_configured"

def test_pagination_clamps_limit_offset():
    p = qc_admin_pagination_v1({"limit": 999999, "offset": -20})
    assert p["version"] == QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION
    assert p["limit"] == 100
    assert p["offset"] == 0
    assert p["largeExportMustUseBackgroundJob"] is True

def test_registry_contains_expected_groups():
    reg = qc_admin_endpoint_registry_v1()
    groups = [g["group"] for g in reg["groups"]]
    assert "NFT_MONTHLY" in groups
    assert "TGWT_WITHDRAW_CLAIM" in groups
    assert "WATCH_EARN" in groups
    assert "NODE_SECURITY" in groups

def test_lock_summary():
    s = qc_admin_lock_summary_v1()
    assert s["version"] == QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION
    assert "X-Admin-Token" in s["auth"]["headers"]
    assert s["auth"]["queryTokenAccepted"] is False
    assert s["pagination"]["maxLimit"] == 100

if __name__ == "__main__":
    test_admin_auth_local_without_token_allows_dev_fallback()
    test_admin_path_requires_configured_token()
    test_admin_token_header_accepts()
    test_admin_token_bearer_accepts()
    test_admin_bad_token_rejects()
    test_testnet_requires_token_config()
    test_pagination_clamps_limit_offset()
    test_registry_contains_expected_groups()
    test_lock_summary()
    print("QC_ADMIN_PANEL_AUTH_PAGINATION_V1_TEST_PASS")
