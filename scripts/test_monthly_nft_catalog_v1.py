import contextlib
import copy
import hashlib
import io
import json
import tempfile
import unittest
from pathlib import Path

import monthly_nft_catalog_v1 as catalog

REPO = Path(__file__).resolve().parents[1]


class MonthlyNFTCatalogTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="qma0072-catalog-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.stock = catalog.read_json(REPO / catalog.STOCK)
        self.save_stock()

    def save_stock(self):
        path = self.root / catalog.STOCK
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(self.stock), encoding="utf-8")

    def snapshot(self):
        return {str(p.relative_to(self.root)): hashlib.sha256(p.read_bytes()).hexdigest()
                for p in self.root.rglob("*") if p.is_file()}

    def generate(self):
        _, artifacts = catalog.expected_artifacts(self.root)
        catalog.write_drafts(self.root, artifacts)
        return artifacts

    def cli(self, *args):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            code = catalog.main(["--root", str(self.root), *args])
        return code, json.loads(out.getvalue())

    def test_complete_generation_is_idempotent_and_preserves_june_bytes(self):
        june = "nft/catalog/metadata/rare-2026-06-001.json"
        original = (REPO / june).read_bytes()
        path = self.root / june
        path.parent.mkdir(parents=True)
        path.write_bytes(original)
        _, artifacts = catalog.expected_artifacts(self.root)
        self.assertEqual(len(catalog.write_drafts(self.root, artifacts)), 12)
        self.assertEqual(path.read_bytes(), original)
        self.assertEqual(len(list(path.parent.glob("rare-2026-*-001.json"))), 12)
        before = self.snapshot()
        self.assertEqual(catalog.write_drafts(self.root, artifacts), [])
        self.assertEqual(self.snapshot(), before)
        code, result = self.cli("--check")
        self.assertEqual(code, 0, result)
        self.assertEqual(result["metadataFiles"], 12)
        self.assertTrue(result["draftOnly"])
        plan = catalog.read_json(self.root / catalog.PLAN)
        self.assertEqual(plan["awardsPerMonth"], 1)
        self.assertTrue(all(s["mintEnabled"] is False and s["minted"] is False for s in plan["slots"]))

    def test_default_check_is_readonly_and_detects_missing_metadata(self):
        before = self.snapshot()
        code, result = self.cli()
        self.assertEqual(code, 1)
        self.assertEqual(len(result["drift"]), 13)
        self.assertEqual(result["metadataFiles"], 0)
        self.assertEqual(self.snapshot(), before)

    def test_canonical_name_change_requires_both_derived_views_to_change(self):
        self.generate()
        self.stock["items"][0]["name"] = "Updated Quantum Nebula Cartographer"
        self.save_stock()
        _, artifacts = catalog.expected_artifacts(self.root)
        self.assertEqual(catalog.differences(self.root, artifacts), sorted([
            catalog.PLAN, self.stock["items"][0]["metadataPath"]]))
        catalog.write_drafts(self.root, artifacts)
        self.assertEqual(self.cli("--check")[0], 0)

    def test_metadata_minted_and_nonboolean_boundaries_are_rejected(self):
        artifacts = self.generate()
        relative = self.stock["items"][0]["metadataPath"]
        for value in (True, 0):
            with self.subTest(value=value):
                metadata = copy.deepcopy(artifacts[relative])
                metadata["safeBoundary"]["minted"] = value
                (self.root / relative).write_text(json.dumps(metadata))
                before = self.snapshot()
                self.assertIn(relative, catalog.differences(self.root, artifacts))
                with self.assertRaises(catalog.CatalogError):
                    catalog.write_drafts(self.root, artifacts)
                self.assertEqual(self.snapshot(), before)

    def test_invalid_identity_final_or_reviewed_stock_cannot_generate(self):
        original = copy.deepcopy(self.stock)
        changes = (
            lambda s: s.update(awardsPerMonth=2),
            lambda s: s.update(awardsPerMonth=True),
            lambda s: s["items"].append(copy.deepcopy(s["items"][0])),
            lambda s: s["items"][0].update(period="2026-06"),
            lambda s: s["items"][0].update(metadataPath="../outside.json"),
            lambda s: s["items"][0].update(name=s["items"][1]["name"]),
            lambda s: s["items"][0].update(finalReady=True),
            lambda s: s["items"][0].update(finalMetadataSha256="ab" * 32),
            lambda s: s["items"][0].update(aiReviewStatus="APPROVED"),
            lambda s: s["mainnetActivation"].update(enabled=True),
        )
        for index, change in enumerate(changes):
            with self.subTest(index=index):
                self.stock = copy.deepcopy(original)
                change(self.stock)
                self.save_stock()
                before = self.snapshot()
                with self.assertRaises(catalog.CatalogError):
                    catalog.expected_artifacts(self.root)
                self.assertEqual(self.snapshot(), before)

    def test_duplicate_json_keys_are_rejected(self):
        path = self.root / catalog.STOCK
        path.write_text(path.read_text().replace('"year": 2026', '"year": 2026, "year": 2026', 1))
        with self.assertRaises(catalog.CatalogError):
            catalog.expected_artifacts(self.root)

    def test_required_missing_artwork_is_reported_and_never_generated(self):
        self.generate()
        before = self.snapshot()
        code, result = self.cli("--check", "--require-images")
        self.assertEqual(code, 1)
        self.assertEqual(len(result["missingImages"]), 12)
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(self.cli("--write", "--require-images")[0], 2)
        self.assertEqual(self.snapshot(), before)

    def test_final_plan_prevents_any_partial_draft_write(self):
        artifacts = self.generate()
        plan = catalog.read_json(self.root / catalog.PLAN)
        plan["mode"] = "mainnet"
        (self.root / catalog.PLAN).write_text(json.dumps(plan))
        metadata_path = self.root / self.stock["items"][0]["metadataPath"]
        metadata = catalog.read_json(metadata_path)
        metadata["name"] = "Drifted draft name"
        metadata_path.write_text(json.dumps(metadata))
        before = self.snapshot()
        with self.assertRaises(catalog.CatalogError):
            catalog.write_drafts(self.root, artifacts)
        self.assertEqual(self.snapshot(), before)

    def test_repository_artifacts_match_canonical_source(self):
        _, artifacts = catalog.expected_artifacts(REPO)
        self.assertEqual(catalog.differences(REPO, artifacts), [])


if __name__ == "__main__":
    unittest.main(verbosity=2)
