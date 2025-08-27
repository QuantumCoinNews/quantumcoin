import React, { useState } from 'react'

export default function App() {
  const [addr, setAddr] = useState('')
  const [bal, setBal] = useState<number | null>(null)

  async function fetchBalance() {
    const r = await fetch(`/api/address/balance?addr=${encodeURIComponent(addr)}`)
    const j = await r.json()
    setBal(j.balance)
  }

  return (
    <div style={{padding:24,fontFamily:'system-ui'}}>
      <h1>QuantumCoin Wallet</h1>
      <input placeholder="Address" value={addr} onChange={e=>setAddr(e.target.value)} style={{minWidth:420}}/>
      <button onClick={fetchBalance} style={{marginLeft:8}}>Get Balance</button>
      {bal !== null && <div style={{marginTop:12}}><b>Balance:</b> {bal}</div>}
      <p style={{color:'#666',marginTop:24}}>İlk iskelet. Gönder/imzalama sayfasını sonra ekleyeceğiz.</p>
    </div>
  )
}
