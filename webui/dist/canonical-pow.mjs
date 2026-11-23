// The node supplies the actual canonical header, including protocol activation.
// Only its final uint64 nonce changes; all arithmetic remains exact with BigInt.
export function miningInput(job, nonce) {
  if (job.protocol !== 'QC_WEB_BLOCK_POW_V1' || job.nonce_encoding !== 'uint64-big-endian') throw Error('Unsupported mining protocol');
  if (!/^(?:[0-9a-f]{2})+$/i.test(job.header)) throw Error('Invalid header');
  const bytes = Uint8Array.from(job.header.match(/../g), x => parseInt(x, 16));
  if (job.nonce_offset !== bytes.length - 8) throw Error('Invalid nonce offset');
  const value = BigInt(nonce);
  if (value < 0n || value > BigInt(job.max_nonce) || value > 9223372036854775807n) throw Error('Nonce out of range');
  new DataView(bytes.buffer).setBigUint64(job.nonce_offset, value, false);
  return bytes;
}

export function meetsDifficulty(hash, bits) {
  if (!Number.isInteger(bits) || bits < 0 || bits > 255) throw Error('Invalid difficulty');
  const full = Math.floor(bits / 8), remainder = bits % 8;
  for (let i = 0; i < full; i++) if (hash[i] !== 0) return false;
  return remainder === 0 || (hash[full] >>> (8 - remainder)) === 0;
}

export async function hashMiningNonce(job, nonce) {
  return new Uint8Array(await crypto.subtle.digest('SHA-256', miningInput(job, nonce)));
}
