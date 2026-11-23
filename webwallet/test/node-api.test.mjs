import assert from 'node:assert/strict'
import { test } from 'node:test'
import { readBalance, readNetworkLabel } from '../src/node-api.mjs'

const address = '1'.repeat(34)

test('balance uses the canonical authenticated read and preserves exact values', async t => {
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    assert.equal(url, `/api/wallet/balance/${address}`)
    assert.deepEqual(options, {
      headers: { 'X-QC-API-KEY': 'fixture-token' },
      cache: 'no-store', credentials: 'omit', redirect: 'error',
    })
    return Response.json({ balanceQc: '2000000.000001', spendableQc: '0.000001', height: 2 })
  })
  const previous = globalThis.isSecureContext
  globalThis.isSecureContext = true
  t.after(() => { globalThis.isSecureContext = previous })
  assert.deepEqual(await readBalance(address, 'fixture-token'), {
    balanceQc: '2000000.000001', spendableQc: '0.000001', height: 2,
  })
})

test('balance rejects unsafe transport, invalid input, unauthorized and legacy responses', async t => {
  const fetchMock = t.mock.method(globalThis, 'fetch', async () => new Response(null, { status: 401 }))
  const previous = globalThis.isSecureContext
  t.after(() => { globalThis.isSecureContext = previous })
  globalThis.isSecureContext = false
  await assert.rejects(readBalance(address, 'fixture-token'), /HTTPS or localhost/)
  globalThis.isSecureContext = true
  await assert.rejects(readBalance('../admin', 'fixture-token'), /wallet address|QuantumCoin address/)
  assert.equal(fetchMock.mock.callCount(), 0)
  await assert.rejects(readBalance(address, 'fixture-token'), /access denied/)
  fetchMock.mock.mockImplementation(async () => Response.json({ balance: 25 }))
  await assert.rejects(readBalance(address, 'fixture-token'), /invalid balance/)
})

test('network label requires a verified and consistent identity', async t => {
  const network = {
    version: 'QC_TESTNET_LABEL_UI_API_V1', label: 'TESTNET', network: 'testnet', genesis_hash: 'a'.repeat(64),
  }
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    assert.equal(url, '/api/network/label')
    assert.equal(options.credentials, 'omit')
    return Response.json(network)
  })
  assert.equal(await readNetworkLabel(), 'TESTNET')
  network.network = 'mainnet'
  assert.equal(await readNetworkLabel(), 'UNVERIFIED')
  network.network = 'testnet'
  network.genesis_hash = ''
  assert.equal(await readNetworkLabel(), 'UNVERIFIED')
})
