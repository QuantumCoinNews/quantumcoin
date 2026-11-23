import {PublisherError} from '../runtime/types.js';
// Use the canonical runtime; no silent placeholder content.
export async function selectTopic(opts:{id:string,promptDir:string}):Promise<string> {throw new PublisherError('LEGACY_PIPELINE_RETIRED');}
