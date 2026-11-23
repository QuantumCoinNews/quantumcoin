"""Real ASGI responses checked against published contracts, isolated from user data."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[2]
if os.environ.get("QC_ENDPOINT_SCHEMA_TEST_CHILD") != "1":
    with tempfile.TemporaryDirectory(prefix="qc-endpoint-schema-") as temp:
        env = os.environ.copy()
        env.update(QC_ENDPOINT_SCHEMA_TEST_CHILD="1", QC_PLAYER_RUNTIME_TEST_CHILD="1",
                   QC_GAME_DB=str(Path(temp)/"test.sqlite"), QC_GAME_LOAD_DOTENV="0",
                   TELEGRAM_BOT_TOKEN="test-only-schema-bot",
                   TELEGRAM_GAME_ADMIN_TOKEN="test-only-schema-operator",
                   QC_API_BASE="http://127.0.0.1:1", QC_API_TOKEN="test-only-no-node")
        env.update(QC_WALLET_LEDGER_TEST_CHILD="1",QC_EXACT_TEST_CHILD="1",
                   QC_VERIFIED_REWARDS_TEST_CHILD="1",QC_X_REWARD_TEST_CHILD="1",
                   QC_WALLET_LINK_NETWORK_ID="test-only-network",QC_WALLET_LINK_GENESIS_HEX="11"*32)
        raise SystemExit(subprocess.call([sys.executable,str(Path(__file__).resolve())],env=env,cwd=temp))

import time
import unittest
from unittest.mock import patch
sys.path.insert(0, str(ROOT/'telegram_game'))
from fastapi.testclient import TestClient
from pydantic import TypeAdapter, ValidationError
from telegram_game import dev_api as api, player_auth
from telegram_game.endpoint_response_contracts import SUCCESS, ErrorDocument, TGWTBalances


def assert_wire(response, route):
    if "application/json" not in response.headers.get("content-type", ""):
        return
    data = response.json()
    if response.status_code == 422:
        if data.get('ok') is False:
            TypeAdapter(ErrorDocument).validate_python(data)
            return
        assert set(data)=={"detail"}
        for item in data['detail']:
            assert set(item)=={'loc','msg','type'}
        return
    if response.status_code >= 400:
        TypeAdapter(ErrorDocument).validate_python(data)
    else:
        contract = route.response_model or SUCCESS[route.endpoint.__name__]
        TypeAdapter(contract | ErrorDocument).validate_python(data)


class EndpointSchemaTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.client=TestClient(api.app)
        session=player_auth.issue_session(api._db,'tg_9001',api._telegram_bot_token(),int(time.time()))
        cls.player={'Authorization':'Bearer '+session['accessToken']}
        cls.operator={'X-Admin-Token':os.environ['TELEGRAM_GAME_ADMIN_TOKEN']}
        cls.routes=[r for r in api.app.routes if getattr(r,'include_in_schema',False)]

    def test_every_json_operation_has_named_schema_and_unambiguous_identity(self):
        spec=api.app.openapi();seen=set()
        for route in self.routes:
            for method in route.methods:
                identity=(method,route.path)
                self.assertNotIn(identity,seen);seen.add(identity)
                operation=spec['paths'][route.path][method.lower()]
                if '200' not in operation['responses']:
                    self.assertIn('410' if route.path!='/api/admin/tgwt/withdraw/confirm' else '503',operation['responses'])
                    continue
                response=operation['responses']['200']['content']
                if route.path=='/admin/settlement':
                    self.assertIn('text/html',response)
                else:
                    schema=response['application/json']['schema']
                    self.assertTrue(schema, identity)
                    self.assertNotEqual(schema.get('type'),'null',identity)
        self.assertGreater(len(seen),90)

    def test_authenticated_reads_match_wire_contracts_without_live_node(self):
        async def unavailable(*args,**kwargs):
            return {'ok':False,'error':'test_node_unavailable'}
        with patch.object(api,'qc_fetch_health',side_effect=unavailable), \
             patch.object(api,'qc_fetch_address_balance',side_effect=unavailable), \
             patch('telegram_game.utils.qc_api.fetch_runtime_policy',side_effect=unavailable):
            for route in self.routes:
                if 'GET' not in route.methods or route.path=='/admin/settlement':continue
                with self.subTest(path=route.path):
                    headers=self.operator if player_auth.operator_path(route.path) else self.player
                    response=self.client.get(route.path,headers=headers,params={
                        'playerId':'tg_9001','user_id':'tg_9001','userId':'tg_9001',
                        'period':'2026-11','requestId':'test-absent','addr':'invalid','txid':'00'*32})
                    self.assertNotEqual(response.status_code,500,response.text)
                    assert_wire(response,route)

    def test_all_private_routes_describe_actual_unauthorized_envelope(self):
        for route in self.routes:
            for method in route.methods:
                if not (route.path.startswith('/api/') or route.path=='/admin/settlement'):continue
                if (method,route.path) in player_auth.PUBLIC:continue
                with self.subTest(method=method,path=route.path):
                    response=self.client.request(method,route.path,json={})
                    self.assertEqual(response.status_code,401,response.text)
                    TypeAdapter(ErrorDocument).validate_python(response.json())

    def test_exact_ledger_types_reject_float_bool_missing_and_string_coercion(self):
        good=dict(earned_u=1,pending_u=0,verified_u=1,withdrawn_u=0,rev=1,
                  reserved_u=0,legacyUnreconciledRequests=0,accountingConsistent=True)
        TGWTBalances.model_validate(good)
        for value in [1.5,True,'1',float('nan'),float('inf')]:
            with self.subTest(value=value),self.assertRaises(ValidationError):
                TGWTBalances.model_validate({**good,'verified_u':value})
        missing=dict(good);del missing['earned_u']
        with self.assertRaises(ValidationError):TGWTBalances.model_validate(missing)

    def test_bridge_http_failure_closes_stream_and_never_follows_redirect(self):
        import io
        import urllib.error
        from telegram_game.utils import qc_api
        stream=io.BytesIO(b'test-only-rejected-receipt')
        error=urllib.error.HTTPError('http://127.0.0.1:1/test',302,'Found',{},stream)
        with patch.object(qc_api._qc_urllib_request,'build_opener') as opener:
            opener.return_value.open.side_effect=error
            with self.assertRaises(urllib.error.HTTPError):
                qc_api.node_urlopen('http://127.0.0.1:1/test',1)
        self.assertTrue(stream.closed)

    def test_new_json_route_without_contract_is_rejected(self):
        from telegram_game.endpoint_response_contracts import ContractRoute
        def undocumented():return {'ok':True}
        with self.assertRaisesRegex(ValueError,'Missing canonical response contract'):
            ContractRoute('/api/test-only-undocumented',undocumented,methods=['GET'])

    def test_existing_identity_wallet_ledger_and_mining_flows_against_schemas(self):
        sys.path.insert(0,str(Path(__file__).resolve().parent))
        import test_player_runtime_boundary_v1 as identity
        import test_wallet_ledger_security_v1 as wallet
        import test_exact_numbers_node_policy_v1 as accounting
        import test_nft_draw_freeze_api_v1 as freeze
        import test_nft_external_services_v1 as nft_services
        import test_x_reward_verification_v1 as social
        original=TestClient.request
        checked=set()
        def request(client,method,url,*args,**kwargs):
            response=original(client,method,url,*args,**kwargs)
            path=response.request.url.path
            route=next((r for r in self.routes if r.path==path and method.upper() in r.methods),None)
            if route:
                assert_wire(response,route)
                checked.add((method.upper(),path,response.status_code))
            return response
        with patch.object(TestClient,'request',request):
            for module in (identity,wallet,accounting,freeze,nft_services,social):
                environment=({'TELEGRAM_GAME_ADMIN_TOKEN':'test-wallet-ledger-admin',
                              'QC_API_TOKEN':'test-node-token'} if module is wallet else
                             {'TELEGRAM_GAME_ADMIN_TOKEN':'test-only-freeze-admin'} if module is freeze else {})
                with patch.dict(os.environ,environment):
                    suite=unittest.defaultTestLoader.loadTestsFromModule(module)
                    result=unittest.TextTestRunner(verbosity=1).run(suite)
                self.assertTrue(result.wasSuccessful(),module.__name__)
        self.assertGreater(len(checked),25)


if __name__=='__main__':unittest.main(verbosity=2)
