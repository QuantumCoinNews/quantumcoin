// Basit bekleme
export function sleep(ms: number) {
  return new Promise(res => setTimeout(res, ms));
}

// Rastgele jitter (anti-spam)
export function jitter(baseMs: number, rangeMs: number) {
  const delta = Math.floor(Math.random() * (rangeMs * 2 + 1)) - rangeMs;
  return baseMs + delta;
}
