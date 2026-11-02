"""Adversarial mutations of real signed evidence; same isolated fixture chain."""
import copy
import hashlib
import json
import os
import unittest
from contextlib import closing
from unittest.mock import patch

from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec

import test_nft_verified_lifecycle_v1 as fixtures

wallet, draw, mint = fixtures.wallet, fixtures.draw, fixtures.mint


class EvidenceAdversarialTests(unittest.TestCase):
    def setUp(self):
        fixtures.VerifiedLifecycleTests.setUpClass()
        self.f=fixtures.VerifiedLifecycleTests("test_real_drand_upstream_vector")
        self.f.setUp()
        self.addCleanup(self.f.tearDown)

    def test_wrong_key_signature_scope_and_failed_attempt_do_not_consume_nonce(self):
        f=self.f
        issued=wallet.issue_wallet_challenge_v1(f.connect,f.program,"tg_101",f.period,f.frozen_at)
        key=ec.derive_private_key(2,ec.SECP256K1())
        public=key.public_key().public_bytes(serialization.Encoding.X962,serialization.PublicFormat.UncompressedPoint).hex()
        wrong_sig=key.sign(bytes.fromhex(issued["signingBytesHex"]),ec.ECDSA(hashes.SHA256())).hex()
        correct_sig=f.key.sign(bytes.fromhex(issued["signingBytesHex"]),ec.ECDSA(hashes.SHA256())).hex()
        nonce=issued["challenge"]["nonce"]
        self.assertEqual(wallet.consume_wallet_challenge_v1(f.connect,f.program,"tg_101",nonce,public,wrong_sig,f.frozen_at+1)["error"],"wallet_key_owner_mismatch")
        other={**f.program,"campaignID":"test-only-other-campaign"}
        self.assertEqual(wallet.consume_wallet_challenge_v1(f.connect,other,"tg_101",nonce,f.public,correct_sig,f.frozen_at+1)["error"],"wallet_challenge_scope_mismatch")
        changed={**issued["challenge"],"period":f.program["awardPeriods"][1]}
        signature=f.key.sign(wallet.DOMAIN+wallet.canonical(changed).encode("ascii"),ec.ECDSA(hashes.SHA256())).hex()
        self.assertEqual(wallet.consume_wallet_challenge_v1(f.connect,f.program,"tg_101",nonce,f.public,signature,f.frozen_at+1)["error"],"invalid_wallet_signature")
        self.assertTrue(wallet.consume_wallet_challenge_v1(f.connect,f.program,"tg_101",nonce,f.public,correct_sig,f.frozen_at+1)["ok"])

    def test_late_wallet_proof_cannot_retroactively_qualify_frozen_pool(self):
        f=self.f
        with closing(f.connect()) as conn,conn:
            conn.execute("DELETE FROM monthly_nft_wallet_challenges_v1")
        issued=wallet.issue_wallet_challenge_v1(f.connect,f.program,"tg_101",f.period,f.frozen_at+1)
        signature=f.key.sign(bytes.fromhex(issued["signingBytesHex"]),ec.ECDSA(hashes.SHA256())).hex()
        self.assertTrue(wallet.consume_wallet_challenge_v1(f.connect,f.program,"tg_101",issued["challenge"]["nonce"],f.public,signature,f.frozen_at+2)["ok"])
        self.assertEqual(f.register()["error"],"wallet_control_proof_required")

    def test_wallet_change_after_external_commitment_cannot_redirect_draw(self):
        f=self.f
        self.assertTrue(f.register()["ok"])
        with closing(f.connect()) as conn,conn:
            conn.execute("UPDATE players SET qc_address=?",(fixtures.address_for_owner("cc"*20),))
        self.assertEqual(f.select()["error"],"wallet_binding_changed")

    def test_changed_review_after_freeze_cannot_authorize_draw(self):
        f=self.f
        self.assertTrue(f.register()["ok"])
        with closing(f.connect()) as conn,conn:
            conn.execute("UPDATE monthly_nft_candidates SET admin_status='REJECTED'")
        self.assertEqual(f.select()["error"],"draw_candidate_not_approved")

    def test_locally_tampered_receipt_cannot_replace_recomputed_signed_evidence(self):
        f=self.f
        self.assertTrue(f.register()["ok"])
        result=f.select()
        self.assertTrue(result["ok"],result)
        bad=copy.deepcopy(result["receipt"])
        bad["winner"]["candidateId"]="attacker"
        with closing(f.connect()) as conn,conn:
            # Model privileged DB tampering: even a recalculated local hash
            # cannot replace the beacon/frozen-manifest-derived winner.
            conn.execute("DROP TRIGGER monthly_nft_draw_receipts_v1_update")
            conn.execute("UPDATE monthly_nft_draw_receipts_v1 SET receipt_json=?,receipt_hash=?",
                         (wallet.canonical(bad),draw.digest(draw.VERSION,bad)))
        with closing(f.connect()) as conn:
            with self.assertRaisesRegex(draw.DrawRejected,"draw_receipt_tampered"):
                draw.require_draw_receipt_v1(conn,f.program,f.policy,"candidate-1",f.period,f.owner,f.now)

    def test_role_forgery_and_catalog_changes_invalidate_final_approval(self):
        f=self.f
        self.assertTrue(f.register()["ok"])
        selected=f.select();self.assertTrue(selected["ok"],selected)
        policy_path,approval=f.final_catalog(selected["receiptHash"])
        original=json.loads(approval.read_text(encoding="utf-8"))
        def check():
            with closing(f.connect()) as conn:
                return mint._lifecycle_evidence(conn,f.root,f.program,"candidate-1","test-stock-1",f.period,f.owner,f.payload["record"]["metadataHash"])
        with patch.dict(os.environ,{"QC_NFT_EVIDENCE_POLICY_PATH":str(policy_path)}),patch.object(mint.time,"time",return_value=f.now+1):
            self.assertEqual(len(check()),2)
            for mutation in ("duplicate-role","wrong-role-key","wrong-owner","unsigned-status"):
                bad=copy.deepcopy(original)
                if mutation=="duplicate-role": bad["approvals"][1]["role"]="ai"
                if mutation=="wrong-role-key": bad["approvals"][1]["signature"]=bad["approvals"][0]["signature"]
                if mutation=="wrong-owner": bad["statement"]["walletOwner"]="cc"*20
                if mutation=="unsigned-status": bad["approvals"]=[{"role":"ai","signature":"00"*64},{"role":"admin","signature":"00"*64}]
                approval.write_text(json.dumps(bad),encoding="utf-8")
                with self.assertRaises(ValueError,msg=mutation): check()
            approval.write_text(json.dumps(original),encoding="utf-8")
            catalog=f.root/"nft/catalog/monthly_rare_nft_stock_2026.json"
            catalog.write_bytes(catalog.read_bytes()+b" ")
            with self.assertRaisesRegex(draw.DrawRejected,"final_approval_binding_mismatch"): check()

    def test_wrong_local_policy_scheme_and_program_fail_closed(self):
        f=self.f
        bad=copy.deepcopy(f.policy);bad["beacon"]["scheme"]="latest"
        with self.assertRaisesRegex(draw.DrawRejected,"unsupported_beacon_scheme"):
            draw.commitment_v1(f.frozen,f.program,bad)
        bad=copy.deepcopy(f.policy);bad["campaignID"]="other"
        with self.assertRaisesRegex(draw.DrawRejected,"evidence_policy_program_mismatch"):
            draw.commitment_v1(f.frozen,f.program,bad)
        with patch.dict(os.environ,{"QC_NFT_EVIDENCE_POLICY_PATH":"relative-policy.json"}):
            with self.assertRaises(draw.DrawRejected):draw.load_policy_v1(f.program)


if __name__=="__main__":
    unittest.main(verbosity=2)
