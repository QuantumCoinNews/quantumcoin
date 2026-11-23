"""Build the shared QC recovery worker; never starts the game or opens a wallet."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def build(output):
    go = shutil.which("go")
    if not go:
        raise ValueError("Go is required to build the local QC wallet")
    env = {k: v for k, v in os.environ.items() if not k.upper().startswith(("GO", "CGO_", "QC_", "TG_", "TELEGRAM_"))}
    env.update(GOTOOLCHAIN="local", GOENV="off", GOWORK="off")
    version = subprocess.check_output([go, "env", "GOVERSION"], env=env, text=True).strip()
    expected = json.loads((ROOT / "configs/windows-build-toolchain.json").read_text(encoding="utf-8"))["go"]["version"]
    if version != expected:
        raise ValueError("Use the pinned " + expected + " toolchain for wallet assets")
    goroot = Path(subprocess.check_output([go, "env", "GOROOT"], env=env, text=True).strip())
    runtime = goroot / "lib/wasm/wasm_exec.js"
    if not runtime.is_file():
        raise ValueError("The matching Go browser runtime is missing")
    output = output.resolve()
    expected_output = (ROOT / "telegram_game/telegram_game/dev_ui/assets/qc_wallet_crypto").resolve()
    if output != expected_output:
        raise ValueError("Wallet output must be the generated wallet asset directory")
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="qc-wallet-build-") as temporary:
        wasm = Path(temporary) / "wallet_crypto.wasm"
        subprocess.run([go, "build", "-trimpath", "-buildvcs=false", "-mod=readonly", "-ldflags=-s -w",
                        "-o", str(wasm), "./cmd/wallet-crypto"], cwd=ROOT,
                       env={**env, "GOOS": "js", "GOARCH": "wasm", "CGO_ENABLED": "0"}, check=True)
        manifest = {"version": "QC_WALLET_CRYPTO_BUILD_V1", "go": version, "files": {}}
        for name, data in (("wallet_crypto.wasm", wasm.read_bytes()), ("wasm_exec.js", runtime.read_bytes())):
            staged = output / (name + ".tmp")
            staged.write_bytes(data)
            os.replace(staged, output / name)
            manifest["files"][name] = hashlib.sha256(data).hexdigest()
        (output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print("Built local QC recovery assets with " + version)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=ROOT / "telegram_game/telegram_game/dev_ui/assets/qc_wallet_crypto")
    args = parser.parse_args()
    build(args.output)
