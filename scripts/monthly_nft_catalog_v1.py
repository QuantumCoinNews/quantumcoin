"""Check/generate draft catalogue views from the canonical monthly stock.

Default is read-only. No artwork, hashes, approvals or activation are created.
"""
import argparse
import json
import os
import tempfile
from pathlib import Path

STOCK = "nft/catalog/monthly_rare_nft_stock_2026.json"
PLAN = "nft/catalog/monthly_rare_nft_2026_placeholder_plan.json"
DESCRIPTION = "Monthly rare NFT candidate reward for verified QuantumCoin Telegram Space Miner activity."
METADATA_BOUNDARY = {key: False for key in ("minted", "chainMetadataWritten", "balanceChanged", "qcSupplyChanged")}
PLAN_BOUNDARY = {key: True for key in (
    "testnetPlaceholderOnly", "mainnetMintDisabledByDefault", "requiresAIReview",
    "requiresAdminApproval", "requiresFinalMetadataBeforeMainnet",
    "requiresUniqueMonthlyNFTs", "maxOneAwardPerMonth", "doesNotChangeQCSupply",
    "doesNotChangeQCBalance", "doesNotMintInTestnet")}
STOCK_BOUNDARY = {key: True for key in (
    "doesNotMint", "doesNotWriteChainMetadata", "doesNotChangeQCSupply", "doesNotChangeQCBalance",
    "preparedBeforeMainnetRequired", "uniqueNFTRequired", "finalArtworkRequiredBeforeMainnet",
    "finalMetadataRequiredBeforeMainnet", "aiReviewRequired", "adminApprovalRequired")}


class CatalogError(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise CatalogError(message)


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate JSON key")
        result[key] = value
    return result


def read_json(path):
    return json.loads(path.read_text(encoding="utf-8-sig"), object_pairs_hook=unique_object,
        parse_constant=lambda _: (_ for _ in ()).throw(CatalogError("non-finite JSON value")))


def same_json(left, right):
    # Python considers False == 0; catalogue booleans must remain booleans.
    return json.dumps(left, sort_keys=True, allow_nan=False) == json.dumps(right, sort_keys=True, allow_nan=False)


def under_root(root, relative):
    path = (root / relative).resolve()
    require(path.is_relative_to(root.resolve()), "catalogue path escapes repository")
    return path


def load_stock(root):
    stock = read_json(under_root(root, STOCK))
    require(isinstance(stock, dict), "stock must be an object")
    require(stock.get("version") == "NFT_MONTHLY_STOCK_PRE_MAINNET_V1", "invalid stock version")
    for key, value in (("year", 2026), ("totalPlanned", 12), ("awardsPerMonth", 1)):
        require(type(stock.get(key)) is int and stock[key] == value, "invalid stock " + key)
    items = stock.get("items")
    require(stock.get("rarity") == "rare" and isinstance(items, list) and len(items) == 12, "twelve rare stock items required")
    activation = stock.get("mainnetActivation", {})
    require(activation.get("enabled") is False and activation.get("testnetMintAllowed") is False and activation.get("requiresManualApproval") is True, "stock activation must remain disabled")
    require(same_json(stock.get("safeBoundary"), STOCK_BOUNDARY), "stock safe boundaries required")
    seen_names, seen_concepts = set(), set()
    for month, item in enumerate(items, 1):
        require(isinstance(item, dict), "stock item must be an object")
        period = f"2026-{month:02d}"
        identity = f"rare-{period}-001"
        require(type(item.get("stockNo")) is int and item["stockNo"] == month and item.get("period") == period and item.get("id") == identity, "invalid chronological stock identity")
        for key, seen in (("name", seen_names), ("concept", seen_concepts)):
            value = item.get(key)
            require(isinstance(value, str) and value.strip() and value == value.strip() and value not in seen, "missing or duplicate " + key)
            seen.add(value)
        require(item.get("rarity") == "rare", "invalid item rarity")
        require(item.get("imagePath") == f"telegram_game/telegram_game/dev_ui/assets/nft/{identity}.png" and item.get("metadataPath") == f"nft/catalog/metadata/{identity}.json", "invalid stock asset path")
        require(item.get("finalReady") is False and item.get("mainnetMintStatus") == "NOT_MINTED", "draft generator cannot process final or minted stock")
        require(item.get("finalImageSha256") == "" and item.get("finalMetadataSha256") == "", "draft generator cannot replace committed metadata")
        require(item.get("artworkStatus") != "FINAL_READY" and item.get("metadataStatus") == "PLACEHOLDER_EXISTS_NOT_FINAL", "stock must declare draft metadata")
        require(item.get("aiReviewStatus") == "PENDING_REVIEW" and item.get("adminApprovalStatus") == "PENDING_ADMIN_REVIEW" and item.get("uniquenessCheck") == "PENDING", "draft generator cannot replace reviewed stock")
    return stock


def expected_artifacts(root):
    stock = load_stock(root)
    slots, artifacts = [], {}
    for item in stock["items"]:
        slot = {key: item[key] for key in ("id", "period", "name", "rarity", "concept", "imagePath", "metadataPath")}
        slot.update(monthIndex=item["stockNo"], preparedBeforeMainnet=False, mintEnabled=False, minted=False, status="placeholder_not_minted")
        slots.append(slot)
        artifacts[item["metadataPath"]] = {
            "name": item["name"], "description": DESCRIPTION, "rarity": item["rarity"], "period": item["period"],
            "attributes": [{"trait_type": key, "value": value} for key, value in (
                ("Project", "QuantumCoin"), ("Reward Type", "Monthly Rare NFT"),
                ("Source", "Telegram Space Miner"), ("Mint Status", "Not Minted"))],
            "safeBoundary": dict(METADATA_BOUNDARY),
        }
    artifacts[PLAN] = {
        "version": "NFT_MONTHLY_PLACEHOLDER_V1", "year": 2026, "totalSlots": 12,
        "awardsPerMonth": 1, "rarity": "rare", "mode": "testnet_placeholder",
        "mainnetMintApproval": "", "safeBoundary": dict(PLAN_BOUNDARY), "slots": slots,
    }
    # Validate all output paths before any write.
    for relative in artifacts:
        under_root(root, relative)
    return stock, artifacts


def differences(root, artifacts):
    changed = []
    for relative, expected in artifacts.items():
        try:
            actual = read_json(under_root(root, relative))
        except (OSError, ValueError):
            actual = None
        if not same_json(actual, expected):
            changed.append(relative)
    return sorted(changed)


def write_drafts(root, artifacts):
    changes = differences(root, artifacts)
    # Refuse to overwrite final/unknown documents even with explicit --write.
    for relative in changes:
        path = under_root(root, relative)
        if not path.exists():
            continue
        current = read_json(path)
        require(isinstance(current, dict), "existing document is not a draft")
        if relative == PLAN:
            require(current.get("mode") == "testnet_placeholder" and not current.get("mainnetMintApproval"), "existing plan is not a draft")
            require(same_json(current.get("safeBoundary"), PLAN_BOUNDARY), "existing plan safe boundaries changed")
            require(isinstance(current.get("slots"), list) and all(isinstance(s, dict) and s.get("mintEnabled", False) is False and s.get("minted", False) is False and s.get("status", "placeholder_not_minted") == "placeholder_not_minted" for s in current["slots"]), "existing plan contains active mint state")
        else:
            require(set(current) == set(artifacts[relative]) and same_json(current.get("safeBoundary"), METADATA_BOUNDARY) and current.get("period") == artifacts[relative]["period"] and same_json(current.get("attributes"), artifacts[relative]["attributes"]), "existing metadata is not an unapproved draft")
    for relative in changes:
        path = under_root(root, relative)
        path.parent.mkdir(parents=True, exist_ok=True)
        data = (json.dumps(artifacts[relative], ensure_ascii=False, indent=2) + "\n").encode("utf-8")
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(dir=path.parent, prefix=".qma-draft-", delete=False) as stream:
                temporary = Path(stream.name)
                stream.write(data)
            os.replace(temporary, path)
        finally:
            if temporary is not None:
                temporary.unlink(missing_ok=True)
    return changes


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--check", action="store_true", help="read-only check (default)")
    mode.add_argument("--write", action="store_true", help="explicitly regenerate draft views")
    parser.add_argument("--require-images", action="store_true", help="also require all twelve planned image files")
    args = parser.parse_args(argv)
    try:
        stock, artifacts = expected_artifacts(args.root)
        missing_images = [i["imagePath"] for i in stock["items"] if args.require_images and not under_root(args.root, i["imagePath"]).is_file()]
        require(not (args.write and missing_images), "required artwork is missing; no files written")
        written = write_drafts(args.root, artifacts) if args.write else []
        changed = differences(args.root, artifacts)
        print(json.dumps({"ok": not changed and not missing_images, "draftOnly": True,
            "metadataExpected": 12, "metadataFiles": sum(under_root(args.root, key).is_file() for key in artifacts if key != PLAN),
            "awardsPerMonth": 1, "written": written, "drift": changed,
            "imageCheck": "required" if args.require_images else "not_requested_draft", "missingImages": missing_images}))
        return int(bool(changed or missing_images))
    except (OSError, ValueError, TypeError, KeyError) as error:
        print(json.dumps({"ok": False, "error": str(error)}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
