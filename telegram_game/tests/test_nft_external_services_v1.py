"""Independent witness service + game adapter + fixed drand + final preparation.

Real Ed25519/BLS/secp signatures and separate SQLite databases. ASGI transport
maps HTTPS test origins to the test services; production uses urllib HTTPS.
"""
import concurrent.futures
import copy
import email.message
import hashlib
import io
import json
import os
from pathlib import Path
import sqlite3
import tempfile
import unittest
import urllib.error
from contextlib import closing
from unittest.mock import patch

from fastapi.testclient import TestClient
import test_nft_verified_lifecycle_v1 as fixtures
from telegram_game.utils import nft_external_evidence_v1 as external
from telegram_game.utils.nft_publication_witness_v1 import create_witness_app_v1, publication_id_v1
from telegram_game.utils import nft_final_readiness_v1 as final

draw, wallet = fixtures.draw, fixtures.wallet


class ExternalServiceTests(unittest.TestCase):
    def setUp(self):
        fixtures.VerifiedLifecycleTests.setUpClass()
        self.f=fixtures.VerifiedLifecycleTests("test_real_drand_upstream_vector")
        self.f.setUp(); self.addCleanup(self.f.tearDown)
        f=self.f
        self.clock=[f.frozen_at+2]
        self.token="test-only-independent-witness-token-123456"
        f.policy["publicationWitness"]["urlPrefix"]="https://witness.test/entries/"
        f.policy["externalServices"]={"witnessSubmitURL":"https://witness.test/v1/commitments","beaconBaseURL":"https://beacon.test"}
        f.commitment=draw.commitment_v1(f.frozen,f.program,f.policy)
        self.witness_db=f.root/"independent-witness.sqlite"
        self.scope=wallet.program_scope(f.program)
        self.app=self.new_witness()
        self.witness=TestClient(self.app);self.addCleanup(self.witness.close)
        self.calls=[];self.mode="normal"
        owner=self
        class Response:
            def __init__(self,status,body):
                self.status=status;self.raw=io.BytesIO(json.dumps(body).encode());self.headers=email.message.Message();self.headers['Content-Type']='application/json'
            def read(self,n):return self.raw.read(n)
            def __enter__(self):return self
            def __exit__(self,*args):self.raw.close()
        class Opener:
            def open(self,request,timeout):
                owner.calls.append((request.full_url,request.get_method(),dict(request.header_items())))
                if request.full_url.startswith('https://witness.test/'):
                    path=request.full_url[len('https://witness.test'):]
                    response=owner.witness.request(request.get_method(),path,content=request.data,headers=dict(request.header_items()))
                    if owner.mode=='lost-post' and request.get_method()=='POST' and response.status_code==201:
                        raise urllib.error.URLError('test-only-lost-response')
                    status,body=response.status_code,response.json()
                    if owner.mode=='forged-public' and request.get_method()=='GET' and status==200:body['signature']='00'*64
                elif request.full_url.startswith('https://beacon.test/'):
                    b=f.policy['beacon']
                    if request.full_url.endswith('/info'):
                        status,body=200,{'hash':b['chainHash'],'public_key':b['publicKey'],'schemeID':b['scheme'],'period':b['periodSeconds'],'genesis_time':b['genesisTime']}
                        if owner.mode=='wrong-info':body['public_key']='00'*48
                    else:
                        owner.assertEqual(request.full_url.rsplit('/',1)[-1],str(f.commitment['round']))
                        status,body=200,{k:v for k,v in f.beacon.items() if k!='chainHash'}
                        if owner.mode=='wrong-round':body['round']+=1
                        if owner.mode=='missing-round':status,body=404,{}
                else:raise AssertionError('unconfigured origin requested')
                if status not in (200,201):raise urllib.error.HTTPError(request.full_url,status,'test-only-error',{},io.BytesIO())
                return Response(status,body)
        self.patch=patch.object(external.urllib.request,'build_opener',return_value=Opener());self.patch.start();self.addCleanup(self.patch.stop)
        self.env=patch.dict(os.environ,{'QC_NFT_WITNESS_SUBMIT_TOKEN':self.token,'QC_API_TOKEN':'MUST-NOT-LEAVE-NODE-ORIGIN'})
        self.env.start();self.addCleanup(self.env.stop)

    def new_witness(self):
        f=self.f
        return create_witness_app_v1(lambda:sqlite3.connect(self.witness_db,timeout=20),f.witness,
            'https://witness.test',self.token,self.scope,lambda:self.clock[0])

    def publish(self):
        f=self.f
        return external.publish_and_register_v1(f.connect,f.program,f.policy,f.period,lambda:self.clock[0])

    def select(self):
        f=self.f
        return external.draw_from_beacon_v1(f.connect,f.program,f.policy,f.period,fixtures.serialize,lambda:self.clock[0])

    def test_publication_real_signatures_public_read_and_fixed_round_draw(self):
        f=self.f
        result=self.publish();self.assertTrue(result['ok'],result)
        uri='/entries/'+publication_id_v1(f.commitment)
        public=self.witness.get(uri)
        self.assertEqual(public.status_code,200)
        draw.verify_publication_v1(f.commitment,public.json(),f.policy,f.frozen_at)
        self.clock[0]=f.now
        selected=self.select();self.assertTrue(selected['ok'],selected)
        self.assertEqual(selected['receipt']['winner']['candidateId'],'candidate-1')
        self.assertFalse(any('latest' in url for url,_,_ in self.calls))
        for url,method,headers in self.calls:
            headers={k.lower():v for k,v in headers.items()}
            self.assertNotIn('x-qc-api-key',headers)
            self.assertNotIn('authorization',headers)
            if method=='GET':self.assertNotIn('x-qma-witness-token',headers)

    def test_lost_post_reply_recovers_same_public_record_without_second_commitment(self):
        self.mode='lost-post'
        result=self.publish();self.assertTrue(result['ok'],result)
        self.mode='normal'
        again=self.publish();self.assertTrue(again['alreadyRegistered'],again)
        with closing(sqlite3.connect(self.witness_db)) as conn:
            self.assertEqual(conn.execute('SELECT count(*) FROM nft_publication_records_v1').fetchone()[0],1)
        with closing(self.f.connect()) as conn:
            self.assertEqual(conn.execute('SELECT count(*) FROM monthly_nft_publication_outbox_v1').fetchone()[0],1)

    def test_witness_first_commitment_auth_scope_restart_and_immutable_storage(self):
        f=self.f
        headers={'X-QMA-Witness-Token':self.token}
        self.assertEqual(self.witness.post('/v1/commitments',json={'commitment':f.commitment}).status_code,401)
        def submit(i):
            return self.witness.post('/v1/commitments',json={'commitment':f.commitment},headers=headers)
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:results=list(pool.map(submit,range(4)))
        self.assertEqual(sum(r.status_code==201 for r in results),1)
        self.assertTrue(all(r.status_code in (200,201) for r in results))
        self.assertEqual(len({json.dumps(r.json(),sort_keys=True) for r in results}),1)
        bad={**f.commitment,'poolHash':'00'*32}
        self.assertEqual(self.witness.post('/v1/commitments',json={'commitment':bad},headers=headers).status_code,409)
        bad={**f.commitment,'campaignID':'other'}
        self.assertEqual(self.witness.post('/v1/commitments',json={'commitment':bad},headers=headers).status_code,422)
        uri='/entries/'+publication_id_v1(f.commitment)
        with TestClient(self.new_witness()) as restarted:self.assertEqual(restarted.get(uri).json(),results[0].json())
        with closing(sqlite3.connect(self.witness_db)) as conn:
            with self.assertRaises(sqlite3.IntegrityError),conn:conn.execute('DELETE FROM nft_publication_records_v1')

    def test_deadline_forgery_and_no_alternative_beacon_fallback(self):
        f=self.f
        self.clock[0]=f.commitment['beaconAt']
        self.assertFalse(self.publish()['ok']);self.assertEqual(self.calls,[])
        self.clock[0]=f.frozen_at+2
        self.mode='forged-public';self.assertFalse(self.publish()['ok'])
        self.mode='normal';self.assertTrue(self.publish()['ok'])
        self.clock[0]=f.now
        for mode in ('wrong-info','wrong-round','missing-round'):
            self.mode=mode
            result=self.select();self.assertFalse(result['ok'],result)
        with closing(f.connect()) as conn:self.assertEqual(conn.execute('SELECT count(*) FROM monthly_nft_draw_receipts_v1').fetchone()[0],0)

    def test_transport_rejects_bad_origins_and_never_redirects(self):
        for value in ('http://witness.test/x','https://user:pass@witness.test/x','https://witness.test/x?token=bad','https://witness.test/x#bad','https://witness.test/\\evil'):
            p=copy.deepcopy(self.f.policy);p['externalServices']['witnessSubmitURL']=value
            with self.assertRaises(draw.DrawRejected):external.ExternalEvidenceClientV1(p)
        handler=external._NoRedirect()
        self.assertIsNone(handler.redirect_request(None,None,302,'test',{},'https://evil.test/'))
        self.assertEqual(self.calls,[])

    def test_witness_rejects_ambiguous_oversize_and_late_requests(self):
        h={'X-QMA-Witness-Token':self.token}
        self.assertEqual(self.witness.post('/v1/commitments',content=b'{"commitment":{},"commitment":{}}',headers=h).status_code,422)
        self.assertEqual(self.witness.post('/v1/commitments',content=b'x'*16385,headers=h).status_code,413)
        self.clock[0]=self.f.commitment['beaconAt']
        self.assertEqual(self.witness.post('/v1/commitments',json={'commitment':self.f.commitment},headers=h).status_code,422)

    def test_quicknet_real_public_fixed_round_and_mutations(self):
        fixture=json.loads((Path(__file__).parent/'fixtures/drand_quicknet_round_1000_v1.json').read_text())
        info,b=fixture['info'],fixture['beacon']
        c={'round':1000,'beaconChainHash':info['hash'],'beaconScheme':info['schemeID'],'beaconPublicKey':info['public_key']}
        self.assertEqual(draw.verify_beacon_v1(c,{'chainHash':info['hash'],**b}).hex(),b['randomness'])
        bad={**b,'signature':'00'*48};bad['randomness']=hashlib.sha256(bytes.fromhex(bad['signature'])).hexdigest()
        with self.assertRaises(draw.DrawRejected):draw.verify_beacon_v1(c,{'chainHash':info['hash'],**bad})
        with self.assertRaises(draw.DrawRejected):draw.verify_beacon_v1({**c,'beaconScheme':'bls-unchained-on-g1'},{'chainHash':info['hash'],**b})

    def test_final_unsigned_statement_roles_draft_final_and_minted_contract(self):
        f=self.f
        self.assertTrue(self.publish()['ok'])
        self.clock[0]=f.now
        selected=self.select();self.assertTrue(selected['ok'],selected)
        with closing(f.connect()) as conn:
            draft=final.inspect_final_readiness_v1(conn,f.root,f.program,f.policy,'candidate-1','test-stock-1',f.now)
            self.assertEqual(draft['assetStage'],'DRAFT');self.assertFalse(draft['readyForSignedAward']);self.assertFalse(draft['mintAllowed'])
        _,approval=f.final_catalog(selected['receiptHash'])
        with closing(f.connect()) as conn:
            prepared=final.prepare_final_approval_v1(conn,f.root,f.program,f.policy,'candidate-1','test-stock-1',f.now)
        self.assertFalse(prepared['signed']);self.assertFalse(prepared['mintAllowed'])
        evidence={'statement':prepared['statement'],'approvals':[{'role':role,'signature':key.sign(bytes.fromhex(prepared['signingBytesHex'][role])).hex()} for role,key in (('ai',f.ai),('admin',f.admin))]}
        approval.write_text(json.dumps(evidence),encoding='utf-8')
        with closing(f.connect()) as conn,conn:
            ready=final.inspect_final_readiness_v1(conn,f.root,f.program,f.policy,'candidate-1','test-stock-1',f.now)
            self.assertEqual(ready['assetStage'],'FINAL_READY');self.assertTrue(ready['readyForSignedAward']);self.assertFalse(ready['mintAllowed'])
            conn.execute("UPDATE monthly_nft_candidates SET minted_status='MINTED_CONFIRMED'")
        with closing(f.connect()) as conn:
            minted=final.inspect_final_readiness_v1(conn,f.root,f.program,f.policy,'candidate-1','test-stock-1',f.now)
            self.assertFalse(minted['readyForSignedAward']);self.assertEqual(minted['chainConfirmation'],'CANONICAL_RECEIPT_RECHECK_REQUIRED')
            with self.assertRaises(draw.DrawRejected):final.prepare_final_approval_v1(conn,f.root,f.program,f.policy,'candidate-1','test-stock-1',f.now)


    def test_draft_periods_follow_verified_activation_across_year_boundary(self):
        root=Path(__file__).resolve().parents[2]
        stock=json.loads((root/'nft/catalog/monthly_rare_nft_stock_2026.json').read_text(encoding='utf-8-sig'))
        before=copy.deepcopy(stock)
        result=final.build_program_catalog_draft_v1(stock,self.f.program)
        self.assertEqual([i['period'] for i in result['items']],self.f.program['awardPeriods'])
        self.assertEqual(len({i['period'][:4] for i in result['items']}),2)
        self.assertTrue(all(not i['finalReady'] and not i['finalImageSha256'] and not i['finalMetadataSha256'] for i in result['items']))
        self.assertFalse(result['mintAllowed']);self.assertEqual(stock,before)
        stock['items'][1]['concept']=stock['items'][0]['concept']
        with self.assertRaises(draw.DrawRejected):final.build_program_catalog_draft_v1(stock,self.f.program)

    def test_authenticated_game_service_and_final_preparation_routes(self):
        f=self.f
        with tempfile.TemporaryDirectory(prefix='qma-external-api-startup-') as startup:
            original=sqlite3.connect
            def isolated(database,*args,**kwargs):
                if str(database)=='dev_api_state.db':database=str(Path(startup)/'startup.sqlite')
                return original(database,*args,**kwargs)
            with patch.object(sqlite3,'connect',side_effect=isolated):
                from telegram_game import dev_api as api
        from telegram_game.utils import nft_program_bridge_v1 as program_bridge
        settings={'TELEGRAM_GAME_ADMIN_TOKEN':'test-only-external-admin'}
        headers={'X-Admin-Token':'test-only-external-admin'}
        with patch.dict(os.environ,settings),patch.object(api,'APP_DB',str(f.db)),patch.object(api,'_now',side_effect=lambda:self.clock[0]),patch.object(api,'_NFT_REPOSITORY_ROOT_V1',f.root),patch.object(api,'_monthly_nft_evidence_context_v1',return_value=(f.program,f.policy)),TestClient(api.app) as client:
            body={'candidateId':'candidate-1','catalogItemId':'test-stock-1'}
            for route,payload in (('publish-draw-commitment',{'period':f.period}),('draw-from-beacon',{'period':f.period}),('final-approval-statement',body),('final-readiness',body)):
                uri='/api/admin/nft/monthly/'+route
                self.assertEqual(client.post(uri,json=payload).status_code,401)
                self.assertEqual(client.post(uri+'?token=test-only-external-admin',json=payload).status_code,401)
            self.assertEqual(client.get('/api/admin/nft/monthly/catalog-draft').status_code,401)
            api._QC_ACTION_BURST_EVENTS_V1.clear();api._QC_ACTION_BURST_BLOCKS_V1.clear()
            repo=Path(__file__).resolve().parents[2]
            with patch.object(api,'_NFT_REPOSITORY_ROOT_V1',repo),patch.object(program_bridge,'fetch_verified_nft_game_program_v1',return_value=f.program):
                result=client.get('/api/admin/nft/monthly/catalog-draft',headers=headers).json()
                self.assertTrue(result['ok'],result);self.assertFalse(result['filesWritten'])
            result=client.post('/api/admin/nft/monthly/publish-draw-commitment',json={'period':f.period},headers=headers).json()
            self.assertTrue(result['ok'],result)
            self.clock[0]=f.now
            result=client.post('/api/admin/nft/monthly/draw-from-beacon',json={'period':f.period},headers=headers).json()
            self.assertTrue(result['ok'],result)
            ready=client.post('/api/admin/nft/monthly/final-readiness',json=body,headers=headers).json()
            self.assertEqual(ready['assetStage'],'DRAFT');self.assertFalse(ready['mintAllowed'])
            f.final_catalog(result['receiptHash'])
            result=client.post('/api/admin/nft/monthly/final-approval-statement',json=body,headers=headers).json()
            self.assertTrue(result['ok'],result);self.assertFalse(result['signed'])
            ready=client.post('/api/admin/nft/monthly/final-readiness',json=body,headers=headers).json()
            self.assertEqual(ready['assetStage'],'FINAL_READY');self.assertTrue(ready['readyForSignedAward']);self.assertFalse(ready['mintAllowed'])


if __name__=='__main__':unittest.main(verbosity=2)
