# QC_LOAD_TEST_SCRIPTS_SAFE_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.load_test_safety_v1 import (
    QC_LOAD_TEST_SCRIPTS_VERSION,
    build_load_test_matrix_v1,
    is_blocked_endpoint_v1,
    validate_load_test_case_v1,
)

def test_safe_case_passes():
    result = validate_load_test_case_v1({
        "baseUrl": "http://127.0.0.1:8082",
        "endpoint": "/health",
        "method": "GET",
        "requests": 20,
        "concurrency": 2,
    })
    assert result["ok"] is True
    assert result["version"] == QC_LOAD_TEST_SCRIPTS_VERSION

def test_remote_blocked_by_default():
    result = validate_load_test_case_v1({
        "baseUrl": "https://example.com",
        "endpoint": "/health",
        "method": "GET",
    })
    assert result["ok"] is False
    assert "remote_base_url_blocked" in result["errors"]

def test_post_blocked():
    result = validate_load_test_case_v1({
        "baseUrl": "http://127.0.0.1:8082",
        "endpoint": "/health",
        "method": "POST",
    })
    assert result["ok"] is False
    assert "only_get_allowed" in result["errors"]

def test_mutating_endpoint_blocked():
    assert is_blocked_endpoint_v1("/api/tx/send")
    result = validate_load_test_case_v1({
        "baseUrl": "http://127.0.0.1:8082",
        "endpoint": "/api/tx/send",
        "method": "GET",
    })
    assert result["ok"] is False
    assert "mutating_endpoint_blocked" in result["errors"]

def test_limits_are_clamped():
    result = validate_load_test_case_v1({
        "baseUrl": "http://127.0.0.1:8082",
        "endpoint": "/health",
        "method": "GET",
        "requests": 999999,
        "concurrency": 999999,
    })
    assert result["normalized"]["requests"] == 500
    assert result["normalized"]["concurrency"] == 25

def test_matrix_contains_core_components():
    matrix = build_load_test_matrix_v1()
    assert matrix["ok"] is True
    components = {case["component"] for case in matrix["cases"]}
    assert "QC_NODE" in components
    assert "TELEGRAM_GAME" in components
    assert "ADMIN_PANEL" in components
    assert matrix["safeBoundary"]["doesNotChangeSupply"] is True

if __name__ == "__main__":
    test_safe_case_passes()
    test_remote_blocked_by_default()
    test_post_blocked()
    test_mutating_endpoint_blocked()
    test_limits_are_clamped()
    test_matrix_contains_core_components()
    print("QC_LOAD_TEST_SCRIPTS_SAFE_V1_TEST_PASS")
