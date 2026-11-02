"""Content-bound AI/admin approval evidence for the canonical mint bridge.

Draft catalogues remain drafts. A status string or the legacy approval token
does not authenticate an approver. The local policy pins distinct role keys.
"""
import hashlib
import json
import re

from .nft_draw_evidence_v1 import (DrawRejected, require, hex_bytes, verify_ed25519,
                                   digest)
from .nft_wallet_proof_v1 import program_scope

VERSION = "QC_NFT_FINAL_ASSET_APPROVAL_V1"


def build_program_catalog_draft_v1(stock, program):
    """Map 12 concepts onto verified activation periods. Returns a draft only.

    Never overwrites catalogue files or claims final content exists. Supports
    programmes crossing a year boundary; old 2026 names are not activation.
    """
    from .nft_program_bridge_v1 import _validate_program_v1
    require(_validate_program_v1(program) is not None, "verified_program_required")
    require(isinstance(stock, dict) and isinstance(stock.get("items"), list) and len(stock["items"]) == 12, "invalid_catalog_template")
    templates = stock["items"]
    require(all(isinstance(item, dict) and type(item.get("stockNo")) is int for item in templates)
            and {item["stockNo"] for item in templates} == set(range(1, 13)), "invalid_catalog_template")
    for field in ("name", "concept"):
        require(all(isinstance(item.get(field), str) and 0 < len(item[field]) <= 512 for item in templates)
                and len({item[field] for item in templates}) == 12, "catalog_template_not_unique")
    items = []
    for template, period in zip(sorted(templates, key=lambda item: item["stockNo"]), program["awardPeriods"]):
        identifier = "rare-" + period + "-001"
        items.append({"stockNo": template["stockNo"], "id": identifier, "period": period,
            "name": template["name"], "concept": template["concept"], "rarity": "rare",
            "imagePath": "telegram_game/telegram_game/dev_ui/assets/nft/" + identifier + ".png",
            "metadataPath": "nft/catalog/metadata/" + identifier + ".json",
            "finalImageSha256": "", "finalMetadataSha256": "", "finalReady": False,
            "artworkStatus": "NOT_READY", "metadataStatus": "DRAFT_REQUIRES_CONTENT",
            "uniquenessCheck": "PENDING", "aiReviewStatus": "PENDING_REVIEW",
            "adminApprovalStatus": "PENDING_ADMIN_REVIEW", "mainnetMintStatus": "NOT_MINTED"})
    return {"version": "QC_NFT_ACTIVATION_CATALOG_DRAFT_V1", **program_scope(program),
            "status": "DRAFT", "totalPlanned": 12, "awardsPerMonth": 1,
            "rarity": "rare", "items": items, "mintAllowed": False,
            "notes": ["Draft generated from verified activation; final assets and signatures are still required."]}


def _approval_context(root, program, candidate_id, catalog_id, period, owner, metadata_hash, draw_hash):
    require(isinstance(catalog_id, str) and re.fullmatch(r"[A-Za-z0-9_-]{1,128}", catalog_id), "invalid_final_catalog_id")
    catalog_path = root / "nft/catalog/monthly_rare_nft_stock_2026.json"
    raw = catalog_path.read_bytes()
    require(len(raw) <= 65536, "catalog_too_large")
    from .nft_mint_bridge_v1 import _unique_object
    catalog = json.loads(raw.decode("utf-8-sig"), object_pairs_hook=_unique_object)
    items = catalog["items"]
    item = next((c for c in items if c["id"] == catalog_id and c["period"] == period), None)
    require(item is not None, "final_catalog_item_required")
    return {"version": VERSION, **program_scope(program), "candidateId": candidate_id,
                "catalogId": catalog_id, "period": period, "walletOwner": owner,
                "metadataHash": metadata_hash, "imageHash": item["finalImageSha256"],
                "catalogHash": hashlib.sha256(raw).hexdigest(), "drawReceiptHash": draw_hash}


def require_final_approval_v1(root, program, policy, candidate_id, catalog_id, period,
                              owner, metadata_hash, draw_hash, now):
    from .nft_mint_bridge_v1 import _unique_object
    expected = _approval_context(root, program, candidate_id, catalog_id, period, owner, metadata_hash, draw_hash)
    path = root / "nft/catalog/approvals" / (catalog_id + ".json")
    require(path.is_file() and path.stat().st_size <= 16384, "final_approval_evidence_required")
    evidence = json.loads(path.read_text(encoding="utf-8-sig"), object_pairs_hook=_unique_object)
    require(isinstance(evidence, dict) and set(evidence) == {"statement", "approvals"}, "invalid_final_approval_evidence")
    statement = evidence["statement"]
    require(isinstance(statement, dict) and set(statement) == set(expected) | {"approvedAt"}
            and all(statement[k] == v for k, v in expected.items()), "final_approval_binding_mismatch")
    require(type(statement["approvedAt"]) is int and 0 < statement["approvedAt"] <= now, "invalid_final_approval_time")
    keys = policy.get("finalApprovers")
    require(isinstance(keys, dict) and set(keys) == {"ai", "admin"}
            and keys["ai"] != keys["admin"], "final_approvers_not_configured")
    for key in keys.values():
        hex_bytes(key, 32)
    approvals = evidence["approvals"]
    require(isinstance(approvals, list) and len(approvals) == 2
            and all(isinstance(a, dict) and set(a) == {"role", "signature"} for a in approvals)
            and {a["role"] for a in approvals} == {"ai", "admin"}, "final_role_approvals_required")
    for a in approvals:
        verify_ed25519(keys[a["role"]], a["signature"], VERSION + ":" + a["role"], statement)
    return digest(VERSION, evidence)


def prepare_final_approval_v1(conn, root, program, policy, candidate_id, catalog_id, now):
    """Unsigned statement + exact role signing bytes. Never signs or mints."""
    from .nft_mint_bridge_v1 import _candidate, _catalog_metadata
    from .nft_draw_evidence_v1 import require_draw_receipt_v1
    from .nft_wallet_proof_v1 import canonical
    require(type(now) is int and now > 0, "invalid_final_approval_time")
    candidate, _, owner = _candidate(conn, candidate_id)
    require(candidate[2:6] == ("AI_REVIEW_PASSED", "LOW", "APPROVED_BY_ADMIN", "SELECTED_MONTHLY_WINNER")
            and candidate[6] == "NOT_MINTED", "candidate_not_approved_for_final_preparation")
    metadata = _catalog_metadata(root, catalog_id, candidate[1], program)
    draw_hash = require_draw_receipt_v1(conn, program, policy, candidate_id, candidate[1], owner, now)
    statement = {**_approval_context(root, program, candidate_id, catalog_id, candidate[1], owner, metadata, draw_hash), "approvedAt": now}
    return {"ok": True, "statement": statement, "signingBytesHex": {
        role: ((VERSION + ":" + role).encode("ascii") + b"\x00" + canonical(statement).encode("ascii")).hex()
        for role in ("ai", "admin")}, "mintAllowed": False, "signed": False}


def inspect_final_readiness_v1(conn, root, program, policy, candidate_id, catalog_id, now):
    """Read-only technical contract: no status field alone grants a mint."""
    from .nft_mint_bridge_v1 import _candidate, _catalog_metadata
    from .nft_draw_evidence_v1 import require_draw_receipt_v1
    result = {"ok": True, "assetStage": "DRAFT", "approvalStage": "MISSING",
              "gameMintStage": "UNKNOWN", "chainConfirmation": "NOT_CHECKED",
              "readyForSignedAward": False, "mintAllowed": False, "reasons": []}
    try:
        candidate, _, owner = _candidate(conn, candidate_id)
        result["gameMintStage"] = candidate[6]
        metadata = _catalog_metadata(root, catalog_id, candidate[1], program)
        result["assetStage"] = "FINAL_CONTENT_VERIFIED"
        hashed = require_draw_receipt_v1(conn, program, policy, candidate_id, candidate[1], owner, now)
        require_final_approval_v1(root, program, policy, candidate_id, catalog_id, candidate[1], owner, metadata, hashed, now)
        result["approvalStage"] = "VERIFIED"
        result["assetStage"] = "FINAL_READY"
        result["readyForSignedAward"] = bool(program["mintProtocolActive"] and candidate[2:6] ==
            ("AI_REVIEW_PASSED", "LOW", "APPROVED_BY_ADMIN", "SELECTED_MONTHLY_WINNER") and candidate[6] == "NOT_MINTED")
        if candidate[6] in {"MINTED_CONFIRMED", "MINT_RECONCILIATION_REQUIRED"}:
            result["chainConfirmation"] = "CANONICAL_RECEIPT_RECHECK_REQUIRED"
    except ValueError as error:
        # Errors from the validators are fixed public codes, not raw remote data.
        if isinstance(error, (DrawRejected,)):
            result["reasons"].append(str(error))
        else:
            from .nft_mint_bridge_v1 import NFTMintBridgeError
            result["reasons"].append(str(error) if isinstance(error, NFTMintBridgeError) else "readiness_evidence_unavailable")
    except (OSError, KeyError, TypeError):
        result["reasons"].append("readiness_evidence_unavailable")
    return result
