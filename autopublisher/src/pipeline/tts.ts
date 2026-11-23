import {PublisherError} from '../runtime/types.js';
// Use the canonical runtime; no silent placeholder content.
export async function generateVoice(opts:{id:string,dir:string,scriptPath:string}):Promise<string> {throw new PublisherError('LEGACY_PIPELINE_RETIRED');}
