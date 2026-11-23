import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import { miningInput, hashMiningNonce, meetsDifficulty } from './dist/canonical-pow.mjs';

test('JavaScript matches Go canonical V1 and NFT V4 headers at all nonce boundaries', async () => {
  const vectors = JSON.parse(fs.readFileSync(process.env.QC_WEB_MINING_VECTOR_FILE, 'utf8'));
  assert.equal(vectors.length, 6);
  for (const vector of vectors) {
    const job = { ...vector, protocol: 'QC_WEB_BLOCK_POW_V1' };
    const hash = await hashMiningNonce(job, vector.nonce);
    assert.equal(Buffer.from(hash).toString('hex'), vector.hash);
  }
});
test('difficulty boundary and invalid protocol or nonce fail closed', () => {
  assert.equal(meetsDifficulty(Uint8Array.from([0, 127]), 9), true);
  assert.equal(meetsDifficulty(Uint8Array.from([0, 128]), 9), false);
  assert.equal(meetsDifficulty(Uint8Array.from([1]), 8), false);
  assert.throws(() => meetsDifficulty(new Uint8Array(32), -1));
  const job={protocol:'QC_WEB_BLOCK_POW_V1',nonce_encoding:'uint64-big-endian',header:'00'.repeat(88),nonce_offset:80,max_nonce:'9223372036854775807'};
  assert.throws(()=>miningInput({...job,protocol:'fake-challenge'},0n));
  assert.throws(()=>miningInput(job,9223372036854775808n));
  assert.throws(()=>miningInput(job,-1n));
});
