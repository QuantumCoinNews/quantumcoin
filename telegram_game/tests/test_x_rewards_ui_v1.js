const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const code=fs.readFileSync(path.resolve(__dirname,'../telegram_game/dev_ui/x_rewards.js'),'utf8');
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function browser({deny=false,already=false}={}){
 const calls=[],events=[],listeners=new Map(),timers=new Map();let refreshes=0;
 const popup={location:{replace(url){this.url=url;}},close(){this.closed=true;}};
 const window={location:{origin:'https://game.test'},open:()=>popup,QC_TELEGRAM_IDENTITY_READY:Promise.resolve(),QCGetPlayerId:()=> 'tg_9',
   addEventListener:(name,listener)=>listeners.set(name,listener),removeEventListener:name=>listeners.delete(name),
   dispatchEvent:event=>events.push(event),QC_refreshWalletFromBackend:async()=>{refreshes++;}};
 const fetch=async(url,options)=>{
  calls.push({url,body:JSON.parse(options.body)});
  const authorize=url.includes('/youtube/')?'https://accounts.google.com/o/oauth2/v2/auth?state=server-state':'https://x.com/i/oauth2/authorize?state=server-state';
  const result=url.endsWith('/start')?(already?{ok:true,alreadyCredited:true,credited_u:0}:{ok:true,state:'server-state',authorizeUrl:authorize}):
      (deny?{ok:false,error:'x_action_not_verified'}:{ok:true,credited_u:1000000,onChainPaid:false});
  return new Response(JSON.stringify(result),{status:200});
 };
 vm.runInNewContext(code,{window,fetch,URL,Response,document:{addEventListener(){}},
   setTimeout:(callback,ms)=>{timers.set(ms,callback);return ms;},clearTimeout:id=>timers.delete(id),
   CustomEvent:class{constructor(type,options){this.type=type;this.detail=options.detail;}}});
 return {window,popup,calls,listeners,timers,events,refreshes:()=>refreshes};
}
async function main(){
 let count=0;
 const b=browser(),result=b.window.QCVerifyXCampaign('x-follow');await tick();
 await assert.rejects(b.window.QCVerifyXCampaign('x-like'),/already_pending/);
 assert.match(b.popup.location.url,/^https:\/\/x\.com\/i\/oauth2\/authorize/);
 const onMessage=b.listeners.get('message');
 const data={type:'qc:x-oauth',state:'server-state',code:'test-code'};
 for(const event of [{origin:'https://evil.test',source:b.popup,data},{origin:'https://game.test',source:{},data},
   {origin:'https://game.test',source:b.popup,data:{...data,state:'forged'}}])await onMessage(event);
 assert.equal(b.calls.length,1,'forged popup must not reach completion');count++;
 await onMessage({origin:'https://game.test',source:b.popup,data});
 assert.equal((await result).credited_u,1000000);
 assert.equal(b.calls[1].url,'/api/social/x/complete');
 assert.equal(b.calls[1].body.playerId,'tg_9');
 assert.equal(b.refreshes(),1);assert.equal(b.events[0].detail.onChainPaid,false);
 assert.equal(b.listeners.size,0);assert.equal(b.timers.size,0);count++;
 const denied=browser({deny:true}),pending=denied.window.QCVerifyXCampaign('x-follow');
 const rejected=assert.rejects(pending,/x_action_not_verified/);await tick();
 await denied.listeners.get('message')({origin:'https://game.test',source:denied.popup,data});await rejected;
 assert.equal(denied.refreshes(),0);assert.equal(denied.events.length,0);count++;
 const old=browser({already:true});assert.equal((await old.window.QCVerifyXCampaign('x-follow')).alreadyCredited,true);
 assert.equal(old.calls.length,1);assert.equal(old.listeners.size,0);count++;
 const cancel=browser(),cancelled=cancel.window.QCVerifyXCampaign('x-follow');
 const cancellation=assert.rejects(cancelled,/cancelled/);await tick();cancel.window.QCCancelXVerification();await cancellation;
 assert.equal(cancel.listeners.size,0);assert.equal(cancel.timers.size,0);count++;
 const telegram=browser();const telegramResult=await telegram.window.QCVerifyTelegramCampaign('tg-join');
 assert.equal(telegram.calls[0].url,'/api/social/telegram/verify');assert.equal(telegramResult.credited_u,1000000);count++;
 const youtube=browser(),youtubeResult=youtube.window.QCVerifyYouTubeCampaign('yt-sub');await tick();
 assert.match(youtube.popup.location.url,/^https:\/\/accounts\.google\.com\/o\/oauth2\/v2\/auth/);
 const youtubeMessage={type:'qc:youtube-oauth',state:'server-state',code:'test-code'};
 await youtube.listeners.get('message')({origin:'https://evil.test',source:youtube.popup,data:youtubeMessage});
 assert.equal(youtube.calls.length,1);count++;
 await youtube.listeners.get('message')({origin:'https://game.test',source:youtube.popup,data:youtubeMessage});
 assert.equal((await youtubeResult).credited_u,1000000);
 assert.equal(youtube.calls[1].url,'/api/social/youtube/complete');assert.equal(youtube.listeners.size,0);count++;
 console.log('PASS: '+count+' X OAuth popup identity and reward UI contracts');
}
main().catch(error=>{console.error(error);process.exitCode=1;});
