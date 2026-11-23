try {
  const response = await fetch('/api/network/label', {
    cache: 'no-store',
    credentials: 'omit',
    redirect: 'error'
  });
  if (response.ok) {
    const network = await response.json();
    const labels = {
      testnet: ['TESTNET'],
      mainnet: ['MAINNET'],
      'local-release': ['LOCAL RELEASE', 'ISOLATED LOCAL RELEASE'],
    };
    if (network.version === 'QC_TESTNET_LABEL_UI_API_V1' &&
        Object.hasOwn(labels, network.network) && labels[network.network].includes(network.label) &&
        /^[0-9a-f]{64}$/.test(network.genesis_hash)) {
      const badge = document.querySelector('#qc-testnet-badge-v1');
      badge.textContent = network.label;
      document.documentElement.dataset.qcNetwork = network.network;
      document.title = network.label + ' - ' + document.title;
    }
  }
} catch {
  // The landing page remains usable if network discovery is unavailable.
}
