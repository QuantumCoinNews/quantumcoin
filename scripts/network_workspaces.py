"""Prepare isolated empty network directories; never import a chain or start a node."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import tempfile

ROOT = Path(__file__).resolve().parents[1]
MARKER = ".quantumcoin-network.json"
VERSION = "QC_FRESH_NETWORK_WORKSPACE_V1"


def marker_for(network):
    if network not in {"testnet", "mainnet"}:
        raise ValueError("Choose testnet or mainnet")
    return {"version": VERSION, "network": network,
            "genesis_profile": "network-v4", "balance_migration": False}


def plan(network, directory=None, package=None):
    marker = marker_for(network)
    if directory is None:
        base = os.environ.get("LOCALAPPDATA")
        if not base:
            raise ValueError("Choose an absolute data directory or set LOCALAPPDATA")
        directory = Path(base) / "QuantumCoin/networks" / (network + "-v4")
    directory = Path(directory)
    if not directory.is_absolute():
        raise ValueError("Network data directory must be absolute")
    absolute = directory.absolute()
    directory = directory.resolve()
    if directory != absolute:
        raise ValueError("Indirect network data directories are not allowed")
    if directory == ROOT or directory.is_relative_to(ROOT):
        raise ValueError("Network runtime data must be outside the source checkout")
    if package is not None:
        package = Path(package).resolve()
        if directory == package or directory.is_relative_to(package):
            raise ValueError("Network runtime data must be outside the signed package")
    selected = directory / MARKER
    prepared = False
    if directory.exists():
        if not directory.is_dir():
            raise ValueError("Network data path is not a directory")
        if selected.exists():
            if selected.is_symlink() or selected.stat().st_size > 4096:
                raise ValueError("Network workspace marker is invalid")
            try:
                def unique_names(pairs):
                    value = {}
                    for key, item in pairs:
                        if key in value:
                            raise ValueError("duplicate marker field")
                        value[key] = item
                    return value
                stored = json.loads(selected.read_text(encoding="utf-8"), object_pairs_hook=unique_names)
            except (OSError, ValueError):
                raise ValueError("Network workspace marker is invalid") from None
            if stored != marker or type(stored.get("balance_migration")) is not bool:
                raise ValueError("Directory belongs to another network/profile; original data preserved")
            prepared = True
        elif any(directory.iterdir()):
            raise ValueError("Existing nonempty directory is historical data; choose a new empty directory")
        expected = "chain_data_testnet.dat" if network == "testnet" else "chain_data.dat"
        for entry in directory.iterdir():
            if "chain" in entry.name.lower() and entry.suffix.lower() == ".dat" and entry.name != expected:
                raise ValueError("Foreign/legacy chain filename in network directory; original data preserved")
    return {**marker, "dataDir": str(directory), "prepared": prepared,
            "chainFile": "chain_data_testnet.dat" if network == "testnet" else "chain_data.dat",
            "launchPerformed": False}


def prepare(network, directory=None, package=None):
    result = plan(network, directory, package)
    if result["prepared"]:
        return result
    directory = Path(result["dataDir"])
    directory.mkdir(parents=True, exist_ok=True, mode=0o700)
    # Recheck after mkdir: an existing file is never erased to make room.
    if any(directory.iterdir()):
        raise ValueError("Directory changed during preparation; no data was overwritten")
    raw = (json.dumps(marker_for(network), sort_keys=True, indent=2) + "\n").encode("utf-8")
    fd, temporary = tempfile.mkstemp(prefix=".qc-network-init-", dir=directory)
    try:
        with os.fdopen(fd, "wb") as file:
            file.write(raw)
            file.flush()
            os.fsync(file.fileno())
        os.link(temporary, directory / MARKER)  # no-clobber publication
    finally:
        Path(temporary).unlink(missing_ok=True)
    return plan(network, directory, package)


def historical_inventory(root=ROOT):
    """Hash chain files only. Do not decode wallets, keys, balances or Gob."""
    root = Path(root).resolve()
    records = []
    excluded = {".git", "node_modules", ".venv", "venv", "__pycache__", "secrets"}
    for base, directories, files in os.walk(root, followlinks=False):
        parent = Path(base)
        directories[:] = [name for name in directories if name not in excluded
                          and (parent / name).resolve() == (parent / name).absolute()]
        for name in files:
            if "chain" not in name.lower() or not name.lower().endswith(".dat"):
                continue
            if any(word in name.lower() for word in ("priv", "key", "seed", "recovery")):
                continue
            path = parent / name
            if path.is_symlink() or not path.resolve().is_relative_to(root):
                continue
            before = path.stat()
            digest = hashlib.sha256()
            with path.open("rb") as file:
                while chunk := file.read(1024 * 1024):
                    digest.update(chunk)
            after = path.stat()
            if (before.st_size, before.st_mtime_ns) != (after.st_size, after.st_mtime_ns):
                raise ValueError("Chain file changed while inventory was being recorded")
            records.append({"path": path.relative_to(root).as_posix(), "bytes": before.st_size,
                            "sha256": digest.hexdigest(), "classification": "historical-only",
                            "networkOrigin": "not-assumed-from-filename", "importToFreshNetworks": False})
    return {"version": VERSION, "mode": "read-only", "balanceMigration": False,
            "records": sorted(records, key=lambda item: item["path"])}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["plan", "prepare", "inventory"])
    parser.add_argument("--network", choices=["testnet", "mainnet"])
    parser.add_argument("--directory", type=Path)
    parser.add_argument("--package", type=Path)
    args = parser.parse_args()
    try:
        if args.action == "inventory":
            result = historical_inventory()
        elif not args.network:
            parser.error("--network is required")
        else:
            operation = prepare if args.action == "prepare" else plan
            result = operation(args.network, args.directory, args.package)
        print(json.dumps(result, ensure_ascii=False, indent=2))
    except (OSError, ValueError) as error:
        parser.exit(1, str(error) + "\n")


if __name__ == "__main__":
    main()
