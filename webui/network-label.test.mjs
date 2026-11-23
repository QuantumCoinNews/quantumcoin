import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';

const asset = name => new URL(`./dist/${name}`, import.meta.url);
const html = readFileSync(asset('index.html'), 'utf8');
const script = readFileSync(asset('network-label.mjs'), 'utf8');

test('embedded landing shows TESTNET only for verified testnet identity', async () => {
  assert.match(html, /<span id="qc-testnet-badge-v1">UNVERIFIED<\/span>/);
  assert.match(html, /src="\/network-label\.mjs"/);
  const identity = {
    version: 'QC_TESTNET_LABEL_UI_API_V1',
    network: 'testnet',
    label: 'TESTNET',
    genesis_hash: 'a'.repeat(64),
  };
  for (const [body, expected] of [
    [identity, 'TESTNET'],
    [{ ...identity, network: 'mainnet', label: 'MAINNET' }, 'MAINNET'],
    [{ ...identity, network: 'mainnet' }, 'UNVERIFIED'],
    [{ ...identity, genesis_hash: '' }, 'UNVERIFIED'],
    [null, 'UNVERIFIED'],
  ]) {
    const badge = { textContent: 'UNVERIFIED' };
    const document = {
      querySelector: () => badge,
      documentElement: { dataset: {} },
      title: 'QuantumCoin Node',
    };
    const fetch = async (path, options) => {
      assert.equal(path, '/api/network/label');
      assert.equal(options.credentials, 'omit');
      if (body === null) throw new Error('offline');
      return Response.json(body);
    };
    await runInNewContext(`(async () => { ${script} })()`, { document, fetch });
    assert.equal(badge.textContent, expected);
    assert.equal(document.documentElement.dataset.qcNetwork, expected === 'UNVERIFIED' ? undefined : body.network);
  }
});
