import {PublisherError} from '../runtime/types.js';
// Use the canonical runtime; no silent placeholder content.
export async function generateImages(opts:{id:string,dir:string,topic:string,count:number}):Promise<string[]> {throw new PublisherError('LEGACY_PIPELINE_RETIRED');}
