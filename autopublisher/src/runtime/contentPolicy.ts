import {editorialSources} from './sources.js';
import { PublisherError, Topic } from './types.js';

// This publisher currently has no verified live-news feed. Evergreen mode
// must not silently present generated market data or release promises as news.
export function validatePublication(topic:Topic):void {
 const fields=[topic.title,topic.script,topic.telegram,topic.x,topic.youtube_description].filter(Boolean).join('\n');
 if(/\b(breaking news|today['’]s (price|news|market)|current (price|market cap)|just launched|launches on|guaranteed (profit|return|reward)s?)\b/i.test(fields))throw new PublisherError('UNVERIFIED_TIMELY_CLAIM');
 const allowed=new Set(Object.values(editorialSources).map(s=>s.url as string));
 for(const link of fields.match(/https?:\/\/[^\s]+/g)||[])if(!allowed.has(link))throw new PublisherError('UNVERIFIED_SOURCE_LINK');
 const sentences=[fields];
 for(const sentence of sentences){
  if(/\bQuantum[\s-]*Coin\b/i.test(sentence) && /\b(staking|NFTs?|token factory|mainnet|payouts?)\b/i.test(sentence) && /\b(live|available|launched|released|ready|enabled|earn|claim|receive)\b/i.test(sentence))throw new PublisherError('UNVERIFIED_PROJECT_RELEASE_CLAIM');
 }
 if(/https?:\/\//i.test(topic.script))throw new PublisherError('UNREVIEWED_NARRATION_LINK');
}
