import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {createXOAuth2AuthorizationRequest,exchangeXOAuth2AuthorizationCode,readXOAuth2Callback} from '../dist/runtime/xOAuth2Authorization.js';

test('X authorization uses loopback callback, state and S256 PKCE with minimum scopes',()=>{
 const request=createXOAuth2AuthorizationRequest('public-client-id','http://127.0.0.1:8765/callback',size=>Buffer.alloc(size,7));
 const url=new URL(request.authorizationUrl);
 assert.equal(url.origin+url.pathname,'https://x.com/i/oauth2/authorize');
 assert.equal(url.searchParams.get('client_id'),'public-client-id');
 assert.equal(url.searchParams.get('redirect_uri'),request.redirectUri);
 assert.equal(url.searchParams.get('scope'),'tweet.write users.read offline.access');
 assert.equal(url.searchParams.get('state'),request.state);
 assert.equal(url.searchParams.get('code_challenge_method'),'S256');
 assert.equal(url.searchParams.get('code_challenge'),createHash('sha256').update(request.codeVerifier).digest('base64url'));
});

test('X authorization rejects non-loopback or malformed callback configuration',()=>{
 assert.throws(()=>createXOAuth2AuthorizationRequest('client','https://example.com/callback'),{code:'PROVIDER_CONFIGURATION_INVALID'});
 assert.throws(()=>createXOAuth2AuthorizationRequest('client','http://0.0.0.0:8765/callback'),{code:'PROVIDER_CONFIGURATION_INVALID'});
 assert.throws(()=>createXOAuth2AuthorizationRequest('client','http://127.0.0.1:8765/other'),{code:'PROVIDER_CONFIGURATION_INVALID'});
});

test('X callback checks exact local route and CSRF state before accepting the code',()=>{
 const request=createXOAuth2AuthorizationRequest('client','http://127.0.0.1:8765/callback',size=>Buffer.alloc(size,3));
 assert.equal(readXOAuth2Callback(`http://127.0.0.1:8765/callback?code=one-time-code&state=${request.state}`,request),'one-time-code');
 assert.throws(()=>readXOAuth2Callback('http://127.0.0.1:8765/callback?code=one-time-code&state=wrong',request),{code:'PROVIDER_AUTHORIZATION_STATE_MISMATCH'});
 assert.throws(()=>readXOAuth2Callback(`http://127.0.0.1:8765/other?code=one-time-code&state=${request.state}`,request),{code:'PROVIDER_AUTHORIZATION_CALLBACK_INVALID'});
});

test('X authorization code exchange uses PKCE, requires offline refresh scope and never returns provider errors',async()=>{
 const request=createXOAuth2AuthorizationRequest('client','http://127.0.0.1:8765/callback',size=>Buffer.alloc(size,5));let captured;
 const tokens=await exchangeXOAuth2AuthorizationCode('client','one-time-code',request,async(url,init)=>{
  captured={url,init};return new Response(JSON.stringify({access_token:'access-fixture',refresh_token:'refresh-fixture',expires_in:7200,scope:'tweet.write users.read offline.access',token_type:'bearer'}),{status:200});
 });
 assert.equal(captured.url,'https://api.x.com/2/oauth2/token');
 assert.equal(captured.init.method,'POST');
 const form=new URLSearchParams(captured.init.body);
 assert.equal(form.get('grant_type'),'authorization_code');assert.equal(form.get('client_id'),'client');assert.equal(form.get('code'),'one-time-code');assert.equal(form.get('redirect_uri'),request.redirectUri);assert.equal(form.get('code_verifier'),request.codeVerifier);
 assert.equal(tokens.access_token,'access-fixture');assert.equal(tokens.refresh_token,'refresh-fixture');assert.equal(tokens.expires_in,7200);assert.equal(tokens.scope,'tweet.write users.read offline.access');
 await assert.rejects(exchangeXOAuth2AuthorizationCode('client','code',request,async()=>new Response(JSON.stringify({access_token:'access-fixture',expires_in:7200,scope:'tweet.write users.read'}),{status:200})),{code:'PROVIDER_AUTHORIZATION_SCOPE_OR_TOKEN_INVALID'});
 await assert.rejects(exchangeXOAuth2AuthorizationCode('client','code',request,async()=>new Response('secret provider body',{status:401})),error=>error.code==='PROVIDER_AUTHORIZATION_EXCHANGE_FAILED'&&!String(error).includes('secret provider body'));
});
