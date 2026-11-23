import { PublisherError } from '../runtime/types.js';
// Not implemented. A missing integration must never produce a success receipt.
export async function publishInstagram():Promise<never>{throw new PublisherError('PLATFORM_NOT_IMPLEMENTED');}
