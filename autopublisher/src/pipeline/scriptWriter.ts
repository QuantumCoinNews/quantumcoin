import {PublisherError} from '../runtime/types.js';
// Use the canonical runtime; no silent placeholder content.
export async function writeScript(opts:{id:string,dir:string,topic:string}):Promise<string> {throw new PublisherError('LEGACY_PIPELINE_RETIRED');}
