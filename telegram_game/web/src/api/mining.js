import { api } from "./client";
import { BATCH_WRITE_SECONDS } from "../game/constants";

const ENDPOINTS = {
    proof: "/api/mining/proof",
    balances: "/api/mining/balances"
};

export function epochSec(date = new Date()) {
    return Math.floor(date.getTime() / 1000);
}

export function isValidAddress(addr) {
    if (!addr) return false;
    const s = String(addr).trim();
    return s.length >= 4;
}

const USE_MOCK = String(import.meta?.env?.VITE_USE_MOCK || "") === "1";

/* ------------------------------ MOCK HELPERS ------------------------------ */

function mockKey(addr) {
    const a = String(addr || "").trim();
    return `qc_mock_wallet_${a}`;
}

function mockRead(addr) {
    try {
        const raw = localStorage.getItem(mockKey(addr));
        if (raw) return JSON.parse(raw);
    } catch { }
    return { confirmed: 0, pending: 0, lastBatchAt: epochSec() };
}

function mockWrite(addr, state) {
    localStorage.setItem(mockKey(addr), JSON.stringify(state));
    return state;
}

function mockMaybeSettle(addr) {
    const now = epochSec();
    const s = mockRead(addr);
    if (!s.lastBatchAt) s.lastBatchAt = now;
    if (now - s.lastBatchAt >= BATCH_WRITE_SECONDS) {
        s.confirmed = Number((s.confirmed + s.pending).toFixed(3));
        s.pending = 0;
        s.lastBatchAt = now;
        mockWrite(addr, s);
    }
    return s;
}

/* ---------------------------------- API ---------------------------------- */

export async function postProofOfPlay({ address, zoneId, blockAt, reward = 50 }) {
    if (!isValidAddress(address)) throw new Error("Invalid address");

    if (USE_MOCK) {
        // pending'e ekle; settle zamanı gelince confirmed'e aktarılır
        const s = mockMaybeSettle(address);
        s.pending = Number((s.pending + Number(reward || 0)).toFixed(3));
        mockWrite(address, s);
        return { ok: true, pending: s.pending, confirmed: s.confirmed };
    }

    const body = { address: String(address).trim(), zoneId, blockAt, reward };
    return api.post(ENDPOINTS.proof, body);
}

export async function getBalances({ address }) {
    if (!isValidAddress(address)) throw new Error("Invalid address");

    if (USE_MOCK) {
        const s = mockMaybeSettle(address);
        return { confirmed: s.confirmed, pending: s.pending, lastBatchAt: s.lastBatchAt };
    }

    const q = new URLSearchParams({ address: String(address).trim() }).toString();
    return api.get(`${ENDPOINTS.balances}?${q}`);
}
