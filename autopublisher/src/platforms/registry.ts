import { Provider, Platform, PublisherError, PublishInput, Receipt } from '../runtime/types.js';
import { publishYouTube, youtubeReadiness } from './youtube.js';
import { publishX, xReadiness } from './x.js';
import { publishTelegram, telegramReadiness } from './telegram.js';

const unavailable: Provider={readiness:()=> 'NOT_IMPLEMENTED',async publish(){throw new PublisherError('PLATFORM_NOT_IMPLEMENTED');}};
export const providers: Record<Platform, Provider>={youtube:{readiness:youtubeReadiness,publish:publishYouTube},x:{readiness:xReadiness,publish:publishX},telegram:{readiness:telegramReadiness,publish:publishTelegram},instagram:unavailable,tiktok:unavailable};

// Only explicit API rejections known not to have delivered may be retried.
// Timeouts, connection loss, 5xx and a crash after submission require review.
export function deliveryError(error: unknown): string {
  if(error instanceof PublisherError && ['PROVIDER_CONFIGURATION_INVALID','PROVIDER_CREDENTIAL_REVOKED'].includes(error.code))return 'PROVIDER_REJECTED';
  const code=providerStatus(error);
  if(code===429)return 'RATE_LIMITED';
  if(code!==undefined && [400,401,402,403,404,422].includes(code))return 'PROVIDER_REJECTED';
  return 'DELIVERY_UNCERTAIN';
}
export type Publisher = (input: PublishInput,env:NodeJS.ProcessEnv)=>Promise<Receipt>;

export function providerStatus(error:unknown):number|undefined {
 if(error instanceof PublisherError&&error.code==='PROVIDER_CREDENTIAL_REVOKED')return 401;
 const v=error as {code?:unknown;response?:{status?:unknown;statusCode?:unknown}};
 const code=Number(v?.response?.status??v?.response?.statusCode??v?.code);
 return Number.isInteger(code)&&code>=100&&code<=599?code:undefined;
}
