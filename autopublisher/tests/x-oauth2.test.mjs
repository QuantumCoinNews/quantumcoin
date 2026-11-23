import test from 'node:test';
import assert from 'node:assert/strict';
import {ensureXOAuth2AccessToken,xOAuth2Lifecycle,xOAuth2RefreshAt,xOAuth2TokenStatus} from '../dist/runtime/xOAuth2.js';

const instant=Date.parse('2026-11-24T12:00:00.000Z');
const token=(patch={})=>JSON.stringify({access_token:'fixture-access',refresh_token:'fixture-refresh',expires_in:7200,expires_at:instant+7200000,token_type:'bearer',...patch});

test('X OAuth2 lifecycle derives renewal deadline from expires_in with a five-minute safety window',()=>{
 const env={X_OAUTH2_CLIENT_ID:'fixture-client',X_OAUTH2_TOKEN_JSON:token()};
 assert.equal(xOAuth2RefreshAt(JSON.parse(env.X_OAUTH2_TOKEN_JSON)),instant+6900000);
 assert.deepEqual(xOAuth2Lifecycle(env,instant),{status:'valid',expiresAt:new Date(instant+7200000).toISOString(),refreshAt:new Date(instant+6900000).toISOString(),remainingSeconds:7200});
 assert.equal(xOAuth2TokenStatus(env,instant+6900000),'refresh-due');
 assert.equal(xOAuth2RefreshAt({access_token:'short',expires_in:10,expires_at:instant+10000}),instant+5000);
});

test('expired-window refresh rotates token and persists the new expiry before use',async()=>{
 const env={X_OAUTH2_CLIENT_ID:'fixture-client',X_OAUTH2_CLIENT_SECRET:'fixture-secret',X_OAUTH2_TOKEN_JSON:token()};let request;let persisted;
 const access=await ensureXOAuth2AccessToken(env,{now:()=>instant+6960000,fetcher:async(url,options)=>{request={url,options};return new Response(JSON.stringify({access_token:'fixture-access-next',refresh_token:'fixture-refresh-next',expires_in:7200,token_type:'bearer'}),{status:200});},persist:async value=>{persisted=value;}});
 assert.equal(access,'fixture-access-next');assert.equal(request.url,'https://api.x.com/2/oauth2/token');
 assert.equal(request.options.method,'POST');assert.match(request.options.headers.get('authorization'),/^Basic /);
 assert.equal(new URLSearchParams(request.options.body).get('grant_type'),'refresh_token');
 assert.equal(persisted.refresh_token,'fixture-refresh-next');assert.equal(persisted.expires_at,instant+14160000);
 assert.equal(JSON.parse(env.X_OAUTH2_TOKEN_JSON).access_token,'fixture-access-next');
});

test('public X OAuth2 client refresh sends client_id in the form and retains omitted refresh token',async()=>{
 const env={X_OAUTH2_CLIENT_ID:'fixture-public-client',X_OAUTH2_TOKEN_JSON:token()};let request;
 await ensureXOAuth2AccessToken(env,{now:()=>instant+6960000,fetcher:async(_url,options)=>{request=options;return new Response(JSON.stringify({access_token:'next',expires_in:3600}),{status:200});},persist:async()=>{}});
 const form=new URLSearchParams(request.body);assert.equal(form.get('client_id'),'fixture-public-client');assert.equal(form.get('refresh_token'),'fixture-refresh');
 assert.equal(JSON.parse(env.X_OAUTH2_TOKEN_JSON).refresh_token,'fixture-refresh');
});

test('X OAuth2 fails closed when refresh token is revoked or rotated credentials cannot be persisted',async()=>{
 const base={X_OAUTH2_CLIENT_ID:'fixture-client',X_OAUTH2_TOKEN_JSON:token()};
 await assert.rejects(ensureXOAuth2AccessToken({...base},{now:()=>instant+6960000,fetcher:async()=>new Response('{}',{status:400}),persist:async()=>{}}),{code:'PROVIDER_CREDENTIAL_REVOKED'});
 await assert.rejects(ensureXOAuth2AccessToken({...base},{now:()=>instant+6960000,fetcher:async()=>new Response(JSON.stringify({access_token:'next',expires_in:7200}),{status:200}),persist:async()=>{throw Error('private persistence detail');}}),{code:'CREDENTIAL_PERSISTENCE_FAILED'});
});

test('X OAuth2 does not attempt renewal until its calculated window',async()=>{
 const env={X_OAUTH2_CLIENT_ID:'fixture-client',X_OAUTH2_TOKEN_JSON:token()};let calls=0;
 assert.equal(await ensureXOAuth2AccessToken(env,{now:()=>instant,fetcher:async()=>{calls++;throw Error('unexpected');}}),'fixture-access');assert.equal(calls,0);
});

test('invalid or extreme local expiry data fails closed without throwing diagnostics',()=>{
 assert.equal(xOAuth2TokenStatus({X_OAUTH2_CLIENT_ID:'fixture',X_OAUTH2_TOKEN_JSON:token({expires_at:9e15})},instant),'reauthorization-required');
 assert.equal(xOAuth2TokenStatus({X_OAUTH2_CLIENT_ID:'fixture',X_OAUTH2_TOKEN_JSON:token({expires_in:Number.MAX_SAFE_INTEGER})},instant),'reauthorization-required');
});
