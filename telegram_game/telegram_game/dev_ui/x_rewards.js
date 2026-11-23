/* Telegram, YouTube and X task verification. Rewards stay off-chain until user-paid BSC claim. */
(function () {
  "use strict";
  let pending = null;
  async function post(path, body) {
    const response=await fetch(path,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)});
    const data=await response.json();
    if(!response.ok||!data.ok)throw new Error(data.error||"verification_failed");
    return data;
  }
  function cleanup(task){
    clearTimeout(task.timer);
    window.removeEventListener("message",task.onMessage);
    if(pending===task)pending=null;
  }
  function cancel(){
    const task=pending;if(!task)return;
    task.cancelled=true;
    if(task.cancel)task.cancel();else{cleanup(task);task.popup.close();}
  }
  window.QCCancelSocialRewardVerification=cancel;
  window.QCCancelXVerification=cancel;
  async function completeReward(result){
    if(typeof window.QC_refreshWalletFromBackend==="function")await window.QC_refreshWalletFromBackend();
    window.dispatchEvent(new CustomEvent("qc:verified-reward",{detail:result}));
    return result;
  }
  async function oauth(provider,campaignId){
    if(pending)throw new Error("social_verification_already_pending");
    const isX=provider==="X";
    const popup=window.open("about:blank",isX?"qc_x_reward_verification":"qc_youtube_reward_verification","width=620,height=720");
    if(!popup)throw new Error("social_sign_in_popup_blocked");
    const task={popup};pending=task;
    const startPath=isX?"/api/social/x/start":"/api/social/youtube/start";
    const completePath=isX?"/api/social/x/complete":"/api/social/youtube/complete";
    const messageType=isX?"qc:x-oauth":"qc:youtube-oauth";
    try{
      await window.QC_TELEGRAM_IDENTITY_READY;
      const playerId=window.QCGetPlayerId();
      if(!playerId)throw new Error("telegram_session_required");
      const started=await post(startPath,{playerId,campaignId});
      if(task.cancelled)throw new Error("verification_cancelled");
      if(started.alreadyCredited){pending=null;popup.close();return started;}
      const url=new URL(started.authorizeUrl);
      const valid=isX?(url.origin==="https://x.com"&&url.pathname==="/i/oauth2/authorize"):
        (url.origin==="https://accounts.google.com"&&url.pathname==="/o/oauth2/v2/auth");
      if(!valid)throw new Error("social_authorize_origin_mismatch");
      return await new Promise(function(resolve,reject){
        let completing=false;
        task.cancel=function(){cleanup(task);popup.close();reject(new Error("verification_cancelled"));};
        task.onMessage=async function(event){
          const data=event.data;
          if(completing||event.origin!==window.location.origin||event.source!==popup||!data||data.type!==messageType||data.state!==started.state)return;
          completing=true;clearTimeout(task.timer);
          if(data.error||!data.code){cleanup(task);popup.close();reject(new Error("social_sign_in_not_completed"));return;}
          try{
            const result=await post(completePath,{playerId,state:started.state,code:data.code});
            await completeReward(result);resolve(result);
          }catch(error){reject(error);}finally{cleanup(task);popup.close();}
        };
        window.addEventListener("message",task.onMessage);
        task.timer=setTimeout(function(){cleanup(task);popup.close();reject(new Error("social_sign_in_expired"));},600000);
        popup.location.replace(url.toString());
      });
    }catch(error){if(pending===task)pending=null;popup.close();throw error;}
  }
  window.QCVerifyXCampaign=function(campaignId){return oauth("X",campaignId);};
  window.QCVerifyYouTubeCampaign=function(campaignId){return oauth("YOUTUBE",campaignId);};
  window.QCVerifyTelegramCampaign=async function(campaignId){
    await window.QC_TELEGRAM_IDENTITY_READY;
    const playerId=window.QCGetPlayerId();
    if(!playerId)throw new Error("telegram_session_required");
    const result=await post("/api/social/telegram/verify",{playerId,campaignId});
    await completeReward(result);
    return result;
  };

  function campaignURL(policy){
    let raw=policy.url||"";
    if(!raw&&policy.provider==="X"&&/^[1-9][0-9]{0,24}$/.test(policy.target))raw="https://x.com/i/user/"+encodeURIComponent(policy.target);
    if(!raw&&policy.provider==="YOUTUBE")raw=policy.target.startsWith("UC")?"https://www.youtube.com/channel/"+policy.target:"https://www.youtube.com/"+policy.target;
    if(!raw&&policy.provider==="TELEGRAM"&&policy.target.startsWith("@"))raw="https://t.me/"+policy.target.slice(1);
    try{
      const url=new URL(raw);
      const hosts={X:["x.com","www.x.com","twitter.com","www.twitter.com"],TELEGRAM:["t.me","www.t.me","web.telegram.org"],YOUTUBE:["youtube.com","www.youtube.com","youtu.be"]};
      return url.protocol==="https:"&&(hosts[policy.provider]||[]).includes(url.hostname)&&!url.username&&!url.password?url.href:"";
    }catch(_){return "";}
  }
  async function renderCampaigns(){
    await window.QC_TELEGRAM_IDENTITY_READY;
    if(!window.QCGetPlayerId())return;
    try{
      const response=await fetch("/api/tgwt/campaigns");
      const data=await response.json();
      const mount=document.getElementById("socialLinksList")||document.getElementById("socialPanel");
      if(!mount||document.getElementById("verified-social-campaigns"))return;
      const box=document.createElement("div");box.id="verified-social-campaigns";
      for(const item of data.campaigns||[]){
        const policy=item.policy||{};
        if(!["X","TELEGRAM","YOUTUBE"].includes(policy.provider))continue;
        const href=campaignURL(policy);if(!href)continue;
        const row=document.createElement("div");row.className="social-task-row";
        const link=document.createElement("a");link.href=href;link.target="_blank";link.rel="noopener noreferrer";
        link.textContent=policy.provider==="X"?"Open X task":policy.provider==="YOUTUBE"?"Open YouTube channel":"Open Telegram channel";
        const button=document.createElement("button");button.type="button";button.className="wallet-btn";
        const action=policy.action;
        button.textContent=(policy.provider==="X"?(action==="SOCIAL_LIKE"?"Verify X like":"Verify X follow"):
          policy.provider==="YOUTUBE"?"Verify YouTube subscription":"Verify Telegram membership")+" · 1 TGWT";
        button.addEventListener("click",async function(){
          button.disabled=true;
          try{
            const verify=policy.provider==="X"?window.QCVerifyXCampaign:
              policy.provider==="YOUTUBE"?window.QCVerifyYouTubeCampaign:window.QCVerifyTelegramCampaign;
            const result=await verify(policy.campaignId);
            button.textContent=result.alreadyCredited?"Already rewarded":"1 TGWT credited";
          }catch(_){button.textContent="Verification unavailable";button.disabled=false;}
        });
        row.append(link,button);box.appendChild(row);
      }
      if(box.childNodes.length)mount.appendChild(box);
    }catch(_){/* No live campaign or provider setup: show no optimistic reward button. */}
  }
  document.addEventListener("DOMContentLoaded",renderCampaigns);
})();
