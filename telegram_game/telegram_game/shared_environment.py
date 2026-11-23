"""Canonical schema adapter; validate before DB selection or app composition."""
import importlib.util
from pathlib import Path

_path=Path(__file__).resolve().parents[2]/'config/runtime_environment.py'
_spec=importlib.util.spec_from_file_location('qc_shared_game_environment',_path)
if _spec is None or _spec.loader is None:
    raise RuntimeError('runtime-config: CONFIG_SCHEMA:loader-unavailable')
_shared=importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_shared)


def require_game_environment(env):
    errors=_shared.validate_environment(env,'game')
    if errors:
        raise ValueError('runtime-config: '+','.join(key+':'+code for key,code in errors.items()))
