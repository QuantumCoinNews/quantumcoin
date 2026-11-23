import {Topic,PublisherError} from './types.js';
export const editorialSources={
 'wallet-custody':{fact:'Bitcoin wallets use private keys to sign spending transactions. Signing and network broadcast can be separated.',url:'https://developer.bitcoin.org/devguide/wallets.html'},
 'pow-header':{fact:'Bitcoin proof of work hashes a block header against a target. The header links to the previous block.',url:'https://developer.bitcoin.org/devguide/block_chain.html'}
} as const;
export const projectStatus={version:'QC_EDITORIAL_STATUS_V1',authority:'user-directed development scope',staking:'not-announced-live',nft:'final-assets-pending',tokenFactory:'planned',mainnetPayouts:'deferred'} as const;
// Curated technical references, not a live-news feed or proof that an LLM's
// paraphrase is correct. News and project release announcements stay disabled.
export function citeSources(topic:Topic,ids:unknown):void{
 if(!Array.isArray(ids)||ids.length<1||ids.length>2||new Set(ids).size!==ids.length||ids.some(id=>typeof id!=='string'||!Object.hasOwn(editorialSources,id)))throw new PublisherError('UNVERIFIED_SOURCE_REFERENCE');
 const urls=ids.map(id=>editorialSources[id as keyof typeof editorialSources].url);
 topic.youtube_description=(topic.youtube_description||'')+'\nTechnical references:\n'+urls.join('\n');
}
