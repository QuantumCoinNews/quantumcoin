import { hashMiningNonce, meetsDifficulty } from './canonical-pow.mjs';

self.onmessage = async ({ data: job }) => {
  try {
    for (let nonce = 0n; nonce <= BigInt(job.max_nonce); nonce++) {
      if (Date.now() >= job.expires * 1000) throw Error('Job expired; request new work');
      const hash = await hashMiningNonce(job, nonce);
      if (meetsDifficulty(hash, job.difficulty)) {
        self.postMessage({ nonce: nonce.toString(), hash: Array.from(hash, x => x.toString(16).padStart(2, '0')).join('') });
        return;
      }
      if (nonce % 256n === 0n) self.postMessage({ hashes: (nonce + 1n).toString() });
    }
    throw Error('Nonce range exhausted');
  } catch (error) { self.postMessage({ error: error.message }); }
};
