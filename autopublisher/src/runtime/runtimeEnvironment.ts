import {readFileSync} from 'node:fs';
import path from 'node:path';
import {isIP} from 'node:net';

type Entry={['x-kind']:string;['x-owners']:string[];['x-status']:string;['x-case']?:string;enum?:string[];minimum?:number;maximum?:number};
type Schema={['x-version']:number;['x-services']:string[];properties:Record<string,Entry>;['x-networkKeys']:string[];['x-aliasGroups']:{keys:string[];owners:string[];normalize:string}[]};
const truth=new Set(['1','true','yes','on']);
const booleans=new Set([...truth,'0','false','no','off']);
const modes:Record<string,string>={local:'local','local-release':'local',testnet:'testnet','testnet-local':'testnet','testnet-public':'testnet',mainnet:'mainnet'};
const gameModes:Record<string,string>={...modes,main:'mainnet',production:'mainnet',prod:'mainnet'};
const gameMode=(v:string)=>['TESTNET','SIMULATED','TESTNET_SIM','TESTNET_SIMULATED'].includes(v.toUpperCase())?'simulated':['OFF','DISABLE','DISABLED','NONE'].includes(v.toUpperCase())?'disabled':v.toUpperCase();
function port(v:string):number|undefined{const raw=v.replace(/^:/,'');const n=Number(raw);return /^[0-9]+$/.test(raw)&&Number.isInteger(n)&&n>=1&&n<=65535?n:undefined;}
function loopback(host:string):boolean{host=host.replace(/^\[|\]$/g,'').toLowerCase();if(isIP(host)===6){try{return new URL('http://['+host+']/').hostname==='[::1]';}catch{return false;}}return host==='localhost'||(isIP(host)===4&&host.startsWith('127.'));}
function origin(value:string,monitor=false):boolean{
 try{const u=new URL(value);const rawHost=value.match(/^[^:]+:\/\/(\[[^\]]+\]|[^/:?#]+)/)?.[1]||'';
  if(!rawHost||u.username||u.password||u.port==='0'||(u.pathname&&u.pathname!=='/')||u.search||u.hash||/\s/.test(value))return false;
  if(monitor)return u.protocol==='http:'&&['127.0.0.1','[::1]','::1'].includes(rawHost);
  return u.protocol==='https:'||(u.protocol==='http:'&&loopback(rawHost));
 }catch{return false;}
}
function bind(value:string):{host:string;port:number}|undefined{
 const split=value.lastIndexOf(':');if(split<0)return;
 const host=value.slice(0,split);const n=port(value.slice(split+1));if(n===undefined)return;
 const bracket=host.startsWith('[')&&host.endsWith(']');const raw=bracket?host.slice(1,-1):host;
 if(bracket&&isIP(raw)!==6)return;
 if(bracket&&new URL('http://['+raw+']/').hostname.toLowerCase().startsWith('[::ffff:'))return;
 if(raw&&raw.toLowerCase()!=='localhost'&&(isIP(raw)===0||(isIP(raw)===6)!==bracket))return;
 return {host:raw,port:n};
}

/** Read the same repository-owned schema embedded in Go and used by Python.
 * No dotenv loading, process mutation, credential output or provider call. */
export function validateEnvironment(env:NodeJS.ProcessEnv,service='publisher',strictUnknown=true):Record<string,string>{
 const schema=JSON.parse(readFileSync(new URL('../../../config/runtime-environment.schema.json',import.meta.url),'utf8')) as Schema;
 const errors:Record<string,string>={};
 if(schema['x-version']!==1||!schema['x-services'].includes(service))return {CONFIG_SCHEMA:'unsupported-schema-or-service'};
 const get=(key:string)=>{const v=env[key];if(v===undefined)return '';if(typeof v!=='string'){errors[key]='expected-string';return '';}return v.trim();};
 for(const [key,entry] of Object.entries(schema.properties)){
  if(env[key]===undefined||(service!=='tooling'&&!entry['x-owners'].includes(service)))continue;
  const v=get(key);if(!v)continue;
  if(entry['x-status']==='retired'){errors[key]='retired-setting';continue;}
  let valid=true;
  switch(entry['x-kind']){
   case 'boolean':valid=booleans.has(v.toLowerCase());break;
   case 'boolean-01':valid=env[key]==='0'||env[key]==='1';break;
   case 'integer':{const n=Number(v);valid=/^-?[0-9]+$/.test(v)&&Number.isSafeInteger(n)&&n>=entry.minimum!&&n<=entry.maximum!;break;}
   case 'port':valid=port(v)!==undefined;break;
   case 'bind':valid=bind(v)!==undefined;break;
   case 'origin':valid=origin(v);break;
   case 'monitor-origin':valid=origin(v,true);break;
   case 'absolute-path':valid=path.posix.isAbsolute(v)||/^[A-Za-z]:[\\/]/.test(v)||/^\\\\[^\\/]+[\\/][^\\/]+/.test(v);break;
   case 'network':valid=v.toLowerCase() in modes;break;
   case 'game-network':valid=v.toLowerCase() in gameModes;break;
   case 'enum':valid=entry.enum!.includes(entry['x-case']==='upper'?v.toUpperCase():v);break;
   case 'platforms':{const names=v.split(',').map(n=>n.trim());valid=new Set(names).size===names.length&&names.every(n=>['youtube','x','telegram','instagram','tiktok'].includes(n));break;}
   case 'timezone':valid=v==='UTC'||/^[A-Za-z_]+(?:\/[A-Za-z_+0-9-]+){1,2}$/.test(v);break;
   default:valid=entry['x-kind']==='text';break;
  }
  if(!valid)errors[key]='invalid-'+entry['x-kind'];
 }
 if(strictUnknown)for(const key of Object.keys(env)){if(key.toUpperCase().startsWith('QC_')&&key!==key.toUpperCase())errors[key]='noncanonical-key-case';else if(key.startsWith('QC_')&&!(key in schema.properties)&&!key.startsWith('QC_TEST_'))errors[key]='unknown-setting';}
 for(const group of schema['x-aliasGroups']){
  if(service!=='tooling'&&!group.owners.includes(service))continue;
  const keys=group.keys.filter(k=>get(k));const values=keys.map(k=>group.normalize==='origin'?get(k).replace(/\/+$/,''):group.normalize==='game-mode'?gameMode(get(k)):group.normalize==='port'&&port(get(k))!==undefined?String(port(get(k))):get(k));
  if(new Set(values).size>1)for(const key of keys)errors[key]='conflicting-alias';
 }
 if(['node','game','tooling'].includes(service)){
  const selected:Record<string,string>={};
  for(const key of schema['x-networkKeys']){const v=get(key).toLowerCase();if(!v)continue;const mode=(key.startsWith('TELEGRAM_GAME_')?gameModes:modes)[v];if(mode)selected[key]=mode;else errors[key]='invalid-network';}
  for(const key of ['QC_TESTNET','QC_PUBLIC_TESTNET','QC_API_PUBLIC_TESTNET','QC_TESTNET_PUBLIC_API','QC_MAINNET'])if(truth.has(get(key).toLowerCase()))selected[key]=key==='QC_MAINNET'?'mainnet':'testnet';
  if(get('QC_NET_MAGIC')){const mode:Record<string,string>={QC_LOCAL_RELEASE_V1:'local',QC_TESTNET_V1:'testnet',QC_MAINNET_V1:'mainnet'};if(mode[get('QC_NET_MAGIC').toUpperCase()])selected.QC_NET_MAGIC=mode[get('QC_NET_MAGIC').toUpperCase()];else errors.QC_NET_MAGIC='invalid-enum';}
  const networks=new Set(Object.values(selected));
  if(networks.size>1)for(const key of Object.keys(selected))errors[key]='conflicting-network';
  if(networks.size===1)for(const key of ['QC_API_BASE','QUANTUMCOIN_API_BASE','QC_NODE_API']){if(!get(key)||errors[key])continue;try{const p=new URL(get(key)).port;const testnet=networks.has('testnet');if((testnet&&['8081','8082'].includes(p)&&!truth.has(get('TELEGRAM_GAME_ALLOW_TESTNET_MAIN_PORT').toLowerCase()))||(!testnet&&p==='18082'))errors[key]='network-port-conflict';}catch{/* shape error already recorded */}}
 }
 if(['node','tooling'].includes(service)&&get('QC_API_LISTEN')&&!errors.QC_API_LISTEN){const value=bind(get('QC_API_LISTEN'))!;if(!loopback(value.host)&&!get('QC_API_TOKEN'))errors.QC_API_TOKEN='required-for-non-loopback-bind';if(!loopback(value.host)&&/^(?:__SET_|YOUR_|CHANGE_ME)/i.test(get('QC_API_TOKEN')))errors.QC_API_TOKEN='placeholder-credential';if(get('QC_HTTP_PORT')&&!errors.QC_HTTP_PORT&&value.port!==port(get('QC_HTTP_PORT')))errors.QC_API_LISTEN='http-port-conflict';}
 if(['publisher','tooling'].includes(service)&&get('QC_PUBLISHER_MONITOR_ORIGIN')&&!get('QC_PUBLISHER_MONITOR_TOKEN'))errors.QC_PUBLISHER_MONITOR_TOKEN='required-with-monitor-origin';
 return Object.fromEntries(Object.entries(errors).sort(([a],[b])=>a.localeCompare(b)));
}
