/* Secrets live only in this worker's local Go runtime. No secret HTTP calls. */
"use strict";
let ready;
async function initialize() {
  importScripts("/dev_ui/assets/qc_wallet_crypto/wasm_exec.js");
  const go = new Go();
  const response = await fetch("/dev_ui/assets/qc_wallet_crypto/wallet_crypto.wasm", {cache:"no-store"});
  if (!response.ok) throw Error("Local wallet assets unavailable. Build the wallet assets first.");
  const module = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
  go.run(module.instance).catch(()=>{});
  if (typeof QCWalletCrypto !== "function") throw Error("Local wallet runtime unavailable");
}
self.onmessage = async ({data}) => {
  const id = data?.id;
  try {
    ready ||= initialize(); await ready;
    const result = JSON.parse(QCWalletCrypto(JSON.stringify(data.request)));
    self.postMessage({id, result});
  } catch (_) { self.postMessage({id, result:{ok:false,error:"Local QC wallet unavailable; your saved data has not been changed."}}); }
};
