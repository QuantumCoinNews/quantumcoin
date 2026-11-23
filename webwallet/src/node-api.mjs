const networkLabels = {
  testnet: ['TESTNET'],
  mainnet: ['MAINNET'],
  'local-release': ['LOCAL RELEASE', 'ISOLATED LOCAL RELEASE'],
}

export async function readNetworkLabel() {
  try {
    const response = await fetch('/api/network/label', {
      cache: 'no-store', credentials: 'omit', redirect: 'error',
    })
    if (response.ok) {
      const { version, network, label, genesis_hash: genesisHash } = await response.json()
      if (version === 'QC_TESTNET_LABEL_UI_API_V1' && Object.hasOwn(networkLabels, network) &&
          networkLabels[network].includes(label) && /^[0-9a-f]{64}$/.test(genesisHash)) {
        return label
      }
    }
  } catch {
    // The initial label remains unverified if the node cannot be reached.
  }
  return 'UNVERIFIED'
}

/** @param {string} address @param {string} token */
export async function readBalance(address, token) {
  if (!globalThis.isSecureContext) throw new Error('Open the viewer on HTTPS or localhost.')
  address = address.trim()
  token = token.trim()
  if (!/^[1-9A-HJ-NP-Za-km-z]{1,64}$/.test(address)) throw new Error('Enter a QuantumCoin address.')
  if (!token) throw new Error('Enter the node API token.')

  let response
  try {
    response = await fetch(`/api/wallet/balance/${encodeURIComponent(address)}`, {
      headers: { 'X-QC-API-KEY': token },
      cache: 'no-store',
      credentials: 'omit',
      redirect: 'error',
    })
  } catch {
    throw new Error('Could not reach the node.')
  }
  if (response.status === 401 || response.status === 403) throw new Error('Node access denied.')
  if (response.status === 400) throw new Error('The node rejected this wallet address.')
  if (!response.ok) throw new Error('The node could not return the balance.')

  const balance = await response.json().catch(() => null)
  if (!balance || !['balanceQc', 'spendableQc'].every(field =>
    typeof balance[field] === 'string' && /^\d+(?:\.\d{1,6})?$/.test(balance[field])) ||
    !Number.isSafeInteger(balance.height) || balance.height < 0) {
    throw new Error('The node returned an invalid balance.')
  }
  return { balanceQc: balance.balanceQc, spendableQc: balance.spendableQc, height: balance.height }
}
