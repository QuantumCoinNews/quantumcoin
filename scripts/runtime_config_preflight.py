"""Validate the shared deployment settings. Print key names and error codes only."""
from ipaddress import ip_address
import json
import os
from pathlib import Path
from urllib.parse import urlsplit
import importlib.util

_spec=importlib.util.spec_from_file_location('qc_shared_runtime_environment',Path(__file__).resolve().parents[1]/'config/runtime_environment.py')
if _spec is None or _spec.loader is None:
    raise RuntimeError('runtime-config: CONFIG_SCHEMA:loader-unavailable')
_shared=importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_shared)


def origin(value):
    try:
        url = urlsplit(value)
        if (not url.hostname or url.username is not None or url.password is not None or
                url.path not in ("", "/") or url.query or url.fragment or
                url.port == 0 or any(char.isspace() for char in value)):
            return None
        return url
    except ValueError:
        return None


def preflight(env,service='tooling',*,strict_unknown=True):
    common=_shared.validate_environment(env,service,strict_unknown=strict_unknown)
    if service!='tooling':return common
    errors = {}

    def get(key):
        value=env.get(key, "")
        return value.strip() if isinstance(value,str) else ''

    bind = get("QC_API_LISTEN")
    if bind:
        host, sep, port = bind.rpartition(":")
        try:
            if not sep or not port.isascii() or not port.isdecimal() or not 1 <= int(port) <= 65535:
                raise ValueError
            loopback = host.lower() == "localhost"
            if host and not loopback:
                bracketed = host.startswith("[") and host.endswith("]")
                address = ip_address(host[1:-1] if bracketed else host)
                if bracketed != (address.version == 6):
                    raise ValueError
                loopback = address.is_loopback
            if not loopback and not get("QC_API_TOKEN"):
                errors["QC_API_TOKEN"] = "required-for-non-loopback-bind"
        except ValueError:
            errors["QC_API_LISTEN"] = "invalid-host-or-port"
    bridge = get("QC_API_BASE")
    if bridge:
        url = origin(bridge)
        if not url or (url.scheme != "https" and
                       (url.scheme != "http" or url.hostname not in ("127.0.0.1", "::1", "localhost"))):
            errors["QC_API_BASE"] = "expected-https-or-loopback-http-origin"
    monitor = get("QC_PUBLISHER_MONITOR_ORIGIN")
    if monitor:
        url = origin(monitor)
        if not url or url.scheme != "http" or url.hostname not in ("127.0.0.1", "::1"):
            errors["QC_PUBLISHER_MONITOR_ORIGIN"] = "expected-numeric-loopback-http-origin"
        if not get("QC_PUBLISHER_MONITOR_TOKEN"):
            errors["QC_PUBLISHER_MONITOR_TOKEN"] = "required-with-monitor-origin"
    for key in ("QC_GAME_DB", "QC_GAME_DATA_DIR", "QC_PUBLISHER_STATE_DIR",
                "QC_PUBLISHER_AUDIO_DIR", "QC_PUBLISHER_BACKGROUND_DIR"):
        value = get(key)
        if value and not Path(value).is_absolute():
            errors[key] = "expected-absolute-path"
    for key,code in common.items():errors.setdefault(key,code)
    return errors


if __name__ == "__main__":
    import argparse
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--service',choices=['node','game','publisher','tooling'],default='tooling')
    parser.add_argument('--strict-unknown',action='store_true')
    parser.add_argument('--allow-unknown',action='store_true',help='Tooling compatibility only; production validators remain strict.')
    args=parser.parse_args()
    errors = preflight(os.environ,args.service,strict_unknown=not args.allow_unknown)
    schema=json.loads(_shared.SCHEMA_PATH.read_text(encoding='utf-8'))
    ignored=sorted(key for key,entry in schema['properties'].items()
                   if entry['x-status'] in {'compatibility-ignored','test-only'} and os.environ.get(key))
    print(json.dumps({"ok": not errors, "errors": errors, "ignoredSettings": ignored}, indent=2))
    raise SystemExit(bool(errors))
