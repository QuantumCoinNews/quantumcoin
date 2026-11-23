import React, { useEffect, useRef, useState } from 'react'
import { readBalance, readNetworkLabel } from './node-api.mjs'

export default function App() {
  const tokenInput = useRef<HTMLInputElement>(null)
  const [network, setNetwork] = useState('UNVERIFIED')
  const [result, setResult] = useState<Awaited<ReturnType<typeof readBalance>> | null>(null)
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)

  useEffect(() => {
    readNetworkLabel().then(setNetwork)
  }, [])

  async function fetchBalance(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const input = tokenInput.current!
    const token = input.value
    input.value = ''
    const address = String(new FormData(event.currentTarget).get('address') ?? '')
    setPending(true)
    setResult(null)
    setError('')
    try {
      setResult(await readBalance(address, token))
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : 'Balance lookup failed.')
    } finally {
      setPending(false)
    }
  }

  return (
    <main style={{ margin: '3rem auto', padding: '0 1rem', maxWidth: '36rem', fontFamily: 'system-ui' }}>
      <h1>QuantumCoin Balance Viewer</h1>
      <p>Node network <strong role="status">{network}</strong></p>
      <form autoComplete="off" onSubmit={fetchBalance}>
        <p><label>Wallet address <input name="address" required /></label></p>
        <p><label>Node API token <input ref={tokenInput} type="password" autoComplete="off" required /></label></p>
        <button disabled={pending}>{pending ? 'Looking up…' : 'Get balance'}</button>
      </form>
      {error && <p role="alert">{error}</p>}
      {result && <dl>
        <dt>Total balance</dt><dd>{result.balanceQc} QC</dd>
        <dt>Spendable</dt><dd>{result.spendableQc} QC</dd>
        <dt>Block height</dt><dd>{result.height}</dd>
      </dl>}
      <p>Use the native wallet or CLI to create keys and sign transactions locally.</p>
    </main>
  )
}
